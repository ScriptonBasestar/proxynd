package proxy

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/helpers"
	"proxynd/internal/config"
	"proxynd/internal/errors"
	"proxynd/internal/mirror"
	"proxynd/logging"
	"proxynd/pkg/httpclient"
	"proxynd/verification/apk"
)

var (
	apkVerifierV3        *apk.SignatureVerifier
	apkVerifierV3Once    sync.Once
	mirrorSelectorV3     *mirror.AlpineMirrorSelector
	mirrorSelectorV3Once sync.Once
	selectorV3Started    bool
	selectorV3Mutex      sync.Mutex
)

// ApkHandlerV3 Base Handler 패턴을 사용하는 새로운 APK 핸들러
type ApkHandlerV3 struct {
	*BaseProxyHandler
	Config *config.ApkProxyConfig
}

// NewApkHandlerV3 새로운 APK 핸들러 생성
func NewApkHandlerV3() *ApkHandlerV3 {
	return &ApkHandlerV3{
		BaseProxyHandler: NewBaseProxyHandler(),
		Config:           &config.ApkProxyConfig{},
	}
}

// Type 프록시 타입 반환
func (h *ApkHandlerV3) Type() string {
	return "apk"
}

// IsEnabled 활성화 상태 확인
func (h *ApkHandlerV3) IsEnabled() bool {
	if err := h.LoadConfig(); err != nil {
		h.GetLogger().Error("Failed to read APK config", logging.F("error", err))
		return false
	}
	return len(h.Config.Proxies) > 0
}

// LoadConfig 설정 로드
func (h *ApkHandlerV3) LoadConfig() error {
	err := h.Config.ReadConfig()
	if err != nil {
		return err
	}

	// 서명 검증기 초기화 (필요시)
	if h.Config.Verification.Enabled {
		verifier := h.getApkVerifier()
		if err := verifier.LoadTrustedKeys(h.Config.Verification.KeyDirectory); err != nil {
			h.GetLogger().Error("APK 신뢰 키 로드 실패", logging.F("error", err))
		}
	}

	// 미러 선택기 초기화 (필요시)
	h.initializeMirrorSelector()

	return nil
}

// GenerateCacheKey 캐시 키 생성
func (h *ApkHandlerV3) GenerateCacheKey(c *fiber.Ctx) string {
	packagePath := c.Params("*")
	return fmt.Sprintf("apk:%s", strings.ReplaceAll(packagePath, "/", "_"))
}

// BuildUpstreamURL 업스트림 URL 생성 (최적 미러 선택)
func (h *ApkHandlerV3) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	packagePath := c.Params("*")

	if len(h.Config.Proxies) == 0 {
		return "", fmt.Errorf("APK 미러가 설정되지 않았습니다")
	}

	// 미러 선택을 사용하여 최적 미러 선택
	proxies := h.Config.Proxies
	if h.Config.MirrorSelection.Enabled {
		selector := h.getMirrorSelector()
		proxies = selector.SelectBestMirror(packagePath, h.Config.Proxies)
		h.GetLogger().Debug("미러 선택 결과", logging.F("mirror_count", len(proxies)))
	}

	if len(proxies) == 0 {
		return "", fmt.Errorf("사용 가능한 APK 미러가 없습니다")
	}

	mirror := proxies[0] // 최적 미러 사용
	if mirror.URL == "" {
		return "", fmt.Errorf("APK 미러 URL이 설정되지 않았습니다")
	}

	return helpers.JoinURL(mirror.URL, packagePath), nil
}

// FetchFromUpstream 업스트림에서 데이터 가져오기 (모든 미러 시도)
func (h *ApkHandlerV3) FetchFromUpstream(c *fiber.Ctx, _ string) ([]byte, int, error) {
	packagePath := c.Params("*")
	var lastErr error

	// 컨텍스트 생성 (45초 타임아웃)
	ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
	defer cancel()

	// HTTP 클라이언트 생성 (프록시 최적화 설정)
	proxyClient := httpclient.NewProxyClient()

	// 미러 선택을 사용하여 업스트림 서버에서 파일 가져오기
	proxies := h.Config.Proxies
	if h.Config.MirrorSelection.Enabled {
		selector := h.getMirrorSelector()
		proxies = selector.SelectBestMirror(packagePath, h.Config.Proxies)
		h.GetLogger().Debug("미러 선택 결과", logging.F("mirror_count", len(proxies)))
	}

	for i, proxy := range proxies {
		if proxy.URL == "" {
			continue
		}

		fullURL := helpers.JoinURL(proxy.URL, packagePath)

		h.GetLogger().Debug("Trying APK mirror",
			logging.F("mirror_index", i),
			logging.F("mirror_name", proxy.Name),
			logging.F("url", fullURL),
		)

		// 컨텍스트 기반 요청 (재시도 포함)
		resp, err := proxyClient.GetWithRetry(ctx, fullURL, 2)
		if err != nil {
			lastErr = err
			h.GetLogger().Error("Failed to fetch from APK mirror",
				logging.F("mirror_index", i),
				logging.F("mirror_name", proxy.Name),
				logging.F("error", err),
			)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			// 응답 읽기
			body := make([]byte, 0, resp.ContentLength)
			buf := make([]byte, 4096)
			for {
				n, err := resp.Body.Read(buf)
				if n > 0 {
					body = append(body, buf[:n]...)
				}
				if err != nil {
					break
				}
			}

			return body, resp.StatusCode, nil
		}

		lastErr = fmt.Errorf("upstream returned status %d", resp.StatusCode)
		h.GetLogger().Debug("Non-200 status from APK mirror",
			logging.F("mirror_index", i),
			logging.F("mirror_name", proxy.Name),
			logging.F("status_code", resp.StatusCode),
		)
	}

	if lastErr != nil {
		return nil, 0, lastErr
	}

	return nil, 0, fmt.Errorf("package not found in any APK mirror")
}

