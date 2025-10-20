package containerhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/config"
	"proxynd/internal/container"
	"proxynd/internal/errors"
	"proxynd/internal/logging"
	"proxynd/internal/security"
)

// NPMContainerHandler Container 기반 NPM 핸들러
type NPMContainerHandler struct {
	logger            logging.Logger
	name              string
	proxyType         string
	npmConfig         *config.NpmProxySettings
	storageDir        string
	httpClient        *http.Client
	containerProvider container.ContainerProvider
}

// NewNPMContainerHandler 새로운 Container 기반 NPM 핸들러 생성
func NewNPMContainerHandler(provider container.ContainerProvider) *NPMContainerHandler {
	// HTTP 클라이언트 생성
	httpClient := &http.Client{
		Timeout: 45 * time.Second,
		Transport: &http.Transport{
			IdleConnTimeout: 30 * time.Second,
		},
	}

	return &NPMContainerHandler{
		logger:            logging.GetLogger(),
		name:              "npm-container-handler",
		proxyType:         "npm",
		npmConfig:         &config.NpmProxySettings{},
		httpClient:        httpClient,
		containerProvider: provider,
		storageDir:        provider.GetStorageDir(),
	}
}

// Handle NPM 패키지 프록시 요청 처리
func (h *NPMContainerHandler) Handle(c *fiber.Ctx) error {
	startTime := time.Now()

	h.logger.Info("NPM Container request started",
		logging.F("path", c.Path()),
		logging.F("method", c.Method()),
	)

	defer func() {
		h.logger.Info("NPM Container request completed",
			logging.F("duration_ms", time.Since(startTime).Milliseconds()),
		)
	}()

	// 1. 활성화 상태 확인
	if !h.IsEnabled() {
		return errors.NewError("PROXY001", "NPM 프록시가 비활성화되어 있습니다").
			WithDomain("npm").
			Build()
	}

	// 2. 요청 경로 파싱
	requestPath := c.Params("*")

	// 3. 메타데이터 요청 확인
	isMetadata := h.isNpmMetadataRequest(requestPath)

	// 4. 캐시 확인
	if cached, err := h.serveFromCache(c, requestPath); err == nil && cached {
		return nil
	}

	// 5. 업스트림에서 가져오기
	return h.fetchFromUpstream(c, requestPath, isMetadata)
}

// Name 핸들러 이름 반환
func (h *NPMContainerHandler) Name() string {
	return h.name
}

// Type 프록시 타입 반환
func (h *NPMContainerHandler) Type() string {
	return h.proxyType
}

// SetContainer Container Provider 설정
func (h *NPMContainerHandler) SetContainer(provider container.ContainerProvider) {
	h.containerProvider = provider
}

// GetContainer Container Provider 반환
func (h *NPMContainerHandler) GetContainer() container.ContainerProvider {
	return h.containerProvider
}

// LoadConfig Container를 통한 설정 로드
func (h *NPMContainerHandler) LoadConfig() error {
	if h.containerProvider == nil {
		return fmt.Errorf("container provider not initialized")
	}

	config, err := h.containerProvider.GetNpmProxyConfig()
	if err != nil {
		return fmt.Errorf("failed to load NPM config: %w", err)
	}

	h.npmConfig = config
	return nil
}

// ReloadConfig 설정 다시 로드
func (h *NPMContainerHandler) ReloadConfig() error {
	return h.LoadConfig()
}

// IsEnabled 활성화 상태 확인 (Container 기반)
func (h *NPMContainerHandler) IsEnabled() bool {
	if err := h.LoadConfig(); err != nil {
		h.logger.Error("Failed to load NPM config", logging.F("error", err))
		return false
	}
	// 프록시 맵에서 하나라도 서버가 있으면 활성화
	for _, servers := range h.npmConfig.Proxies {
		if len(servers) > 0 {
			return true
		}
	}
	return false
}

// HealthCheck 헬스체크
func (h *NPMContainerHandler) HealthCheck() error {
	if h.containerProvider == nil {
		return fmt.Errorf("container provider not initialized")
	}
	return nil
}

