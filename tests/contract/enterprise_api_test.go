//go:build contract
// +build contract

package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/internal/adapters/http/fiber/routers"
)

// TestEnterpriseAPIContract validates the contract/schema for all enterprise endpoints
// This ensures API stability before WebUI integration work begins
func TestEnterpriseAPIContract(t *testing.T) {
	app := fiber.New()
	routers.SetupEnterpriseRoutes(app, nil) // nil = dev mode, no license check

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		validateSchema func(*testing.T, map[string]interface{})
	}{
		// RBAC Endpoints (12)
		{
			name:           "List Roles",
			method:         "GET",
			path:           "/api/v1/enterprise/rbac/roles",
			expectedStatus: http.StatusOK,
			validateSchema: validatePaginatedListResponse,
		},
		{
			name:           "Get Role",
			method:         "GET",
			path:           "/api/v1/enterprise/rbac/roles/role_admin",
			expectedStatus: http.StatusOK,
			validateSchema: validateSuccessResponse,
		},
		{
			name:           "List Permissions",
			method:         "GET",
			path:           "/api/v1/enterprise/rbac/permissions",
			expectedStatus: http.StatusOK,
			validateSchema: validateSuccessResponse,
		},
		{
			name:           "Get Role Permissions",
			method:         "GET",
			path:           "/api/v1/enterprise/rbac/roles/role_admin/permissions",
			expectedStatus: http.StatusOK,
			validateSchema: validateSuccessResponse,
		},
		{
			name:           "Get User Roles",
			method:         "GET",
			path:           "/api/v1/enterprise/rbac/users/user_001/roles",
			expectedStatus: http.StatusOK,
			validateSchema: validateSuccessResponse,
		},

		// Audit Endpoints (8)
		{
			name:           "List Audit Events",
			method:         "GET",
			path:           "/api/v1/enterprise/audit/events",
			expectedStatus: http.StatusOK,
			validateSchema: validatePaginatedListResponse,
		},
		{
			name:           "Get Audit Event",
			method:         "GET",
			path:           "/api/v1/enterprise/audit/events/evt_001",
			expectedStatus: http.StatusOK,
			validateSchema: validateSuccessResponse,
		},
		{
			name:           "Search Audit Events",
			method:         "GET",
			path:           "/api/v1/enterprise/audit/events/search?q=cache",
			expectedStatus: http.StatusOK,
			validateSchema: validatePaginatedListResponse,
		},
		{
			name:           "Get Audit Stats",
			method:         "GET",
			path:           "/api/v1/enterprise/audit/stats",
			expectedStatus: http.StatusOK,
			validateSchema: validateSuccessResponse,
		},
		{
			name:           "Get Compliance Report",
			method:         "GET",
			path:           "/api/v1/enterprise/audit/compliance/report",
			expectedStatus: http.StatusOK,
			validateSchema: validateSuccessResponse,
		},

		// Analytics Endpoints (10)
		{
			name:           "Get Analytics Overview",
			method:         "GET",
			path:           "/api/v1/enterprise/analytics/overview",
			expectedStatus: http.StatusOK,
			validateSchema: validateSuccessResponse,
		},
		{
			name:           "Get Usage Stats",
			method:         "GET",
			path:           "/api/v1/enterprise/analytics/usage",
			expectedStatus: http.StatusOK,
			validateSchema: validateSuccessResponse,
		},
		{
			name:           "Get Performance Metrics",
			method:         "GET",
			path:           "/api/v1/enterprise/analytics/performance",
			expectedStatus: http.StatusOK,
			validateSchema: validateSuccessResponse,
		},
		{
			name:           "Get Cache Efficiency",
			method:         "GET",
			path:           "/api/v1/enterprise/analytics/cache-efficiency",
			expectedStatus: http.StatusOK,
			validateSchema: validateSuccessResponse,
		},
		{
			name:           "List Reports",
			method:         "GET",
			path:           "/api/v1/enterprise/analytics/reports",
			expectedStatus: http.StatusOK,
			validateSchema: validatePaginatedListResponse,
		},
		{
			name:           "Get Report",
			method:         "GET",
			path:           "/api/v1/enterprise/analytics/reports/report_001",
			expectedStatus: http.StatusOK,
			validateSchema: validateSuccessResponse,
		},
		{
			name:           "Get Cost Analysis",
			method:         "GET",
			path:           "/api/v1/enterprise/analytics/costs",
			expectedStatus: http.StatusOK,
			validateSchema: validateSuccessResponse,
		},
		{
			name:           "Get Trends",
			method:         "GET",
			path:           "/api/v1/enterprise/analytics/trends",
			expectedStatus: http.StatusOK,
			validateSchema: validateSuccessResponse,
		},

		// Security Endpoints (8)
		{
			name:           "List Vulnerabilities",
			method:         "GET",
			path:           "/api/v1/enterprise/security/vulnerabilities",
			expectedStatus: http.StatusOK,
			validateSchema: validatePaginatedListResponse,
		},
		{
			name:           "Get Vulnerability",
			method:         "GET",
			path:           "/api/v1/enterprise/security/vulnerabilities/vuln_001",
			expectedStatus: http.StatusOK,
			validateSchema: validateSuccessResponse,
		},
		{
			name:           "Get Scan Job",
			method:         "GET",
			path:           "/api/v1/enterprise/security/scan/scan_001",
			expectedStatus: http.StatusOK,
			validateSchema: validateSuccessResponse,
		},
		{
			name:           "List Licenses",
			method:         "GET",
			path:           "/api/v1/enterprise/security/licenses",
			expectedStatus: http.StatusOK,
			validateSchema: validatePaginatedListResponse,
		},
		{
			name:           "List License Violations",
			method:         "GET",
			path:           "/api/v1/enterprise/security/licenses/violations",
			expectedStatus: http.StatusOK,
			validateSchema: validatePaginatedListResponse,
		},
		{
			name:           "List Malware Alerts",
			method:         "GET",
			path:           "/api/v1/enterprise/security/malware/alerts",
			expectedStatus: http.StatusOK,
			validateSchema: validatePaginatedListResponse,
		},
		{
			name:           "List Quarantined Packages",
			method:         "GET",
			path:           "/api/v1/enterprise/security/malware/quarantine",
			expectedStatus: http.StatusOK,
			validateSchema: validatePaginatedListResponse,
		},

		// Alerts Endpoints (5)
		{
			name:           "List Alerts",
			method:         "GET",
			path:           "/api/v1/enterprise/alerts",
			expectedStatus: http.StatusOK,
			validateSchema: validatePaginatedListResponse,
		},
		{
			name:           "Get Alert",
			method:         "GET",
			path:           "/api/v1/enterprise/alerts/alert_001",
			expectedStatus: http.StatusOK,
			validateSchema: validateSuccessResponse,
		},

		// License Endpoints (4)
		{
			name:           "Get License Info",
			method:         "GET",
			path:           "/api/v1/enterprise/license/info",
			expectedStatus: http.StatusOK,
			validateSchema: validateSuccessResponse,
		},
		{
			name:           "List License Features",
			method:         "GET",
			path:           "/api/v1/enterprise/license/features",
			expectedStatus: http.StatusOK,
			validateSchema: validateSuccessResponse,
		},
		{
			name:           "Get License Usage",
			method:         "GET",
			path:           "/api/v1/enterprise/license/usage",
			expectedStatus: http.StatusOK,
			validateSchema: validateSuccessResponse,
		},

		// Missing RBAC Endpoints (7)
		{
			name:           "Get User Events (Audit)",
			method:         "GET",
			path:           "/api/v1/enterprise/audit/users/user_001/events",
			expectedStatus: http.StatusOK,
			validateSchema: validatePaginatedListResponse,
		},
		{
			name:           "Get Resource Events (Audit)",
			method:         "GET",
			path:           "/api/v1/enterprise/audit/resources/role/role_admin",
			expectedStatus: http.StatusOK,
			validateSchema: validatePaginatedListResponse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			resp, err := app.Test(req, -1)

			require.NoError(t, err, "Failed to make request")
			assert.Equal(t, tt.expectedStatus, resp.StatusCode,
				"Expected status %d but got %d for %s %s",
				tt.expectedStatus, resp.StatusCode, tt.method, tt.path)

			// Validate response body schema
			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			require.NoError(t, err, "Failed to decode response body")

			// Apply specific schema validation
			tt.validateSchema(t, body)
		})
	}
}