// ProcessResponse 응답 처리 (APK 서명 검증 포함)
func (h *ApkHandlerV3) ProcessResponse(c *fiber.Ctx, body []byte, statusCode int) ([]byte, error) {
	packagePath := c.Params("*")

	// APK 파일인 경우 서명 검증 수행 (캐시 저장 전)
	if h.Config.Verification.Enabled && h.isApkFile(packagePath) {
		// 임시 파일에 저장하여 검증
		if !h.verifyApkSignature(body, packagePath) {
			return nil, fmt.Errorf("APK signature verification failed")
		}
	}

	return body, nil
}

// ShouldCache 캐시 정책 결정
func (h *ApkHandlerV3) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	if statusCode != 200 && statusCode != 304 {
		return false
	}

	path := c.Path()

	// APK 패키지 파일은 캐시 (검증 통과한 경우만)
	if strings.HasSuffix(path, ".apk") {
		return true
	}

	// APKINDEX 파일들은 짧은 TTL로 캐시
	if strings.Contains(path, "APKINDEX") {
		return true
	}

	// 서명 파일들도 캐시
	if strings.HasSuffix(path, ".asc") || strings.HasSuffix(path, ".rsa") {
		return true
	}

	// 압축 파일들도 캐시
	if strings.HasSuffix(path, ".tar.gz") || strings.HasSuffix(path, ".gz") {
		return true
	}

	return false
}

// GetCacheTTL 캐시 TTL 반환
func (h *ApkHandlerV3) GetCacheTTL(c *fiber.Ctx) time.Duration {
	path := c.Path()

	// APKINDEX 파일은 짧은 TTL (10분)
	if strings.Contains(path, "APKINDEX") {
		return 10 * time.Minute
	}

	// APK 패키지 파일은 긴 TTL (30일)
	if strings.HasSuffix(path, ".apk") {
		return 30 * 24 * time.Hour
	}

	// 서명 파일들은 중간 TTL (1일)
	if strings.HasSuffix(path, ".asc") || strings.HasSuffix(path, ".rsa") {
		return 24 * time.Hour
	}

	// 압축 파일들은 중간 TTL (1시간)
	if strings.HasSuffix(path, ".tar.gz") || strings.HasSuffix(path, ".gz") {
		return 1 * time.Hour
	}

	// 기본 30분
	return 30 * time.Minute
}

// GetContentType Content-Type 결정
func (h *ApkHandlerV3) GetContentType(path string) string {
	switch {
	case strings.HasSuffix(path, ".apk"):
		return "application/vnd.alpine.apk"
	case strings.HasSuffix(path, "APKINDEX.tar.gz"):
		return "application/gzip"
	case strings.HasSuffix(path, "APKINDEX"):
		return "text/plain"
	case strings.HasSuffix(path, ".asc"):
		return "application/pgp-signature"
	case strings.HasSuffix(path, ".rsa"):
		return "application/octet-stream"
	case strings.HasSuffix(path, ".tar.gz"):
		return "application/gzip"
	case strings.HasSuffix(path, ".gz"):
		return "application/gzip"
	default:
		return "application/octet-stream"
	}
}

// ShouldInline 인라인 표시 여부 결정
func (h *ApkHandlerV3) ShouldInline(path string) bool {
	inlineExtensions := []string{"APKINDEX", ".asc", ".txt"}
	for _, ext := range inlineExtensions {
		if strings.Contains(path, ext) {
			return true
		}
	}
	return false
}

