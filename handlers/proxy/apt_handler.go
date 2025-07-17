// Package proxy provides HTTP handlers for various package manager proxies.
package proxy

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/configs"
	"proxynd/helpers"
	"proxynd/internal/errors"
	"proxynd/internal/security"
	"proxynd/logging"
)

// APTHandler APT 패키지 매니저 프록시 핸들러
// Handler 기본 핸들러 인터페이스 (inline to avoid import cycle)
type Handler interface {
	Handle(c *fiber.Ctx) error
	Name() string
	Type() string
}

type APTHandler struct {
	client    *http.Client
	mirrorIdx int // 라운드로빈을 위한 인덱스
	name      string
	proxyType string
	logger    logging.Logger
}

// NewAPTHandler 새로운 APT 핸들러 생성
func NewAPTHandler() *APTHandler {
	// 기본 HTTP 클라이언트 생성
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	handler := &APTHandler{
		client:    httpClient,
		mirrorIdx: 0,
		name:      "apt-proxy",
		proxyType: "apt",
		logger:    logging.GetLogger(),
	}

	return handler
}

// Handle APT 프록시 요청 처리
func (h *APTHandler) Handle(c *fiber.Ctx) error {
	h.logRequest(c)
	start := time.Now()

	// 설정 로드
	config := &configs.AptProxyConfig{}
	if err := config.ReadConfig(); err != nil {
		h.logger.Error("Failed to read APT config", logging.F("error", err))
		return errors.WrapAPTError(err, "APT006", "APT 설정 파일을 읽을 수 없습니다")
	}

	if len(config.Proxies) == 0 {
		return errors.ErrAPTProxyDisabled
	}

	// 요청 경로 파싱
	osType := c.Params("osType")
	packagePath := c.Params("*")

	// 보안 체크
	if err := h.validatePath(packagePath); err != nil {
		return errors.WrapAPTError(err, "APT005", "잘못된 APT 패키지 경로입니다")
	}

	// 캐시 키 생성 (미래 사용을 위해)
	_ = h.generateCacheKey(osType, packagePath)

	// 캐시 확인 (현재는 비활성화)
	// 캐시 기능은 추후 구현

	// OS별 프록시 설정 확인
	proxies, exists := config.Proxies[osType]
	if !exists || len(proxies) == 0 {
		return errors.NewAPTError("APT002", 
			fmt.Sprintf("OS 타입 '%s'에 대한 APT 미러가 설정되지 않았습니다", osType)).Build()
	}

	// 파일 경로 생성
	storageDir := helpers.GetStorageDir()
	baseDir := filepath.Join(storageDir, config.Path)
	filePath, err := security.SafeJoinPath(baseDir, packagePath)
	if err != nil {
		return errors.WrapAPTError(err, "APT005", "잘못된 APT 패키지 경로입니다")
	}

	// 업스트림에서 다운로드
	if err := h.downloadFromUpstream(filePath, proxies, packagePath); err != nil {
		h.logger.Error("Failed to download from upstream",
			logging.F("path", packagePath),
			logging.F("error", err),
		)
		return errors.WrapAPTError(err, "APT002", "APT 미러 서버에 접근할 수 없습니다")
	}

	// 캐시에 저장 (현재는 비활성화)

	c.Set("X-Cache-Status", "MISS")
	h.logResponse(c, time.Since(start))
	return h.sendFile(c, filePath, filepath.Base(packagePath))
}

// validatePath 경로 유효성 검사
func (h *APTHandler) validatePath(path string) error {
	// 상위 디렉토리 접근 방지
	if strings.Contains(path, "..") {
		return fmt.Errorf("invalid path: contains '..'")
	}
	// 절대 경로 방지
	if strings.HasPrefix(path, "/") {
		return fmt.Errorf("invalid path: absolute path not allowed")
	}
	return nil
}

