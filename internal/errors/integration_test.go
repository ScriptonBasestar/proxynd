package errors

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock middleware and handler setup for integration testing
func setupTestApp() *fiber.App {
	app := fiber.New()

	// Recovery middleware
	app.Use(func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				err := NewError(ErrCodePanic, "Panic recovered").
					WithDomain("system").
					WithLevel(ErrorLevelCritical).
					WithDetails(map[string]interface{}{
						"panic": r,
						"path":  c.Path(),
					}).
					Build()
				c.Locals("error", err)
				_ = c.Next()
			}
		}()
		return c.Next()
	})

	// Error handler middleware
	app.Use(func(c *fiber.Ctx) error {
		err := c.Next()
		if err != nil {
			return handleError(c, err)
		}

		// Check if panic was recovered
		if panicErr := c.Locals("error"); panicErr != nil {
			return handleError(c, panicErr.(error))
		}

		return nil
	})

	return app
}

func handleError(c *fiber.Ctx, err error) error {
	// Handle DomainError
	if domainErr, ok := err.(*DomainError); ok {
		status := getHTTPStatusForError(domainErr)
		return c.Status(status).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    domainErr.Code,
				"message": domainErr.Message,
				"domain":  domainErr.Domain,
				"details": domainErr.Details,
			},
		})
	}

	// Handle Fiber error
	if fiberErr, ok := err.(*fiber.Error); ok {
		return c.Status(fiberErr.Code).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "FIBER_ERROR",
				"message": fiberErr.Message,
			},
		})
	}

	// Generic error
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error": fiber.Map{
			"code":    "INTERNAL_ERROR",
			"message": err.Error(),
		},
	})
}

func getHTTPStatusForError(err *DomainError) int {
	// Simple mapping for testing
	switch err.Code {
	case ErrCodeAuthFailed:
		return fiber.StatusUnauthorized
	case ErrCodeAuthzFailed:
		return fiber.StatusForbidden
	case "NOT_FOUND":
		return fiber.StatusNotFound
	case "BAD_REQUEST":
		return fiber.StatusBadRequest
	default:
		return fiber.StatusInternalServerError
	}
}

func TestErrorIntegration_CompleteFlow(t *testing.T) {
	app := setupTestApp()

	// Simulate different layers of the application

	// Repository layer
	app.Get("/user/:id", func(c *fiber.Ctx) error {
		userID := c.Params("id")

		// Simulate user not found
		if userID == "999" {
			return NewError("NOT_FOUND", "User not found").
				WithDomain("repository").
				WithDetails(map[string]interface{}{
					"user_id": userID,
					"table":   "users",
				}).
				Build()
		}

		// Simulate database error
		if userID == "500" {
			dbErr := errors.New("connection refused")
			return NewError("DB_ERROR", "Database query failed").
				WithDomain("database").
				WithCause(dbErr).
				WithLevel(ErrorLevelError).
				WithDetails(map[string]interface{}{
					"query":    "SELECT * FROM users WHERE id = ?",
					"params":   []interface{}{userID},
					"duration": "30s",
				}).
				Build()
		}

		return c.JSON(fiber.Map{"id": userID, "name": "John Doe"})
	})

	// Service layer with auth
	app.Post("/admin/action", func(c *fiber.Ctx) error {
		// Check auth header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return NewAuthError(ErrCodeAuthFailed, "Missing authentication token").
				WithDetails(map[string]interface{}{
					"header": "Authorization",
					"ip":     c.IP(),
				}).
				Build()
		}

		// Simulate invalid token
		if authHeader == "Bearer invalid" {
			return NewAuthError(ErrCodeAuthFailed, "Invalid authentication token").
				WithDetails(map[string]interface{}{
					"token_prefix": "Bearer",
				}).
				Build()
		}

		// Simulate insufficient permissions
		if authHeader == "Bearer user-token" {
			return NewAuthError(ErrCodeAuthzFailed, "Insufficient permissions").
				WithDetails(map[string]interface{}{
					"required_role": "admin",
					"user_roles":    []string{"user"},
					"resource":      "/admin/action",
				}).
				Build()
		}

		return c.JSON(fiber.Map{"status": "success"})
	})

	// Handler that panics
	app.Get("/panic", func(_ *fiber.Ctx) error {
		panic("something went terribly wrong")
	})

	// Cache layer simulation
	app.Get("/cache/:key", func(c *fiber.Ctx) error {
		key := c.Params("key")

		if key == "expired" {
			return ErrCacheExpired
		}

		if key == "invalid" {
			return WrapCacheError(
				errors.New("corrupted data"),
				ErrCodeCacheRead,
				"Failed to deserialize cache data",
			)
		}

		return c.JSON(fiber.Map{"key": key, "value": "cached-value"})
	})

	// Test scenarios
	tests := []struct {
		name           string
		method         string
		path           string
		headers        map[string]string
		expectedStatus int
		expectedCode   string
		expectedDomain string
	}{
		{
			name:           "User not found",
			method:         "GET",
			path:           "/user/999",
			expectedStatus: fiber.StatusNotFound,
			expectedCode:   "NOT_FOUND",
			expectedDomain: "repository",
		},
		{
			name:           "Database error",
			method:         "GET",
			path:           "/user/500",
			expectedStatus: fiber.StatusInternalServerError,
			expectedCode:   "DB_ERROR",
			expectedDomain: "database",
		},
		{
			name:           "Missing auth",
			method:         "POST",
			path:           "/admin/action",
			expectedStatus: fiber.StatusUnauthorized,
			expectedCode:   ErrCodeAuthFailed,
			expectedDomain: "auth",
		},
		{
			name:           "Invalid token",
			method:         "POST",
			path:           "/admin/action",
			headers:        map[string]string{"Authorization": "Bearer invalid"},
			expectedStatus: fiber.StatusUnauthorized,
			expectedCode:   ErrCodeAuthFailed,
			expectedDomain: "auth",
		},
		{
			name:           "Insufficient permissions",
			method:         "POST",
			path:           "/admin/action",
			headers:        map[string]string{"Authorization": "Bearer user-token"},
			expectedStatus: fiber.StatusForbidden,
			expectedCode:   ErrCodeAuthzFailed,
			expectedDomain: "auth",
		},
		{
			name:           "Panic recovery",
			method:         "GET",
			path:           "/panic",
			expectedStatus: fiber.StatusInternalServerError,
			expectedCode:   ErrCodePanic,
			expectedDomain: "system",
		},
		{
			name:           "Cache expired",
			method:         "GET",
			path:           "/cache/expired",
			expectedStatus: fiber.StatusInternalServerError,
			expectedCode:   "CACHE002",
			expectedDomain: "cache",
		},
		{
			name:           "Cache read error",
			method:         "GET",
			path:           "/cache/invalid",
			expectedStatus: fiber.StatusInternalServerError,
			expectedCode:   ErrCodeCacheRead,
			expectedDomain: "cache",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, tt.path, nil)

			// Add headers
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			resp, err := app.Test(req, -1)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Parse response
			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			errorData := result["error"].(map[string]interface{})
			assert.Equal(t, tt.expectedCode, errorData["code"])
			assert.Equal(t, tt.expectedDomain, errorData["domain"])
			assert.NotEmpty(t, errorData["message"])

			// Check if details exist for some errors
			if tt.expectedCode == ErrCodePanic {
				details := errorData["details"].(map[string]interface{})
				assert.NotNil(t, details["panic"])
				assert.Equal(t, "/panic", details["path"])
			}
		})
	}
}

