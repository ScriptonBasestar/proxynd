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

	// 시스템 관련 메트릭
	ConfigReloads        prometheus.Counter
	ConfigReloadFailures prometheus.Counter
	UptimeSeconds        prometheus.Counter
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
	}
}

// 전역 메트릭 인스턴스
var globalMetrics *Metrics

// InitMetrics 메트릭 초기화
func InitMetrics() {
	globalMetrics = NewMetrics()
}

// GetMetrics 전역 메트릭 인스턴스 반환
func GetMetrics() *Metrics {
	if globalMetrics == nil {
		InitMetrics()
	}
	return globalMetrics
}

// ResetMetrics 메트릭 리셋 (테스트용)
func ResetMetrics() {
	globalMetrics = NewMetrics()
}
