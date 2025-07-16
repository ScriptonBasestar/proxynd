package app

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	// proxyHandlers "proxynd/handlers/proxy" // Commented out until handlers are properly implemented
	"proxynd/configs"
	"proxynd/internal/repositories/cache"
	"proxynd/internal/repositories/config"
	"proxynd/internal/services/adapters"
	"proxynd/internal/services/proxy"
	"proxynd/logging"
	"proxynd/pkg/types"
)

// Container is the dependency injection container
type Container struct {
	mu     sync.RWMutex
	config *Config
	logger logging.Logger

	// 설정 캐싱 및 핫 리로드 관련 필드
	configLoader    *configs.ConfigLoader
	configCache     *configs.UnifiedConfig
	configCacheMu   sync.RWMutex
	configWatcher   *fsnotify.Watcher
	configChangeCbs []func(*configs.UnifiedConfig)
	configNotifier  *ConfigChangeNotifier

	// Repositories
	cacheRepo  cache.Repository
	configRepo config.Repository

	// Services
	cacheService   proxy.CacheService
	configService  proxy.ConfigService
	upstreamClient proxy.UpstreamClient
	serviceFactory *proxy.ServiceFactory

	// Handlers (commented out until properly implemented)
	handlerFactory types.ProxyHandlerFactory
	// unifiedRouter  *proxyHandlers.UnifiedProxyRouter

	// Singleton instances
	singletons map[string]interface{}
}

// NewContainer creates a new dependency injection container
func NewContainer(cfg *Config) *Container {
	container := &Container{
		config:          cfg,
		logger:          logging.GetLogger(),
		singletons:      make(map[string]interface{}),
		configChangeCbs: make([]func(*configs.UnifiedConfig), 0),
		configNotifier:  NewConfigChangeNotifier(),
	}
	
	// ConfigLoader 초기화 (기본 설정 파일 경로 사용)
	configPath := filepath.Join(cfg.ConfigDir, "config.yaml")
	container.configLoader = configs.NewConfigLoader(configPath)
	
	// 초기 설정 로드
	if unifiedConfig, err := container.configLoader.Load(); err == nil {
		container.configCache = unifiedConfig
	} else {
		container.logger.Error("Failed to load initial config", logging.F("error", err))
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
func (c *Container) GetUnifiedConfig() *configs.UnifiedConfig {
	c.configCacheMu.RLock()
	defer c.configCacheMu.RUnlock()
	return c.configCache
}

// ReloadConfig reloads the configuration from files
func (c *Container) ReloadConfig() error {
	c.configCacheMu.Lock()
	defer c.configCacheMu.Unlock()
	
	// configLoader가 nil인지 확인
	if c.configLoader == nil {
		c.logger.Error("Config loader is nil, cannot reload config")
		return fmt.Errorf("config loader is nil")
	}
	
	// 새 설정 로드
	newConfig, err := c.configLoader.Load()
	if err != nil {
		c.logger.Error("Failed to reload config", logging.F("error", err))
		return err
	}
	
	// 설정 검증
	if validationErrors := configs.ValidateConfig(newConfig); len(validationErrors) > 0 {
		c.logger.Error("Config validation failed", logging.F("errors", validationErrors))
		return fmt.Errorf("config validation failed: %v", validationErrors)
	}
	
	// 기존 설정과 비교하여 변경된 경우에만 업데이트
	if c.configCache == nil || !c.isConfigEqual(c.configCache, newConfig) {
		oldConfig := c.configCache
		c.configCache = newConfig
		
		c.logger.Info("Configuration reloaded successfully")
		
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
			c.logger.Info("Initial configuration loaded")
		} else {
			c.logger.Info("Configuration updated")
		}
	}
	
	return nil
}

// AddConfigChangeCallback adds a callback to be called when config changes
func (c *Container) AddConfigChangeCallback(callback func(*configs.UnifiedConfig)) {
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
	// configLoader가 nil인지 확인
	if c.configLoader == nil {
		c.logger.Error("Config loader not initialized, cannot start file watcher")
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
		defer watcher.Close()
		
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
					
					// configLoader가 여전히 유효한지 확인
					c.configCacheMu.RLock()
					if c.configLoader != nil && c.configNotifier != nil {
						c.configCacheMu.RUnlock()
						if err := c.ReloadConfig(); err != nil {
							c.logger.Error("Failed to reload config after file change", 
								logging.F("error", err), logging.F("file", event.Name))
						}
					} else {
						c.configCacheMu.RUnlock()
						c.logger.Warn("Config loader or notifier is nil, skipping reload")
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
func (c *Container) notifyConfigChange(config *configs.UnifiedConfig) {
	for _, callback := range c.configChangeCbs {
		go func(cb func(*configs.UnifiedConfig)) {
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
func (c *Container) isConfigEqual(old, new *configs.UnifiedConfig) bool {
	// 간단한 구조체 비교 (실제로는 더 정교한 비교가 필요할 수 있음)
	return old.Server.Port == new.Server.Port &&
		old.Server.Host == new.Server.Host &&
		old.Cache.Backend == new.Cache.Backend &&
		old.Logging.Level == new.Logging.Level
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
func (c *Container) GetConfigRepository() (config.Repository, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.configRepo != nil {
		return c.configRepo, nil
	}

	// Create config repository
	repo, err := config.NewFileRepository(c.config.ConfigDir)
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

	// TODO: Register other handlers (Docker, PIP, YUM, APK)

	c.handlerFactory = factory
	return factory, nil
}

// GetUnifiedRouter returns the unified proxy router (placeholder)
func (c *Container) GetUnifiedRouter() (interface{}, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// TODO: Implement unified router when handlers are ready
	/*
	if c.unifiedRouter != nil {
		return c.unifiedRouter, nil
	}

	// Get service factory
	serviceFactory, err := c.GetServiceFactory()
	if err != nil {
		return nil, err
	}

	// Create unified router
	c.unifiedRouter = proxyHandlers.NewUnifiedProxyRouter(serviceFactory)
	*/
	return nil, fmt.Errorf("unified router not implemented yet")
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

// Close gracefully shuts down all components
func (c *Container) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Close config watcher
	if c.configWatcher != nil {
		if err := c.configWatcher.Close(); err != nil {
			c.logger.Error("Failed to close config watcher", logging.F("error", err))
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

	// Clear all references
	c.cacheRepo = nil
	c.configRepo = nil
	c.cacheService = nil
	c.configService = nil
	c.upstreamClient = nil
	c.serviceFactory = nil
	c.handlerFactory = nil
	c.configWatcher = nil
	c.configCache = nil
	c.configLoader = nil
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
