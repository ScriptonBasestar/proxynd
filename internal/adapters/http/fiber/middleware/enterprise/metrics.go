package enterprise

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// EnterpriseMetrics holds all Enterprise API Prometheus metrics
type EnterpriseMetrics struct {
	// Request metrics
	requestsTotal     *prometheus.CounterVec
	requestDuration   *prometheus.HistogramVec
	activeRequests    *prometheus.GaugeVec
	requestSizeBytes  *prometheus.HistogramVec
	responseSizeBytes *prometheus.HistogramVec

	// Error metrics
	errorsTotal *prometheus.CounterVec

	// Rate limiting metrics
	rateLimitExceeded *prometheus.CounterVec
	rateLimitRemaining *prometheus.GaugeVec

	// License metrics
	licenseCheckTotal    *prometheus.CounterVec
	licenseCheckDuration *prometheus.HistogramVec

	// Feature-specific metrics
	rbacOperationsTotal        *prometheus.CounterVec
	auditEventsTotal           *prometheus.CounterVec
	securityScansTotal         *prometheus.CounterVec
	securityVulnerabilitiesFound *prometheus.GaugeVec
	alertsTriggeredTotal       *prometheus.CounterVec

	// Cache metrics for Enterprise API
	cacheHitsTotal   *prometheus.CounterVec
	cacheMissesTotal *prometheus.CounterVec
}

// NewEnterpriseMetrics creates and registers all Enterprise API metrics
func NewEnterpriseMetrics() *EnterpriseMetrics {
	namespace := "proxynd"
	subsystem := "enterprise_api"

	metrics := &EnterpriseMetrics{
		// Request metrics
		requestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "requests_total",
				Help:      "Total number of Enterprise API requests",
			},
			[]string{"category", "endpoint", "method", "status_code"},
		),

		requestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "request_duration_seconds",
				Help:      "Enterprise API request duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"category", "endpoint", "method"},
		),

		activeRequests: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "active_requests",
				Help:      "Number of active Enterprise API requests",
			},
			[]string{"category", "endpoint"},
		),

		requestSizeBytes: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "request_size_bytes",
				Help:      "Enterprise API request size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"category", "endpoint"},
		),

		responseSizeBytes: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "response_size_bytes",
				Help:      "Enterprise API response size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"category", "endpoint"},
		),

		// Error metrics
		errorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "errors_total",
				Help:      "Total number of Enterprise API errors",
			},
			[]string{"category", "endpoint", "error_type", "status_code"},
		),

		// Rate limiting metrics
		rateLimitExceeded: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "rate_limit_exceeded_total",
				Help:      "Total number of rate limit exceeded events",
			},
			[]string{"category", "endpoint", "user_id"},
		),

		rateLimitRemaining: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "rate_limit_remaining",
				Help:      "Remaining rate limit quota",
			},
			[]string{"category", "endpoint", "user_id"},
		),

		// License metrics
		licenseCheckTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "license_check_total",
				Help:      "Total number of license validation checks",
			},
			[]string{"result", "license_type"},
		),

		licenseCheckDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "license_check_duration_seconds",
				Help:      "License validation check duration in seconds",
				Buckets:   []float64{.0001, .0005, .001, .005, .01, .05, .1},
			},
			[]string{"result"},
		),

		// Feature-specific metrics
		rbacOperationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "rbac_operations_total",
				Help:      "Total number of RBAC operations",
			},
			[]string{"operation", "resource_type", "result"},
		),

		auditEventsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "audit_events_total",
				Help:      "Total number of audit events logged",
			},
			[]string{"event_type", "user_id", "result"},
		),

		securityScansTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "security_scans_total",
				Help:      "Total number of security scans performed",
			},
			[]string{"scan_type", "status"},
		),

		securityVulnerabilitiesFound: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "security_vulnerabilities_found",
				Help:      "Number of vulnerabilities found by severity",
			},
			[]string{"severity", "package_manager"},
		),

		alertsTriggeredTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "alerts_triggered_total",
				Help:      "Total number of alerts triggered",
			},
			[]string{"alert_type", "severity"},
		),

		// Cache metrics
		cacheHitsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "cache_hits_total",
				Help:      "Total number of Enterprise API cache hits",
			},
			[]string{"category", "endpoint"},
		),

		cacheMissesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "cache_misses_total",
				Help:      "Total number of Enterprise API cache misses",
			},
			[]string{"category", "endpoint"},
		),
	}

	return metrics
}

