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

	"proxynd/logging"
	"proxynd/routers"
)

// ApplicationRefactored represents the main application with dependency injection
type ApplicationRefactored struct {
	config    *Config
	logger    logging.Logger
	fiberApp  *fiber.App
	container *Container
}

// NewRefactored creates a new application instance with dependency injection
func NewRefactored(cfg *Config) (*ApplicationRefactored, error) {
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
	
	// Create dependency injection container
	container := NewContainer(cfg)
	
	app := &ApplicationRefactored{
		config:    cfg,
		logger:    logger,
		container: container,
	}
	
	// Initialize Fiber app
	if err := app.initializeFiberApp(); err != nil {
		return nil, fmt.Errorf("failed to initialize Fiber app: %w", err)
	}
	
	return app, nil
}

// Run starts the application
func (app *ApplicationRefactored) Run() error {
	// Start configuration watcher
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Get config repository from container
	configRepo, err := app.container.GetConfigRepository()
	if err != nil {
		app.logger.Warn("Failed to get config repository", logging.F("error", err))
	} else {
		// Watch for configuration changes
		if err := configRepo.WatchConfig(ctx, app.handleConfigChange); err != nil {
			app.logger.Warn("Failed to start config watcher", logging.F("error", err))
		}
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
	
	// Close container
	if err := app.container.Close(); err != nil {
		app.logger.Error("Failed to close container", logging.F("error", err))
	}
	
	app.logger.Info("Server shutdown complete")
	return nil
}

// Stop gracefully stops the application
func (app *ApplicationRefactored) Stop() error {
	return app.fiberApp.Shutdown()
}

// GetFiberApp returns the underlying Fiber app
func (app *ApplicationRefactored) GetFiberApp() *fiber.App {
	return app.fiberApp
}

// GetContainer returns the dependency injection container
func (app *ApplicationRefactored) GetContainer() *Container {
	return app.container
}

// Private methods

func (app *ApplicationRefactored) initializeFiberApp() error {
	// Create base router
	app.fiberApp = routers.BaseRouter()
	
	// Create a middleware to inject container into context
	app.fiberApp.Use(func(c *fiber.Ctx) error {
		c.Locals("container", app.container)
		return c.Next()
	})
	
	// Get service factory from container
	serviceFactory, err := app.container.GetServiceFactory()
	if err != nil {
		return fmt.Errorf("failed to get service factory: %w", err)
	}
	
	// Store service factory in app locals for backward compatibility
	app.fiberApp.Use(func(c *fiber.Ctx) error {
		c.Locals("serviceFactory", serviceFactory)
		return c.Next()
	})
	
	// Register all routers with dependency injection
	routers.HealthRouter(app.fiberApp)
	
	// Create proxy router with DI
	if err := app.registerProxyRouter(); err != nil {
		return fmt.Errorf("failed to register proxy router: %w", err)
	}
	
	// Register other routers
	routers.CacheRouter(app.fiberApp)
	routers.ConfigRouter(app.fiberApp)
	routers.StatusRouter(app.fiberApp)
	routers.UserRouter(app.fiberApp)
	routers.TestRouter(app.fiberApp)
	routers.WebhookRouter(app.fiberApp)
	
	return nil
}

func (app *ApplicationRefactored) registerProxyRouter() error {
	// Get unified router from container
	unifiedRouter, err := app.container.GetUnifiedRouter()
	if err != nil {
		return fmt.Errorf("failed to get unified router: %w", err)
	}
	
	// Setup routes
	return unifiedRouter.Setup(app.fiberApp)
}

func (app *ApplicationRefactored) handleConfigChange(proxyType string, config interface{}) {
	app.logger.Info("Configuration changed",
		logging.F("proxy_type", proxyType))
	
	// Get service factory from container and reload
	serviceFactory, err := app.container.GetServiceFactory()
	if err != nil {
		app.logger.Error("Failed to get service factory for reload",
			logging.F("error", err))
		return
	}
	
	serviceFactory.ReloadServices()
}