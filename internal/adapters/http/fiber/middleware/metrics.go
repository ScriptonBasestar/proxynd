package middlewares

import (
	"strconv"
	"strings"
	"time"

	"proxynd/internal/ports"
)

// MetricsMiddleware implements ports.EnhancedHTTPMiddleware for metrics collection
type MetricsMiddleware struct {
	config  *ports.MetricsConfig
	metrics ports.MetricsCollector
	logger  ports.Logger
}

// NewMetricsMiddleware creates a new metrics middleware
func NewMetricsMiddleware(
	config *ports.MetricsConfig,
	metrics ports.MetricsCollector,
	logger ports.Logger,
) ports.EnhancedHTTPMiddleware {
	if config == nil {
		config = &ports.MetricsConfig{
			Enabled:   true,
			Path:      "/metrics",
			SkipPaths: []string{"/health"},
			Namespace: "proxynd",
			Subsystem: "http",
		}
	}

	return &MetricsMiddleware{
		config:  config,
		metrics: metrics,
		logger:  logger,
	}
}

// Process implements ports.EnhancedHTTPMiddleware
func (m *MetricsMiddleware) Process(next ports.HTTPHandler) ports.HTTPHandler {
	return &MetricsHandler{
		next:    next,
		config:  m.config,
		metrics: m.metrics,
		logger:  m.logger,
	}
}

// Order returns the middleware execution order
func (m *MetricsMiddleware) Order() ports.MiddlewareOrder {
	return ports.OrderMetrics
}

// Name returns the middleware name
func (m *MetricsMiddleware) Name() string {
	return "metrics"
}

// IsEnabled returns whether the middleware is enabled
func (m *MetricsMiddleware) IsEnabled() bool {
	return m.config.Enabled && m.metrics != nil
}

// MetricsHandler handles metrics collection
type MetricsHandler struct {
	next    ports.HTTPHandler
	config  *ports.MetricsConfig
	metrics ports.MetricsCollector
	logger  ports.Logger
}

// Handle processes the request with metrics collection
func (h *MetricsHandler) Handle(ctx ports.HTTPContext) error {
	// Skip metrics for configured paths
	if h.shouldSkipPath(ctx.Path()) {
		return h.next.Handle(ctx)
	}

	// Skip if metrics collection is not enabled or collector is not available
	if !h.config.Enabled || h.metrics == nil {
		return h.next.Handle(ctx)
	}

	startTime := time.Now()

	// Start timer for request duration
	timer := h.metrics.StartTimer("http_request_duration_seconds", h.getLabels(ctx, 0))
	defer timer.Stop()

	// Increment in-flight requests
	h.metrics.IncCounter("http_requests_in_flight", h.getLabels(ctx, 0))
	defer h.metrics.AddCounter("http_requests_in_flight", -1, h.getLabels(ctx, 0))

	// Process request
	err := h.next.Handle(ctx)

	// Get status code from context or error
	statusCode := h.getStatusCode(ctx, err)

	// Record metrics
	h.recordMetrics(ctx, statusCode, time.Since(startTime), err)

	return err
}

// shouldSkipPath checks if the path should be skipped for metrics
func (h *MetricsHandler) shouldSkipPath(path string) bool {
	for _, skipPath := range h.config.SkipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}

// recordMetrics records various HTTP metrics
func (h *MetricsHandler) recordMetrics(ctx ports.HTTPContext, statusCode int, duration time.Duration, err error) {
	labels := h.getLabels(ctx, statusCode)

	// Request count
	h.metrics.IncCounter("http_requests_total", labels)

	// Request duration
	h.metrics.ObserveHistogram("http_request_duration_seconds", duration.Seconds(), labels)

	// Request size
	if body := ctx.Body(); len(body) > 0 {
		h.metrics.ObserveHistogram("http_request_size_bytes", float64(len(body)), labels)
	}

	// Response size (if available from context)
	if responseSize := ctx.Get("response_size"); responseSize != nil {
		if size, ok := responseSize.(int64); ok {
			h.metrics.ObserveHistogram("http_response_size_bytes", float64(size), labels)
		}
	}

	// Error count
	if err != nil {
		errorLabels := h.getErrorLabels(ctx, err)
		h.metrics.IncCounter("http_errors_total", errorLabels)
	}

	// Status code specific metrics
	h.recordStatusCodeMetrics(ctx, statusCode)

	// Cache metrics (if available)
	h.recordCacheMetrics(ctx)

	// Custom business metrics
	h.recordBusinessMetrics(ctx)
}

