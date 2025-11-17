//go:build integration
// +build integration

package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/internal/adapters/http/fiber/middleware/enterprise"
	"proxynd/internal/adapters/http/fiber/routers"
)

// setupEnterpriseTestApp creates a test Fiber app with enterprise routes
func setupEnterpriseTestApp() *fiber.App {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})
	routers.SetupEnterpriseRoutes(app, nil)
	return app
}

// TestEnterpriseAPIIntegration_BasicRouting tests basic enterprise API routing
func TestEnterpriseAPIIntegration_BasicRouting(t *testing.T) {
	app := setupEnterpriseTestApp()

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		expectedFields []string
	}{
		{
			name:           "RBAC - List Roles",
			method:         "GET",
			path:           "/api/v1/enterprise/rbac/roles",
			expectedStatus: 200,
			expectedFields: []string{"success", "data", "pagination", "metadata"},
		},
		{
			name:           "Audit - List Events",
			method:         "GET",
			path:           "/api/v1/enterprise/audit/events",
			expectedStatus: 200,
			expectedFields: []string{"success", "data", "pagination", "metadata"},
		},
		{
			name:           "Analytics - Overview",
			method:         "GET",
			path:           "/api/v1/enterprise/analytics/overview",
			expectedStatus: 200,
			expectedFields: []string{"success", "data", "metadata"},
		},
		{
			name:           "Security - Vulnerabilities",
			method:         "GET",
			path:           "/api/v1/enterprise/security/vulnerabilities",
			expectedStatus: 200,
			expectedFields: []string{"success", "data", "pagination", "metadata"},
		},
		{
			name:           "Alerts - List",
			method:         "GET",
			path:           "/api/v1/enterprise/alerts",
			expectedStatus: 200,
			expectedFields: []string{"success", "data", "pagination", "metadata"},
		},
		{
			name:           "License - Info",
			method:         "GET",
			path:           "/api/v1/enterprise/license/info",
			expectedStatus: 200,
			expectedFields: []string{"success", "data", "metadata"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Status code mismatch")

			// Verify response structure
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			var result map[string]interface{}
			err = json.Unmarshal(body, &result)
			require.NoError(t, err)

			for _, field := range tt.expectedFields {
				assert.Contains(t, result, field, "Response should contain %s field", field)
			}
		})
	}
}

