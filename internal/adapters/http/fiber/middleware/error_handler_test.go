package middlewares

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainErrors "proxynd/internal/errors"
)

func TestErrorHandlerMiddleware(t *testing.T) {
	// Test setup
	app := fiber.New()

	// Apply error handler middleware
	app.Use(ErrorHandler())

	// Test routes
	app.Get("/domain-error", func(_ *fiber.Ctx) error {
		return domainErrors.NewError("SYS001", "Test domain error").
			WithDomain("test").
			WithCause(errors.New("original cause")).
			WithDetails(map[string]interface{}{
				"user_id": "12345",
				"action":  "test_action",
			}).
			Build()
	})

	app.Get("/fiber-error", func(_ *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusBadRequest, "Bad request from Fiber")
	})

	app.Get("/generic-error", func(_ *fiber.Ctx) error {
		return errors.New("generic error occurred")
	})

	app.Get("/auth-error", func(_ *fiber.Ctx) error {
		return domainErrors.NewError(domainErrors.ErrCodeAuthFailed, "Authentication failed").
			WithDomain("auth").
			Build()
	})

	app.Get("/cache-error", func(_ *fiber.Ctx) error {
		return domainErrors.NewError(domainErrors.ErrCodeCacheRead, "Cache read failed").
			WithDomain("cache").
			WithCause(errors.New("redis connection failed")).
			Build()
	})

	app.Get("/no-error", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	t.Run("DomainError with details", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/domain-error", nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

		// Parse response
		var errorResp ErrorResponse
		err = parseJSONResponse(resp, &errorResp)
		require.NoError(t, err)

		// Verify error response structure
		assert.Equal(t, "SYS001", errorResp.Error)
		assert.Equal(t, "Test domain error", errorResp.Message)
		assert.Equal(t, "test", errorResp.Domain)
		assert.NotNil(t, errorResp.Details)
		details := errorResp.Details.(map[string]interface{})
		assert.Equal(t, "12345", details["user_id"])
		assert.Equal(t, "test_action", details["action"])
		assert.NotEmpty(t, errorResp.TraceID)
		assert.NotZero(t, errorResp.Timestamp)
	})

	t.Run("Fiber error", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/fiber-error", nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var errorResp ErrorResponse
		err = parseJSONResponse(resp, &errorResp)
		require.NoError(t, err)

		assert.Equal(t, "FIBER_ERROR", errorResp.Error)
		assert.Equal(t, "Bad request from Fiber", errorResp.Message)
	})

	t.Run("Generic error", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/generic-error", nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

		var errorResp ErrorResponse
		err = parseJSONResponse(resp, &errorResp)
		require.NoError(t, err)

		assert.Equal(t, "INTERNAL_ERROR", errorResp.Error)
		// Generic errors return a generic message for security (not exposing internal details)
		assert.Equal(t, "내부 서버 오류가 발생했습니다", errorResp.Message)
	})

	t.Run("Auth error with proper status", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/auth-error", nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// AUTH_FAILED should map to 401 Unauthorized
		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

		var errorResp ErrorResponse
		err = parseJSONResponse(resp, &errorResp)
		require.NoError(t, err)

		assert.Equal(t, domainErrors.ErrCodeAuthFailed, errorResp.Error)
		assert.Equal(t, "auth", errorResp.Domain)
	})

	t.Run("Cache error with cause", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/cache-error", nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

		var errorResp ErrorResponse
		err = parseJSONResponse(resp, &errorResp)
		require.NoError(t, err)

		assert.Equal(t, domainErrors.ErrCodeCacheRead, errorResp.Error)
		assert.Equal(t, "cache", errorResp.Domain)
		// Cause is included in the message
		assert.Contains(t, errorResp.Message, "Cache read failed")
	})

	t.Run("No error - normal response", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/no-error", nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var result map[string]string
		err = parseJSONResponse(resp, &result)
		require.NoError(t, err)
		assert.Equal(t, "ok", result["status"])
	})
}

