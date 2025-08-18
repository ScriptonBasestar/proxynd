package middlewares

import (
	"strings"
	"time"

	"proxynd/internal/ports"
)

// LoggingMiddleware implements ports.EnhancedHTTPMiddleware for request logging
type LoggingMiddleware struct {
	config *ports.LoggingConfig
	logger ports.Logger
}

// NewLoggingMiddleware creates a new logging middleware
func NewLoggingMiddleware(
	config *ports.LoggingConfig,
	logger ports.Logger,
) ports.EnhancedHTTPMiddleware {
	if config == nil {
		config = &ports.LoggingConfig{
			Enabled:     true,
			Format:      "json",
			SkipPaths:   []string{"/health", "/metrics"},
			LogBody:     false,
			LogHeaders:  false,
			MaxBodySize: 1024, // 1KB
		}
	}

	return &LoggingMiddleware{
		config: config,
		logger: logger,
	}
}

// Process implements ports.EnhancedHTTPMiddleware
func (m *LoggingMiddleware) Process(next ports.HTTPHandler) ports.HTTPHandler {
	return &LoggingHandler{
		next:   next,
		config: m.config,
		logger: m.logger,
	}
}

// Order returns the middleware execution order
func (m *LoggingMiddleware) Order() ports.MiddlewareOrder {
	return ports.OrderLogging
}

// Name returns the middleware name
func (m *LoggingMiddleware) Name() string {
	return "logging"
}

// IsEnabled returns whether the middleware is enabled
func (m *LoggingMiddleware) IsEnabled() bool {
	return m.config.Enabled && m.logger != nil
}

// LoggingHandler handles request logging
type LoggingHandler struct {
	next   ports.HTTPHandler
	config *ports.LoggingConfig
	logger ports.Logger
}

// Handle processes the request with logging
func (h *LoggingHandler) Handle(ctx ports.HTTPContext) error {
	// Skip logging for configured paths
	if h.shouldSkipPath(ctx.Path()) {
		return h.next.Handle(ctx)
	}

	// Skip if logging is not enabled or logger is not available
	if !h.config.Enabled || h.logger == nil {
		return h.next.Handle(ctx)
	}

	startTime := time.Now()

	// Log request
	h.logRequest(ctx, startTime)

	// Process request
	err := h.next.Handle(ctx)

	// Log response
	h.logResponse(ctx, startTime, err)

	return err
}

// shouldSkipPath checks if the path should be skipped for logging
func (h *LoggingHandler) shouldSkipPath(path string) bool {
	for _, skipPath := range h.config.SkipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}

// logRequest logs the incoming request
func (h *LoggingHandler) logRequest(ctx ports.HTTPContext, startTime time.Time) {
	fields := []ports.Field{
		&LoggingField{key: "type", value: "request"},
		&LoggingField{key: "method", value: ctx.Method()},
		&LoggingField{key: "path", value: ctx.Path()},
		&LoggingField{key: "client_ip", value: ctx.ClientIP()},
		&LoggingField{key: "user_agent", value: ctx.UserAgent()},
		&LoggingField{key: "timestamp", value: startTime},
	}

	// Add request ID if available
	if requestID := ctx.Get("request_id"); requestID != nil {
		fields = append(fields, &LoggingField{key: "request_id", value: requestID})
	}

	// Add user ID if available (from auth)
	if userID := ctx.Get("user_id"); userID != nil {
		fields = append(fields, &LoggingField{key: "user_id", value: userID})
	}

	// Add trace ID if available
	if traceID := ctx.Get("trace_id"); traceID != nil {
		fields = append(fields, &LoggingField{key: "trace_id", value: traceID})
	}

	// Log headers if enabled
	if h.config.LogHeaders {
		headers := h.extractSafeHeaders(ctx)
		if len(headers) > 0 {
			fields = append(fields, &LoggingField{key: "headers", value: headers})
		}
	}

	// Log body if enabled and safe
	if h.config.LogBody && h.shouldLogBody(ctx) {
		body := h.extractSafeBody(ctx)
		if body != "" {
			fields = append(fields, &LoggingField{key: "body", value: body})
		}
	}

	// Add query parameters
	if queryParams := h.extractQueryParams(ctx); len(queryParams) > 0 {
		fields = append(fields, &LoggingField{key: "query", value: queryParams})
	}

	h.logger.Info(ctx.Context(), "HTTP Request", fields...)
}

