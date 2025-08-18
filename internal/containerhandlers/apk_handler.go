package containerhandlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/helpers"
	"proxynd/internal/config"
	"proxynd/internal/container"
	"proxynd/internal/logging"
	"proxynd/internal/mirror"
	"proxynd/internal/security"
	"proxynd/pkg/httpclient"
	"proxynd/verification/apk"
)

const (
	mimeApplicationGzip        = "application/gzip"
	mimeTextPlain              = "text/plain"
	mimeApplicationOctetStream = "application/octet-stream"
	mimeApplicationJSON        = "application/json"
)

// APKContainerHandler Container 기반 APK 프록시 핸들러
type APKContainerHandler struct {
	logger            logging.Logger
	name              string
	proxyType         string
	apkConfig         *config.ApkProxySettings
	storageDir        string
	serverIdx         uint32 // atomic counter for round-robin
	httpClient        *http.Client
	containerProvider container.ContainerProvider
	enabled           bool

	// APK 전용 기능들
	verifier        *apk.SignatureVerifier
	verifierOnce    sync.Once
	mirrorSelector  *mirror.AlpineMirrorSelector
	selectorOnce    sync.Once
	selectorStarted bool
	selectorMutex   sync.Mutex
}

// NewAPKContainerHandler APK Container 핸들러 생성
func NewAPKContainerHandler(provider container.ContainerProvider) *APKContainerHandler {
	// HTTP 클라이언트 최적화 (APK는 중간 크기 패키지 파일 처리)
	httpClient := &http.Client{
		Timeout: 45 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:          100,
			IdleConnTimeout:       30 * time.Second,
			ResponseHeaderTimeout: 45 * time.Second,
			DisableCompression:    false, // 압축 지원
		},
	}

	handler := &APKContainerHandler{
		logger:            logging.GetLogger(),
		name:              "apk-container-handler",
		proxyType:         "apk",
		apkConfig:         &config.ApkProxySettings{},
		storageDir:        provider.GetStorageDir(),
		serverIdx:         0,
		httpClient:        httpClient,
		containerProvider: provider,
		enabled:           false,
		selectorStarted:   false,
	}

	// 설정 로딩 시도
	if err := handler.LoadConfig(); err != nil {
		handler.logger.Error("Failed to load APK configuration", logging.F("error", err))
		handler.enabled = false
		return handler
	}

	handler.enabled = true
	handler.logger.Info("APK Container handler initialized successfully",
		logging.F("proxies_count", len(handler.apkConfig.Proxies)),
		logging.F("verification_enabled", handler.apkConfig.Verification.Enabled),
		logging.F("mirror_selection_enabled", handler.apkConfig.MirrorSelection.Enabled))

	return handler
}

// LoadConfig APK 프록시 설정 로딩
func (h *APKContainerHandler) LoadConfig() error {
	config, err := h.containerProvider.GetApkProxyConfig()
	if err != nil {
		return fmt.Errorf("failed to get APK proxy config: %w", err)
	}

	h.apkConfig = config

	h.logger.Debug("APK configuration loaded",
		logging.F("path", config.Path),
		logging.F("use_cache", config.UseCache),
		logging.F("proxies_count", len(config.Proxies)),
		logging.F("verification_enabled", config.Verification.Enabled))

	return nil
}

// ReloadConfig 설정 다시 로딩
func (h *APKContainerHandler) ReloadConfig() error {
	if err := h.LoadConfig(); err != nil {
		h.enabled = false
		return err
	}
	h.enabled = true

	// 설정이 변경되면 미러 선택기 재초기화 필요
	h.selectorStarted = false

	return nil
}

// IsEnabled 핸들러 활성화 상태 확인
func (h *APKContainerHandler) IsEnabled() bool {
	return h.enabled && h.apkConfig != nil && len(h.apkConfig.Proxies) > 0
}

// GenerateCacheKey 캐시 키 생성
func (h *APKContainerHandler) GenerateCacheKey(c *fiber.Ctx) string {
	requestPath := c.Params("*")
	return fmt.Sprintf("apk:%s:%s", c.Method(), requestPath)
}

