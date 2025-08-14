package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics 구조체 - 모든 메트릭을 포함
type Metrics struct {
	// HTTP 요청 관련 메트릭
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPRequestSize     *prometheus.HistogramVec
	HTTPResponseSize    *prometheus.HistogramVec
	HTTPActiveRequests  prometheus.Gauge

	// 캐시 관련 메트릭
	CacheHitsTotal      *prometheus.CounterVec
	CacheMissesTotal    *prometheus.CounterVec
	CacheEvictionsTotal *prometheus.CounterVec
	CacheSizeBytes      *prometheus.GaugeVec
	CacheItemsCount     *prometheus.GaugeVec
	CacheBandwidthSaved *prometheus.CounterVec

	// 프록시 관련 메트릭
	ProxyRequestsTotal    *prometheus.CounterVec
	ProxyErrorsTotal      *prometheus.CounterVec
	ProxyUpstreamDuration *prometheus.HistogramVec
	ProxyBytesTransferred *prometheus.CounterVec

	// 인증 관련 메트릭
	AuthAttemptsTotal *prometheus.CounterVec
	AuthFailuresTotal *prometheus.CounterVec
	ActiveSessions    prometheus.Gauge

	// 패키지 검증 관련 메트릭
	PackageVerifications *prometheus.CounterVec
	VerificationFailures *prometheus.CounterVec
	VerificationDuration *prometheus.HistogramVec

	// TTL 관련 메트릭
	TTLCalculationsTotal *prometheus.CounterVec
	TTLSourceTotal       *prometheus.CounterVec
	CacheTTLGauge        *prometheus.GaugeVec
	TTLExpirationTotal   *prometheus.CounterVec
	TTLStatistics        *prometheus.GaugeVec

	// 시스템 관련 메트릭
	ConfigReloads        prometheus.Counter
	ConfigReloadFailures prometheus.Counter
	UptimeSeconds        prometheus.Counter

	// 비즈니스 메트릭 강화
	PackageDownloadsTotal *prometheus.CounterVec
	PackageUploadsTotal   *prometheus.CounterVec
	PopularPackagesGauge  *prometheus.GaugeVec
	UserAgentRequests     *prometheus.CounterVec
	GeographicRequests    *prometheus.CounterVec

	// 성능 메트릭 강화
	ConcurrentConnections    *prometheus.GaugeVec
	ThroughputRequestsPerSec *prometheus.GaugeVec
	LatencyPercentiles       *prometheus.GaugeVec
	ResourceUtilization      *prometheus.GaugeVec

	// 에러 메트릭 강화
	HTTPStatusDistribution *prometheus.CounterVec
	RetryAttempts          *prometheus.CounterVec
	RetrySuccessRate       *prometheus.GaugeVec
	TimeoutErrors          *prometheus.CounterVec
	ConnectionErrors       *prometheus.CounterVec
	UpstreamFailureRate    *prometheus.GaugeVec

	// 사용자 활동 메트릭
	UniqueUsers           *prometheus.GaugeVec
	RequestPatterns       *prometheus.CounterVec
	SessionDuration       *prometheus.HistogramVec
	AuthenticationMetrics *prometheus.CounterVec
	UserBehaviorMetrics   *prometheus.CounterVec
}

