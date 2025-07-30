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

// PipHandlerV3 Base Handler 패턴을 사용하는 새로운 PIP 핸들러
type PipHandlerV3 struct {
	*BaseProxyHandler
	Config *config.PipProxyConfig
}

// NewPipHandlerV3 새로운 PIP 핸들러 생성
func NewPipHandlerV3() *PipHandlerV3 {
	return &PipHandlerV3{
		BaseProxyHandler: NewBaseProxyHandler(),
		Config:           &config.PipProxyConfig{},
	}
}

// Type 프록시 타입 반환
func (h *PipHandlerV3) Type() string {
	return "pip"
}

// IsEnabled 활성화 상태 확인
func (h *PipHandlerV3) IsEnabled() bool {
	if err := h.LoadConfig(); err != nil {
		h.GetLogger().Error("Failed to read PIP config", logging.F("error", err))
		return false
	}
	return len(h.Config.Proxies) > 0
}

// LoadConfig 설정 로드
func (h *PipHandlerV3) LoadConfig() error {
	return h.Config.ReadConfig()
}

// GenerateCacheKey 캐시 키 생성
func (h *PipHandlerV3) GenerateCacheKey(c *fiber.Ctx) string {
	packagePath := c.Params("*")
	return fmt.Sprintf("pip:%s", strings.ReplaceAll(packagePath, "/", "_"))
}

// BuildUpstreamURL 업스트림 URL 생성 (첫 번째 PyPI 서버 사용)
func (h *PipHandlerV3) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	packagePath := c.Params("*")

	if len(h.Config.Proxies) == 0 {
		return "", fmt.Errorf("PIP 미러가 설정되지 않았습니다")
	}

	server := h.Config.Proxies[0] // 첫 번째 서버 사용
	if server.URL == "" {
		return "", fmt.Errorf("PIP 미러 URL이 설정되지 않았습니다")
	}

	return h.buildPipURL(server.URL, packagePath), nil
}

