package app

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gofiber/fiber/v2"

	configadapter "proxynd/internal/adapters/config"
	"proxynd/internal/config"
	"proxynd/internal/container"
	"proxynd/internal/containerhandlers"
	"proxynd/internal/factory"
	"proxynd/internal/handlers"
	"proxynd/internal/logging"
	"proxynd/internal/metrics"
	"proxynd/internal/ports"
	"proxynd/internal/repositories/cache"
	configRepo "proxynd/internal/repositories/config"
	"proxynd/internal/services/adapters"
	"proxynd/internal/services/proxy"
	"proxynd/pkg/types"
)

// Container is the dependency injection container
type Container struct {
	mu     sync.RWMutex
	config *Config
	logger logging.Logger

	// 설정 캐싱 및 핫 리로드 관련 필드
	unifiedConfigLoader *config.UnifiedConfigLoader
	unifiedConfig       interface{} // Using interface{} to avoid type issues for now
	configCacheMu       sync.RWMutex
	configWatcher       *fsnotify.Watcher
	configChangeCbs     []func(interface{})
	configNotifier      *ConfigChangeNotifier

	// Repositories
	cacheRepo  cache.Repository
	configRepo configRepo.Repository

	// Services
	cacheService   proxy.CacheService
	configService  proxy.ConfigService
	upstreamClient proxy.UpstreamClient
	serviceFactory *proxy.ServiceFactory

	// Handlers
	handlerFactory        types.ProxyHandlerFactory
	handlerAdapterFactory *factory.HandlerAdapterFactory
	// unifiedRouter  *proxyHandlers.UnifiedProxyRouter

	// Singleton instances
	singletons map[string]interface{}
}

// Hexagonal Architecture usecase layer will be imported separately to avoid circular dependency
// The ProxyService will be stored in singletons map with key "proxy-service"

// NewContainer creates a new dependency injection container
func NewContainer(cfg *Config) *Container {
	container := &Container{
		config:          cfg,
		logger:          logging.GetLogger(),
		singletons:      make(map[string]interface{}),
		configChangeCbs: make([]func(interface{}), 0),
		configNotifier:  NewConfigChangeNotifier(),
	}

	// UnifiedConfigLoader 초기화
	loader, err := config.NewUnifiedConfigLoader(cfg.ConfigDir)
	if err != nil {
		container.logger.Error("Failed to create unified config loader", logging.F("error", err))
		return container
	}
	container.unifiedConfigLoader = loader

	// 초기 설정 로드
	if unifiedConfig, err := container.unifiedConfigLoader.LoadConfig(); err == nil {
		container.unifiedConfig = unifiedConfig
		container.logger.Info("Initial unified configuration loaded successfully")
	} else {
		container.logger.Error("Failed to load initial unified config", logging.F("error", err))
	}

	// 설정 파일 감시 시작
	container.startConfigWatcher()

	return container
}

// GetLogger returns the logger instance
func (c *Container) GetLogger() logging.Logger {
	return c.logger
}

// GetConfig returns the application configuration
func (c *Container) GetConfig() *Config {
	return c.config
}

// GetUnifiedConfig returns the cached unified configuration
func (c *Container) GetUnifiedConfig() interface{} {
	c.configCacheMu.RLock()
	defer c.configCacheMu.RUnlock()
	return c.unifiedConfig
}

// GetAptProxyConfig returns the APT proxy configuration
func (c *Container) GetAptProxyConfig() (*config.AptProxyConfig, error) {
	c.mu.RLock()

	// Check singleton cache first
	if cached, exists := c.singletons["apt-proxy-config"]; exists {
		c.mu.RUnlock()
		if cfg, ok := cached.(*config.AptProxyConfig); ok {
			return cfg, nil
		}
	}
	c.mu.RUnlock()

	// Load configuration
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if cached, exists := c.singletons["apt-proxy-config"]; exists {
		if cfg, ok := cached.(*config.AptProxyConfig); ok {
			return cfg, nil
		}
	}

	// For now, return empty config - will be handled by config service
	cfg := &config.AptProxyConfig{}

	// Cache for future use
	c.singletons["apt-proxy-config"] = cfg

	return cfg, nil
}

// GetMavenProxyConfig returns the Maven proxy configuration
func (c *Container) GetMavenProxyConfig() (*config.MavenProxySettings, error) {
	c.mu.RLock()

	// Check singleton cache first
	if cached, exists := c.singletons["maven-proxy-config"]; exists {
		c.mu.RUnlock()
		if cfg, ok := cached.(*config.MavenProxySettings); ok {
			return cfg, nil
		}
	}
	c.mu.RUnlock()

	// Load configuration
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if cached, exists := c.singletons["maven-proxy-config"]; exists {
		if cfg, ok := cached.(*config.MavenProxySettings); ok {
			return cfg, nil
		}
	}

	// For now, return empty config - will be handled by config service
	cfg := &config.MavenProxySettings{}

	// Cache for future use
	c.singletons["maven-proxy-config"] = cfg

	return cfg, nil
}

