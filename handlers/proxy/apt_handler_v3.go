package proxy

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/config"
	"proxynd/internal/errors"
	"proxynd/logging"
)

// APTHandlerV3 Base Handler 패턴을 사용하는 새로운 APT 핸들러
type APTHandlerV3 struct {
	*BaseProxyHandler
	Config    *config.AptProxyConfig
	mirrorIdx int32 // 라운드로빈을 위한 atomic counter
}

// NewAPTHandlerV3 새로운 APT 핸들러 생성
func NewAPTHandlerV3() *APTHandlerV3 {
	return &APTHandlerV3{
		BaseProxyHandler: NewBaseProxyHandler(),
		Config:           &config.AptProxyConfig{},
		mirrorIdx:        0,
	}
}

// Type 프록시 타입 반환
func (h *APTHandlerV3) Type() string {
	return "apt"
}

// IsEnabled 활성화 상태 확인
func (h *APTHandlerV3) IsEnabled() bool {
	if err := h.LoadConfig(); err != nil {
		h.GetLogger().Error("Failed to read APT config", logging.F("error", err))
		return false
	}
	return len(h.Config.Proxies) > 0
}

// LoadConfig 설정 로드
func (h *APTHandlerV3) LoadConfig() error {
	return h.Config.ReadConfig()
}

// GenerateCacheKey 캐시 키 생성
func (h *APTHandlerV3) GenerateCacheKey(c *fiber.Ctx) string {
	osType := c.Params("osType", "ubuntu")
	path := c.Params("*")
	return fmt.Sprintf("apt:%s:%s", osType, strings.ReplaceAll(path, "/", "_"))
}

// BuildUpstreamURL 업스트림 URL 생성 (라운드로빈 지원)
func (h *APTHandlerV3) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	osType := c.Params("osType", "ubuntu")
	packagePath := c.Params("*")

	// OS별 프록시 설정 확인
	proxies, exists := h.Config.Proxies[osType]
	if !exists || len(proxies) == 0 {
		return "", fmt.Errorf("OS 타입 '%s'에 대한 APT 미러가 설정되지 않았습니다", osType)
	}

	// 라운드로빈으로 미러 선택
	idx := atomic.AddInt32(&h.mirrorIdx, 1) - 1
	mirror := proxies[idx%int32(len(proxies))]

	if mirror.URL == "" {
		return "", fmt.Errorf("APT 미러 URL이 설정되지 않았습니다")
	}

	// URL 구성
	baseURL := strings.TrimRight(mirror.URL, "/")
	cleanPath := strings.TrimLeft(packagePath, "/")

	return fmt.Sprintf("%s/%s", baseURL, cleanPath), nil
}

// FetchFromUpstream 업스트림에서 데이터 가져오기
func (h *APTHandlerV3) FetchFromUpstream(c *fiber.Ctx, upstreamURL string) ([]byte, int, error) {
	// Fiber Agent로 요청
	agent := fiber.Get(upstreamURL)

	// 헤더 설정
	agent.Set("User-Agent", "ProxyND/1.0 APT-Proxy")
	agent.Set("X-APT-Proxy", "ProxyND")

	// 압축 지원
	if c.Get("Accept-Encoding") == "" {
		agent.Set("Accept-Encoding", "gzip, deflate")
	}

	// 요청 실행
	statusCode, body, errs := agent.Bytes()
	if len(errs) > 0 {
		return nil, statusCode, errs[0]
	}

	if statusCode != fiber.StatusOK {
		return nil, statusCode, fmt.Errorf("upstream returned status %d", statusCode)
	}

	return body, statusCode, nil
}

// ProcessResponse 응답 처리 (APT는 일반적으로 원본 그대로 반환)
func (h *APTHandlerV3) ProcessResponse(c *fiber.Ctx, body []byte, statusCode int) ([]byte, error) {
	// APT는 특별한 응답 처리가 필요하지 않음
	return body, nil
}

// ShouldCache 캐시 정책 결정
func (h *APTHandlerV3) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	if statusCode != 200 && statusCode != 304 {
		return false
	}

	path := c.Path()

	// 메타데이터 파일은 캐시
	if h.isMetadataFile(path) {
		return true
	}

	// .deb 패키지 파일은 캐시
	if strings.HasSuffix(path, ".deb") || strings.HasSuffix(path, ".udeb") {
		return true
	}

	// 압축 파일들도 캐시
	if strings.HasSuffix(path, ".gz") || strings.HasSuffix(path, ".xz") || strings.HasSuffix(path, ".bz2") {
		return true
	}

	return false
}