// FetchFromUpstream 업스트림에서 데이터 가져오기 (모든 PyPI 서버 시도)
func (h *PipHandlerV3) FetchFromUpstream(c *fiber.Ctx, _ string) ([]byte, int, error) {
	packagePath := c.Params("*")
	var lastErr error

	// 컨텍스트 생성 (45초 타임아웃)
	ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
	defer cancel()

	// HTTP 클라이언트 생성 (프록시 최적화 설정)
	proxyClient := httpclient.NewProxyClient()

	for i, server := range h.Config.Proxies {
		if server.URL == "" {
			continue
		}

		fullURL := h.buildPipURL(server.URL, packagePath)

		h.GetLogger().Debug("Trying PIP server",
			logging.F("server_index", i),
			logging.F("server_name", server.Name),
			logging.F("url", fullURL),
		)

		// 컨텍스트 기반 요청 (재시도 포함)
		resp, err := proxyClient.GetWithRetry(ctx, fullURL, 2)
		if err != nil {
			lastErr = err
			h.GetLogger().Error("Failed to fetch from PIP server",
				logging.F("server_index", i),
				logging.F("server_name", server.Name),
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
	}

	if lastErr != nil {
		return nil, 0, lastErr
	}

	return nil, 0, fmt.Errorf("package not found in any PyPI server")
}

// ProcessResponse 응답 처리 (PIP는 일반적으로 원본 그대로 반환)
func (h *PipHandlerV3) ProcessResponse(c *fiber.Ctx, body []byte, statusCode int) ([]byte, error) {
	// PIP는 특별한 응답 처리가 필요하지 않음
	return body, nil
}

// ShouldCache 캐시 정책 결정
func (h *PipHandlerV3) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	if statusCode != 200 && statusCode != 304 {
		return false
	}

	path := c.Path()

	// simple API HTML 응답은 짧은 TTL로 캐시
	if strings.Contains(path, "/simple/") {
		return true
	}

	// JSON API 응답은 캐시
	if strings.Contains(path, "/json") {
		return true
	}

	// 패키지 파일들은 캐시
	if h.isPackageFile(path) {
		return true
	}

	return false
}

// GetCacheTTL 캐시 TTL 반환
func (h *PipHandlerV3) GetCacheTTL(c *fiber.Ctx) time.Duration {
	path := c.Path()

	// simple API HTML 응답은 짧은 TTL (30분)
	if strings.Contains(path, "/simple/") {
		return 30 * time.Minute
	}

	// JSON API 응답은 중간 TTL (1시간)
	if strings.Contains(path, "/json") {
		return 1 * time.Hour
	}

	// 패키지 파일들은 긴 TTL (30일)
	if h.isPackageFile(path) {
		return 30 * 24 * time.Hour
	}

	// 기본 1시간
	return 1 * time.Hour
}

// GetContentType Content-Type 결정
func (h *PipHandlerV3) GetContentType(path string) string {
	filename := h.extractFilename(path)

	// API 응답
	if strings.Contains(path, "/simple/") && !strings.Contains(filename, ".") {
		return "text/html; charset=utf-8"
	}

	// JSON API
	if strings.Contains(path, "/json") {
		return "application/json"
	}

	// 패키지 파일
	switch {
	case strings.HasSuffix(filename, ".whl"):
		return "application/zip"
	case strings.HasSuffix(filename, ".tar.gz"):
		return "application/x-gzip"
	case strings.HasSuffix(filename, ".tar.bz2"):
		return "application/x-bzip2"
	case strings.HasSuffix(filename, ".zip"):
		return "application/zip"
	case strings.HasSuffix(filename, ".egg"):
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}

// ShouldInline 인라인 표시 여부 결정
func (h *PipHandlerV3) ShouldInline(path string) bool {
	contentType := h.GetContentType(path)
	// JSON API와 HTML 응답은 inline
	return strings.Contains(contentType, "json") || strings.Contains(contentType, "html")
}

// HandleError 에러 처리
func (h *PipHandlerV3) HandleError(err error, c *fiber.Ctx) error {
	// 이미 DomainError인 경우 그대로 전송
	if _, ok := err.(*errors.DomainError); ok {
		return errors.SendProxyError(c, err)
	}

	// 에러 메시지 기반 도메인 에러 변환
	var domainErr *errors.DomainError
	errorMsg := err.Error()

	switch {
	case strings.Contains(errorMsg, "미러가 설정되지 않았습니다") || strings.Contains(errorMsg, "mirror"):
		domainErr = errors.WrapPipError(err, "PIP002", "PIP 미러 서버에 접근할 수 없습니다")
	case strings.Contains(errorMsg, "context deadline exceeded") || strings.Contains(errorMsg, "타임아웃"):
		domainErr = errors.WrapPipError(err, "PIP004", "PIP 서버 응답 타임아웃")
	case strings.Contains(errorMsg, "path") || strings.Contains(errorMsg, "Invalid path"):
		domainErr = errors.WrapPipError(err, "PIP005", "잘못된 패키지 경로입니다")
	case strings.Contains(errorMsg, "설정") || strings.Contains(errorMsg, "config"):
		domainErr = errors.WrapPipError(err, "PIP006", "PIP 설정 파일을 읽을 수 없습니다")
	case strings.Contains(errorMsg, "wheel") || strings.Contains(errorMsg, "휴"):
		domainErr = errors.WrapPipError(err, "PIP007", "PIP wheel 파일이 손상되었습니다")
	case strings.Contains(errorMsg, "index") || strings.Contains(errorMsg, "인덱스"):
		domainErr = errors.WrapPipError(err, "PIP008", "PIP 패키지 인덱스가 손상되었습니다")
	case strings.Contains(errorMsg, "비활성화") || strings.Contains(errorMsg, "disabled"):
		domainErr = errors.WrapPipError(err, "PIP009", "PIP 프록시가 비활성화되어 있습니다")
	default:
		// 기본 PIP 에러
		domainErr = errors.WrapPipError(err, "PIP001", "PIP 패키지를 찾을 수 없습니다")
	}

	return errors.SendProxyError(c, domainErr)
}

// Handle 메인 핸들러 - Base Handler의 Template Method 사용
func (h *PipHandlerV3) Handle(c *fiber.Ctx) error {
	return h.BaseProxyHandler.Handle(h, c)
}

// 헬퍼 메서드들

// buildPipURL PyPI URL 구성
func (h *PipHandlerV3) buildPipURL(serverURL, requestPath string) string {
	// /simple/ 경로 처리
	if strings.HasPrefix(requestPath, "simple/") {
		return helpers.JoinURL(serverURL, requestPath)
	}

	// /packages/ 경로 처리
	if strings.HasPrefix(requestPath, "packages/") {
		return helpers.JoinURL(serverURL, requestPath)
	}

	// 기본적으로 simple API 사용
	return helpers.JoinURL(serverURL, "simple", requestPath)
}

// isPackageFile 패키지 파일인지 확인
func (h *PipHandlerV3) isPackageFile(path string) bool {
	filename := h.extractFilename(path)
	return strings.HasSuffix(filename, ".whl") ||
		strings.HasSuffix(filename, ".tar.gz") ||
		strings.HasSuffix(filename, ".tar.bz2") ||
		strings.HasSuffix(filename, ".zip") ||
		strings.HasSuffix(filename, ".egg")
}

// extractFilename 경로에서 파일명 추출
func (h *PipHandlerV3) extractFilename(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}
