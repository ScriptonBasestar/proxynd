package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"proxynd/internal/logging"
)

// AdvancedCollector provides enhanced metrics collection capabilities
type AdvancedCollector struct {
	logger logging.Logger
	mu     sync.RWMutex

	// Handler-specific metrics
	handlerMetrics map[string]*HandlerMetrics

	// Advanced system metrics
	goRoutines prometheus.Gauge
	heapAlloc  prometheus.Gauge
	heapSys    prometheus.Gauge
	gcRuns     prometheus.Counter
	gcDuration prometheus.Histogram
	allocRate  prometheus.Gauge
	freedRate  prometheus.Gauge

	// Performance metrics
	requestLatencyP50 *prometheus.GaugeVec
	requestLatencyP90 *prometheus.GaugeVec
	requestLatencyP95 *prometheus.GaugeVec
	requestLatencyP99 *prometheus.GaugeVec
	throughputRPS     *prometheus.GaugeVec
	errorRatePercent  *prometheus.GaugeVec

	// Cache performance metrics
	cacheLatency    *prometheus.HistogramVec
	cacheOperations *prometheus.CounterVec
	cacheThroughput *prometheus.GaugeVec
	cacheEfficiency *prometheus.GaugeVec

	// Connection metrics
	connectionLatency *prometheus.HistogramVec
	connectionReuse   *prometheus.CounterVec
	connectionHealth  *prometheus.GaugeVec
	poolUtilization   *prometheus.GaugeVec

	// Business metrics
	packageDownloads *prometheus.CounterVec
	packageUploads   *prometheus.CounterVec
	registryHealth   *prometheus.GaugeVec
	upstreamLatency  *prometheus.HistogramVec

	// Collection context
	ctx      context.Context
	cancel   context.CancelFunc
	running  bool
	interval time.Duration
	stopCh   chan struct{}

	// Data aggregation
	latencyBuffer map[string][]float64
	bufferMu      sync.RWMutex
	bufferSize    int
}

// HandlerMetrics contains metrics specific to each handler type
type HandlerMetrics struct {
	Type            string                   `json:"type"`
	RequestsTotal   *prometheus.CounterVec   `json:"-"`
	ResponseTime    *prometheus.HistogramVec `json:"-"`
	ErrorsTotal     *prometheus.CounterVec   `json:"-"`
	CacheMetrics    *HandlerCacheMetrics     `json:"cache_metrics"`
	UpstreamMetrics *HandlerUpstreamMetrics  `json:"upstream_metrics"`
	PackageMetrics  *HandlerPackageMetrics   `json:"package_metrics"`

	// Runtime statistics
	LastRequest    time.Time `json:"last_request"`
	TotalRequests  int64     `json:"total_requests"`
	AverageLatency float64   `json:"average_latency"`
	ErrorRate      float64   `json:"error_rate"`
	P99Latency     float64   `json:"p99_latency"`
}

// HandlerCacheMetrics contains cache-specific metrics for each handler
type HandlerCacheMetrics struct {
	HitRate          float64   `json:"hit_rate"`
	MissRate         float64   `json:"miss_rate"`
	AverageLatency   float64   `json:"average_latency"`
	TotalItems       int64     `json:"total_items"`
	SizeBytes        int64     `json:"size_bytes"`
	EvictionsPerHour float64   `json:"evictions_per_hour"`
	LastEviction     time.Time `json:"last_eviction"`
}

// HandlerUpstreamMetrics contains upstream-specific metrics for each handler
type HandlerUpstreamMetrics struct {
	AverageLatency   float64   `json:"average_latency"`
	P99Latency       float64   `json:"p99_latency"`
	SuccessRate      float64   `json:"success_rate"`
	TimeoutRate      float64   `json:"timeout_rate"`
	ConnectionErrors int64     `json:"connection_errors"`
	LastFailure      time.Time `json:"last_failure"`
	FailureRate      float64   `json:"failure_rate"`
}

