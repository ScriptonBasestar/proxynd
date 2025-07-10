package app

import (
	"fmt"

	"proxynd/handlers"
	"proxynd/internal/repositories/cache"
	"proxynd/internal/repositories/config"
	"proxynd/internal/services/proxy"
	"proxynd/pkg/types"
)

// Provider is a function that provides a dependency
type Provider func(c *Container) (interface{}, error)

// Providers holds all dependency providers
type Providers struct {
	providers map[string]Provider
}

// NewProviders creates a new providers registry
func NewProviders() *Providers {
	p := &Providers{
		providers: make(map[string]Provider),
	}

	// Register all providers
	p.registerProviders()

	return p
}

// registerProviders registers all dependency providers
func (p *Providers) registerProviders() {
	// Repositories
	p.Register("cache.Repository", ProvideCacheRepository)
	p.Register("config.Repository", ProvideConfigRepository)

	// Services
	p.Register("proxy.CacheService", ProvideCacheService)
	p.Register("proxy.ConfigService", ProvideConfigService)
	p.Register("proxy.UpstreamClient", ProvideUpstreamClient)
	p.Register("proxy.ServiceFactory", ProvideServiceFactory)

	// Handlers
	p.Register("types.ProxyHandlerFactory", ProvideHandlerFactory)
	p.Register("handlers.UnifiedProxyRouter", ProvideUnifiedRouter)
}

// Register registers a provider
func (p *Providers) Register(key string, provider Provider) {
	p.providers[key] = provider
}

// Get retrieves a dependency using its provider
func (p *Providers) Get(key string, container *Container) (interface{}, error) {
	provider, exists := p.providers[key]
	if !exists {
		return nil, fmt.Errorf("no provider registered for key: %s", key)
	}

	return provider(container)
}

// Provider functions

// ProvideCacheRepository provides a cache repository
func ProvideCacheRepository(c *Container) (interface{}, error) {
	// Check singleton
	if instance, exists := c.GetSingleton("cache.Repository"); exists {
		return instance.(cache.Repository), nil
	}

	// Create new instance
	repo, err := cache.NewFileRepository(
		c.GetConfig().StorageDir,
		10*1024*1024*1024, // 10GB max cache size
		c.GetConfig().CacheMaxAge,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache repository: %w", err)
	}

	// Store as singleton
	c.SetSingleton("cache.Repository", repo)

	return repo, nil
}

// ProvideConfigRepository provides a config repository
func ProvideConfigRepository(c *Container) (interface{}, error) {
	// Check singleton
	if instance, exists := c.GetSingleton("config.Repository"); exists {
		return instance.(config.Repository), nil
	}

	// Create new instance
	repo, err := config.NewFileRepository(c.GetConfig().ConfigDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create config repository: %w", err)
	}

	// Store as singleton
	c.SetSingleton("config.Repository", repo)

	return repo, nil
}

// ProvideCacheService provides a cache service
func ProvideCacheService(c *Container) (interface{}, error) {
	// Get cache repository
	repoInterface, err := ProvideCacheRepository(c)
	if err != nil {
		return nil, err
	}

	repo := repoInterface.(cache.Repository)

	// Create cache service (not singleton - stateless)
	return c.GetCacheService()
}

// ProvideConfigService provides a config service
func ProvideConfigService(c *Container) (interface{}, error) {
	// Get config repository
	repoInterface, err := ProvideConfigRepository(c)
	if err != nil {
		return nil, err
	}

	repo := repoInterface.(config.Repository)
	_ = repo // Would be used in the adapter

	// Create config service (not singleton - stateless)
	return c.GetConfigService()
}

// ProvideUpstreamClient provides an upstream client
func ProvideUpstreamClient(c *Container) (interface{}, error) {
	// Upstream client is stateless, return new instance
	return c.GetUpstreamClient(), nil
}

// ProvideServiceFactory provides a service factory
func ProvideServiceFactory(c *Container) (interface{}, error) {
	// Check singleton
	if instance, exists := c.GetSingleton("proxy.ServiceFactory"); exists {
		return instance.(*proxy.ServiceFactory), nil
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
	factory := proxy.NewServiceFactory(
		cacheService,
		configService,
		upstreamClient,
	)

	// Store as singleton
	c.SetSingleton("proxy.ServiceFactory", factory)

	return factory, nil
}

// ProvideHandlerFactory provides a handler factory
func ProvideHandlerFactory(c *Container) (interface{}, error) {
	// Check singleton
	if instance, exists := c.GetSingleton("types.ProxyHandlerFactory"); exists {
		return instance.(types.ProxyHandlerFactory), nil
	}

	// Get service factory
	serviceFactoryInterface, err := ProvideServiceFactory(c)
	if err != nil {
		return nil, err
	}

	serviceFactory := serviceFactoryInterface.(*proxy.ServiceFactory)

	// Create handler factory
	factory := types.NewStandardProxyHandlerFactory()

	// Register all handlers
	if err := registerAllHandlers(factory, serviceFactory); err != nil {
		return nil, err
	}

	// Store as singleton
	c.SetSingleton("types.ProxyHandlerFactory", factory)

	return factory, nil
}

// ProvideUnifiedRouter provides a unified router
func ProvideUnifiedRouter(c *Container) (interface{}, error) {
	// Get service factory
	serviceFactoryInterface, err := ProvideServiceFactory(c)
	if err != nil {
		return nil, err
	}

	serviceFactory := serviceFactoryInterface.(*proxy.ServiceFactory)

	// Create unified router
	return handlers.NewUnifiedProxyRouter(serviceFactory), nil
}

// Helper function to register all handlers
func registerAllHandlers(factory types.ProxyHandlerFactory, serviceFactory *proxy.ServiceFactory) error {
	// Maven
	if err := factory.RegisterHandler(
		types.ProxyTypeMaven,
		handlers.MavenHandlerCreator(serviceFactory),
	); err != nil {
		return fmt.Errorf("failed to register Maven handler: %w", err)
	}

	// APT
	if err := factory.RegisterHandler(
		types.ProxyTypeAPT,
		handlers.AptHandlerCreator(serviceFactory),
	); err != nil {
		return fmt.Errorf("failed to register APT handler: %w", err)
	}

	// NPM
	if err := factory.RegisterHandler(
		types.ProxyTypeNPM,
		handlers.NpmHandlerCreator(serviceFactory),
	); err != nil {
		return fmt.Errorf("failed to register NPM handler: %w", err)
	}

	// TODO: Register other handlers (Docker, PIP, YUM, APK)

	return nil
}
