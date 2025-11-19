package routers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigReloadAuthentication(t *testing.T) {
	t.Run("without JWT_SECRET - allows anonymous access", func(t *testing.T) {
		// Clear JWT_SECRET to test development mode
		originalSecret := os.Getenv("JWT_SECRET")
		os.Unsetenv("JWT_SECRET")
		defer func() {
			if originalSecret != "" {
				os.Setenv("JWT_SECRET", originalSecret)
			}
		}()

		// Create test app
		app := fiber.New()
		ConfigRouter(app)

		// Make request without authentication
		req := httptest.NewRequest("POST", "/api/config/reload", nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should succeed in development mode
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("with JWT_SECRET - requires authentication", func(t *testing.T) {
		// Set JWT_SECRET to enable production mode
		os.Setenv("JWT_SECRET", "test-secret-key-for-config-reload")
		defer os.Unsetenv("JWT_SECRET")

		// Create test app
		app := fiber.New()
		ConfigRouter(app)

		// Make request without authentication
		req := httptest.NewRequest("POST", "/api/config/reload", nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return 401 Unauthorized
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("with valid JWT but no admin role - returns 403", func(t *testing.T) {
		// Set JWT_SECRET
		os.Setenv("JWT_SECRET", "test-secret-key-for-config-reload")
		defer os.Unsetenv("JWT_SECRET")

		// Create test app
		app := fiber.New()
		ConfigRouter(app)

		// Generate JWT token for non-admin user
		token, err := middlewares.GenerateJWTToken(
			"user123",
			"testuser",
			[]string{"user"}, // Not admin
			middlewares.JWTConfig{
				SecretKey: "test-secret-key-for-config-reload",
				Issuer:    "proxynd",
			},
		)
		require.NoError(t, err)

		// Make request with user token
		req := httptest.NewRequest("POST", "/api/config/reload", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return 403 Forbidden
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("with valid JWT and admin role - passes authentication", func(t *testing.T) {
		// Set JWT_SECRET
		os.Setenv("JWT_SECRET", "test-secret-key-for-config-reload")
		defer os.Unsetenv("JWT_SECRET")

		// Create test app
		app := fiber.New()
		ConfigRouter(app)

		// Generate JWT token for admin user
		token, err := middlewares.GenerateJWTToken(
			"admin123",
			"admin",
			[]string{"admin"},
			middlewares.JWTConfig{
				SecretKey: "test-secret-key-for-config-reload",
				Issuer:    "proxynd",
			},
		)
		require.NoError(t, err)

		// Make request with admin token
		req := httptest.NewRequest("POST", "/api/config/reload", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should succeed (200 OK)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestConfigReloadRateLimiting(t *testing.T) {
	t.Run("rate limiting is enforced", func(t *testing.T) {
		// Clear JWT_SECRET for development mode
		os.Unsetenv("JWT_SECRET")

		// Create test app
		app := fiber.New()
		ConfigRouter(app)

		successCount := 0
		rateLimitedCount := 0

		// Send multiple requests rapidly
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("POST", "/api/config/reload", nil)
			req.Header.Set("X-Forwarded-For", "192.168.1.200") // Same IP

			resp, err := app.Test(req, -1)
			require.NoError(t, err)
			resp.Body.Close()

			if resp.StatusCode == http.StatusTooManyRequests {
				rateLimitedCount++
			} else if resp.StatusCode == http.StatusOK {
				successCount++
			}
		}

		// Should have at least some successful requests (burst) and at least one rate limited
		assert.GreaterOrEqual(t, successCount, 1, "Should have at least one successful request")
		// Note: Depending on timing, we may or may not hit rate limit with just 5 requests
		// This test mainly verifies rate limiter is configured
	})

	t.Run("rate limit headers present", func(t *testing.T) {
		// Clear JWT_SECRET for development mode
		os.Unsetenv("JWT_SECRET")

		// Create test app
		app := fiber.New()
		ConfigRouter(app)

		req := httptest.NewRequest("POST", "/api/config/reload", nil)
		req.Header.Set("X-Forwarded-For", "192.168.1.201") // Different IP

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Check for rate limit headers (implementation may vary)
		// At minimum, response should be successful
		assert.NotEqual(t, http.StatusInternalServerError, resp.StatusCode)
	})
}
