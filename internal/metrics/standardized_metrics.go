package metrics

import (
	"strconv"

	"proxynd/internal/adapters/pm/common"
)

// StandardizedMetrics provides helper methods for recording metrics with standardized labels
type StandardizedMetrics struct {
	*Metrics
}

// StandardizedLabels defines standardized label names for ProxyND metrics
var StandardizedLabels = struct {
	RequestID    string
	Principal    string
	Manager      string
	Path         string
	Method       string
	StatusCode   string
	CacheHit     string
	Upstream     string
	PackageName  string
	Version      string
	ErrorType    string
	RegistryType string
	CacheBackend string
	UserAgent    string
	Country      string
	Region       string
}{
	RequestID:    "request_id",
	Principal:    "principal",
	Manager:      "manager",
	Path:         "path",
	Method:       "method",
	StatusCode:   "status_code",
	CacheHit:     "cache_hit",
	Upstream:     "upstream",
	PackageName:  "package_name",
	Version:      "version",
	ErrorType:    "error_type",
	RegistryType: "registry_type",
	CacheBackend: "cache_backend",
	UserAgent:    "user_agent",
	Country:      "country",
	Region:       "region",
}

// MetricsContext contains standardized context information for metrics
type MetricsContext struct {
	RequestID    string
	Principal    string
	Manager      string
	Path         string
	Method       string
	StatusCode   int
	CacheHit     bool
	Upstream     string
	PackageName  string
	Version      string
	RegistryType string
	UserAgent    string
	Country      string
	Region       string
}

// NewStandardizedMetrics creates a new StandardizedMetrics instance
func NewStandardizedMetrics() *StandardizedMetrics {
	return &StandardizedMetrics{
		Metrics: GetMetrics(),
	}
}

// GetStandardizedMetrics returns the global standardized metrics instance
func GetStandardizedMetrics() *StandardizedMetrics {
	return &StandardizedMetrics{
		Metrics: GetMetrics(),
	}
}

// RecordHTTPRequest records an HTTP request with standardized labels
func (s *StandardizedMetrics) RecordHTTPRequest(
	ctx MetricsContext, latencySeconds float64, requestSizeBytes, responseSizeBytes int64,
) {
	statusStr := strconv.Itoa(ctx.StatusCode)

	// Record basic HTTP metrics
	s.HTTPRequestsTotal.WithLabelValues(
		ctx.Method,
		ctx.Path,
		statusStr,
		ctx.RegistryType,
	).Inc()

	s.HTTPRequestDuration.WithLabelValues(
		ctx.Method,
		ctx.Path,
		statusStr,
		ctx.RegistryType,
	).Observe(latencySeconds)

	if requestSizeBytes > 0 {
		s.HTTPRequestSize.WithLabelValues(
			ctx.Method,
			ctx.Path,
			ctx.RegistryType,
		).Observe(float64(requestSizeBytes))
	}

	if responseSizeBytes > 0 {
		s.HTTPResponseSize.WithLabelValues(
			ctx.Method,
			ctx.Path,
			statusStr,
			ctx.RegistryType,
		).Observe(float64(responseSizeBytes))
	}

	// Record status distribution
	statusClass := getStandardizedStatusClass(ctx.StatusCode)
	s.HTTPStatusDistribution.WithLabelValues(
		ctx.RegistryType,
		statusClass,
		statusStr,
	).Inc()

	// Record user agent metrics
	if ctx.UserAgent != "" {
		userAgentCategory := categorizeUserAgent(ctx.UserAgent)
		s.UserAgentRequests.WithLabelValues(
			userAgentCategory,
			"", // version can be extracted if needed
			ctx.RegistryType,
		).Inc()
	}

	// Record geographic metrics if available
	if ctx.Country != "" {
		s.GeographicRequests.WithLabelValues(
			ctx.Country,
			ctx.Region,
			ctx.RegistryType,
		).Inc()
	}
}

// RecordCacheOperation records cache operation metrics with standardized labels
func (s *StandardizedMetrics) RecordCacheOperation(
	ctx MetricsContext, operation string, hit bool, backend string, bandwidthSavedBytes int64,
) {
	if hit {
		s.CacheHitsTotal.WithLabelValues(
			ctx.RegistryType,
			backend,
		).Inc()

		if bandwidthSavedBytes > 0 {
			s.CacheBandwidthSaved.WithLabelValues(
				ctx.RegistryType,
			).Add(float64(bandwidthSavedBytes))
		}
	} else {
		s.CacheMissesTotal.WithLabelValues(
			ctx.RegistryType,
			backend,
		).Inc()
	}
}

