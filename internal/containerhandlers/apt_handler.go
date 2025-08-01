package containerhandlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/config"
	"proxynd/internal/container"
	"proxynd/internal/errors"
	"proxynd/internal/security"
	"proxynd/logging"
)

// APTContainerHandler Container 기반 APT 핸들러
type APTContainerHandler struct {
	logger            logging.Logger
	name              string
	proxyType         string
	aptConfig         *config.AptProxyConfig
	storageDir        string
	mirrorIdx         int32 // 라운드로빈을 위한 atomic counter
	containerProvider container.ContainerProvider
}

// NewAPTContainerHandler 새로운 Container 기반 APT 핸들러 생성
func NewAPTContainerHandler(provider container.ContainerProvider) *APTContainerHandler {
	return &APTContainerHandler{
		logger:            logging.GetLogger(),
		name:              "apt-container-handler",
		proxyType:         "apt",
		aptConfig:         &config.AptProxyConfig{},
		mirrorIdx:         0,
		containerProvider: provider,
		storageDir:        provider.GetStorageDir(),
	}
}

// Handle APT 패키지 프록시 요청 처리
func (h *APTContainerHandler) Handle(c *fiber.Ctx) error {
	startTime := time.Now()

	h.logger.Info("APT Container request started",
		logging.F("path", c.Path()),
		logging.F("method", c.Method()),
	)

	defer func() {
		h.logger.Info("APT Container request completed",
			logging.F("duration_ms", time.Since(startTime).Milliseconds()),
		)
	}()

	// 1. 활성화 상태 확인
	if !h.IsEnabled() {
		return errors.NewError("PROXY001", "APT 프록시가 비활성화되어 있습니다").
			WithDomain("apt").
			Build()
	}

	// 2. 캐시에서 파일 확인
	osType := c.Params("osType", "ubuntu")
	packagePath := c.Params("*")

	if cached, err := h.serveFromCache(c, osType, packagePath); err == nil && cached {
		return nil
	}

	// 3. 업스트림에서 가져오기
	return h.fetchFromUpstream(c, osType, packagePath)
}

// Name 핸들러 이름 반환
func (h *APTContainerHandler) Name() string {
	return h.name
}

// Type 프록시 타입 반환
func (h *APTContainerHandler) Type() string {
	return h.proxyType
}

// SetContainer Container Provider 설정
func (h *APTContainerHandler) SetContainer(provider container.ContainerProvider) {
	h.containerProvider = provider
}

// GetContainer Container Provider 반환
func (h *APTContainerHandler) GetContainer() container.ContainerProvider {
	return h.containerProvider
}

// LoadConfig Container를 통한 설정 로드
func (h *APTContainerHandler) LoadConfig() error {
	if h.containerProvider == nil {
		return fmt.Errorf("container provider not initialized")
	}

	config, err := h.containerProvider.GetAptProxyConfig()
	if err != nil {
		return fmt.Errorf("failed to load APT config: %w", err)
	}

	h.aptConfig = config
	return nil
}

// ReloadConfig 설정 다시 로드
func (h *APTContainerHandler) ReloadConfig() error {
	return h.LoadConfig()
}

// IsEnabled 활성화 상태 확인 (Container 기반)
func (h *APTContainerHandler) IsEnabled() bool {
	if err := h.LoadConfig(); err != nil {
		h.logger.Error("Failed to load APT config", logging.F("error", err))
		return false
	}
	return len(h.aptConfig.Proxies) > 0
}

// HealthCheck 헬스체크
func (h *APTContainerHandler) HealthCheck() error {
	if h.containerProvider == nil {
		return fmt.Errorf("container provider not initialized")
	}
	return nil
}

// GenerateCacheKey APT 전용 캐시 키 생성
func (h *APTContainerHandler) GenerateCacheKey(c *fiber.Ctx) string {
	osType := c.Params("osType", "ubuntu")
	path := c.Params("*")
	return fmt.Sprintf("apt:%s:%s", osType, strings.ReplaceAll(path, "/", "_"))
}

// BuildUpstreamURL 업스트림 URL 구성 (라운드로빈 지원)
func (h *APTContainerHandler) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	if err := h.LoadConfig(); err != nil {
		return "", fmt.Errorf("APT 설정 로드 실패: %w", err)
	}

	osType := c.Params("osType", "ubuntu")
	packagePath := c.Params("*")

	// OS별 프록시 설정 확인
	proxies, exists := h.aptConfig.Proxies[osType]
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

// IsCacheable 캐시 가능 여부 확인
func (h *APTContainerHandler) IsCacheable(c *fiber.Ctx) bool {
	return c.Method() == fiber.MethodGet
}

// GetCacheKey 캐시 키 반환
func (h *APTContainerHandler) GetCacheKey(c *fiber.Ctx) string {
	return h.GenerateCacheKey(c)
}

// ShouldCache APT 캐싱 정책
func (h *APTContainerHandler) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	// 200 OK만 캐시
	if statusCode != fiber.StatusOK {
		return false
	}

	// Release, Packages, Sources 파일들은 캐시
	path := c.Params("*")
	cachableFiles := []string{"Release", "Packages", "Sources", ".deb", ".tar.", ".diff/"}

	for _, suffix := range cachableFiles {
		if strings.Contains(path, suffix) {
			return true
		}
	}

	return false
}

