package middlewares

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/errors"
	"proxynd/logging"
)

// ErrorResponse 에러 응답 구조체
type ErrorResponse struct {
	Error     string      `json:"error"`
	Message   string      `json:"message"`
	Domain    string      `json:"domain,omitempty"`
	Details   interface{} `json:"details,omitempty"`
	TraceID   string      `json:"trace_id,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// ErrorHandler 에러 핸들링 미들웨어
type ErrorHandlerMiddleware struct {
	logger logging.Logger
}

// NewErrorHandler 새로운 에러 핸들러 생성
func NewErrorHandler() *ErrorHandlerMiddleware {
	return &ErrorHandlerMiddleware{
		logger: logging.GetLogger(),
	}
}

// ErrorHandler 에러 핸들링 미들웨어 함수
func ErrorHandler() fiber.Handler {
	handler := NewErrorHandler()

	return func(c *fiber.Ctx) error {
		// 다음 핸들러 실행
		err := c.Next()
		if err == nil {
			return nil
		}

		// 추적 ID 가져오기 (RequestID 미들웨어에서 설정)
		traceID := c.Locals("requestID")
		if traceID == nil {
			traceID = "unknown"
		}

		// 도메인 에러 처리
		if domainErr, ok := err.(*errors.DomainError); ok {
			return handler.handleDomainError(c, domainErr, traceID.(string))
		}

		// Fiber 에러 처리
		if fiberErr, ok := err.(*fiber.Error); ok {
			return handler.handleFiberError(c, fiberErr, traceID.(string))
		}

		// 기본 에러 처리
		return handler.handleGenericError(c, err, traceID.(string))
	}
}

// handleDomainError 도메인 에러 처리
func (h *ErrorHandlerMiddleware) handleDomainError(c *fiber.Ctx, err *errors.DomainError, traceID string) error {
	// 구조화된 로깅
	h.logError(c, err, traceID)

	// HTTP 상태 코드 결정
	status := errors.GetHTTPStatus(err)

	// 응답 생성
	response := ErrorResponse{
		Error:     err.Code,
		Message:   err.Message,
		Domain:    err.Domain,
		Details:   err.Details,
		TraceID:   traceID,
		Timestamp: err.Timestamp,
	}

	return c.Status(status).JSON(response)
}

// handleFiberError Fiber 에러 처리
func (h *ErrorHandlerMiddleware) handleFiberError(c *fiber.Ctx, err *fiber.Error, traceID string) error {
	// Fiber 에러 로깅
	h.logger.Warn("Fiber error occurred",
		logging.F("trace_id", traceID),
		logging.F("code", err.Code),
		logging.F("message", err.Message),
		logging.F("path", c.Path()),
		logging.F("method", c.Method()),
		logging.F("ip", c.IP()),
	)

	response := ErrorResponse{
		Error:     "FIBER_ERROR",
		Message:   err.Message,
		TraceID:   traceID,
		Timestamp: time.Now(),
	}

	return c.Status(err.Code).JSON(response)
}

// handleGenericError 일반 에러 처리
func (h *ErrorHandlerMiddleware) handleGenericError(c *fiber.Ctx, err error, traceID string) error {
	// 알 수 없는 에러 로깅
	h.logger.Error("Unknown error occurred",
		logging.F("trace_id", traceID),
		logging.F("error", err.Error()),
		logging.F("path", c.Path()),
		logging.F("method", c.Method()),
		logging.F("ip", c.IP()),
		logging.F("user_agent", c.Get("User-Agent")),
	)

	response := ErrorResponse{
		Error:     "INTERNAL_ERROR",
		Message:   "내부 서버 오류가 발생했습니다",
		TraceID:   traceID,
		Timestamp: time.Now(),
	}

	return c.Status(fiber.StatusInternalServerError).JSON(response)
}

// logError 도메인 에러 로깅 (구조화된 형식)
func (h *ErrorHandlerMiddleware) logError(c *fiber.Ctx, err *errors.DomainError, traceID string) {
	logLevel := h.getLogLevel(err.Level)

	fields := []logging.Field{
		logging.F("trace_id", traceID),
		logging.F("domain", err.Domain),
		logging.F("code", err.Code),
		logging.F("message", err.Message),
		logging.F("path", c.Path()),
		logging.F("method", c.Method()),
		logging.F("ip", c.IP()),
		logging.F("user_agent", c.Get("User-Agent")),
		logging.F("timestamp", err.Timestamp.Format(time.RFC3339)),
	}

	if err.Details != nil {
		fields = append(fields, logging.F("details", err.Details))
	}

	if err.Cause != nil {
		fields = append(fields, logging.F("cause", err.Cause.Error()))
	}

	message := "Domain error occurred"

	switch logLevel {
	case "info":
		h.logger.Info(message, fields...)
	case "warning":
		h.logger.Warn(message, fields...)
	case "error":
		h.logger.Error(message, fields...)
	case "critical":
		h.logger.Error(message, fields...)
	default:
		h.logger.Error(message, fields...)
	}
}

// getLogLevel 에러 레벨을 로그 레벨로 변환
func (h *ErrorHandlerMiddleware) getLogLevel(level errors.ErrorLevel) string {
	switch level {
	case errors.ErrorLevelInfo:
		return "info"
	case errors.ErrorLevelWarning:
		return "warning"
	case errors.ErrorLevelError:
		return "error"
	case errors.ErrorLevelCritical:
		return "critical"
	default:
		return "error"
	}
}

// LegacyErrorHandler 레거시 JSON 에러 응답용 (기존 코드 호환성)
func LegacyErrorHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()
		if err == nil {
			return nil
		}

		// 다른 에러는 기본 핸들러로 전달
		return err
	}
}