// GetCacheTTL 캐시 TTL 반환
func (h *APTHandlerV3) GetCacheTTL(c *fiber.Ctx) time.Duration {
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

// GetContentType Content-Type 결정
func (h *APTHandlerV3) GetContentType(path string) string {
	switch {
	case strings.HasSuffix(path, ".deb"):
		return "application/vnd.debian.binary-package"
	case strings.HasSuffix(path, ".udeb"):
		return "application/vnd.debian.binary-package"
	case strings.HasSuffix(path, ".gz"):
		return "application/gzip"
	case strings.HasSuffix(path, ".xz"):
		return "application/x-xz"
	case strings.HasSuffix(path, ".bz2"):
		return "application/x-bzip2"
	case strings.Contains(path, "Release"):
		return "text/plain"
	case strings.Contains(path, "Packages"):
		return "text/plain"
	case strings.Contains(path, "Sources"):
		return "text/plain"
	case strings.HasSuffix(path, ".gpg"):
		return "application/pgp-signature"
	default:
		return "application/octet-stream"
	}
}

// ShouldInline 인라인 표시 여부 결정
func (h *APTHandlerV3) ShouldInline(path string) bool {
	return strings.Contains(path, "Release") ||
		strings.Contains(path, "Packages") ||
		strings.Contains(path, "Sources") ||
		strings.Contains(path, "Contents")
}

// HandleError 에러 처리
func (h *APTHandlerV3) HandleError(err error, c *fiber.Ctx) error {
	// 이미 DomainError인 경우 그대로 전송
	if _, ok := err.(*errors.DomainError); ok {
		return errors.SendProxyError(c, err)
	}

	// 에러 메시지 기반 도메인 에러 변환
	var domainErr *errors.DomainError
	errorMsg := err.Error()

	switch {
	case strings.Contains(errorMsg, "미러가 설정되지 않았습니다") || strings.Contains(errorMsg, "mirror"):
		domainErr = errors.WrapAPTError(err, "APT002", "APT 미러 서버에 접근할 수 없습니다")
	case strings.Contains(errorMsg, "패키지 형식") || strings.Contains(errorMsg, "format"):
		domainErr = errors.WrapAPTError(err, "APT003", "잘못된 APT 패키지 형식입니다")
	case strings.Contains(errorMsg, "비활성화") || strings.Contains(errorMsg, "disabled"):
		domainErr = errors.WrapAPTError(err, "APT004", "APT 프록시가 비활성화되어 있습니다")
	case strings.Contains(errorMsg, "path") || strings.Contains(errorMsg, "Invalid path"):
		domainErr = errors.WrapAPTError(err, "APT005", "잘못된 APT 패키지 경로입니다")
	case strings.Contains(errorMsg, "설정") || strings.Contains(errorMsg, "config"):
		domainErr = errors.WrapAPTError(err, "APT006", "APT 설정 파일을 읽을 수 없습니다")
	default:
		// 기본 APT 에러
		domainErr = errors.WrapAPTError(err, "APT001", "APT 패키지를 찾을 수 없습니다")
	}

	return errors.SendProxyError(c, domainErr)
}

// Handle 메인 핸들러 - Base Handler의 Template Method 사용
func (h *APTHandlerV3) Handle(c *fiber.Ctx) error {
	return h.BaseProxyHandler.Handle(h, c)
}

// 헬퍼 메서드들

// isMetadataFile 메타데이터 파일인지 확인
func (h *APTHandlerV3) isMetadataFile(path string) bool {
	return strings.Contains(path, "Release") ||
		strings.Contains(path, "Packages") ||
		strings.Contains(path, "Sources") ||
		strings.Contains(path, "Contents") ||
		strings.HasSuffix(path, "Release.gpg") ||
		strings.HasSuffix(path, "InRelease")
}

// HandleMultipleUpstreams 다중 업스트림 처리 (Base Handler에서 지원하지 않는 경우)
func (h *APTHandlerV3) HandleMultipleUpstreams(c *fiber.Ctx) error {
	osType := c.Params("osType", "ubuntu")
	proxies, exists := h.Config.Proxies[osType]
	if !exists || len(proxies) == 0 {
		return fmt.Errorf("no APT mirrors configured for %s", osType)
	}

	packagePath := c.Params("*")
	var lastErr error

	for i, mirror := range proxies {
		if mirror.URL == "" {
			continue
		}

		baseURL := strings.TrimRight(mirror.URL, "/")
		cleanPath := strings.TrimLeft(packagePath, "/")
		upstreamURL := fmt.Sprintf("%s/%s", baseURL, cleanPath)

		h.GetLogger().Debug("Trying mirror",
			logging.F("mirror_index", i),
			logging.F("url", upstreamURL),
		)

		body, statusCode, err := h.FetchFromUpstream(c, upstreamURL)
		if err != nil {
			lastErr = err
			continue
		}

		if statusCode == fiber.StatusOK {
			// 성공한 경우 응답 처리
			processedBody, err := h.ProcessResponse(c, body, statusCode)
			if err != nil {
				processedBody = body
			}

			// 캐시 저장
			if h.ShouldCache(c, statusCode) {
				requestPath := h.extractRequestPath(c)
				h.BaseProxyHandler.saveToCache(h, c, requestPath, processedBody)
			}

			return h.BaseProxyHandler.sendResponse(h, c, processedBody, h.Type(), false)
		}

		lastErr = fmt.Errorf("upstream returned status %d", statusCode)
	}

	if lastErr != nil {
		return h.HandleError(lastErr, c)
	}

	return fmt.Errorf("failed to fetch from all mirrors")
}

// extractRequestPath Base Handler에서 사용하는 메서드 오버라이드
func (h *APTHandlerV3) extractRequestPath(c *fiber.Ctx) string {
	osType := c.Params("osType", "ubuntu")
	packagePath := c.Params("*")
	return fmt.Sprintf("%s/%s", osType, packagePath)
}
