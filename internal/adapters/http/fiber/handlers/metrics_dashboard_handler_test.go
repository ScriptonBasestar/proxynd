package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"

	"proxynd/internal/logging"
	"proxynd/internal/metrics"
)

func TestNewMetricsDashboardHandler(t *testing.T) {
	logger := logging.NewLogger("test")

	handler := NewMetricsDashboardHandler(logger)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.aggregator)
	assert.NotNil(t, handler.logger)
}

func TestMetricsDashboardHandler_GetDashboardMetrics(t *testing.T) {
	// Initialize metrics
	metrics.InitMetrics()
	defer metrics.ResetMetrics()
	defer metrics.ResetAggregator()

	// Add some test metrics
	m := metrics.GetMetrics()
	m.HTTPRequestsTotal.WithLabelValues("GET", "/npm/package", "200", "npm").Add(1000)
	m.HTTPRequestsTotal.WithLabelValues("GET", "/npm/package", "404", "npm").Add(50)
	m.CacheHitsTotal.WithLabelValues("npm", "filesystem").Add(850)
	m.CacheMissesTotal.WithLabelValues("npm", "filesystem").Add(150)
	m.PackageDownloadsTotal.WithLabelValues("npm", "lodash", "4.17.21", "tgz").Add(500)

	// Create handler
	logger := logging.NewLogger("test")
	handler := NewMetricsDashboardHandler(logger)

	// Create test app
	app := fiber.New()
	app.Get("/api/v1/metrics/dashboard", handler.GetDashboardMetrics)

	// Test request
	req := httptest.NewRequest("GET", "/api/v1/metrics/dashboard", nil)
	resp, err := app.Test(req)
	defer func() { _ = resp.Body.Close() }()

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	// Parse response
	var dashboard metrics.DashboardMetrics
	err = json.NewDecoder(resp.Body).Decode(&dashboard)
	assert.NoError(t, err)

	// Verify data
	assert.True(t, dashboard.Overview.TotalRequests > 0, "Should have total requests")
	assert.True(t, dashboard.Cache.HitRate > 0, "Should have cache hit rate")
	assert.NotZero(t, dashboard.LastUpdated, "Should have last updated timestamp")
}

func TestMetricsDashboardHandler_GetDashboardMetrics_EmptyMetrics(t *testing.T) {
	// Reset metrics to ensure clean state
	metrics.ResetMetrics()
	metrics.ResetAggregator()
	metrics.InitMetrics()
	defer metrics.ResetMetrics()
	defer metrics.ResetAggregator()

	// Create handler
	logger := logging.NewLogger("test")
	handler := NewMetricsDashboardHandler(logger)

	// Create test app
	app := fiber.New()
	app.Get("/api/v1/metrics/dashboard", handler.GetDashboardMetrics)

	// Test request
	req := httptest.NewRequest("GET", "/api/v1/metrics/dashboard", nil)
	resp, err := app.Test(req)
	defer func() { _ = resp.Body.Close() }()

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	// Parse response
	var dashboard metrics.DashboardMetrics
	err = json.NewDecoder(resp.Body).Decode(&dashboard)
	assert.NoError(t, err)

	// Should return empty but valid dashboard
	assert.NotZero(t, dashboard.LastUpdated)
	assert.NotNil(t, dashboard.Requests.ByPackageManager)
	assert.NotNil(t, dashboard.Packages.ByPackageManager)
}

func TestMetricsDashboardHandler_GetDashboardMetrics_MultiplePackageManagers(t *testing.T) {
	// Initialize metrics
	metrics.InitMetrics()
	defer metrics.ResetMetrics()
	defer metrics.ResetAggregator()

	// Add metrics for multiple package managers
	m := metrics.GetMetrics()
	m.ProxyRequestsTotal.WithLabelValues("npm", "registry.npmjs.org", "GET").Add(100)
	m.ProxyRequestsTotal.WithLabelValues("maven", "repo.maven.apache.org", "GET").Add(50)
	m.ProxyRequestsTotal.WithLabelValues("pip", "pypi.org", "GET").Add(75)

	m.PackageDownloadsTotal.WithLabelValues("npm", "react", "18.2.0", "tgz").Add(100)
	m.PackageDownloadsTotal.WithLabelValues("maven", "spring-boot", "3.0.0", "jar").Add(50)
	m.PackageDownloadsTotal.WithLabelValues("pip", "django", "4.2.0", "whl").Add(75)

	// Create handler
	logger := logging.NewLogger("test")
	handler := NewMetricsDashboardHandler(logger)

	// Create test app
	app := fiber.New()
	app.Get("/api/v1/metrics/dashboard", handler.GetDashboardMetrics)

	// Test request
	req := httptest.NewRequest("GET", "/api/v1/metrics/dashboard", nil)
	resp, err := app.Test(req)
	defer func() { _ = resp.Body.Close() }()

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	// Parse response
	var dashboard metrics.DashboardMetrics
	err = json.NewDecoder(resp.Body).Decode(&dashboard)
	assert.NoError(t, err)

	// Verify multiple package managers
	assert.Contains(t, dashboard.Requests.ByPackageManager, "npm")
	assert.Contains(t, dashboard.Requests.ByPackageManager, "maven")
	assert.Contains(t, dashboard.Requests.ByPackageManager, "pip")

	assert.Contains(t, dashboard.Packages.ByPackageManager, "npm")
	assert.Contains(t, dashboard.Packages.ByPackageManager, "maven")
	assert.Contains(t, dashboard.Packages.ByPackageManager, "pip")
}

func TestMetricsDashboardHandler_ResponseStructure(t *testing.T) {
	// Initialize metrics
	metrics.InitMetrics()
	defer metrics.ResetMetrics()
	defer metrics.ResetAggregator()

	// Add test data
	m := metrics.GetMetrics()
	m.HTTPRequestsTotal.WithLabelValues("GET", "/npm/package", "200", "npm").Add(100)
	m.CacheHitsTotal.WithLabelValues("npm", "filesystem").Add(80)

	// Create handler
	logger := logging.NewLogger("test")
	handler := NewMetricsDashboardHandler(logger)

	// Create test app
	app := fiber.New()
	app.Get("/api/v1/metrics/dashboard", handler.GetDashboardMetrics)

	// Test request
	req := httptest.NewRequest("GET", "/api/v1/metrics/dashboard", nil)
	resp, err := app.Test(req)
	defer func() { _ = resp.Body.Close() }()

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	// Parse as generic map to verify structure
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)

	// Verify top-level keys
	assert.Contains(t, result, "overview")
	assert.Contains(t, result, "cache")
	assert.Contains(t, result, "requests")
	assert.Contains(t, result, "packages")
	assert.Contains(t, result, "time_series")
	assert.Contains(t, result, "last_updated")

	// Verify overview structure
	overview := result["overview"].(map[string]interface{})
	assert.Contains(t, overview, "total_requests")
	assert.Contains(t, overview, "total_packages")
	assert.Contains(t, overview, "cache_hit_rate")
	assert.Contains(t, overview, "storage_used_bytes")
	assert.Contains(t, overview, "uptime_seconds")

	// Verify cache structure
	cache := result["cache"].(map[string]interface{})
	assert.Contains(t, cache, "hits")
	assert.Contains(t, cache, "misses")
	assert.Contains(t, cache, "hit_rate")
	assert.Contains(t, cache, "size_bytes")
	assert.Contains(t, cache, "evictions")
}
