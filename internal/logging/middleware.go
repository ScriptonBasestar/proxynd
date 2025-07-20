package logging

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// MiddlewareConfig represents the configuration for middleware settings
type MiddlewareConfig struct {
	Logger          Logger
	SkipPaths       []string
	SkipSuccessLogs bool
	LogRequestBody  bool
	LogResponseBody bool
	MaxBodySize     int
	TimeFormat      string
	CustomFields    func(*fiber.Ctx) map[string]interface{}
}

// DefaultMiddlewareConfig provides the default middleware configuration
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

		// Initialize request context
		requestData := initializeRequestContext(c)

		// Process request and capture response
		start := time.Now()
		err := c.Next()
		duration := time.Since(start)

		// Log the request
		logRequest(cfg, c, requestData, duration, err)

		return err
	}
}

// requestData holds the request-specific data for logging
type requestData struct {
	requestID     string
	correlationID string
	userID        string
	sessionID     string
	ctx           context.Context
}

// initializeRequestContext sets up request context and IDs
func initializeRequestContext(c *fiber.Ctx) *requestData {
	data := &requestData{}

	// Generate or get request ID
	data.requestID = getOrGenerateID(c, "X-Request-ID")

	// Generate or get correlation ID
	data.correlationID = getOrGenerateID(c, "X-Correlation-ID")

	// Build context with IDs
	ctx := WithRequestID(c.Context(), data.requestID)
	ctx = WithCorrelationID(ctx, data.correlationID)

	// Extract optional IDs
	data.userID = c.Get("X-User-ID")
	if data.userID != "" {
		ctx = WithUserID(ctx, data.userID)
	}

	data.sessionID = c.Get("X-Session-ID")
	if data.sessionID != "" {
		ctx = WithSessionID(ctx, data.sessionID)
	}

	// Store context
	c.SetUserContext(ctx)
	data.ctx = ctx

	return data
}

// getOrGenerateID gets existing ID from header or generates new one
func getOrGenerateID(c *fiber.Ctx, headerName string) string {
	id := c.Get(headerName)
	if id == "" {
		id = uuid.New().String()
		c.Set(headerName, id)
	}
	return id
}

// captureRequestBody captures request body if enabled
func captureRequestBody(cfg MiddlewareConfig, c *fiber.Ctx) []byte {
	if !cfg.LogRequestBody || len(c.Body()) == 0 || len(c.Body()) > cfg.MaxBodySize {
		return nil
	}

	body := make([]byte, len(c.Body()))
	copy(body, c.Body())
	return body
}

// captureResponseBody captures response body if enabled
func captureResponseBody(cfg MiddlewareConfig, c *fiber.Ctx) []byte {
	if !cfg.LogResponseBody || len(c.Response().Body()) == 0 || len(c.Response().Body()) > cfg.MaxBodySize {
		return nil
	}

	body := make([]byte, len(c.Response().Body()))
	copy(body, c.Response().Body())
	return body
}

// logRequest performs the actual logging
func logRequest(cfg MiddlewareConfig, c *fiber.Ctx, data *requestData, duration time.Duration, err error) {
	status := c.Response().StatusCode()

	// Skip success logs if configured
	if cfg.SkipSuccessLogs && status >= 200 && status < 300 {
		return
	}

	// Build log fields
	fields := buildBaseFields(c, data, duration, status)

	// Add optional fields
	fields = addOptionalFields(fields, c, data)

	// Capture and add bodies if enabled
	if requestBody := captureRequestBody(cfg, c); len(requestBody) > 0 {
		fields = append(fields, String("request_body", string(requestBody)))
	}

	if responseBody := captureResponseBody(cfg, c); len(responseBody) > 0 {
		fields = append(fields, String("response_body", string(responseBody)))
	}

	// Add custom fields
	if cfg.CustomFields != nil {
		addCustomFields(&fields, cfg.CustomFields(c))
	}

	// Add error information
	if err != nil {
		fields = append(fields, Error(err))
		if status >= 500 {
			fields = append(fields, StackTrace(err))
		}
	}

	// Log with appropriate level
	logWithLevel(cfg.Logger.WithContext(data.ctx).WithComponent("http.middleware"), status, err, fields)
}

// buildBaseFields creates the base set of log fields
func buildBaseFields(c *fiber.Ctx, data *requestData, duration time.Duration, status int) []Field {
	return []Field{
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
		String("request_id", data.requestID),
		String("correlation_id", data.correlationID),
	}
}

// addOptionalFields adds optional fields to the log
func addOptionalFields(fields []Field, c *fiber.Ctx, data *requestData) []Field {
	// Add user and session IDs if available
	if data.userID != "" {
		fields = append(fields, String("user_id", data.userID))
	}
	if data.sessionID != "" {
		fields = append(fields, String("session_id", data.sessionID))
	}

	// Add query parameters
	if len(c.Queries()) > 0 {
		fields = append(fields, NewField("query", c.Queries()))
	}

	// Add safe headers
	if headers := collectSafeHeaders(c); len(headers) > 0 {
		fields = append(fields, NewField("headers", headers))
	}

	return fields
}

// collectSafeHeaders collects headers that are safe to log
func collectSafeHeaders(c *fiber.Ctx) map[string]string {
	headers := make(map[string]string)
	for key, value := range c.Request().Header.All() {
		headerKey := string(key)
		if isSafeHeader(headerKey) {
			headers[headerKey] = string(value)
		}
	}
	return headers
}

// addCustomFields adds custom fields to the log fields
func addCustomFields(fields *[]Field, customFields map[string]interface{}) {
	for key, value := range customFields {
		*fields = append(*fields, NewField(key, value))
	}
}

// logWithLevel logs the message with the appropriate level based on status
func logWithLevel(logger Logger, status int, err error, fields []Field) {
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
}

// isSafeHeader checks if a header is safe to log
func isSafeHeader(header string) bool {
	// List of headers that are safe to log
	safeHeaders := map[string]bool{
		"Content-Type":     true,
		"Content-Length":   true,
		"Accept":           true,
		"Accept-Encoding":  true,
		"Accept-Language":  true,
		"Cache-Control":    true,
		"Connection":       true,
		"Host":             true,
		"Origin":           true,
		"Referer":          true,
		"User-Agent":       true,
		"X-Forwarded-For":  true,
		"X-Real-IP":        true,
		"X-Request-ID":     true,
		"X-Correlation-ID": true,
		"X-User-ID":        true,
		"X-Session-ID":     true,
	}

	// Don't log sensitive headers
	sensitiveHeaders := map[string]bool{
		"Authorization":  true,
		"Cookie":         true,
		"Set-Cookie":     true,
		"X-API-Key":      true,
		"X-Auth-Token":   true,
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
