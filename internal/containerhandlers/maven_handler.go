package containerhandlers

import (
	"crypto/sha1"
	"encoding/hex"
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
	"proxynd/internal/logging"
	"proxynd/internal/security"
)

const (
	headerHost         = "Host"
	mimeApplicationXML = "application/xml"
)

// MavenContainerHandler Container 기반 Maven 핸들러
type MavenContainerHandler struct {
	logger            logging.Logger
	name              string
	proxyType         string
	mavenConfig       *config.MavenProxySettings
	storageDir        string
	repoIdx           int32 // 라운드로빈을 위한 atomic counter
	containerProvider container.ContainerProvider
}

// NewMavenContainerHandler 새로운 Container 기반 Maven 핸들러 생성
func NewMavenContainerHandler(provider container.ContainerProvider) *MavenContainerHandler {
	return &MavenContainerHandler{
		logger:            logging.GetLogger(),
		name:              "maven-container-handler",
		proxyType:         "maven",
		mavenConfig:       &config.MavenProxySettings{},
		repoIdx:           0,
		containerProvider: provider,
		storageDir:        provider.GetStorageDir(),
	}
}

// Handle Maven 아티팩트 프록시 요청 처리
func (h *MavenContainerHandler) Handle(c *fiber.Ctx) error {
	startTime := time.Now()

	h.logger.Info("Maven Container request started",
		logging.F("path", c.Path()),
		logging.F("method", c.Method()),
	)

	defer func() {
		h.logger.Info("Maven Container request completed",
			logging.F("duration_ms", time.Since(startTime).Milliseconds()),
		)
	}()

	// 1. 활성화 상태 확인
	if !h.IsEnabled() {
		return errors.NewError("PROXY001", "Maven 프록시가 비활성화되어 있습니다").
			WithDomain("maven").
			Build()
	}

	// 2. 아티팩트 경로 파싱
	artifactPath := c.Params("*")
	isSnapshot := strings.Contains(artifactPath, "-SNAPSHOT")

	// 3. 캐시 확인 (SNAPSHOT이 아닌 경우만)
	if !isSnapshot {
		if cached, err := h.serveFromCache(c, artifactPath); err == nil && cached {
			return nil
		}
	}

	// 4. 업스트림에서 가져오기
	return h.fetchFromUpstream(c, artifactPath)
}

// Name 핸들러 이름 반환
func (h *MavenContainerHandler) Name() string {
	return h.name
}

// Type 프록시 타입 반환
func (h *MavenContainerHandler) Type() string {
	return h.proxyType
}

// SetContainer Container Provider 설정
func (h *MavenContainerHandler) SetContainer(provider container.ContainerProvider) {
	h.containerProvider = provider
}

// GetContainer Container Provider 반환
func (h *MavenContainerHandler) GetContainer() container.ContainerProvider {
	return h.containerProvider
}

// LoadConfig Container를 통한 설정 로드
func (h *MavenContainerHandler) LoadConfig() error {
	if h.containerProvider == nil {
		return fmt.Errorf("container provider not initialized")
	}

	config, err := h.containerProvider.GetMavenProxyConfig()
	if err != nil {
		return fmt.Errorf("failed to load Maven config: %w", err)
	}

	h.mavenConfig = config
	return nil
}

// ReloadConfig 설정 다시 로드
func (h *MavenContainerHandler) ReloadConfig() error {
	return h.LoadConfig()
}

// IsEnabled 활성화 상태 확인 (Container 기반)
func (h *MavenContainerHandler) IsEnabled() bool {
	if err := h.LoadConfig(); err != nil {
		h.logger.Error("Failed to load Maven config", logging.F("error", err))
		return false
	}
	return len(h.mavenConfig.Proxies) > 0
}

// HealthCheck 헬스체크
func (h *MavenContainerHandler) HealthCheck() error {
	if h.containerProvider == nil {
		return fmt.Errorf("container provider not initialized")
	}
	return nil
}

// GenerateCacheKey Maven 전용 캐시 키 생성
func (h *MavenContainerHandler) GenerateCacheKey(c *fiber.Ctx) string {
	artifactPath := c.Params("*")
	return fmt.Sprintf("maven:%s", strings.ReplaceAll(artifactPath, "/", "_"))
}

