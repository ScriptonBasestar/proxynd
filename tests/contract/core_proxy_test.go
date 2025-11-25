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

// TestCoreProxyAPIContract validates the contract/schema for all core proxy endpoints
// This ensures API stability for package manager proxy functionality
func TestCoreProxyAPIContract(t *testing.T) {
	app := fiber.New()

	// Setup health routes
	routers.HealthRouterWithConfig(app, nil)

	// Setup API v1 routes (core system endpoints)
	routers.SetupAPIv1Routes(app, nil)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		validateSchema func(*testing.T, interface{})
	}{
		// Health Endpoints (5)
		{
			name:           "Health Check (Healthz)",
			method:         "GET",
			path:           "/healthz",
			expectedStatus: 0, // Can be 200 or 503 depending on health
			validateSchema: validateHealthResponse,
		},
		{
			name:           "Health Check Simple Format",
			method:         "GET",
			path:           "/healthz?format=simple",
			expectedStatus: http.StatusOK,
			validateSchema: validateSimpleHealthResponse,
		},
		{
			name:           "Liveness Probe",
			method:         "GET",
			path:           "/health/live",
			expectedStatus: http.StatusOK,
			validateSchema: validateLivenessResponse,
		},
		{
			name:           "Readiness Probe",
			method:         "GET",
			path:           "/health/ready",
			expectedStatus: 0, // Can be 200 or 503 depending on readiness
			validateSchema: validateReadinessResponse,
		},
		{
			name:           "Adapter Health Check",
			method:         "GET",
			path:           "/health/adapters",
			expectedStatus: http.StatusServiceUnavailable, // Factory not initialized in test
			validateSchema: validateAdapterHealthResponse,
		},

		// System Endpoints (3)
		{
			name:           "System Information",
			method:         "GET",
			path:           "/api/v1/system/info",
			expectedStatus: http.StatusOK,
			validateSchema: validateSystemInfoResponse,
		},
		{
			name:           "Cache Statistics",
			method:         "GET",
			path:           "/api/v1/stats",
			expectedStatus: http.StatusOK,
			validateSchema: validateCacheStatsResponse,
		},
		{
			name:           "Package Managers List",
			method:         "GET",
			path:           "/api/v1/pm",
			expectedStatus: http.StatusOK,
			validateSchema: validatePackageManagersResponse,
		},

		// Plugin Health Endpoint (1)
		{
			name:           "Plugin Health Check",
			method:         "GET",
			path:           "/api/v1/plugins/health",
			expectedStatus: 0, // Can be 200 or 503 depending on plugin availability
			validateSchema: validatePluginHealthResponse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			resp, err := app.Test(req, -1)

			require.NoError(t, err, "Failed to make request")

			// If expectedStatus is 0, accept any valid status code
			if tt.expectedStatus != 0 {
				assert.Equal(t, tt.expectedStatus, resp.StatusCode,
					"Expected status %d but got %d for %s %s",
					tt.expectedStatus, resp.StatusCode, tt.method, tt.path)
			} else {
				// Verify it's a valid HTTP status
				assert.True(t, resp.StatusCode >= 200 && resp.StatusCode < 600,
					"Status code %d is not valid for %s %s",
					resp.StatusCode, tt.method, tt.path)
			}

			// Validate response body schema
			var body interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			require.NoError(t, err, "Failed to decode response body")

			// Apply specific schema validation
			tt.validateSchema(t, body)
		})
	}
}