// GetNpmProxyConfig returns the NPM proxy configuration
func (c *Container) GetNpmProxyConfig() (*config.NpmProxySettings, error) {
	c.mu.RLock()

	// Check singleton cache first
	if cached, exists := c.singletons["npm-proxy-config"]; exists {
		c.mu.RUnlock()
		if cfg, ok := cached.(*config.NpmProxySettings); ok {
			return cfg, nil
		}
	}
	c.mu.RUnlock()

	// Load configuration
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if cached, exists := c.singletons["npm-proxy-config"]; exists {
		if cfg, ok := cached.(*config.NpmProxySettings); ok {
			return cfg, nil
		}
	}

	// For now, return empty config - will be handled by config service
	cfg := &config.NpmProxySettings{}

	// Cache for future use
	c.singletons["npm-proxy-config"] = cfg

	return cfg, nil
}

// GetDockerProxyConfig returns the Docker proxy configuration
func (c *Container) GetDockerProxyConfig() (*config.DockerProxySettings, error) {
	c.mu.RLock()

	// Check singleton cache first
	if cached, exists := c.singletons["docker-proxy-config"]; exists {
		c.mu.RUnlock()
		if cfg, ok := cached.(*config.DockerProxySettings); ok {
			return cfg, nil
		}
	}
	c.mu.RUnlock()

	// Load configuration
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if cached, exists := c.singletons["docker-proxy-config"]; exists {
		if cfg, ok := cached.(*config.DockerProxySettings); ok {
			return cfg, nil
		}
	}

	// For now, return empty config - will be handled by config service
	cfg := &config.DockerProxySettings{}

	// Cache for future use
	c.singletons["docker-proxy-config"] = cfg

	return cfg, nil
}

// GetPipProxyConfig returns the PIP proxy configuration
func (c *Container) GetPipProxyConfig() (*config.PipProxySettings, error) {
	c.mu.RLock()

	// Check singleton cache first
	if cached, exists := c.singletons["pip-proxy-config"]; exists {
		c.mu.RUnlock()
		if cfg, ok := cached.(*config.PipProxySettings); ok {
			return cfg, nil
		}
	}
	c.mu.RUnlock()

	// Load configuration
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if cached, exists := c.singletons["pip-proxy-config"]; exists {
		if cfg, ok := cached.(*config.PipProxySettings); ok {
			return cfg, nil
		}
	}

	// For now, return empty config - will be handled by config service
	cfg := &config.PipProxySettings{}

	// Cache for future use
	c.singletons["pip-proxy-config"] = cfg

	return cfg, nil
}

// GetYumProxyConfig returns the YUM proxy configuration
func (c *Container) GetYumProxyConfig() (*config.YumProxySettings, error) {
	c.mu.RLock()

	// Check singleton cache first
	if cached, exists := c.singletons["yum-proxy-config"]; exists {
		c.mu.RUnlock()
		if cfg, ok := cached.(*config.YumProxySettings); ok {
			return cfg, nil
		}
	}
	c.mu.RUnlock()

	// Load configuration
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if cached, exists := c.singletons["yum-proxy-config"]; exists {
		if cfg, ok := cached.(*config.YumProxySettings); ok {
			return cfg, nil
		}
	}

	// For now, return empty config - will be handled by config service
	cfg := &config.YumProxySettings{}

	// Cache for future use
	c.singletons["yum-proxy-config"] = cfg

	return cfg, nil
}

// GetApkProxyConfig returns the APK proxy configuration
func (c *Container) GetApkProxyConfig() (*config.ApkProxySettings, error) {
	c.mu.RLock()

	// Check singleton cache first
	if cached, exists := c.singletons["apk-proxy-config"]; exists {
		c.mu.RUnlock()
		if cfg, ok := cached.(*config.ApkProxySettings); ok {
			return cfg, nil
		}
	}
	c.mu.RUnlock()

	// Load configuration
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if cached, exists := c.singletons["apk-proxy-config"]; exists {
		if cfg, ok := cached.(*config.ApkProxySettings); ok {
			return cfg, nil
		}
	}

	// For now, return empty config - will be handled by config service
	cfg := &config.ApkProxySettings{}

	// Cache for future use
	c.singletons["apk-proxy-config"] = cfg

	return cfg, nil
}

