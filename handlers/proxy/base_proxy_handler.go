package proxy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/helpers"
	"proxynd/internal/errors"
	"proxynd/internal/security"
	"proxynd/logging"
)

// ProxyHandlerInterface 프록시 핸들러 공통 인터페이스
type ProxyHandlerInterface interface {
	// 기본 정보
	Type() string
	IsEnabled() bool

	// 캐시 관련
	GenerateCacheKey(c *fiber.Ctx) string
	ShouldCache(c *fiber.Ctx, statusCode int) bool
	GetCacheTTL(c *fiber.Ctx) time.Duration

	// 업스트림 처리 (각 구현체에서 정의)
	BuildUpstreamURL(c *fiber.Ctx) (string, error)
	FetchFromUpstream(c *fiber.Ctx, upstreamURL string) ([]byte, int, error)
	ProcessResponse(c *fiber.Ctx, body []byte, statusCode int) ([]byte, error)

	// 설정 및 에러 처리
	LoadConfig() error
	HandleError(err error, c *fiber.Ctx) error
	GetContentType(path string) string
	ShouldInline(path string) bool
}

// BaseProxyHandler 프록시 핸들러의 공통 구현체 (Template Method 패턴)
type BaseProxyHandler struct {
	logger     logging.Logger
	storageDir string
}

// NewBaseProxyHandler Base 핸들러 생성자
func NewBaseProxyHandler() *BaseProxyHandler {
	return &BaseProxyHandler{
		logger:     logging.GetLogger(),
		storageDir: helpers.GetStorageDir(),
	}
}

// Handle Template Method 패턴의 주 메서드 - 공통 플로우 정의
func (h *BaseProxyHandler) Handle(handler ProxyHandlerInterface, c *fiber.Ctx) error {
	proxyType := handler.Type()
	startTime := time.Now()

	// 요청 로깅
	h.logger.Info("Proxy request started",
		logging.F("proxy_type", proxyType),
		logging.F("path", c.Path()),
		logging.F("method", c.Method()),
		logging.F("ip", c.IP()),
	)

	defer func() {
		h.logger.Info("Proxy request completed",
			logging.F("proxy_type", proxyType),
			logging.F("duration_ms", time.Since(startTime).Milliseconds()),
		)
	}()

	// 1. 프록시 활성화 확인
	if !handler.IsEnabled() {
		return h.createProxyError("PROXY001", "프록시가 비활성화되어 있습니다", proxyType)
	}

	// 2. 설정 로드
	if err := handler.LoadConfig(); err != nil {
		h.logger.Error("Failed to load config",
			logging.F("proxy_type", proxyType),
			logging.F("error", err),
		)
		return h.createProxyError("PROXY002", "설정 로드에 실패했습니다", proxyType)
	}

	// 3. 캐시 확인
	requestPath := h.extractRequestPath(c)
	cacheInfo := h.checkCache(handler, c, requestPath)

	if cacheInfo.Hit {
		return h.serveCachedFile(handler, c, cacheInfo, proxyType)
	}

	// 4. 업스트림에서 가져오기
	body, statusCode, err := h.fetchFromUpstream(handler, c)
	if err != nil {
		// 에러 로깅
		h.LogError(err, proxyType, map[string]interface{}{
			"request_path": requestPath,
			"status_code":  statusCode,
		})
		return handler.HandleError(err, c)
	}

	// 5. 응답 처리 (프록시별 커스텀 로직)
	processedBody, err := handler.ProcessResponse(c, body, statusCode)
	if err != nil {
		h.logger.Error("Failed to process response",
			logging.F("proxy_type", proxyType),
			logging.F("error", err),
		)
		processedBody = body // 처리 실패 시 원본 사용
	}

	// 6. 캐시에 저장 (필요한 경우)
	if handler.ShouldCache(c, statusCode) {
		h.saveToCache(handler, c, requestPath, processedBody)
	}

	// 7. 응답 반환
	return h.sendResponse(handler, c, processedBody, proxyType, false)
}

// CacheInfo 캐시 정보 구조체
type CacheInfo struct {
	Hit      bool
	Path     string
	FileInfo os.FileInfo
}

// extractRequestPath 요청 경로 추출 (프록시 타입별로 다를 수 있음)
func (h *BaseProxyHandler) extractRequestPath(c *fiber.Ctx) string {
	return c.Params("*")
}