// validateSuccessResponse validates standard success response schema
func validateSuccessResponse(t *testing.T, body map[string]interface{}) {
	assert.Contains(t, body, "success", "Response must contain 'success' field")
	assert.Contains(t, body, "data", "Response must contain 'data' field")
	assert.Contains(t, body, "metadata", "Response must contain 'metadata' field")

	success, ok := body["success"].(bool)
	assert.True(t, ok, "'success' field must be a boolean")
	assert.True(t, success, "'success' should be true for successful responses")

	// Validate metadata structure
	metadata, ok := body["metadata"].(map[string]interface{})
	assert.True(t, ok, "'metadata' must be an object")
	assert.Contains(t, metadata, "timestamp", "Metadata must contain 'timestamp'")
}

// validatePaginatedListResponse validates paginated list response schema
func validatePaginatedListResponse(t *testing.T, body map[string]interface{}) {
	validateSuccessResponse(t, body)

	assert.Contains(t, body, "pagination", "Paginated response must contain 'pagination' field")

	pagination, ok := body["pagination"].(map[string]interface{})
	assert.True(t, ok, "'pagination' must be an object")

	// Validate pagination fields
	assert.Contains(t, pagination, "page", "Pagination must contain 'page'")
	assert.Contains(t, pagination, "per_page", "Pagination must contain 'per_page'")
	assert.Contains(t, pagination, "total", "Pagination must contain 'total'")
	assert.Contains(t, pagination, "total_pages", "Pagination must contain 'total_pages'")
	assert.Contains(t, pagination, "has_next", "Pagination must contain 'has_next'")
	assert.Contains(t, pagination, "has_prev", "Pagination must contain 'has_prev'")

	// Validate data is an array
	data, ok := body["data"].([]interface{})
	assert.True(t, ok, "'data' must be an array for paginated responses, got %T", body["data"])

	// Validate pagination constraints
	perPage, _ := pagination["per_page"].(float64)
	assert.LessOrEqual(t, len(data), int(perPage),
		"Data length should not exceed per_page value")
}

