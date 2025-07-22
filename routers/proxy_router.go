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
	proxyGroup := app.Group("/proxy")

	// Apply proxy middleware
	proxyGroup.Use(middlewares.ProxyPolicyMiddleware())
	proxyGroup.Use(middlewares.DefaultAccessLogMiddleware())

	// Apply unified auth middleware (JWT token first, BasicAuth fallback)
	proxyGroup.Use(authHandlers.OptionalAuth())      // Optional OAuth2/JWT authentication
	proxyGroup.Use(authHandlers.BasicAuthFallback()) // BasicAuth fallback

	// Add package verification middleware
	proxyGroup.Use(verificationHandler.VerificationMiddleware())

	// Handle all proxy types with unified proxy handler
	proxyGroup.Get("/:type/*", proxynd.UnifiedProxyHandler)
	proxyGroup.Post("/:type/*", proxynd.UnifiedProxyHandler)
	proxyGroup.Put("/:type/*", proxynd.UnifiedProxyHandler)

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
