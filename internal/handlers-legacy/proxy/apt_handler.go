package proxy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/config"
	"proxynd/internal/errors"
	"proxynd/internal/helpers"
	"proxynd/internal/logging"
	"proxynd/internal/security"
)

// APTHandler V1과 V2의 기능을 통합한 APT 핸들러
type APTHandler struct {
	logger     logging.Logger
	Config     *config.AptProxyConfig
	storageDir string
	mirrorIdx  int32 // 라운드로빈을 위한 atomic counter
}

// NewAPTHandler 새로운 APT 핸들러 생성
func NewAPTHandler() *APTHandler {
	return &APTHandler{
		logger:     logging.GetLogger(),
		Config:     &config.AptProxyConfig{},
		storageDir: helpers.GetStorageDir(),
		mirrorIdx:  0,
	}
}

// Type 프록시 타입 반환
func (h *APTHandler) Type() string {
	return ProxyTypeAPT
}

// IsEnabled 활성화 상태 확인
func (h *APTHandler) IsEnabled() bool {
	if err := h.Config.ReadConfig(); err != nil {
		h.logger.Error("Failed to read APT config", logging.F("error", err))
		return false
	}
	return len(h.Config.Proxies) > 0
}

// GenerateCacheKey 캐시 키 생성
func (h *APTHandler) GenerateCacheKey(c *fiber.Ctx) string {
	osType := c.Params("osType", "ubuntu")
	path := c.Params("*")
	return fmt.Sprintf("apt:%s:%s", osType, strings.ReplaceAll(path, "/", "_"))
}

// BuildUpstreamURL 여러 미러에 대한 라운드로빈 지원
func (h *APTHandler) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	if err := h.Config.ReadConfig(); err != nil {
		return "", fmt.Errorf("APT 설정 로드 실패: %w", err)
	}

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

// Handle 파일 시스템 저장 및 다중 미러 재시도를 포함한 핸들러
func (h *APTHandler) Handle(c *fiber.Ctx) error {
	proxyType := h.Type()
	startTime := time.Now()

	// 요청 로깅
	h.logger.Info("APT request started",
		logging.F("path", c.Path()),
		logging.F("method", c.Method()),
	)

	defer func() {
		h.logger.Info("APT request completed",
			logging.F("duration_ms", time.Since(startTime).Milliseconds()),
		)
	}()

	// 1. 프록시 활성화 확인
	if !h.IsEnabled() {
		return errors.NewError("PROXY001", "프록시가 비활성화되어 있습니다").
			WithDomain(proxyType).
			Build()
	}

	// 2. 파일 시스템 캐시 확인
	osType := c.Params("osType", "ubuntu")
	packagePath := c.Params("*")

	// 보안 경로 조합
	safeBasePath := filepath.Join(h.storageDir, "apt", osType)
	safePath, err := security.SafeJoinPath(safeBasePath, packagePath)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid path")
	}

	// 파일이 이미 존재하는지 확인
	if fileInfo, err := os.Stat(safePath); err == nil && !fileInfo.IsDir() {
		h.logger.Debug("Serving from cache", logging.F("path", safePath))

		// Content-Type 설정
		c.Set("Content-Type", getAptContentType(packagePath))

		// Content-Disposition 설정
		if shouldInline(packagePath) {
			c.Set("Content-Disposition", "inline")
		} else {
			filename := filepath.Base(packagePath)
			c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		}

		c.Set("X-Cache-Status", "HIT")
		c.Set("X-Proxy-Type", proxyType)

		return c.SendFile(safePath)
	}

	// 3. 업스트림에서 가져오기 (모든 미러 시도)
	proxies, exists := h.Config.Proxies[osType]
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

		// 성공적으로 가져온 경우 파일로 저장
		dir := filepath.Dir(safePath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			h.logger.Error("Failed to create directory",
				logging.F("dir", dir),
				logging.F("error", err),
			)
		} else {
			// 임시 파일에 쓰고 원자적으로 이동
			tmpFile := safePath + ".tmp"
			if err := os.WriteFile(tmpFile, body, 0o644); err == nil {
				if err := os.Rename(tmpFile, safePath); err != nil {
					_ = os.Remove(tmpFile)
					h.logger.Error("Failed to rename temp file",
						logging.F("error", err),
					)
				}
			}
		}

		// 응답 헤더 설정
		c.Set("Content-Type", getAptContentType(packagePath))

		if shouldInline(packagePath) {
			c.Set("Content-Disposition", "inline")
		} else {
			filename := filepath.Base(packagePath)
			c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		}

		c.Set("X-Cache-Status", "MISS")
		c.Set("X-Proxy-Type", proxyType)

		// 메트릭 기록
		h.RecordRequestMetrics(c, statusCode, time.Since(startTime))

		return c.Send(body)
	}

	// 모든 미러 실패
	if lastErr != nil {
		return h.HandleError(lastErr, c)
	}

	return c.Status(fiber.StatusNotFound).SendString("Failed to fetch from all mirrors")
}