// getLabels returns standard labels for metrics
func (h *MetricsHandler) getLabels(ctx ports.HTTPContext, statusCode int) map[string]string {
	labels := map[string]string{
		"method": strings.ToLower(ctx.Method()),
		"path":   h.normalizePath(ctx.Path()),
	}

	if statusCode > 0 {
		labels["status_code"] = strconv.Itoa(statusCode)
		labels["status_class"] = h.getStatusClass(statusCode)
	}

	// Add handler name if available
	if handler := ctx.Get("handler_name"); handler != nil {
		if name, ok := handler.(string); ok {
			labels["handler"] = name
		}
	}

	return labels
}

// getErrorLabels returns error-specific labels
func (h *MetricsHandler) getErrorLabels(ctx ports.HTTPContext, err error) map[string]string {
	labels := map[string]string{
		"method":     strings.ToLower(ctx.Method()),
		"path":       h.normalizePath(ctx.Path()),
		"error_type": h.getErrorType(err),
	}

	return labels
}

// getStatusCode extracts status code from context or determines from error
func (h *MetricsHandler) getStatusCode(ctx ports.HTTPContext, err error) int {
	// Try to get from context first
	if statusCode := ctx.Get("status_code"); statusCode != nil {
		if code, ok := statusCode.(int); ok {
			return code
		}
	}

	// Determine from error
	if err != nil {
		return 500 // Internal Server Error
	}

	return 200 // OK
}

// getStatusClass returns the status code class (2xx, 3xx, 4xx, 5xx)
func (h *MetricsHandler) getStatusClass(statusCode int) string {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return "2xx"
	case statusCode >= 300 && statusCode < 400:
		return "3xx"
	case statusCode >= 400 && statusCode < 500:
		return "4xx"
	case statusCode >= 500:
		return "5xx"
	default:
		return "1xx"
	}
}

// normalizePath normalizes the path for metrics (remove IDs, etc.)
func (h *MetricsHandler) normalizePath(path string) string {
	// Simple path normalization - replace dynamic segments
	// This is a basic implementation; a more sophisticated one would use route patterns

	parts := strings.Split(path, "/")
	normalized := make([]string, len(parts))

	for i, part := range parts {
		// Replace numeric IDs
		if h.isNumeric(part) {
			normalized[i] = "{id}"
		} else if h.isUUID(part) {
			normalized[i] = "{uuid}"
		} else {
			normalized[i] = part
		}
	}

	return strings.Join(normalized, "/")
}

// isNumeric checks if a string is numeric
func (h *MetricsHandler) isNumeric(s string) bool {
	if s == "" {
		return false
	}

	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}

// isUUID checks if a string looks like a UUID
func (h *MetricsHandler) isUUID(s string) bool {
	// Simple UUID pattern check: 8-4-4-4-12 hex digits
	if len(s) != 36 {
		return false
	}

	parts := strings.Split(s, "-")
	if len(parts) != 5 {
		return false
	}

	return len(parts[0]) == 8 && len(parts[1]) == 4 && len(parts[2]) == 4 &&
		len(parts[3]) == 4 && len(parts[4]) == 12
}

// getErrorType determines the error type for metrics
func (h *MetricsHandler) getErrorType(err error) string {
	// Simple error classification
	errStr := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errStr, "timeout"):
		return "timeout"
	case strings.Contains(errStr, "connection"):
		return "connection"
	case strings.Contains(errStr, "auth"):
		return "auth"
	case strings.Contains(errStr, "permission"):
		return "permission"
	case strings.Contains(errStr, "not found"):
		return "not_found"
	case strings.Contains(errStr, "validation"):
		return "validation"
	default:
		return "unknown"
	}
}

// recordStatusCodeMetrics records status code specific metrics
func (h *MetricsHandler) recordStatusCodeMetrics(ctx ports.HTTPContext, statusCode int) {
	labels := map[string]string{
		"method": strings.ToLower(ctx.Method()),
		"path":   h.normalizePath(ctx.Path()),
	}

	// Count by status class
	statusClass := h.getStatusClass(statusCode)
	statusLabels := make(map[string]string)
	for k, v := range labels {
		statusLabels[k] = v
	}
	statusLabels["status_class"] = statusClass

	h.metrics.IncCounter("http_responses_by_status_class_total", statusLabels)

	// Special metrics for error responses
	if statusCode >= 400 {
		errorLabels := make(map[string]string)
		for k, v := range labels {
			errorLabels[k] = v
		}
		errorLabels["status_code"] = strconv.Itoa(statusCode)

		h.metrics.IncCounter("http_error_responses_total", errorLabels)
	}
}

