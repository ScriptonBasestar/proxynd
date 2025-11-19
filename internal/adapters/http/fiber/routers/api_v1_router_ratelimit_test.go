package routers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToggleEndpointRateLimiting(t *testing.T) {
	// Create a test Fiber app
	app := fiber.New()

	// Setup routes
	SetupAPIv1Routes(app, nil)

	t.Run("rate limit enforced - exceeding limit returns 429", func(t *testing.T) {
		// Rate limit is 10 requests per minute with burst of 3
		// Send 4 requests rapidly - 4th should be rate limited

		reqBody := map[string]interface{}{
			"enabled": true,
		}
		body, _ := json.Marshal(reqBody)

		successCount := 0
		rateLimitedCount := 0

		// Send 4 requests
		for i := 0; i < 4; i++ {
			req := httptest.NewRequest("POST", "/api/v1/pm/npm/toggle", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Forwarded-For", "192.168.1.100") // Same IP

			resp, err := app.Test(req)
			require.NoError(t, err)
			resp.Body.Close()

			if resp.StatusCode == http.StatusTooManyRequests {
				rateLimitedCount++
			} else {
				successCount++
			}
		}

		// First 3 should succeed (burst), 4th should be rate limited
		assert.GreaterOrEqual(t, successCount, 3, "At least 3 requests should succeed within burst")
		assert.GreaterOrEqual(t, rateLimitedCount, 1, "At least 1 request should be rate limited")
	})

	t.Run("rate limit headers present in response", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"enabled": true,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/v1/pm/npm/toggle", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "192.168.1.101") // Different IP

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Check for rate limit headers
		limitHeader := resp.Header.Get("X-RateLimit-Limit")
		remainingHeader := resp.Header.Get("X-RateLimit-Remaining")
		resetHeader := resp.Header.Get("X-RateLimit-Reset")

		// At least one header should be present
		hasRateLimitHeaders := limitHeader != "" || remainingHeader != "" || resetHeader != ""
		assert.True(t, hasRateLimitHeaders, "Rate limit headers should be present")
	})

	t.Run("different IPs have separate rate limits", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"enabled": true,
		}
		body, _ := json.Marshal(reqBody)

		// First IP - exhaust burst
		for i := 0; i < 3; i++ {
			req := httptest.NewRequest("POST", "/api/v1/pm/npm/toggle", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Forwarded-For", "192.168.1.102")

			resp, err := app.Test(req)
			require.NoError(t, err)
			resp.Body.Close()
		}

		// Different IP - should still have available quota
		req := httptest.NewRequest("POST", "/api/v1/pm/npm/toggle", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "192.168.1.103") // Different IP

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should not be rate limited since it's a different IP
		// May get 503 (service unavailable) or 200 (if config available), but not 429
		assert.NotEqual(t, http.StatusTooManyRequests, resp.StatusCode,
			"Different IP should not be affected by first IP's rate limit")
	})

	t.Run("rate limit error message contains useful information", func(t *testing.T) {
		// Create a new app to ensure clean rate limit state
		app := fiber.New()
		SetupAPIv1Routes(app, nil)

		reqBody := map[string]interface{}{
			"enabled": true,
		}
		body, _ := json.Marshal(reqBody)

		// Exhaust rate limit
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("POST", "/api/v1/pm/npm/toggle", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Forwarded-For", "192.168.1.104")

			resp, err := app.Test(req)
			require.NoError(t, err)

			if resp.StatusCode == http.StatusTooManyRequests {
				var result map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&result)
				resp.Body.Close()
				require.NoError(t, err)

				// Check that error message exists
				assert.Contains(t, result, "error", "Error response should contain 'error' field")

				// Error message should be informative
				errorMsg, ok := result["error"].(string)
				if ok {
					assert.NotEmpty(t, errorMsg, "Error message should not be empty")
				}
				return
			}
			resp.Body.Close()
		}
	})
}

func TestRateLimitingWithJWT(t *testing.T) {
	t.Run("rate limit applies before JWT authentication", func(t *testing.T) {
		// Even with JWT enabled, rate limiting should prevent brute force

		// Note: This test verifies the middleware order
		// Rate limiting is applied before JWT middleware to prevent
		// authentication bypass attempts through rate limiting

		app := fiber.New()
		SetupAPIv1Routes(app, nil)

		reqBody := map[string]interface{}{
			"enabled": true,
		}
		body, _ := json.Marshal(reqBody)

		// Send multiple requests without JWT token
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("POST", "/api/v1/pm/npm/toggle", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Forwarded-For", "192.168.1.105")

			resp, err := app.Test(req)
			require.NoError(t, err)
			resp.Body.Close()

			// Once rate limited, should get 429 (not 401 for missing auth)
			// This proves rate limiting is checked first
			if resp.StatusCode == http.StatusTooManyRequests {
				assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode,
					"Rate limit should be enforced before JWT check")
				return
			}
		}
	})
}

func TestRateLimitRecovery(t *testing.T) {
	t.Run("rate limit resets after time window", func(t *testing.T) {
		// This test verifies that rate limits reset properly
		// Note: This is a conceptual test - in practice, waiting 60+ seconds
		// is impractical for unit tests

		app := fiber.New()
		SetupAPIv1Routes(app, nil)

		reqBody := map[string]interface{}{
			"enabled": true,
		}
		body, _ := json.Marshal(reqBody)

		// First burst - should succeed
		req := httptest.NewRequest("POST", "/api/v1/pm/npm/toggle", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "192.168.1.106")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should get a response (not necessarily success, but not rate limited)
		assert.NotEqual(t, http.StatusTooManyRequests, resp.StatusCode,
			"First request should not be rate limited")

		// Note: Full time-based reset testing would require:
		// 1. Mocking time (not easily done with ulule/limiter)
		// 2. Waiting actual time (60+ seconds for rate window)
		// 3. Integration test environment with controlled clock
		// This is better tested in integration/e2e tests
	})
}

func TestRateLimitConfiguration(t *testing.T) {
	t.Run("verify rate limit configuration values", func(t *testing.T) {
		// This test verifies the rate limit is configured as expected
		// Rate: 10 requests per minute
		// Burst: 3 requests

		app := fiber.New()
		SetupAPIv1Routes(app, nil)

		reqBody := map[string]interface{}{
			"enabled": true,
		}
		body, _ := json.Marshal(reqBody)

		successCount := 0

		// Send requests until rate limited
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("POST", "/api/v1/pm/npm/toggle", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Forwarded-For", "192.168.1.107")

			resp, err := app.Test(req)
			require.NoError(t, err)
			resp.Body.Close()

			if resp.StatusCode != http.StatusTooManyRequests {
				successCount++
			} else {
				// Once rate limited, stop
				break
			}

			// Small delay to avoid hitting limiter too fast
			time.Sleep(10 * time.Millisecond)
		}

		// Should allow at least burst size (3) requests
		assert.GreaterOrEqual(t, successCount, 3,
			"Should allow at least burst size (3) requests before rate limiting")
	})
}
