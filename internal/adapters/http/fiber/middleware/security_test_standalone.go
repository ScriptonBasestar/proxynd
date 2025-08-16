package middlewares

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSecurityHeadersStandalone tests security headers without logging dependencies.
func TestSecurityHeadersStandalone(t *testing.T) {
	app := fiber.New()

	// Use basic security headers without logging
	app.Use(func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "1; mode=block")
		return c.Next()
	})

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", resp.Header.Get("X-XSS-Protection"))
}

// TestBasicInputValidationStandalone is exported
// TestBasicInputValidationStandalone is a test function
func TestBasicInputValidationStandalone(t *testing.T) {
	app := fiber.New()

	// Simple input validation without dependencies
	app.Use(func(c *fiber.Ctx) error {
		// Check for basic XSS patterns
		body := string(c.Body())
		if strings.Contains(strings.ToLower(body), "<script") {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid input"})
		}
		return c.Next()
	})

	app.Post("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name:           "Valid input",
			body:           `{"name": "test"}`,
			expectedStatus: 200,
		},
		{
			name:           "XSS attempt",
			body:           `{"content": "<script>alert('xss')</script>"}`,
			expectedStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
	// TestSimpleRateLimitStandalone is exported
}

// TestSimpleRateLimitStandalone is exported

// TestSimpleRateLimitStandalone performs test simple rate limit standalone operation
func TestSimpleRateLimitStandalone(t *testing.T) {
	app := fiber.New()

	// Simple counter-based rate limiting
	requestCount := 0
	maxRequests := 2

	app.Use(func(c *fiber.Ctx) error {
		requestCount++
		if requestCount > maxRequests {
			return c.Status(429).JSON(fiber.Map{"error": "Rate limit exceeded"})
		}
		return c.Next()
	})

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// First request should succeed
	req1 := httptest.NewRequest("GET", "/test", nil)
	resp1, err := app.Test(req1)
	require.NoError(t, err)
	defer func() { _ = resp1.Body.Close() }()
	assert.Equal(t, 200, resp1.StatusCode)

	// Second request should succeed
	req2 := httptest.NewRequest("GET", "/test", nil)
	resp2, err := app.Test(req2)
	require.NoError(t, err)
	defer func() { _ = resp2.Body.Close() }()
	assert.Equal(t, 200, resp2.StatusCode)

	// Third request should fail
	req3 := httptest.NewRequest("GET", "/test", nil)
	resp3, err := app.Test(req3)
	require.NoError(t, err)
	defer func() { _ = resp3.Body.Close() }()
	assert.Equal(t, 429, resp3.StatusCode)
}

// TestCORSHeadersStandalone tests CORS headers in standalone mode
func TestCORSHeadersStandalone(t *testing.T) {
	app := fiber.New()

	allowedOrigins := []string{"https://example.com"}

	app.Use(func(c *fiber.Ctx) error {
		origin := c.Get("Origin")

		isAllowed := false
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				isAllowed = true
				break
			}
		}

		if isAllowed {
			c.Set("Access-Control-Allow-Origin", origin)
		} else {
			c.Set("Access-Control-Allow-Origin", "null")
		}

		if c.Method() == "OPTIONS" {
			return c.SendStatus(204)
		}

		return c.Next()
	})

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	tests := []struct {
		name           string
		origin         string
		expectedOrigin string
	}{
		{
			name:           "Allowed origin",
			origin:         "https://example.com",
			expectedOrigin: "https://example.com",
		},
		{
			name:           "Non-allowed origin",
			origin:         "https://malicious.com",
			expectedOrigin: "null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Origin", tt.origin)

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			// TestContentTypeValidationStandalone is exported
			assert.Equal(t, 200, resp.StatusCode)
			assert.Equal(t, tt.expectedOrigin, resp.Header.Get("Access-Control-Allow-Origin"))
		})
	}
}

// TestContentTypeValidationStandalone tests content type validation in standalone mode
func TestContentTypeValidationStandalone(t *testing.T) {
	app := fiber.New()

	allowedTypes := []string{"application/json", "application/xml", "text/plain"}

	app.Use(func(c *fiber.Ctx) error {
		if c.Method() == HTTPMethodPOST && len(c.Body()) > 0 {
			contentType := c.Get("Content-Type")

			isValid := false
			for _, allowed := range allowedTypes {
				if strings.HasPrefix(contentType, allowed) {
					isValid = true
					break
				}
			}

			if !isValid {
				return c.Status(400).JSON(fiber.Map{"error": "Invalid content type"})
			}
		}
		return c.Next()
	})

	app.Post("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	tests := []struct {
		name           string
		contentType    string
		body           string
		expectedStatus int
	}{
		{
			name:           "Valid JSON",
			contentType:    "application/json",
			body:           `{"test": "data"}`,
			expectedStatus: 200,
		},
		{
			name:           "Valid XML",
			contentType:    "application/xml",
			body:           `<root></root>`,
			expectedStatus: 200,
		},
		{
			name:           "Invalid HTML",
			contentType:    "text/html",
			body:           `<html></html>`,
			expectedStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", tt.contentType)

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}
