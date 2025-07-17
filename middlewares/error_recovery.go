package middlewares

import (
	"fmt"
	"runtime/debug"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/errors"
	"proxynd/logging"
)

// ErrorRecoveryMiddleware 패닉 복구 미들웨어
type ErrorRecoveryMiddleware struct {
	logger logging.Logger
}

// NewErrorRecovery 새로운 에러 복구 미들웨어 생성
func NewErrorRecovery() *ErrorRecoveryMiddleware {
	return &ErrorRecoveryMiddleware{
		logger: logging.GetLogger(),
	}
}

// ErrorRecovery 패닉 복구 미들웨어 함수
func ErrorRecovery() fiber.Handler {
	recovery := NewErrorRecovery()

	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				// 스택 트레이스 캡처
				stack := debug.Stack()

				// 추적 ID 가져오기
				traceID := c.Locals("requestID")
				if traceID == nil {
					traceID = "unknown"
				}

				// 패닉 정보 로깅
				recovery.logPanic(c, r, stack, traceID.(string))

				// 패닉 에러 생성
				panicErr := errors.NewError("PANIC001", "서버 패닉이 발생했습니다").
					WithDomain("system").
					WithLevel(errors.ErrorLevelCritical).
					WithDetails(map[string]interface{}{
						"panic":    fmt.Sprintf("%v", r),
						"stack":    string(stack),
						"trace_id": traceID,
					}).
					Build()

				// 패닉 에러를 컨텍스트에 저장하고 에러 핸들러로 전달
				c.Locals("panic_error", panicErr)

				// 에러 응답 생성
				response := ErrorResponse{
					Error:     panicErr.Code,
					Message:   panicErr.Message,
					Domain:    panicErr.Domain,
					TraceID:   traceID.(string),
					Timestamp: panicErr.Timestamp,
				}

				// 패닉 상황에서는 직접 응답
				c.Status(fiber.StatusInternalServerError).JSON(response)
			}
		}()

		return c.Next()
	}
}

// logPanic 패닉 정보 로깅
func (m *ErrorRecoveryMiddleware) logPanic(c *fiber.Ctx, panicValue interface{}, stack []byte, traceID string) {
	m.logger.Error("Panic recovered",
		logging.F("trace_id", traceID),
		logging.F("panic", fmt.Sprintf("%v", panicValue)),
		logging.F("path", c.Path()),
		logging.F("method", c.Method()),
		logging.F("ip", c.IP()),
		logging.F("user_agent", c.Get("User-Agent")),
		logging.F("headers", c.GetReqHeaders()),
		logging.F("stack_trace", string(stack)),
		logging.F("timestamp", time.Now().Format(time.RFC3339)),
	)
}

// RecoveryConfig 복구 미들웨어 설정
type RecoveryConfig struct {
	// EnableStackTrace 스택 트레이스 포함 여부
	EnableStackTrace bool

	// StackTraceHandler 커스텀 스택 트레이스 핸들러
	StackTraceHandler func(c *fiber.Ctx, e interface{})
}

// RecoveryWithConfig 설정 가능한 복구 미들웨어
func RecoveryWithConfig(config RecoveryConfig) fiber.Handler {
	recovery := NewErrorRecovery()

	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				traceID := c.Locals("requestID")
				if traceID == nil {
					traceID = "unknown"
				}

				// 커스텀 스택 트레이스 핸들러 실행
				if config.StackTraceHandler != nil {
					config.StackTraceHandler(c, r)
				}

				// 기본 패닉 로깅
				if config.EnableStackTrace {
					stack := debug.Stack()
					recovery.logPanic(c, r, stack, traceID.(string))
				} else {
					recovery.logger.Error("Panic recovered (stack trace disabled)",
						logging.F("trace_id", traceID),
						logging.F("panic", fmt.Sprintf("%v", r)),
						logging.F("path", c.Path()),
						logging.F("method", c.Method()),
					)
				}

				// 패닉 에러 생성
				panicDetails := map[string]interface{}{
					"panic":    fmt.Sprintf("%v", r),
					"trace_id": traceID,
				}

				if config.EnableStackTrace {
					panicDetails["stack"] = string(debug.Stack())
				}

				panicErr := errors.NewError("PANIC001", "서버 패닉이 발생했습니다").
					WithDomain("system").
					WithLevel(errors.ErrorLevelCritical).
					WithDetails(panicDetails).
					Build()

				response := ErrorResponse{
					Error:     panicErr.Code,
					Message:   panicErr.Message,
					Domain:    panicErr.Domain,
					TraceID:   traceID.(string),
					Timestamp: panicErr.Timestamp,
				}

				c.Status(fiber.StatusInternalServerError).JSON(response)
			}
		}()

		return c.Next()
	}
}
