package docker

import (
	"proxynd/internal/domain/docker"
	"proxynd/logging"
)

// ServiceFactory Docker 서비스 팩토리
type ServiceFactory struct {
	config docker.ProxyConfig
	logger logging.Logger
}

// NewServiceFactory ServiceFactory 생성자
func NewServiceFactory(config docker.ProxyConfig, logger logging.Logger) *ServiceFactory {
	return &ServiceFactory{
		config: config,
		logger: logger,
	}
}

// CreateRegistryService RegistryService 생성
func (f *ServiceFactory) CreateRegistryService() docker.RegistryHandler {
	manifestManager := f.CreateManifestManager()
	blobManager := f.CreateBlobManager()
	authManager := f.CreateAuthenticationManager()
	registryManager := f.CreateRegistryManager()
	cacheManager := f.CreateCacheManager()
	metricsCollector := f.CreateMetricsCollector()

	return NewRegistryService(
		f.config,
		manifestManager,
		blobManager,
		authManager,
		registryManager,
		cacheManager,
		metricsCollector,
		f.logger,
	)
}

// CreateManifestManager ManifestManager 생성
func (f *ServiceFactory) CreateManifestManager() docker.ManifestManager {
	registryManager := f.CreateRegistryManager()
	authManager := f.CreateAuthenticationManager()
	cacheManager := f.CreateCacheManager()

	return NewManifestManager(
		f.config,
		registryManager,
		authManager,
		cacheManager,
		f.logger,
	)
}

// CreateBlobManager BlobManager 생성
func (f *ServiceFactory) CreateBlobManager() docker.BlobManager {
	registryManager := f.CreateRegistryManager()
	authManager := f.CreateAuthenticationManager()
	cacheManager := f.CreateCacheManager()

	return NewBlobManager(
		f.config,
		registryManager,
		authManager,
		cacheManager,
		f.logger,
	)
}

// CreateAuthenticationManager AuthenticationManager 생성
func (f *ServiceFactory) CreateAuthenticationManager() docker.AuthenticationManager {
	return NewAuthenticationManager(f.config, f.logger)
}

// CreateRegistryManager RegistryManager 생성
func (f *ServiceFactory) CreateRegistryManager() docker.RegistryManager {
	return NewRegistryManager(f.config, f.logger)
}

// CreateCacheManager CacheManager 생성
func (f *ServiceFactory) CreateCacheManager() docker.CacheManager {
	return NewCacheManager(f.config, f.logger)
}

// CreateMetricsCollector MetricsCollector 생성
func (f *ServiceFactory) CreateMetricsCollector() docker.MetricsCollector {
	return NewMetricsCollector(f.config, f.logger)
}
