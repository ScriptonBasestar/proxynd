package proxy

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/config"
	"proxynd/internal/errors"
	"proxynd/logging"
)

// NPMHandlerV3 Base Handler 패턴을 사용하는 새로운 NPM 핸들러
type NPMHandlerV3 struct {
	*BaseProxyHandler
	Config *config.NpmProxySettings
}

// NewNPMHandlerV3 새로운 NPM 핸들러 생성
func NewNPMHandlerV3() *NPMHandlerV3 {
	return &NPMHandlerV3{
		BaseProxyHandler: NewBaseProxyHandler(),
		Config:           &config.NpmProxySettings{},
	}
}

// Type 프록시 타입 반환
func (h *NPMHandlerV3) Type() string {
	return "npm"
}

// IsEnabled 활성화 상태 확인
func (h *NPMHandlerV3) IsEnabled() bool {
	if err := h.LoadConfig(); err != nil {
		h.GetLogger().Error("Failed to read NPM config", logging.F("error", err))
		return false
	}
	return len(h.Config.Proxies) > 0
}

// LoadConfig 설정 로드
func (h *NPMHandlerV3) LoadConfig() error {
	return h.Config.ReadConfig()
}

// GenerateCacheKey 캐시 키 생성
func (h *NPMHandlerV3) GenerateCacheKey(c *fiber.Ctx) string {
	packagePath := c.Params("*")
	return fmt.Sprintf("npm:%s", strings.ReplaceAll(packagePath, "/", "_"))
}

// BuildUpstreamURL 업스트림 URL 생성 (첫 번째 레지스트리 사용)
func (h *NPMHandlerV3) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	packagePath := c.Params("*")

	// default 프록시 설정 확인
	proxies, exists := h.Config.Proxies["default"]
	if !exists || len(proxies) == 0 {
		return "", fmt.Errorf("NPM 레지스트리가 설정되지 않았습니다")
	}

	registry := proxies[0] // 첫 번째 레지스트리 사용
	if registry.URL == "" {
		return "", fmt.Errorf("NPM 레지스트리 URL이 설정되지 않았습니다")
	}

	baseURL := strings.TrimRight(registry.URL, "/")
	cleanPath := strings.TrimLeft(packagePath, "/")

	return fmt.Sprintf("%s/%s", baseURL, cleanPath), nil
}

// FetchFromUpstream 업스트림에서 데이터 가져오기 (모든 레지스트리 시도)
func (h *NPMHandlerV3) FetchFromUpstream(c *fiber.Ctx, _ string) ([]byte, int, error) {
	packagePath := c.Params("*")

	// default 프록시 설정 확인
	proxies, exists := h.Config.Proxies["default"]
	if !exists || len(proxies) == 0 {
		return nil, 0, fmt.Errorf("no NPM registries configured")
	}

	var lastErr error

	for i, registry := range proxies {
		if registry.URL == "" {
			continue
		}

		baseURL := strings.TrimRight(registry.URL, "/")
		cleanPath := strings.TrimLeft(packagePath, "/")
		upstreamURL := fmt.Sprintf("%s/%s", baseURL, cleanPath)

		h.GetLogger().Debug("Trying NPM registry",
			logging.F("registry_index", i),
			logging.F("registry_name", registry.Name),
			logging.F("url", upstreamURL),
		)

		// Fiber Agent로 요청
		agent := fiber.Get(upstreamURL)

		// 헤더 설정
		agent.Set("User-Agent", "ProxyND/1.0 NPM-Proxy")
		agent.Set("X-NPM-Proxy", "ProxyND")
		agent.Set("Accept", "application/json")

		// 레지스트리 인증 구현
		if registry.BasicAuth.Username != "" && registry.BasicAuth.Password != "" {
			h.GetLogger().Debug("Using basic auth for NPM registry",
				logging.F("registry_name", registry.Name),
				logging.F("username", registry.BasicAuth.Username),
			)
			agent.BasicAuth(registry.BasicAuth.Username, registry.BasicAuth.Password)
		}

		// 요청 실행
		statusCode, body, errs := agent.Bytes()
		if len(errs) > 0 {
			lastErr = errs[0]
			h.GetLogger().Error("Failed to fetch from registry",
				logging.F("registry_index", i),
				logging.F("registry_name", registry.Name),
				logging.F("error", lastErr),
			)
			continue
		}

		if statusCode != fiber.StatusOK {
			lastErr = fmt.Errorf("upstream returned status %d", statusCode)
			h.GetLogger().Error("Non-200 status from registry",
				logging.F("registry_index", i),
				logging.F("registry_name", registry.Name),
				logging.F("status_code", statusCode),
			)
			continue
		}

		return body, statusCode, nil
	}

	if lastErr != nil {
		return nil, 0, lastErr
	}

	return nil, 0, fmt.Errorf("package not found in any registry")
}