// NewMetrics 새 메트릭 인스턴스 생성
func NewMetrics() *Metrics {
	return &Metrics{
		// HTTP 요청 메트릭
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status", "registry_type"},
		),

		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "proxynd_http_request_duration_seconds",
				Help:    "HTTP request latencies in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "status", "registry_type"},
		),

		HTTPRequestSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "proxynd_http_request_size_bytes",
				Help:    "HTTP request sizes in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "path", "registry_type"},
		),

		HTTPResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "proxynd_http_response_size_bytes",
				Help:    "HTTP response sizes in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "path", "status", "registry_type"},
		),

		HTTPActiveRequests: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "proxynd_http_active_requests",
				Help: "Number of active HTTP requests",
			},
		),

		// 캐시 메트릭
		CacheHitsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_cache_hits_total",
				Help: "Total number of cache hits",
			},
			[]string{"registry_type", "cache_backend"},
		),

		CacheMissesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_cache_misses_total",
				Help: "Total number of cache misses",
			},
			[]string{"registry_type", "cache_backend"},
		),

		CacheEvictionsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_cache_evictions_total",
				Help: "Total number of cache evictions",
			},
			[]string{"registry_type", "cache_backend", "reason"},
		),

		CacheSizeBytes: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "proxynd_cache_size_bytes",
				Help: "Current cache size in bytes",
			},
			[]string{"registry_type", "cache_backend"},
		),

		CacheItemsCount: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "proxynd_cache_items_count",
				Help: "Number of items in cache",
			},
			[]string{"registry_type", "cache_backend"},
		),

		CacheBandwidthSaved: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_cache_bandwidth_saved_bytes",
				Help: "Total bandwidth saved by cache hits",
			},
			[]string{"registry_type"},
		),

		// 프록시 메트릭
		ProxyRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_proxy_requests_total",
				Help: "Total number of proxy requests",
			},
			[]string{"registry_type", "upstream", "method"},
		),

		ProxyErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_proxy_errors_total",
				Help: "Total number of proxy errors",
			},
			[]string{"registry_type", "upstream", "error_type"},
		),

		ProxyUpstreamDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "proxynd_proxy_upstream_duration_seconds",
				Help:    "Duration of upstream requests in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"registry_type", "upstream"},
		),

		ProxyBytesTransferred: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_proxy_bytes_transferred_total",
				Help: "Total bytes transferred through proxy",
			},
			[]string{"registry_type", "direction"}, // direction: upload, download
		),

		// 인증 메트릭
		AuthAttemptsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_auth_attempts_total",
				Help: "Total number of authentication attempts",
			},
			[]string{"method", "result"},
		),

		AuthFailuresTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_auth_failures_total",
				Help: "Total number of authentication failures",
			},
			[]string{"method", "reason"},
		),

		ActiveSessions: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "proxynd_active_sessions",
				Help: "Number of active authenticated sessions",
			},
		),

		// 패키지 검증 메트릭
		PackageVerifications: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_package_verifications_total",
				Help: "Total number of package verifications",
			},
			[]string{"registry_type", "result"},
		),

		VerificationFailures: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_verification_failures_total",
				Help: "Total number of verification failures",
			},
			[]string{"registry_type", "failure_type"},
		),

		VerificationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "proxynd_verification_duration_seconds",
				Help:    "Duration of package verification in seconds",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
			},
			[]string{"registry_type"},
		),

		// TTL 메트릭
		TTLCalculationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_ttl_calculations_total",
				Help: "Total number of TTL calculations",
			},
			[]string{"registry_type", "package_name", "source"},
		),

		TTLSourceTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_ttl_source_total",
				Help: "Total TTL calculations by source type",
			},
			[]string{"source_type"}, // pattern, header, package_type, global
		),

		CacheTTLGauge: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "proxynd_cache_ttl_seconds",
				Help: "Current TTL values for cached items",
			},
			[]string{"registry_type", "package_name", "ttl_source"},
		),

		TTLExpirationTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_ttl_expirations_total",
				Help: "Total number of TTL expirations",
			},
			[]string{"registry_type", "expiration_type"}, // natural, early, stale
		),

		TTLStatistics: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "proxynd_ttl_statistics",
				Help: "TTL statistics including averages and percentiles",
			},
			[]string{"registry_type", "metric_type"}, // avg, min, max, p50, p90, p95, p99
		),

		// 시스템 메트릭
		ConfigReloads: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "proxynd_config_reloads_total",
				Help: "Total number of configuration reloads",
			},
		),

		ConfigReloadFailures: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "proxynd_config_reload_failures_total",
				Help: "Total number of configuration reload failures",
			},
		),

		UptimeSeconds: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "proxynd_uptime_seconds",
				Help: "Uptime in seconds",
			},
		),

		// 비즈니스 메트릭 강화
		PackageDownloadsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_package_downloads_total",
				Help: "Total package downloads by name and version",
			},
			[]string{"registry_type", "package_name", "package_version", "file_type"},
		),

		PackageUploadsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_package_uploads_total",
				Help: "Total package uploads by name and version",
			},
			[]string{"registry_type", "package_name", "package_version"},
		),

		PopularPackagesGauge: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "proxynd_popular_packages_rank",
				Help: "Popular packages ranking by download count",
			},
			[]string{"registry_type", "package_name", "rank_position"},
		),

		UserAgentRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_user_agent_requests_total",
				Help: "Total requests by user agent category",
			},
			[]string{"user_agent_category", "user_agent_version", "registry_type"},
		),

		GeographicRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_geographic_requests_total",
				Help: "Total requests by geographic location",
			},
			[]string{"country_code", "region", "registry_type"},
		),

		// 성능 메트릭 강화
		ConcurrentConnections: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "proxynd_concurrent_connections",
				Help: "Current number of concurrent connections",
			},
			[]string{"registry_type", "connection_type"},
		),

		ThroughputRequestsPerSec: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "proxynd_throughput_requests_per_second",
				Help: "Current throughput in requests per second",
			},
			[]string{"registry_type", "time_window"},
		),

		LatencyPercentiles: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "proxynd_latency_percentiles_seconds",
				Help: "Response time percentiles (P50, P90, P95, P99)",
			},
			[]string{"registry_type", "percentile", "method"},
		),

		ResourceUtilization: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "proxynd_resource_utilization_percent",
				Help: "Resource utilization percentages",
			},
			[]string{"resource_type", "measurement"},
		),

		// 에러 메트릭 강화
		HTTPStatusDistribution: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_http_status_distribution_total",
				Help: "HTTP status code distribution",
			},
			[]string{"registry_type", "status_class", "status_code"},
		),

		RetryAttempts: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_retry_attempts_total",
				Help: "Total retry attempts by type and outcome",
			},
			[]string{"registry_type", "retry_reason", "attempt_number", "outcome"},
		),

		RetrySuccessRate: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "proxynd_retry_success_rate",
				Help: "Retry success rate by registry type",
			},
			[]string{"registry_type", "retry_reason"},
		),

		TimeoutErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_timeout_errors_total",
				Help: "Total timeout errors by type",
			},
			[]string{"registry_type", "timeout_type", "upstream"},
		),

		ConnectionErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_connection_errors_total",
				Help: "Total connection errors by type",
			},
			[]string{"registry_type", "error_type", "upstream"},
		),

		UpstreamFailureRate: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "proxynd_upstream_failure_rate",
				Help: "Upstream failure rate by registry type",
			},
			[]string{"registry_type", "upstream", "failure_type"},
		),

		// 사용자 활동 메트릭
		UniqueUsers: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "proxynd_unique_users",
				Help: "Number of unique users by time window",
			},
			[]string{"registry_type", "time_window", "user_type"},
		),

		RequestPatterns: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_request_patterns_total",
				Help: "Request patterns by time and behavior",
			},
			[]string{"registry_type", "pattern_type", "time_period"},
		),

		SessionDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "proxynd_session_duration_seconds",
				Help:    "User session duration distribution",
				Buckets: prometheus.ExponentialBuckets(60, 2, 12), // 1min to ~68 hours
			},
			[]string{"registry_type", "user_type"},
		),

		AuthenticationMetrics: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_authentication_events_total",
				Help: "Authentication events by type and outcome",
			},
			[]string{"auth_method", "event_type", "outcome", "failure_reason"},
		),

		UserBehaviorMetrics: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "proxynd_user_behavior_events_total",
				Help: "User behavior events and patterns",
			},
			[]string{"registry_type", "behavior_type", "user_category"},
		),
	}
}

// 전역 메트릭 인스턴스
var globalMetrics *Metrics

// InitMetrics 메트릭 초기화
func InitMetrics() {
	if globalMetrics == nil {
		globalMetrics = NewMetrics()
		// Container 메트릭도 함께 초기화
		InitContainerMetrics()
	}
}

// GetMetrics 전역 메트릭 인스턴스 반환
func GetMetrics() *Metrics {
	if globalMetrics == nil {
		InitMetrics()
	}
	return globalMetrics
}

// ResetMetrics 테스트용 메트릭 리셋
func ResetMetrics() {
	globalMetrics = nil
}
