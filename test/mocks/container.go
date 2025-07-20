package mocks

import (
	"sync"

	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"proxynd/configs"
	"proxynd/internal/app"
	"proxynd/internal/handlers"
	"proxynd/internal/repositories/cache"
	configrepo "proxynd/internal/repositories/config"
	"proxynd/internal/services/config"
	"proxynd/internal/services/proxy"
	"proxynd/pkg/types"
)

// MockContainer is a test implementation of the Container interface
type MockContainer struct {
	mock.Mock
	mu               sync.RWMutex
	logger           *zap.Logger
	config           *app.Config
	unifiedConfig    *configs.UnifiedConfig
	cacheRepository  cache.Repository
	configRepository configrepo.Repository
	cacheService     *proxy.CacheService
	configService    config.Service
	upstreamClient   proxy.UpstreamClient
	serviceFactory   types.ProxyHandlerFactory
	handlerFactory   handlers.HandlerFactory
	singletons       map[string]interface{}
}

// NewMockContainer creates a new MockContainer instance
func NewMockContainer() *MockContainer {
	return &MockContainer{
		singletons: make(map[string]interface{}),
		logger:     zap.NewNop(), // 기본값으로 no-op logger 사용
		config:     &app.Config{},
	}
}

// GetLogger returns the logger instance
func (m *MockContainer) GetLogger() *zap.Logger {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.logger != nil {
		return m.logger
	}
	args := m.Called()
	return args.Get(0).(*zap.Logger)
}

// GetConfig returns the configuration
func (m *MockContainer) GetConfig() *app.Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.config != nil {
		return m.config
	}
	args := m.Called()
	return args.Get(0).(*app.Config)
}

// GetUnifiedConfig returns the unified configuration
func (m *MockContainer) GetUnifiedConfig() (*configs.UnifiedConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.unifiedConfig != nil {
		return m.unifiedConfig, nil
	}
	args := m.Called()
	return args.Get(0).(*configs.UnifiedConfig), args.Error(1)
}

// GetCacheRepository returns the cache repository
func (m *MockContainer) GetCacheRepository() cache.Repository {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.cacheRepository != nil {
		return m.cacheRepository
	}
	args := m.Called()
	return args.Get(0).(cache.Repository)
}

// GetConfigRepository returns the config repository
func (m *MockContainer) GetConfigRepository() configrepo.Repository {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.configRepository != nil {
		return m.configRepository
	}
	args := m.Called()
	return args.Get(0).(configrepo.Repository)
}

// GetCacheService returns the cache service
func (m *MockContainer) GetCacheService() *proxy.CacheService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.cacheService != nil {
		return m.cacheService
	}
	args := m.Called()
	return args.Get(0).(*proxy.CacheService)
}

// GetConfigService returns the config service
func (m *MockContainer) GetConfigService() config.Service {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.configService != nil {
		return m.configService
	}
	args := m.Called()
	return args.Get(0).(config.Service)
}

// GetUpstreamClient returns the upstream client
func (m *MockContainer) GetUpstreamClient() proxy.UpstreamClient {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.upstreamClient != nil {
		return m.upstreamClient
	}
	args := m.Called()
	return args.Get(0).(proxy.UpstreamClient)
}

// GetServiceFactory returns the service factory
func (m *MockContainer) GetServiceFactory() types.ProxyHandlerFactory {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.serviceFactory != nil {
		return m.serviceFactory
	}
	args := m.Called()
	return args.Get(0).(types.ProxyHandlerFactory)
}

// GetHandlerFactory returns the handler factory
func (m *MockContainer) GetHandlerFactory() handlers.HandlerFactory {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.handlerFactory != nil {
		return m.handlerFactory
	}
	args := m.Called()
	return args.Get(0).(handlers.HandlerFactory)
}

// GetSingleton returns a singleton instance
func (m *MockContainer) GetSingleton(key string) interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if instance, ok := m.singletons[key]; ok {
		return instance
	}
	args := m.Called(key)
	return args.Get(0)
}

// SetSingleton sets a singleton instance
func (m *MockContainer) SetSingleton(key string, instance interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.singletons[key] = instance
	m.Called(key, instance)
}

// Helper methods for setting test values

// SetLogger sets the test logger
func (m *MockContainer) SetLogger(logger *zap.Logger) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logger = logger
}

// SetConfig sets the test configuration
func (m *MockContainer) SetConfig(config *app.Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = config
}

// SetUnifiedConfig sets the test unified configuration
func (m *MockContainer) SetUnifiedConfig(config *configs.UnifiedConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unifiedConfig = config
}

// SetCacheRepository sets the test cache repository
func (m *MockContainer) SetCacheRepository(repo cache.Repository) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cacheRepository = repo
}

// SetConfigRepository sets the test config repository
func (m *MockContainer) SetConfigRepository(repo configrepo.Repository) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configRepository = repo
}

// SetCacheService sets the test cache service
func (m *MockContainer) SetCacheService(service *proxy.CacheService) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cacheService = service
}

// SetConfigService sets the test config service
func (m *MockContainer) SetConfigService(service config.Service) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configService = service
}

// SetUpstreamClient sets the test upstream client
func (m *MockContainer) SetUpstreamClient(client proxy.UpstreamClient) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.upstreamClient = client
}

// SetServiceFactory sets the test service factory
func (m *MockContainer) SetServiceFactory(factory types.ProxyHandlerFactory) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.serviceFactory = factory
}

// SetHandlerFactory sets the test handler factory
func (m *MockContainer) SetHandlerFactory(factory handlers.HandlerFactory) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlerFactory = factory
}

// AsAppContainer converts MockContainer to app.Container
// This is an adapter for cases where actual Container is needed
func (m *MockContainer) AsAppContainer() *app.Container {
	// 실제 Container 타입이 필요한 경우를 위한 변환
	// 주의: 이는 제한적인 기능만 제공
	return nil
}