// ProcessResponse 응답 처리 (NPM 메타데이터 URL 재작성)
func (h *NPMHandlerV3) ProcessResponse(c *fiber.Ctx, body []byte, statusCode int) ([]byte, error) {
	requestPath := c.Params("*")

	// NPM 메타데이터 요청인지 확인하고 URL 재작성
	if h.isNpmMetadataRequest(requestPath) {
		contentType := c.Get("Content-Type")
		if strings.Contains(contentType, "json") || h.looksLikeJSON(body) {
			return h.rewriteNpmMetadata(body, c.BaseURL(), requestPath), nil
		}
	}

	return body, nil
}

// ShouldCache 캐시 정책 결정
func (h *NPMHandlerV3) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	if statusCode != 200 && statusCode != 304 {
		return false
	}

	path := c.Path()

	// 패키지 파일은 긴 TTL로 캐시
	if strings.Contains(path, ".tgz") || strings.Contains(path, ".tar.gz") {
		return true
	}

	// 메타데이터는 짧은 TTL로 캐시
	if h.isNpmMetadataRequest(path) {
		return true
	}

	return true // NPM은 대부분 캐시
}

// GetCacheTTL 캐시 TTL 반환
func (h *NPMHandlerV3) GetCacheTTL(c *fiber.Ctx) time.Duration {
	path := c.Path()

	// 패키지 파일은 긴 TTL (7일)
	if strings.Contains(path, ".tgz") || strings.Contains(path, ".tar.gz") {
		return 7 * 24 * time.Hour
	}

	// 메타데이터는 짧은 TTL (30분)
	if h.isNpmMetadataRequest(path) {
		return 30 * time.Minute
	}

	// 기본 1시간
	return 1 * time.Hour
}

// GetContentType Content-Type 결정
func (h *NPMHandlerV3) GetContentType(path string) string {
	// 메타데이터
	if h.isNpmMetadataRequest(path) {
		return "application/json; charset=utf-8"
	}

	// 패키지 파일
	switch {
	case strings.HasSuffix(path, ".tgz"):
		return MimeApplicationXGzip
	case strings.HasSuffix(path, ".tar.gz"):
		return MimeApplicationXGzip
	case strings.HasSuffix(path, ".json"):
		return MimeApplicationJSON
	default:
		return MimeApplicationOctetStream
	}
}

// ShouldInline 인라인 표시 여부 결정
func (h *NPMHandlerV3) ShouldInline(path string) bool {
	// JSON 응답은 inline
	return h.isNpmMetadataRequest(path) || strings.HasSuffix(path, ".json")
}

// HandleError 에러 처리
func (h *NPMHandlerV3) HandleError(err error, c *fiber.Ctx) error {
	// 이미 DomainError인 경우 그대로 전송
	if _, ok := err.(*errors.DomainError); ok {
		return errors.SendProxyError(c, err)
	}

	// 에러 메시지 기반 도메인 에러 변환
	var domainErr *errors.DomainError
	errorMsg := err.Error()

	switch {
	case strings.Contains(errorMsg, "레지스트리가 설정되지 않았습니다") || strings.Contains(errorMsg, "registry"):
		domainErr = errors.WrapNPMError(err, "NPM002", "NPM 레지스트리 서버에 접근할 수 없습니다")
	case strings.Contains(errorMsg, "비활성화") || strings.Contains(errorMsg, "disabled"):
		domainErr = errors.WrapNPMError(err, "NPM004", "NPM 프록시가 비활성화되어 있습니다")
	case strings.Contains(errorMsg, "path") || strings.Contains(errorMsg, "Invalid path"):
		domainErr = errors.WrapNPMError(err, "NPM005", "잘못된 패키지 경로입니다")
	case strings.Contains(errorMsg, "설정") || strings.Contains(errorMsg, "config"):
		domainErr = errors.WrapNPMError(err, "NPM006", "NPM 설정 파일을 읽을 수 없습니다")
	case strings.Contains(errorMsg, "metadata") || strings.Contains(errorMsg, "메타데이터"):
		domainErr = errors.WrapNPMError(err, "NPM007", "NPM 패키지 메타데이터가 손상되었습니다")
	case strings.Contains(errorMsg, "tarball"):
		domainErr = errors.WrapNPMError(err, "NPM008", "NPM tarball 다운로드에 실패했습니다")
	default:
		// 기본 NPM 에러
		domainErr = errors.WrapNPMError(err, "NPM001", "NPM 패키지를 찾을 수 없습니다")
	}

	return errors.SendProxyError(c, domainErr)
}