// recordCacheMetrics records cache-related metrics
func (h *MetricsHandler) recordCacheMetrics(ctx ports.HTTPContext) {
	cacheStatus := ctx.Get("cache_status")
	if cacheStatus == nil {
		return
	}

	status, ok := cacheStatus.(string)
	if !ok {
		return
	}

	labels := map[string]string{
		"method":       strings.ToLower(ctx.Method()),
		"path":         h.normalizePath(ctx.Path()),
		"cache_status": status,
	}

	h.metrics.IncCounter("http_cache_requests_total", labels)

	// Cache hit/miss counters
	switch status {
	case "hit":
		h.metrics.IncCounter("http_cache_hits_total", labels)
	case "miss":
		h.metrics.IncCounter("http_cache_misses_total", labels)
	}
}

// recordBusinessMetrics records application-specific business metrics
func (h *MetricsHandler) recordBusinessMetrics(ctx ports.HTTPContext) {
	// Add business-specific metrics based on context

	// Package type metrics (for proxy requests)
	if packageType := ctx.Get("package_type"); packageType != nil {
		if pkgType, ok := packageType.(string); ok {
			labels := map[string]string{
				"package_type": pkgType,
				"method":       strings.ToLower(ctx.Method()),
			}
			h.metrics.IncCounter("proxy_package_requests_total", labels)
		}
	}

	// User type metrics (authenticated vs anonymous)
	userType := "anonymous"
	if userID := ctx.Get("user_id"); userID != nil {
		userType = "authenticated"
	}

	labels := map[string]string{
		"user_type": userType,
		"method":    strings.ToLower(ctx.Method()),
		"path":      h.normalizePath(ctx.Path()),
	}

	h.metrics.IncCounter("http_requests_by_user_type_total", labels)
}

// MetricsField implements ports.Field for metrics logging
type MetricsField struct {
	key   string
	value interface{}
}

// Key returns the field key
func (f *MetricsField) Key() string {
	return f.key
}

// Value returns the field value
func (f *MetricsField) Value() interface{} {
	return f.value
}

// HTTPMetrics represents HTTP request metrics
type HTTPMetrics struct {
	RequestsTotal          uint64             `json:"requests_total"`
	RequestDurationSeconds map[string]float64 `json:"request_duration_seconds"`
	RequestSizeBytes       map[string]float64 `json:"request_size_bytes"`
	ResponseSizeBytes      map[string]float64 `json:"response_size_bytes"`
	ErrorsTotal            uint64             `json:"errors_total"`
	ResponsesByStatusClass map[string]uint64  `json:"responses_by_status_class"`
	CacheMetrics           *CacheMetrics      `json:"cache_metrics,omitempty"`
}

// CacheMetrics represents cache-related metrics
type CacheMetrics struct {
	HitsTotal   uint64  `json:"hits_total"`
	MissesTotal uint64  `json:"misses_total"`
	HitRate     float64 `json:"hit_rate"`
}

// GetHTTPMetrics returns current HTTP metrics
func GetHTTPMetrics(collector ports.MetricsCollector) (*HTTPMetrics, error) {
	if collector == nil {
		return &HTTPMetrics{}, nil
	}

	snapshot, err := collector.GetMetrics()
	if err != nil {
		return nil, err
	}

	metrics := &HTTPMetrics{
		RequestDurationSeconds: make(map[string]float64),
		RequestSizeBytes:       make(map[string]float64),
		ResponseSizeBytes:      make(map[string]float64),
		ResponsesByStatusClass: make(map[string]uint64),
	}

	// Extract relevant metrics from snapshot
	for name, value := range snapshot.Counters {
		switch {
		case strings.Contains(name, "http_requests_total"):
			metrics.RequestsTotal += uint64(value)
		case strings.Contains(name, "http_errors_total"):
			metrics.ErrorsTotal += uint64(value)
		}
	}

	return metrics, nil
}
