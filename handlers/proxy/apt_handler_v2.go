package proxy

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/configs"
	"proxynd/helpers"
	"proxynd/internal/errors"
	"proxynd/logging"
)

// APTHandlerV2 Template Method 패턴을 사용하는 APT 핸들러
type APTHandlerV2 struct {
	logger logging.Logger
	config *configs.AptProxyConfig
}

// NewAPTHandlerV2 새로운 APT 핸들러 v2 생성
func NewAPTHandlerV2() *APTHandlerV2 {
	return &APTHandlerV2{
		logger: logging.GetLogger(),
		config: &configs.AptProxyConfig{},
	}
}

// BaseProxyHandler 인터페이스 구현

// Type 프록시 타입 반환
func (h *APTHandlerV2) Type() string {
	return "apt"
}

// IsEnabled 활성화 상태 확인
func (h *APTHandlerV2) IsEnabled() bool {
	if err := h.config.ReadConfig(); err != nil {
		h.logger.Error("Failed to read APT config", logging.F("error", err))
		return false
	}
	return len(h.config.Proxies) > 0
}

// GenerateCacheKey 캐시 키 생성
func (h *APTHandlerV2) GenerateCacheKey(c *fiber.Ctx) string {
	osType := c.Params("osType", "ubuntu")
	path := c.Params("*")
	return fmt.Sprintf("apt:%s:%s", osType, strings.ReplaceAll(path, "/", "_"))
}

// BuildUpstreamURL 업스트림 URL 구성
func (h *APTHandlerV2) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	if err := h.config.ReadConfig(); err != nil {
		return "", fmt.Errorf("APT 설정 로드 실패: %w", err)
	}

	osType := c.Params("osType", "ubuntu")
	packagePath := c.Params("*")

	// OS별 프록시 설정 확인
	proxies, exists := h.config.Proxies[osType]
	if !exists || len(proxies) == 0 {
		return "", fmt.Errorf("OS 타입 '%s'에 대한 APT 미러가 설정되지 않았습니다", osType)
	}

	// 첫 번째 미러 사용 (추후 로드밸런싱 구현)
	mirror := proxies[0]
	if mirror.URL == "" {
		return "", fmt.Errorf("APT 미러 URL이 설정되지 않았습니다")
	}

	// URL 구성
	baseURL := strings.TrimRight(mirror.URL, "/")
	cleanPath := strings.TrimLeft(packagePath, "/")
	
	return fmt.Sprintf("%s/%s", baseURL, cleanPath), nil
}

// TransformRequest 요청 변환
func (h *APTHandlerV2) TransformRequest(c *fiber.Ctx, upstreamReq *fiber.Request) error {
	// APT 특화 헤더 추가
	upstreamReq.Header.Set("X-APT-Proxy", "ProxyND")
	upstreamReq.Header.Set("User-Agent", "ProxyND/1.0 APT-Proxy")

	// 압축 지원
	if !upstreamReq.Header.Contains("Accept-Encoding") {
		upstreamReq.Header.Set("Accept-Encoding", "gzip, deflate")
	}

	// 캐시 제어 헤더 (메타데이터의 경우)
	path := c.Path()
	if h.isMetadataFile(path) {
		upstreamReq.Header.Set("Cache-Control", "max-age=300") // 5분
	}

	h.logger.Debug("APT request transformed",
		logging.F("path", path),
		logging.F("upstream_url", string(upstreamReq.URI().FullURI())),
	)

	return nil
}

// TransformResponse 응답 변환
func (h *APTHandlerV2) TransformResponse(resp []byte, c *fiber.Ctx) ([]byte, error) {
	path := c.Path()

	// Release 파일 서명 검증 (선택적)
	if strings.Contains(path, "Release") && !strings.Contains(path, "Release.gpg") {
		h.logger.Debug("Processing Release file", logging.F("path", path))
		
		// Release 파일에 ProxyND 정보 추가 (선택적)
		if h.shouldAddProxyInfo() {
			proxyInfo := fmt.Sprintf("\nX-ProxyND-Cache: %s\nX-ProxyND-Timestamp: %s\n", 
				c.Get("X-Cache-Status", "MISS"), 
				time.Now().Format(time.RFC3339))
			
			// 응답 끝에 추가 (실제로는 해시 무결성을 위해 주의 필요)
			return append(resp, []byte(proxyInfo)...), nil
		}
	}

	// Packages 파일 처리
	if strings.Contains(path, "Packages") {
		h.logger.Debug("Processing Packages file", logging.F("path", path))
		// 패키지 목록 후처리 로직 (필요 시)
	}

	return resp, nil
}

// ShouldCache 캐시 정책 결정
func (h *APTHandlerV2) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	if statusCode != 200 && statusCode != 304 {
		return false
	}

	path := c.Path()
	
	// 메타데이터 파일은 캐시
	if h.isMetadataFile(path) {
		return true
	}

	// .deb 패키지 파일은 캐시
	if strings.HasSuffix(path, ".deb") {
		return true
	}

	// .udeb 패키지 파일도 캐시
	if strings.HasSuffix(path, ".udeb") {
		return true
	}

	// 압축 파일들도 캐시
	if strings.HasSuffix(path, ".gz") || strings.HasSuffix(path, ".xz") || strings.HasSuffix(path, ".bz2") {
		return true
	}

	return false
}