// BuildUpstreamURL 업스트림 URL 생성
func (h *APKContainerHandler) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	if !h.IsEnabled() {
		return "", errors.New("APK handler not enabled")
	}

	requestPath := c.Params("*")
	if requestPath == "" {
		return "", errors.New("empty request path")
	}

	// 미러 선택 기능이 활성화된 경우
	if h.apkConfig.MirrorSelection.Enabled {
		selector := h.getMirrorSelector()
		proxies := selector.SelectBestMirror(requestPath, h.apkConfig.Proxies)
		if len(proxies) > 0 {
			upstreamURL := helpers.JoinURL(proxies[0].URL, requestPath)
			h.logger.Debug("Built upstream URL using mirror selection",
				logging.F("server", proxies[0].Name),
				logging.F("upstream_url", upstreamURL))
			return upstreamURL, nil
		}
	}

	// Round-robin 서버 선택 (미러 선택이 비활성화되거나 실패한 경우)
	server := h.selectUpstreamServer()
	upstreamURL := helpers.JoinURL(server.URL, requestPath)

	h.logger.Debug("Built upstream URL using round-robin",
		logging.F("server", server.Name),
		logging.F("upstream_url", upstreamURL))

	return upstreamURL, nil
}

// selectUpstreamServer Round-robin 방식으로 업스트림 서버 선택
func (h *APKContainerHandler) selectUpstreamServer() config.ApkProxy {
	if len(h.apkConfig.Proxies) == 1 {
		return h.apkConfig.Proxies[0]
	}

	idx := atomic.AddUint32(&h.serverIdx, 1) % uint32(len(h.apkConfig.Proxies))
	return h.apkConfig.Proxies[idx]
}

// getApkVerifier APK 서명 검증기 싱글톤 인스턴스 반환
func (h *APKContainerHandler) getApkVerifier() *apk.SignatureVerifier {
	h.verifierOnce.Do(func() {
		h.verifier = apk.NewSignatureVerifier()

		// 신뢰할 수 있는 키 로딩
		if h.apkConfig.Verification.Enabled && h.apkConfig.Verification.KeyDirectory != "" {
			if err := h.verifier.LoadTrustedKeys(h.apkConfig.Verification.KeyDirectory); err != nil {
				h.logger.Error("Failed to load APK trusted keys",
					logging.F("key_directory", h.apkConfig.Verification.KeyDirectory),
					logging.F("error", err))
			}
		}
	})
	return h.verifier
}

// getMirrorSelector 미러 선택기 싱글톤 인스턴스 반환
func (h *APKContainerHandler) getMirrorSelector() *mirror.AlpineMirrorSelector {
	h.selectorOnce.Do(func() {
		h.mirrorSelector = mirror.NewAlpineMirrorSelector()
		h.initializeMirrorSelector()
	})
	return h.mirrorSelector
}

// initializeMirrorSelector 미러 선택기 초기화
func (h *APKContainerHandler) initializeMirrorSelector() {
	if !h.apkConfig.MirrorSelection.Enabled {
		return
	}

	h.selectorMutex.Lock()
	defer h.selectorMutex.Unlock()

	if h.selectorStarted {
		return
	}

	mirrorConfig := h.convertToMirrorConfig(h.apkConfig.MirrorSelection)
	h.mirrorSelector.Start(mirrorConfig, h.apkConfig.Proxies)
	h.selectorStarted = true

	h.logger.Info("Mirror selector initialized",
		logging.F("health_check_interval", mirrorConfig.HealthCheckInterval),
		logging.F("preferred_regions", mirrorConfig.PreferredRegions))
}

// convertToMirrorConfig 설정을 미러 설정으로 변환
func (h *APKContainerHandler) convertToMirrorConfig(config config.ApkMirrorSelectionConfig) mirror.AlpineMirrorConfig {
	// 문자열을 time.Duration으로 변환
	healthCheckInterval, err := time.ParseDuration(config.HealthCheckInterval)
	if err != nil || healthCheckInterval == 0 {
		healthCheckInterval = 5 * time.Minute
	}

	healthCheckTimeout, err := time.ParseDuration(config.HealthCheckTimeout)
	if err != nil || healthCheckTimeout == 0 {
		healthCheckTimeout = 10 * time.Second
	}

	maxErrorCount := config.MaxErrorCount
	if maxErrorCount == 0 {
		maxErrorCount = 3
	}

	return mirror.AlpineMirrorConfig{
		HealthCheckInterval: healthCheckInterval,
		HealthCheckTimeout:  healthCheckTimeout,
		PreferredRegions:    config.PreferredRegions,
		FallbackToGlobal:    config.FallbackToGlobal,
		MaxErrorCount:       maxErrorCount,
		RegionDetectionMode: config.RegionDetectionMode,
	}
}

