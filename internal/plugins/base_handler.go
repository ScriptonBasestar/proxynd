// Package plugins provides plugin base implementations
package plugins

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/logging"
)

// ErrNotImplemented is exported
// ErrNotImplemented represents an error condition
var (
	ErrNotImplemented = errors.New("method not implemented")
	// ErrUnsupportedMode is a var that err unsupported mode
	// ErrHandlerNotReady is a var that err handler not ready
	// ErrInvalidConfig is a var that err invalid config
	ErrUnsupportedMode = errors.New("operation mode not supported")
	ErrHandlerNotReady = errors.New("handler not ready")
	ErrInvalidConfig   = errors.New("invalid configuration")
)

// BasePackageHandler 기본 패키지 핸들러 구현
// 다른 핸들러들이 상속받아 필요한 메서드만 오버라이드할 수 있습니다
type BasePackageHandler struct {
	packageType    string
	name           string
	version        string
	supportedModes map[OperationMode]bool
	cachePolicy    CachePolicy
	initialized    bool
	logger         logging.Logger
}

// NewBasePackageHandler 새로운 기본 패키지 핸들러 생성
func NewBasePackageHandler(packageType, name, version string) *BasePackageHandler {
	return &BasePackageHandler{
		packageType: packageType,
		name:        name,
		version:     version,
		supportedModes: map[OperationMode]bool{
			ProxyMode:  true,
			MirrorMode: false,
		},
		cachePolicy: CachePolicy{
			TTL:              24 * time.Hour,
			MaxSize:          1024 * 1024 * 1024, // 1GB
			EnableGzipCache:  true,
			CacheHeaders:     []string{"Content-Type", "Content-Length", "ETag", "Last-Modified"},
			SkipCacheHeaders: []string{"Authorization", "Cookie", "Set-Cookie"},
		},
		logger: logging.GetLogger(),
	}
}

// GetType 패키지 타입 반환
func (h *BasePackageHandler) GetType() string {
	return h.packageType
}

// GetName 핸들러 이름 반환
func (h *BasePackageHandler) GetName() string {
	return h.name
}

// GetVersion 핸들러 버전 반환
func (h *BasePackageHandler) GetVersion() string {
	return h.version
}

// SupportsMode 운영 모드 지원 여부 확인
func (h *BasePackageHandler) SupportsMode(mode OperationMode) bool {
	supported, exists := h.supportedModes[mode]
	return exists && supported
}

// SetSupportedModes 지원하는 운영 모드 설정
func (h *BasePackageHandler) SetSupportedModes(modes map[OperationMode]bool) {
	h.supportedModes = modes
}