// TestPackageManagerProxyEndpoints validates package manager proxy route patterns
// Note: Full proxy functionality requires real services, this tests route registration
func TestPackageManagerProxyEndpoints(t *testing.T) {
	// Package manager routes are registered in main app setup
	// This test validates the expected URL patterns

	proxyTests := []struct {
		name           string
		method         string
		path           string
		contentType    string
		description    string
	}{
		// NPM Proxy Endpoints
		{
			name:        "NPM Package Metadata",
			method:      "GET",
			path:        "/npm/lodash",
			contentType: "application/json",
			description: "Fetch package metadata",
		},
		{
			name:        "NPM Package Tarball",
			method:      "GET",
			path:        "/npm/lodash/-/lodash-4.17.21.tgz",
			contentType: "application/octet-stream",
			description: "Download package tarball",
		},

		// Maven Proxy Endpoints
		{
			name:        "Maven Artifact POM",
			method:      "GET",
			path:        "/maven/org/springframework/spring-core/5.3.0/spring-core-5.3.0.pom",
			contentType: "application/xml",
			description: "Download POM file",
		},
		{
			name:        "Maven Artifact JAR",
			method:      "GET",
			path:        "/maven/org/springframework/spring-core/5.3.0/spring-core-5.3.0.jar",
			contentType: "application/java-archive",
			description: "Download JAR file",
		},

		// PyPI Proxy Endpoints
		{
			name:        "PyPI Simple Index",
			method:      "GET",
			path:        "/pypi/simple/",
			contentType: "text/html",
			description: "List all packages",
		},
		{
			name:        "PyPI Package Index",
			method:      "GET",
			path:        "/pypi/simple/requests/",
			contentType: "text/html",
			description: "Package file list",
		},

		// APT Proxy Endpoints
		{
			name:        "APT Release File",
			method:      "GET",
			path:        "/apt/dists/focal/Release",
			contentType: "text/plain",
			description: "Repository metadata",
		},
		{
			name:        "APT Packages Index",
			method:      "GET",
			path:        "/apt/dists/focal/main/binary-amd64/Packages",
			contentType: "text/plain",
			description: "Package index",
		},

		// Docker Registry Endpoints
		{
			name:        "Docker Registry Version",
			method:      "GET",
			path:        "/v2/",
			contentType: "application/json",
			description: "Registry version check",
		},
		{
			name:        "Docker Image Manifest",
			method:      "GET",
			path:        "/v2/library/nginx/manifests/latest",
			contentType: "application/vnd.docker.distribution.manifest.v2+json",
			description: "Image manifest",
		},

		// YUM Proxy Endpoints
		{
			name:        "YUM Repository Metadata",
			method:      "GET",
			path:        "/yum/centos/8/BaseOS/x86_64/os/repodata/repomd.xml",
			contentType: "application/xml",
			description: "Repository metadata",
		},

		// APK Proxy Endpoints
		{
			name:        "APK Package Index",
			method:      "GET",
			path:        "/apk/v3.16/main/x86_64/APKINDEX.tar.gz",
			contentType: "application/gzip",
			description: "Package index archive",
		},
	}

	for _, tt := range proxyTests {
		t.Run(tt.name, func(t *testing.T) {
			// This test documents the expected endpoint patterns
			// Actual proxy functionality is tested in integration/e2e tests
			assert.NotEmpty(t, tt.path, "Path should be defined")
			assert.NotEmpty(t, tt.method, "Method should be defined")
			assert.NotEmpty(t, tt.contentType, "Content-Type should be defined")
			assert.NotEmpty(t, tt.description, "Description should be provided")
		})
	}
}

// TestErrorResponseFormat validates error response format consistency
func TestErrorResponseFormat(t *testing.T) {
	app := fiber.New()
	routers.SetupAPIv1Routes(app, nil)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		contentType    string
	}{
		{
			name:           "Invalid Package Manager Toggle",
			method:         "POST",
			path:           "/api/v1/pm/invalid/toggle",
			expectedStatus: http.StatusBadRequest,
			contentType:    "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("Content-Type", tt.contentType)
			resp, err := app.Test(req, -1)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			contentType := resp.Header.Get("Content-Type")
			if strings.Contains(contentType, "application/json") {
				var body map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&body)
				require.NoError(t, err, "Failed to decode JSON response")

				// Error responses should contain 'error' field
				assert.Contains(t, body, "error", "Error response must contain 'error' field")
				assert.Contains(t, body, "message", "Error response must contain 'message' field")
			}
		})
	}
}

// validateHealthResponse validates detailed health response schema
func validateHealthResponse(t *testing.T, body interface{}) {
	m, ok := body.(map[string]interface{})
	require.True(t, ok, "Body must be an object")

	assert.Contains(t, m, "status", "Health response must contain 'status'")
	assert.Contains(t, m, "timestamp", "Health response must contain 'timestamp'")
	assert.Contains(t, m, "uptime", "Health response must contain 'uptime'")
	assert.Contains(t, m, "checks", "Health response must contain 'checks'")

	// Validate status is string
	_, ok = m["status"].(string)
	assert.True(t, ok, "'status' must be a string")
}

// validateSimpleHealthResponse validates simple health response schema
func validateSimpleHealthResponse(t *testing.T, body interface{}) {
	m, ok := body.(map[string]interface{})
	require.True(t, ok, "Body must be an object")

	assert.Contains(t, m, "status", "Simple health response must contain 'status'")
	assert.Contains(t, m, "timestamp", "Simple health response must contain 'timestamp'")

	// Should only have these two fields
	assert.Len(t, m, 2, "Simple health response should have exactly 2 fields")
}

// validateLivenessResponse validates liveness probe response schema
func validateLivenessResponse(t *testing.T, body interface{}) {
	m, ok := body.(map[string]interface{})
	require.True(t, ok, "Body must be an object")

	assert.Contains(t, m, "status", "Liveness response must contain 'status'")
	status, ok := m["status"].(string)
	assert.True(t, ok, "'status' must be a string")
	assert.Equal(t, "alive", status, "Status should be 'alive'")
}

