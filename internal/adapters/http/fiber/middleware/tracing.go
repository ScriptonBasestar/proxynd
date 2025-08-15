package middlewares

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"proxynd/internal/ports"
)

// TracingMiddleware implements ports.HTTPMiddleware for distributed tracing
type TracingMiddleware struct {
	config  *ports.TracingConfig
	tracer  ports.TraceService
	logger  ports.Logger
}

// NewTracingMiddleware creates a new tracing middleware
func NewTracingMiddleware(
	config *ports.TracingConfig,
	tracer ports.TraceService,
	logger ports.Logger,
) ports.EnhancedHTTPMiddleware {
	if config == nil {
		config = &ports.TracingConfig{
			Enabled:       false,
			ServiceName:   "proxynd",
			SampleRate:    0.1,
			SkipPaths:     []string{"/health", "/metrics"},
			TraceIDHeader: "X-Trace-ID",
		}
	}
	
	return &TracingMiddleware{
		config: config,
		tracer: tracer,
		logger: logger,
	}
}

// Process implements ports.HTTPMiddleware
func (m *TracingMiddleware) Process(next ports.HTTPHandler) ports.HTTPHandler {
	return &TracingHandler{
		next:   next,
		config: m.config,
		tracer: m.tracer,
		logger: m.logger,
	}
}

// Order returns the middleware execution order
func (m *TracingMiddleware) Order() ports.MiddlewareOrder {
	return ports.OrderTracing
}

// Name returns the middleware name
func (m *TracingMiddleware) Name() string {
	return "tracing"
}

// IsEnabled returns whether the middleware is enabled
func (m *TracingMiddleware) IsEnabled() bool {
	return m.config.Enabled && m.tracer != nil && m.tracer.IsEnabled()
}

// TracingHandler handles distributed tracing
type TracingHandler struct {
	next   ports.HTTPHandler
	config *ports.TracingConfig
	tracer ports.TraceService
	logger ports.Logger
}

// Handle processes the request with tracing
func (h *TracingHandler) Handle(ctx ports.HTTPContext) error {
	// Skip tracing for configured paths
	if h.shouldSkipPath(ctx.Path()) {
		return h.next.Handle(ctx)
	}
	
	// Skip if tracing is not enabled or tracer is not available
	if !h.config.Enabled || h.tracer == nil || !h.tracer.IsEnabled() {
		return h.next.Handle(ctx)
	}
	
	startTime := time.Now()
	
	// Extract trace context from headers
	headers := h.extractHeaders(ctx)
	traceContext, err := h.tracer.ExtractHeaders(headers)
	if err != nil && h.logger != nil {
		h.logger.Debug(ctx.Context(), "Failed to extract trace context",
			&TracingField{key: "error", value: err.Error()},
		)
	}
	
	// Create operation name
	operationName := fmt.Sprintf("%s %s", ctx.Method(), ctx.Path())
	
	// Start span
	span, spanCtx := h.tracer.StartSpan(ctx.Context(), operationName)
	defer span.Finish()
	
	// Set span tags
	h.setSpanTags(span, ctx)
	
	// Set trace ID in response headers
	if traceContext != nil && traceContext.TraceID() != "" {
		ctx.SetHeader(h.config.TraceIDHeader, traceContext.TraceID())
	}
	
	// Store span in context for downstream use
	ctx.Set("trace_span", span)
	ctx.Set("trace_context", traceContext)
	
	// Process request
	err = h.next.Handle(ctx)
	
	// Set span status and tags based on response
	h.setResponseTags(span, ctx, err, time.Since(startTime))
	
	// Log tracing info if logger is available
	if h.logger != nil {
		fields := []*TracingField{
			{key: "trace_id", value: traceContext.TraceID()},
			{key: "span_id", value: span.Context().SpanID()},
			{key: "operation", value: operationName},
			{key: "duration_ms", value: time.Since(startTime).Milliseconds()},
		}
		
		if err != nil {
			fields = append(fields, &TracingField{key: "error", value: err.Error()})
		}
		
		h.logger.Debug(spanCtx, "Request traced", convertTracingFields(fields)...)
	}
	
	return err
}

// shouldSkipPath checks if the path should be skipped for tracing
func (h *TracingHandler) shouldSkipPath(path string) bool {
	for _, skipPath := range h.config.SkipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}

// extractHeaders extracts HTTP headers for trace propagation
func (h *TracingHandler) extractHeaders(ctx ports.HTTPContext) map[string]string {
	headers := make(map[string]string)
	
	// Common trace headers
	traceHeaders := []string{
		"traceparent",
		"tracestate",
		"x-trace-id",
		"x-span-id",
		"x-request-id",
		"uber-trace-id",
		"x-b3-traceid",
		"x-b3-spanid",
		"x-b3-parentspanid",
		"x-b3-sampled",
		"x-b3-flags",
	}
	
	for _, header := range traceHeaders {
		if value := ctx.Header(header); value != "" {
			headers[header] = value
		}
	}
	
	return headers
}