// ReloadConfig reloads the configuration from files
func (c *Container) ReloadConfig() error {
	c.configCacheMu.Lock()
	defer c.configCacheMu.Unlock()

	// unifiedConfigLoader가 nil인지 확인
	if c.unifiedConfigLoader == nil {
		c.logger.Error("Unified config loader is nil, cannot reload config")
		return fmt.Errorf("unified config loader is nil")
	}

	// 새 설정 로드
	newConfig, err := c.unifiedConfigLoader.LoadConfig()
	if err != nil {
		c.logger.Error("Failed to reload unified config", logging.F("error", err))
		return err
	}

	// 기존 설정과 비교하여 변경된 경우에만 업데이트
	if c.unifiedConfig == nil || !c.isConfigEqual(c.unifiedConfig, newConfig) {
		oldConfig := c.unifiedConfig
		c.unifiedConfig = newConfig

		c.logger.Info("Unified configuration reloaded successfully")

		// 변경 콜백 실행
		c.notifyConfigChange(newConfig)

		// 새로운 알림 시스템을 통한 알림
		if c.configNotifier != nil {
			c.configNotifier.NotifyChange(newConfig)
		}

		// 서비스 팩토리에 설정 변경 알림
		if c.serviceFactory != nil {
			c.serviceFactory.ReloadServices()
		}

		if oldConfig == nil {
			c.logger.Info("Initial unified configuration loaded")
		} else {
			c.logger.Info("Unified configuration updated")
		}
	}

	return nil
}

// AddConfigChangeCallback adds a callback to be called when config changes
func (c *Container) AddConfigChangeCallback(callback func(interface{})) {
	c.configCacheMu.Lock()
	defer c.configCacheMu.Unlock()
	c.configChangeCbs = append(c.configChangeCbs, callback)

	// 새로운 알림 시스템에도 추가
	c.configNotifier.AddListener(callback)
}

// GetConfigNotifier returns the config change notifier
func (c *Container) GetConfigNotifier() *ConfigChangeNotifier {
	return c.configNotifier
}

// startConfigWatcher starts watching for configuration file changes
func (c *Container) startConfigWatcher() {
	// unifiedConfigLoader가 nil인지 확인
	if c.unifiedConfigLoader == nil {
		c.logger.Error("Unified config loader not initialized, cannot start file watcher")
		return
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		c.logger.Error("Failed to create config watcher", logging.F("error", err))
		return
	}

	c.configWatcher = watcher

	// 설정 디렉토리 감시
	err = watcher.Add(c.config.ConfigDir)
	if err != nil {
		c.logger.Error("Failed to watch config directory", logging.F("error", err))
		return
	}

	// 고루틴에서 파일 변경 감지
	go func() {
		defer func() { _ = watcher.Close() }()

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// YAML 파일 변경 감지
				if event.Op&fsnotify.Write == fsnotify.Write &&
					(strings.HasSuffix(event.Name, ".yaml") || strings.HasSuffix(event.Name, ".yml")) {
					c.logger.Info("Config file changed", logging.F("file", event.Name))

					// 약간의 지연 후 리로드 (파일 쓰기 완료 대기)
					time.Sleep(100 * time.Millisecond)

					// unifiedConfigLoader가 여전히 유효한지 확인
					c.configCacheMu.RLock()
					if c.unifiedConfigLoader != nil && c.configNotifier != nil {
						c.configCacheMu.RUnlock()
						if err := c.ReloadConfig(); err != nil {
							c.logger.Error("Failed to reload unified config after file change",
								logging.F("error", err), logging.F("file", event.Name))
						}
					} else {
						c.configCacheMu.RUnlock()
						c.logger.Warn("Unified config loader or notifier is nil, skipping reload")
						return // 정리되었으므로 고루틴 종료
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				c.logger.Error("Config watcher error", logging.F("error", err))
			}
		}
	}()
}

// notifyConfigChange notifies all registered callbacks about config changes
func (c *Container) notifyConfigChange(config interface{}) {
	for _, callback := range c.configChangeCbs {
		go func(cb func(interface{})) {
			defer func() {
				if r := recover(); r != nil {
					c.logger.Error("Config change callback panicked", logging.F("panic", r))
				}
			}()
			cb(config)
		}(callback)
	}
}

// isConfigEqual compares two configurations for equality (simplified check)
func (c *Container) isConfigEqual(old, newVal interface{}) bool {
	// 간단한 포인터 비교 (실제로는 더 정교한 비교가 필요할 수 있음)
	if old == nil && newVal == nil {
		return true
	}
	if old == nil || newVal == nil {
		return false
	}
	// For now, assume configs are different if they're different instances
	// TODO: Implement deep comparison if needed
	return old == newVal
}

