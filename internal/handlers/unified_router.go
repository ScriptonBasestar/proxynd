package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/services/proxy"
	"proxynd/logging"
	"proxynd/pkg/types"
)

// UnifiedProxyRouter sets up unified proxy routes
type UnifiedProxyRouter struct {
	serviceFactory *proxy.ServiceFactory
	handlerFactory types.ProxyHandlerFactory
	adapter        types.FiberProxyAdapter
	logger         logging.Logger
}

// NewUnifiedProxyRouter creates a new unified proxy router
func NewUnifiedProxyRouter(serviceFactory *proxy.ServiceFactory) *UnifiedProxyRouter {
	// Create handler factory
	handlerFactory := types.NewStandardProxyHandlerFactory()

	// Create Fiber adapter
	adapter := types.NewStandardFiberProxyAdapter(30 * time.Second)

	return &UnifiedProxyRouter{
		serviceFactory: serviceFactory,
		handlerFactory: handlerFactory,
		adapter:        adapter,
		logger:         logging.GetLogger(),
	}
}

// Setup registers all proxy routes
func (r *UnifiedProxyRouter) Setup(app *fiber.App) error {
	// Register all handler creators
	if err := r.registerHandlers(); err != nil {
		return err
	}

	// Create proxy group
	proxyGroup := app.Group("/proxy")

	// Add unified proxy route
	proxyGroup.All("/:type/*", r.handleUnifiedProxy)

	// Add health check for each proxy type
	proxyGroup.Get("/:type/health", r.handleHealthCheck)

	return nil
}

// registerHandlers registers all proxy handler creators
func (r *UnifiedProxyRouter) registerHandlers() error {
	// Register Maven handler
	if err := r.handlerFactory.RegisterHandler(
		types.ProxyTypeMaven,
		MavenHandlerCreator(r.serviceFactory),
	); err != nil {
		return err
	}

	// Register APT handler
	if err := r.handlerFactory.RegisterHandler(
		types.ProxyTypeAPT,
		AptHandlerCreator(r.serviceFactory),
	); err != nil {
		return err
	}

	// Register NPM handler
	if err := r.handlerFactory.RegisterHandler(
		types.ProxyTypeNPM,
		NpmHandlerCreator(r.serviceFactory),
	); err != nil {
		return err
	}

	// Register other handlers...
	// TODO: Add Docker, PIP, YUM, APK handlers

	return nil
}

// handleUnifiedProxy handles all proxy requests through the unified interface
func (r *UnifiedProxyRouter) handleUnifiedProxy(c *fiber.Ctx) error {
	// Extract proxy type
	proxyTypeStr := c.Params("type")
	proxyType := types.ProxyType(proxyTypeStr)

	// Get handler for proxy type
	handler, err := r.handlerFactory.CreateHandler(proxyType)
	if err != nil {
		r.logger.Error("Failed to get proxy handler",
			logging.F("type", proxyType),
			logging.F("error", err))
		return c.Status(fiber.StatusNotFound).SendString("Unsupported proxy type")
	}

	// Use adapter to handle request
	fiberHandler := r.adapter.Adapt(handler)
	return fiberHandler(c)
}

// handleHealthCheck handles health check requests for proxy types
func (r *UnifiedProxyRouter) handleHealthCheck(c *fiber.Ctx) error {
	// Extract proxy type
	proxyTypeStr := c.Params("type")
	proxyType := types.ProxyType(proxyTypeStr)

	// Get handler for proxy type
	handler, err := r.handlerFactory.CreateHandler(proxyType)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status": "error",
			"type":   proxyType,
			"error":  "Unsupported proxy type",
		})
	}

	// Check health
	healthy := handler.IsHealthy(c.Context())

	status := "healthy"
	statusCode := fiber.StatusOK
	if !healthy {
		status = "unhealthy"
		statusCode = fiber.StatusServiceUnavailable
	}

	return c.Status(statusCode).JSON(fiber.Map{
		"status": status,
		"type":   proxyType,
	})
}

// Stub handler creators for other proxy types
// These would be implemented similarly to MavenHandlerCreator

func AptHandlerCreator(serviceFactory *proxy.ServiceFactory) func() types.ProxyHandler {
	// TODO: Implement APT handler
	return func() types.ProxyHandler {
		return nil
	}
}

func NpmHandlerCreator(serviceFactory *proxy.ServiceFactory) func() types.ProxyHandler {
	// TODO: Implement NPM handler
	return func() types.ProxyHandler {
		return nil
	}
}