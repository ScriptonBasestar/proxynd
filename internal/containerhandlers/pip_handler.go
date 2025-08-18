package containerhandlers

import (
	"context"
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

const (
	mimeApplicationZip = "application/zip"
)

// PIPContainerHandler Container 기반 PIP 핸들러
type PIPContainerHandler struct {
	logger            logging.Logger
	name              string
	proxyType         string
	pipConfig         *config.PipProxySettings
	storageDir        string
	serverIdx         int32 // 라운드로빈을 위한 atomic counter
	httpClient        *http.Client
	containerProvider container.ContainerProvider
}

// NewPIPContainerHandler 새로운 Container 기반 PIP 핸들러 생성
func NewPIPContainerHandler(provider container.ContainerProvider) *PIPContainerHandler {
	// HTTP 클라이언트 생성 (프록시 최적화 설정)
	httpClient := &http.Client{
		Timeout: 45 * time.Second,
		Transport: &http.Transport{
			IdleConnTimeout:       30 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
		},
	}

	return &PIPContainerHandler{
		logger:            logging.GetLogger(),
		name:              "pip-container-handler",
		proxyType:         "pip",
		pipConfig:         &config.PipProxySettings{},
		serverIdx:         0,
		httpClient:        httpClient,
		containerProvider: provider,
		storageDir:        provider.GetStorageDir(),
	}
}

// Handle PIP 패키지 프록시 요청 처리
func (h *PIPContainerHandler) Handle(c *fiber.Ctx) error {
	startTime := time.Now()

	h.logger.Info("PIP Container request started",
		logging.F("path", c.Path()),
		logging.F("method", c.Method()),
	)

	defer func() {
		h.logger.Info("PIP Container request completed",
			logging.F("duration_ms", time.Since(startTime).Milliseconds()),
		)
	}()

	// 1. 활성화 상태 확인
	if !h.IsEnabled() {
		return errors.NewError("PROXY001", "PIP 프록시가 비활성화되어 있습니다").
			WithDomain("pip").
			Build()
	}

	// 2. 요청 경로 파싱
	requestPath := c.Params("*")
	h.logger.Debug("PIP request", logging.F("path", requestPath))

	// 3. 캐시에서 파일 확인
	if cached, err := h.serveFromCache(c, requestPath); err == nil && cached {
		return nil
	}

	// 4. 업스트림에서 가져오기
	return h.fetchFromUpstream(c, requestPath)
}

// Name 핸들러 이름 반환
func (h *PIPContainerHandler) Name() string {
	return h.name
}

// Type 프록시 타입 반환
func (h *PIPContainerHandler) Type() string {
	return h.proxyType
}

// SetContainer Container Provider 설정
func (h *PIPContainerHandler) SetContainer(provider container.ContainerProvider) {
	h.containerProvider = provider
}

// GetContainer Container Provider 반환
func (h *PIPContainerHandler) GetContainer() container.ContainerProvider {
	return h.containerProvider
}

// LoadConfig Container를 통한 설정 로드
func (h *PIPContainerHandler) LoadConfig() error {
	if h.containerProvider == nil {
		return fmt.Errorf("container provider not initialized")
	}

	config, err := h.containerProvider.GetPipProxyConfig()
	if err != nil {
		return fmt.Errorf("failed to load PIP config: %w", err)
	}

	h.pipConfig = config
	return nil
}

// ReloadConfig 설정 다시 로드
func (h *PIPContainerHandler) ReloadConfig() error {
	return h.LoadConfig()
}

// IsEnabled 활성화 상태 확인 (Container 기반)
func (h *PIPContainerHandler) IsEnabled() bool {
	if err := h.LoadConfig(); err != nil {
		h.logger.Error("Failed to load PIP config", logging.F("error", err))
		return false
	}
	return len(h.pipConfig.Proxies) > 0
}

// HealthCheck 헬스체크
func (h *PIPContainerHandler) HealthCheck() error {
	if h.containerProvider == nil {
		return fmt.Errorf("container provider not initialized")
	}
	return nil
}

// GenerateCacheKey PIP 전용 캐시 키 생성
func (h *PIPContainerHandler) GenerateCacheKey(c *fiber.Ctx) string {
	path := c.Params("*")
	return fmt.Sprintf("pip:%s", strings.ReplaceAll(path, "/", "_"))
}

// BuildUpstreamURL 업스트림 URL 구성 (라운드로빈 지원)
func (h *PIPContainerHandler) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	if err := h.LoadConfig(); err != nil {
		return "", fmt.Errorf("PIP 설정 로드 실패: %w", err)
	}

	requestPath := c.Params("*")

	if len(h.pipConfig.Proxies) == 0 {
		return "", fmt.Errorf("PIP 서버가 설정되지 않았습니다")
	}

	// 라운드로빈으로 서버 선택
	idx := atomic.AddInt32(&h.serverIdx, 1) - 1
	server := h.pipConfig.Proxies[idx%int32(len(h.pipConfig.Proxies))]

	if server.URL == "" {
		return "", fmt.Errorf("PIP 서버 URL이 설정되지 않았습니다")
	}

	return h.buildPipURL(server.URL, requestPath), nil
}