// GenerateCacheKey NPM 전용 캐시 키 생성
func (h *NPMContainerHandler) GenerateCacheKey(c *fiber.Ctx) string {
	requestPath := c.Params("*")
	return fmt.Sprintf("npm:%s", strings.ReplaceAll(requestPath, "/", "_"))
}

// BuildUpstreamURL NPM 레지스트리 URL 구성 (라운드로빈 지원)
func (h *NPMContainerHandler) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	if err := h.LoadConfig(); err != nil {
		return "", fmt.Errorf("NPM 설정 로드 실패: %w", err)
	}

	requestPath := c.Params("*")

	if len(h.npmConfig.Proxies) == 0 {
		return "", fmt.Errorf("NPM 레지스트리가 설정되지 않았습니다")
	}

	// 첫 번째 레지스트리의 첫 번째 서버를 사용 (단순화)
	for _, servers := range h.npmConfig.Proxies {
		if len(servers) > 0 {
			server := servers[0]
			if server.URL == "" {
				continue
			}

			// URL 구성
			baseURL := strings.TrimRight(server.URL, "/")
			cleanPath := strings.TrimLeft(requestPath, "/")

			return fmt.Sprintf("%s/%s", baseURL, cleanPath), nil
		}
	}

	return "", fmt.Errorf("사용 가능한 NPM 레지스트리가 없습니다")
}

// IsCacheable 캐시 가능 여부 확인
func (h *NPMContainerHandler) IsCacheable(c *fiber.Ctx) bool {
	return c.Method() == fiber.MethodGet
}

// GetCacheKey 캐시 키 반환
func (h *NPMContainerHandler) GetCacheKey(c *fiber.Ctx) string {
	return h.GenerateCacheKey(c)
}

// ShouldCache NPM 캐싱 정책
func (h *NPMContainerHandler) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	// 200 OK만 캐시
	if statusCode != fiber.StatusOK {
		return false
	}

	requestPath := c.Params("*")

	// 패키지 tarball은 항상 캐시
	if strings.HasSuffix(requestPath, ".tgz") || strings.HasSuffix(requestPath, ".tar.gz") {
		return true
	}

	// 메타데이터도 캐시 (짧은 TTL)
	if h.isNpmMetadataRequest(requestPath) {
		return true
	}

	return false
}

// GetCacheTTL NPM 캐시 TTL 설정
func (h *NPMContainerHandler) GetCacheTTL(c *fiber.Ctx) time.Duration {
	requestPath := c.Params("*")

	// 패키지 tarball은 긴 TTL
	if strings.HasSuffix(requestPath, ".tgz") || strings.HasSuffix(requestPath, ".tar.gz") {
		return 24 * time.Hour
	}

	// 메타데이터는 짧은 TTL
	if h.isNpmMetadataRequest(requestPath) {
		return 5 * time.Minute
	}

	// 기본 TTL
	return time.Hour
}

// serveFromCache 캐시에서 파일 제공
func (h *NPMContainerHandler) serveFromCache(c *fiber.Ctx, requestPath string) (bool, error) {
	// 보안 경로 조합
	baseDir := filepath.Join(h.storageDir, h.npmConfig.Path)
	safeFilePath, err := security.SafeJoinPath(baseDir, requestPath)
	if err != nil {
		return false, fmt.Errorf("invalid path: %w", err)
	}

	// 파일 존재 확인
	fileInfo, err := os.Stat(safeFilePath)
	if err != nil || fileInfo.IsDir() {
		return false, nil
	}

	h.logger.Debug("Serving from cache", logging.F("path", safeFilePath))

	// 헤더 설정
	c.Set("Content-Type", h.getNpmContentType(requestPath))
	c.Set("X-Cache-Status", "HIT")
	c.Set("X-Proxy-Type", "npm")
	c.Set("X-Handler", "container")

	// ETag 설정
	etag := fmt.Sprintf("\"%d-%d\"", fileInfo.Size(), fileInfo.ModTime().Unix())
	c.Set("ETag", etag)

	// 조건부 요청 처리
	if match := c.Get("If-None-Match"); match == etag {
		return true, c.SendStatus(fiber.StatusNotModified)
	}

	// 메타데이터 파일인 경우 JSON으로 처리
	if h.isNpmMetadataRequest(requestPath) {
		content, err := os.ReadFile(safeFilePath)
		if err != nil {
			return false, err
		}

		// JSON 형태로 응답
		var jsonData interface{}
		if err := json.Unmarshal(content, &jsonData); err != nil {
			// JSON이 아닌 경우 원본 파일 전송
			return true, c.SendFile(safeFilePath)
		}

		return true, c.JSON(jsonData)
	}

	return true, c.SendFile(safeFilePath)
}