// GetCacheTTL APT 캐시 TTL 설정
func (h *APTContainerHandler) GetCacheTTL(c *fiber.Ctx) time.Duration {
	path := c.Params("*")

	// Release 파일은 짧은 TTL
	if strings.Contains(path, "Release") {
		return 15 * time.Minute
	}

	// 패키지 파일들은 긴 TTL
	if strings.HasSuffix(path, ".deb") {
		return 24 * time.Hour
	}

	// 기본 TTL
	return time.Hour
}

// serveFromCache 캐시에서 파일 제공
func (h *APTContainerHandler) serveFromCache(c *fiber.Ctx, osType, packagePath string) (bool, error) {
	// 보안 경로 조합
	safeBasePath := filepath.Join(h.storageDir, "apt", osType)
	safePath, err := security.SafeJoinPath(safeBasePath, packagePath)
	if err != nil {
		return false, fmt.Errorf("invalid path: %w", err)
	}

	// 파일 존재 확인
	fileInfo, err := os.Stat(safePath)
	if err != nil || fileInfo.IsDir() {
		return false, nil
	}

	h.logger.Debug("Serving from cache", logging.F("path", safePath))

	// 헤더 설정
	c.Set("Content-Type", getAptContentType(packagePath))

	if shouldInline(packagePath) {
		c.Set("Content-Disposition", "inline")
	} else {
		filename := filepath.Base(packagePath)
		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	}

	c.Set("X-Cache-Status", "HIT")
	c.Set("X-Proxy-Type", "apt")
	c.Set("X-Handler", "container")

	return true, c.SendFile(safePath)
}

// fetchFromUpstream 업스트림에서 패키지 가져오기
func (h *APTContainerHandler) fetchFromUpstream(c *fiber.Ctx, osType, packagePath string) error {
	proxies, exists := h.aptConfig.Proxies[osType]
	if !exists || len(proxies) == 0 {
		return c.Status(fiber.StatusNotFound).SendString("No APT mirrors configured for " + osType)
	}

	var lastErr error
	for i, mirror := range proxies {
		if mirror.URL == "" {
			continue
		}

		baseURL := strings.TrimRight(mirror.URL, "/")
		cleanPath := strings.TrimLeft(packagePath, "/")
		upstreamURL := fmt.Sprintf("%s/%s", baseURL, cleanPath)

		h.logger.Debug("Trying mirror",
			logging.F("mirror_index", i),
			logging.F("url", upstreamURL),
		)

		// 요청 실행
		agent := fiber.Get(upstreamURL)
		agent.Set("User-Agent", "ProxyND/2.0 APT-Container-Proxy")
		agent.Set("X-APT-Proxy", "ProxyND")

		statusCode, body, errs := agent.Bytes()
		if len(errs) > 0 {
			lastErr = errs[0]
			h.logger.Error("Failed to fetch from mirror",
				logging.F("mirror_index", i),
				logging.F("error", lastErr),
			)
			continue
		}

		if statusCode != fiber.StatusOK {
			lastErr = fmt.Errorf("upstream returned status %d", statusCode)
			h.logger.Error("Non-200 status from mirror",
				logging.F("mirror_index", i),
				logging.F("status_code", statusCode),
			)
			continue
		}

		// 성공 - 파일 저장 및 응답
		if h.ShouldCache(c, statusCode) {
			if err := h.saveToCache(osType, packagePath, body); err != nil {
				h.logger.Error("Failed to save to cache", logging.F("error", err))
			}
		}

		// 응답 헤더 설정
		c.Set("Content-Type", getAptContentType(packagePath))
		c.Set("X-Cache-Status", "MISS")
		c.Set("X-Proxy-Type", "apt")
		c.Set("X-Handler", "container")
		c.Status(statusCode)

		return c.Send(body)
	}

	// 모든 미러 실패
	if lastErr != nil {
		h.logger.Error("All mirrors failed", logging.F("error", lastErr))
		return c.Status(fiber.StatusBadGateway).SendString("All APT mirrors failed: " + lastErr.Error())
	}

	return c.Status(fiber.StatusNotFound).SendString("Package not found")
}

// saveToCache 파일을 캐시에 저장
func (h *APTContainerHandler) saveToCache(osType, packagePath string, content []byte) error {
	safeBasePath := filepath.Join(h.storageDir, "apt", osType)
	safePath, err := security.SafeJoinPath(safeBasePath, packagePath)
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

// 유틸리티 함수들 (기존 코드에서 복사)
func getAptContentType(path string) string {
	if strings.HasSuffix(path, ".deb") {
		return "application/vnd.debian.binary-package"
	}
	if strings.Contains(path, "Release") {
		return "text/plain; charset=utf-8"
	}
	if strings.Contains(path, "Packages") {
		return "text/plain; charset=utf-8"
	}
	if strings.Contains(path, "Sources") {
		return "text/plain; charset=utf-8"
	}
	return "application/octet-stream"
}

func shouldInline(path string) bool {
	inlineTypes := []string{"Release", "Packages", "Sources"}
	for _, inlineType := range inlineTypes {
		if strings.Contains(path, inlineType) {
			return true
		}
	}
	return false
}