// ShouldCache 캐시 정책 결정
func (h *APTHandler) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	if statusCode != 200 && statusCode != 304 {
		return false
	}

	path := c.Path()

	// 메타데이터 파일은 캐시
	if isMetadataFile(path) {
		return true
	}

	// .deb 패키지 파일은 캐시
	if strings.HasSuffix(path, ".deb") {
		return true
	}

	// .udeb 패키지 파일도 캐시
	if strings.HasSuffix(path, ".udeb") {
		return true
	}

	// 압축 파일들도 캐시
	if strings.HasSuffix(path, ".gz") || strings.HasSuffix(path, ".xz") || strings.HasSuffix(path, ".bz2") {
		return true
	}

	return false
}

// GetCacheTTL 캐시 TTL 반환
func (h *APTHandler) GetCacheTTL(c *fiber.Ctx) time.Duration {
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

// HandleError 에러 처리
func (h *APTHandler) HandleError(err error, _ *fiber.Ctx) error {
	// APT 도메인 에러로 변환
	if strings.Contains(err.Error(), "미러가 설정되지 않았습니다") {
		return errors.WrapAPTError(err, "APT002", "APT 미러 서버에 접근할 수 없습니다")
	}

	if strings.Contains(err.Error(), "설정 로드 실패") {
		return errors.WrapAPTError(err, "APT006", "APT 설정 파일을 읽을 수 없습니다")
	}

	if strings.Contains(err.Error(), "path") {
		return errors.WrapAPTError(err, "APT005", "잘못된 APT 패키지 경로입니다")
	}

	// 기본 APT 에러
	return errors.WrapAPTError(err, "APT001", "APT 패키지를 찾을 수 없습니다")
}

// RecordRequestMetrics 요청 메트릭 기록
func (h *APTHandler) RecordRequestMetrics(c *fiber.Ctx, statusCode int, duration time.Duration) {
	h.logger.Info("APT request metrics",
		logging.F("status_code", statusCode),
		logging.F("duration_ms", duration.Milliseconds()),
		logging.F("path", c.Path()),
		logging.F("method", c.Method()),
		logging.F("os_type", c.Params("osType", "ubuntu")),
	)
}

// 헬퍼 함수들

// isMetadataFile 메타데이터 파일인지 확인
func isMetadataFile(path string) bool {
	return strings.Contains(path, "Release") ||
		strings.Contains(path, "Packages") ||
		strings.Contains(path, "Sources") ||
		strings.Contains(path, "Contents") ||
		strings.HasSuffix(path, "Release.gpg") ||
		strings.HasSuffix(path, "InRelease")
}

// shouldInline 인라인으로 표시할 파일인지 확인
func shouldInline(path string) bool {
	return strings.Contains(path, "Release") ||
		strings.Contains(path, "Packages") ||
		strings.Contains(path, "Sources") ||
		strings.Contains(path, "Contents")
}