// BuildUpstreamURL Maven 레포지토리 URL 구성 (라운드로빈 지원)
func (h *MavenContainerHandler) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	if err := h.LoadConfig(); err != nil {
		return "", fmt.Errorf("maven 설정 로드 실패: %w", err)
	}

	artifactPath := c.Params("*")

	if len(h.mavenConfig.Proxies) == 0 {
		return "", fmt.Errorf("maven 레포지토리가 설정되지 않았습니다")
	}

	// 라운드로빈으로 레포지토리 선택
	idx := atomic.AddInt32(&h.repoIdx, 1) - 1
	repository := h.mavenConfig.Proxies[idx%int32(len(h.mavenConfig.Proxies))]

	if repository.URL == "" {
		return "", fmt.Errorf("maven 레포지토리 URL이 설정되지 않았습니다")
	}

	// URL 구성
	baseURL := strings.TrimRight(repository.URL, "/")
	cleanPath := strings.TrimLeft(artifactPath, "/")

	return fmt.Sprintf("%s/%s", baseURL, cleanPath), nil
}

// IsCacheable 캐시 가능 여부 확인
func (h *MavenContainerHandler) IsCacheable(c *fiber.Ctx) bool {
	return c.Method() == fiber.MethodGet
}

// GetCacheKey 캐시 키 반환
func (h *MavenContainerHandler) GetCacheKey(c *fiber.Ctx) string {
	return h.GenerateCacheKey(c)
}

// ShouldCache Maven 캐싱 정책
func (h *MavenContainerHandler) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	// 200 OK만 캐시
	if statusCode != fiber.StatusOK {
		return false
	}

	artifactPath := c.Params("*")

	// SNAPSHOT은 캐시하지 않음
	if strings.Contains(artifactPath, "-SNAPSHOT") {
		return false
	}

	// JAR, POM, XML 파일은 캐시
	cachableExtensions := []string{".jar", ".pom", ".xml", ".war", ".ear", ".aar"}
	for _, ext := range cachableExtensions {
		if strings.HasSuffix(artifactPath, ext) {
			return true
		}
	}

	// 메타데이터 파일들도 캐시
	if strings.Contains(artifactPath, "maven-metadata.xml") {
		return true
	}

	return false
}

// GetCacheTTL Maven 캐시 TTL 설정
func (h *MavenContainerHandler) GetCacheTTL(c *fiber.Ctx) time.Duration {
	artifactPath := c.Params("*")

	// SNAPSHOT은 짧은 TTL
	if strings.Contains(artifactPath, "-SNAPSHOT") {
		return 5 * time.Minute
	}

	// 메타데이터는 중간 TTL
	if strings.Contains(artifactPath, "maven-metadata.xml") {
		return 30 * time.Minute
	}

	// 일반 아티팩트는 긴 TTL
	return 24 * time.Hour
}