func TestErrorIntegration_ErrorChaining(t *testing.T) {
	app := setupTestApp()

	// Simulate a multi-layer error scenario
	app.Get("/complex-error", func(_ *fiber.Ctx) error {
		// Layer 1: System error
		fileErr := errors.New("permission denied")

		// Layer 2: Cache error wrapping system error
		cacheErr := NewCacheError(ErrCodeCacheWrite, "Failed to write cache file").
			WithCause(fileErr).
			WithDetails(map[string]interface{}{
				"cache_dir": "/var/cache/app",
				"key":       "user:123",
			}).
			Build()

		// Layer 3: Service error wrapping cache error
		serviceErr := NewError("SERVICE_ERROR", "Failed to update user data").
			WithDomain("service").
			WithCause(cacheErr).
			WithDetails(map[string]interface{}{
				"operation": "UpdateUser",
				"user_id":   "123",
			}).
			Build()

		return serviceErr
	})

	req, _ := http.NewRequest("GET", "/complex-error", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	errorData := result["error"].(map[string]interface{})
	assert.Equal(t, "SERVICE_ERROR", errorData["code"])
	assert.Equal(t, "service", errorData["domain"])

	details := errorData["details"].(map[string]interface{})
	assert.Equal(t, "UpdateUser", details["operation"])
	assert.Equal(t, "123", details["user_id"])
}

func TestErrorIntegration_ConcurrentRequests(t *testing.T) {
	app := setupTestApp()

	// Counter for concurrent requests
	requestCount := 0

	app.Get("/concurrent/:type", func(c *fiber.Ctx) error {
		requestCount++
		errorType := c.Params("type")

		switch errorType {
		case "auth":
			return NewAuthError(ErrCodeAuthFailed, "Concurrent auth error").
				WithDetails(map[string]interface{}{
					"request_num": requestCount,
				}).
				Build()
		case "cache":
			return NewCacheError(ErrCodeCacheRead, "Concurrent cache error").
				WithDetails(map[string]interface{}{
					"request_num": requestCount,
				}).
				Build()
		case "panic":
			panic("concurrent panic")
		default:
			return c.JSON(fiber.Map{"type": errorType})
		}
	})

	// Run concurrent requests
	types := []string{"auth", "cache", "panic", "normal"}
	done := make(chan bool, len(types)*5)

	for i := 0; i < 5; i++ {
		for _, errorType := range types {
			go func(errType string) {
				req, _ := http.NewRequest("GET", "/concurrent/"+errType, nil)
				resp, err := app.Test(req, -1)
				assert.NoError(t, err)
				defer func() { _ = resp.Body.Close() }()

				if errType == "normal" {
					assert.Equal(t, fiber.StatusOK, resp.StatusCode)
				} else {
					assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
				}

				done <- true
			}(errorType)
		}
	}

	// Wait for all requests
	for i := 0; i < len(types)*5; i++ {
		<-done
	}
}
