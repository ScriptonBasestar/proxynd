package metrics

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// MetricsAggregator aggregates metrics from Prometheus collectors
type MetricsAggregator struct {
	metrics    *Metrics
	cache      *MetricsCache
	interval   time.Duration
	timeSeries struct {
		mu              sync.RWMutex
		requestsHourly  []TimeSeriesPoint
		cacheHitsHourly []TimeSeriesPoint
		maxPoints       int
	}
}

// NewMetricsAggregator creates a new metrics aggregator
func NewMetricsAggregator(metrics *Metrics, cacheTTL time.Duration) *MetricsAggregator {
	agg := &MetricsAggregator{
		metrics:  metrics,
		cache:    NewMetricsCache(cacheTTL),
		interval: 1 * time.Minute,
	}
	agg.timeSeries.maxPoints = 24 // Last 24 hours
	return agg
}

// StartAggregationLoop starts the background aggregation loop
func (a *MetricsAggregator) StartAggregationLoop(ctx context.Context) {
	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()

	// Initial aggregation
	a.collectTimeSeries()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.collectTimeSeries()
		}
	}
}

// GetDashboardMetrics returns aggregated dashboard metrics
func (a *MetricsAggregator) GetDashboardMetrics() (*DashboardMetrics, error) {
	// Check cache first
	if cached, ok := a.cache.Get(); ok {
		return cached, nil
	}

	// Collect fresh metrics
	dashboard := &DashboardMetrics{
		Overview:    a.collectOverview(),
		Cache:       a.collectCacheStats(),
		Requests:    a.collectRequestStats(),
		Packages:    a.collectPackageStats(),
		TimeSeries:  a.getTimeSeries(),
		LastUpdated: time.Now(),
	}

	// Update cache
	a.cache.Set(dashboard)

	return dashboard, nil
}

// collectOverview collects overview statistics
func (a *MetricsAggregator) collectOverview() OverviewStats {
	totalRequests := a.getCounterVecTotal(a.metrics.HTTPRequestsTotal)
	cacheStats := a.collectCacheStats()

	// Calculate total storage from all package managers
	totalStorage := float64(0)
	packageManagers := []string{"npm", "maven", "pip", "docker", "apt", "yum", "apk"}
	for _, pm := range packageManagers {
		totalStorage += a.getGaugeVecValue(a.metrics.CacheSizeBytes, pm, "filesystem")
	}

	// Count total packages from package downloads
	totalPackages := a.getCounterVecTotal(a.metrics.PackageDownloadsTotal)

	return OverviewStats{
		TotalRequests:    totalRequests,
		TotalPackages:    totalPackages,
		CacheHitRate:     cacheStats.HitRate,
		StorageUsedBytes: totalStorage,
		UptimeSeconds:    a.getCounterValue(a.metrics.UptimeSeconds),
	}
}

// collectCacheStats collects cache statistics
func (a *MetricsAggregator) collectCacheStats() CacheStats {
	totalHits := a.getCounterVecTotal(a.metrics.CacheHitsTotal)
	totalMisses := a.getCounterVecTotal(a.metrics.CacheMissesTotal)
	total := totalHits + totalMisses

	hitRate := 0.0
	if total > 0 {
		hitRate = totalHits / total
	}

	totalEvictions := a.getCounterVecTotal(a.metrics.CacheEvictionsTotal)
	totalSize := a.getGaugeVecTotal(a.metrics.CacheSizeBytes)

	return CacheStats{
		Hits:      totalHits,
		Misses:    totalMisses,
		HitRate:   hitRate,
		SizeBytes: totalSize,
		Evictions: totalEvictions,
	}
}

// collectRequestStats collects request statistics
func (a *MetricsAggregator) collectRequestStats() RequestStats {
	totalRequests := a.getCounterVecTotal(a.metrics.HTTPRequestsTotal)

	// Count successful requests (status 2xx, 3xx)
	successRequests := a.getCounterVecTotalWithFilter(a.metrics.HTTPRequestsTotal, func(labels map[string]string) bool {
		status := labels["status"]
		return len(status) > 0 && (status[0] == '2' || status[0] == '3')
	})

	// Count error requests (status 4xx, 5xx)
	errorRequests := a.getCounterVecTotalWithFilter(a.metrics.HTTPRequestsTotal, func(labels map[string]string) bool {
		status := labels["status"]
		return len(status) > 0 && (status[0] == '4' || status[0] == '5')
	})

	// Collect by package manager
	byPM := make(map[string]float64)
	packageManagers := []string{"npm", "maven", "pip", "docker", "apt", "yum", "apk", "go", "nuget", "cargo"}
	for _, pm := range packageManagers {
		count := a.getCounterVecTotalWithFilter(a.metrics.ProxyRequestsTotal, func(labels map[string]string) bool {
			return labels["registry_type"] == pm
		})
		if count > 0 {
			byPM[pm] = count
		}
	}

	return RequestStats{
		Total:            totalRequests,
		Success:          successRequests,
		Errors:           errorRequests,
		ByPackageManager: byPM,
	}
}

