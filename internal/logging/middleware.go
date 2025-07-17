package logging

import (
	"bytes"
	"io"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Config for logging middleware
type MiddlewareConfig struct {
	Logger           Logger
	SkipPaths        []string
	SkipSuccessLogs  bool
	LogRequestBody   bool
	LogResponseBody  bool
	MaxBodySize      int
	TimeFormat       string
	CustomFields     func(*fiber.Ctx) map[string]interface{}
}

// Default configuration
var DefaultMiddlewareConfig = MiddlewareConfig{
	SkipPaths:       []string{"/health", "/metrics"},
	SkipSuccessLogs: false,
	LogRequestBody:  false,
	LogResponseBody: false,
	MaxBodySize:     1024, // 1KB
	TimeFormat:      time.RFC3339Nano,
}

// New creates a new logging middleware
func New(config ...MiddlewareConfig) fiber.Handler {
	cfg := DefaultMiddlewareConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	// Skip paths map for faster lookup
	skipPaths := make(map[string]bool)
	for _, path := range cfg.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *fiber.Ctx) error {
		// Skip logging for certain paths
		if skipPaths[c.Path()] {
			return c.Next()
		}

		start := time.Now()
		
		// Generate request ID if not present
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
			c.Set("X-Request-ID", requestID)
		}

		// Generate correlation ID if not present
		correlationID := c.Get("X-Correlation-ID")
		if correlationID == "" {
			correlationID = uuid.New().String()
			c.Set("X-Correlation-ID", correlationID)
		}

		// Add IDs to context
		ctx := WithRequestID(c.Context(), requestID)
		ctx = WithCorrelationID(ctx, correlationID)
		
		// Extract user ID from headers or context
		userID := c.Get("X-User-ID")
		if userID != "" {
			ctx = WithUserID(ctx, userID)
		}
		
		// Extract session ID
		sessionID := c.Get("X-Session-ID")
		if sessionID != "" {
			ctx = WithSessionID(ctx, sessionID)
		}

		// Store context
		c.SetUserContext(ctx)

		// Capture request body if enabled
		var requestBody []byte
		if cfg.LogRequestBody && len(c.Body()) > 0 && len(c.Body()) <= cfg.MaxBodySize {
			requestBody = make([]byte, len(c.Body()))
			copy(requestBody, c.Body())
		}

		// Create response body capturer
		var responseBody []byte
		if cfg.LogResponseBody {
			originalWrite := c.Response().BodyWriter()
			bodyCapture := &bodyWriter{
				ResponseWriter: originalWrite,
				body:          &bytes.Buffer{},
				maxSize:       cfg.MaxBodySize,
			}
			c.Response().SetBodyWriter(bodyCapture)
			defer func() {
				responseBody = bodyCapture.body.Bytes()
			}()
		}

		// Process request
		err := c.Next()
		
		// Calculate duration
		duration := time.Since(start)
		status := c.Response().StatusCode()

		// Skip success logs if configured
		if cfg.SkipSuccessLogs && status >= 200 && status < 300 {
			return err
		}

		// Prepare log fields
		fields := []Field{
			String("method", c.Method()),
			String("path", c.Path()),
			String("route", c.Route().Path),
			Int("status", status),
			Duration("duration", duration),
			String("ip", c.IP()),
			String("user_agent", c.Get("User-Agent")),
			String("referer", c.Get("Referer")),
			Int("request_size", len(c.Body())),
			Int("response_size", len(c.Response().Body())),
			String("request_id", requestID),
			String("correlation_id", correlationID),
		}

		// Add user and session IDs if available
		if userID != "" {
			fields = append(fields, String("user_id", userID))
		}
		if sessionID != "" {
			fields = append(fields, String("session_id", sessionID))
		}

		// Add query parameters
		if len(c.Queries()) > 0 {
			fields = append(fields, NewField("query", c.Queries()))
		}

		// Add request headers (selective)
		headers := make(map[string]string)
		c.Request().Header.VisitAll(func(key, value []byte) {
			headerKey := string(key)
			// Only log safe headers
			if isSafeHeader(headerKey) {
				headers[headerKey] = string(value)
			}
		})
		if len(headers) > 0 {
			fields = append(fields, NewField("headers", headers))
		}

		// Add request body if captured
		if len(requestBody) > 0 {
			fields = append(fields, String("request_body", string(requestBody)))
		}

		// Add response body if captured
		if len(responseBody) > 0 {
			fields = append(fields, String("response_body", string(responseBody)))
		}

		// Add custom fields if provided
		if cfg.CustomFields != nil {
			customFields := cfg.CustomFields(c)
			for key, value := range customFields {
				fields = append(fields, NewField(key, value))
			}
		}

		// Add error information if present
		if err != nil {
			fields = append(fields, Error(err))
			if status >= 500 {
				fields = append(fields, StackTrace(err))
			}
		}

		// Log the request
		logger := cfg.Logger.WithContext(ctx).WithComponent("http.middleware")
		
		message := "HTTP request processed"
		if err != nil {
			message = "HTTP request failed"
		}

		switch {
		case status >= 500:
			logger.Error(message, fields...)
		case status >= 400:
			logger.Warn(message, fields...)
		default:
			logger.Info(message, fields...)
		}

		return err
	}
}

// bodyWriter captures response body
type bodyWriter struct {
	io.Writer
	ResponseWriter io.Writer
	body          *bytes.Buffer
	maxSize       int
}

func (bw *bodyWriter) Write(b []byte) (int, error) {
	// Write to original response
	n, err := bw.ResponseWriter.Write(b)
	
	// Capture body if under size limit
	if bw.body.Len()+len(b) <= bw.maxSize {
		bw.body.Write(b[:n])
	}
	
	return n, err
}

// isSafeHeader checks if a header is safe to log
func isSafeHeader(header string) bool {
	// List of headers that are safe to log
	safeHeaders := map[string]bool{
		"Content-Type":    true,
		"Content-Length":  true,
		"Accept":          true,
		"Accept-Encoding": true,
		"Accept-Language": true,
		"Cache-Control":   true,
		"Connection":      true,
		"Host":            true,
		"Origin":          true,
		"Referer":         true,
		"User-Agent":      true,
		"X-Forwarded-For": true,
		"X-Real-IP":       true,
		"X-Request-ID":    true,
		"X-Correlation-ID": true,
		"X-User-ID":       true,
		"X-Session-ID":    true,
	}

	// Don't log sensitive headers
	sensitiveHeaders := map[string]bool{
		"Authorization": true,
		"Cookie":        true,
		"Set-Cookie":    true,
		"X-API-Key":     true,
		"X-Auth-Token":  true,
		"Authentication": true,
	}

	if sensitiveHeaders[header] {
		return false
	}

	return safeHeaders[header]
}

// RequestLoggerFromContext gets the logger with request context
func RequestLoggerFromContext(c *fiber.Ctx, baseLogger Logger) Logger {
	return baseLogger.WithContext(c.UserContext())
}