// HandlerPackageMetrics contains package-specific metrics for each handler
type HandlerPackageMetrics struct {
	DownloadsPerHour float64   `json:"downloads_per_hour"`
	UploadsPerHour   float64   `json:"uploads_per_hour"`
	AverageSize      float64   `json:"average_size"`
	TotalBandwidth   int64     `json:"total_bandwidth"`
	PopularPackages  []string  `json:"popular_packages"`
	LastActivity     time.Time `json:"last_activity"`
	UniqueUsers      int64     `json:"unique_users"`
}

// MetricsSnapshot contains a point-in-time snapshot of all metrics
type MetricsSnapshot struct {
	Timestamp         time.Time                     `json:"timestamp"`
	SystemMetrics     *SystemMetrics                `json:"system_metrics"`
	HandlerMetrics    map[string]*HandlerMetrics    `json:"handler_metrics"`
	CacheMetrics      *CachePerformanceMetrics      `json:"cache_metrics"`
	ConnectionMetrics *ConnectionPerformanceMetrics `json:"connection_metrics"`
	BusinessMetrics   *BusinessMetrics              `json:"business_metrics"`
}

// SystemMetrics contains system-level performance metrics
type SystemMetrics struct {
	GoRoutines      int     `json:"go_routines"`
	HeapAllocMB     float64 `json:"heap_alloc_mb"`
	HeapSysMB       float64 `json:"heap_sys_mb"`
	GCRuns          int64   `json:"gc_runs"`
	GCDurationMS    float64 `json:"gc_duration_ms"`
	AllocRateMBPS   float64 `json:"alloc_rate_mbps"`
	FreedRateMBPS   float64 `json:"freed_rate_mbps"`
	CPUUsagePercent float64 `json:"cpu_usage_percent"`
}

// CachePerformanceMetrics contains cache performance data
type CachePerformanceMetrics struct {
	GlobalHitRate    float64            `json:"global_hit_rate"`
	GlobalMissRate   float64            `json:"global_miss_rate"`
	AverageLatencyMS float64            `json:"average_latency_ms"`
	ThroughputOPS    float64            `json:"throughput_ops"`
	EfficiencyScore  float64            `json:"efficiency_score"`
	HandlerBreakdown map[string]float64 `json:"handler_breakdown"`
}

// ConnectionPerformanceMetrics contains connection performance data
type ConnectionPerformanceMetrics struct {
	AverageLatencyMS   float64            `json:"average_latency_ms"`
	ReuseRate          float64            `json:"reuse_rate"`
	HealthScore        float64            `json:"health_score"`
	PoolUtilization    map[string]float64 `json:"pool_utilization"`
	ConnectionFailures int64              `json:"connection_failures"`
	ActiveConnections  int64              `json:"active_connections"`
}

// BusinessMetrics contains business-level metrics
type BusinessMetrics struct {
	TotalDownloads   int64              `json:"total_downloads"`
	TotalUploads     int64              `json:"total_uploads"`
	BandwidthSavedGB float64            `json:"bandwidth_saved_gb"`
	RegistryHealth   map[string]float64 `json:"registry_health"`
	PopularPackages  map[string]int64   `json:"popular_packages"`
	UserActivity     int64              `json:"user_activity"`
	CostSavings      float64            `json:"cost_savings"`
}

// NewAdvancedCollector creates a new advanced metrics collector
func NewAdvancedCollector(logger logging.Logger) *AdvancedCollector {
	ctx, cancel := context.WithCancel(context.Background())

	collector := &AdvancedCollector{
		logger:         logger.WithField("component", "metrics.advanced_collector"),
		handlerMetrics: make(map[string]*HandlerMetrics),
		ctx:            ctx,
		cancel:         cancel,
		interval:       30 * time.Second,
		stopCh:         make(chan struct{}),
		latencyBuffer:  make(map[string][]float64),
		bufferSize:     1000,
	}

	collector.initializeMetrics()
	return collector
}