// TransformRequest 업스트림 요청 변환
func (h *APKContainerHandler) TransformRequest(c *fiber.Ctx, upstreamReq *fiber.Agent) error {
	// APK 관련 헤더 설정
	upstreamReq.Set("User-Agent", "ProxyND-APK/1.0")
	upstreamReq.Set("Accept", "*/*")

	// 원본 요청의 Accept-Encoding 유지 (압축 지원)
	if acceptEncoding := c.Get("Accept-Encoding"); acceptEncoding != "" {
		upstreamReq.Set("Accept-Encoding", acceptEncoding)
	}

	return nil
}

// TransformResponse 응답 변환
func (h *APKContainerHandler) TransformResponse(resp []byte, c *fiber.Ctx) ([]byte, error) {
	// APK 응답은 주로 바이너리이므로 변환하지 않음
	return resp, nil
}

// ShouldCache 캐시 여부 결정
func (h *APKContainerHandler) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	if !h.apkConfig.UseCache {
		return false
	}

	// 성공적인 응답만 캐시
	if statusCode != http.StatusOK {
		return false
	}

	requestPath := c.Params("*")

	// APK 파일과 인덱스 파일 캐시
	cacheable := []string{".apk", "APKINDEX", ".asc", ".rsa", ".tar.gz", ".gz"}

	for _, ext := range cacheable {
		if strings.HasSuffix(requestPath, ext) || strings.Contains(requestPath, ext) {
			// 서명 검증이 활성화된 경우, 검증된 파일만 캐시
			if h.apkConfig.Verification.Enabled && h.apkConfig.Verification.CacheValidated {
				if h.isApkFile(requestPath) {
					// APK 파일인 경우 검증 후 캐시 여부 결정
					return h.shouldCacheAfterVerification(c, requestPath)
				}
			}
			return true
		}
	}

	return false
}

// shouldCacheAfterVerification 검증 후 캐시 여부 결정
func (h *APKContainerHandler) shouldCacheAfterVerification(c *fiber.Ctx, requestPath string) bool {
	if !h.apkConfig.Verification.CacheValidated {
		return true // 검증과 관계없이 캐시
	}

	// TODO: 실제 검증 결과를 바탕으로 캐시 여부 결정
	// 현재는 단순히 true 반환 (실제 구현에서는 검증 결과를 참조해야 함)
	return true
}

// GetCacheTTL 캐시 TTL 설정
func (h *APKContainerHandler) GetCacheTTL(c *fiber.Ctx) time.Duration {
	requestPath := c.Params("*")

	switch {
	case strings.HasSuffix(requestPath, ".apk"):
		// APK 패키지는 장기간 캐시 (변경되지 않음)
		return 7 * 24 * time.Hour
	case strings.Contains(requestPath, "APKINDEX"):
		// 패키지 인덱스는 짧은 TTL (자주 업데이트됨)
		return 1 * time.Hour
	case strings.HasSuffix(requestPath, ".asc") || strings.HasSuffix(requestPath, ".rsa"):
		// 서명 파일은 중간 TTL
		return 24 * time.Hour
	default:
		// 기본 TTL
		return 6 * time.Hour
	}
}

// HandleError 에러 처리
func (h *APKContainerHandler) HandleError(err error, c *fiber.Ctx) error {
	h.logger.Error("APK proxy error",
		logging.F("error", err),
		logging.F("path", c.Path()),
		logging.F("method", c.Method()))

	if strings.Contains(err.Error(), "not enabled") {
		return c.Status(fiber.StatusServiceUnavailable).SendString("APK proxy service unavailable")
	}

	if strings.Contains(err.Error(), "signature verification failed") {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":   "APK signature verification failed",
			"message": "The requested APK file failed signature verification",
		})
	}

	if strings.Contains(err.Error(), "timeout") {
		return c.Status(fiber.StatusGatewayTimeout).SendString("Upstream timeout")
	}

	return c.Status(fiber.StatusInternalServerError).SendString("Internal server error")
}

// Handle 메인 요청 처리
func (h *APKContainerHandler) Handle(c *fiber.Ctx) error {
	if !h.IsEnabled() {
		return h.HandleError(errors.New("APK handler not enabled"), c)
	}

	requestPath := c.Params("*")
	if requestPath == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid request path")
	}

	h.logger.Debug("Processing APK request",
		logging.F("path", requestPath),
		logging.F("method", c.Method()))

	// 캐시된 파일 확인
	if h.apkConfig.UseCache {
		if cachedFile, exists := h.getCachedFile(requestPath); exists {
			return h.serveCachedFile(c, cachedFile, requestPath)
		}
	}

	// 업스트림에서 파일 가져오기
	return h.fetchFromUpstream(c, requestPath)
}

