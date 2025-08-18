package containerhandlers

import (
	"encoding/json"
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
	"proxynd/internal/config"
	"proxynd/internal/container"
	"proxynd/internal/errors"
	"proxynd/internal/logging"
	"proxynd/internal/security"
)

// DockerContainerHandler Container 기반 Docker 핸들러
type DockerContainerHandler struct {
	logger            logging.Logger
	name              string
	proxyType         string
	dockerConfig      *config.DockerProxySettings
	storageDir        string
	serverIdx         int32 // 라운드로빈을 위한 atomic counter
	containerProvider container.ContainerProvider
}

// NewDockerContainerHandler 새로운 Container 기반 Docker 핸들러 생성
func NewDockerContainerHandler(provider container.ContainerProvider) *DockerContainerHandler {
	return &DockerContainerHandler{
		logger:            logging.GetLogger(),
		name:              "docker-container-handler",
		proxyType:         "docker",
		dockerConfig:      &config.DockerProxySettings{},
		serverIdx:         0,
		containerProvider: provider,
		storageDir:        provider.GetStorageDir(),
	}
}

// Handle Docker 레지스트리 프록시 요청 처리
func (h *DockerContainerHandler) Handle(c *fiber.Ctx) error {
	startTime := time.Now()

	h.logger.Info("Docker Container request started",
		logging.F("path", c.Path()),
		logging.F("method", c.Method()),
	)

	defer func() {
		h.logger.Info("Docker Container request completed",
			logging.F("duration_ms", time.Since(startTime).Milliseconds()),
		)
	}()

	// 1. 활성화 상태 확인
	if !h.IsEnabled() {
		return errors.NewError("PROXY001", "Docker 프록시가 비활성화되어 있습니다").
			WithDomain("docker").
			Build()
	}

	requestPath := c.Params("*")
	h.logger.Debug("Docker request", logging.F("path", requestPath))

	// 2. Docker Registry v2 API 라우팅
	if requestPath == "v2" || requestPath == "v2/" {
		return h.handleDockerV2Base(c)
	}

	// 3. 캐시에서 파일 확인
	if cached, err := h.serveFromCache(c, requestPath); err == nil && cached {
		return nil
	}

	// 4. 업스트림에서 가져오기
	return h.fetchFromUpstream(c, requestPath)
}

// Name 핸들러 이름 반환
func (h *DockerContainerHandler) Name() string {
	return h.name
}

// Type 프록시 타입 반환
func (h *DockerContainerHandler) Type() string {
	return h.proxyType
}

// SetContainer Container Provider 설정
func (h *DockerContainerHandler) SetContainer(provider container.ContainerProvider) {
	h.containerProvider = provider
}

// GetContainer Container Provider 반환
func (h *DockerContainerHandler) GetContainer() container.ContainerProvider {
	return h.containerProvider
}

// LoadConfig Container를 통한 설정 로드
func (h *DockerContainerHandler) LoadConfig() error {
	if h.containerProvider == nil {
		return fmt.Errorf("container provider not initialized")
	}

	config, err := h.containerProvider.GetDockerProxyConfig()
	if err != nil {
		return fmt.Errorf("failed to load Docker config: %w", err)
	}

	h.dockerConfig = config
	return nil
}

// ReloadConfig 설정 다시 로드
func (h *DockerContainerHandler) ReloadConfig() error {
	return h.LoadConfig()
}

// IsEnabled 활성화 상태 확인 (Container 기반)
func (h *DockerContainerHandler) IsEnabled() bool {
	if err := h.LoadConfig(); err != nil {
		h.logger.Error("Failed to load Docker config", logging.F("error", err))
		return false
	}
	return len(h.dockerConfig.Proxies) > 0
}

// HealthCheck 헬스체크
func (h *DockerContainerHandler) HealthCheck() error {
	if h.containerProvider == nil {
		return fmt.Errorf("container provider not initialized")
	}
	return nil
}

// GenerateCacheKey Docker 전용 캐시 키 생성
func (h *DockerContainerHandler) GenerateCacheKey(c *fiber.Ctx) string {
	path := c.Params("*")
	return fmt.Sprintf("docker:%s", strings.ReplaceAll(path, "/", "_"))
}

// BuildUpstreamURL 업스트림 URL 구성 (라운드로빈 지원)
func (h *DockerContainerHandler) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	if err := h.LoadConfig(); err != nil {
		return "", fmt.Errorf("docker 설정 로드 실패: %w", err)
	}

	requestPath := c.Params("*")

	if len(h.dockerConfig.Proxies) == 0 {
		return "", fmt.Errorf("docker 프록시 서버가 설정되지 않았습니다")
	}

	// 라운드로빈으로 서버 선택
	idx := atomic.AddInt32(&h.serverIdx, 1) - 1
	server := h.dockerConfig.Proxies[idx%int32(len(h.dockerConfig.Proxies))]

	if server.URL == "" {
		return "", fmt.Errorf("docker 프록시 URL이 설정되지 않았습니다")
	}

	return h.buildDockerURL(server.URL, requestPath), nil
}

