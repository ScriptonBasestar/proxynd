package app

import (
	"github.com/gofiber/fiber/v2"

	// Import new adapter structure
	fiberHandlers "proxynd/internal/adapters/http/fiber/handlers"
	fiberRouters "proxynd/internal/adapters/http/fiber/routers"
	"proxynd/internal/config"

	// TODO: HEXAGONAL_MIGRATION - Remove legacy imports after full migration
	"proxynd/internal/handlers-legacy"
	"proxynd/internal/logging"
	middlewares "proxynd/internal/middleware-legacy"
	"proxynd/internal/routers"
)

// RouteConfig holds configuration for route setup
type RouteConfig struct {
	UseNewArchitecture bool // Feature flag for gradual migration
	UnifiedConfig      *config.UnifiedConfig
	Logger             logging.Logger
}

// SetupRoutes configures all application routes with hybrid old/new architecture
func SetupRoutes(app *fiber.App, config *RouteConfig) {
	logger := config.Logger
	if logger == nil {
		logger = logging.GetLogger()
	}

	logger.Info("Setting up routes",
		logging.Bool("new_architecture", config.UseNewArchitecture))

	// TODO: HEXAGONAL_MIGRATION - Phase out legacy routes gradually
	if config.UseNewArchitecture {
		setupNewArchitectureRoutes(app, config)
	} else {
		setupLegacyRoutes(app, config)
	}
}

// setupNewArchitectureRoutes configures routes using new hexagonal architecture
func setupNewArchitectureRoutes(app *fiber.App, config *RouteConfig) {
	logger := config.Logger

	// Setup middlewares using new architecture
	setupNewMiddlewares(app, config)

	// Health routes using new architecture
	logger.Info("Setting up health routes (new architecture)")
	fiberRouters.HealthRouterWithConfig(app, config.UnifiedConfig)

	// API routes
	api := app.Group("/api")
	v1 := api.Group("/v1")

	// Proxy routes using hybrid approach
	logger.Info("Setting up proxy routes (hybrid architecture)")
	fiberRouters.ProxyRouter(app)   // New structure
	fiberRouters.ProxyRouterV3(app) // V3 routes are added to main app

	// Search routes
	logger.Info("Setting up search routes")
	v1.Get("/search", fiberHandlers.SearchHandler)

	// Pool routes
	logger.Info("Setting up pool routes")
	fiberRouters.PoolRouter(app)

	// Cache routes
	logger.Info("Setting up cache routes")
	fiberRouters.CacheRouter(app)

	// API v1 routes (for WebUI integration)

	// WebUI static file serving (Enterprise/Cloud only)
	logger.Info("Setting up WebUI routes")
	fiberRouters.SetupWebUIRoutes(app, getEdition())
	logger.Info("Setting up API v1 routes")
	fiberRouters.SetupAPIv1Routes(app, config.UnifiedConfig)

	// TODO: HEXAGONAL_MIGRATION - Add other routers as they are migrated
	logger.Info("New architecture routes setup completed")
}

// setupLegacyRoutes configures routes using legacy structure (fallback)
func setupLegacyRoutes(app *fiber.App, config *RouteConfig) {
	logger := config.Logger
	logger.Info("Setting up legacy routes")

	// Legacy middleware setup
	setupLegacyMiddlewares(app, config)

	// Legacy routers - maintain existing behavior
	routers.HealthRouterWithConfig(app, config.UnifiedConfig)
	routers.ProxyRouter(app)
	routers.ProxyRouterV3(app) // V3 routes use main app

	app.Get("/api/v1/search", handlers.SearchHandler)

	routers.PoolRouter(app)
	routers.CacheRouter(app)
	routers.ConfigRouter(app)
	routers.StatusRouter(app)
	routers.UserRouter(app)
	routers.AuthRouter(app)
	routers.MetricsRouter(app, config.UnifiedConfig)
	routers.WebhookRouter(app)
	// TODO: HEXAGONAL_MIGRATION - Add container support when available
	// routers.ContainerProxyRouterSetup(app, container) // Requires container parameter
	routers.APKMirrorRouter(app)
	routers.APKVerificationRouter(app)
	routers.UnifiedRouterV1(app)

	logger.Info("Legacy routes setup completed")
}

// setupNewMiddlewares configures middlewares using new architecture
func setupNewMiddlewares(app *fiber.App, config *RouteConfig) {
	logger := config.Logger
	logger.Info("Setting up middlewares (new architecture)")

	// TODO: HEXAGONAL_MIGRATION - Implement new middleware architecture
	// For now, fall back to legacy middlewares to maintain functionality
	setupLegacyMiddlewares(app, config)

	logger.Info("New middleware setup completed")
}

// setupLegacyMiddlewares configures middlewares using legacy structure
func setupLegacyMiddlewares(app *fiber.App, config *RouteConfig) {
	logger := config.Logger
	logger.Info("Setting up legacy middlewares")

	// Basic middlewares
	app.Use(middlewares.ErrorRecovery())
	app.Use(middlewares.DefaultAccessLogMiddleware())

	// Security middlewares
	app.Use(middlewares.SecurityHeaders())
	app.Use(middlewares.RateLimit())

	// Authentication middleware (conditional)
	// TODO: HEXAGONAL_MIGRATION - Add auth config support
	// if config.UnifiedConfig != nil && config.UnifiedConfig.Auth.Enabled {
	//	app.Use(middlewares.JWTAuthMiddleware())
	// }

	logger.Info("Legacy middlewares setup completed")
}

// InitializeRouteConfig creates a default route configuration
func InitializeRouteConfig(unifiedConfig *config.UnifiedConfig, useNewArch bool) *RouteConfig {
	return &RouteConfig{
		UseNewArchitecture: useNewArch,
		UnifiedConfig:      unifiedConfig,
		Logger:             logging.GetLogger(),
	}
}

// TODO: HEXAGONAL_MIGRATION - Migration utilities

// EnableNewArchitecture switches to new architecture mode
func (rc *RouteConfig) EnableNewArchitecture() {
	rc.UseNewArchitecture = true
	rc.Logger.Info("Switched to new architecture mode")
}

// DisableNewArchitecture switches to legacy mode (for rollback)
func (rc *RouteConfig) DisableNewArchitecture() {
	rc.UseNewArchitecture = false
	rc.Logger.Info("Switched to legacy architecture mode")
}

// GetMigrationStatus returns current migration status
func (rc *RouteConfig) GetMigrationStatus() map[string]interface{} {
	return map[string]interface{}{
		"new_architecture_enabled": rc.UseNewArchitecture,
		"migration_phase":          "handlers_copied",
		"legacy_fallback_active":   true,
	}
}
