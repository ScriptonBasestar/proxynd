package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/config"
	"proxynd/internal/errors"
	"proxynd/internal/helpers"
	"proxynd/internal/logging"
)

// DockerHandlerV3 Base Handler 패턴을 사용하는 새로운 Docker 핸들러
type DockerHandlerV3 struct {
	*BaseProxyHandler
	Config *config.DockerProxySettings
}

// NewDockerHandlerV3 새로운 Docker 핸들러 생성
func NewDockerHandlerV3() *DockerHandlerV3 {
	return &DockerHandlerV3{
		BaseProxyHandler: NewBaseProxyHandler(),
		Config:           &config.DockerProxySettings{},
	}
}

// Type 프록시 타입 반환
func (h *DockerHandlerV3) Type() string {
	return "docker"
}

// IsEnabled 활성화 상태 확인
func (h *DockerHandlerV3) IsEnabled() bool {
	if err := h.LoadConfig(); err != nil {
		h.GetLogger().Error("Failed to read Docker config", logging.F("error", err))
		return false
	}
	return len(h.Config.Proxies) > 0
}

// LoadConfig 설정 로드
func (h *DockerHandlerV3) LoadConfig() error {
	return h.Config.ReadConfig()
}

// GenerateCacheKey 캐시 키 생성
func (h *DockerHandlerV3) GenerateCacheKey(c *fiber.Ctx) string {
	imagePath := c.Params("*")
	return fmt.Sprintf("docker:%s", strings.ReplaceAll(imagePath, "/", "_"))
}

// BuildUpstreamURL 업스트림 URL 생성 (첫 번째 레지스트리 사용)
func (h *DockerHandlerV3) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	imagePath := c.Params("*")

	if len(h.Config.Proxies) == 0 {
		return "", fmt.Errorf("docker 레지스트리가 설정되지 않았습니다")
	}

	registry := h.Config.Proxies[0] // 첫 번째 레지스트리 사용
	if registry.URL == "" {
		return "", fmt.Errorf("docker 레지스트리 URL이 설정되지 않았습니다")
	}

	// Docker Registry v2 API 경로 처리
	if imagePath == "v2" || imagePath == DockerAPIV2Path {
		return helpers.JoinURL(registry.URL, "v2/"), nil
	}

	// v2 prefix 추가 (없는 경우)
	if !strings.HasPrefix(imagePath, "v2/") {
		imagePath = "v2/" + imagePath
	}

	return helpers.JoinURL(registry.URL, imagePath), nil
}

// FetchFromUpstream 업스트림에서 데이터 가져오기 (모든 레지스트리 시도)
func (h *DockerHandlerV3) FetchFromUpstream(c *fiber.Ctx, _ string) ([]byte, int, error) {
	imagePath := c.Params("*")
	var lastErr error

	for i, registry := range h.Config.Proxies {
		if registry.URL == "" {
			continue
		}

		// Docker Registry URL 구성
		upstreamURL := h.buildDockerURL(registry.URL, imagePath)

		h.GetLogger().Debug("Trying Docker registry",
			logging.F("registry_index", i),
			logging.F("registry_name", registry.Name),
			logging.F("url", upstreamURL),
		)

		// HTTP 요청 생성
		req, err := http.NewRequest("GET", upstreamURL, nil)
		if err != nil {
			lastErr = err
			continue
		}

		// Docker 관련 헤더 복사
		h.copyDockerHeaders(c, req)

		// 인증 처리
		if registry.Auth.Username != "" && registry.Auth.Password != "" {
			req.SetBasicAuth(registry.Auth.Username, registry.Auth.Password)
		}

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			h.GetLogger().Error("Failed to fetch from Docker registry",
				logging.F("registry_index", i),
				logging.F("registry_name", registry.Name),
				logging.F("error", err),
			)
			continue
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode == http.StatusOK {
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

			// Docker 특별 헤더들을 컨텍스트에 저장
			h.storeDockerHeadersInContext(c, resp.Header)

			return body, resp.StatusCode, nil
		} else if resp.StatusCode == http.StatusUnauthorized {
			// 인증 챌린지의 경우 헤더 정보 보존
			h.copyResponseHeaders(resp.Header, c)
			return nil, resp.StatusCode, fmt.Errorf("authentication required")
		}

		lastErr = fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}

	if lastErr != nil {
		return nil, 0, lastErr
	}

	return nil, 0, fmt.Errorf("image not found in any registry")
}

