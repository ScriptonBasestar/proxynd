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
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/helpers"
	"proxynd/internal/adapters/pm/common"
	"proxynd/internal/config"
	"proxynd/internal/container"
	"proxynd/internal/security"
	"proxynd/logging"
	"proxynd/pkg/httpclient"
)

// YUMContainerHandler Container 기반 YUM 프록시 핸들러
type YUMContainerHandler struct {
	logger            logging.Logger
	name              string
	proxyType         string
	yumConfig         *config.YumProxySettings
	storageDir        string
	serverIdx         uint32 // atomic counter for round-robin
	httpClient        *http.Client
	containerProvider container.ContainerProvider
	enabled           bool
}

// NewYUMContainerHandler YUM Container 핸들러 생성
func NewYUMContainerHandler(provider container.ContainerProvider) *YUMContainerHandler {
	// HTTP 클라이언트 최적화 (YUM은 대용량 패키지 파일 처리)
	httpClient := &http.Client{
		Timeout: 120 * time.Second, // YUM 파일은 크므로 긴 타임아웃
		Transport: &http.Transport{
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			ResponseHeaderTimeout: 60 * time.Second,
			DisableCompression:    false, // 압축 지원
		},
	}

	handler := &YUMContainerHandler{
		logger:            logging.GetLogger(),
		name:              "yum-container-handler",
		proxyType:         "yum",
		yumConfig:         &config.YumProxySettings{},
		storageDir:        provider.GetStorageDir(),
		serverIdx:         0,
		httpClient:        httpClient,
		containerProvider: provider,
		enabled:           false,
	}

	// 설정 로딩 시도
	if err := handler.LoadConfig(); err != nil {
		handler.logger.Error("Failed to load YUM configuration", logging.F("error", err))
		handler.enabled = false
		return handler
	}

	handler.enabled = true
	handler.logger.Info("YUM Container handler initialized successfully",
		logging.F("proxies_count", len(handler.yumConfig.Proxies)))

	return handler
}

// LoadConfig YUM 프록시 설정 로딩
func (h *YUMContainerHandler) LoadConfig() error {
	config, err := h.containerProvider.GetYumProxyConfig()
	if err != nil {
		return fmt.Errorf("failed to get YUM proxy config: %w", err)
	}

	h.yumConfig = config

	h.logger.Debug("YUM configuration loaded",
		logging.F("path", config.Path),
		logging.F("use_cache", config.UseCache),
		logging.F("proxies_count", len(config.Proxies)))

	return nil
}

// ReloadConfig 설정 다시 로딩
func (h *YUMContainerHandler) ReloadConfig() error {
	if err := h.LoadConfig(); err != nil {
		h.enabled = false
		return err
	}
	h.enabled = true
	return nil
}

// IsEnabled 핸들러 활성화 상태 확인
func (h *YUMContainerHandler) IsEnabled() bool {
	return h.enabled && h.yumConfig != nil && len(h.yumConfig.Proxies) > 0
}

// GenerateCacheKey 캐시 키 생성
func (h *YUMContainerHandler) GenerateCacheKey(c *fiber.Ctx) string {
	requestPath := c.Params("*")
	return fmt.Sprintf("yum:%s:%s", c.Method(), requestPath)
}

// BuildUpstreamURL 업스트림 URL 생성
func (h *YUMContainerHandler) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	if !h.IsEnabled() {
		return "", errors.New("YUM handler not enabled")
	}

	requestPath := c.Params("*")
	if requestPath == "" {
		return "", errors.New("empty request path")
	}

	// Round-robin 서버 선택
	server := h.selectUpstreamServer()
	upstreamURL := helpers.JoinURL(server.URL, requestPath)

	h.logger.Debug("Built upstream URL",
		logging.F("server", server.Name),
		logging.F("upstream_url", upstreamURL))

	return upstreamURL, nil
}