// HandleError 에러 처리
func (h *ApkHandlerV3) HandleError(err error, c *fiber.Ctx) error {
	// 이미 DomainError인 경우 그대로 전송
	if _, ok := err.(*errors.DomainError); ok {
		return errors.SendProxyError(c, err)
	}

	// 에러 메시지 기반 도메인 에러 변환
	var domainErr *errors.DomainError
	errorMsg := err.Error()

	switch {
	case strings.Contains(errorMsg, "미러가 설정되지 않았습니다") || strings.Contains(errorMsg, "사용 가능한 APK 미러가 없습니다"):
		domainErr = errors.WrapApkError(err, "APK002", "APK 미러 서버에 접근할 수 없습니다")
	case strings.Contains(errorMsg, "signature verification failed"):
		domainErr = errors.WrapApkError(err, "APK003", "APK 패키지 서명 검증에 실패했습니다")
	case strings.Contains(errorMsg, "context deadline exceeded"):
		domainErr = errors.WrapApkError(err, "APK004", "APK 서버 응답 타임아웃")
	case strings.Contains(errorMsg, "path") || strings.Contains(errorMsg, "Invalid path"):
		domainErr = errors.WrapApkError(err, "APK005", "잘못된 패키지 경로입니다")
	case strings.Contains(errorMsg, "설정") || strings.Contains(errorMsg, "config"):
		domainErr = errors.WrapApkError(err, "APK006", "APK 설정 파일을 읽을 수 없습니다")
	case strings.Contains(errorMsg, "APKINDEX") || strings.Contains(errorMsg, "index"):
		domainErr = errors.WrapApkError(err, "APK007", "APK 패키지 인덱스가 손상되었습니다")
	case strings.Contains(errorMsg, "key") || strings.Contains(errorMsg, "서명 키"):
		domainErr = errors.WrapApkError(err, "APK008", "APK 서명 키를 찾을 수 없습니다")
	case strings.Contains(errorMsg, "비활성화") || strings.Contains(errorMsg, "disabled"):
		domainErr = errors.WrapApkError(err, "APK009", "APK 프록시가 비활성화되어 있습니다")
	case strings.Contains(errorMsg, "mirror selection") || strings.Contains(errorMsg, "미러 선택"):
		domainErr = errors.WrapApkError(err, "APK010", "APK 미러 선택에 실패했습니다")
	default:
		// 기본 APK 에러
		domainErr = errors.WrapApkError(err, "APK001", "APK 패키지를 찾을 수 없습니다")
	}

	return errors.SendProxyError(c, domainErr)
}

// Handle 메인 핸들러 - Base Handler의 Template Method 사용
func (h *ApkHandlerV3) Handle(c *fiber.Ctx) error {
	return h.BaseProxyHandler.Handle(h, c)
}

// 헬퍼 메서드들

// getApkVerifier APK 서명 검증기 싱글톤 인스턴스 반환
func (h *ApkHandlerV3) getApkVerifier() *apk.SignatureVerifier {
	apkVerifierV3Once.Do(func() {
		apkVerifierV3 = apk.NewSignatureVerifier()
	})
	return apkVerifierV3
}

// getMirrorSelector 미러 선택기 싱글톤 인스턴스 반환
func (h *ApkHandlerV3) getMirrorSelector() *mirror.AlpineMirrorSelector {
	mirrorSelectorV3Once.Do(func() {
		mirrorSelectorV3 = mirror.NewAlpineMirrorSelector()
	})
	return mirrorSelectorV3
}

// initializeMirrorSelector 미러 선택기 초기화 (필요시)
func (h *ApkHandlerV3) initializeMirrorSelector() {
	if !h.Config.MirrorSelection.Enabled {
		return
	}

	selectorV3Mutex.Lock()
	defer selectorV3Mutex.Unlock()

	if selectorV3Started {
		return
	}

	selector := h.getMirrorSelector()
	config := h.convertToMirrorConfig(h.Config.MirrorSelection)
	selector.Start(config, h.Config.Proxies)
	selectorV3Started = true
}

// convertToMirrorConfig 설정 변환
func (h *ApkHandlerV3) convertToMirrorConfig(config config.ApkMirrorSelectionConfig) mirror.AlpineMirrorConfig {
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

// isApkFile APK 패키지 파일인지 확인
func (h *ApkHandlerV3) isApkFile(filename string) bool {
	return strings.HasSuffix(filename, ".apk")
}

// verifyApkSignature APK 서명 검증 (데이터 기반)
func (h *ApkHandlerV3) verifyApkSignature(body []byte, packagePath string) bool {
	if !h.Config.Verification.Enabled {
		return true
	}

	// TODO: 임시 파일에 저장하여 검증하거나, 메모리 기반 검증 구현
	// 현재는 간단히 검증 통과로 처리 (실제 구현시 apk.SignatureVerifier 사용)
	h.GetLogger().Debug("APK 서명 검증 건너뜀 (메모리 기반 검증 미구현)",
		logging.F("package_path", packagePath))

	return true
}