// GetCacheRepository returns the cache repository instance
func (c *Container) GetCacheRepository() (cache.Repository, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cacheRepo != nil {
		return c.cacheRepo, nil
	}

	// Create cache repository
	repo, err := cache.NewFileRepository(
		c.config.StorageDir,
		10*1024*1024*1024, // 10GB max cache size
		c.config.CacheMaxAge,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache repository: %w", err)
	}

	c.cacheRepo = repo
	return repo, nil
}

// GetConfigRepository returns the config repository instance
func (c *Container) GetConfigRepository() (configRepo.Repository, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.configRepo != nil {
		return c.configRepo, nil
	}

	// Create config repository
	repo, err := configRepo.NewFileRepository(c.config.ConfigDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create config repository: %w", err)
	}

	c.configRepo = repo
	return repo, nil
}

// GetCacheService returns the cache service instance
func (c *Container) GetCacheService() (proxy.CacheService, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cacheService != nil {
		return c.cacheService, nil
	}

	// Get cache repository
	repo, err := c.GetCacheRepository()
	if err != nil {
		return nil, err
	}

	// Create cache service adapter
	c.cacheService = adapters.NewCacheAdapter(repo, c.config.CacheMaxAge)
	return c.cacheService, nil
}

// GetConfigService returns the config service instance
func (c *Container) GetConfigService() (proxy.ConfigService, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.configService != nil {
		return c.configService, nil
	}

	// Get config repository
	repo, err := c.GetConfigRepository()
	if err != nil {
		return nil, err
	}

	// Create config service adapter
	c.configService = adapters.NewConfigAdapter(repo)
	return c.configService, nil
}

// GetUpstreamClient returns the upstream client instance
func (c *Container) GetUpstreamClient() proxy.UpstreamClient {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.upstreamClient != nil {
		return c.upstreamClient
	}

	// Create HTTP upstream client
	c.upstreamClient = adapters.NewHTTPUpstreamClient(30 * time.Second)
	return c.upstreamClient
}

// GetServiceFactory returns the proxy service factory
func (c *Container) GetServiceFactory() (*proxy.ServiceFactory, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.serviceFactory != nil {
		return c.serviceFactory, nil
	}

	// Get dependencies
	cacheService, err := c.GetCacheService()
	if err != nil {
		return nil, err
	}

	configService, err := c.GetConfigService()
	if err != nil {
		return nil, err
	}

	upstreamClient := c.GetUpstreamClient()

	// Create service factory
	c.serviceFactory = proxy.NewServiceFactory(
		cacheService,
		configService,
		upstreamClient,
	)

	return c.serviceFactory, nil
}

// GetHandlerFactory returns the proxy handler factory
func (c *Container) GetHandlerFactory() (types.ProxyHandlerFactory, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.handlerFactory != nil {
		return c.handlerFactory, nil
	}

	// Get service factory
	_, err := c.GetServiceFactory()
	if err != nil {
		return nil, err
	}

	// Create handler factory
	factory := types.NewStandardProxyHandlerFactory()

	// Register all handlers (commented out until handlers are implemented)
	/*
		// Maven
		if err := factory.RegisterHandler(
			types.ProxyTypeMaven,
			proxyHandlers.MavenHandlerCreator(serviceFactory),
		); err != nil {
			return nil, fmt.Errorf("failed to register Maven handler: %w", err)
		}

		// APT
		if err := factory.RegisterHandler(
			types.ProxyTypeAPT,
			proxyHandlers.AptHandlerCreator(serviceFactory),
		); err != nil {
			return nil, fmt.Errorf("failed to register APT handler: %w", err)
		}

		// NPM
		if err := factory.RegisterHandler(
			types.ProxyTypeNPM,
			proxyHandlers.NpmHandlerCreator(serviceFactory),
		); err != nil {
			return nil, fmt.Errorf("failed to register NPM handler: %w", err)
		}
	*/

	// NOTE: Handler registration superseded by ProxyFactoryV3
	// All 7 package manager handlers (apt, maven, npm, docker, pip, yum, apk) are now
	// registered automatically via ProxyFactoryV3.registerDefaultHandlers()
	// See: internal/adapters/http/fiber/handlers/proxy/proxy_factory_v3.go
	//
	// The V3 factory pattern provides:
	// - Automatic handler registration at initialization
	// - Runtime handler availability checking
	// - Simplified routing via unified endpoint: /v3/proxy/:type/*
	//
	// This legacy factory approach is maintained for backward compatibility but is
	// no longer the primary method for handler management.

	c.handlerFactory = factory
	return factory, nil
}

// GetHandlerAdapterFactory returns the handler adapter factory instance
func (c *Container) GetHandlerAdapterFactory() (*factory.HandlerAdapterFactory, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.handlerAdapterFactory != nil {
		return c.handlerAdapterFactory, nil
	}

	// Create handler adapter factory
	c.handlerAdapterFactory = factory.NewHandlerAdapterFactory()

	c.logger.Info("Handler adapter factory created")
	return c.handlerAdapterFactory, nil
}

