package metrics

import (
	"sync"
	"time"
)

// DashboardMetrics represents aggregated metrics for the dashboard
type DashboardMetrics struct {
	Overview    OverviewStats  `json:"overview"`
	Cache       CacheStats     `json:"cache"`
	Requests    RequestStats   `json:"requests"`
	Packages    PackageStats   `json:"packages"`
	TimeSeries  TimeSeriesData `json:"time_series"`
	LastUpdated time.Time      `json:"last_updated"`
}

// OverviewStats represents high-level overview statistics
type OverviewStats struct {
	TotalRequests    float64 `json:"total_requests"`
	TotalPackages    float64 `json:"total_packages"`
	CacheHitRate     float64 `json:"cache_hit_rate"`
	StorageUsedBytes float64 `json:"storage_used_bytes"`
	UptimeSeconds    float64 `json:"uptime_seconds"`
}

// CacheStats represents cache statistics
type CacheStats struct {
	Hits      float64 `json:"hits"`
	Misses    float64 `json:"misses"`
	HitRate   float64 `json:"hit_rate"`
	SizeBytes float64 `json:"size_bytes"`
	Evictions float64 `json:"evictions"`
}

// RequestStats represents request statistics
type RequestStats struct {
	Total            float64            `json:"total"`
	Success          float64            `json:"success"`
	Errors           float64            `json:"errors"`
	ByPackageManager map[string]float64 `json:"by_package_manager"`
}

// PackageStats represents package statistics
type PackageStats struct {
	Total            float64            `json:"total"`
	ByPackageManager map[string]float64 `json:"by_package_manager"`
}

// TimeSeriesData represents time-series metrics
type TimeSeriesData struct {
	RequestsHourly  []TimeSeriesPoint `json:"requests_hourly"`
	CacheHitsHourly []TimeSeriesPoint `json:"cache_hits_hourly"`
}

// TimeSeriesPoint represents a single time-series data point
type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// MetricsCache provides caching for dashboard metrics
type MetricsCache struct {
	mu         sync.RWMutex
	data       *DashboardMetrics
	lastUpdate time.Time
	ttl        time.Duration
}

// NewMetricsCache creates a new metrics cache with specified TTL
func NewMetricsCache(ttl time.Duration) *MetricsCache {
	return &MetricsCache{
		ttl: ttl,
	}
}

// Get retrieves cached metrics if still valid
func (c *MetricsCache) Get() (*DashboardMetrics, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.data == nil || time.Since(c.lastUpdate) > c.ttl {
		return nil, false
	}

	return c.data, true
}

// Set updates the cache with new metrics
func (c *MetricsCache) Set(data *DashboardMetrics) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = data
	c.lastUpdate = time.Now()
}

// Clear removes cached data
func (c *MetricsCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = nil
	c.lastUpdate = time.Time{}
}