// generateCacheKey 캐시 키 생성
func (h *APTHandler) generateCacheKey(osType, path string) string {
	return fmt.Sprintf("apt:%s:%s", osType, strings.ReplaceAll(path, "/", "_"))
}


// downloadFromUpstream 업스트림에서 파일 다운로드
func (h *APTHandler) downloadFromUpstream(filePath string, proxies []configs.AptProxy, packagePath string) error {
	// 디렉토리 생성
	dirPath := filepath.Dir(filePath)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 여러 미러 시도
	var lastErr error
	for _, proxy := range proxies {
		if err := h.tryDownloadFromMirror(filePath, proxy.URL, packagePath); err == nil {
			return nil // 성공
		} else {
			lastErr = err
			h.logger.Warn("Mirror download failed, trying next",
				logging.F("mirror", proxy.URL),
				logging.F("error", err),
			)
		}
	}

	return fmt.Errorf("all mirrors failed: %w", lastErr)
}

// tryDownloadFromMirror 특정 미러에서 다운로드 시도
func (h *APTHandler) tryDownloadFromMirror(filePath, mirrorURL, packagePath string) error {
	// URL 생성
	downloadURL := helpers.JoinURL(mirrorURL, packagePath)

	// HTTP 요청
	resp, err := h.client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to fetch from mirror: %w", err)
	}
	defer resp.Body.Close()

	// 상태 코드 확인
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	// 파일 생성
	out, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// 데이터 복사
	if _, err := io.Copy(out, resp.Body); err != nil {
		os.Remove(filePath) // 실패 시 파일 삭제
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}


// sendFile 파일 전송
func (h *APTHandler) sendFile(c *fiber.Ctx, filePath, filename string) error {
	// Content-Type 설정
	contentType := getAptContentType(filename)
	c.Set("Content-Type", contentType)

	// Content-Disposition 설정
	if isInlineFile(filename) {
		c.Set("Content-Disposition", fmt.Sprintf("inline; filename=%s", filename))
	} else {
		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	}

	return c.SendFile(filePath)
}

// Name 핸들러 이름 반환
func (h *APTHandler) Name() string {
	return h.name
}

// Type 핸들러 타입 반환
func (h *APTHandler) Type() string {
	return h.proxyType
}

// logRequest 요청 로깅
func (h *APTHandler) logRequest(c *fiber.Ctx) {
	h.logger.Info("Request received",
		logging.F("handler", h.name),
		logging.F("method", c.Method()),
		logging.F("path", c.Path()),
		logging.F("user_agent", c.Get("User-Agent")),
		logging.F("remote_ip", c.IP()),
	)
}

// logResponse 응답 로깅
func (h *APTHandler) logResponse(c *fiber.Ctx, duration time.Duration) {
	level := "info"
	if c.Response().StatusCode() >= 400 {
		level = "error"
	}

	message := "Request completed"
	fields := []logging.Field{
		logging.F("handler", h.name),
		logging.F("method", c.Method()),
		logging.F("path", c.Path()),
		logging.F("status", c.Response().StatusCode()),
		logging.F("duration_ms", duration.Milliseconds()),
	}

	switch level {
	case "error":
		h.logger.Error(message, fields...)
	default:
		h.logger.Info(message, fields...)
	}
}

// HealthCheck APT 핸들러 헬스체크
func (h *APTHandler) HealthCheck() error {
	// APT 설정 확인
	config := &configs.AptProxyConfig{}
	if err := config.ReadConfig(); err != nil {
		return fmt.Errorf("failed to read APT configuration: %w", err)
	}

	if len(config.Proxies) == 0 {
		return fmt.Errorf("APT proxy has no configured mirrors")
	}

	// 최소 하나의 프록시가 설정되어 있는지 확인
	hasProxy := false
	for _, proxies := range config.Proxies {
		if len(proxies) > 0 {
			hasProxy = true
			break
		}
	}

	if !hasProxy {
		return fmt.Errorf("no upstream proxies configured")
	}

	return nil
}