// RecordProxyOperation records proxy operation metrics with standardized labels
func (s *StandardizedMetrics) RecordProxyOperation(
	ctx MetricsContext, upstreamLatencySeconds float64,
	bytesTransferred int64, direction string, err error,
) {
	// Record proxy requests
	s.ProxyRequestsTotal.WithLabelValues(
		ctx.RegistryType,
		ctx.Upstream,
		ctx.Method,
	).Inc()

	// Record upstream latency
	if upstreamLatencySeconds > 0 {
		s.ProxyUpstreamDuration.WithLabelValues(
			ctx.RegistryType,
			ctx.Upstream,
		).Observe(upstreamLatencySeconds)
	}

	// Record bytes transferred
	if bytesTransferred > 0 {
		s.ProxyBytesTransferred.WithLabelValues(
			ctx.RegistryType,
			direction, // "upload" or "download"
		).Add(float64(bytesTransferred))
	}

	// Record errors if any
	if err != nil {
		errorType := categorizeError(err)
		s.ProxyErrorsTotal.WithLabelValues(
			ctx.RegistryType,
			ctx.Upstream,
			errorType,
		).Inc()
	}
}

// RecordPackageOperation records package download/upload metrics
func (s *StandardizedMetrics) RecordPackageOperation(ctx MetricsContext, operation string) {
	switch operation {
	case "download":
		s.PackageDownloadsTotal.WithLabelValues(
			ctx.RegistryType,
			ctx.PackageName,
			ctx.Version,
			getFileType(ctx.Path),
		).Inc()
	case "upload":
		s.PackageUploadsTotal.WithLabelValues(
			ctx.RegistryType,
			ctx.PackageName,
			ctx.Version,
		).Inc()
	}
}

// RecordAuthentication records authentication metrics
func (s *StandardizedMetrics) RecordAuthentication(authMethod, eventType, outcome, failureReason string) {
	s.AuthenticationMetrics.WithLabelValues(
		authMethod,
		eventType,
		outcome,
		failureReason,
	).Inc()

	if outcome == "failure" {
		s.AuthFailuresTotal.WithLabelValues(
			authMethod,
			failureReason,
		).Inc()
	}

	s.AuthAttemptsTotal.WithLabelValues(
		authMethod,
		outcome,
	).Inc()
}

// RecordTTLOperation records TTL-related metrics
func (s *StandardizedMetrics) RecordTTLOperation(ctx MetricsContext, source string, ttlSeconds int64) {
	s.TTLCalculationsTotal.WithLabelValues(
		ctx.RegistryType,
		ctx.PackageName,
		source,
	).Inc()

	s.TTLSourceTotal.WithLabelValues(
		source,
	).Inc()

	if ttlSeconds > 0 {
		s.CacheTTLGauge.WithLabelValues(
			ctx.RegistryType,
			ctx.PackageName,
			source,
		).Set(float64(ttlSeconds))
	}
}

// RecordVerificationOperation records package verification metrics
func (s *StandardizedMetrics) RecordVerificationOperation(
	ctx MetricsContext, result string, latencySeconds float64, err error,
) {
	s.PackageVerifications.WithLabelValues(
		ctx.RegistryType,
		result,
	).Inc()

	if latencySeconds > 0 {
		s.VerificationDuration.WithLabelValues(
			ctx.RegistryType,
		).Observe(latencySeconds)
	}

	if err != nil {
		failureType := categorizeVerificationFailure(err)
		s.VerificationFailures.WithLabelValues(
			ctx.RegistryType,
			failureType,
		).Inc()
	}
}

// RecordRetryOperation records retry metrics
func (s *StandardizedMetrics) RecordRetryOperation(
	ctx MetricsContext, reason string, attemptNumber int, outcome string,
) {
	s.RetryAttempts.WithLabelValues(
		ctx.RegistryType,
		reason,
		strconv.Itoa(attemptNumber),
		outcome,
	).Inc()
}

// RecordUserBehavior records user behavior metrics
func (s *StandardizedMetrics) RecordUserBehavior(ctx MetricsContext, behaviorType, userCategory string) {
	s.UserBehaviorMetrics.WithLabelValues(
		ctx.RegistryType,
		behaviorType,
		userCategory,
	).Inc()
}

// RecordSessionDuration records session duration
func (s *StandardizedMetrics) RecordSessionDuration(registryType, userType string, durationSeconds float64) {
	s.SessionDuration.WithLabelValues(
		registryType,
		userType,
	).Observe(durationSeconds)
}

// UpdateGaugeMetrics updates various gauge metrics
func (s *StandardizedMetrics) UpdateGaugeMetrics(ctx MetricsContext) {
	// Update active requests
	s.HTTPActiveRequests.Inc()

	// Update cache size if available
	// This would typically be called from cache management components

	// Update resource utilization
	// This would typically be called from system monitoring components
}

// Helper functions for categorization