// GetContainerProxyHandlerFactory Container 기반 프록시 핸들러 팩토리 반환
func (c *Container) GetContainerProxyHandlerFactory() (*handlers.StandardProxyHandlerFactory, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 싱글톤에서 확인
	if cached, exists := c.singletons["container-proxy-handler-factory"]; exists {
		if factory, ok := cached.(*handlers.StandardProxyHandlerFactory); ok {
			return factory, nil
		}
	}

	// 새로운 팩토리 생성
	factory := handlers.NewStandardProxyHandlerFactory()

	// Container 기반 핸들러들 등록
	if err := c.registerContainerProxyHandlers(factory); err != nil {
		return nil, fmt.Errorf("failed to register container proxy handlers: %w", err)
	}

	// 싱글톤에 저장
	c.singletons["container-proxy-handler-factory"] = factory

	c.logger.Info("Container proxy handler factory created with handlers",
		logging.F("supported_types", factory.SupportedTypes()),
	)

	return factory, nil
}

// registerContainerProxyHandlers Container 기반 프록시 핸들러들을 팩토리에 등록
func (c *Container) registerContainerProxyHandlers(factory *handlers.StandardProxyHandlerFactory) error {
	containerMetrics := metrics.GetContainerMetrics()
	// APT 핸들러 등록
	if err := factory.RegisterHandler("apt", func(provider container.ContainerProvider) (handlers.ContainerProxyHandler, error) { //nolint:lll
		containerMetrics.RecordHandlerFactoryOperation("create", "apt", true)
		// 여기서 별도 패키지의 핸들러를 생성하고 인터페이스로 반환
		return &containerHandlerAdapter{
			name:      "apt-container-handler",
			proxyType: "apt",
			provider:  provider,
		}, nil
	}); err != nil {
		containerMetrics.RecordHandlerFactoryOperation("register", "apt", false)
		return fmt.Errorf("failed to register APT handler: %w", err)
	}
	containerMetrics.RecordHandlerFactoryOperation("register", "apt", true)

	// Maven 핸들러 등록
	if err := factory.RegisterHandler("maven", func(provider container.ContainerProvider) (handlers.ContainerProxyHandler, error) { //nolint:lll
		containerMetrics.RecordHandlerFactoryOperation("create", "maven", true)
		return &containerHandlerAdapter{
			name:      "maven-container-handler",
			proxyType: "maven",
			provider:  provider,
		}, nil
	}); err != nil {
		containerMetrics.RecordHandlerFactoryOperation("register", "maven", false)
		return fmt.Errorf("failed to register Maven handler: %w", err)
	}
	containerMetrics.RecordHandlerFactoryOperation("register", "maven", true)

	// NPM 핸들러 등록
	if err := factory.RegisterHandler("npm", func(provider container.ContainerProvider) (handlers.ContainerProxyHandler, error) { //nolint:lll
		containerMetrics.RecordHandlerFactoryOperation("create", "npm", true)
		return &containerHandlerAdapter{
			name:      "npm-container-handler",
			proxyType: "npm",
			provider:  provider,
		}, nil
	}); err != nil {
		containerMetrics.RecordHandlerFactoryOperation("register", "npm", false)
		return fmt.Errorf("failed to register NPM handler: %w", err)
	}
	containerMetrics.RecordHandlerFactoryOperation("register", "npm", true)

	// Docker 핸들러 등록
	if err := factory.RegisterHandler("docker", func(provider container.ContainerProvider) (handlers.ContainerProxyHandler, error) { //nolint:lll
		containerMetrics.RecordHandlerFactoryOperation("create", "docker", true)
		return &containerHandlerAdapter{
			name:      "docker-container-handler",
			proxyType: "docker",
			provider:  provider,
		}, nil
	}); err != nil {
		containerMetrics.RecordHandlerFactoryOperation("register", "docker", false)
		return fmt.Errorf("failed to register Docker handler: %w", err)
	}
	containerMetrics.RecordHandlerFactoryOperation("register", "docker", true)

	// PIP 핸들러 등록
	if err := factory.RegisterHandler("pip", func(provider container.ContainerProvider) (handlers.ContainerProxyHandler, error) { //nolint:lll
		containerMetrics.RecordHandlerFactoryOperation("create", "pip", true)
		return &containerHandlerAdapter{
			name:      "pip-container-handler",
			proxyType: "pip",
			provider:  provider,
		}, nil
	}); err != nil {
		containerMetrics.RecordHandlerFactoryOperation("register", "pip", false)
		return fmt.Errorf("failed to register PIP handler: %w", err)
	}
	containerMetrics.RecordHandlerFactoryOperation("register", "pip", true)

	// YUM 핸들러 등록
	if err := factory.RegisterHandler("yum", func(provider container.ContainerProvider) (handlers.ContainerProxyHandler, error) { //nolint:lll
		containerMetrics.RecordHandlerFactoryOperation("create", "yum", true)
		return &containerHandlerAdapter{
			name:      "yum-container-handler",
			proxyType: "yum",
			provider:  provider,
		}, nil
	}); err != nil {
		containerMetrics.RecordHandlerFactoryOperation("register", "yum", false)
		return fmt.Errorf("failed to register YUM handler: %w", err)
	}
	containerMetrics.RecordHandlerFactoryOperation("register", "yum", true)

	// APK 핸들러 등록
	if err := factory.RegisterHandler("apk", func(provider container.ContainerProvider) (handlers.ContainerProxyHandler, error) { //nolint:lll
		containerMetrics.RecordHandlerFactoryOperation("create", "apk", true)
		return &containerHandlerAdapter{
			name:      "apk-container-handler",
			proxyType: "apk",
			provider:  provider,
		}, nil
	}); err != nil {
		containerMetrics.RecordHandlerFactoryOperation("register", "apk", false)
		return fmt.Errorf("failed to register APK handler: %w", err)
	}
	containerMetrics.RecordHandlerFactoryOperation("register", "apk", true)

	return nil
}

