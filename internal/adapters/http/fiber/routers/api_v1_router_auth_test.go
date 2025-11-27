package routers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/internal/adapters/http/fiber/middleware"
)

func TestToggleEndpointAuthentication(t *testing.T) {
	// Create a test Fiber app
	app := fiber.New()

	// Setup routes with mock config
	SetupAPIv1Routes(app, nil)

	t.Run("without JWT secret - allows anonymous access", func(t *testing.T) {
		// Ensure JWT_SECRET is not set
		_ = os.Unsetenv("JWT_SECRET")

		reqBody := map[string]interface{}{
			"enabled": true,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/v1/pm/npm/toggle", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return 503 (service unavailable) since configService is not available
		// But authentication should not block the request
		assert.Contains(t, []int{http.StatusServiceUnavailable, http.StatusOK}, resp.StatusCode)
	})

	t.Run("with JWT secret - requires authentication", func(t *testing.T) {
		// Set JWT_SECRET to enforce authentication
		_ = os.Setenv("JWT_SECRET", "test-secret-key-for-testing")
		defer func() { _ = os.Unsetenv("JWT_SECRET") }()

		// Recreate app with JWT enabled
		app := fiber.New()
		SetupAPIv1Routes(app, nil)

		reqBody := map[string]interface{}{
			"enabled": true,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/v1/pm/npm/toggle", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return 401 (unauthorized) without valid JWT token
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Contains(t, result, "error")
		assert.Contains(t, result["error"], "authorization")
	})

	t.Run("with valid JWT but no admin role - forbidden", func(t *testing.T) {
		// Set JWT_SECRET
		_ = os.Setenv("JWT_SECRET", "test-secret-key-for-testing")
		defer func() { _ = os.Unsetenv("JWT_SECRET") }()

		// Recreate app
		app := fiber.New()
		SetupAPIv1Routes(app, nil)

		// Generate a valid JWT token with user role (not admin)
		jwtConfig := middlewares.JWTConfig{
			SecretKey:     "test-secret-key-for-testing",
			TokenDuration: 0, // Will use default
			Issuer:        "proxynd",
		}

		token, err := middlewares.GenerateJWTToken("user123", "testuser", []string{"user"}, jwtConfig)
		require.NoError(t, err)

		reqBody := map[string]interface{}{
			"enabled": true,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/v1/pm/npm/toggle", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return 403 (forbidden) without admin role
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Contains(t, result, "error")
	})

	t.Run("with valid JWT and admin role - allowed", func(t *testing.T) {
		// Set JWT_SECRET
		_ = os.Setenv("JWT_SECRET", "test-secret-key-for-testing")
		defer func() { _ = os.Unsetenv("JWT_SECRET") }()

		// Recreate app
		app := fiber.New()
		SetupAPIv1Routes(app, nil)

		// Generate a valid JWT token with admin role
		jwtConfig := middlewares.JWTConfig{
			SecretKey:     "test-secret-key-for-testing",
			TokenDuration: 0,
			Issuer:        "proxynd",
		}

		token, err := middlewares.GenerateJWTToken("admin123", "admin", []string{"admin"}, jwtConfig)
		require.NoError(t, err)

		reqBody := map[string]interface{}{
			"enabled": true,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/v1/pm/npm/toggle", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Should pass authentication but fail at configService
		// (503 because configService is not available in test)
		assert.Contains(t, []int{http.StatusServiceUnavailable, http.StatusOK}, resp.StatusCode)
	})
}

func TestInvalidPackageManagerAuditLog(t *testing.T) {
	app := fiber.New()
	SetupAPIv1Routes(app, nil)

	// Test that invalid PM attempt triggers audit log (even without audit service)
	reqBody := map[string]interface{}{
		"enabled": true,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/pm/invalid-pm/toggle", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should return 400 (bad request)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "invalid_package_manager", result["error"])
	assert.Contains(t, result, "valid")
}