// initializeMetrics initializes all Prometheus metrics
func (ac *AdvancedCollector) initializeMetrics() {
	// Advanced system metrics
	ac.goRoutines = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "proxynd_goroutines_count",
		Help: "Current number of goroutines",
	})

	ac.heapAlloc = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "proxynd_heap_alloc_bytes",
		Help: "Current heap allocation in bytes",
	})

	ac.heapSys = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "proxynd_heap_sys_bytes",
		Help: "Heap system memory in bytes",
	})

	ac.gcRuns = promauto.NewCounter(prometheus.CounterOpts{
		Name: "proxynd_gc_runs_total",
		Help: "Total number of garbage collection runs",
	})

	ac.gcDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "proxynd_gc_duration_seconds",
		Help:    "Garbage collection duration in seconds",
		Buckets: prometheus.ExponentialBuckets(0.000001, 2, 20),
	})

	ac.allocRate = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "proxynd_alloc_rate_bytes_per_second",
		Help: "Memory allocation rate in bytes per second",
	})

	ac.freedRate = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "proxynd_freed_rate_bytes_per_second",
		Help: "Memory freed rate in bytes per second",
	})

	// Performance metrics
	ac.requestLatencyP50 = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "proxynd_request_latency_p50_seconds",
		Help: "50th percentile request latency",
	}, []string{"handler_type", "method"})

	ac.requestLatencyP90 = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "proxynd_request_latency_p90_seconds",
		Help: "90th percentile request latency",
	}, []string{"handler_type", "method"})

	ac.requestLatencyP95 = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "proxynd_request_latency_p95_seconds",
		Help: "95th percentile request latency",
	}, []string{"handler_type", "method"})

	ac.requestLatencyP99 = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "proxynd_request_latency_p99_seconds",
		Help: "99th percentile request latency",
	}, []string{"handler_type", "method"})

	ac.throughputRPS = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "proxynd_throughput_requests_per_second",
		Help: "Current throughput in requests per second",
	}, []string{"handler_type"})

	ac.errorRatePercent = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "proxynd_error_rate_percent",
		Help: "Current error rate as percentage",
	}, []string{"handler_type", "error_type"})

	// Cache performance metrics
	ac.cacheLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "proxynd_cache_operation_duration_seconds",
		Help:    "Cache operation latency",
		Buckets: prometheus.ExponentialBuckets(0.0001, 2, 15),
	}, []string{"handler_type", "operation"})

	ac.cacheOperations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "proxynd_cache_operations_total",
		Help: "Total cache operations",
	}, []string{"handler_type", "operation", "result"})

	ac.cacheThroughput = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "proxynd_cache_throughput_ops_per_second",
		Help: "Cache throughput in operations per second",
	}, []string{"handler_type"})

	ac.cacheEfficiency = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "proxynd_cache_efficiency_score",
		Help: "Cache efficiency score (0-1)",
	}, []string{"handler_type"})

	// Connection metrics
	ac.connectionLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "proxynd_connection_establishment_duration_seconds",
		Help:    "Connection establishment latency",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 15),
	}, []string{"handler_type", "upstream"})

	ac.connectionReuse = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "proxynd_connection_reuse_total",
		Help: "Total connection reuse events",
	}, []string{"handler_type", "upstream"})

	ac.connectionHealth = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "proxynd_connection_health_score",
		Help: "Connection health score (0-1)",
	}, []string{"handler_type", "upstream"})

	ac.poolUtilization = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "proxynd_connection_pool_utilization_percent",
		Help: "Connection pool utilization percentage",
	}, []string{"handler_type", "pool_type"})

	// Business metrics
	ac.packageDownloads = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "proxynd_package_downloads_total",
		Help: "Total package downloads",
	}, []string{"handler_type", "package_name", "version"})

	ac.packageUploads = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "proxynd_package_uploads_total",
		Help: "Total package uploads",
	}, []string{"handler_type", "package_name", "version"})

	ac.registryHealth = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "proxynd_registry_health_score",
		Help: "Registry health score (0-1)",
	}, []string{"registry_type", "upstream"})

	ac.upstreamLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "proxynd_upstream_request_duration_seconds",
		Help:    "Upstream request latency",
		Buckets: prometheus.DefBuckets,
	}, []string{"handler_type", "upstream", "method"})
}