// collectPackageStats collects package statistics
func (a *MetricsAggregator) collectPackageStats() PackageStats {
	totalPackages := a.getCounterVecTotal(a.metrics.PackageDownloadsTotal)

	// Collect by package manager
	byPM := make(map[string]float64)
	packageManagers := []string{"npm", "maven", "pip", "docker", "apt", "yum", "apk", "go", "nuget", "cargo"}
	for _, pm := range packageManagers {
		count := a.getCounterVecTotalWithFilter(a.metrics.PackageDownloadsTotal, func(labels map[string]string) bool {
			return labels["registry_type"] == pm
		})
		if count > 0 {
			byPM[pm] = count
		}
	}

	return PackageStats{
		Total:            totalPackages,
		ByPackageManager: byPM,
	}
}

// collectTimeSeries collects time-series data point
func (a *MetricsAggregator) collectTimeSeries() {
	a.timeSeries.mu.Lock()
	defer a.timeSeries.mu.Unlock()

	now := time.Now().Truncate(time.Hour)

	// Add new data point for requests
	requestsPoint := TimeSeriesPoint{
		Timestamp: now,
		Value:     a.getCounterVecTotal(a.metrics.HTTPRequestsTotal),
	}
	a.timeSeries.requestsHourly = append(a.timeSeries.requestsHourly, requestsPoint)

	// Add new data point for cache hits
	cacheHitsPoint := TimeSeriesPoint{
		Timestamp: now,
		Value:     a.getCounterVecTotal(a.metrics.CacheHitsTotal),
	}
	a.timeSeries.cacheHitsHourly = append(a.timeSeries.cacheHitsHourly, cacheHitsPoint)

	// Keep only last N points
	if len(a.timeSeries.requestsHourly) > a.timeSeries.maxPoints {
		a.timeSeries.requestsHourly = a.timeSeries.requestsHourly[len(a.timeSeries.requestsHourly)-a.timeSeries.maxPoints:]
	}
	if len(a.timeSeries.cacheHitsHourly) > a.timeSeries.maxPoints {
		a.timeSeries.cacheHitsHourly = a.timeSeries.cacheHitsHourly[len(a.timeSeries.cacheHitsHourly)-a.timeSeries.maxPoints:]
	}
}

// getTimeSeries returns current time-series data
func (a *MetricsAggregator) getTimeSeries() TimeSeriesData {
	a.timeSeries.mu.RLock()
	defer a.timeSeries.mu.RUnlock()

	return TimeSeriesData{
		RequestsHourly:  append([]TimeSeriesPoint{}, a.timeSeries.requestsHourly...),
		CacheHitsHourly: append([]TimeSeriesPoint{}, a.timeSeries.cacheHitsHourly...),
	}
}

// Helper functions to extract metric values

func (a *MetricsAggregator) getCounterValue(metric prometheus.Counter) float64 {
	if metric == nil {
		return 0.0
	}
	var m dto.Metric
	if err := metric.Write(&m); err != nil {
		return 0.0
	}
	if m.Counter != nil {
		return m.Counter.GetValue()
	}
	return 0.0
}

func (a *MetricsAggregator) getGaugeVecValue(vec *prometheus.GaugeVec, labelValues ...string) float64 {
	if vec == nil {
		return 0.0
	}
	metric, err := vec.GetMetricWithLabelValues(labelValues...)
	if err != nil {
		return 0.0
	}
	var m dto.Metric
	if err := metric.Write(&m); err != nil {
		return 0.0
	}
	if m.Gauge != nil {
		return m.Gauge.GetValue()
	}
	return 0.0
}

func (a *MetricsAggregator) getCounterVecTotal(vec *prometheus.CounterVec) float64 {
	if vec == nil {
		return 0.0
	}
	ch := make(chan prometheus.Metric, 100)
	go func() {
		vec.Collect(ch)
		close(ch)
	}()

	var total float64
	for metric := range ch {
		var m dto.Metric
		if err := metric.Write(&m); err == nil && m.Counter != nil {
			total += m.Counter.GetValue()
		}
	}
	return total
}

func (a *MetricsAggregator) getGaugeVecTotal(vec *prometheus.GaugeVec) float64 {
	if vec == nil {
		return 0.0
	}
	ch := make(chan prometheus.Metric, 100)
	go func() {
		vec.Collect(ch)
		close(ch)
	}()

	var total float64
	for metric := range ch {
		var m dto.Metric
		if err := metric.Write(&m); err == nil && m.Gauge != nil {
			total += m.Gauge.GetValue()
		}
	}
	return total
}

func (a *MetricsAggregator) getCounterVecTotalWithFilter(vec *prometheus.CounterVec, filter func(map[string]string) bool) float64 {
	if vec == nil {
		return 0.0
	}
	ch := make(chan prometheus.Metric, 100)
	go func() {
		vec.Collect(ch)
		close(ch)
	}()

	var total float64
	for metric := range ch {
		var m dto.Metric
		if err := metric.Write(&m); err == nil && m.Counter != nil {
			// Extract labels
			labels := make(map[string]string)
			for _, label := range m.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}

			if filter(labels) {
				total += m.Counter.GetValue()
			}
		}
	}
	return total
}

// Global aggregator instance
var (
	globalAggregator *MetricsAggregator
	aggregatorOnce   sync.Once
)

// GetAggregator returns the global metrics aggregator
func GetAggregator() *MetricsAggregator {
	aggregatorOnce.Do(func() {
		globalAggregator = NewMetricsAggregator(GetMetrics(), 10*time.Second)
	})
	return globalAggregator
}

// ResetAggregator resets the global aggregator (for testing)
func ResetAggregator() {
	globalAggregator = nil
	aggregatorOnce = sync.Once{}
}