// ProcessResponse 응답 처리 (Docker 매니페스트 및 메타데이터 처리)
func (h *DockerHandlerV3) ProcessResponse(c *fiber.Ctx, body []byte, statusCode int) ([]byte, error) {
	imagePath := c.Params("*")

	// v2 base endpoint 처리
	if imagePath == "v2" || imagePath == DockerAPIV2Path {
		response := map[string]interface{}{
			"errors": []interface{}{},
		}
		c.Set("Docker-Distribution-Api-Version", "registry/2.0")
		processedBody, err := json.Marshal(response)
		if err != nil {
			return body, nil // 실패시 원본 반환
		}
		return processedBody, nil
	}

	// 매니페스트 요청인 경우 Content-Type 자동 감지
	if h.isManifestRequest(imagePath) {
		contentType := h.detectManifestType(body)
		c.Set("Content-Type", contentType)
	}

	// Blob 요청인 경우
	if h.isBlobRequest(imagePath) {
		c.Set("Content-Type", "application/octet-stream")
	}

	// Docker-Content-Digest 헤더 설정
	if digest := h.extractDigest(imagePath); digest != "" {
		c.Set("Docker-Content-Digest", digest)
	}

	return body, nil
}

// ShouldCache 캐시 정책 결정
func (h *DockerHandlerV3) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	if statusCode != 200 && statusCode != 304 {
		return false
	}

	path := c.Path()

	// v2 base endpoint는 캐시하지 않음
	if path == "v2" || path == DockerAPIV2Path {
		return false
	}

	// 매니페스트는 짧은 TTL로 캐시
	if h.isManifestRequest(path) {
		return true
	}

	// Blob(이미지 레이어)는 긴 TTL로 캐시
	if h.isBlobRequest(path) {
		return true
	}

	// 태그 리스트 등은 짧은 TTL로 캐시
	if strings.Contains(path, "/tags/list") {
		return true
	}

	return false
}

// GetCacheTTL 캐시 TTL 반환
func (h *DockerHandlerV3) GetCacheTTL(c *fiber.Ctx) time.Duration {
	path := c.Path()

	// 매니페스트는 짧은 TTL (10분)
	if h.isManifestRequest(path) {
		return 10 * time.Minute
	}

	// Blob(이미지 레이어)는 긴 TTL (30일)
	if h.isBlobRequest(path) {
		return 30 * 24 * time.Hour
	}

	// 태그 리스트는 짧은 TTL (5분)
	if strings.Contains(path, "/tags/list") {
		return 5 * time.Minute
	}

	// 기본 1시간
	return 1 * time.Hour
}

// GetContentType Content-Type 결정
func (h *DockerHandlerV3) GetContentType(path string) string {
	if path == "v2" || path == DockerAPIV2Path {
		return MimeApplicationJSON
	}

	if h.isManifestRequest(path) {
		return MimeApplicationDockerManifestV2JSON
	}

	if h.isBlobRequest(path) {
		return MimeApplicationOctetStream
	}

	if strings.Contains(path, "/tags/list") {
		return MimeApplicationJSON
	}

	return MimeApplicationJSON
}

// ShouldInline 인라인 표시 여부 결정
func (h *DockerHandlerV3) ShouldInline(path string) bool {
	return h.isManifestRequest(path) ||
		strings.Contains(path, "/tags/list") ||
		path == "v2" || path == "v2/"
}

// HandleError 에러 처리
func (h *DockerHandlerV3) HandleError(err error, c *fiber.Ctx) error {
	// 이미 DomainError인 경우 그대로 전송
	if _, ok := err.(*errors.DomainError); ok {
		return errors.SendProxyError(c, err)
	}

	// 에러 메시지 기반 도메인 에러 변환
	var domainErr *errors.DomainError
	errorMsg := err.Error()

	switch {
	case strings.Contains(errorMsg, "레지스트리가 설정되지 않았습니다") || strings.Contains(errorMsg, "registry"):
		domainErr = errors.WrapDockerError(err, "DOCKER002", "Docker 레지스트리 서버에 접근할 수 없습니다")
	case strings.Contains(errorMsg, "authentication required") || strings.Contains(errorMsg, "인증"):
		domainErr = errors.WrapDockerError(err, "DOCKER003", "Docker 레지스트리 인증이 필요합니다")
	case strings.Contains(errorMsg, "비활성화") || strings.Contains(errorMsg, "disabled"):
		domainErr = errors.WrapDockerError(err, "DOCKER004", "Docker 프록시가 비활성화되어 있습니다")
	case strings.Contains(errorMsg, "path") || strings.Contains(errorMsg, "Invalid path"):
		domainErr = errors.WrapDockerError(err, "DOCKER005", "잘못된 이미지 경로입니다")
	case strings.Contains(errorMsg, "설정") || strings.Contains(errorMsg, "config"):
		domainErr = errors.WrapDockerError(err, "DOCKER006", "Docker 설정 파일을 읽을 수 없습니다")
	case strings.Contains(errorMsg, "manifest") || strings.Contains(errorMsg, "매니페스트"):
		domainErr = errors.WrapDockerError(err, "DOCKER007", "Docker 매니페스트가 손상되었습니다")
	case strings.Contains(errorMsg, "blob"):
		domainErr = errors.WrapDockerError(err, "DOCKER008", "Docker blob 다운로드에 실패했습니다")
	case strings.Contains(errorMsg, "digest") || strings.Contains(errorMsg, "다이제스트"):
		domainErr = errors.WrapDockerError(err, "DOCKER009", "Docker 이미지 다이제스트가 일치하지 않습니다")
	default:
		// 기본 Docker 에러
		domainErr = errors.WrapDockerError(err, "DOCKER001", "Docker 이미지를 찾을 수 없습니다")
	}

	return errors.SendProxyError(c, domainErr)
}