func getStandardizedStatusClass(statusCode int) string {
	switch {
	case statusCode >= 500:
		return "5xx"
	case statusCode >= 400:
		return "4xx"
	case statusCode >= 300:
		return "3xx"
	case statusCode >= 200:
		return "2xx"
	default:
		return "1xx"
	}
}

func categorizeUserAgent(userAgent string) string {
	// Simple categorization logic - can be enhanced
	switch {
	case contains(userAgent, "apt"):
		return "apt"
	case contains(userAgent, "maven"):
		return "maven"
	case contains(userAgent, "npm"):
		return "npm"
	case contains(userAgent, "pip"):
		return "pip"
	case contains(userAgent, "docker"):
		return common.UserAgentDocker
	case contains(userAgent, "curl"):
		return "curl"
	case contains(userAgent, "wget"):
		return "wget"
	case contains(userAgent, "browser"), contains(userAgent, "Mozilla"):
		return "browser"
	default:
		return common.UserAgentUnknown
	}
}

func categorizeError(err error) string {
	errStr := err.Error()
	switch {
	case contains(errStr, "timeout"):
		return "timeout"
	case contains(errStr, "connection"):
		return "connection"
	case contains(errStr, "not found"), contains(errStr, "404"):
		return "not_found"
	case contains(errStr, "unauthorized"), contains(errStr, "401"):
		return "unauthorized"
	case contains(errStr, "forbidden"), contains(errStr, "403"):
		return common.UserAgentForbidden
	case contains(errStr, "internal"), contains(errStr, "500"):
		return "internal"
	default:
		return statusUnknown
	}
}

func categorizeVerificationFailure(err error) string {
	errStr := err.Error()
	switch {
	case contains(errStr, "signature"):
		return "signature"
	case contains(errStr, "checksum"):
		return "checksum"
	case contains(errStr, "certificate"):
		return "certificate"
	case contains(errStr, "key"):
		return "key"
	default:
		return "unknown"
	}
}

func getFileType(path string) string {
	// Extract file type from path
	switch {
	case contains(path, ".jar"):
		return "jar"
	case contains(path, ".pom"):
		return "pom"
	case contains(path, ".tgz"), contains(path, ".tar.gz"):
		return "tarball"
	case contains(path, ".deb"):
		return "deb"
	case contains(path, ".rpm"):
		return "rpm"
	case contains(path, ".apk"):
		return common.UserAgentApk
	case contains(path, ".whl"):
		return "wheel"
	case contains(path, ".tar.gz"):
		return "source"
	default:
		return "unknown"
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
		(len(s) > len(substr) && s[:len(substr)] == substr) ||
		(len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// MetricsRecorder provides a convenient interface for recording metrics
type MetricsRecorder struct {
	metrics *StandardizedMetrics
	ctx     MetricsContext
}

// NewMetricsRecorder creates a new metrics recorder with base context
func NewMetricsRecorder(ctx MetricsContext) *MetricsRecorder {
	return &MetricsRecorder{
		metrics: GetStandardizedMetrics(),
		ctx:     ctx,
	}
}

// WithContext updates the metrics context
func (r *MetricsRecorder) WithContext(ctx MetricsContext) *MetricsRecorder {
	return &MetricsRecorder{
		metrics: r.metrics,
		ctx:     ctx,
	}
}

// RecordRequest records a complete HTTP request
func (r *MetricsRecorder) RecordRequest(latencySeconds float64, requestSize, responseSize int64) {
	r.metrics.RecordHTTPRequest(r.ctx, latencySeconds, requestSize, responseSize)
}

// RecordCacheHit records a cache hit
func (r *MetricsRecorder) RecordCacheHit(backend string, bandwidthSaved int64) {
	ctx := r.ctx
	ctx.CacheHit = true
	r.metrics.RecordCacheOperation(ctx, "get", true, backend, bandwidthSaved)
}

// RecordCacheMiss records a cache miss
func (r *MetricsRecorder) RecordCacheMiss(backend string) {
	ctx := r.ctx
	ctx.CacheHit = false
	r.metrics.RecordCacheOperation(ctx, "get", false, backend, 0)
}

// RecordProxy records a proxy operation
func (r *MetricsRecorder) RecordProxy(upstreamLatency float64, bytesTransferred int64, direction string, err error) {
	r.metrics.RecordProxyOperation(r.ctx, upstreamLatency, bytesTransferred, direction, err)
}

// RecordPackageDownload records a package download
func (r *MetricsRecorder) RecordPackageDownload() {
	r.metrics.RecordPackageOperation(r.ctx, "download")
}

// RecordPackageUpload records a package upload
func (r *MetricsRecorder) RecordPackageUpload() {
	r.metrics.RecordPackageOperation(r.ctx, "upload")
}