// Handle 메인 핸들러 - Base Handler의 Template Method 사용
func (h *NPMHandlerV3) Handle(c *fiber.Ctx) error {
	return h.BaseProxyHandler.Handle(h, c)
}

// 헬퍼 메서드들

// isNpmMetadataRequest NPM 메타데이터 요청인지 확인
func (h *NPMHandlerV3) isNpmMetadataRequest(path string) bool {
	// 패키지 메타데이터는 패키지명만 있거나 @scope/package 형태
	parts := strings.Split(path, "/")

	// .tgz, .tar.gz 등 패키지 파일이 아닌 경우
	if strings.Contains(path, ".tgz") || strings.Contains(path, ".tar.gz") {
		return false
	}

	// /-/ 를 포함하는 특수 경로가 아닌 경우
	if strings.Contains(path, "/-/") {
		return false
	}

	// 일반 패키지명 또는 스코프 패키지명
	return len(parts) == 1 || (len(parts) == 2 && strings.HasPrefix(parts[0], "@"))
}

// looksLikeJSON 데이터가 JSON인지 간단히 확인
func (h *NPMHandlerV3) looksLikeJSON(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	// JSON은 { 또는 [로 시작
	firstChar := data[0]
	return firstChar == '{' || firstChar == '['
}

// rewriteNpmMetadata NPM 메타데이터의 URL을 프록시 서버로 재작성
func (h *NPMHandlerV3) rewriteNpmMetadata(data []byte, baseURL, requestPath string) []byte {
	var metadata map[string]interface{}
	if err := json.Unmarshal(data, &metadata); err != nil {
		h.GetLogger().Warn("Failed to parse NPM metadata as JSON",
			logging.F("error", err),
			logging.F("request_path", requestPath),
		)
		return data
	}

	h.GetLogger().Debug("Rewriting NPM metadata URLs",
		logging.F("request_path", requestPath),
		logging.F("base_url", baseURL),
	)

	// tarball URL 재작성
	if versions, ok := metadata["versions"].(map[string]interface{}); ok {
		for _, versionData := range versions {
			if version, ok := versionData.(map[string]interface{}); ok {
				if dist, ok := version["dist"].(map[string]interface{}); ok {
					if tarball, ok := dist["tarball"].(string); ok {
						// 원본 tarball URL을 프록시 URL로 변경
						proxyURL := baseURL + "/proxy/npm/" + h.extractPackagePath(tarball)
						dist["tarball"] = proxyURL

						h.GetLogger().Debug("Rewritten tarball URL",
							logging.F("original", tarball),
							logging.F("proxy", proxyURL),
						)
					}
				}
			}
		}
	}

	rewritten, err := json.Marshal(metadata)
	if err != nil {
		h.GetLogger().Error("Failed to marshal rewritten NPM metadata",
			logging.F("error", err),
		)
		// 에러 발생 시 원본 데이터 반환
		return data
	}

	return rewritten
}

// extractPackagePath tarball URL에서 패키지 경로 추출
func (h *NPMHandlerV3) extractPackagePath(tarballURL string) string {
	// https://registry.npmjs.org/package/-/package-1.0.0.tgz 형태에서 패키지 경로 추출
	parts := strings.Split(tarballURL, "registry.npmjs.org/")
	if len(parts) > 1 {
		return parts[1]
	}

	// 다른 레지스트리의 경우 전체 URL을 그대로 사용
	return tarballURL
}
