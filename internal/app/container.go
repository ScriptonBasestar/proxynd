package app

import (
	"fmt"
	"sync"
	"time"

	"proxynd/internal/handlers"
	"proxynd/internal/repositories/cache"
	"proxynd/internal/repositories/config"
	"proxynd/internal/services/adapters"
	"proxynd/internal/services/proxy"
	"proxynd/logging"
	"proxynd/pkg/types"
)

// Container is the dependency injection container
type Container struct {
	mu       sync.RWMutex
	config   *Config
	logger   logging.Logger
	
	// Repositories
	cacheRepo  cache.Repository
	configRepo config.Repository
	
	// Services
	cacheService   proxy.CacheService
	configService  proxy.ConfigService
	upstreamClient proxy.UpstreamClient
	serviceFactory *proxy.ServiceFactory
	
	// Handlers
	handlerFactory   types.ProxyHandlerFactory
	unifiedRouter    *handlers.UnifiedProxyRouter
	
	// Singleton instances
	singletons map[string]interface{}
}

// NewContainer creates a new dependency injection container
func NewContainer(cfg *Config) *Container {
	return &Container{
		config:     cfg,
		logger:     logging.GetLogger(),
		singletons: make(map[string]interface{}),
	}
}

// GetLogger returns the logger instance
func (c *Container) GetLogger() logging.Logger {
	return c.logger
}

// GetConfig returns the application configuration
func (c *Container) GetConfig() *Config {
	return c.config
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
	serviceFactory, err := c.GetServiceFactory()
	if err != nil {
		return nil, err
	}
	
	// Create handler factory
	factory := types.NewStandardProxyHandlerFactory()
	
	// Register all handlers
	// Maven
	if err := factory.RegisterHandler(
		types.ProxyTypeMaven,
		handlers.MavenHandlerCreator(serviceFactory),
	); err != nil {
		return nil, fmt.Errorf("failed to register Maven handler: %w", err)
	}
	
	// APT
	if err := factory.RegisterHandler(
		types.ProxyTypeAPT,
		handlers.AptHandlerCreator(serviceFactory),
	); err != nil {
		return nil, fmt.Errorf("failed to register APT handler: %w", err)
	}
	
	// NPM
	if err := factory.RegisterHandler(
		types.ProxyTypeNPM,
		handlers.NpmHandlerCreator(serviceFactory),
	); err != nil {
		return nil, fmt.Errorf("failed to register NPM handler: %w", err)
	}
	
	// TODO: Register other handlers (Docker, PIP, YUM, APK)
	
	c.handlerFactory = factory
	return factory, nil
}

// GetUnifiedRouter returns the unified proxy router
func (c *Container) GetUnifiedRouter() (*handlers.UnifiedProxyRouter, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.unifiedRouter != nil {
		return c.unifiedRouter, nil
	}
	
	// Get service factory
	serviceFactory, err := c.GetServiceFactory()
	if err != nil {
		return nil, err
	}
	
	// Create unified router
	c.unifiedRouter = handlers.NewUnifiedProxyRouter(serviceFactory)
	return c.unifiedRouter, nil
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
	c.unifiedRouter = nil
	c.singletons = make(map[string]interface{})
	
	return nil
}