package routers

import (
	"log"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/alerts"
	"proxynd/internal/config"
	authHandlers "proxynd/internal/handlers-legacy/auth"
	legacyProxy "proxynd/internal/handlers-legacy/proxy"
	middlewares "proxynd/internal/middleware-legacy"
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
	verificationHandler := legacyProxy.NewVerificationHandler(&globalConfig, alertManager)

	// Setup unified proxy router (legacy mode)
	// Handle all proxy requests with /proxy/:type/*path format
	// Note: Using direct app registration instead of Group to ensure correct routing

	// Handle all proxy types with unified proxy handler (with deprecation warning)
	app.Get("/proxy/:type/*",
		createDeprecationMiddleware("GET", "/proxy/:type/*", "/api/v1/proxy/:type/*"),
		middlewares.ProxyPolicyMiddleware(),
		middlewares.DefaultAccessLogMiddleware(),
		authHandlers.OptionalAuth(),      // Optional OAuth2/JWT authentication
		authHandlers.BasicAuthFallback(), // BasicAuth fallback
		verificationHandler.VerificationMiddleware(),
		legacyProxy.UnifiedProxyHandler,
	)
	app.Post("/proxy/:type/*",
		createDeprecationMiddleware("POST", "/proxy/:type/*", "/api/v1/proxy/:type/*"),
		middlewares.ProxyPolicyMiddleware(),
		middlewares.DefaultAccessLogMiddleware(),
		authHandlers.OptionalAuth(),      // Optional OAuth2/JWT authentication
		authHandlers.BasicAuthFallback(), // BasicAuth fallback
		verificationHandler.VerificationMiddleware(),
		legacyProxy.UnifiedProxyHandler,
	)
	app.Put("/proxy/:type/*",
		createDeprecationMiddleware("PUT", "/proxy/:type/*", "/api/v1/proxy/:type/*"),
		middlewares.ProxyPolicyMiddleware(),
		middlewares.DefaultAccessLogMiddleware(),
		authHandlers.OptionalAuth(),      // Optional OAuth2/JWT authentication
		authHandlers.BasicAuthFallback(), // BasicAuth fallback
		verificationHandler.VerificationMiddleware(),
		legacyProxy.UnifiedProxyHandler,
	)

	// Keep existing individual routes for backward compatibility (optional)
	// Can be removed in the future
}

// ProxyRouterWithContainer creates proxy routes using Container-based DI (Hexagonal Architecture)
func ProxyRouterWithContainer(app *fiber.App, container ContainerProvider) {
	log.Printf("Setting up proxy router with container-based dependency injection")

	// Get logger from container (will be used when full DI is implemented)
	_ = container.GetLogger()

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

	// Type assertions with nil checks
	var proxyService interface{}
	if proxyServiceInterface != nil {
		proxyService = proxyServiceInterface
	}

	var adapterFactory interface{}
	if adapterFactoryInterface != nil {
		adapterFactory = adapterFactoryInterface
	}

	// Create new architecture handler (will be implemented after import issues resolved)
	// For now, use factory-based approach
	log.Printf("Container-based proxy router initialized (ProxyService: %v, Factory: %v)",
		proxyService != nil, adapterFactory != nil)

	// Note: Actual handler setup will be completed after resolving circular import
	// Temporary: Fall back to legacy for now
	log.Printf("Note: Using legacy proxy handler temporarily until full DI integration")
	ProxyRouter(app)

	// TODO: Implement proper DI-based routing once circular imports are resolved
	/*
		logger := container.GetLogger()
		handler := proxy.NewUnifiedProxyHandler(proxyService, adapterFactory, logger)
		app.Get("/proxy/:type/*", handler.Handle)
		app.Post("/proxy/:type/*", handler.Handle)
		app.Put("/proxy/:type/*", handler.Handle)
	*/
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
