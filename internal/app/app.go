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

	"proxynd/cache"
	fiberRouters "proxynd/internal/adapters/http/fiber/routers"
	configTypes "proxynd/internal/config"
	"proxynd/internal/logging"
	cacheRepo "proxynd/internal/repositories/cache"
	"proxynd/internal/repositories/config"
	"proxynd/internal/routers"
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

	if err := plugins.Shutdown(shutdownCtx); err != nil {
		app.logger.Warn("Plugin shutdown encountered errors",
			logging.F("error", err))
	}

	app.logger.Info("Server shutdown complete")
	return nil
}

// Stop gracefully stops the application
func (app *Application) Stop() error {
	if err := app.fiberApp.Shutdown(); err != nil {
		return err
	}
	return plugins.Shutdown(context.Background())
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
	// Create base router
	app.fiberApp = routers.BaseRouter()

	// Connection Pool 초기화 (v1 API에서 사용)
	if err := InitializeConnectionPool(); err != nil {
		app.logger.Warn("Connection Pool 초기화 실패, 기본값 사용",
			logging.F("error", err))
	}

	// TODO: HEXAGONAL_MIGRATION - Convert to unified config loading
	// Load unified config for new routing system
	var unifiedConfig *configTypes.RootConfig
	config := app.container.GetConfig()
	if config != nil {
		// 포트를 정수로 변환
		port := 8080 // 기본값
		if config.Port != "" {
			if p, err := strconv.Atoi(config.Port); err == nil {
				port = p
			}
		}

		// RootConfig 구성
		unifiedConfig = &configTypes.RootConfig{
			Server: configTypes.ServerConfig{
				Host: "0.0.0.0",
				Port: port,
				// TODO: Add Auth config when available
			},
			Cache: configTypes.CacheSettings{
				TTL: 3600,
			},
		}
	}

	// === NEW ARCHITECTURE ROUTING (Hybrid Mode) ===
	// TODO: HEXAGONAL_MIGRATION - Start with legacy mode, gradually enable new architecture
	routeConfig := InitializeRouteConfig(unifiedConfig, false) // Start with legacy mode
	SetupRoutes(app.fiberApp, routeConfig)

	// === Hexagonal Architecture Proxy Router (최우선순위) ===
	// ProxyService 기반 새로운 아키텍처 라우터
	app.logger.Info("Setting up Hexagonal Architecture proxy router")
	containerAdapter := &containerAdapter{container: app.container}
	fiberRouters.ProxyRouterWithContainer(app.fiberApp, containerAdapter)

	// === Container 기반 프록시 라우터 (레거시 Container 시스템) ===
	routers.ContainerProxyRouterSetup(app.fiberApp, app.container)

	// === 새로운 v1 통합 API 등록 (우선순위 높음) ===
	routers.UnifiedRouterV1(app.fiberApp)

	// === 기존 라우터들 (v1으로 마이그레이션 예정) ===
	// 헬스 라우터 초기화 (어댑터 팩토리와 통합)
	if adapterFactory, err := app.container.GetHandlerAdapterFactory(); err == nil {
		routers.InitHandlerAdapterFactory(adapterFactory)
	} else {
		app.logger.Warn("Handler adapter factory initialization failed for health router",
			logging.F("error", err))
	}
	routers.HealthRouter(app.fiberApp)

	// === 강화된 헬스 모니터링 라우터 ===
	if _, err := app.container.GetCacheRepository(); err == nil {
		// FileSystemBackend을 사용해 cache.Manager 생성
		cacheBackend, err := cache.NewFileSystemBackend(config.StorageDir)
		if err == nil {
			cacheOptions := cache.CacheOptions{
				MaxSize:    1024 * 1024 * 1024, // 1GB
				DefaultTTL: time.Hour,
				BasePath:   config.StorageDir,
			}
			cacheManager := cache.NewManager(cacheBackend, cacheOptions)

			enhancedHealthRouter := routers.NewEnhancedHealthRouter(unifiedConfig, cacheManager)
			enhancedHealthRouter.RegisterRoutes(app.fiberApp)
			app.logger.Info("Enhanced health monitoring system initialized")
		} else {
			app.logger.Warn("Enhanced health router initialization failed: cache backend creation failed",
				logging.F("error", err))
		}
	} else {
		app.logger.Warn("Enhanced health router initialization failed: cache repository not available",
			logging.F("error", err))
	}

	routers.ProxyRouter(app.fiberApp)      // 레거시 호환용
	routers.ProxyRouterV3(app.fiberApp)    // V3 라우터 (deprecated)
	routers.RegisterProxyAPI(app.fiberApp) // V3 API (이미 v1)
	routers.PoolRouter(app.fiberApp)       // Connection Pool API (이미 v1)
	routers.CacheRouter(app.fiberApp)
	routers.ConfigRouter(app.fiberApp)
	routers.StatusRouter(app.fiberApp)
	routers.UserRouter(app.fiberApp)
	routers.TestRouter(app.fiberApp)
	routers.WebhookRouter(app.fiberApp) // 이미 v1
	routers.AuthRouter(app.fiberApp)    // 인증 라우터 추가

	// === CLI 호환성 라우터 ===
	routers.APICompatibilityRouter(app.fiberApp) // CLI API 호환성
	// TODO: MetricsRouter 시그니처 수정 필요
	// routers.MetricsRouter(app.fiberApp)     // 메트릭 라우터 추가
	// TODO: APK 라우터들 구현 필요
	// routers.ApkMirrorRouter(app.fiberApp)   // APK 라우터들 추가
	// routers.ApkVerificationRouter(app.fiberApp)

	// === 레거시 호환성 라우터 (가장 낮은 우선순위) ===
	routers.LegacyCompatibilityRouter(app.fiberApp)
	routers.LegacyAPIInfo(app.fiberApp)

	// Store service factory and container in app locals for handlers to use
	app.fiberApp.Use(func(c *fiber.Ctx) error {
		c.Locals("serviceFactory", app.serviceFactory)
		c.Locals("container", app.container)
		c.Locals("routeConfig", routeConfig) // TODO: HEXAGONAL_MIGRATION - Add route config for runtime switching
		return c.Next()
	})

	registeredPlugins := plugins.All()
	if len(registeredPlugins) > 0 {
		app.pluginContext = app.buildPluginContext()
		if err := plugins.Initialize(app.fiberApp, app.pluginContext); err != nil {
			app.logger.Error("Plugin initialization failed",
				logging.F("error", err))
		} else {
			pluginNames := make([]string, 0, len(registeredPlugins))
			for _, plugin := range registeredPlugins {
				pluginNames = append(pluginNames, plugin.Name())
			}
			app.logger.Info("Plugins initialized",
				logging.F("count", len(registeredPlugins)),
				logging.F("plugins", pluginNames))
		}
	}

	app.logger.Info("Container-based proxy router initialized",
		logging.F("supported_handlers", []string{"apt", "maven", "npm"}),
		logging.F("new_architecture_enabled", routeConfig.UseNewArchitecture),
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