// selectUpstreamServer Round-robin 방식으로 업스트림 서버 선택
func (h *YUMContainerHandler) selectUpstreamServer() config.YumProxy {
	if len(h.yumConfig.Proxies) == 1 {
		return h.yumConfig.Proxies[0]
	}

	idx := atomic.AddUint32(&h.serverIdx, 1) % uint32(len(h.yumConfig.Proxies))
	return h.yumConfig.Proxies[idx]
}

// TransformRequest 업스트림 요청 변환
func (h *YUMContainerHandler) TransformRequest(c *fiber.Ctx, upstreamReq *fiber.Agent) error {
	// YUM 관련 헤더 설정
	upstreamReq.Set("User-Agent", "ProxyND-YUM/1.0")
	upstreamReq.Set("Accept", "*/*")

	// 원본 요청의 Accept-Encoding 유지 (압축 지원)
	if acceptEncoding := c.Get("Accept-Encoding"); acceptEncoding != "" {
		upstreamReq.Set("Accept-Encoding", acceptEncoding)
	}

	return nil
}

// TransformResponse 응답 변환
func (h *YUMContainerHandler) TransformResponse(resp []byte, c *fiber.Ctx) ([]byte, error) {
	// YUM 응답은 주로 바이너리이므로 변환하지 않음
	return resp, nil
}

// ShouldCache 캐시 여부 결정
func (h *YUMContainerHandler) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	if !h.yumConfig.UseCache {
		return false
	}

	// 성공적인 응답만 캐시
	if statusCode != http.StatusOK {
		return false
	}

	requestPath := c.Params("*")

	// RPM 패키지 파일과 메타데이터 캐시
	cacheable := []string{
		".rpm", ".xml", ".xml.gz", ".xml.bz2", ".xml.xz",
		".sqlite", ".sqlite.gz", ".sqlite.bz2", ".sqlite.xz", ".asc", ".gpg",
	}

	for _, ext := range cacheable {
		if strings.HasSuffix(requestPath, ext) {
			return true
		}
	}

	// repomd.xml 관련 파일들
	if strings.Contains(requestPath, "repomd.xml") || strings.Contains(requestPath, "repodata/") {
		return true
	}

	return false
}

// GetCacheTTL 캐시 TTL 설정
func (h *YUMContainerHandler) GetCacheTTL(c *fiber.Ctx) time.Duration {
	requestPath := c.Params("*")

	switch {
	case strings.HasSuffix(requestPath, ".rpm"):
		// RPM 패키지는 장기간 캐시 (변경되지 않음)
		return 7 * 24 * time.Hour
	case strings.Contains(requestPath, "repomd.xml"):
		// 저장소 메타데이터는 짧은 TTL
		return 1 * time.Hour
	case strings.HasSuffix(requestPath, ".xml") || strings.Contains(requestPath, "repodata/"):
		// 패키지 메타데이터는 중간 TTL
		return 6 * time.Hour
	default:
		// 기본 TTL
		return 24 * time.Hour
	}
}

// HandleError 에러 처리
func (h *YUMContainerHandler) HandleError(err error, c *fiber.Ctx) error {
	h.logger.Error("YUM proxy error",
		logging.F("error", err),
		logging.F("path", c.Path()),
		logging.F("method", c.Method()))

	if strings.Contains(err.Error(), "not enabled") {
		return c.Status(fiber.StatusServiceUnavailable).SendString("YUM proxy service unavailable")
	}

	if strings.Contains(err.Error(), "timeout") {
		return c.Status(fiber.StatusGatewayTimeout).SendString("Upstream timeout")
	}

	return c.Status(fiber.StatusInternalServerError).SendString("Internal server error")
}

// Handle 메인 요청 처리
func (h *YUMContainerHandler) Handle(c *fiber.Ctx) error {
	if !h.IsEnabled() {
		return h.HandleError(errors.New("YUM handler not enabled"), c)
	}

	requestPath := c.Params("*")
	if requestPath == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid request path")
	}

	h.logger.Debug("Processing YUM request",
		logging.F("path", requestPath),
		logging.F("method", c.Method()))

	// 캐시된 파일 확인
	if h.yumConfig.UseCache {
		if cachedFile, exists := h.getCachedFile(requestPath); exists {
			return h.serveCachedFile(c, cachedFile, requestPath)
		}
	}

	// 업스트림에서 파일 가져오기
	return h.fetchFromUpstream(c, requestPath)
}