// getCachedFile 캐시된 파일 확인
func (h *APKContainerHandler) getCachedFile(requestPath string) (string, bool) {
	baseDir := filepath.Join(h.storageDir, h.apkConfig.Path)
	cachedFile, err := security.SafeJoinPath(baseDir, requestPath)
	if err != nil {
		return "", false
	}

	if helpers.FileExists(cachedFile) {
		h.logger.Debug("Found cached file", logging.F("file", cachedFile))
		return cachedFile, true
	}

	return "", false
}

// serveCachedFile 캐시된 파일 서빙
func (h *APKContainerHandler) serveCachedFile(c *fiber.Ctx, cachedFile, requestPath string) error {
	// APK 파일인 경우 서명 검증 수행
	if h.apkConfig.Verification.Enabled && h.isApkFile(requestPath) {
		if !h.verifyApkFileSignature(cachedFile) {
			return h.HandleError(errors.New("signature verification failed"), c)
		}
	}

	// Content-Type 설정
	contentType := h.getApkContentType(requestPath)
	c.Set("Content-Type", contentType)

	// Content-Disposition 설정
	filename := filepath.Base(cachedFile)
	if h.isApkInlineFile(filename) {
		c.Set("Content-Disposition", "inline; filename="+filename)
	} else {
		c.Set("Content-Disposition", "attachment; filename="+filename)
	}

	h.logger.Debug("Serving cached file",
		logging.F("file", cachedFile),
		logging.F("content_type", contentType))

	return c.SendFile(cachedFile)
}

// fetchFromUpstream 업스트림에서 파일 가져오기
func (h *APKContainerHandler) fetchFromUpstream(c *fiber.Ctx, requestPath string) error {
	ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
	defer cancel()

	// 캐시 파일 경로 생성
	baseDir := filepath.Join(h.storageDir, h.apkConfig.Path)
	cachedFile, err := security.SafeJoinPath(baseDir, requestPath)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid path")
	}

	// 디렉토리 생성
	dirPath := filepath.Dir(cachedFile)
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		h.logger.Error("Failed to create directory",
			logging.F("dir", dirPath),
			logging.F("error", err))
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to create directory")
	}

	// 업스트림에서 파일 다운로드
	proxyClient := httpclient.NewProxyClient()

	// 미러 선택을 사용하여 프록시 목록 가져오기
	proxies := h.apkConfig.Proxies
	if h.apkConfig.MirrorSelection.Enabled {
		selector := h.getMirrorSelector()
		proxies = selector.SelectBestMirror(requestPath, h.apkConfig.Proxies)
		h.logger.Debug("Mirror selection result",
			logging.F("selected_mirrors", len(proxies)))
	}

	for _, proxy := range proxies {
		fullURL := helpers.JoinURL(proxy.URL, requestPath)
		h.logger.Debug("Fetching from upstream",
			logging.F("server", proxy.Name),
			logging.F("url", fullURL))

		resp, err := proxyClient.GetWithRetry(ctx, fullURL, 2)
		if err != nil {
			h.logger.Warn("Failed to fetch from upstream",
				logging.F("server", proxy.Name),
				logging.F("error", err))
			continue
		}

		if resp.StatusCode == http.StatusOK {
			// 파일 저장
			if err := h.saveFile(cachedFile, resp.Body); err != nil {
				func() { _ = resp.Body.Close() }()
				h.logger.Error("Failed to save file",
					logging.F("file", cachedFile),
					logging.F("error", err))
				return c.Status(fiber.StatusInternalServerError).SendString("Failed to save file")
			}
			func() { _ = resp.Body.Close() }()

			// 새로 다운로드한 APK 파일인 경우 서명 검증 수행
			if h.apkConfig.Verification.Enabled && h.isApkFile(requestPath) {
				if !h.verifyApkFileSignature(cachedFile) {
					return h.HandleError(errors.New("signature verification failed"), c)
				}
			}

			// 성공적으로 저장된 파일 서빙
			return h.serveCachedFile(c, cachedFile, requestPath)
		}

		func() { _ = resp.Body.Close() }()
		h.logger.Warn("Upstream returned non-OK status",
			logging.F("server", proxy.Name),
			logging.F("status", resp.StatusCode))
	}

	return c.Status(fiber.StatusNotFound).SendString("File not found in any upstream")
}

