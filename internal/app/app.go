package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"

	fiberRouters "proxynd/internal/adapters/http/fiber/routers"
	"proxynd/internal/logging"
	cacheRepo "proxynd/internal/repositories/cache"
	"proxynd/internal/repositories/config"
	"proxynd/internal/services/adapters"
	configService "proxynd/internal/services/config"
	"proxynd/internal/services/proxy"
	"proxynd/plugins"
)

// Application represents the main application
type Application struct {
	config         *Config
	logger         logging.Logger
	fiberApp       *fiber.App
	container      *Container
	serviceFactory *proxy.ServiceFactory
	configService  configService.Service
	configRepo     *config.FileRepository
	cacheRepo      *cacheRepo.FileRepository
	pluginContext  plugins.Context
	pluginManager  *plugins.Manager // New plugin manager
}

// Config holds application configuration
type Config struct {
	Port        string
	Version     string
	BuildTime   string
	CommitSHA   string
	StorageDir  string
	ConfigDir   string
	CacheMaxAge time.Duration
}

// New creates a new application instance
func New(cfg *Config) (*Application, error) {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		// Not fatal - .env file is optional
		fmt.Printf("Warning: %v\n", err)
	}

	// Initialize logging
	if err := logging.SetupLogging(); err != nil {
		return nil, fmt.Errorf("failed to setup logging: %w", err)
	}

	logger := logging.GetLogger()

	// Validate required environment variables and set defaults
	if err := validateAndSetEnvironmentDefaults(cfg, logger); err != nil {
		return nil, fmt.Errorf("environment validation failed: %w", err)
	}

	app := &Application{
		config: cfg,
		logger: logger,
	}

	// Initialize repositories
	if err := app.initializeRepositories(); err != nil {
		return nil, fmt.Errorf("failed to initialize repositories: %w", err)
	}

	// Initialize configuration service
	ctx := context.Background()
	configService, err := configService.NewService(ctx, cfg.ConfigDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize config service: %w", err)
	}
	app.configService = configService

	// Initialize Container (의존성 주입 컨테이너)
	if err := app.initializeContainer(); err != nil {
		return nil, fmt.Errorf("failed to initialize container: %w", err)
	}

	// Initialize services
	if err := app.initializeServices(); err != nil {
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	// Initialize Fiber app
	app.initializeFiberApp()

	return app, nil
}

// Run starts the application
func (app *Application) Run() error {
	// Start configuration watcher
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Configuration changes are now handled by the centralized config service
	// The service validates configurations on load automatically

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		url := fmt.Sprintf("http://0.0.0.0:%s", app.config.Port)
		app.logger.Info("Server starting",
			logging.F("url", url),
			logging.F("port", app.config.Port),
			logging.F("version", app.config.Version))

		if err := app.fiberApp.Listen(":" + app.config.Port); err != nil {
			app.logger.Fatal("Server failed to start",
				logging.F("error", err.Error()),
				logging.F("port", app.config.Port))
		}
	}()

	// Wait for interrupt signal
	<-quit
	app.logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := app.fiberApp.ShutdownWithContext(shutdownCtx); err != nil {
		app.logger.Error("Server forced to shutdown", logging.F("error", err))
		return err
	}

	// Shutdown plugins using manager
	if app.pluginManager != nil {
		if err := app.pluginManager.Shutdown(shutdownCtx); err != nil {
			app.logger.Warn("Plugin shutdown encountered errors",
				logging.F("error", err))
		}
	}

	app.logger.Info("Server shutdown complete")
	return nil
}

// Stop gracefully stops the application
func (app *Application) Stop() error {
	if err := app.fiberApp.Shutdown(); err != nil {
		return err
	}

	// Shutdown plugins using manager
	if app.pluginManager != nil {
		return app.pluginManager.Shutdown(context.Background())
	}

	return nil
}

// GetFiberApp returns the underlying Fiber app
func (app *Application) GetFiberApp() *fiber.App {
	return app.fiberApp
}

func (app *Application) buildPluginContext() plugins.Context {
	env := make(map[string]string)
	for _, kv := range os.Environ() {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}

	return plugins.Context{
		Logger:      plugins.NewLoggerAdapter(app.logger),
		ConfigDir:   app.config.ConfigDir,
		DataDir:     app.config.StorageDir,
		Environment: env,
	}
}

// Private methods