// TestEnterpriseAPIPaginationBehavior validates pagination behavior
func TestEnterpriseAPIPaginationBehavior(t *testing.T) {
	app := fiber.New()
	routers.SetupEnterpriseRoutes(app, nil) // nil = dev mode, no license check

	endpoints := []string{
		"/api/v1/enterprise/rbac/roles",
		"/api/v1/enterprise/audit/events",
		"/api/v1/enterprise/analytics/reports",
		"/api/v1/enterprise/security/vulnerabilities",
		"/api/v1/enterprise/alerts",
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			// Test page=1&per_page=5
			req := httptest.NewRequest("GET", endpoint+"?page=1&per_page=5", nil)
			resp, err := app.Test(req, -1)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			require.NoError(t, err)

			pagination := body["pagination"].(map[string]interface{})
			assert.Equal(t, float64(1), pagination["page"])
			assert.Equal(t, float64(5), pagination["per_page"])

			// Test max per_page limit (should cap at 100)
			req = httptest.NewRequest("GET", endpoint+"?page=1&per_page=200", nil)
			resp, err = app.Test(req, -1)
			require.NoError(t, err)

			err = json.NewDecoder(resp.Body).Decode(&body)
			require.NoError(t, err)

			pagination = body["pagination"].(map[string]interface{})
			assert.LessOrEqual(t, pagination["per_page"].(float64), float64(100),
				"per_page should be capped at 100")
		})
	}
}

