package proxy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/helpers"
	"proxynd/internal/config"
	"proxynd/internal/errors"
	"proxynd/logging"
	"proxynd/pkg/httpclient"
)

// YumHandlerV3 Base Handler 패턴을 사용하는 새로운 YUM 핸들러
type YumHandlerV3 struct {
	*BaseProxyHandler
	Config *config.YumProxyConfig
}

// NewYumHandlerV3 새로운 YUM 핸들러 생성
func NewYumHandlerV3() *YumHandlerV3 {
	return &YumHandlerV3{
		BaseProxyHandler: NewBaseProxyHandler(),
		Config:           &config.YumProxyConfig{},
	}
}

// Type 프록시 타입 반환
func (h *YumHandlerV3) Type() string {
	return "yum"
}

// IsEnabled 활성화 상태 확인
func (h *YumHandlerV3) IsEnabled() bool {
	if err := h.LoadConfig(); err != nil {
		h.GetLogger().Error("Failed to read YUM config", logging.F("error", err))
		return false
	}
	return len(h.Config.Proxies) > 0
}

// LoadConfig 설정 로드
func (h *YumHandlerV3) LoadConfig() error {
	return h.Config.ReadConfig()
}

// GenerateCacheKey 캐시 키 생성
func (h *YumHandlerV3) GenerateCacheKey(c *fiber.Ctx) string {
	packagePath := c.Params("*")
	return fmt.Sprintf("yum:%s", strings.ReplaceAll(packagePath, "/", "_"))
}

// BuildUpstreamURL 업스트림 URL 생성 (첫 번째 미러 사용)
func (h *YumHandlerV3) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	packagePath := c.Params("*")

	if len(h.Config.Proxies) == 0 {
		return "", fmt.Errorf("YUM 미러가 설정되지 않았습니다")
	}

	mirror := h.Config.Proxies[0] // 첫 번째 미러 사용
	if mirror.URL == "" {
		return "", fmt.Errorf("YUM 미러 URL이 설정되지 않았습니다")
	}

	return helpers.JoinURL(mirror.URL, packagePath), nil
}

