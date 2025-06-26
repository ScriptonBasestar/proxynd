package logging

import (
	"fmt"
	"runtime/debug"
	"strings"
	"time"
	
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// MiddlewareConfig 로깅 미들웨어 설정
type MiddlewareConfig struct {
	// Logger 사용할 로거
	Logger Logger
	
	// SkipPaths 로깅을 건너뛸 경로
	SkipPaths []string
	
	// SkipStatuses 로깅을 건너뛸 상태 코드
	SkipStatuses []int
	
	// IncludeBody 요청/응답 본문 포함 여부
	IncludeBody bool
	
	// IncludeHeaders 헤더 포함 여부
	IncludeHeaders bool
	
	// HeadersToLog 로깅할 헤더 목록 (비어있으면 모든 헤더)
	HeadersToLog []string
	
	// CustomFields 추가 필드 함수
	CustomFields func(*fiber.Ctx) []Field
}

// DefaultMiddlewareConfig 기본 미들웨어 설정
var DefaultMiddlewareConfig = MiddlewareConfig{
	SkipPaths: []string{
		"/healthz",
		"/health/live",
		"/metrics",
	},
	SkipStatuses: []int{},
}

// New 구조화된 로깅 미들웨어 생성
func New(config ...MiddlewareConfig) fiber.Handler {
	cfg := DefaultMiddlewareConfig
	if len(config) > 0 {
		cfg = config[0]
	}
	
	if cfg.Logger == nil {
		cfg.Logger = GetLogger()
	}
	
	return func(c *fiber.Ctx) error {
		// 요청 ID 생성
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
			c.Set("X-Request-ID", requestID)
		}
		
		// 컨텍스트에 요청 ID 저장
		c.Locals("request_id", requestID)
		
		// 경로 확인
		if shouldSkipPath(c.Path(), cfg.SkipPaths) {
			return c.Next()
		}
		
		// 시작 시간 기록
		start := time.Now()
		
		// 요청 로거 생성
		logger := cfg.Logger.WithFields(
			F("request_id", requestID),
			F("method", c.Method()),
			F("path", c.Path()),
			F("ip", c.IP()),
			F("user_agent", c.Get("User-Agent")),
		)
		
		// 사용자 정보 추가
		if username := c.Locals("username"); username != nil {
			logger = logger.WithField("username", username)
		}
		
		// 프록시 타입 추가
		if proxyType := c.Params("type"); proxyType != "" {
			logger = logger.WithField("proxy_type", proxyType)
		}
		
		// 요청 헤더 로깅
		if cfg.IncludeHeaders {
			headers := extractHeaders(c, cfg.HeadersToLog, true)
			if len(headers) > 0 {
				logger = logger.WithField("request_headers", headers)
			}
		}
		
		// 요청 본문 크기
		if c.Request().Header.ContentLength() > 0 {
			logger = logger.WithField("request_size", c.Request().Header.ContentLength())
		}
		
		// 요청 로깅
		logger.Info("Request started")
		
		// 다음 핸들러 실행
		err := c.Next()
		
		// 응답 시간 계산
		duration := time.Since(start)
		
		// 응답 로거 생성
		respLogger := logger.WithFields(
			F("status", c.Response().StatusCode()),
			F("duration_ms", duration.Milliseconds()),
			F("duration", duration.String()),
		)
		
		// 응답 크기
		if size := len(c.Response().Body()); size > 0 {
			respLogger = respLogger.WithField("response_size", size)
		}
		
		// 캐시 상태
		if cacheHit := c.Locals("cache_hit"); cacheHit != nil {
			respLogger = respLogger.WithField("cache_hit", cacheHit)
		}
		
		// 응답 헤더 로깅
		if cfg.IncludeHeaders {
			headers := extractResponseHeaders(c, cfg.HeadersToLog)
			if len(headers) > 0 {
				respLogger = respLogger.WithField("response_headers", headers)
			}
		}
		
		// 커스텀 필드 추가
		if cfg.CustomFields != nil {
			fields := cfg.CustomFields(c)
			if len(fields) > 0 {
				respLogger = respLogger.WithFields(fields...)
			}
		}
		
		// 오류 처리
		if err != nil {
			respLogger = respLogger.WithField("error", err.Error())
		}
		
		// 상태 코드 확인
		statusCode := c.Response().StatusCode()
		if shouldSkipStatus(statusCode, cfg.SkipStatuses) {
			return err
		}
		
		// 로그 레벨 결정 및 로깅
		message := fmt.Sprintf("%s %s", c.Method(), c.Path())
		
		switch {
		case statusCode >= 500:
			respLogger.Error(message, F("error_type", "server_error"))
		case statusCode >= 400:
			respLogger.Warn(message, F("error_type", "client_error"))
		case statusCode >= 300:
			respLogger.Info(message, F("redirect", true))
		default:
			respLogger.Info(message)
		}
		
		return err
	}
}