// TestEnterpriseAPIErrorFormat validates error response format
func TestEnterpriseAPIErrorFormat(t *testing.T) {
	app := fiber.New()
	routers.SetupEnterpriseRoutes(app, nil) // nil = dev mode, no license check

	// Test 404 error
	req := httptest.NewRequest("GET", "/api/v1/enterprise/rbac/roles/nonexistent_role", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	// Validate error response structure
	assert.Contains(t, body, "success", "Error response must contain 'success' field")
	success, _ := body["success"].(bool)
	assert.False(t, success, "'success' should be false for error responses")

	assert.Contains(t, body, "error", "Error response must contain 'error' field")
	errorObj, ok := body["error"].(map[string]interface{})
	assert.True(t, ok, "'error' must be an object")
	assert.Contains(t, errorObj, "code", "Error object must contain 'code'")
	assert.Contains(t, errorObj, "message", "Error object must contain 'message'")
}
// TestEnterpriseAPIMutationEndpoints validates POST/PUT/DELETE endpoints
func TestEnterpriseAPIMutationEndpoints(t *testing.T) {
	app := fiber.New()
	routers.SetupEnterpriseRoutes(app, nil) // nil = dev mode, no license check

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		expectedStatus int
	}{
		// RBAC Mutations
		{
			name:           "Create Role",
			method:         "POST",
			path:           "/api/v1/enterprise/rbac/roles",
			body:           `{"name":"test_role","description":"Test role","permissions":["read:packages"]}`,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Update Role",
			method:         "PUT",
			path:           "/api/v1/enterprise/rbac/roles/role_admin",
			body:           `{"name":"updated_role","permissions":["read:packages","write:packages"]}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Delete Role",
			method:         "DELETE",
			path:           "/api/v1/enterprise/rbac/roles/role_test",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Assign Permissions to Role",
			method:         "POST",
			path:           "/api/v1/enterprise/rbac/roles/role_admin/permissions",
			body:           `{"permissions":["read:analytics","write:analytics"]}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Revoke Permission from Role",
			method:         "DELETE",
			path:           "/api/v1/enterprise/rbac/roles/role_admin/permissions/write:packages",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Assign Role to User",
			method:         "POST",
			path:           "/api/v1/enterprise/rbac/users/user_001/roles",
			body:           `{"role_id":"role_admin"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Remove Role from User",
			method:         "DELETE",
			path:           "/api/v1/enterprise/rbac/users/user_001/roles/role_viewer",
			expectedStatus: http.StatusOK,
		},

		// Audit Mutations
		{
			name:           "Export Audit Events",
			method:         "POST",
			path:           "/api/v1/enterprise/audit/export",
			body:           `{"format":"json","start_date":"2025-01-01T00:00:00Z","end_date":"2025-01-31T23:59:59Z"}`,
			expectedStatus: http.StatusOK,
		},

		// Analytics Mutations
		{
			name:           "Create Custom Report",
			method:         "POST",
			path:           "/api/v1/enterprise/analytics/reports",
			body:           `{"name":"Monthly Usage","type":"usage","parameters":{"period":"30d"}}`,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Delete Report",
			method:         "DELETE",
			path:           "/api/v1/enterprise/analytics/reports/report_001",
			expectedStatus: http.StatusOK,
		},

		// Security Mutations
		{
			name:           "Trigger Security Scan",
			method:         "POST",
			path:           "/api/v1/enterprise/security/scan",
			body:           `{"scope":"all","package_manager":"npm"}`,
			expectedStatus: http.StatusAccepted,
		},

		// Alerts Mutations
		{
			name:           "Create Alert Rule",
			method:         "POST",
			path:           "/api/v1/enterprise/alerts/rules",
			body:           `{"name":"High CPU Alert","type":"threshold","severity":"high","conditions":{"metric":"cpu","threshold":80},"actions":[{"type":"email","config":{"recipients":["ops@example.com"]}}]}`,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Update Alert Rule",
			method:         "PUT",
			path:           "/api/v1/enterprise/alerts/rules/rule_001",
			body:           `{"name":"Updated Alert","enabled":true}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Delete Alert Rule",
			method:         "DELETE",
			path:           "/api/v1/enterprise/alerts/rules/rule_001",
			expectedStatus: http.StatusOK,
		},

		// License Mutations
		{
			name:           "Validate License",
			method:         "POST",
			path:           "/api/v1/enterprise/license/validate",
			body:           `{"license_key":"test-license-key-12345"}`,
			expectedStatus: http.StatusOK,
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

			resp, err := app.Test(req, -1)
			require.NoError(t, err, "Failed to make request")
			
			assert.Equal(t, tt.expectedStatus, resp.StatusCode,
				"Expected status %d but got %d for %s %s",
				tt.expectedStatus, resp.StatusCode, tt.method, tt.path)

			// Validate response has proper structure
			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			require.NoError(t, err, "Failed to decode response body")
			
			// All responses should have success and metadata fields
			assert.Contains(t, body, "success", "Response must contain 'success' field")
			assert.Contains(t, body, "metadata", "Response must contain 'metadata' field")
		})
	}
}
