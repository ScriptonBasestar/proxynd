package middlewares

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainErrors "proxynd/internal/errors"
)

func TestErrorRecoveryMiddleware(t *testing.T) {
	// Test with default config
	t.Run("Default recovery", func(t *testing.T) {
		app := fiber.New()

		// Apply recovery middleware
		app.Use(ErrorRecovery())

		// Apply error handler to see the converted error
		app.Use(ErrorHandler())

		// Route that panics
		app.Get("/panic", func(_ *fiber.Ctx) error {
			panic("test panic")
		})

		// Route that panics with error
		app.Get("/panic-error", func(_ *fiber.Ctx) error {
			panic(errors.New("panic with error"))
		})

		// Route that panics with nil
		app.Get("/panic-nil", func(_ *fiber.Ctx) error {
			panic(nil)
		})

		// Normal route
		app.Get("/normal", func(c *fiber.Ctx) error {
			return c.SendString("ok")
		})

		// Test string panic
		req, _ := http.NewRequest("GET", "/panic", nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

		var errorResp ErrorResponse
		err = parseJSONResponse(resp, &errorResp)
		require.NoError(t, err)

		assert.Equal(t, domainErrors.ErrCodePanic, errorResp.Error)
		assert.Contains(t, errorResp.Message, "panic recovered")
		assert.Equal(t, "system", errorResp.Domain)
		assert.NotNil(t, errorResp.Details)
		details := errorResp.Details.(map[string]interface{})
		assert.Equal(t, "test panic", details["panic"])
		assert.NotEmpty(t, details["stack"])
	})

	t.Run("Panic with error type", func(t *testing.T) {
		app := fiber.New()
		app.Use(ErrorRecovery())
		app.Use(ErrorHandler())

		app.Get("/panic-error", func(_ *fiber.Ctx) error {
			panic(errors.New("error panic"))
		})

		req, _ := http.NewRequest("GET", "/panic-error", nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

		var errorResp ErrorResponse
		err = parseJSONResponse(resp, &errorResp)
		require.NoError(t, err)

		assert.Equal(t, domainErrors.ErrCodePanic, errorResp.Error)
		details := errorResp.Details.(map[string]interface{})
		assert.Contains(t, details["panic"].(string), "error panic")
	})

	t.Run("Panic with nil", func(t *testing.T) {
		app := fiber.New()
		app.Use(ErrorRecovery())
		app.Use(ErrorHandler())

		app.Get("/panic-nil", func(_ *fiber.Ctx) error {
			var nilPtr *string
			_ = *nilPtr // This will panic with nil pointer dereference
			return nil
		})

		req, _ := http.NewRequest("GET", "/panic-nil", nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("Normal request without panic", func(t *testing.T) {
		app := fiber.New()
		app.Use(ErrorRecovery())

		app.Get("/normal", func(c *fiber.Ctx) error {
			return c.SendString("ok")
		})

		req, _ := http.NewRequest("GET", "/normal", nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		body := make([]byte, 2)
		_, _ = resp.Body.Read(body)
		assert.Equal(t, "ok", string(body))
	})
}

func TestErrorRecoveryMiddleware_CustomConfig(t *testing.T) {
	t.Run("Stack trace disabled", func(t *testing.T) {
		app := fiber.New()

		// Custom config without stack trace
		app.Use(RecoveryWithConfig(RecoveryConfig{
			EnableStackTrace: false,
		}))
		app.Use(ErrorHandler())

		app.Get("/panic", func(_ *fiber.Ctx) error {
			panic("no stack trace")
		})

		req, _ := http.NewRequest("GET", "/panic", nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		var errorResp ErrorResponse
		err = parseJSONResponse(resp, &errorResp)
		require.NoError(t, err)

		// Stack should not be included
		details := errorResp.Details.(map[string]interface{})
		_, hasStack := details["stack"]
		assert.False(t, hasStack)
	})

	t.Run("Custom stack trace handler", func(t *testing.T) {
		app := fiber.New()

		customHandlerCalled := false

		// Custom stack trace handler
		app.Use(RecoveryWithConfig(RecoveryConfig{
			EnableStackTrace: true,
			StackTraceHandler: func(_ *fiber.Ctx, _ interface{}) {
				customHandlerCalled = true
				// This handler is called but doesn't override the response
			},
		}))
		app.Use(ErrorHandler())

		app.Get("/panic", func(_ *fiber.Ctx) error {
			panic("custom handler test")
		})

		req, _ := http.NewRequest("GET", "/panic", nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.True(t, customHandlerCalled)
		// Default panic response is still sent
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

		var errorResp ErrorResponse
		err = parseJSONResponse(resp, &errorResp)
		require.NoError(t, err)

		assert.Equal(t, domainErrors.ErrCodePanic, errorResp.Error)
		details := errorResp.Details.(map[string]interface{})
		assert.Contains(t, details["panic"].(string), "custom handler test")
	})
}

func TestErrorRecoveryMiddleware_ConcurrentPanics(t *testing.T) {
	app := fiber.New()
	app.Use(ErrorRecovery())
	app.Use(ErrorHandler())

	// Route that panics with request ID
	app.Get("/concurrent-panic/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		panic("panic-" + id)
	})

	// Run concurrent panicking requests
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			req, _ := http.NewRequest("GET", "/concurrent-panic/"+string(rune(id+'0')), nil)
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

			var errorResp ErrorResponse
			err = parseJSONResponse(resp, &errorResp)
			assert.NoError(t, err)
			assert.Equal(t, domainErrors.ErrCodePanic, errorResp.Error)

			done <- true
		}(i)
	}

	// Wait for all requests
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestErrorRecoveryMiddleware_NestedPanic(t *testing.T) {
	app := fiber.New()
	app.Use(ErrorRecovery())
	app.Use(ErrorHandler())

	// Middleware that might panic
	app.Use(func(c *fiber.Ctx) error {
		if c.Path() == "/middleware-panic" {
			panic("panic in middleware")
		}
		return c.Next()
	})

	app.Get("/middleware-panic", func(c *fiber.Ctx) error {
		// This won't be reached
		return c.SendString("ok")
	})

	req, _ := http.NewRequest("GET", "/middleware-panic", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

	var errorResp ErrorResponse
	err = parseJSONResponse(resp, &errorResp)
	require.NoError(t, err)

	assert.Equal(t, domainErrors.ErrCodePanic, errorResp.Error)
	details := errorResp.Details.(map[string]interface{})
	assert.Contains(t, details["panic"].(string), "panic in middleware")
}

func TestErrorRecoveryMiddleware_PanicTypes(t *testing.T) {
	app := fiber.New()
	app.Use(ErrorRecovery())
	app.Use(ErrorHandler())

	// Different panic types
	app.Get("/panic-int", func(_ *fiber.Ctx) error {
		panic(42)
	})

	app.Get("/panic-struct", func(_ *fiber.Ctx) error {
		type CustomError struct {
			Code    string
			Message string
		}
		panic(CustomError{Code: "CUSTOM", Message: "custom panic"})
	})

	app.Get("/panic-slice", func(_ *fiber.Ctx) error {
		panic([]string{"multiple", "values"})
	})

	tests := []struct {
		name string
		path string
	}{
		{"Integer panic", "/panic-int"},
		{"Struct panic", "/panic-struct"},
		{"Slice panic", "/panic-slice"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tt.path, nil)
			resp, err := app.Test(req, -1)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

			var errorResp ErrorResponse
			err = parseJSONResponse(resp, &errorResp)
			require.NoError(t, err)

			assert.Equal(t, domainErrors.ErrCodePanic, errorResp.Error)
			details := errorResp.Details.(map[string]interface{})
			assert.NotNil(t, details["panic"])
		})
	}
}

// Benchmark recovery performance
func BenchmarkErrorRecovery_NoPanic(b *testing.B) {
	app := fiber.New()
	app.Use(ErrorRecovery())

	app.Get("/bench", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req, _ := http.NewRequest("GET", "/bench", nil)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, _ := app.Test(req, -1)
			_ = resp.Body.Close()
		}
	})
}

func BenchmarkErrorRecovery_WithPanic(b *testing.B) {
	app := fiber.New()
	app.Use(ErrorRecovery())

	app.Get("/panic", func(_ *fiber.Ctx) error {
		panic("benchmark panic")
	})

	req, _ := http.NewRequest("GET", "/panic", nil)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, _ := app.Test(req, -1)
			_ = resp.Body.Close()
		}
	})
}