// IsCacheable 캐시 가능 여부 확인
func (h *DockerContainerHandler) IsCacheable(c *fiber.Ctx) bool {
	return c.Method() == fiber.MethodGet
}

// GetCacheKey 캐시 키 반환
func (h *DockerContainerHandler) GetCacheKey(c *fiber.Ctx) string {
	return h.GenerateCacheKey(c)
}

// ShouldCache Docker 캐싱 정책
func (h *DockerContainerHandler) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	// 200 OK만 캐시
	if statusCode != fiber.StatusOK {
		return false
	}

	// 매니페스트와 블롭만 캐시
	path := c.Params("*")
	return strings.Contains(path, "/manifests/") || strings.Contains(path, "/blobs/")
}

// GetCacheTTL Docker 캐시 TTL 설정
func (h *DockerContainerHandler) GetCacheTTL(c *fiber.Ctx) time.Duration {
	path := c.Params("*")

	// 매니페스트는 짧은 TTL
	if strings.Contains(path, "/manifests/") {
		return 30 * time.Minute
	}

	// 블롭은 긴 TTL
	if strings.Contains(path, "/blobs/") {
		return 24 * time.Hour
	}

	// 기본 TTL
	return 2 * time.Hour
}

// handleDockerV2Base Docker Registry v2 베이스 엔드포인트 처리
func (h *DockerContainerHandler) handleDockerV2Base(c *fiber.Ctx) error {
	// Docker Registry v2 API 응답
	response := map[string]interface{}{
		"errors": []interface{}{},
	}

	c.Set("Docker-Distribution-Api-Version", "registry/2.0")
	c.Set("X-Cache-Status", "GENERATED")
	c.Set("X-Proxy-Type", "docker")
	c.Set("X-Handler", "container")

	return c.JSON(response)
}

// serveFromCache 캐시에서 파일 제공
func (h *DockerContainerHandler) serveFromCache(c *fiber.Ctx, requestPath string) (bool, error) {
	// 보안 경로 조합
	safeBasePath := filepath.Join(h.storageDir, h.dockerConfig.Path)
	safePath, err := security.SafeJoinPath(safeBasePath, requestPath)
	if err != nil {
		return false, fmt.Errorf("invalid path: %w", err)
	}

	// 파일 존재 확인
	fileInfo, err := os.Stat(safePath)
	if err != nil || fileInfo.IsDir() {
		return false, nil
	}

	h.logger.Debug("Serving from cache", logging.F("path", safePath))

	// 파일 읽기
	bytes, err := os.ReadFile(safePath)
	if err != nil {
		return false, err
	}

	// 매니페스트와 블롭 타입 확인
	isManifest := strings.Contains(requestPath, "/manifests/")
	isBlob := strings.Contains(requestPath, "/blobs/")

	// 매니페스트의 경우 저장된 헤더도 읽기
	if isManifest {
		if headers, err := h.loadDockerHeaders(safePath + ".headers"); err == nil {
			h.copyResponseHeaders(headers, c)
		}
	}

	// Content-Type 설정
	if isManifest {
		contentType := h.detectManifestType(bytes)
		c.Set("Content-Type", contentType)
	} else if isBlob {
		c.Set("Content-Type", "application/octet-stream")
	}

	// Docker-Content-Digest 헤더 설정
	if digest := h.extractDigest(requestPath); digest != "" {
		c.Set("Docker-Content-Digest", digest)
	}

	c.Set("X-Cache-Status", "HIT")
	c.Set("X-Proxy-Type", "docker")
	c.Set("X-Handler", "container")

	return true, c.Send(bytes)
}