// RegisterHandler registers a new handler for metrics collection
func (ac *AdvancedCollector) RegisterHandler(handlerType string) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if _, exists := ac.handlerMetrics[handlerType]; exists {
		return
	}

	handlerMetrics := &HandlerMetrics{
		Type: handlerType,
		RequestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: fmt.Sprintf("proxynd_%s_requests_total", handlerType),
			Help: fmt.Sprintf("Total requests for %s handler", handlerType),
		}, []string{"method", "status", "path"}),
		ResponseTime: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    fmt.Sprintf("proxynd_%s_response_time_seconds", handlerType),
			Help:    fmt.Sprintf("Response time for %s handler", handlerType),
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "path"}),
		ErrorsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: fmt.Sprintf("proxynd_%s_errors_total", handlerType),
			Help: fmt.Sprintf("Total errors for %s handler", handlerType),
		}, []string{"error_type", "upstream"}),
		CacheMetrics:    &HandlerCacheMetrics{},
		UpstreamMetrics: &HandlerUpstreamMetrics{},
		PackageMetrics:  &HandlerPackageMetrics{},
	}

	ac.handlerMetrics[handlerType] = handlerMetrics
	ac.latencyBuffer[handlerType] = make([]float64, 0, ac.bufferSize)

	ac.logger.Info("Handler registered for advanced metrics collection",
		logging.F("handler_type", handlerType))
}

// Start starts the advanced metrics collection
func (ac *AdvancedCollector) Start() error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if ac.running {
		return nil
	}

	ac.logger.Info("Starting advanced metrics collector")

	// Start collection goroutines
	go ac.collectSystemMetrics()
	go ac.collectPerformanceMetrics()
	go ac.aggregateLatencyMetrics()

	ac.running = true
	ac.logger.Info("Advanced metrics collector started")

	return nil
}

// Stop stops the advanced metrics collection
func (ac *AdvancedCollector) Stop() error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if !ac.running {
		return nil
	}

	ac.logger.Info("Stopping advanced metrics collector")

	ac.cancel()
	close(ac.stopCh)

	ac.running = false
	ac.logger.Info("Advanced metrics collector stopped")

	return nil
}

// collectSystemMetrics collects system-level metrics
func (ac *AdvancedCollector) collectSystemMetrics() {
	ticker := time.NewTicker(ac.interval)
	defer ticker.Stop()

	var lastGCRuns uint32
	var lastAlloc, lastFrees uint64

	for {
		select {
		case <-ac.ctx.Done():
			return
		case <-ac.stopCh:
			return
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			// Update basic metrics
			ac.goRoutines.Set(float64(runtime.NumGoroutine()))
			ac.heapAlloc.Set(float64(m.HeapAlloc))
			ac.heapSys.Set(float64(m.HeapSys))

			// GC metrics
			if m.NumGC > lastGCRuns {
				ac.gcRuns.Add(float64(m.NumGC - lastGCRuns))
				lastGCRuns = m.NumGC
			}

			// Calculate rates
			if lastAlloc > 0 {
				allocDiff := float64(m.TotalAlloc - lastAlloc)
				freedDiff := float64(m.Frees - lastFrees)
				intervalSeconds := ac.interval.Seconds()

				ac.allocRate.Set(allocDiff / intervalSeconds)
				ac.freedRate.Set(freedDiff / intervalSeconds)
			}

			lastAlloc = m.TotalAlloc
			lastFrees = m.Frees
		}
	}
}