// TestEnterpriseAPIIntegration_MiddlewareChain tests middleware execution order
func TestEnterpriseAPIIntegration_MiddlewareChain(t *testing.T) {
	app := fiber.New()

	// Track middleware execution order
	executionOrder := []string{}

	// Add custom tracking middleware
	app.Use(func(c *fiber.Ctx) error {
		executionOrder = append(executionOrder, "request-id")
		c.Locals("requestid", "test-123")
		return c.Next()
	})

	// Add rate limiter
	rateLimiter := enterprise.NewRateLimiter(enterprise.RateLimitConfig{
		Max:    5,
		Window: 1 * time.Minute,
		Burst:  2,
	})
	app.Use(func(c *fiber.Ctx) error {
		executionOrder = append(executionOrder, "rate-limiter")
		return rateLimiter.Middleware()(c)
	})

	// Setup enterprise routes
	routers.SetupEnterpriseRoutes(app, nil)

	// Test request
	req := httptest.NewRequest("GET", "/api/v1/enterprise/rbac/roles", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	assert.Contains(t, executionOrder, "request-id", "Request ID middleware should execute")
	assert.Contains(t, executionOrder, "rate-limiter", "Rate limiter should execute")

	// Verify rate limit headers are set
	assert.NotEmpty(t, resp.Header.Get("X-RateLimit-Limit"))
	assert.NotEmpty(t, resp.Header.Get("X-RateLimit-Remaining"))
}

// TestEnterpriseAPIIntegration_RateLimiting tests rate limiting in integration
func TestEnterpriseAPIIntegration_RateLimiting(t *testing.T) {
	app := fiber.New()

	// Add strict rate limiter
	rateLimiter := enterprise.NewRateLimiter(enterprise.RateLimitConfig{
		Max:    3,
		Window: 1 * time.Minute,
		Burst:  1,
	})
	app.Use(rateLimiter.Middleware())

	// Setup routes
	routers.SetupEnterpriseRoutes(app, nil)

	// Send requests up to limit
	for i := 1; i <= 4; i++ {
		req := httptest.NewRequest("GET", "/api/v1/enterprise/rbac/roles", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode, "Request %d should succeed", i)
	}

	// Next request should be rate limited
	req := httptest.NewRequest("GET", "/api/v1/enterprise/rbac/roles", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 429, resp.StatusCode, "Should be rate limited")
	assert.NotEmpty(t, resp.Header.Get("Retry-After"))
}

// TestEnterpriseAPIIntegration_LicenseMiddleware tests license checking
func TestEnterpriseAPIIntegration_LicenseMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		licenseFeatures enterprise.EnterpriseFeatures
		expectedStatus  int
	}{
		{
			name:           "Dev mode (nil license) - should allow",
			licenseFeatures: nil,
			expectedStatus:  200,
		},
		{
			name: "Valid license - should allow",
			licenseFeatures: &mockEnterpriseFeatures{},
			expectedStatus:  200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			// Setup routes with specified license features
			routers.SetupEnterpriseRoutes(app, tt.licenseFeatures)

			req := httptest.NewRequest("GET", "/api/v1/enterprise/rbac/roles", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

// TestEnterpriseAPIIntegration_EndToEndScenario tests realistic usage scenarios
func TestEnterpriseAPIIntegration_EndToEndScenario(t *testing.T) {
	app := fiber.New()
	routers.SetupEnterpriseRoutes(app, nil)

	// Scenario: Admin creates a role, assigns permissions, checks audit log
	t.Run("Create Role and Check Audit", func(t *testing.T) {
		// Step 1: List existing roles
		req := httptest.NewRequest("GET", "/api/v1/enterprise/rbac/roles", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 200, resp.StatusCode)

		// Step 2: Create a new role
		rolePayload := `{"name":"test_role","description":"Test role","permissions":["read:packages","write:packages"]}`
		req = httptest.NewRequest("POST", "/api/v1/enterprise/rbac/roles", strings.NewReader(rolePayload))
		req.Header.Set("Content-Type", "application/json")
		resp, err = app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 201, resp.StatusCode)

		// Step 3: Check audit events
		req = httptest.NewRequest("GET", "/api/v1/enterprise/audit/events", nil)
		resp, err = app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 200, resp.StatusCode)

		// Verify audit response structure
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var auditResult map[string]interface{}
		err = json.Unmarshal(body, &auditResult)
		require.NoError(t, err)
		assert.True(t, auditResult["success"].(bool))
		assert.Contains(t, auditResult, "data")
		assert.Contains(t, auditResult, "pagination")
	})

	t.Run("Security Scan and Alert Creation", func(t *testing.T) {
		// Step 1: List vulnerabilities
		req := httptest.NewRequest("GET", "/api/v1/enterprise/security/vulnerabilities", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 200, resp.StatusCode)

		// Step 2: Trigger a scan
		scanPayload := `{"scope":"all","priority":"high"}`
		req = httptest.NewRequest("POST", "/api/v1/enterprise/security/scan", strings.NewReader(scanPayload))
		req.Header.Set("Content-Type", "application/json")
		resp, err = app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 202, resp.StatusCode)

		// Step 3: Create alert rule for vulnerabilities
		alertPayload := `{"name":"Critical Vuln Alert","type":"threshold","severity":"critical","conditions":{"threshold":0},"actions":[{"type":"email","config":{"recipients":["security@example.com"]}}]}`
		req = httptest.NewRequest("POST", "/api/v1/enterprise/alerts/rules", strings.NewReader(alertPayload))
		req.Header.Set("Content-Type", "application/json")
		resp, err = app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 201, resp.StatusCode)
	})

	t.Run("Analytics and Reporting Workflow", func(t *testing.T) {
		// Step 1: Get analytics overview
		req := httptest.NewRequest("GET", "/api/v1/enterprise/analytics/overview", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 200, resp.StatusCode)

		// Step 2: Get usage stats
		req = httptest.NewRequest("GET", "/api/v1/enterprise/analytics/usage", nil)
		resp, err = app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 200, resp.StatusCode)

		// Step 3: Create custom report
		reportPayload := `{"name":"Weekly Report","type":"usage","schedule":"0 0 * * 1"}`
		req = httptest.NewRequest("POST", "/api/v1/enterprise/analytics/reports", strings.NewReader(reportPayload))
		req.Header.Set("Content-Type", "application/json")
		resp, err = app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 201, resp.StatusCode)

		// Step 4: Export audit logs
		exportPayload := `{"format":"json","start_date":"2025-01-01T00:00:00Z","end_date":"2025-01-31T23:59:59Z"}`
		req = httptest.NewRequest("POST", "/api/v1/enterprise/audit/export", strings.NewReader(exportPayload))
		req.Header.Set("Content-Type", "application/json")
		resp, err = app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 200, resp.StatusCode)
	})
}