// Handle 메인 핸들러 - Base Handler의 Template Method 사용
func (h *DockerHandlerV3) Handle(c *fiber.Ctx) error {
	return h.BaseProxyHandler.Handle(h, c)
}

// 헬퍼 메서드들

// buildDockerURL Docker Registry URL 구성
func (h *DockerHandlerV3) buildDockerURL(serverURL, requestPath string) string {
	// v2 base endpoint 처리
	if requestPath == "v2" || requestPath == "v2/" {
		return helpers.JoinURL(serverURL, "v2/")
	}

	// v2 prefix 추가 (없는 경우)
	if !strings.HasPrefix(requestPath, "v2/") {
		requestPath = "v2/" + requestPath
	}

	return helpers.JoinURL(serverURL, requestPath)
}

// copyDockerHeaders 클라이언트 요청 헤더를 프록시 요청으로 복사
func (h *DockerHandlerV3) copyDockerHeaders(c *fiber.Ctx, req *http.Request) {
	// Accept 헤더
	if accept := c.Get("Accept"); accept != "" {
		req.Header.Set("Accept", accept)
	}

	// Docker 관련 헤더들
	dockerHeaders := []string{
		"Docker-Distribution-Api-Version",
		"Authorization",
		"User-Agent",
	}

	for _, header := range dockerHeaders {
		if value := c.Get(header); value != "" {
			req.Header.Set(header, value)
		}
	}
}

// copyResponseHeaders 응답 헤더 복사
func (h *DockerHandlerV3) copyResponseHeaders(from http.Header, to *fiber.Ctx) {
	for key, values := range from {
		if len(values) > 0 {
			to.Set(key, values[0])
		}
	}
}

// storeDockerHeadersInContext Docker 특별 헤더들을 컨텍스트에 저장
func (h *DockerHandlerV3) storeDockerHeadersInContext(c *fiber.Ctx, headers http.Header) {
	// Docker 관련 중요 헤더들 저장
	dockerHeaders := []string{
		"Docker-Content-Digest",
		"Docker-Distribution-Api-Version",
		"Content-Type",
		"Content-Length",
	}

	for _, header := range dockerHeaders {
		if value := headers.Get(header); value != "" {
			c.Set(header, value)
		}
	}
}

// isManifestRequest 매니페스트 요청인지 확인
func (h *DockerHandlerV3) isManifestRequest(path string) bool {
	return strings.Contains(path, "/manifests/")
}

// isBlobRequest Blob 요청인지 확인
func (h *DockerHandlerV3) isBlobRequest(path string) bool {
	return strings.Contains(path, "/blobs/")
}

// detectManifestType 매니페스트 타입 감지
func (h *DockerHandlerV3) detectManifestType(data []byte) string {
	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return MimeApplicationDockerManifestV2JSON
	}

	if schemaVersion, ok := manifest["schemaVersion"].(float64); ok {
		if schemaVersion == 1 {
			return "application/vnd.docker.distribution.manifest.v1+json"
		}

		// v2 매니페스트 타입 구분
		if _, ok := manifest["manifests"]; ok {
			return "application/vnd.docker.distribution.manifest.list.v2+json"
		}
	}

	return "application/vnd.docker.distribution.manifest.v2+json"
}

// extractDigest 경로에서 다이제스트 추출
func (h *DockerHandlerV3) extractDigest(path string) string {
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, "sha256:") {
			return part
		}
	}
	return ""
}
