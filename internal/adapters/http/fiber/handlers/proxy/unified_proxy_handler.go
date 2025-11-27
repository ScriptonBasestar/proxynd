package proxy

import (
	"github.com/gofiber/fiber/v2"

	"proxynd/internal/factory"
	"proxynd/internal/logging"
	"proxynd/internal/usecase"
)

// UnifiedProxyHandlerStruct 통합 프록시 핸들러 (Hexagonal Architecture)
type UnifiedProxyHandlerStruct struct {
	proxyService   *usecase.ProxyService
	adapterFactory *factory.HandlerAdapterFactory
	logger         logging.Logger
}

// NewUnifiedProxyHandler creates a new unified proxy handler with dependency injection
func NewUnifiedProxyHandler(
	proxyService *usecase.ProxyService,
	adapterFactory *factory.HandlerAdapterFactory,
	logger logging.Logger,
) *UnifiedProxyHandlerStruct {
	return &UnifiedProxyHandlerStruct{
		proxyService:   proxyService,
		adapterFactory: adapterFactory,
		logger:         logger,
	}
}

// Handle processes unified proxy requests using injected dependencies
func (h *UnifiedProxyHandlerStruct) Handle(c *fiber.Ctx) error {
	proxyType := c.Params("type")
	path := c.Params("*")

	h.logger.Info("Access unified proxy",
		logging.F("type", proxyType),
		logging.F("path", path),
		logging.F("method", c.Method()))

	// Try ProxyService first (new architecture)
	if h.proxyService != nil {
		req := &usecase.ProxyRequest{
			PackageType: proxyType,
			Path:        path,
			Method:      c.Method(),
			Headers:     make(map[string]string),
			QueryParams: make(map[string]string),
			Body:        c.Body(),
		}

		// Copy headers
		for key, values := range c.GetReqHeaders() {
			if len(values) > 0 {
				req.Headers[key] = values[0]
			}
		}

		// Copy query params
		for key, value := range c.Context().QueryArgs().All() {
			req.QueryParams[string(key)] = string(value)
		}

		resp, err := h.proxyService.HandleProxyRequest(c.Context(), req)
		if err != nil {
			h.logger.Error("ProxyService error",
				logging.F("error", err),
				logging.F("type", proxyType))
			// Return error to let ErrorHandler middleware handle HTTP status mapping
			return err
		}

		// Set Content-Type if provided
		if resp.ContentType != "" {
			c.Set("Content-Type", resp.ContentType)
		}

		// Set response headers
		for key, value := range resp.Headers {
			c.Set(key, value)
		}

		return c.Status(resp.StatusCode).Send(resp.Content)
	}

	// Fallback to adapter factory (legacy support during migration)
	if h.adapterFactory != nil {
		adapter, err := h.adapterFactory.GetAdapter(proxyType)
		if err != nil {
			h.logger.Error("Failed to get adapter",
				logging.F("type", proxyType),
				logging.F("error", err))
			return c.Status(fiber.StatusBadRequest).SendString("Unknown proxy type: " + proxyType)
		}
		return adapter.Handle(c)
	}

	// No service available
	h.logger.Error("No proxy service or adapter factory available")
	return c.Status(fiber.StatusInternalServerError).SendString("Proxy service not initialized")
}

// Backward-compatible function wrappers for router migration
// These allow routers to continue using function-style handlers while we migrate to struct-based DI

var (
	globalHandlerInstance *UnifiedProxyHandlerStruct
	globalAdapterFactory  *factory.HandlerAdapterFactory
)

// InitializeGlobalHandler initializes the global handler instance for backward compatibility
func InitializeGlobalHandler() {
	if globalAdapterFactory == nil {
		globalAdapterFactory = factory.NewHandlerAdapterFactory()
	}
	if globalHandlerInstance == nil {
		logger := logging.GetLogger()
		globalHandlerInstance = NewUnifiedProxyHandler(nil, globalAdapterFactory, logger)
		logger.Info("Global unified proxy handler initialized for backward compatibility")
	}
}

// UnifiedProxyHandler backward-compatible function wrapper
// Deprecated: Use NewUnifiedProxyHandler() with dependency injection instead
func UnifiedProxyHandler(c *fiber.Ctx) error {
	if globalHandlerInstance == nil {
		InitializeGlobalHandler()
	}
	return globalHandlerInstance.Handle(c)
}

// Deprecated: This function wrapper is for backward compatibility only.
// For new code, use NewUnifiedProxyHandler() with dependency injection.
// Migration path: Router → UnifiedProxyHandlerStruct (Phase 2b complete, 2025-11-27)
// Planned removal: Phase 3 (legacy cleanup, target v2.0.0)