// serveFromCache 캐시에서 파일 제공
func (h *MavenContainerHandler) serveFromCache(c *fiber.Ctx, artifactPath string) (bool, error) {
	// 보안 경로 조합
	safeBasePath := filepath.Join(h.storageDir, "maven")
	safePath, err := security.SafeJoinPath(safeBasePath, artifactPath)
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
	c.Set("Content-Type", getMavenContentType(artifactPath))

	// ETag 생성 (파일 크기 + 수정 시간 기반)
	etag := fmt.Sprintf("\"%d-%d\"", fileInfo.Size(), fileInfo.ModTime().Unix())
	c.Set("ETag", etag)

	// Last-Modified 설정
	c.Set("Last-Modified", fileInfo.ModTime().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT"))

	c.Set("X-Cache-Status", "HIT")
	c.Set("X-Proxy-Type", "maven")
	c.Set("X-Handler", "container")

	// 조건부 요청 처리
	if match := c.Get("If-None-Match"); match == etag {
		return true, c.SendStatus(fiber.StatusNotModified)
	}

	return true, c.SendFile(safePath)
}

// fetchFromUpstream 업스트림에서 아티팩트 가져오기
func (h *MavenContainerHandler) fetchFromUpstream(c *fiber.Ctx, artifactPath string) error {
	if len(h.mavenConfig.Proxies) == 0 {
		return c.Status(fiber.StatusNotFound).SendString("No Maven repositories configured")
	}

	var lastErr error
	for i, repository := range h.mavenConfig.Proxies {
		if repository.URL == "" {
			continue
		}

		baseURL := strings.TrimRight(repository.URL, "/")
		cleanPath := strings.TrimLeft(artifactPath, "/")
		upstreamURL := fmt.Sprintf("%s/%s", baseURL, cleanPath)

		h.logger.Debug("Trying repository",
			logging.F("repo_index", i),
			logging.F("url", upstreamURL),
		)

		// 요청 실행
		agent := fiber.Get(upstreamURL)
		h.transformRequest(c, agent)

		statusCode, body, errs := agent.Bytes()
		if len(errs) > 0 {
			lastErr = errs[0]
			h.logger.Error("Failed to fetch from repository",
				logging.F("repo_index", i),
				logging.F("error", lastErr),
			)
			continue
		}

		if statusCode != fiber.StatusOK {
			lastErr = fmt.Errorf("upstream returned status %d", statusCode)
			h.logger.Debug("Non-200 status from repository",
				logging.F("repo_index", i),
				logging.F("status_code", statusCode),
			)
			continue
		}

		// 성공 - 파일 저장 및 응답
		if h.ShouldCache(c, statusCode) {
			if err := h.saveToCache(artifactPath, body); err != nil {
				h.logger.Error("Failed to save to cache", logging.F("error", err))
			}
		}

		// 응답 헤더 설정
		c.Set("Content-Type", getMavenContentType(artifactPath))

		// SHA1 체크섬 생성 (JAR, POM 파일의 경우)
		if strings.HasSuffix(artifactPath, ".jar") || strings.HasSuffix(artifactPath, ".pom") {
			checksum := generateSHA1(body)
			c.Set("X-Checksum-SHA1", checksum)
		}

		c.Set("X-Cache-Status", "MISS")
		c.Set("X-Proxy-Type", "maven")
		c.Set("X-Handler", "container")
		c.Status(statusCode)

		return c.Send(body)
	}

	// 모든 레포지토리 실패
	if lastErr != nil {
		h.logger.Error("All repositories failed", logging.F("error", lastErr))
		return c.Status(fiber.StatusBadGateway).SendString("All Maven repositories failed: " + lastErr.Error())
	}

	return c.Status(fiber.StatusNotFound).SendString("Artifact not found")
}

// saveToCache 파일을 캐시에 저장
func (h *MavenContainerHandler) saveToCache(artifactPath string, content []byte) error {
	safeBasePath := filepath.Join(h.storageDir, "maven")
	safePath, err := security.SafeJoinPath(safeBasePath, artifactPath)
	if err != nil {
		return err
	}

	// 디렉토리 생성
	dir := filepath.Dir(safePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// 파일 저장
	if err := os.WriteFile(safePath, content, 0o644); err != nil {
		return err
	}

	// SHA1 체크섬 파일 생성 (JAR, POM 파일의 경우)
	if strings.HasSuffix(artifactPath, ".jar") || strings.HasSuffix(artifactPath, ".pom") {
		checksum := generateSHA1(content)
		checksumPath := safePath + ".sha1"
		if err := os.WriteFile(checksumPath, []byte(checksum), 0o644); err != nil {
			h.logger.Warn("Failed to save checksum file", logging.F("error", err))
		}
	}

	return nil
}

// transformRequest Maven 요청 변환
func (h *MavenContainerHandler) transformRequest(c *fiber.Ctx, upstreamReq *fiber.Agent) {
	// 기본 헤더 설정
	upstreamReq.Set("User-Agent", "ProxyND/2.0 Maven-Container-Proxy")
	upstreamReq.Set("X-Maven-Proxy", "ProxyND")

	// Maven 특화 헤더
	upstreamReq.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	upstreamReq.Set("Accept-Language", "en-US,en;q=0.5")

	// 클라이언트 헤더 복사 (Host 제외)
	for key, value := range c.Request().Header.All() {
		keyStr := string(key)
		if keyStr != headerHost {
			upstreamReq.Set(keyStr, string(value))
		}
	}
}

// 유틸리티 함수들
func getMavenContentType(path string) string {
	if strings.HasSuffix(path, ".jar") || strings.HasSuffix(path, ".war") || strings.HasSuffix(path, ".ear") {
		return "application/java-archive"
	}
	if strings.HasSuffix(path, ".pom") {
		return mimeApplicationXML
	}
	if strings.HasSuffix(path, ".xml") {
		return mimeApplicationXML
	}
	if strings.HasSuffix(path, ".sha1") {
		return "text/plain"
	}
	if strings.HasSuffix(path, ".md5") {
		return "text/plain"
	}
	return "application/octet-stream" //nolint:goconst // Already has constant
}

func generateSHA1(data []byte) string {
	hash := sha1.Sum(data)
	return hex.EncodeToString(hash[:])
}