// logResponse logs the response
func (h *LoggingHandler) logResponse(ctx ports.HTTPContext, startTime time.Time, err error) {
	duration := time.Since(startTime)

	fields := []ports.Field{
		&LoggingField{key: "type", value: "response"},
		&LoggingField{key: "method", value: ctx.Method()},
		&LoggingField{key: "path", value: ctx.Path()},
		&LoggingField{key: "duration_ms", value: duration.Milliseconds()},
		&LoggingField{key: "duration", value: duration.String()},
	}

	// Add request ID if available
	if requestID := ctx.Get("request_id"); requestID != nil {
		fields = append(fields, &LoggingField{key: "request_id", value: requestID})
	}

	// Add status code if available
	if statusCode := ctx.Get("status_code"); statusCode != nil {
		fields = append(fields, &LoggingField{key: "status_code", value: statusCode})
	}

	// Add response size if available
	if responseSize := ctx.Get("response_size"); responseSize != nil {
		fields = append(fields, &LoggingField{key: "response_size", value: responseSize})
	}

	// Add cache status if available
	if cacheStatus := ctx.Get("cache_status"); cacheStatus != nil {
		fields = append(fields, &LoggingField{key: "cache_status", value: cacheStatus})
	}

	// Log error if present
	if err != nil {
		fields = append(fields, &LoggingField{key: "error", value: err.Error()})
		h.logger.Error(ctx.Context(), "HTTP Response Error", fields...)
	} else {
		h.logger.Info(ctx.Context(), "HTTP Response", fields...)
	}
}

// extractSafeHeaders extracts safe headers for logging
func (h *LoggingHandler) extractSafeHeaders(ctx ports.HTTPContext) map[string]string {
	safeHeaders := []string{
		"Content-Type",
		"Content-Length",
		"Accept",
		"Accept-Encoding",
		"Accept-Language",
		"Cache-Control",
		"X-Forwarded-For",
		"X-Forwarded-Proto",
		"X-Request-ID",
		"X-Trace-ID",
	}

	headers := make(map[string]string)
	for _, header := range safeHeaders {
		if value := ctx.Header(header); value != "" {
			headers[header] = value
		}
	}

	return headers
}

// extractSafeBody extracts safe body content for logging
func (h *LoggingHandler) extractSafeBody(ctx ports.HTTPContext) string {
	body := ctx.Body()
	if len(body) == 0 {
		return ""
	}

	// Limit body size
	if int64(len(body)) > h.config.MaxBodySize {
		return string(body[:h.config.MaxBodySize]) + "... [truncated]"
	}

	// Only log if content type is safe
	contentType := ctx.Header("Content-Type")
	if !h.isSafeContentType(contentType) {
		return "[binary content]"
	}

	return string(body)
}

// shouldLogBody determines if body should be logged
func (h *LoggingHandler) shouldLogBody(ctx ports.HTTPContext) bool {
	// Don't log body for GET requests
	if ctx.Method() == HTTPMethodGET {
		return false
	}

	// Don't log large bodies
	body := ctx.Body()
	if int64(len(body)) > h.config.MaxBodySize {
		return false
	}

	// Only log safe content types
	contentType := ctx.Header("Content-Type")
	return h.isSafeContentType(contentType)
}

// isSafeContentType checks if content type is safe to log
func (h *LoggingHandler) isSafeContentType(contentType string) bool {
	safeTypes := []string{
		"application/json",
		"application/xml",
		"text/plain",
		"text/xml",
		"application/x-www-form-urlencoded",
	}

	for _, safeType := range safeTypes {
		if strings.Contains(strings.ToLower(contentType), safeType) {
			return true
		}
	}

	return false
}

// extractQueryParams extracts query parameters for logging
func (h *LoggingHandler) extractQueryParams(ctx ports.HTTPContext) map[string]string {
	// This is a simplified implementation
	// In a real implementation, you might want to limit which query params are logged
	params := make(map[string]string)

	// Add common safe parameters
	safeParams := []string{"limit", "offset", "page", "size", "sort", "order", "format", "q", "query"}

	for _, param := range safeParams {
		if value := ctx.Query(param); value != "" {
			params[param] = value
		}
	}

	return params
}

// LoggingField implements ports.Field for logging
type LoggingField struct {
	key   string
	value interface{}
}

// Key returns the field key
func (f *LoggingField) Key() string {
	return f.key
}

// Value returns the field value
func (f *LoggingField) Value() interface{} {
	return f.value
}