// setSpanTags sets initial span tags
func (h *TracingHandler) setSpanTags(span ports.TraceSpan, ctx ports.HTTPContext) {
	span.SetTag("http.method", ctx.Method())
	span.SetTag("http.url", ctx.Path())
	span.SetTag("http.user_agent", ctx.UserAgent())
	span.SetTag("http.remote_addr", ctx.ClientIP())
	span.SetTag("component", "http")
	span.SetTag("span.kind", "server")
	
	// Add custom tags based on context
	if requestID := ctx.Get("request_id"); requestID != nil {
		span.SetTag("request.id", requestID)
	}
	
	if userID := ctx.Get("user_id"); userID != nil {
		span.SetTag("user.id", userID)
	}
	
	// Add query parameters if present
	if queryParams := h.extractQueryParams(ctx); len(queryParams) > 0 {
		for key, value := range queryParams {
			span.SetTag(fmt.Sprintf("http.query.%s", key), value)
		}
	}
}

// setResponseTags sets response-related span tags
func (h *TracingHandler) setResponseTags(span ports.TraceSpan, ctx ports.HTTPContext, err error, duration time.Duration) {
	span.SetTag("http.duration_ms", duration.Milliseconds())
	
	// Set status code if available from context
	if statusCode := ctx.Get("status_code"); statusCode != nil {
		if code, ok := statusCode.(int); ok {
			span.SetTag("http.status_code", code)
			
			// Mark span as error for 4xx and 5xx responses
			if code >= 400 {
				span.SetTag("error", true)
				span.SetTag("http.error", true)
			}
		}
	}
	
	// Set error information
	if err != nil {
		span.SetError(err)
		span.SetTag("error", true)
		span.SetTag("error.message", err.Error())
	}
	
	// Add response size if available
	if responseSize := ctx.Get("response_size"); responseSize != nil {
		span.SetTag("http.response.size", responseSize)
	}
}

// extractQueryParams extracts query parameters for tracing
func (h *TracingHandler) extractQueryParams(ctx ports.HTTPContext) map[string]string {
	// This is a simplified implementation
	// In a real implementation, you might want to limit which query params are traced
	params := make(map[string]string)
	
	// Add common parameters that are safe to trace
	safeParams := []string{"limit", "offset", "page", "size", "sort", "order", "format"}
	
	for _, param := range safeParams {
		if value := ctx.Query(param); value != "" {
			params[param] = value
		}
	}
	
	return params
}

// TracingField implements ports.Field for tracing logging
type TracingField struct {
	key   string
	value interface{}
}

// Key returns the field key
func (f *TracingField) Key() string {
	return f.key
}

// Value returns the field value
func (f *TracingField) Value() interface{} {
	return f.value
}

// convertTracingFields converts TracingField slice to ports.Field slice
func convertTracingFields(fields []*TracingField) []ports.Field {
	result := make([]ports.Field, len(fields))
	for i, field := range fields {
		result[i] = field
	}
	return result
}

// SpanLogger provides logging capabilities within a span context
type SpanLogger struct {
	span   ports.TraceSpan
	logger ports.Logger
}

// NewSpanLogger creates a new span logger
func NewSpanLogger(span ports.TraceSpan, logger ports.Logger) *SpanLogger {
	return &SpanLogger{
		span:   span,
		logger: logger,
	}
}

// LogInfo logs an info message with span context
func (sl *SpanLogger) LogInfo(message string, fields map[string]interface{}) {
	if sl.span != nil {
		sl.span.LogEvent("info", map[string]interface{}{
			"message": message,
			"level":   "info",
			"fields":  fields,
		})
	}
	
	if sl.logger != nil {
		logFields := make([]ports.Field, 0, len(fields))
		for key, value := range fields {
			logFields = append(logFields, &TracingField{key: key, value: value})
		}
		sl.logger.Info(nil, message, logFields...)
	}
}

// LogError logs an error message with span context
func (sl *SpanLogger) LogError(message string, err error, fields map[string]interface{}) {
	if sl.span != nil {
		eventFields := map[string]interface{}{
			"message": message,
			"level":   "error",
			"error":   err.Error(),
		}
		
		for key, value := range fields {
			eventFields[key] = value
		}
		
		sl.span.LogEvent("error", eventFields)
		sl.span.SetError(err)
	}
	
	if sl.logger != nil {
		logFields := make([]ports.Field, 0, len(fields)+1)
		logFields = append(logFields, &TracingField{key: "error", value: err.Error()})
		
		for key, value := range fields {
			logFields = append(logFields, &TracingField{key: key, value: value})
		}
		
		sl.logger.Error(nil, message, logFields...)
	}
}

// SpanFromContext extracts span from HTTP context
func SpanFromContext(ctx ports.HTTPContext) ports.TraceSpan {
	if span, ok := ctx.Get("trace_span").(ports.TraceSpan); ok {
		return span
	}
	return nil
}

// TraceContextFromContext extracts trace context from HTTP context
func TraceContextFromContext(ctx ports.HTTPContext) ports.TraceContext {
	if traceCtx, ok := ctx.Get("trace_context").(ports.TraceContext); ok {
		return traceCtx
	}
	return nil
}

// ShouldSample determines if a request should be sampled for tracing
func ShouldSample(sampleRate float64, traceID string) bool {
	if sampleRate <= 0 {
		return false
	}
	if sampleRate >= 1.0 {
		return true
	}
	
	// Use deterministic sampling based on trace ID
	if traceID == "" {
		return false
	}
	
	// Simple hash-based sampling
	hash := simpleHash(traceID)
	threshold := uint32(sampleRate * float64(^uint32(0)))
	
	return hash < threshold
}

// simpleHash provides a simple hash function for sampling
func simpleHash(s string) uint32 {
	hash := uint32(2166136261) // FNV offset basis
	for _, c := range s {
		hash ^= uint32(c)
		hash *= 16777619 // FNV prime
	}
	return hash
}