// checkCache 캐시 확인
func (h *BaseProxyHandler) checkCache(handler ProxyHandlerInterface, c *fiber.Ctx, requestPath string) CacheInfo {
	// 보안 경로 생성
	safeBasePath := filepath.Join(h.storageDir, handler.Type())
	safePath, err := security.SafeJoinPath(safeBasePath, requestPath)
	if err != nil {
		h.logger.Warn("Invalid cache path",
			logging.F("proxy_type", handler.Type()),
			logging.F("path", requestPath),
			logging.F("error", err),
		)
		return CacheInfo{Hit: false, Path: ""}
	}

	// 파일 존재 확인
	fileInfo, err := os.Stat(safePath)
	if err != nil || fileInfo.IsDir() {
		return CacheInfo{Hit: false, Path: safePath}
	}

	// TTL 확인 (필요한 경우)
	if h.isCacheExpired(handler, c, fileInfo) {
		h.logger.Debug("Cache expired",
			logging.F("proxy_type", handler.Type()),
			logging.F("path", safePath),
			logging.F("age", time.Since(fileInfo.ModTime())),
		)
		return CacheInfo{Hit: false, Path: safePath}
	}

	return CacheInfo{Hit: true, Path: safePath, FileInfo: fileInfo}
}

// isCacheExpired 캐시 만료 확인
func (h *BaseProxyHandler) isCacheExpired(handler ProxyHandlerInterface, c *fiber.Ctx, fileInfo os.FileInfo) bool {
	ttl := handler.GetCacheTTL(c)
	if ttl == 0 {
		return false // TTL이 0이면 만료되지 않음
	}
	return time.Since(fileInfo.ModTime()) > ttl
}

// serveCachedFile 캐시된 파일 제공
func (h *BaseProxyHandler) serveCachedFile(handler ProxyHandlerInterface, c *fiber.Ctx, cacheInfo CacheInfo, proxyType string) error {
	h.logger.Debug("Serving from cache",
		logging.F("proxy_type", proxyType),
		logging.F("path", cacheInfo.Path),
	)

	// 파일 읽기
	body, err := os.ReadFile(cacheInfo.Path)
	if err != nil {
		h.logger.Error("Failed to read cached file",
			logging.F("path", cacheInfo.Path),
			logging.F("error", err),
		)
		return h.createProxyError("PROXY003", "캐시 파일 읽기에 실패했습니다", proxyType)
	}

	return h.sendResponse(handler, c, body, proxyType, true)
}

// fetchFromUpstream 업스트림에서 데이터 가져오기
func (h *BaseProxyHandler) fetchFromUpstream(handler ProxyHandlerInterface, c *fiber.Ctx) ([]byte, int, error) {
	upstreamURL, err := handler.BuildUpstreamURL(c)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to build upstream URL: %w", err)
	}

	h.logger.Debug("Fetching from upstream",
		logging.F("proxy_type", handler.Type()),
		logging.F("url", upstreamURL),
	)

	return handler.FetchFromUpstream(c, upstreamURL)
}

// saveToCache 캐시에 저장
func (h *BaseProxyHandler) saveToCache(handler ProxyHandlerInterface, c *fiber.Ctx, requestPath string, body []byte) {
	safeBasePath := filepath.Join(h.storageDir, handler.Type())
	safePath, err := security.SafeJoinPath(safeBasePath, requestPath)
	if err != nil {
		h.logger.Error("Invalid cache save path",
			logging.F("proxy_type", handler.Type()),
			logging.F("error", err),
		)
		return
	}

	// 디렉토리 생성
	dir := filepath.Dir(safePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		h.logger.Error("Failed to create cache directory",
			logging.F("dir", dir),
			logging.F("error", err),
		)
		return
	}

	// 원자적 파일 쓰기 (임시 파일 사용)
	tmpFile := safePath + ".tmp"
	if err := os.WriteFile(tmpFile, body, 0o644); err != nil {
		h.logger.Error("Failed to write temp file",
			logging.F("path", tmpFile),
			logging.F("error", err),
		)
		return
	}

	if err := os.Rename(tmpFile, safePath); err != nil {
		os.Remove(tmpFile)
		h.logger.Error("Failed to rename temp file",
			logging.F("temp_path", tmpFile),
			logging.F("final_path", safePath),
			logging.F("error", err),
		)
		return
	}

	h.logger.Debug("Saved to cache",
		logging.F("proxy_type", handler.Type()),
		logging.F("path", safePath),
		logging.F("size", len(body)),
	)
}