// Initialize 핸들러 초기화 (기본 구현)
func (h *BasePackageHandler) Initialize(config interface{}) error {
	h.logger.Info("Initializing handler",
		logging.F("type", h.packageType),
		logging.F("name", h.name),
		logging.F("version", h.version))

	if err := h.ValidateConfig(config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	h.initialized = true
	return nil
}

// Shutdown 핸들러 종료 (기본 구현)
func (h *BasePackageHandler) Shutdown(_ context.Context) error {
	h.logger.Info("Shutting down handler",
		logging.F("type", h.packageType),
		logging.F("name", h.name))

	h.initialized = false
	return nil
}

// HandleRequest 요청 처리 (하위 클래스에서 구현 필요)
func (h *BasePackageHandler) HandleRequest(ctx *fiber.Ctx, mode OperationMode) error {
	if !h.initialized {
		return ErrHandlerNotReady
	}

	if !h.SupportsMode(mode) {
		return fmt.Errorf("mode %s: %w", mode, ErrUnsupportedMode)
	}

	h.logger.Debug("Handling request",
		logging.F("type", h.packageType),
		logging.F("mode", string(mode)),
		logging.F("path", ctx.Path()),
		logging.F("method", ctx.Method()))

	return ErrNotImplemented
}

// GetCachePolicy 캐시 정책 반환
func (h *BasePackageHandler) GetCachePolicy() CachePolicy {
	return h.cachePolicy
}

// SetCachePolicy 캐시 정책 설정
func (h *BasePackageHandler) SetCachePolicy(policy CachePolicy) {
	h.cachePolicy = policy
}

// HealthCheck 헬스체크 (기본 구현)
func (h *BasePackageHandler) HealthCheck(_ context.Context) error {
	if !h.initialized {
		return ErrHandlerNotReady
	}

	h.logger.Debug("Health check passed",
		logging.F("type", h.packageType),
		logging.F("name", h.name))

	return nil
}

// ValidateConfig 설정 검증 (기본 구현)
func (h *BasePackageHandler) ValidateConfig(_ interface{}) error {
	// nil 설정도 허용 (기본값 사용)
	// 하위 클래스에서 더 상세한 검증 구현
	return nil
}

// GetDefaultConfig 기본 설정 반환 (하위 클래스에서 구현)
func (h *BasePackageHandler) GetDefaultConfig() interface{} {
	return map[string]interface{}{
		"type":    h.packageType,
		"name":    h.name,
		"version": h.version,
		"cache": map[string]interface{}{
			"ttl":      h.cachePolicy.TTL.String(),
			"max_size": h.cachePolicy.MaxSize,
			"gzip":     h.cachePolicy.EnableGzipCache,
		},
	}
}

// IsInitialized 초기화 여부 확인
func (h *BasePackageHandler) IsInitialized() bool {
	return h.initialized
}

// GetLogger 로거 반환
func (h *BasePackageHandler) GetLogger() logging.Logger {
	return h.logger
}

// SetLogger 로거 설정
func (h *BasePackageHandler) SetLogger(logger logging.Logger) {
	h.logger = logger
}

// HandleProxyMode 프록시 모드 요청 처리 헬퍼
func (h *BasePackageHandler) HandleProxyMode(ctx *fiber.Ctx, upstreams []UpstreamConfig) error {
	if !h.SupportsMode(ProxyMode) {
		return fmt.Errorf("proxy mode: %w", ErrUnsupportedMode)
	}

	h.logger.Debug("Handling proxy mode request",
		logging.F("path", ctx.Path()),
		logging.F("upstream_count", len(upstreams)))

	// 기본 프록시 로직은 하위 클래스에서 구현
	return ErrNotImplemented
}

// HandleMirrorMode 미러 모드 요청 처리 헬퍼
func (h *BasePackageHandler) HandleMirrorMode(ctx *fiber.Ctx, mirrors []MirrorConfig) error {
	if !h.SupportsMode(MirrorMode) {
		return fmt.Errorf("mirror mode: %w", ErrUnsupportedMode)
	}

	h.logger.Debug("Handling mirror mode request",
		logging.F("path", ctx.Path()),
		logging.F("mirror_count", len(mirrors)))

	// 기본 미러 로직은 하위 클래스에서 구현
	return ErrNotImplemented
}

// GetRequestMetadata 요청 메타데이터 추출 헬퍼
func (h *BasePackageHandler) GetRequestMetadata(ctx *fiber.Ctx) map[string]string {
	metadata := make(map[string]string)

	metadata["client_ip"] = ctx.IP()
	metadata["user_agent"] = ctx.Get("User-Agent")
	metadata["method"] = ctx.Method()
	metadata["path"] = ctx.Path()
	metadata["query"] = ctx.OriginalURL()

	// 인증 정보
	if username := ctx.Locals("username"); username != nil {
		if usernameStr, ok := username.(string); ok {
			metadata["username"] = usernameStr
		}
	}

	// 추가 헤더 정보
	for _, header := range h.cachePolicy.CacheHeaders {
		if value := ctx.Get(header); value != "" {
			metadata[header] = value
		}
	}

	return metadata
}

// ShouldCacheResponse 응답 캐시 여부 확인 헬퍼
func (h *BasePackageHandler) ShouldCacheResponse(statusCode int, headers map[string]string) bool {
	// 성공 응답만 캐시
	if statusCode < 200 || statusCode >= 300 {
		return false
	}

	// 캐시하지 않을 헤더가 있는지 확인
	for _, skipHeader := range h.cachePolicy.SkipCacheHeaders {
		if _, exists := headers[skipHeader]; exists {
			return false
		}
	}

	return true
}

// BuildCacheKey 캐시 키 생성 헬퍼
func (h *BasePackageHandler) BuildCacheKey(ctx *fiber.Ctx) string {
	return fmt.Sprintf("%s:%s:%s", h.packageType, ctx.Method(), ctx.Path())
}
