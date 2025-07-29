// Package handlers provides base handler implementations
package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/app"
	"proxynd/internal/config"
	"proxynd/logging"
)

// Log level constants
const (
	// LogLevelError is a const that log level error
	// LogLevelInfo is a const that log level info
	LogLevelError = "error"
	LogLevelInfo  = "info"
)

// BaseHandler 기본 핸들러 구현체
type BaseHandler struct {
	container *app.Container
	name      string
	proxyType string
	logger    logging.Logger
}

// NewBaseHandler 기본 핸들러 생성
func NewBaseHandler(container *app.Container, name, proxyType string) *BaseHandler {
	return &BaseHandler{
		container: container,
		name:      name,
		proxyType: proxyType,
		logger:    logging.GetLogger(),
	}
}

// Handle 기본 핸들러 로직 (서브클래스에서 오버라이드)
func (h *BaseHandler) Handle(c *fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
		"error":   "Handler not implemented",
		"handler": h.name,
		"type":    h.proxyType,
	})
}

// Name 핸들러 이름 반환
func (h *BaseHandler) Name() string {
	return h.name
}

// Type 핸들러 타입 반환
func (h *BaseHandler) Type() string {
	return h.proxyType
}

// GetContainer 컨테이너 반환
func (h *BaseHandler) GetContainer() *app.Container {
	return h.container
}

// GetConfig 설정 반환
func (h *BaseHandler) GetConfig() *config.UnifiedConfig {
	return h.container.GetUnifiedConfig()
}

// GetLogger 로거 반환
func (h *BaseHandler) GetLogger() logging.Logger {
	return h.logger
}

// HealthCheck 기본 헬스체크 구현
func (h *BaseHandler) HealthCheck() error {
	// 기본적으로 컨테이너와 설정이 정상인지 확인
	if h.container == nil {
		return fmt.Errorf("container is nil")
	}

	config := h.GetConfig()
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	return nil
}

// IsCacheable 기본 캐시 가능 여부 (GET 요청만 캐시)
func (h *BaseHandler) IsCacheable(c *fiber.Ctx) bool {
	return c.Method() == fiber.MethodGet
}

// GetCacheKey 기본 캐시 키 생성
func (h *BaseHandler) GetCacheKey(c *fiber.Ctx) string {
	return fmt.Sprintf("%s:%s:%s", h.proxyType, c.Method(), c.Path())
}

// RequiresAuth 기본 인증 요구 여부 (false)
func (h *BaseHandler) RequiresAuth(_ *fiber.Ctx) bool {
	return false
}

// Authenticate 기본 인증 로직 (통과)
func (h *BaseHandler) Authenticate(_ *fiber.Ctx) error {
	return nil
}

// ShouldLog 기본 로깅 여부 (모든 요청 로깅)
func (h *BaseHandler) ShouldLog(_ *fiber.Ctx) bool {
	return true
}

// GetLogLevel 기본 로그 레벨 반환
func (h *BaseHandler) GetLogLevel(c *fiber.Ctx) string {
	// 에러 상태코드는 error 레벨, 나머지는 info 레벨
	if c.Response().StatusCode() >= 400 {
		return LogLevelError
	}
	return LogLevelInfo
}

// RecordMetrics 기본 메트릭 기록
func (h *BaseHandler) RecordMetrics(c *fiber.Ctx, duration int64, statusCode int) {
	h.logger.Info("Handler metrics",
		logging.F("handler", h.name),
		logging.F("type", h.proxyType),
		logging.F("method", c.Method()),
		logging.F("path", c.Path()),
		logging.F("status", statusCode),
		logging.F("duration_ms", duration/int64(time.Millisecond)),
	)
}

// Test 기본 테스트 구현
func (h *BaseHandler) Test() error {
	// 기본적으로 헬스체크 실행
	return h.HealthCheck()
}

// Configure 기본 설정 적용 (아무것도 안함)
func (h *BaseHandler) Configure(config interface{}) error {
	h.logger.Debug("Base handler configure called",
		logging.F("handler", h.name),
		logging.F("config_type", fmt.Sprintf("%T", config)),
	)
	return nil
}

// LogRequest 요청 로깅 헬퍼
func (h *BaseHandler) LogRequest(c *fiber.Ctx) {
	if h.ShouldLog(c) {
		h.logger.Info("Request received",
			logging.F("handler", h.name),
			logging.F("method", c.Method()),
			logging.F("path", c.Path()),
			logging.F("user_agent", c.Get("User-Agent")),
			logging.F("remote_ip", c.IP()),
		)
	}
}

// LogResponse 응답 로깅 헬퍼
func (h *BaseHandler) LogResponse(c *fiber.Ctx, duration time.Duration) {
	if h.ShouldLog(c) {
		level := h.GetLogLevel(c)
		message := "Request completed"

		fields := []logging.Field{
			logging.F("handler", h.name),
			logging.F("method", c.Method()),
			logging.F("path", c.Path()),
			logging.F("status", c.Response().StatusCode()),
			logging.F("duration_ms", duration.Milliseconds()),
		}

		switch level {
		case LogLevelError:
			h.logger.Error(message, fields...)
		case "warn":
			h.logger.Warn(message, fields...)
		default:
			h.logger.Info(message, fields...)
		}
	}
}
