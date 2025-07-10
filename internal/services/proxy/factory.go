package proxy

import (
	"fmt"
	"sync"
)

// ServiceFactory creates proxy service instances
type ServiceFactory struct {
	cache          CacheService
	config         ConfigService
	upstreamClient UpstreamClient
	services       map[string]ProxyService
	mu             sync.RWMutex
}

// NewServiceFactory creates a new service factory
func NewServiceFactory(
	cache CacheService,
	config ConfigService,
	upstreamClient UpstreamClient,
) *ServiceFactory {
	return &ServiceFactory{
		cache:          cache,
		config:         config,
		upstreamClient: upstreamClient,
		services:       make(map[string]ProxyService),
	}
}

// GetService returns a proxy service for the given type
func (f *ServiceFactory) GetService(proxyType string) (ProxyService, error) {
	// Check if service already exists
	f.mu.RLock()
	if service, exists := f.services[proxyType]; exists {
		f.mu.RUnlock()
		return service, nil
	}
	f.mu.RUnlock()

	// Create new service
	f.mu.Lock()
	defer f.mu.Unlock()

	// Double-check after acquiring write lock
	if service, exists := f.services[proxyType]; exists {
		return service, nil
	}

	// Create service based on type
	var service ProxyService
	var err error

	switch proxyType {
	case "maven":
		service, err = NewMavenService(f.cache, f.config, f.upstreamClient)
	case "apt":
		service, err = NewAptService(f.cache, f.config, f.upstreamClient)
	case "npm":
		service, err = NewNpmService(f.cache, f.config, f.upstreamClient)
	case "docker":
		service, err = NewDockerService(f.cache, f.config, f.upstreamClient)
	case "pip":
		service, err = NewPipService(f.cache, f.config, f.upstreamClient)
	case "yum":
		service, err = NewYumService(f.cache, f.config, f.upstreamClient)
	case "apk":
		service, err = NewApkService(f.cache, f.config, f.upstreamClient)
	default:
		return nil, fmt.Errorf("unsupported proxy type: %s", proxyType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create %s service: %w", proxyType, err)
	}

	// Cache the service instance
	f.services[proxyType] = service

	return service, nil
}

// ReloadServices clears the service cache, forcing recreation on next request
func (f *ServiceFactory) ReloadServices() {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Clear all cached services
	f.services = make(map[string]ProxyService)
}