// IsCacheable 캐시 가능 여부 확인
func (h *PIPContainerHandler) IsCacheable(c *fiber.Ctx) bool {
	return c.Method() == fiber.MethodGet
}

// GetCacheKey 캐시 키 반환
func (h *PIPContainerHandler) GetCacheKey(c *fiber.Ctx) string {
	return h.GenerateCacheKey(c)
}

// ShouldCache PIP 캐싱 정책
func (h *PIPContainerHandler) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	// 200 OK만 캐시
	if statusCode != fiber.StatusOK {
		return false
	}

	path := c.Params("*")

	// 패키지 파일들은 캐시
	cachableExtensions := []string{".whl", ".tar.gz", ".tar.bz2", ".zip", ".egg"}
	for _, ext := range cachableExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}

	// Simple API 응답도 캐시 (짧은 TTL)
	if strings.Contains(path, "/simple/") {
		return true
	}

	// JSON API 응답도 캐시
	if strings.Contains(path, "/json") {
		return true
	}

	return false
}

// GetCacheTTL PIP 캐시 TTL 설정
func (h *PIPContainerHandler) GetCacheTTL(c *fiber.Ctx) time.Duration {
	path := c.Params("*")

	// 패키지 파일은 긴 TTL
	if strings.HasSuffix(path, ".whl") || strings.HasSuffix(path, ".tar.gz") ||
		strings.HasSuffix(path, ".zip") || strings.HasSuffix(path, ".egg") {
		return 24 * time.Hour
	}

	// Simple API 응답은 중간 TTL
	if strings.Contains(path, "/simple/") {
		return 30 * time.Minute
	}

	// JSON API 응답은 짧은 TTL
	if strings.Contains(path, "/json") {
		return 10 * time.Minute
	}

	// 기본 TTL
	return time.Hour
}

// serveFromCache 캐시에서 파일 제공
func (h *PIPContainerHandler) serveFromCache(c *fiber.Ctx, requestPath string) (bool, error) {
	// 보안 경로 조합
	safeBasePath := filepath.Join(h.storageDir, h.pipConfig.Path)
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

	filename := filepath.Base(safePath)

	// 헤더 설정
	contentType := h.getPipContentType(filename, requestPath)
	c.Set("Content-Type", contentType)

	// JSON API 응답은 inline, 패키지 파일은 attachment
	if strings.Contains(contentType, "json") || strings.Contains(contentType, "html") {
		c.Set("Content-Disposition", "inline; filename="+filename)
	} else {
		c.Set("Content-Disposition", "attachment; filename="+filename)
	}

	c.Set("X-Cache-Status", "HIT")
	c.Set("X-Proxy-Type", "pip")
	c.Set("X-Handler", "container")

	return true, c.SendFile(safePath)
}