// collectPerformanceMetrics collects performance metrics
func (ac *AdvancedCollector) collectPerformanceMetrics() {
	ticker := time.NewTicker(ac.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ac.ctx.Done():
			return
		case <-ac.stopCh:
			return
		case <-ticker.C:
			ac.updatePerformanceMetrics()
		}
	}
}

// updatePerformanceMetrics updates all performance metrics
func (ac *AdvancedCollector) updatePerformanceMetrics() {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	for handlerType, metrics := range ac.handlerMetrics {
		// Update cache metrics
		ac.updateCacheMetrics(handlerType, metrics)

		// Update connection metrics
		ac.updateConnectionMetrics(handlerType, metrics)

		// Update business metrics
		ac.updateBusinessMetrics(handlerType, metrics)
	}
}

// updateCacheMetrics updates cache-related metrics
func (ac *AdvancedCollector) updateCacheMetrics(handlerType string, metrics *HandlerMetrics) {
	// Update cache efficiency score
	if metrics.CacheMetrics.HitRate > 0 {
		efficiency := (metrics.CacheMetrics.HitRate * 0.7) +
			((1 - metrics.CacheMetrics.AverageLatency/1000) * 0.3)
		ac.cacheEfficiency.WithLabelValues(handlerType).Set(efficiency)
	}

	// Calculate cache throughput
	if metrics.TotalRequests > 0 {
		throughput := float64(metrics.TotalRequests) / time.Since(metrics.LastRequest).Hours()
		ac.cacheThroughput.WithLabelValues(handlerType).Set(throughput)
	}
}

// updateConnectionMetrics updates connection-related metrics
func (ac *AdvancedCollector) updateConnectionMetrics(handlerType string, metrics *HandlerMetrics) {
	// Update connection health score
	successRate := metrics.UpstreamMetrics.SuccessRate
	latencyScore := 1.0 - (metrics.UpstreamMetrics.AverageLatency / 5000) // 5s max
	if latencyScore < 0 {
		latencyScore = 0
	}

	healthScore := (successRate * 0.6) + (latencyScore * 0.4)
	ac.connectionHealth.WithLabelValues(handlerType, "default").Set(healthScore)
}

// updateBusinessMetrics updates business-related metrics
func (ac *AdvancedCollector) updateBusinessMetrics(handlerType string, metrics *HandlerMetrics) {
	// Update registry health
	if metrics.UpstreamMetrics.FailureRate < 0.1 {
		ac.registryHealth.WithLabelValues(handlerType, "default").Set(0.9 + (0.1 * (1 - metrics.UpstreamMetrics.FailureRate*10))) //nolint:lll
	} else {
		ac.registryHealth.WithLabelValues(handlerType, "default").Set(1.0 - metrics.UpstreamMetrics.FailureRate)
	}
}

// aggregateLatencyMetrics aggregates latency data and calculates percentiles
func (ac *AdvancedCollector) aggregateLatencyMetrics() {
	ticker := time.NewTicker(time.Minute) // Calculate percentiles every minute
	defer ticker.Stop()

	for {
		select {
		case <-ac.ctx.Done():
			return
		case <-ac.stopCh:
			return
		case <-ticker.C:
			ac.calculateLatencyPercentiles()
		}
	}
}