// containerHandlerAdapter Container 핸들러 어댑터
type containerHandlerAdapter struct {
	name      string
	proxyType string
	provider  container.ContainerProvider
}

func (a *containerHandlerAdapter) Handle(c *fiber.Ctx) error {
	// 실제 Container 기반 핸들러 생성 및 호출
	switch a.proxyType {
	case "apt":
		handler := containerhandlers.NewAPTContainerHandler(a.provider)
		return handler.Handle(c)
	case "maven":
		handler := containerhandlers.NewMavenContainerHandler(a.provider)
		return handler.Handle(c)
	case "npm":
		handler := containerhandlers.NewNPMContainerHandler(a.provider)
		return handler.Handle(c)
	case "docker":
		handler := containerhandlers.NewDockerContainerHandler(a.provider)
		return handler.Handle(c)
	case "pip":
		handler := containerhandlers.NewPIPContainerHandler(a.provider)
		return handler.Handle(c)
	case "yum":
		handler := containerhandlers.NewYUMContainerHandler(a.provider)
		return handler.Handle(c)
	case "apk":
		handler := containerhandlers.NewAPKContainerHandler(a.provider)
		return handler.Handle(c)
	default:
		return c.Status(fiber.StatusNotImplemented).SendString(
			fmt.Sprintf("%s handler not implemented yet", a.proxyType),
		)
	}
}

func (a *containerHandlerAdapter) Name() string {
	return a.name
}

func (a *containerHandlerAdapter) Type() string {
	return a.proxyType
}

func (a *containerHandlerAdapter) SetContainer(provider container.ContainerProvider) {
	a.provider = provider
}

func (a *containerHandlerAdapter) GetContainer() container.ContainerProvider {
	return a.provider
}

func (a *containerHandlerAdapter) LoadConfig() error {
	return nil
}

func (a *containerHandlerAdapter) ReloadConfig() error {
	return nil
}

func (a *containerHandlerAdapter) IsEnabled() bool {
	return true
}

func (a *containerHandlerAdapter) GenerateCacheKey(c *fiber.Ctx) string {
	return fmt.Sprintf("%s_%s_%s", a.proxyType, c.Method(), c.Path())
}

func (a *containerHandlerAdapter) BuildUpstreamURL(c *fiber.Ctx) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (a *containerHandlerAdapter) TransformRequest(c *fiber.Ctx, upstreamReq *fiber.Agent) error {
	return nil
}

func (a *containerHandlerAdapter) TransformResponse(resp []byte, c *fiber.Ctx) ([]byte, error) {
	return resp, nil
}

func (a *containerHandlerAdapter) ShouldCache(c *fiber.Ctx, statusCode int) bool {
	return statusCode == fiber.StatusOK
}

func (a *containerHandlerAdapter) GetCacheTTL(c *fiber.Ctx) time.Duration {
	return time.Hour
}

func (a *containerHandlerAdapter) HandleError(err error, c *fiber.Ctx) error {
	return err
}

func (a *containerHandlerAdapter) HealthCheck() error {
	return nil
}

func (a *containerHandlerAdapter) IsCacheable(c *fiber.Ctx) bool {
	return c.Method() == fiber.MethodGet
}

func (a *containerHandlerAdapter) GetCacheKey(c *fiber.Ctx) string {
	return a.GenerateCacheKey(c)
}