// FetchFromUpstream 업스트림에서 데이터 가져오기 (모든 미러 시도)
func (h *YumHandlerV3) FetchFromUpstream(c *fiber.Ctx, _ string) ([]byte, int, error) {
	packagePath := c.Params("*")
	var lastErr error

	// 컨텍스트 생성 (60초 타임아웃 - YUM은 큰 파일이 많음)
	ctx, cancel := context.WithTimeout(c.Context(), 60*time.Second)
	defer cancel()

	// HTTP 클라이언트 생성 (프록시 최적화 설정)
	proxyClient := httpclient.NewProxyClient()

	for i, mirror := range h.Config.Proxies {
		if mirror.URL == "" {
			continue
		}

		fullURL := helpers.JoinURL(mirror.URL, packagePath)

		h.GetLogger().Debug("Trying YUM mirror",
			logging.F("mirror_index", i),
			logging.F("mirror_name", mirror.Name),
			logging.F("url", fullURL),
		)

		// 컨텍스트 기반 요청 (재시도 포함)
		resp, err := proxyClient.GetWithRetry(ctx, fullURL, 2)
		if err != nil {
			lastErr = err
			h.GetLogger().Error("Failed to fetch from YUM mirror",
				logging.F("mirror_index", i),
				logging.F("mirror_name", mirror.Name),
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
		h.GetLogger().Debug("Non-200 status from YUM mirror",
			logging.F("mirror_index", i),
			logging.F("mirror_name", mirror.Name),
			logging.F("status_code", resp.StatusCode),
		)
	}

	if lastErr != nil {
		return nil, 0, lastErr
	}

	return nil, 0, fmt.Errorf("package not found in any YUM mirror")
}

// ProcessResponse 응답 처리 (YUM은 일반적으로 원본 그대로 반환)
func (h *YumHandlerV3) ProcessResponse(c *fiber.Ctx, body []byte, statusCode int) ([]byte, error) {
	// YUM은 특별한 응답 처리가 필요하지 않음
	return body, nil
}

// ShouldCache 캐시 정책 결정
func (h *YumHandlerV3) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	if statusCode != 200 && statusCode != 304 {
		return false
	}

	path := c.Path()

	// RPM 패키지 파일은 캐시
	if strings.HasSuffix(path, ".rpm") {
		return true
	}

	// 메타데이터 파일들은 짧은 TTL로 캐시
	if h.isMetadataFile(path) {
		return true
	}

	// 압축 파일들도 캐시
	if h.isCompressedFile(path) {
		return true
	}

	// GPG 서명 파일들도 캐시
	if strings.HasSuffix(path, ".asc") || strings.HasSuffix(path, ".gpg") {
		return true
	}

	return false
}

// GetCacheTTL 캐시 TTL 반환
func (h *YumHandlerV3) GetCacheTTL(c *fiber.Ctx) time.Duration {
	path := c.Path()

	// repomd.xml 파일은 짧은 TTL (10분)
	if strings.Contains(path, "repomd.xml") {
		return 10 * time.Minute
	}

	// 메타데이터 파일들은 중간 TTL (30분)
	if h.isMetadataFile(path) {
		return 30 * time.Minute
	}

	// RPM 패키지 파일은 긴 TTL (30일)
	if strings.HasSuffix(path, ".rpm") {
		return 30 * 24 * time.Hour
	}

	// 압축된 메타데이터는 중간 TTL (1시간)
	if h.isCompressedFile(path) {
		return 1 * time.Hour
	}

	// GPG 서명 파일들은 하루
	if strings.HasSuffix(path, ".asc") || strings.HasSuffix(path, ".gpg") {
		return 24 * time.Hour
	}

	// 기본 1시간
	return 1 * time.Hour
}

// GetContentType Content-Type 결정
func (h *YumHandlerV3) GetContentType(path string) string {
	switch {
	case strings.HasSuffix(path, ".rpm"):
		return "application/x-rpm"
	case strings.HasSuffix(path, ".xml") || strings.HasSuffix(path, ".xml.gz"):
		return "application/xml"
	case strings.HasSuffix(path, ".xml.bz2") || strings.HasSuffix(path, ".xml.xz"):
		return "application/xml"
	case strings.HasSuffix(path, ".sqlite") || strings.HasSuffix(path, ".sqlite.bz2"):
		return "application/octet-stream"
	case strings.HasSuffix(path, ".sqlite.gz") || strings.HasSuffix(path, ".sqlite.xz"):
		return "application/octet-stream"
	case strings.HasSuffix(path, ".asc") || strings.HasSuffix(path, ".gpg"):
		return "application/pgp-signature"
	case strings.Contains(path, "repomd.xml"):
		return "text/xml"
	default:
		return "application/octet-stream"
	}
}

// ShouldInline 인라인 표시 여부 결정
func (h *YumHandlerV3) ShouldInline(path string) bool {
	inlineExtensions := []string{".xml", ".txt", ".asc", ".gpg"}
	for _, ext := range inlineExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

// HandleError 에러 처리
func (h *YumHandlerV3) HandleError(err error, c *fiber.Ctx) error {
	// 이미 DomainError인 경우 그대로 전송
	if _, ok := err.(*errors.DomainError); ok {
		return errors.SendProxyError(c, err)
	}

	// 에러 메시지 기반 도메인 에러 변환
	var domainErr *errors.DomainError
	errorMsg := err.Error()

	switch {
	case strings.Contains(errorMsg, "미러가 설정되지 않았습니다") || strings.Contains(errorMsg, "mirror"):
		domainErr = errors.WrapYumError(err, "YUM002", "YUM 미러 서버에 접근할 수 없습니다")
	case strings.Contains(errorMsg, "rpm") || strings.Contains(errorMsg, "RPM"):
		domainErr = errors.WrapYumError(err, "YUM003", "YUM RPM 파일이 손상되었습니다")
	case strings.Contains(errorMsg, "context deadline exceeded") || strings.Contains(errorMsg, "타임아웃"):
		domainErr = errors.WrapYumError(err, "YUM004", "YUM 서버 응답 타임아웃")
	case strings.Contains(errorMsg, "path") || strings.Contains(errorMsg, "Invalid path"):
		domainErr = errors.WrapYumError(err, "YUM005", "잘못된 패키지 경로입니다")
	case strings.Contains(errorMsg, "설정") || strings.Contains(errorMsg, "config"):
		domainErr = errors.WrapYumError(err, "YUM006", "YUM 설정 파일을 읽을 수 없습니다")
	case strings.Contains(errorMsg, "metadata") || strings.Contains(errorMsg, "메타데이터"):
		domainErr = errors.WrapYumError(err, "YUM007", "YUM 메타데이터가 손상되었습니다")
	case strings.Contains(errorMsg, "signature") || strings.Contains(errorMsg, "서명"):
		domainErr = errors.WrapYumError(err, "YUM008", "YUM 패키지 서명 검증에 실패했습니다")
	case strings.Contains(errorMsg, "비활성화") || strings.Contains(errorMsg, "disabled"):
		domainErr = errors.WrapYumError(err, "YUM009", "YUM 프록시가 비활성화되어 있습니다")
	case strings.Contains(errorMsg, "repomd") || strings.Contains(errorMsg, "repomd.xml"):
		domainErr = errors.WrapYumError(err, "YUM010", "YUM repomd.xml을 찾을 수 없습니다")
	default:
		// 기본 YUM 에러
		domainErr = errors.WrapYumError(err, "YUM001", "YUM 패키지를 찾을 수 없습니다")
	}

	return errors.SendProxyError(c, domainErr)
}

// Handle 메인 핸들러 - Base Handler의 Template Method 사용
func (h *YumHandlerV3) Handle(c *fiber.Ctx) error {
	return h.BaseProxyHandler.Handle(h, c)
}

// 헬퍼 메서드들

// isMetadataFile 메타데이터 파일인지 확인
func (h *YumHandlerV3) isMetadataFile(path string) bool {
	return strings.Contains(path, "repomd.xml") ||
		strings.Contains(path, "primary.xml") ||
		strings.Contains(path, "filelists.xml") ||
		strings.Contains(path, "other.xml") ||
		strings.Contains(path, "updateinfo.xml") ||
		strings.Contains(path, "comps.xml")
}

// isCompressedFile 압축 파일인지 확인
func (h *YumHandlerV3) isCompressedFile(path string) bool {
	return strings.HasSuffix(path, ".gz") ||
		strings.HasSuffix(path, ".bz2") ||
		strings.HasSuffix(path, ".xz") ||
		strings.HasSuffix(path, ".sqlite.gz") ||
		strings.HasSuffix(path, ".sqlite.bz2") ||
		strings.HasSuffix(path, ".sqlite.xz")
}