// calculateLatencyPercentiles calculates and updates latency percentiles
func (ac *AdvancedCollector) calculateLatencyPercentiles() {
	ac.bufferMu.RLock()
	defer ac.bufferMu.RUnlock()

	for handlerType, latencies := range ac.latencyBuffer {
		if len(latencies) < 10 { // Need minimum data points
			continue
		}

		// Calculate percentiles
		p50 := calculatePercentile(latencies, 0.50)
		p90 := calculatePercentile(latencies, 0.90)
		p95 := calculatePercentile(latencies, 0.95)
		p99 := calculatePercentile(latencies, 0.99)

		// Update metrics
		ac.requestLatencyP50.WithLabelValues(handlerType, "all").Set(p50)
		ac.requestLatencyP90.WithLabelValues(handlerType, "all").Set(p90)
		ac.requestLatencyP95.WithLabelValues(handlerType, "all").Set(p95)
		ac.requestLatencyP99.WithLabelValues(handlerType, "all").Set(p99)

		// Update handler metrics
		ac.mu.Lock()
		if metrics, exists := ac.handlerMetrics[handlerType]; exists {
			metrics.P99Latency = p99
			metrics.AverageLatency = calculateAverage(latencies)
		}
		ac.mu.Unlock()
	}
}

// RecordLatency records a latency measurement
func (ac *AdvancedCollector) RecordLatency(handlerType string, latency float64) {
	ac.bufferMu.Lock()
	defer ac.bufferMu.Unlock()

	if buffer, exists := ac.latencyBuffer[handlerType]; exists {
		// Add to buffer
		if len(buffer) >= ac.bufferSize {
			// Remove oldest entry
			buffer = buffer[1:]
		}
		ac.latencyBuffer[handlerType] = append(buffer, latency)
	}
}

// GetSnapshot returns a complete metrics snapshot
func (ac *AdvancedCollector) GetSnapshot() *MetricsSnapshot {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	snapshot := &MetricsSnapshot{
		Timestamp:      time.Now(),
		HandlerMetrics: make(map[string]*HandlerMetrics),
	}

	// Copy handler metrics
	for handlerType, metrics := range ac.handlerMetrics {
		snapshot.HandlerMetrics[handlerType] = &HandlerMetrics{
			Type:            metrics.Type,
			LastRequest:     metrics.LastRequest,
			TotalRequests:   metrics.TotalRequests,
			AverageLatency:  metrics.AverageLatency,
			ErrorRate:       metrics.ErrorRate,
			P99Latency:      metrics.P99Latency,
			CacheMetrics:    metrics.CacheMetrics,
			UpstreamMetrics: metrics.UpstreamMetrics,
			PackageMetrics:  metrics.PackageMetrics,
		}
	}

	// Collect system metrics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	snapshot.SystemMetrics = &SystemMetrics{
		GoRoutines:  runtime.NumGoroutine(),
		HeapAllocMB: float64(m.HeapAlloc) / 1024 / 1024,
		HeapSysMB:   float64(m.HeapSys) / 1024 / 1024,
		GCRuns:      int64(m.NumGC),
	}

	// Additional metrics would be populated here
	snapshot.CacheMetrics = &CachePerformanceMetrics{}
	snapshot.ConnectionMetrics = &ConnectionPerformanceMetrics{}
	snapshot.BusinessMetrics = &BusinessMetrics{}

	return snapshot
}

// GetHandlerMetrics returns metrics for a specific handler
func (ac *AdvancedCollector) GetHandlerMetrics(handlerType string) (*HandlerMetrics, bool) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	metrics, exists := ac.handlerMetrics[handlerType]
	return metrics, exists
}

// ExportMetrics exports metrics in JSON format
func (ac *AdvancedCollector) ExportMetrics() ([]byte, error) {
	snapshot := ac.GetSnapshot()
	return json.MarshalIndent(snapshot, "", "  ")
}

// Helper functions

func calculatePercentile(data []float64, percentile float64) float64 {
	if len(data) == 0 {
		return 0
	}

	// Simple percentile calculation (would use a proper sorting algorithm in production)
	sorted := make([]float64, len(data))
	copy(sorted, data)

	// Basic bubble sort for simplicity
	for i := 0; i < len(sorted); i++ {
		for j := 0; j < len(sorted)-1-i; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	index := int(float64(len(sorted)) * percentile)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}

func calculateAverage(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}
