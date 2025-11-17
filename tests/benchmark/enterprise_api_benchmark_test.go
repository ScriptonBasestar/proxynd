//go:build benchmark
// +build benchmark

package benchmark

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/adapters/http/fiber/middleware/enterprise"
	"proxynd/internal/adapters/http/fiber/routers"
)

// setupBenchmarkApp creates a test app for benchmarking
func setupBenchmarkApp() *fiber.App {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})
	routers.SetupEnterpriseRoutes(app, nil) // Dev mode
	return app
}

// setupBenchmarkAppWithCache creates a test app with caching enabled
func setupBenchmarkAppWithCache() *fiber.App {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	// Add cache middleware
	cache := enterprise.AnalyticsCacheMiddleware()
	app.Use(cache.Middleware())

	routers.SetupEnterpriseRoutes(app, nil)
	return app
}

// setupBenchmarkAppWithRateLimit creates a test app with rate limiting
func setupBenchmarkAppWithRateLimit() *fiber.App {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	// Add rate limiter
	rateLimiter := enterprise.DefaultEnterpriseRateLimiter()
	app.Use(rateLimiter.Middleware())

	routers.SetupEnterpriseRoutes(app, nil)
	return app
}

// BenchmarkRBACListRoles benchmarks listing roles endpoint
func BenchmarkRBACListRoles(b *testing.B) {
	app := setupBenchmarkApp()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/enterprise/rbac/roles", nil)
		_, _ = app.Test(req, -1)
	}
}

// BenchmarkRBACListRolesWithCache benchmarks with caching
func BenchmarkRBACListRolesWithCache(b *testing.B) {
	app := setupBenchmarkAppWithCache()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/enterprise/rbac/roles", nil)
		_, _ = app.Test(req, -1)
	}
}

// BenchmarkAuditListEvents benchmarks audit events listing
func BenchmarkAuditListEvents(b *testing.B) {
	app := setupBenchmarkApp()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/enterprise/audit/events", nil)
		_, _ = app.Test(req, -1)
	}
}

// BenchmarkAuditListEventsWithCache benchmarks with caching
func BenchmarkAuditListEventsWithCache(b *testing.B) {
	app := setupBenchmarkAppWithCache()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/enterprise/audit/events", nil)
		_, _ = app.Test(req, -1)
	}
}

// BenchmarkAnalyticsOverview benchmarks analytics overview
func BenchmarkAnalyticsOverview(b *testing.B) {
	app := setupBenchmarkApp()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/enterprise/analytics/overview", nil)
		_, _ = app.Test(req, -1)
	}
}

// BenchmarkAnalyticsOverviewWithCache benchmarks with caching
func BenchmarkAnalyticsOverviewWithCache(b *testing.B) {
	app := setupBenchmarkAppWithCache()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/enterprise/analytics/overview", nil)
		_, _ = app.Test(req, -1)
	}
}

// BenchmarkSecurityVulnerabilities benchmarks security vulnerabilities listing
func BenchmarkSecurityVulnerabilities(b *testing.B) {
	app := setupBenchmarkApp()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/enterprise/security/vulnerabilities", nil)
		_, _ = app.Test(req, -1)
	}
}

// BenchmarkSecurityVulnerabilitiesWithCache benchmarks with caching
func BenchmarkSecurityVulnerabilitiesWithCache(b *testing.B) {
	app := setupBenchmarkAppWithCache()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/enterprise/security/vulnerabilities", nil)
		_, _ = app.Test(req, -1)
	}
}

// BenchmarkAlertsList benchmarks alerts listing
func BenchmarkAlertsList(b *testing.B) {
	app := setupBenchmarkApp()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/enterprise/alerts", nil)
		_, _ = app.Test(req, -1)
	}
}

// BenchmarkAlertsListWithCache benchmarks with caching
func BenchmarkAlertsListWithCache(b *testing.B) {
	app := setupBenchmarkAppWithCache()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/enterprise/alerts", nil)
		_, _ = app.Test(req, -1)
	}
}

// BenchmarkRateLimiterOverhead benchmarks rate limiter overhead
func BenchmarkRateLimiterOverhead(b *testing.B) {
	app := setupBenchmarkAppWithRateLimit()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/enterprise/rbac/roles", nil)
		_, _ = app.Test(req, -1)
	}
}

// BenchmarkPagination benchmarks pagination performance
func BenchmarkPagination(b *testing.B) {
	app := setupBenchmarkApp()

	b.Run("Page1", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("GET", "/api/v1/enterprise/rbac/roles?page=1&per_page=20", nil)
			_, _ = app.Test(req, -1)
		}
	})

	b.Run("Page2", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("GET", "/api/v1/enterprise/rbac/roles?page=2&per_page=20", nil)
			_, _ = app.Test(req, -1)
		}
	})

	b.Run("LargePage", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("GET", "/api/v1/enterprise/rbac/roles?page=1&per_page=100", nil)
			_, _ = app.Test(req, -1)
		}
	})
}

// BenchmarkEndToEnd benchmarks end-to-end workflow
func BenchmarkEndToEnd(b *testing.B) {
	app := setupBenchmarkApp()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Step 1: List roles
		req1 := httptest.NewRequest("GET", "/api/v1/enterprise/rbac/roles", nil)
		_, _ = app.Test(req1, -1)

		// Step 2: Check audit events
		req2 := httptest.NewRequest("GET", "/api/v1/enterprise/audit/events", nil)
		_, _ = app.Test(req2, -1)

		// Step 3: Get analytics
		req3 := httptest.NewRequest("GET", "/api/v1/enterprise/analytics/overview", nil)
		_, _ = app.Test(req3, -1)
	}
}

// BenchmarkCacheImpact measures cache performance impact
func BenchmarkCacheImpact(b *testing.B) {
	b.Run("WithoutCache", func(b *testing.B) {
		app := setupBenchmarkApp()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("GET", "/api/v1/enterprise/analytics/overview", nil)
			_, _ = app.Test(req, -1)
		}
	})

	b.Run("WithCache", func(b *testing.B) {
		app := setupBenchmarkAppWithCache()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("GET", "/api/v1/enterprise/analytics/overview", nil)
			_, _ = app.Test(req, -1)
		}
	})
}