func (app *Application) initializeRepositories() error {
	// Initialize cache repository
	cacheRepo, err := cacheRepo.NewFileRepository(
		app.config.StorageDir,
		10*1024*1024*1024, // 10GB max cache size
		app.config.CacheMaxAge,
	)
	if err != nil {
		return fmt.Errorf("failed to create cache repository: %w", err)
	}
	app.cacheRepo = cacheRepo

	// Initialize config repository
	configRepo, err := config.NewFileRepository(app.config.ConfigDir)
	if err != nil {
		return fmt.Errorf("failed to create config repository: %w", err)
	}
	app.configRepo = configRepo

	return nil
}

func (app *Application) initializeServices() error {
	// Create service adapters
	cacheAdapter := adapters.NewCacheAdapter(app.cacheRepo, app.config.CacheMaxAge)
	configAdapter := adapters.NewConfigServiceAdapter(app.configService)
	upstreamClient := adapters.NewHTTPUpstreamClient(30 * time.Second)

	// Create service factory
	app.serviceFactory = proxy.NewServiceFactory(
		cacheAdapter,
		configAdapter,
		upstreamClient,
	)

	// Initialize ProxyService with hexagonal architecture (using stub adapters)
	if err := app.initializeProxyService(); err != nil {
		app.logger.Warn("Failed to initialize ProxyService with hexagonal architecture",
			logging.F("error", err))
		// Non-fatal - fallback to legacy architecture
	}

	return nil
}

// initializeProxyService creates ProxyService with NoOp stub adapters
func (app *Application) initializeProxyService() error {
	// Dynamically import stub package to avoid circular dependency
	// Using late binding pattern
	return app.container.InitializeProxyService()
}

// initializeContainer Container 의존성 주입 컨테이너 초기화
func (app *Application) initializeContainer() error {
	// Container 생성
	app.container = NewContainer(app.config)

	app.logger.Info("Container initialized successfully")
	return nil
}

func (app *Application) initializeFiberApp() {
	// Create base Fiber app
	app.fiberApp = fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Connection Pool 초기화 (v1 API에서 사용)
	if err := InitializeConnectionPool(); err != nil {
		app.logger.Warn("Connection Pool 초기화 실패, 기본값 사용",
			logging.F("error", err))
	}

	// Load unified config using hexagonal architecture
	unifiedConfig, err := app.container.GetRootConfig()
	if err != nil {
		app.logger.Warn("Failed to load unified config", logging.F("error", err))
		// System cannot function without config, but let routing system handle the error
		unifiedConfig = nil
	}

	// === NEW ARCHITECTURE ROUTING (Exclusive Mode) ===
	// HEXAGONAL_MIGRATION Phase 3 Complete - Using hexagonal architecture exclusively (2025-11-29)
	routeConfig := InitializeRouteConfig(unifiedConfig, app.container)
	SetupRoutes(app.fiberApp, routeConfig)

	// === Hexagonal Architecture Proxy Router (Dependency Injection) ===
	// ProxyService-based architecture with container
	app.logger.Info("Setting up Hexagonal Architecture proxy router")
	containerAdapter := &containerAdapter{container: app.container}
	fiberRouters.ProxyRouterWithContainer(app.fiberApp, containerAdapter)

	// All other routes are now configured via SetupRoutes() above
	// This includes:
	// - Health routes (new architecture)
	// - Proxy routes (V1 and V3 APIs)
	// - Search routes
	// - Pool routes (Connection Pool API)
	// - Cache routes
	// - WebUI routes
	// - API v1 routes
	// - Enterprise API routes
	// - Swagger documentation
	// - Metrics routes
	// - Ansible routes (Galaxy v3 API)

	// Store service factory and container in app locals for handlers to use
	app.fiberApp.Use(func(c *fiber.Ctx) error {
		c.Locals("serviceFactory", app.serviceFactory)
		c.Locals("container", app.container)
		c.Locals("routeConfig", routeConfig)         // TODO: HEXAGONAL_MIGRATION - Add route config for runtime switching
		c.Locals("configService", app.configService) // Add config service for persistence
		c.Locals("pluginManager", app.pluginManager) // Add plugin manager for event notifications
		return c.Next()
	})

	// Initialize plugin system with new manager
	app.pluginContext = app.buildPluginContext()

	// Load plugin configuration from YAML (falls back to defaults if not found)
	pluginConfig, err := plugins.LoadConfig(app.config.ConfigDir)
	if err != nil {
		app.logger.Warn("Failed to load plugin config, using defaults",
			logging.F("error", err))
		pluginConfig = plugins.DefaultConfig()
	} else {
		app.logger.Info("Plugin configuration loaded",
			logging.F("config_dir", app.config.ConfigDir),
			logging.F("enabled", pluginConfig.Enabled))
	}

	// Create plugin manager
	app.pluginManager = plugins.NewManager(pluginConfig, plugins.NewLoggerAdapter(app.logger))

	// Discover and initialize plugins
	if err := app.pluginManager.Discover(); err != nil {
		app.logger.Error("Plugin discovery failed", logging.F("error", err))
	} else {
		// Initialize plugins
		if err := app.pluginManager.Initialize(app.fiberApp, app.pluginContext); err != nil {
			app.logger.Error("Plugin initialization failed", logging.F("error", err))
		} else {
			// Run ready hooks
			if err := app.pluginManager.Ready(app.pluginContext); err != nil {
				app.logger.Error("Plugin ready hooks failed", logging.F("error", err))
			} else {
				// Log summary
				enabledPlugins := app.pluginManager.GetEnabledPlugins()
				pluginNames := make([]string, 0, len(enabledPlugins))
				for _, p := range enabledPlugins {
					pluginNames = append(pluginNames, p.Name)
				}
				app.logger.Info("Plugins initialized and ready",
					logging.F("count", len(enabledPlugins)),
					logging.F("plugins", pluginNames))
			}
		}
	}

	app.logger.Info("Container-based proxy router initialized",
		logging.F("supported_handlers", []string{"apt", "maven", "npm"}),
		logging.F("architecture", "hexagonal"),
	)
}