// fetchFromUpstream 업스트림에서 패키지 가져오기
func (h *NPMContainerHandler) fetchFromUpstream(c *fiber.Ctx, requestPath string, isMetadata bool) error {
	hasRegistry := false
	for _, servers := range h.npmConfig.Proxies {
		if len(servers) > 0 {
			hasRegistry = true
			break
		}
	}

	if !hasRegistry {
		return c.Status(fiber.StatusNotFound).SendString("No NPM registries configured")
	}

	// 업스트림 URL 구성
	upstreamURL, err := h.BuildUpstreamURL(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	h.logger.Debug("Fetching from upstream", logging.F("url", upstreamURL))

	// HTTP 요청 생성
	ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", upstreamURL, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to create request")
	}

	// 헤더 설정
	req.Header.Set("User-Agent", "ProxyND/2.0 NPM-Container-Proxy")
	req.Header.Set("Accept", "application/vnd.npm.install-v1+json; q=1.0, application/json; q=0.8, */*")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")

	// 클라이언트 헤더 복사
	for key, value := range c.Request().Header.All() {
		keyStr := string(key)
		if keyStr != "Host" && keyStr != "Authorization" {
			req.Header.Set(keyStr, string(value))
		}
	}

	// 요청 실행
	resp, err := h.httpClient.Do(req)
	if err != nil {
		h.logger.Error("Failed to fetch from upstream", logging.F("error", err))
		return c.Status(fiber.StatusBadGateway).SendString("Failed to fetch from upstream")
	}
	defer func() { _ = resp.Body.Close() }()

	// 응답 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to read response")
	}

	// 성공적인 응답인 경우 캐시에 저장
	if resp.StatusCode == http.StatusOK && h.ShouldCache(c, resp.StatusCode) {
		if err := h.saveToCache(requestPath, body); err != nil {
			h.logger.Error("Failed to save to cache", logging.F("error", err))
		}
	}

	// 응답 헤더 설정
	c.Set("Content-Type", h.getNpmContentType(requestPath))
	c.Set("X-Cache-Status", "MISS")
	c.Set("X-Proxy-Type", "npm")
	c.Set("X-Handler", "container")

	// 업스트림 헤더 복사
	for key, values := range resp.Header {
		if len(values) > 0 && key != "Content-Length" {
			c.Set(key, values[0])
		}
	}

	c.Status(resp.StatusCode)

	// 메타데이터인 경우 JSON으로 파싱해서 응답
	if isMetadata && resp.StatusCode == http.StatusOK {
		var jsonData interface{}
		if err := json.Unmarshal(body, &jsonData); err == nil {
			return c.JSON(jsonData)
		}
	}

	return c.Send(body)
}

// saveToCache 파일을 캐시에 저장
func (h *NPMContainerHandler) saveToCache(requestPath string, content []byte) error {
	baseDir := filepath.Join(h.storageDir, h.npmConfig.Path)
	safeFilePath, err := security.SafeJoinPath(baseDir, requestPath)
	if err != nil {
		return err
	}

	// 디렉토리 생성
	dir := filepath.Dir(safeFilePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// 파일 저장
	return os.WriteFile(safeFilePath, content, 0o644)
}

// 유틸리티 함수들
func (h *NPMContainerHandler) isNpmMetadataRequest(path string) bool {
	// 패키지 메타데이터 요청인지 확인
	return !strings.Contains(path, "/-/") && !strings.HasSuffix(path, ".tgz") && !strings.HasSuffix(path, ".tar.gz")
}

func (h *NPMContainerHandler) getNpmContentType(path string) string {
	if strings.HasSuffix(path, ".tgz") || strings.HasSuffix(path, ".tar.gz") {
		return mimeApplicationOctetStream
	}
	if h.isNpmMetadataRequest(path) {
		return mimeApplicationJSON
	}
	return mimeApplicationOctetStream
}