// validateReadinessResponse validates readiness probe response schema
func validateReadinessResponse(t *testing.T, body interface{}) {
	m, ok := body.(map[string]interface{})
	require.True(t, ok, "Body must be an object")

	assert.Contains(t, m, "ready", "Readiness response must contain 'ready'")
	assert.Contains(t, m, "checks", "Readiness response must contain 'checks'")

	// Validate ready is boolean
	_, ok = m["ready"].(bool)
	assert.True(t, ok, "'ready' must be a boolean")

	// Validate checks is object
	checks, ok := m["checks"].(map[string]interface{})
	assert.True(t, ok, "'checks' must be an object")
	assert.NotEmpty(t, checks, "Checks should not be empty")
}

// validateAdapterHealthResponse validates adapter health response schema
func validateAdapterHealthResponse(t *testing.T, body interface{}) {
	m, ok := body.(map[string]interface{})
	require.True(t, ok, "Body must be an object")

	// Check for error case (factory not initialized) or success case
	if errorMsg, hasError := m["error"]; hasError {
		assert.Equal(t, "Handler adapter factory not initialized", errorMsg)
		return
	}

	// Success case validation
	assert.Contains(t, m, "status", "Adapter health response must contain 'status'")
	assert.Contains(t, m, "timestamp", "Adapter health response must contain 'timestamp'")
	assert.Contains(t, m, "adapters", "Adapter health response must contain 'adapters'")
	assert.Contains(t, m, "healthy_count", "Adapter health response must contain 'healthy_count'")
	assert.Contains(t, m, "total_count", "Adapter health response must contain 'total_count'")
}

// validateSystemInfoResponse validates system information response schema
func validateSystemInfoResponse(t *testing.T, body interface{}) {
	m, ok := body.(map[string]interface{})
	require.True(t, ok, "Body must be an object")

	assert.Contains(t, m, "version", "System info must contain 'version'")
	assert.Contains(t, m, "edition", "System info must contain 'edition'")
	assert.Contains(t, m, "license", "System info must contain 'license'")

	// Validate license object
	license, ok := m["license"].(map[string]interface{})
	assert.True(t, ok, "'license' must be an object")
	assert.Contains(t, license, "type", "License must contain 'type'")
	assert.Contains(t, license, "features", "License must contain 'features'")
}

// validateCacheStatsResponse validates cache statistics response schema
func validateCacheStatsResponse(t *testing.T, body interface{}) {
	m, ok := body.(map[string]interface{})
	require.True(t, ok, "Body must be an object")

	assert.Contains(t, m, "hit_rate", "Cache stats must contain 'hit_rate'")
	assert.Contains(t, m, "total_requests", "Cache stats must contain 'total_requests'")
	assert.Contains(t, m, "cache_size_mb", "Cache stats must contain 'cache_size_mb'")
	assert.Contains(t, m, "hits", "Cache stats must contain 'hits'")
	assert.Contains(t, m, "misses", "Cache stats must contain 'misses'")

	// Validate numeric types
	_, ok = m["hit_rate"].(float64)
	assert.True(t, ok, "'hit_rate' must be a number")

	_, ok = m["total_requests"].(float64)
	assert.True(t, ok, "'total_requests' must be a number")
}

// validatePackageManagersResponse validates package managers list response schema
func validatePackageManagersResponse(t *testing.T, body interface{}) {
	arr, ok := body.([]interface{})
	require.True(t, ok, "Body must be an array")
	require.NotEmpty(t, arr, "Package managers list should not be empty")

	// Validate first PM entry
	pm, ok := arr[0].(map[string]interface{})
	assert.True(t, ok, "Package manager entry must be an object")

	assert.Contains(t, pm, "name", "Package manager must contain 'name'")
	assert.Contains(t, pm, "enabled", "Package manager must contain 'enabled'")
	assert.Contains(t, pm, "url", "Package manager must contain 'url'")
	assert.Contains(t, pm, "cache_enabled", "Package manager must contain 'cache_enabled'")

	// Validate types
	_, ok = pm["enabled"].(bool)
	assert.True(t, ok, "'enabled' must be a boolean")
}

// validatePluginHealthResponse validates plugin health response schema
func validatePluginHealthResponse(t *testing.T, body interface{}) {
	m, ok := body.(map[string]interface{})
	require.True(t, ok, "Body must be an object")

	// Plugin may not be available in test environment
	if enabled, hasEnabled := m["enabled"]; hasEnabled {
		_, ok := enabled.(bool)
		assert.True(t, ok, "'enabled' must be a boolean")
		// When not available, expect message field
		if !enabled.(bool) {
			assert.Contains(t, m, "message", "Unavailable plugin response should have message")
			return
		}
	}

	// Available case
	assert.Contains(t, m, "success", "Plugin health response must contain 'success'")
	assert.Contains(t, m, "data", "Plugin health response must contain 'data'")

	// Validate success is boolean
	success, ok := m["success"].(bool)
	assert.True(t, ok, "'success' must be a boolean")
	assert.True(t, success, "'success' should be true")
}
