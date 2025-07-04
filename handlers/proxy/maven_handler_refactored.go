package proxy

import (
	"context"
	"io"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/services/proxy"
	"proxynd/logging"
)

// MavenHandlerRefactored is the refactored Maven proxy handler using the service layer
type MavenHandlerRefactored struct {
	proxyService proxy.ProxyService
	logger       logging.Logger
}

// NewMavenHandlerRefactored creates a new Maven handler with the service layer
func NewMavenHandlerRefactored(proxyService proxy.ProxyService) *MavenHandlerRefactored {
	return &MavenHandlerRefactored{
		proxyService: proxyService,
		logger:       logging.GetLogger(),
	}
}

// Handle processes Maven proxy requests
func (h *MavenHandlerRefactored) Handle(c *fiber.Ctx) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
	defer cancel()
	
	// Extract request information
	requestPath := c.Params("*")
	
	// Log the request
	h.logger.Info("Maven proxy request",
		logging.F("path", requestPath),
		logging.F("method", c.Method()),
		logging.F("ip", c.IP()))
	
	// Build proxy request
	proxyReq := proxy.ProxyRequest{
		Path:       requestPath,
		ProxyType:  "maven",
		Method:     c.Method(),
		Headers:    h.extractHeaders(c),
		RemoteAddr: c.IP(),
	}
	
	// Handle request through service
	response, err := h.proxyService.HandleRequest(ctx, proxyReq)
	if err != nil {
		h.logger.Error("Failed to handle Maven request",
			logging.F("path", requestPath),
			logging.F("error", err))
		return c.Status(fiber.StatusInternalServerError).SendString("Internal server error")
	}
	
	// Ensure body is closed
	if response.Body != nil {
		defer response.Body.Close()
	}
	
	// Set response headers
	for key, value := range response.Headers {
		c.Set(key, value)
	}
	
	// Set content type
	if response.ContentType != "" {
		c.Set("Content-Type", response.ContentType)
	}
	
	// Read the body
	body, err := io.ReadAll(response.Body)
	if err != nil {
		h.logger.Error("Failed to read response body",
			logging.F("error", err))
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to read response")
	}
	
	// Send response
	return c.Status(response.StatusCode).Send(body)
}

// extractHeaders extracts relevant headers from the Fiber context
func (h *MavenHandlerRefactored) extractHeaders(c *fiber.Ctx) map[string]string {
	headers := make(map[string]string)
	
	// Extract common headers
	if userAgent := c.Get("User-Agent"); userAgent != "" {
		headers["User-Agent"] = userAgent
	}
	
	if accept := c.Get("Accept"); accept != "" {
		headers["Accept"] = accept
	}
	
	if acceptEncoding := c.Get("Accept-Encoding"); acceptEncoding != "" {
		headers["Accept-Encoding"] = acceptEncoding
	}
	
	// Extract authorization if present
	if auth := c.Get("Authorization"); auth != "" {
		headers["Authorization"] = auth
	}
	
	return headers
}

// MavenProxyRefactored is the refactored function that uses the handler
func MavenProxyRefactored(serviceFactory *proxy.ServiceFactory) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get Maven service from factory
		service, err := serviceFactory.GetService("maven")
		if err != nil {
			logging.GetLogger().Error("Failed to get Maven service",
				logging.F("error", err))
			return c.Status(fiber.StatusInternalServerError).SendString("Service unavailable")
		}
		
		// Create handler and process request
		handler := NewMavenHandlerRefactored(service)
		return handler.Handle(c)
	}
}