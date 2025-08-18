package routers

import (
	"log"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/alerts"
	"proxynd/internal/config"
	authHandlers "proxynd/internal/handlers-legacy/auth"
	proxynd "proxynd/internal/handlers-legacy/proxy"
	middlewares "proxynd/internal/middleware-legacy"
)

// ProxyRouter is exported
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
	verificationHandler := proxynd.NewVerificationHandler(&globalConfig, alertManager)

	// Setup unified proxy router
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
		proxynd.UnifiedProxyHandler,
	)
	app.Post("/proxy/:type/*",
		createDeprecationMiddleware("POST", "/proxy/:type/*", "/api/v1/proxy/:type/*"),
		middlewares.ProxyPolicyMiddleware(),
		middlewares.DefaultAccessLogMiddleware(),
		authHandlers.OptionalAuth(),      // Optional OAuth2/JWT authentication
		authHandlers.BasicAuthFallback(), // BasicAuth fallback
		verificationHandler.VerificationMiddleware(),
		proxynd.UnifiedProxyHandler,
	)
	app.Put("/proxy/:type/*",
		createDeprecationMiddleware("PUT", "/proxy/:type/*", "/api/v1/proxy/:type/*"),
		middlewares.ProxyPolicyMiddleware(),
		middlewares.DefaultAccessLogMiddleware(),
		authHandlers.OptionalAuth(),      // Optional OAuth2/JWT authentication
		authHandlers.BasicAuthFallback(), // BasicAuth fallback
		verificationHandler.VerificationMiddleware(),
		proxynd.UnifiedProxyHandler,
	)

	// Keep existing individual routes for backward compatibility (optional)
	// Can be removed in the future
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
