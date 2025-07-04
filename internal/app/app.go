package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"

	"proxynd/internal/repositories/cache"
	"proxynd/internal/repositories/config"
	"proxynd/internal/services/adapters"
	"proxynd/internal/services/proxy"
	"proxynd/logging"
	"proxynd/routers"
)

// Application represents the main application
type Application struct {
	config         *Config
	logger         logging.Logger
	fiberApp       *fiber.App
	serviceFactory *proxy.ServiceFactory
	configRepo     *config.FileRepository
	cacheRepo      *cache.FileRepository
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

	// Set defaults from environment if not provided
	if cfg.Port == "" {
		cfg.Port = os.Getenv("SERVER_PORT")
		if cfg.Port == "" {
			return nil, fmt.Errorf("SERVER_PORT not set")
		}
	}

	if cfg.StorageDir == "" {
		cfg.StorageDir = os.Getenv("STORAGE_DIR")
		if cfg.StorageDir == "" {
			cfg.StorageDir = "./storage"
		}
	}

	if cfg.ConfigDir == "" {
		cfg.ConfigDir = os.Getenv("CONFIG_DIR")
		if cfg.ConfigDir == "" {
			cfg.ConfigDir = "./config"
		}
	}

	if cfg.CacheMaxAge == 0 {
		cfg.CacheMaxAge = 24 * time.Hour
	}

	app := &Application{
		config: cfg,
		logger: logger,
	}

	// Initialize repositories
	if err := app.initializeRepositories(); err != nil {
		return nil, fmt.Errorf("failed to initialize repositories: %w", err)
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Watch for configuration changes
	if err := app.configRepo.WatchConfig(ctx, app.handleConfigChange); err != nil {
		app.logger.Warn("Failed to start config watcher", logging.F("error", err))
	}

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

	app.logger.Info("Server shutdown complete")
	return nil
}

// Stop gracefully stops the application
func (app *Application) Stop() error {
	return app.fiberApp.Shutdown()
}

// GetFiberApp returns the underlying Fiber app
func (app *Application) GetFiberApp() *fiber.App {
	return app.fiberApp
}

// Private methods

func (app *Application) initializeRepositories() error {
	// Initialize cache repository
	cacheRepo, err := cache.NewFileRepository(
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
	configAdapter := adapters.NewConfigAdapter(app.configRepo)
	upstreamClient := adapters.NewHTTPUpstreamClient(30 * time.Second)

	// Create service factory
	app.serviceFactory = proxy.NewServiceFactory(
		cacheAdapter,
		configAdapter,
		upstreamClient,
	)

	return nil
}

func (app *Application) initializeFiberApp() {
	// Create base router
	app.fiberApp = routers.BaseRouter()

	// Register all routers
	routers.HealthRouter(app.fiberApp)
	routers.ProxyRouter(app.fiberApp)
	routers.CacheRouter(app.fiberApp)
	routers.ConfigRouter(app.fiberApp)
	routers.StatusRouter(app.fiberApp)
	routers.UserRouter(app.fiberApp)
	routers.TestRouter(app.fiberApp)
	routers.WebhookRouter(app.fiberApp)

	// Store service factory in app locals for handlers to use
	app.fiberApp.Use(func(c *fiber.Ctx) error {
		c.Locals("serviceFactory", app.serviceFactory)
		return c.Next()
	})
}

func (app *Application) handleConfigChange(proxyType string, config interface{}) {
	app.logger.Info("Configuration changed",
		logging.F("proxy_type", proxyType))

	// Reload services to pick up new configuration
	app.serviceFactory.ReloadServices()
}