// getCachedFile 캐시된 파일 확인
func (h *YUMContainerHandler) getCachedFile(requestPath string) (string, bool) {
	baseDir := filepath.Join(h.storageDir, h.yumConfig.Path)
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
func (h *YUMContainerHandler) serveCachedFile(c *fiber.Ctx, cachedFile, requestPath string) error {
	// Content-Type 설정
	contentType := h.getYumContentType(requestPath)
	c.Set("Content-Type", contentType)

	// Content-Disposition 설정
	filename := filepath.Base(cachedFile)
	if h.isYumInlineFile(filename) {
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
func (h *YUMContainerHandler) fetchFromUpstream(c *fiber.Ctx, requestPath string) error {
	ctx, cancel := context.WithTimeout(c.Context(), 120*time.Second)
	defer cancel()

	// 캐시 파일 경로 생성
	baseDir := filepath.Join(h.storageDir, h.yumConfig.Path)
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

	for _, proxy := range h.yumConfig.Proxies {
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
func (h *YUMContainerHandler) saveFile(filePath string, body io.ReadCloser) error {
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

// getYumContentType YUM 파일 타입별 Content-Type 반환
func (h *YUMContainerHandler) getYumContentType(filename string) string {
	switch {
	case strings.HasSuffix(filename, ".rpm"):
		return "application/x-rpm"
	case strings.HasSuffix(filename, ".xml") || strings.HasSuffix(filename, ".xml.gz"):
		return "application/xml"
	case strings.HasSuffix(filename, ".xml.bz2") || strings.HasSuffix(filename, ".xml.xz"):
		return "application/xml"
	case strings.HasSuffix(filename, ".sqlite") || strings.HasSuffix(filename, ".sqlite.bz2"):
		return common.MimeApplicationOctetStream
	case strings.HasSuffix(filename, ".sqlite.gz") || strings.HasSuffix(filename, ".sqlite.xz"):
		return common.MimeApplicationOctetStream
	case strings.HasSuffix(filename, ".asc") || strings.HasSuffix(filename, ".gpg"):
		return "application/pgp-signature"
	case strings.Contains(filename, "repomd.xml"):
		return "text/xml"
	default:
		return common.MimeApplicationOctetStream
	}
}

// isYumInlineFile 인라인으로 표시할 파일 타입 확인
func (h *YUMContainerHandler) isYumInlineFile(filename string) bool {
	inlineExtensions := []string{".xml", ".txt", ".asc", ".gpg"}
	for _, ext := range inlineExtensions {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}
	return false
}

// HealthCheck 핸들러 상태 확인
func (h *YUMContainerHandler) HealthCheck() error {
	if !h.IsEnabled() {
		return errors.New("YUM handler is disabled")
	}

	if len(h.yumConfig.Proxies) == 0 {
		return errors.New("no YUM proxy servers configured")
	}

	// 기본적인 설정 유효성 확인
	for _, proxy := range h.yumConfig.Proxies {
		if proxy.Name == "" || proxy.URL == "" {
			return fmt.Errorf("invalid proxy configuration: %+v", proxy)
		}
	}

	return nil
}

// Name 핸들러 이름 반환
func (h *YUMContainerHandler) Name() string {
	return h.name
}

// Type 핸들러 타입 반환
func (h *YUMContainerHandler) Type() string {
	return h.proxyType
}

// SetContainer Container Provider 설정
func (h *YUMContainerHandler) SetContainer(provider container.ContainerProvider) {
	h.containerProvider = provider
}

// GetContainer Container Provider 반환
func (h *YUMContainerHandler) GetContainer() container.ContainerProvider {
	return h.containerProvider
}