func (a *containerHandlerAdapter) GetUpstreamURL(c *fiber.Ctx) (string, error) {
	return a.BuildUpstreamURL(c)
}

func (a *containerHandlerAdapter) ModifyRequest(c *fiber.Ctx) error {
	return nil
}

func (a *containerHandlerAdapter) ModifyResponse(c *fiber.Ctx) error {
	return nil
}

// GetUnifiedRouter returns the unified proxy router (placeholder)
// NOTE: This method is deprecated. Routing is now handled by ProxyFactoryV3.
//
// Current Implementation:
// - ProxyFactoryV3 provides unified routing via GetGlobalFactoryV3()
// - All 7 package managers registered automatically
// - Routes configured in: internal/adapters/http/fiber/routers/proxy_router_v3.go
// - Unified endpoint: /v3/proxy/:type/* handles all proxy requests
// - Handler selection is automatic based on :type parameter
//
// See:
// - internal/adapters/http/fiber/handlers/proxy/proxy_factory_v3.go
// - internal/adapters/http/fiber/handlers/proxy/unified_proxy_handler_v3.go
// - internal/adapters/http/fiber/routers/proxy_router_v3.go
func (c *Container) GetUnifiedRouter() (interface{}, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Deprecated: Use ProxyFactoryV3.GetGlobalFactoryV3() instead
	return nil, fmt.Errorf("unified router not implemented yet - use ProxyFactoryV3 pattern")
}

// GetProxyService returns the hexagonal architecture ProxyService instance
// Note: ProxyService is stored as interface{} in singletons to avoid circular import
// Returns nil if ProxyService has not been initialized
func (c *Container) GetProxyService() interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if proxyService, exists := c.singletons["proxy-service"]; exists {
		return proxyService
	}

	return nil
}

// SetProxyService stores the ProxyService instance in the container
// This should be called during application initialization
func (c *Container) SetProxyService(proxyService interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.singletons["proxy-service"] = proxyService
	c.logger.Info("ProxyService registered in container")
}

