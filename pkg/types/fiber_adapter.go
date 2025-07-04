package types

import (
	"context"
	"io"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/logging"
)

// StandardFiberProxyAdapter is the standard implementation of FiberProxyAdapter
type StandardFiberProxyAdapter struct {
	logger  logging.Logger
	timeout time.Duration
}

// NewStandardFiberProxyAdapter creates a new standard Fiber proxy adapter
func NewStandardFiberProxyAdapter(timeout time.Duration) *StandardFiberProxyAdapter {
	return &StandardFiberProxyAdapter{
		logger:  logging.GetLogger(),
		timeout: timeout,
	}
}

// Adapt converts a ProxyHandler to a Fiber handler
func (a *StandardFiberProxyAdapter) Adapt(handler ProxyHandler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(c.Context(), a.timeout)
		defer cancel()

		// Extract proxy type from URL parameter
		proxyTypeStr := c.Params("type")
		if proxyTypeStr == "" {
			return c.Status(fiber.StatusBadRequest).SendString("Missing proxy type")
		}

		proxyType := ProxyType(proxyTypeStr)

		// Verify handler matches the requested type
		if handler.GetType() != proxyType {
			return c.Status(fiber.StatusBadRequest).SendString("Handler type mismatch")
		}

		// Build proxy request from Fiber context
		proxyReq := a.buildProxyRequest(c, proxyType)

		// Validate request
		if err := handler.ValidateRequest(proxyReq); err != nil {
			a.logger.Warn("Invalid proxy request",
				logging.F("type", proxyType),
				logging.F("path", proxyReq.Path),
				logging.F("error", err))
			return c.Status(fiber.StatusBadRequest).SendString(err.Error())
		}

		// Handle the request
		startTime := time.Now()
		proxyResp, err := handler.Handle(ctx, proxyReq)
		duration := time.Since(startTime)

		if err != nil {
			// Check if it's a proxy error with specific status code
			if proxyErr, ok := err.(*ProxyError); ok {
				a.logger.Error("Proxy handler error",
					logging.F("type", proxyType),
					logging.F("path", proxyReq.Path),
					logging.F("status", proxyErr.StatusCode),
					logging.F("error", err))
				return c.Status(proxyErr.StatusCode).SendString(proxyErr.Message)
			}

			// Generic error
			a.logger.Error("Proxy handler error",
				logging.F("type", proxyType),
				logging.F("path", proxyReq.Path),
				logging.F("error", err))
			return c.Status(fiber.StatusInternalServerError).SendString("Internal server error")
		}

		// Ensure response body is closed
		if proxyResp.Body != nil {
			defer proxyResp.Body.Close()
		}

		// Log successful request
		a.logger.Info("Proxy request handled",
			logging.F("type", proxyType),
			logging.F("path", proxyReq.Path),
			logging.F("status", proxyResp.StatusCode),
			logging.F("cached", proxyResp.Cached),
			logging.F("duration_ms", duration.Milliseconds()))

		// Set response headers
		for key, value := range proxyResp.Headers {
			c.Set(key, value)
		}

		// Set content type
		if proxyResp.ContentType != "" {
			c.Set("Content-Type", proxyResp.ContentType)
		}

		// Set cache status header
		if proxyResp.Cached {
			c.Set("X-Cache", "HIT")
		} else {
			c.Set("X-Cache", "MISS")
		}

		// Set proxy type header
		c.Set("X-Proxy-Type", string(proxyType))

		// Read response body
		body, err := io.ReadAll(proxyResp.Body)
		if err != nil {
			a.logger.Error("Failed to read response body",
				logging.F("type", proxyType),
				logging.F("error", err))
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to read response")
		}

		// Send response
		return c.Status(proxyResp.StatusCode).Send(body)
	}
}

// buildProxyRequest builds a ProxyRequest from Fiber context
func (a *StandardFiberProxyAdapter) buildProxyRequest(c *fiber.Ctx, proxyType ProxyType) *ProxyRequest {
	// Extract headers
	headers := make(map[string]string)
	c.Request().Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
	})

	// Extract query parameters
	queryParams := make(map[string]string)
	c.Request().URI().QueryArgs().VisitAll(func(key, value []byte) {
		queryParams[string(key)] = string(value)
	})

	// Get request path
	path := c.Params("*")
	if path == "" {
		path = c.Path()
		// Remove proxy prefix if present
		prefix := "/proxy/" + string(proxyType) + "/"
		if len(path) > len(prefix) && path[:len(prefix)] == prefix {
			path = path[len(prefix):]
		}
	}

	return &ProxyRequest{
		Type:        proxyType,
		Method:      c.Method(),
		Path:        path,
		Headers:     headers,
		QueryParams: queryParams,
		Body:        c.Request().BodyStream(),
		RemoteAddr:  c.IP(),
	}
}

// Ensure StandardFiberProxyAdapter implements FiberProxyAdapter
var _ FiberProxyAdapter = (*StandardFiberProxyAdapter)(nil)