// RequestLogger 요청별 로거를 컨텍스트에 저장하는 미들웨어
func RequestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 요청 ID 생성
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
			c.Set("X-Request-ID", requestID)
		}
		
		// 요청별 로거 생성
		logger := GetLogger().WithFields(
			F("request_id", requestID),
			F("method", c.Method()),
			F("path", c.Path()),
			F("ip", c.IP()),
		)
		
		// 컨텍스트에 로거 저장
		c.Locals("logger", logger)
		
		return c.Next()
	}
}

// GetRequestLogger 요청 컨텍스트에서 로거 추출
func GetRequestLogger(c *fiber.Ctx) Logger {
	if logger := c.Locals("logger"); logger != nil {
		if l, ok := logger.(Logger); ok {
			return l
		}
	}
	// 기본 로거 반환
	return GetLogger().WithField("fallback", true)
}

// ErrorLogger 오류 로깅 미들웨어
func ErrorLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()
		
		if err != nil {
			logger := GetRequestLogger(c)
			
			// Fiber 오류 타입 확인
			if e, ok := err.(*fiber.Error); ok {
				logger.Error("Request failed",
					F("error", e.Error()),
					F("code", e.Code),
					F("status", c.Response().StatusCode()),
				)
			} else {
				logger.Error("Request failed",
					F("error", err.Error()),
					F("status", c.Response().StatusCode()),
				)
			}
		}
		
		return err
	}
}

// RecoveryLogger 패닉 복구 및 로깅 미들웨어
func RecoveryLogger() fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				logger := GetRequestLogger(c)
				
				// 오류 메시지 추출
				var msg string
				switch v := r.(type) {
				case error:
					msg = v.Error()
				case string:
					msg = v
				default:
					msg = fmt.Sprintf("%v", v)
				}
				
				// 스택 트레이스 포함 로깅
				logger.Error("Panic recovered",
					F("panic", msg),
					F("stack", string(debug.Stack())),
				)
				
				// 500 오류 반환
				err = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal Server Error",
					"request_id": c.Locals("request_id"),
				})
			}
		}()
		
		return c.Next()
	}
}

// shouldSkipPath 경로 스킵 여부 확인
func shouldSkipPath(path string, skipPaths []string) bool {
	for _, skip := range skipPaths {
		if path == skip || strings.HasPrefix(path, skip) {
			return true
		}
	}
	return false
}

// shouldSkipStatus 상태 코드 스킵 여부 확인
func shouldSkipStatus(status int, skipStatuses []int) bool {
	for _, skip := range skipStatuses {
		if status == skip {
			return true
		}
	}
	return false
}

// extractHeaders 요청 헤더 추출
func extractHeaders(c *fiber.Ctx, headersToLog []string, isRequest bool) map[string]string {
	headers := make(map[string]string)
	
	// 민감한 헤더 목록
	sensitiveHeaders := map[string]bool{
		"authorization": true,
		"cookie":        true,
		"set-cookie":    true,
		"x-auth-token":  true,
	}
	
	if len(headersToLog) > 0 {
		// 특정 헤더만 로깅
		for _, header := range headersToLog {
			value := c.Get(header)
			if value != "" {
				if sensitiveHeaders[strings.ToLower(header)] {
					headers[header] = "***MASKED***"
				} else {
					headers[header] = value
				}
			}
		}
	} else {
		// 모든 헤더 로깅 (민감한 정보 제외)
		c.Request().Header.VisitAll(func(key, value []byte) {
			k := string(key)
			if !sensitiveHeaders[strings.ToLower(k)] {
				headers[k] = string(value)
			}
		})
	}
	
	return headers
}

// extractResponseHeaders 응답 헤더 추출
func extractResponseHeaders(c *fiber.Ctx, headersToLog []string) map[string]string {
	headers := make(map[string]string)
	
	if len(headersToLog) > 0 {
		// 특정 헤더만 로깅
		for _, header := range headersToLog {
			c.Response().Header.VisitAll(func(key, value []byte) {
				if string(key) == header {
					headers[header] = string(value)
				}
			})
		}
	}
	
	return headers
}