// GetCacheTTL 캐시 TTL 반환
func (h *APTHandlerV2) GetCacheTTL(c *fiber.Ctx) time.Duration {
	path := c.Path()

	// Release 파일은 짧은 TTL (10분)
	if strings.Contains(path, "Release") {
		return 10 * time.Minute
	}

	// Packages 파일은 중간 TTL (30분)
	if strings.Contains(path, "Packages") {
		return 30 * time.Minute
	}

	// .deb 패키지 파일은 긴 TTL (7일)
	if strings.HasSuffix(path, ".deb") || strings.HasSuffix(path, ".udeb") {
		return 7 * 24 * time.Hour
	}

	// 압축 파일들은 하루
	if strings.HasSuffix(path, ".gz") || strings.HasSuffix(path, ".xz") || strings.HasSuffix(path, ".bz2") {
		return 24 * time.Hour
	}

	// 기본 1시간
	return 1 * time.Hour
}

// HandleError 에러 처리
func (h *APTHandlerV2) HandleError(err error, c *fiber.Ctx) error {
	// APT 도메인 에러로 변환
	if strings.Contains(err.Error(), "미러가 설정되지 않았습니다") {
		return errors.WrapAPTError(err, "APT002", "APT 미러 서버에 접근할 수 없습니다")
	}

	if strings.Contains(err.Error(), "설정 로드 실패") {
		return errors.WrapAPTError(err, "APT006", "APT 설정 파일을 읽을 수 없습니다")
	}

	if strings.Contains(err.Error(), "path") {
		return errors.WrapAPTError(err, "APT005", "잘못된 APT 패키지 경로입니다")
	}

	// 기본 APT 에러
	return errors.WrapAPTError(err, "APT001", "APT 패키지를 찾을 수 없습니다")
}

// 헬프 메서드들

// isMetadataFile 메타데이터 파일인지 확인
func (h *APTHandlerV2) isMetadataFile(path string) bool {
	return strings.Contains(path, "Release") ||
		strings.Contains(path, "Packages") ||
		strings.Contains(path, "Sources") ||
		strings.Contains(path, "Contents") ||
		strings.HasSuffix(path, "Release.gpg") ||
		strings.HasSuffix(path, "InRelease")
}

// shouldAddProxyInfo 프록시 정보 추가 여부
func (h *APTHandlerV2) shouldAddProxyInfo() bool {
	// 설정에서 확인하거나 기본값 false
	return false
}

// AuthenticatedProxyHandler 인터페이스 구현 (선택적)

// GetUpstreamAuth 업스트림 인증 정보 반환
func (h *APTHandlerV2) GetUpstreamAuth(c *fiber.Ctx) (string, string, error) {
	if err := h.config.ReadConfig(); err != nil {
		return "", "", err
	}

	osType := c.Params("osType", "ubuntu")
	proxies, exists := h.config.Proxies[osType]
	if !exists || len(proxies) == 0 {
		return "", "", nil // 인증 정보 없음
	}

	proxy := proxies[0]
	if proxy.BasicAuth.Username != "" {
		return proxy.BasicAuth.Username, proxy.BasicAuth.Password, nil
	}

	return "", "", nil
}

// ValidateClientAuth 클라이언트 인증 검증
func (h *APTHandlerV2) ValidateClientAuth(c *fiber.Ctx) error {
	// 현재는 클라이언트 인증 없음
	return nil
}

// MetricsAwareProxyHandler 인터페이스 구현 (선택적)

// RecordRequestMetrics 요청 메트릭 기록
func (h *APTHandlerV2) RecordRequestMetrics(c *fiber.Ctx, statusCode int, duration time.Duration) {
	h.logger.Info("APT request metrics",
		logging.F("status_code", statusCode),
		logging.F("duration_ms", duration.Milliseconds()),
		logging.F("path", c.Path()),
		logging.F("method", c.Method()),
		logging.F("os_type", c.Params("osType", "ubuntu")),
	)
}

// RecordCacheMetrics 캐시 메트릭 기록
func (h *APTHandlerV2) RecordCacheMetrics(cacheKey string, hit bool, size int) {
	status := "miss"
	if hit {
		status = "hit"
	}
	
	h.logger.Info("APT cache metrics",
		logging.F("cache_key", cacheKey),
		logging.F("cache_status", status),
		logging.F("size_bytes", size),
	)
}

// Healthable 인터페이스 구현

// HealthCheck APT 핸들러 헬스체크
func (h *APTHandlerV2) HealthCheck() error {
	if err := h.config.ReadConfig(); err != nil {
		return fmt.Errorf("APT 설정 파일 읽기 실패: %w", err)
	}

	if len(h.config.Proxies) == 0 {
		return fmt.Errorf("APT 프록시가 설정되지 않았습니다")
	}

	// 최소 하나의 유효한 미러가 있는지 확인
	hasValidMirror := false
	for osType, proxies := range h.config.Proxies {
		for _, proxy := range proxies {
			if proxy.URL != "" {
				h.logger.Debug("Found valid APT mirror",
					logging.F("os_type", osType),
					logging.F("mirror_url", proxy.URL),
				)
				hasValidMirror = true
				break
			}
		}
		if hasValidMirror {
			break
		}
	}

	if !hasValidMirror {
		return fmt.Errorf("유효한 APT 미러가 설정되지 않았습니다")
	}

	return nil
}