// sendResponse 응답 전송
func (h *BaseProxyHandler) sendResponse(handler ProxyHandlerInterface, c *fiber.Ctx, body []byte, proxyType string, fromCache bool) error {
	// Content-Type 설정
	contentType := handler.GetContentType(c.Path())
	c.Set("Content-Type", contentType)

	// Content-Disposition 설정
	filename := filepath.Base(c.Path())
	if handler.ShouldInline(c.Path()) {
		c.Set("Content-Disposition", "inline")
	} else {
		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	}

	// 프록시 정보 헤더
	if fromCache {
		c.Set("X-Cache-Status", "HIT")
	} else {
		c.Set("X-Cache-Status", "MISS")
	}
	c.Set("X-Proxy-Type", proxyType)

	return c.Send(body)
}

// createProxyError 프록시 에러 생성
func (h *BaseProxyHandler) createProxyError(code, message, proxyType string) error {
	return errors.NewError(code, message).
		WithDomain(proxyType).
		Build()
}

// RecordRequestMetrics 공통 메트릭 기록
func (h *BaseProxyHandler) RecordRequestMetrics(handler ProxyHandlerInterface, c *fiber.Ctx, statusCode int, duration time.Duration) {
	h.logger.Info("Proxy request metrics",
		logging.F("proxy_type", handler.Type()),
		logging.F("status_code", statusCode),
		logging.F("duration_ms", duration.Milliseconds()),
		logging.F("path", c.Path()),
		logging.F("method", c.Method()),
		logging.F("user_agent", c.Get("User-Agent")),
		logging.F("ip", c.IP()),
	)
}

// ValidatePath 경로 검증 공통 유틸리티
func (h *BaseProxyHandler) ValidatePath(path string) error {
	// 기본 경로 검증
	if path == "" {
		return fmt.Errorf("empty path")
	}

	// Directory traversal 공격 방어
	if strings.Contains(path, "..") {
		return fmt.Errorf("path contains directory traversal")
	}

	// 특수 문자 검증
	if strings.ContainsAny(path, "<>:\"|?*") {
		return fmt.Errorf("path contains invalid characters")
	}

	return nil
}

// GetLogger 로거 반환 (하위 클래스에서 사용)
func (h *BaseProxyHandler) GetLogger() logging.Logger {
	return h.logger
}

// GetStorageDir 저장소 디렉토리 반환 (하위 클래스에서 사용)
func (h *BaseProxyHandler) GetStorageDir() string {
	return h.storageDir
}

// HandleProxyError 공통 프록시 에러 처리 (Base Handler에서 사용)
func (h *BaseProxyHandler) HandleProxyError(err error, c *fiber.Ctx) error {
	// 이미 DomainError인 경우 SendProxyError로 전송
	if _, ok := err.(*errors.DomainError); ok {
		return errors.SendProxyError(c, err)
	}

	// 일반 에러를 프록시 에러로 변환
	proxyErr := errors.NewProxyError("PROXY999", "알 수 없는 프록시 에러").
		WithCause(err).
		Build()

	return errors.SendProxyError(c, proxyErr)
}

// LogError 에러 로깅 (공통)
func (h *BaseProxyHandler) LogError(err error, proxyType string, context map[string]interface{}) {
	// 에러 심각도에 따라 로그 레벨 결정
	severity := errors.GetErrorSeverity(err)

	// 로깅 필드 구성
	fields := []logging.Field{
		logging.F("proxy_type", proxyType),
		logging.F("component", "proxy_handler"),
		logging.F("error", err.Error()),
	}

	// DomainError인 경우 추가 정보
	if domainErr, ok := err.(*errors.DomainError); ok {
		fields = append(fields,
			logging.F("error_code", domainErr.Code),
			logging.F("error_domain", domainErr.Domain),
			logging.F("error_level", domainErr.Level.String()),
		)
	}

	// 컨텍스트 정보 추가
	for k, v := range context {
		fields = append(fields, logging.F(k, v))
	}

	// 심각도에 따른 로깅
	switch severity {
	case "critical":
		h.logger.Error("Critical proxy error", fields...)
	case "error":
		h.logger.Error("Proxy error", fields...)
	case "warning":
		h.logger.Warn("Proxy warning", fields...)
	default:
		h.logger.Info("Proxy info", fields...)
	}
}
