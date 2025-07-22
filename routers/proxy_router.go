package routers

import (
	"log"

	"github.com/gofiber/fiber/v2"

	"proxynd/alerts"
	"proxynd/configs"
	authHandlers "proxynd/handlers/auth"
	proxynd "proxynd/handlers/proxy"
	"proxynd/middlewares"
)

// ProxyRouter is exported
// ProxyRouter performs an operation
func ProxyRouter(app *fiber.App) {
	// Read global configuration
	globalConfig := configs.GlobalConfig{}
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

	// Handle all proxy types with unified proxy handler
	app.Get("/proxy/:type/*",
		middlewares.ProxyPolicyMiddleware(),
		middlewares.DefaultAccessLogMiddleware(),
		authHandlers.OptionalAuth(),      // Optional OAuth2/JWT authentication
		authHandlers.BasicAuthFallback(), // BasicAuth fallback
		verificationHandler.VerificationMiddleware(),
		proxynd.UnifiedProxyHandler,
	)
	app.Post("/proxy/:type/*",
		middlewares.ProxyPolicyMiddleware(),
		middlewares.DefaultAccessLogMiddleware(),
		authHandlers.OptionalAuth(),      // Optional OAuth2/JWT authentication
		authHandlers.BasicAuthFallback(), // BasicAuth fallback
		verificationHandler.VerificationMiddleware(),
		proxynd.UnifiedProxyHandler,
	)
	app.Put("/proxy/:type/*",
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