func TestErrorHandlerMiddleware_EdgeCases(t *testing.T) {
	app := fiber.New()
	app.Use(ErrorHandler())

	// Test nil error
	app.Get("/nil-error", func(_ *fiber.Ctx) error {
		var err error
		return err
	})

	// Test error with nil cause (using mapped error code SYS002)
	app.Get("/nil-cause", func(_ *fiber.Ctx) error {
		return domainErrors.NewError("SYS002", "Error with nil cause").
			WithCause(nil).
			Build()
	})

	// Test error with empty details (using mapped error code SYS003)
	app.Get("/empty-details", func(_ *fiber.Ctx) error {
		return domainErrors.NewError("SYS003", "Error with empty details").
			WithDetails(map[string]interface{}{}).
			Build()
	})

	t.Run("Nil error", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/nil-error", nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Should pass through without error
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Error with nil cause", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/nil-cause", nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

		var errorResp ErrorResponse
		err = parseJSONResponse(resp, &errorResp)
		require.NoError(t, err)

		assert.Equal(t, "SYS002", errorResp.Error)
		assert.Equal(t, "Error with nil cause", errorResp.Message)
	})

	t.Run("Error with empty details", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/empty-details", nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

		var errorResp ErrorResponse
		err = parseJSONResponse(resp, &errorResp)
		require.NoError(t, err)

		assert.Equal(t, "SYS003", errorResp.Error)
		assert.Empty(t, errorResp.Details)
	})
}

func TestErrorHandlerMiddleware_ConcurrentErrors(t *testing.T) {
	app := fiber.New()
	app.Use(ErrorHandler())

	// Create a route that generates errors (using SYS001 which is mapped to 500)
	app.Get("/concurrent-error/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		return domainErrors.NewError("SYS001", "Concurrent error").
			WithDetails(map[string]interface{}{"request_id": id}).
			Build()
	})

	// Run concurrent requests
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			req, _ := http.NewRequest("GET", fmt.Sprintf("/concurrent-error/%d", id), nil)
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
			done <- true
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestErrorHandlerMiddleware_ErrorLevels(t *testing.T) {
	app := fiber.New()
	app.Use(ErrorHandler())

	// Test different error levels
	app.Get("/info-error", func(_ *fiber.Ctx) error {
		return domainErrors.NewError("INFO001", "Info level error").
			WithLevel(domainErrors.ErrorLevelInfo).
			Build()
	})

	app.Get("/warning-error", func(_ *fiber.Ctx) error {
		return domainErrors.NewError("WARN001", "Warning level error").
			WithLevel(domainErrors.ErrorLevelWarning).
			Build()
	})

	app.Get("/critical-error", func(_ *fiber.Ctx) error {
		return domainErrors.NewError("CRIT001", "Critical level error").
			WithLevel(domainErrors.ErrorLevelCritical).
			Build()
	})

	tests := []struct {
		name     string
		path     string
		code     string
		expected int // Expected status code based on error level (when code is unmapped)
	}{
		{
			name:     "Info level",
			path:     "/info-error",
			code:     "INFO001",
			expected: fiber.StatusInternalServerError, // Default for unmapped codes
		},
		{
			name:     "Warning level",
			path:     "/warning-error",
			code:     "WARN001",
			expected: fiber.StatusNotFound, // Warning level maps to 404 when code is unmapped
		},
		{
			name:     "Critical level",
			path:     "/critical-error",
			code:     "CRIT001",
			expected: fiber.StatusInternalServerError, // Critical level maps to 500
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tt.path, nil)
			resp, err := app.Test(req, -1)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, tt.expected, resp.StatusCode)

			var errorResp ErrorResponse
			err = parseJSONResponse(resp, &errorResp)
			require.NoError(t, err)

			assert.Equal(t, tt.code, errorResp.Error)
		})
	}
}

// Helper function to parse JSON response
func parseJSONResponse(resp *http.Response, v interface{}) error {
	decoder := json.NewDecoder(resp.Body)
	return decoder.Decode(v)
}
