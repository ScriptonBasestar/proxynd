package enterprise

import (
	"fmt"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	adminHeaderValue = "true"
)

// TestNewRateLimiter tests rate limiter creation with default config
func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{})

	assert.NotNil(t, rl, "RateLimiter should not be nil")
	assert.Equal(t, 100, rl.config.Max, "Default max should be 100")
	assert.Equal(t, 1*time.Minute, rl.config.Window, "Default window should be 1 minute")
	assert.Equal(t, 10, rl.config.Burst, "Default burst should be 10")
	assert.NotNil(t, rl.config.KeyGenerator, "KeyGenerator should not be nil")
	assert.NotNil(t, rl.config.LimitReached, "LimitReached handler should not be nil")
}

// TestRateLimiter_WithinLimit tests requests within rate limit
func TestRateLimiter_WithinLimit(t *testing.T) {
	app := fiber.New()
	rl := NewRateLimiter(RateLimitConfig{
		Max:    5,
		Window: 1 * time.Minute,
		Burst:  2,
	})

	app.Use(rl.Middleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"success": true})
	})

	// Send 5 requests (within limit)
	for i := 1; i <= 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 200, resp.StatusCode, "Request %d should succeed", i)
		assert.Equal(t, "5", resp.Header.Get("X-RateLimit-Limit"), "Limit header should be 5")
		assert.Equal(t, fmt.Sprintf("%d", 5-i), resp.Header.Get("X-RateLimit-Remaining"), "Remaining should be %d", 5-i)
	}
}

// TestRateLimiter_ExceedsLimit tests rate limit exceeded
func TestRateLimiter_ExceedsLimit(t *testing.T) {
	app := fiber.New()
	rl := NewRateLimiter(RateLimitConfig{
		Max:    3,
		Window: 1 * time.Minute,
		Burst:  1,
	})

	app.Use(rl.Middleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"success": true})
	})

	// Send requests up to limit + burst
	for i := 1; i <= 4; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, 200, resp.StatusCode, "Request %d should succeed", i)
	}

	// This request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, 429, resp.StatusCode, "Request should be rate limited")
	assert.NotEmpty(t, resp.Header.Get("Retry-After"), "Retry-After header should be set")

	// Verify error response
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "RATE_LIMIT_EXCEEDED", "Response should contain rate limit error")
}

// TestRateLimiter_BurstAllowance tests burst capacity
func TestRateLimiter_BurstAllowance(t *testing.T) {
	app := fiber.New()
	rl := NewRateLimiter(RateLimitConfig{
		Max:    5,
		Window: 1 * time.Minute,
		Burst:  3, // Allow 3 extra burst requests
	})

	app.Use(rl.Middleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"success": true})
	})

	// Send max + burst requests
	totalAllowed := 5 + 3 // max + burst
	for i := 1; i <= totalAllowed; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode, "Request %d should succeed (within burst)", i)
	}

	// Next request should fail
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 429, resp.StatusCode, "Request beyond burst should fail")
}

// TestRateLimiter_Skip tests conditional skipping
func TestRateLimiter_Skip(t *testing.T) {
	app := fiber.New()
	rl := NewRateLimiter(RateLimitConfig{
		Max:    2,
		Window: 1 * time.Minute,
		Burst:  0,
		Skip: func(c *fiber.Ctx) bool {
			// Skip rate limiting for admin users
			return c.Get("X-Admin") == adminHeaderValue
		},
	})

	app.Use(rl.Middleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"success": true})
	})

	// Admin user should bypass rate limiting
	for i := 1; i <= 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Admin", adminHeaderValue)
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode, "Admin request %d should succeed", i)
	}
}

// TestDefaultEnterpriseRateLimiter tests default enterprise rate limiter
func TestDefaultEnterpriseRateLimiter(t *testing.T) {
	rl := DefaultEnterpriseRateLimiter()

	assert.NotNil(t, rl)
	assert.Equal(t, 1000, rl.config.Max, "Default enterprise max should be 1000")
	assert.Equal(t, 1*time.Minute, rl.config.Window)
	assert.Equal(t, 100, rl.config.Burst, "Default enterprise burst should be 100")
}

// TestStrictEnterpriseRateLimiter tests strict enterprise rate limiter
func TestStrictEnterpriseRateLimiter(t *testing.T) {
	rl := StrictEnterpriseRateLimiter()

	assert.NotNil(t, rl)
	assert.Equal(t, 100, rl.config.Max, "Strict max should be 100")
	assert.Equal(t, 1*time.Minute, rl.config.Window)
	assert.Equal(t, 10, rl.config.Burst, "Strict burst should be 10")
}

// TestPerIPRateLimiter tests PerIPRateLimiter creation
func TestPerIPRateLimiter(t *testing.T) {
	rl := PerIPRateLimiter(50, 1*time.Minute)

	assert.NotNil(t, rl)
	assert.Equal(t, 50, rl.config.Max)
	assert.Equal(t, 5, rl.config.Burst, "Burst should be 10% of max (5)")
	assert.NotNil(t, rl.config.KeyGenerator)
}

// TestPerUserRateLimiter tests PerUserRateLimiter creation
func TestPerUserRateLimiter(t *testing.T) {
	rl := PerUserRateLimiter(100, 1*time.Minute)

	assert.NotNil(t, rl)
	assert.Equal(t, 100, rl.config.Max)
	assert.Equal(t, 10, rl.config.Burst, "Burst should be 10% of max (10)")
	assert.NotNil(t, rl.config.KeyGenerator)
}

// TestRateLimiter_MinimumBurst tests that burst has a minimum of 1
func TestRateLimiter_MinimumBurst(t *testing.T) {
	rl := PerIPRateLimiter(5, 1*time.Minute) // 5 / 10 = 0, should become 1

	assert.NotNil(t, rl)
	assert.Equal(t, 5, rl.config.Max)
	assert.Equal(t, 1, rl.config.Burst, "Burst should have minimum of 1")
}
