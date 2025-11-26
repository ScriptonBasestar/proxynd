package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMetricsAggregator(t *testing.T) {
	// Initialize metrics first
	InitMetrics()
	defer ResetMetrics()

	m := GetMetrics()
	cacheTTL := 10 * time.Second

	agg := NewMetricsAggregator(m, cacheTTL)

	assert.NotNil(t, agg)
	assert.NotNil(t, agg.metrics)
	assert.NotNil(t, agg.cache)
	assert.Equal(t, 1*time.Minute, agg.interval)
	assert.Equal(t, 24, agg.timeSeries.maxPoints)
}

func TestMetricsAggregator_CollectOverview(t *testing.T) {
	// Initialize metrics
	InitMetrics()
	defer ResetMetrics()

	m := GetMetrics()
	agg := NewMetricsAggregator(m, 10*time.Second)

	// Add some test metrics
	m.HTTPRequestsTotal.WithLabelValues("GET", "/npm/package", "200", "npm").Add(100)
	m.CacheHitsTotal.WithLabelValues("npm", "filesystem").Add(80)
	m.CacheMissesTotal.WithLabelValues("npm", "filesystem").Add(20)

	// Collect overview
	overview := agg.collectOverview()

	assert.True(t, overview.TotalRequests > 0)
	assert.Equal(t, 0.8, overview.CacheHitRate)
}

func TestMetricsAggregator_CollectCacheStats(t *testing.T) {
	// Initialize metrics
	InitMetrics()
	defer ResetMetrics()

	m := GetMetrics()
	agg := NewMetricsAggregator(m, 10*time.Second)

	// Add test metrics
	m.CacheHitsTotal.WithLabelValues("npm", "filesystem").Add(800)
	m.CacheMissesTotal.WithLabelValues("npm", "filesystem").Add(200)
	m.CacheEvictionsTotal.WithLabelValues("npm", "filesystem", "lru").Add(50)
	m.CacheSizeBytes.WithLabelValues("npm", "filesystem").Set(1024 * 1024 * 100) // 100MB

	// Collect cache stats
	stats := agg.collectCacheStats()

	assert.Equal(t, float64(800), stats.Hits)
	assert.Equal(t, float64(200), stats.Misses)
	assert.Equal(t, 0.8, stats.HitRate)
	assert.True(t, stats.SizeBytes > 0)
	assert.Equal(t, float64(50), stats.Evictions)
}

func TestMetricsAggregator_CollectRequestStats(t *testing.T) {
	// Initialize metrics
	InitMetrics()
	defer ResetMetrics()

	m := GetMetrics()
	agg := NewMetricsAggregator(m, 10*time.Second)

	// Add test metrics
	m.HTTPRequestsTotal.WithLabelValues("GET", "/npm/package", "200", "npm").Add(100)
	m.HTTPRequestsTotal.WithLabelValues("GET", "/npm/package", "404", "npm").Add(10)
	m.HTTPRequestsTotal.WithLabelValues("GET", "/npm/package", "500", "npm").Add(5)
	m.ProxyRequestsTotal.WithLabelValues("npm", "registry.npmjs.org", "GET").Add(115)

	// Collect request stats
	stats := agg.collectRequestStats()

	assert.True(t, stats.Total >= 115)
	assert.True(t, stats.Success >= 100)
	assert.True(t, stats.Errors >= 15)
	assert.Contains(t, stats.ByPackageManager, "npm")
}

func TestMetricsAggregator_CollectPackageStats(t *testing.T) {
	// Initialize metrics
	InitMetrics()
	defer ResetMetrics()

	m := GetMetrics()
	agg := NewMetricsAggregator(m, 10*time.Second)

	// Add test metrics
	m.PackageDownloadsTotal.WithLabelValues("npm", "lodash", "4.17.21", "tgz").Add(50)
	m.PackageDownloadsTotal.WithLabelValues("npm", "react", "18.2.0", "tgz").Add(30)
	m.PackageDownloadsTotal.WithLabelValues("maven", "spring-boot", "3.0.0", "jar").Add(20)

	// Collect package stats
	stats := agg.collectPackageStats()

	assert.True(t, stats.Total >= 100)
	assert.Contains(t, stats.ByPackageManager, "npm")
	assert.Contains(t, stats.ByPackageManager, "maven")
}