// validateAndSetEnvironmentDefaults validates required environment variables and sets defaults
func validateAndSetEnvironmentDefaults(cfg *Config, logger logging.Logger) error {
	// Environment variables (matching README requirements)
	envVars := []struct {
		name        string
		description string
		envKey      string
		cfgField    *string
		required    bool
		defaultVal  string
	}{
		{"CONFIG_DIR", "Configuration directory path", "CONFIG_DIR", &cfg.ConfigDir, true, ""},
		{"STORAGE_DIR", "Storage directory path for cache and data", "STORAGE_DIR", &cfg.StorageDir, true, ""},
		{"SERVER_PORT", "Server port number", "SERVER_PORT", &cfg.Port, false, "8080"},
	}

	for _, envVar := range envVars {
		if *envVar.cfgField == "" {
			value := os.Getenv(envVar.envKey)
			if value == "" {
				if envVar.required {
					return fmt.Errorf("required environment variable %s is not set (%s)", envVar.name, envVar.description)
				}
				// Set default value for optional variables
				if envVar.defaultVal != "" {
					*envVar.cfgField = envVar.defaultVal
					logger.Debug("Set config default value",
						logging.F("field", envVar.name),
						logging.F("value", envVar.defaultVal))
				}
			} else {
				*envVar.cfgField = value
				logger.Debug("Set config value from environment",
					logging.F("field", envVar.name),
					logging.F("value", value))
			}
		}
	}

	// Validate SERVER_PORT is a valid integer
	if cfg.Port != "" {
		if _, err := strconv.Atoi(cfg.Port); err != nil {
			return fmt.Errorf("SERVER_PORT must be a valid integer, got: %s", cfg.Port)
		}
	}

	// Set defaults for other optional fields
	if cfg.CacheMaxAge == 0 {
		cfg.CacheMaxAge = 24 * time.Hour
	}

	logger.Info("Environment variables validated and defaults set",
		logging.F("port", cfg.Port),
		logging.F("storage_dir", cfg.StorageDir),
		logging.F("config_dir", cfg.ConfigDir),
		logging.F("cache_max_age", cfg.CacheMaxAge))

	return nil
}

// containerAdapter adapts app.Container to fiberRouters.ContainerProvider interface
type containerAdapter struct {
	container *Container
}

func (ca *containerAdapter) GetLogger() interface{} {
	return ca.container.GetLogger()
}

func (ca *containerAdapter) GetHandlerAdapterFactory() (interface{}, error) {
	return ca.container.GetHandlerAdapterFactory()
}

func (ca *containerAdapter) GetProxyService() interface{} {
	return ca.container.GetProxyService()
}