// fetchFromUpstream 업스트림에서 패키지 가져오기
func (h *PIPContainerHandler) fetchFromUpstream(c *fiber.Ctx, requestPath string) error {
	if len(h.pipConfig.Proxies) == 0 {
		return c.Status(fiber.StatusNotFound).SendString("No PIP servers configured")
	}

	// 요청 컨텍스트 생성 (45초 타임아웃)
	ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
	defer cancel()

	var lastErr error
	for i, server := range h.pipConfig.Proxies {
		if server.URL == "" {
			continue
		}

		fullURL := h.buildPipURL(server.URL, requestPath)

		h.logger.Debug("Trying PIP server",
			logging.F("server_index", i),
			logging.F("server_name", server.Name),
			logging.F("url", fullURL),
		)

		// HTTP 요청 생성
		req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
		if err != nil {
			lastErr = err
			h.logger.Error("Failed to create request",
				logging.F("server_index", i),
				logging.F("error", err),
			)
			continue
		}

		// 헤더 설정
		req.Header.Set("User-Agent", "ProxyND/2.0 PIP-Container-Proxy")
		req.Header.Set("Accept", "*/*")

		// 클라이언트 헤더 복사
		c.Request().Header.VisitAll(func(key, value []byte) {
			keyStr := string(key)
			if keyStr != "Host" {
				req.Header.Set(keyStr, string(value))
			}
		})

		// 요청 실행
		resp, err := h.httpClient.Do(req)
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

			// 캐시에 저장
			if h.ShouldCache(c, resp.StatusCode) {
				if err := h.saveToCache(requestPath, bytes); err != nil {
					h.logger.Error("Failed to save to cache", logging.F("error", err))
				}
			}

			filename := filepath.Base(requestPath)

			// 응답 헤더 설정
			contentType := h.getPipContentType(filename, requestPath)
			c.Set("Content-Type", contentType)

			// JSON API 응답은 inline, 패키지 파일은 attachment
			if strings.Contains(contentType, "json") || strings.Contains(contentType, "html") {
				c.Set("Content-Disposition", "inline; filename="+filename)
			} else {
				c.Set("Content-Disposition", "attachment; filename="+filename)
			}

			c.Set("X-Cache-Status", "MISS")
			c.Set("X-Proxy-Type", "pip")
			c.Set("X-Handler", "container")
			c.Status(resp.StatusCode)

			_ = resp.Body.Close()
			h.logger.Info("Successfully fetched from PIP server", logging.F("server_name", server.Name))
			return c.Send(bytes)
		}

		lastErr = fmt.Errorf("upstream returned status %d", resp.StatusCode)
		h.logger.Error("Non-200 status from server",
			logging.F("server_index", i),
			logging.F("status_code", resp.StatusCode),
		)
		_ = resp.Body.Close()
	}

	// 모든 서버 실패
	if lastErr != nil {
		h.logger.Error("All PIP servers failed", logging.F("error", lastErr))
		return c.Status(fiber.StatusBadGateway).SendString("All PIP servers failed: " + lastErr.Error())
	}

	return c.Status(fiber.StatusNotFound).SendString("Package not found")
}

// saveToCache 파일을 캐시에 저장
func (h *PIPContainerHandler) saveToCache(requestPath string, content []byte) error {
	safeBasePath := filepath.Join(h.storageDir, h.pipConfig.Path)
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

// 유틸리티 함수들
func (h *PIPContainerHandler) buildPipURL(serverURL, requestPath string) string {
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

func (h *PIPContainerHandler) getPipContentType(filename, path string) string {
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
		return mimeApplicationZip
	case strings.HasSuffix(filename, ".tar.gz"):
		return "application/x-gzip"
	case strings.HasSuffix(filename, ".tar.bz2"):
		return "application/x-bzip2"
	case strings.HasSuffix(filename, ".zip"):
		return mimeApplicationZip
	case strings.HasSuffix(filename, ".egg"):
		return mimeApplicationZip
	default:
		return "application/octet-stream"
	}
}