// TestEnterpriseAPIIntegration_Pagination tests pagination across endpoints
func TestEnterpriseAPIIntegration_Pagination(t *testing.T) {
	app := fiber.New()
	routers.SetupEnterpriseRoutes(app, nil)

	tests := []struct {
		name     string
		endpoint string
		page     int
		perPage  int
	}{
		{"Roles pagination", "/api/v1/enterprise/rbac/roles?page=1&per_page=10", 1, 10},
		{"Events pagination", "/api/v1/enterprise/audit/events?page=2&per_page=20", 2, 20},
		{"Max per_page cap", "/api/v1/enterprise/rbac/roles?page=1&per_page=200", 1, 100}, // Should cap at 100
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.endpoint, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, 200, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			var result map[string]interface{}
			err = json.Unmarshal(body, &result)
			require.NoError(t, err)

			assert.Contains(t, result, "pagination")
			pagination := result["pagination"].(map[string]interface{})

			// For max per_page cap test, verify it's capped at 100
			if tt.perPage == 100 {
				assert.Equal(t, float64(100), pagination["per_page"].(float64), "per_page should be capped at 100")
			}
		})
	}
}

// TestEnterpriseAPIIntegration_ErrorHandling tests error scenarios
func TestEnterpriseAPIIntegration_ErrorHandling(t *testing.T) {
	app := fiber.New()
	routers.SetupEnterpriseRoutes(app, nil)

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Invalid event ID - 404",
			method:         "GET",
			path:           "/api/v1/enterprise/audit/events/invalid_id",
			expectedStatus: 404,
			expectedError:  "EVENT_NOT_FOUND",
		},
		{
			name:           "Invalid alert ID - 404",
			method:         "GET",
			path:           "/api/v1/enterprise/alerts/invalid_id",
			expectedStatus: 404,
			expectedError:  "ALERT_NOT_FOUND",
		},
		{
			name:           "Invalid JSON payload - 400",
			method:         "POST",
			path:           "/api/v1/enterprise/rbac/roles",
			body:           `{invalid json`,
			expectedStatus: 400,
			expectedError:  "INVALID_REQUEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.Contains(t, string(body), tt.expectedError)
		})
	}
}

// Mock implementation for testing
type mockEnterpriseFeatures struct{}

func (m *mockEnterpriseFeatures) IsEnabled() bool {
	return true
}

func (m *mockEnterpriseFeatures) HasFeature(feature string) bool {
	return true
}