// GetOrCreateProxyService creates and registers ProxyService with NoOp adapters
// This method is used for gradual hexagonal architecture migration
func (c *Container) GetOrCreateProxyService() (interface{}, error) {
	// Check if already created
	if ps := c.GetProxyService(); ps != nil {
		if ps == serviceStatusPending {
			c.logger.Debug("ProxyService creation is pending")
			return nil, fmt.Errorf("ProxyService creation is pending")
		}
		c.logger.Debug("ProxyService already exists in container")
		return ps, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring lock
	if ps := c.singletons["proxy-service"]; ps != nil {
		if ps == "pending" {
			return nil, fmt.Errorf("ProxyService creation is pending")
		}
		return ps, nil
	}

	c.logger.Info("ProxyService will be created via InitializeProxyService()")
	c.singletons["proxy-service"] = "pending"

	return nil, nil
}

// InitializeProxyService creates ProxyService with all dependencies
// This is called from app.initializeServices() to avoid circular imports
func (c *Container) InitializeProxyService() error {
	// Delegate to proxy_service_initializer.go to wire all dependencies
	return c.InitializeProxyServiceWithStubs()
}

// ContainerProvider 인터페이스 구현
func (c *Container) GetStorageDir() string {
	return c.config.StorageDir
}

func (c *Container) GetConfigDir() string {
	return c.config.ConfigDir
}

// GetSingleton returns a singleton instance by key
func (c *Container) GetSingleton(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	instance, exists := c.singletons[key]
	return instance, exists
}

// SetSingleton stores a singleton instance by key
func (c *Container) SetSingleton(key string, instance interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.singletons[key] = instance
}

// ProvideConfigLoader provides the hexagonal architecture config loader.
// This is the new way to access configuration following ports and adapters pattern.
func (c *Container) ProvideConfigLoader() (ports.ConfigLoader, error) {
	c.mu.RLock()
	if loader, exists := c.singletons["config-loader"]; exists {
		c.mu.RUnlock()
		return loader.(ports.ConfigLoader), nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if loader, exists := c.singletons["config-loader"]; exists {
		return loader.(ports.ConfigLoader), nil
	}

	// Determine config file path from config directory
	configPath := c.config.ConfigDir + "/config.yaml"

	// Create new unified config loader
	loader, err := configadapter.NewUnifiedConfigLoader(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create config loader: %w", err)
	}

	c.singletons["config-loader"] = loader
	c.logger.Info("Config loader initialized", logging.F("path", configPath))

	return loader, nil
}

// ProvideConfigWatcher provides the configuration file watcher.
func (c *Container) ProvideConfigWatcher() (ports.ConfigWatcher, error) {
	c.mu.RLock()
	if watcher, exists := c.singletons["config-watcher"]; exists {
		c.mu.RUnlock()
		return watcher.(ports.ConfigWatcher), nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if watcher, exists := c.singletons["config-watcher"]; exists {
		return watcher.(ports.ConfigWatcher), nil
	}

	// Get config loader first
	loader, err := c.ProvideConfigLoader()
	if err != nil {
		return nil, fmt.Errorf("failed to get config loader for watcher: %w", err)
	}

	// Determine config file path from config directory
	configPath := c.config.ConfigDir + "/config.yaml"

	// Create file watcher
	watcher, err := configadapter.NewFileWatcher(configPath, loader, 500*time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("failed to create config watcher: %w", err)
	}

	// Start the watcher
	if err := watcher.Start(); err != nil {
		return nil, fmt.Errorf("failed to start config watcher: %w", err)
	}

	c.singletons["config-watcher"] = watcher
	c.logger.Info("Config watcher initialized and started", logging.F("path", configPath))

	return watcher, nil
}

// GetRootConfig returns the current RootConfig using the new hexagonal architecture.
// This replaces GetUnifiedConfig() for type-safe config access.
func (c *Container) GetRootConfig() (*config.RootConfig, error) {
	loader, err := c.ProvideConfigLoader()
	if err != nil {
		return nil, fmt.Errorf("failed to get config loader: %w", err)
	}

	cfg := loader.GetCurrent()
	if cfg == nil {
		return nil, fmt.Errorf("no configuration loaded")
	}

	return cfg, nil
}

// ProvideHotReloadManager provides the hot reload manager and connects it to the config loader.
// This bridges the existing hot reload system with the new hexagonal architecture.
func (c *Container) ProvideHotReloadManager() (*config.HotReloadManager, error) {
	c.mu.RLock()
	if manager, exists := c.singletons["hot-reload-manager"]; exists {
		c.mu.RUnlock()
		return manager.(*config.HotReloadManager), nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if manager, exists := c.singletons["hot-reload-manager"]; exists {
		return manager.(*config.HotReloadManager), nil
	}

	// Create hot reload manager
	manager := config.NewHotReloadManager()

	// Register default handlers
	manager.RegisterHandler(config.NewLoggingReloadHandler())
	// Add other default handlers as needed (cache, metrics, security, webhook, proxy, custom)

	// Create adapter to connect hot reload manager with config loader observer pattern
	adapter := configadapter.NewHotReloadAdapter(manager)

	// Get config loader and subscribe the adapter
	loader, err := c.ProvideConfigLoader()
	if err != nil {
		return nil, fmt.Errorf("failed to get config loader for hot reload: %w", err)
	}

	// Config loader also implements ConfigProvider which has Subscribe method
	provider, ok := loader.(ports.ConfigProvider)
	if !ok {
		return nil, fmt.Errorf("config loader does not implement ConfigProvider interface")
	}

	provider.Subscribe(adapter)

	c.singletons["hot-reload-manager"] = manager
	c.logger.Info("Hot reload manager initialized and connected to config loader")

	return manager, nil
}

// Close gracefully shuts down all components
func (c *Container) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Close new hexagonal config watcher
	if watcher, exists := c.singletons["config-watcher"]; exists {
		if configWatcher, ok := watcher.(ports.ConfigWatcher); ok {
			if err := configWatcher.Stop(); err != nil {
				c.logger.Error("Failed to stop config watcher", logging.F("error", err))
			}
		}
	}

	// Close legacy config watcher (will be removed in Step 6)
	if c.configWatcher != nil {
		if err := c.configWatcher.Close(); err != nil {
			c.logger.Error("Failed to close legacy config watcher", logging.F("error", err))
		}
	}

	// Close config repository watcher
	if c.configRepo != nil {
		if closer, ok := c.configRepo.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				c.logger.Error("Failed to close config repository", logging.F("error", err))
			}
		}
	}

	// Shutdown handler adapter factory
	if c.handlerAdapterFactory != nil {
		if err := c.handlerAdapterFactory.Shutdown(); err != nil {
			c.logger.Error("Failed to shutdown handler adapter factory", logging.F("error", err))
		}
	}

	// Clear all references
	c.cacheRepo = nil
	c.configRepo = nil
	c.cacheService = nil
	c.configService = nil
	c.upstreamClient = nil
	c.serviceFactory = nil
	c.handlerFactory = nil
	c.handlerAdapterFactory = nil
	c.configWatcher = nil
	c.unifiedConfig = nil
	c.unifiedConfigLoader = nil
	c.configChangeCbs = nil

	// 설정 변경 알림자 정리
	if c.configNotifier != nil {
		c.configNotifier.ClearListeners()
		c.configNotifier = nil
	}

	// c.unifiedRouter = nil // Commented out since field is removed
	c.singletons = make(map[string]interface{})

	return nil
}
