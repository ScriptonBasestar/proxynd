package routers

import (
	"log"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/adapters/http/fiber/handlers/proxy"
	fiberMiddlewares "proxynd/internal/adapters/http/fiber/middleware"
	"proxynd/internal/alerts"
	"proxynd/internal/config"
	"proxynd/internal/factory"
	authHandlers "proxynd/internal/adapters/http/fiber/handlers/auth"
	proxyHandlers "proxynd/internal/adapters/http/fiber/handlers/proxy"
	"proxynd/internal/logging"
	middlewares "proxynd/internal/adapters/http/fiber/middleware"
	"proxynd/internal/usecase"
)

// ContainerProvider interface for dependency injection
type ContainerProvider interface {
	GetLogger() interface{}
	GetHandlerAdapterFactory() (interface{}, error)
	GetProxyService() interface{}
}

// ProxyRouter is exported - Legacy version without DI
// ProxyRouter performs an operation
func ProxyRouter(app *fiber.App) {
	// Read global configuration
	globalConfig := config.GlobalConfig{}
	if err := globalConfig.ReadConfig(); err != nil {
		log.Printf("Warning: Failed to read global config: %v", err)
	}

	// Initialize alert manager
	alertConfig := loadAlertConfig()
	alertManager := alerts.NewAlertManager(alertConfig)

	// Register log alert channel
	if logAlerter, err := alerts.NewLogAlerter(alerts.LogAlerterConfig{
		Enabled:    true,
		LogFile:    "./logs/verification-alerts.log",
		JSONFormat: true,
	}); err == nil {
		alertManager.RegisterAlerter(logAlerter)
	}

	// Create verification handler
	verificationHandler := proxyHandlers.NewVerificationHandler(&globalConfig, alertManager)

	// Setup unified proxy router (legacy mode)
	// Handle all proxy requests with /proxy/:type/*path format
	// Note: Using direct app registration instead of Group to ensure correct routing

	// HTTP method validator for read-only proxy
	methodValidator := middlewares.ReadOnlyMethodValidator()

	log.Printf("⭐ ProxyRouter: Setting up read-only proxy routes (GET, HEAD only)")
	log.Printf("⭐ ProxyRouter: POST/PUT/DELETE/PATCH will return 405")

	// Handle all proxy types with unified proxy handler (with deprecation warning)
	// Only GET and HEAD methods are allowed for read-only proxy
	app.Get("/proxy/:type/*",
		createDeprecationMiddleware("GET", "/proxy/:type/*", "/api/v1/proxy/:type/*"),
		middlewares.ProxyPolicyMiddleware(),
		middlewares.DefaultAccessLogMiddleware(),
		authHandlers.OptionalAuth(),      // Optional OAuth2/JWT authentication
		authHandlers.BasicAuthFallback(), // BasicAuth fallback
		verificationHandler.VerificationMiddleware(),
		proxyHandlers.UnifiedProxyHandler,
	)
	app.Head("/proxy/:type/*",
		middlewares.ProxyPolicyMiddleware(),
		middlewares.DefaultAccessLogMiddleware(),
		authHandlers.OptionalAuth(),      // Optional OAuth2/JWT authentication
		authHandlers.BasicAuthFallback(), // BasicAuth fallback
		verificationHandler.VerificationMiddleware(),
		proxyHandlers.UnifiedProxyHandler,
	)

	// Reject all other HTTP methods with 405 Method Not Allowed
	app.Post("/proxy/:type/*", methodValidator)
	app.Put("/proxy/:type/*", methodValidator)
	app.Delete("/proxy/:type/*", methodValidator)
	app.Patch("/proxy/:type/*", methodValidator)

	// Keep existing individual routes for backward compatibility (optional)
	// Can be removed in the future
}

// ProxyRouterWithContainer creates proxy routes using Container-based DI (Hexagonal Architecture)
func ProxyRouterWithContainer(app *fiber.App, container ContainerProvider) {
	log.Printf("Setting up proxy router with container-based dependency injection")

	// Get logger from container
	logger := container.GetLogger()

	// Get ProxyService from container (may be nil during gradual migration)
	proxyServiceInterface := container.GetProxyService()

	// Get adapter factory from container
	adapterFactoryInterface, err := container.GetHandlerAdapterFactory()
	if err != nil {
		log.Printf("Warning: Failed to get adapter factory from container: %v", err)
		// Fallback to legacy router
		ProxyRouter(app)
		return
	}

	// Check if we have both ProxyService and AdapterFactory
	if proxyServiceInterface != nil && adapterFactoryInterface != nil {
		// Type assertions
		proxyService, psOk := proxyServiceInterface.(*usecase.ProxyService)
		adapterFactory, afOk := adapterFactoryInterface.(*factory.HandlerAdapterFactory)
		loggerTyped, logOk := logger.(logging.Logger)

		if psOk && afOk && logOk {
			// Create new architecture handler with DI
			handler := proxy.NewUnifiedProxyHandler(proxyService, adapterFactory, loggerTyped)

			log.Printf("Container-based proxy router initialized with ProxyService (hexagonal architecture)")

			// Create method validator middleware for read-only proxy
			methodValidator := fiberMiddlewares.ReadOnlyMethodValidator()

			// Register unified proxy routes with new architecture
			// Only GET and HEAD methods are allowed for read-only proxy
			app.Get("/proxy/:type/*", handler.Handle)
			app.Head("/proxy/:type/*", handler.Handle)

			// Reject all other HTTP methods with 405 Method Not Allowed
			app.Post("/proxy/:type/*", methodValidator)
			app.Put("/proxy/:type/*", methodValidator)
			app.Delete("/proxy/:type/*", methodValidator)
			app.Patch("/proxy/:type/*", methodValidator)

			return
		}

		log.Printf("Warning: Type assertion failed (ProxyService: %v, Factory: %v, Logger: %v)", psOk, afOk, logOk)
	}

	// Fallback: Use legacy router if new architecture is not ready
	log.Printf("Note: ProxyService not available, falling back to legacy proxy router")
	ProxyRouter(app)
}

// createDeprecationMiddleware creates a middleware that adds deprecation headers
func createDeprecationMiddleware(method, oldPath, newPath string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Add deprecation headers
		c.Set("X-API-Deprecated", "true")
		c.Set("X-API-Deprecated-Info", "Use "+newPath+" instead")
		c.Set("X-API-Migration-Guide", "https://docs.proxynd.io/api/migration")

		// Log deprecation warning
		log.Printf("DEPRECATION WARNING: %s %s used by %s - Use %s instead",
			method, oldPath, c.IP(), newPath)

		return c.Next()
	}
}

// loadAlertConfig loads alert configuration
func loadAlertConfig() *alerts.AlertConfig {
	// Default configuration
	return &alerts.AlertConfig{
		Enabled: true,
		RateLimit: alerts.RateLimitConfig{
			Enabled:      true,
			MaxPerMinute: 10,
			MaxPerHour:   100,
			BurstSize:    5,
		},
	}
}