func TestMetricsAggregator_GetDashboardMetrics(t *testing.T) {
	// Initialize metrics
	InitMetrics()
	defer ResetMetrics()

	m := GetMetrics()
	agg := NewMetricsAggregator(m, 10*time.Second)

	// Add some test data
	m.HTTPRequestsTotal.WithLabelValues("GET", "/npm/package", "200", "npm").Add(1000)
	m.CacheHitsTotal.WithLabelValues("npm", "filesystem").Add(800)
	m.CacheMissesTotal.WithLabelValues("npm", "filesystem").Add(200)

	// Get dashboard metrics
	dashboard, err := agg.GetDashboardMetrics()

	assert.NoError(t, err)
	assert.NotNil(t, dashboard)
	assert.True(t, dashboard.Overview.TotalRequests > 0)
	assert.Equal(t, 0.8, dashboard.Cache.HitRate)
	assert.NotZero(t, dashboard.LastUpdated)
}

func TestMetricsAggregator_GetDashboardMetrics_UseCache(t *testing.T) {
	// Initialize metrics
	InitMetrics()
	defer ResetMetrics()

	m := GetMetrics()
	agg := NewMetricsAggregator(m, 10*time.Second)

	// Add test data
	m.HTTPRequestsTotal.WithLabelValues("GET", "/npm/package", "200", "npm").Add(1000)

	// First call - should populate cache
	dashboard1, err := agg.GetDashboardMetrics()
	assert.NoError(t, err)
	assert.NotNil(t, dashboard1)

	// Modify metrics
	m.HTTPRequestsTotal.WithLabelValues("GET", "/npm/package", "200", "npm").Add(500)

	// Second call - should return cached data (no change)
	dashboard2, err := agg.GetDashboardMetrics()
	assert.NoError(t, err)
	assert.NotNil(t, dashboard2)

	// Should return same data from cache
	assert.Equal(t, dashboard1.Overview.TotalRequests, dashboard2.Overview.TotalRequests)
}

func TestMetricsAggregator_CollectTimeSeries(t *testing.T) {
	// Initialize metrics
	InitMetrics()
	defer ResetMetrics()

	m := GetMetrics()
	agg := NewMetricsAggregator(m, 10*time.Second)

	// Add test data
	m.HTTPRequestsTotal.WithLabelValues("GET", "/npm/package", "200", "npm").Add(100)
	m.CacheHitsTotal.WithLabelValues("npm", "filesystem").Add(80)

	// Collect time series
	agg.collectTimeSeries()

	// Get time series
	ts := agg.getTimeSeries()

	assert.Len(t, ts.RequestsHourly, 1)
	assert.Len(t, ts.CacheHitsHourly, 1)
	assert.True(t, ts.RequestsHourly[0].Value > 0)
	assert.True(t, ts.CacheHitsHourly[0].Value > 0)
}

func TestMetricsAggregator_TimeSeriesMaxPoints(t *testing.T) {
	// Initialize metrics
	InitMetrics()
	defer ResetMetrics()

	m := GetMetrics()
	agg := NewMetricsAggregator(m, 10*time.Second)
	agg.timeSeries.maxPoints = 3 // Limit to 3 for testing

	// Add more than maxPoints data points
	for i := 0; i < 5; i++ {
		m.HTTPRequestsTotal.WithLabelValues("GET", "/npm/package", "200", "npm").Add(float64(i * 100))
		agg.collectTimeSeries()
	}

	ts := agg.getTimeSeries()

	// Should only keep last 3 points
	assert.Len(t, ts.RequestsHourly, 3)
	assert.Len(t, ts.CacheHitsHourly, 3)
}

func TestGetAggregator(t *testing.T) {
	// Reset before test
	ResetMetrics()
	ResetAggregator()

	// Initialize metrics
	InitMetrics()

	// Get aggregator
	agg1 := GetAggregator()
	assert.NotNil(t, agg1)

	// Should return same instance
	agg2 := GetAggregator()
	assert.Equal(t, agg1, agg2)

	// Cleanup
	ResetAggregator()
}