// MetricsMiddleware returns a Fiber middleware for collecting Enterprise API metrics
func MetricsMiddleware(metrics *EnterpriseMetrics) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Extract metadata
		category := getCategory(c.Path())
		endpoint := normalizeEndpoint(c.Path())
		method := c.Method()

		// Track active requests
		metrics.activeRequests.WithLabelValues(category, endpoint).Inc()
		defer metrics.activeRequests.WithLabelValues(category, endpoint).Dec()

		// Track request size
		if c.Request().Header.ContentLength() > 0 {
			metrics.requestSizeBytes.WithLabelValues(category, endpoint).Observe(float64(c.Request().Header.ContentLength()))
		}

		// Process request
		err := c.Next()

		// Calculate duration
		duration := time.Since(start).Seconds()

		// Get status code
		statusCode := c.Response().StatusCode()
		statusCodeStr := strconv.Itoa(statusCode)

		// Record request metrics
		metrics.requestsTotal.WithLabelValues(category, endpoint, method, statusCodeStr).Inc()
		metrics.requestDuration.WithLabelValues(category, endpoint, method).Observe(duration)

		// Record response size
		responseSize := len(c.Response().Body())
		if responseSize > 0 {
			metrics.responseSizeBytes.WithLabelValues(category, endpoint).Observe(float64(responseSize))
		}

		// Record errors
		if statusCode >= 400 {
			errorType := getErrorType(statusCode)
			metrics.errorsTotal.WithLabelValues(category, endpoint, errorType, statusCodeStr).Inc()
		}

		// Record rate limit metrics if present
		if rateLimitRemaining := c.Get("X-RateLimit-Remaining"); rateLimitRemaining != "" {
			if remaining, err := strconv.ParseFloat(rateLimitRemaining, 64); err == nil {
				userID := getUserID(c)
				metrics.rateLimitRemaining.WithLabelValues(category, endpoint, userID).Set(remaining)
			}
		}

		// Record rate limit exceeded
		if statusCode == 429 {
			userID := getUserID(c)
			metrics.rateLimitExceeded.WithLabelValues(category, endpoint, userID).Inc()
		}

		// Record cache metrics if present
		if cacheStatus := c.Locals("cache_status"); cacheStatus != nil {
			if status, ok := cacheStatus.(string); ok {
				switch status {
				case "hit":
					metrics.cacheHitsTotal.WithLabelValues(category, endpoint).Inc()
				case "miss":
					metrics.cacheMissesTotal.WithLabelValues(category, endpoint).Inc()
				}
			}
		}

		return err
	}
}

// getCategory extracts the API category from the path
func getCategory(path string) string {
	// /api/v1/enterprise/{category}/...
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 4 && parts[0] == "api" && parts[1] == "v1" && parts[2] == "enterprise" {
		return parts[3]
	}
	return "unknown"
}

// normalizeEndpoint normalizes the endpoint path by replacing dynamic segments
func normalizeEndpoint(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	normalized := make([]string, len(parts))

	for i, part := range parts {
		// Replace UUIDs, IDs, and other dynamic segments
		if isUUID(part) {
			normalized[i] = "{uuid}"
		} else if isNumeric(part) {
			normalized[i] = "{id}"
		} else {
			normalized[i] = part
		}
	}

	return "/" + strings.Join(normalized, "/")
}

// isUUID checks if a string looks like a UUID
func isUUID(s string) bool {
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

// isNumeric checks if a string is numeric
func isNumeric(s string) bool {
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

// getErrorType determines the error type based on status code
func getErrorType(statusCode int) string {
	switch statusCode {
	case 400:
		return "bad_request"
	case 401:
		return "unauthorized"
	case 402:
		return "payment_required"
	case 403:
		return "forbidden"
	case 404:
		return "not_found"
	case 405:
		return "method_not_allowed"
	case 409:
		return "conflict"
	case 422:
		return "validation_error"
	case 429:
		return "rate_limit_exceeded"
	case 500:
		return "internal_server_error"
	case 502:
		return "bad_gateway"
	case 503:
		return "service_unavailable"
	case 504:
		return "gateway_timeout"
	default:
		if statusCode >= 400 && statusCode < 500 {
			return "client_error"
		}
		return "server_error"
	}
}

// getUserID extracts user ID from context
func getUserID(c *fiber.Ctx) string {
	// Try to get from locals first
	if userID := c.Locals("user_id"); userID != nil {
		if id, ok := userID.(string); ok {
			return id
		}
	}

	// Try to get from header
	if userID := c.Get("X-User-ID"); userID != "" {
		return userID
	}

	return "anonymous"
}

// RecordRBACOperation records an RBAC operation metric
func (m *EnterpriseMetrics) RecordRBACOperation(operation, resourceType, result string) {
	m.rbacOperationsTotal.WithLabelValues(operation, resourceType, result).Inc()
}

// RecordAuditEvent records an audit event metric
func (m *EnterpriseMetrics) RecordAuditEvent(eventType, userID, result string) {
	m.auditEventsTotal.WithLabelValues(eventType, userID, result).Inc()
}

// RecordSecurityScan records a security scan metric
func (m *EnterpriseMetrics) RecordSecurityScan(scanType, status string) {
	m.securityScansTotal.WithLabelValues(scanType, status).Inc()
}

// SetVulnerabilitiesFound sets the number of vulnerabilities found
func (m *EnterpriseMetrics) SetVulnerabilitiesFound(severity, packageManager string, count float64) {
	m.securityVulnerabilitiesFound.WithLabelValues(severity, packageManager).Set(count)
}

// RecordAlertTriggered records an alert trigger metric
func (m *EnterpriseMetrics) RecordAlertTriggered(alertType, severity string) {
	m.alertsTriggeredTotal.WithLabelValues(alertType, severity).Inc()
}

// RecordLicenseCheck records a license validation check
func (m *EnterpriseMetrics) RecordLicenseCheck(result, licenseType string, duration time.Duration) {
	m.licenseCheckTotal.WithLabelValues(result, licenseType).Inc()
	m.licenseCheckDuration.WithLabelValues(result).Observe(duration.Seconds())
}