// fetchFromUpstream 업스트림에서 패키지 가져오기
func (h *DockerContainerHandler) fetchFromUpstream(c *fiber.Ctx, requestPath string) error {
	if len(h.dockerConfig.Proxies) == 0 {
		return c.Status(fiber.StatusNotFound).SendString("No Docker proxies configured")
	}

	isManifest := strings.Contains(requestPath, "/manifests/")
	var lastErr error
	var responseContent []byte
	var headers http.Header

	for i, server := range h.dockerConfig.Proxies {
		if server.URL == "" {
			continue
		}

		url := h.buildDockerURL(server.URL, requestPath)

		h.logger.Debug("Trying Docker server",
			logging.F("server_index", i),
			logging.F("server_name", server.Name),
			logging.F("url", url),
		)

		// HTTP 요청 생성
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			lastErr = err
			h.logger.Error("Failed to create request",
				logging.F("server_index", i),
				logging.F("error", err),
			)
			continue
		}

		// Docker 레지스트리 헤더 복사
		h.copyDockerHeaders(c, req)

		// 인증 처리
		if server.Auth.Username != "" && server.Auth.Password != "" {
			req.SetBasicAuth(server.Auth.Username, server.Auth.Password)
		}

		client := &http.Client{
			Timeout: 30 * time.Second,
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			h.logger.Error("Failed to fetch from server",
				logging.F("server_index", i),
				logging.F("error", err),
			)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			bytes, err := io.ReadAll(resp.Body)
			if err != nil {
				_ = resp.Body.Close()
				lastErr = err
				continue
			}
			headers = resp.Header

			// 캐시에 저장
			if h.ShouldCache(c, resp.StatusCode) {
				if err := h.saveToCache(requestPath, bytes); err != nil {
					h.logger.Error("Failed to save to cache", logging.F("error", err))
				}

				// 헤더 정보도 저장 (매니페스트의 경우)
				if isManifest {
					if err := h.saveDockerHeaders(requestPath, headers); err != nil {
						h.logger.Warn("Failed to save Docker headers", logging.F("error", err))
					}
				}
			}

			responseContent = bytes
			_ = resp.Body.Close()
			h.logger.Info("Successfully fetched from Docker server", logging.F("server_name", server.Name))
			break
		} else if resp.StatusCode == http.StatusUnauthorized {
			// 인증 챌린지 반환
			h.copyResponseHeaders(resp.Header, c)
			_ = resp.Body.Close()
			return c.Status(resp.StatusCode).Send(nil)
		}

		lastErr = fmt.Errorf("upstream returned status %d", resp.StatusCode)
		h.logger.Error("Non-200 status from server",
			logging.F("server_index", i),
			logging.F("status_code", resp.StatusCode),
		)
		_ = resp.Body.Close()
	}

	if responseContent == nil {
		if lastErr != nil {
			h.logger.Error("All Docker servers failed", logging.F("error", lastErr))
			return c.Status(fiber.StatusBadGateway).SendString("All Docker servers failed: " + lastErr.Error())
		}
		return c.Status(fiber.StatusNotFound).SendString("Resource not found")
	}

	// 매니페스트의 경우 저장된 헤더도 설정
	if isManifest && headers != nil {
		h.copyResponseHeaders(headers, c)
	}

	// Content-Type 설정
	if isManifest := strings.Contains(requestPath, "/manifests/"); isManifest {
		contentType := h.detectManifestType(responseContent)
		c.Set("Content-Type", contentType)
	} else if isBlob := strings.Contains(requestPath, "/blobs/"); isBlob {
		c.Set("Content-Type", "application/octet-stream")
	}

	// Docker-Content-Digest 헤더 설정
	if digest := h.extractDigest(requestPath); digest != "" {
		c.Set("Docker-Content-Digest", digest)
	}

	c.Set("X-Cache-Status", "MISS")
	c.Set("X-Proxy-Type", "docker")
	c.Set("X-Handler", "container")

	return c.Send(responseContent)
}

// saveToCache 파일을 캐시에 저장
func (h *DockerContainerHandler) saveToCache(requestPath string, content []byte) error {
	safeBasePath := filepath.Join(h.storageDir, h.dockerConfig.Path)
	safePath, err := security.SafeJoinPath(safeBasePath, requestPath)
	if err != nil {
		return err
	}

	// 디렉토리 생성
	dir := filepath.Dir(safePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// 파일 저장
	return os.WriteFile(safePath, content, 0o644)
}

// saveDockerHeaders 헤더 정보를 파일로 저장
func (h *DockerContainerHandler) saveDockerHeaders(requestPath string, headers http.Header) error {
	safeBasePath := filepath.Join(h.storageDir, h.dockerConfig.Path)
	safePath, err := security.SafeJoinPath(safeBasePath, requestPath+".headers")
	if err != nil {
		return err
	}

	headerMap := make(map[string][]string)
	for key, values := range headers {
		headerMap[key] = values
	}

	data, err := json.Marshal(headerMap)
	if err != nil {
		return err
	}

	return os.WriteFile(safePath, data, 0o644)
}

// loadDockerHeaders 저장된 헤더 정보 로드
func (h *DockerContainerHandler) loadDockerHeaders(filename string) (http.Header, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var headerMap map[string][]string
	if err := json.Unmarshal(data, &headerMap); err != nil {
		return nil, err
	}

	headers := make(http.Header)
	for key, values := range headerMap {
		for _, value := range values {
			headers.Add(key, value)
		}
	}

	return headers, nil
}

// 유틸리티 함수들
func (h *DockerContainerHandler) buildDockerURL(serverURL, requestPath string) string {
	// v2 prefix 추가
	if !strings.HasPrefix(requestPath, "v2/") {
		requestPath = "v2/" + requestPath
	}
	return helpers.JoinURL(serverURL, requestPath)
}

func (h *DockerContainerHandler) copyDockerHeaders(c *fiber.Ctx, req *http.Request) {
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

func (h *DockerContainerHandler) copyResponseHeaders(from http.Header, to *fiber.Ctx) {
	for key, values := range from {
		if len(values) > 0 {
			to.Set(key, values[0])
		}
	}
}

func (h *DockerContainerHandler) detectManifestType(data []byte) string {
	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return "application/vnd.docker.distribution.manifest.v2+json"
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

func (h *DockerContainerHandler) extractDigest(path string) string {
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, "sha256:") {
			return part
		}
	}
	return ""
}