// saveFile 파일 저장
func (h *APKContainerHandler) saveFile(filePath string, body io.ReadCloser) error {
	out, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, body)
	if err != nil {
		// 실패한 파일 정리
		_ = os.Remove(filePath)
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// verifyApkFileSignature APK 파일의 서명 검증 수행
func (h *APKContainerHandler) verifyApkFileSignature(filePath string) bool {
	verifier := h.getApkVerifier()
	result := verifier.VerifyApkSignature(filePath)

	h.logger.Debug("APK signature verification result",
		logging.F("file", filePath),
		logging.F("valid", result.IsValid),
		logging.F("signature_file", result.SignatureFile),
		logging.F("key_fingerprint", result.KeyFingerprint))

	// 검증 성공 시
	if result.IsValid {
		h.logger.Info("APK signature verification succeeded",
			logging.F("file", filePath),
			logging.F("key", result.KeyFingerprint))
		return true
	}

	// 검증 실패 시 처리
	h.logger.Warn("APK signature verification failed",
		logging.F("file", filePath),
		logging.F("error", result.Error))

	// 검증 실패 시 요청 차단 설정이 활성화된 경우
	if h.apkConfig.Verification.FailOnInvalid {
		h.logger.Error("APK signature verification failed, blocking request",
			logging.F("file", filePath))
		return false
	}

	// 경고만 하고 계속 진행
	h.logger.Warn("APK signature verification failed but continuing",
		logging.F("file", filePath),
		logging.F("reason", "fail_on_invalid is disabled"))

	return true
}

// getApkContentType APK 파일 타입별 Content-Type 반환
func (h *APKContainerHandler) getApkContentType(filename string) string {
	switch {
	case strings.HasSuffix(filename, ".apk"):
		return "application/vnd.alpine.apk"
	case strings.HasSuffix(filename, "APKINDEX.tar.gz"):
		return mimeApplicationGzip
	case strings.HasSuffix(filename, "APKINDEX"):
		return mimeTextPlain
	case strings.HasSuffix(filename, ".asc"):
		return "application/pgp-signature"
	case strings.HasSuffix(filename, ".rsa"):
		return mimeApplicationOctetStream
	case strings.HasSuffix(filename, ".tar.gz"):
		return mimeApplicationGzip
	case strings.HasSuffix(filename, ".gz"):
		return mimeApplicationGzip
	default:
		return mimeApplicationOctetStream
	}
}

// isApkInlineFile 인라인으로 표시할 파일 타입 확인
func (h *APKContainerHandler) isApkInlineFile(filename string) bool {
	inlineExtensions := []string{"APKINDEX", ".asc", ".txt"}
	for _, ext := range inlineExtensions {
		if strings.Contains(filename, ext) {
			return true
		}
	}
	return false
}

// isApkFile APK 패키지 파일인지 확인
func (h *APKContainerHandler) isApkFile(filename string) bool {
	return strings.HasSuffix(filename, ".apk")
}

// HealthCheck 핸들러 상태 확인
func (h *APKContainerHandler) HealthCheck() error {
	if !h.IsEnabled() {
		return errors.New("APK handler is disabled")
	}

	if len(h.apkConfig.Proxies) == 0 {
		return errors.New("no APK proxy servers configured")
	}

	// 기본적인 설정 유효성 확인
	for _, proxy := range h.apkConfig.Proxies {
		if proxy.Name == "" || proxy.URL == "" {
			return fmt.Errorf("invalid proxy configuration: %+v", proxy)
		}
	}

	// 서명 검증 설정 확인
	if h.apkConfig.Verification.Enabled && h.apkConfig.Verification.KeyDirectory == "" {
		return errors.New("verification enabled but key directory not specified")
	}

	return nil
}

// Name 핸들러 이름 반환
func (h *APKContainerHandler) Name() string {
	return h.name
}

// Type 핸들러 타입 반환
func (h *APKContainerHandler) Type() string {
	return h.proxyType
}

// SetContainer Container Provider 설정
func (h *APKContainerHandler) SetContainer(provider container.ContainerProvider) {
	h.containerProvider = provider
}

// GetContainer Container Provider 반환
func (h *APKContainerHandler) GetContainer() container.ContainerProvider {
	return h.containerProvider
}
