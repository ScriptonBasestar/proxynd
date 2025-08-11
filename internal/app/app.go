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
	configService "proxynd/internal/services/config"
	"proxynd/internal/services/proxy"
	"proxynd/logging"
	"proxynd/routers"
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
	configAdapter := adapters.NewConfigServiceAdapter(app.configService)
	upstreamClient := adapters.NewHTTPUpstreamClient(30 * time.Second)

	// Create service factory
	app.serviceFactory = proxy.NewServiceFactory(
		cacheAdapter,
		configAdapter,
		upstreamClient,
	)

	return nil
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

	// === Container 기반 프록시 라우터 (우선순위 최고) ===
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
		return c.Next()
	})

	app.logger.Info("Container-based proxy router initialized",
		logging.F("supported_handlers", []string{"apt", "maven", "npm"}),
	)
}
