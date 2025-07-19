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
	"proxynd/pkg/client"
	"proxynd/pkg/types"
)

// MockContainer는 테스트용 Container 구현
type MockContainer struct {
	mock.Mock
	mu               sync.RWMutex
	logger           *zap.Logger
	config           *configs.Config
	unifiedConfig    *configs.UnifiedConfig
	cacheRepository  cache.Repository
	configRepository configrepo.Repository
	cacheService     *proxy.CacheService
	configService    config.Service
	upstreamClient   client.UpstreamClient
	serviceFactory   types.ProxyServiceFactory
	handlerFactory   handlers.HandlerFactory
	singletons       map[string]interface{}
}

// NewMockContainer는 새로운 MockContainer를 생성
func NewMockContainer() *MockContainer {
	return &MockContainer{
		singletons: make(map[string]interface{}),
		logger:     zap.NewNop(), // 기본값으로 no-op logger 사용
		config:     &configs.Config{},
	}
}

// GetLogger는 logger를 반환
func (m *MockContainer) GetLogger() *zap.Logger {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.logger != nil {
		return m.logger
	}
	args := m.Called()
	return args.Get(0).(*zap.Logger)
}

// GetConfig는 설정을 반환
func (m *MockContainer) GetConfig() *configs.Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.config != nil {
		return m.config
	}
	args := m.Called()
	return args.Get(0).(*configs.Config)
}

// GetUnifiedConfig는 통합 설정을 반환
func (m *MockContainer) GetUnifiedConfig() (*configs.UnifiedConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.unifiedConfig != nil {
		return m.unifiedConfig, nil
	}
	args := m.Called()
	return args.Get(0).(*configs.UnifiedConfig), args.Error(1)
}

// GetCacheRepository는 캐시 리포지토리를 반환
func (m *MockContainer) GetCacheRepository() cache.Repository {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.cacheRepository != nil {
		return m.cacheRepository
	}
	args := m.Called()
	return args.Get(0).(cache.Repository)
}

// GetConfigRepository는 설정 리포지토리를 반환
func (m *MockContainer) GetConfigRepository() configrepo.Repository {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.configRepository != nil {
		return m.configRepository
	}
	args := m.Called()
	return args.Get(0).(configrepo.Repository)
}

// GetCacheService는 캐시 서비스를 반환
func (m *MockContainer) GetCacheService() *proxy.CacheService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.cacheService != nil {
		return m.cacheService
	}
	args := m.Called()
	return args.Get(0).(*proxy.CacheService)
}

// GetConfigService는 설정 서비스를 반환
func (m *MockContainer) GetConfigService() config.Service {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.configService != nil {
		return m.configService
	}
	args := m.Called()
	return args.Get(0).(config.Service)
}

// GetUpstreamClient는 업스트림 클라이언트를 반환
func (m *MockContainer) GetUpstreamClient() client.UpstreamClient {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.upstreamClient != nil {
		return m.upstreamClient
	}
	args := m.Called()
	return args.Get(0).(client.UpstreamClient)
}

// GetServiceFactory는 서비스 팩토리를 반환
func (m *MockContainer) GetServiceFactory() types.ProxyServiceFactory {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.serviceFactory != nil {
		return m.serviceFactory
	}
	args := m.Called()
	return args.Get(0).(types.ProxyServiceFactory)
}

// GetHandlerFactory는 핸들러 팩토리를 반환
func (m *MockContainer) GetHandlerFactory() handlers.HandlerFactory {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.handlerFactory != nil {
		return m.handlerFactory
	}
	args := m.Called()
	return args.Get(0).(handlers.HandlerFactory)
}

// GetSingleton은 싱글톤 인스턴스를 반환
func (m *MockContainer) GetSingleton(key string) interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if instance, ok := m.singletons[key]; ok {
		return instance
	}
	args := m.Called(key)
	return args.Get(0)
}

// SetSingleton은 싱글톤 인스턴스를 설정
func (m *MockContainer) SetSingleton(key string, instance interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.singletons[key] = instance
	m.Called(key, instance)
}

// Helper methods for setting test values

// SetLogger는 테스트용 logger를 설정
func (m *MockContainer) SetLogger(logger *zap.Logger) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logger = logger
}

// SetConfig는 테스트용 설정을 설정
func (m *MockContainer) SetConfig(config *configs.Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = config
}

// SetUnifiedConfig는 테스트용 통합 설정을 설정
func (m *MockContainer) SetUnifiedConfig(config *configs.UnifiedConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unifiedConfig = config
}

// SetCacheRepository는 테스트용 캐시 리포지토리를 설정
func (m *MockContainer) SetCacheRepository(repo cache.Repository) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cacheRepository = repo
}

// SetConfigRepository는 테스트용 설정 리포지토리를 설정
func (m *MockContainer) SetConfigRepository(repo configrepo.Repository) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configRepository = repo
}

// SetCacheService는 테스트용 캐시 서비스를 설정
func (m *MockContainer) SetCacheService(service *proxy.CacheService) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cacheService = service
}

// SetConfigService는 테스트용 설정 서비스를 설정
func (m *MockContainer) SetConfigService(service config.Service) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configService = service
}

// SetUpstreamClient는 테스트용 업스트림 클라이언트를 설정
func (m *MockContainer) SetUpstreamClient(client client.UpstreamClient) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.upstreamClient = client
}

// SetServiceFactory는 테스트용 서비스 팩토리를 설정
func (m *MockContainer) SetServiceFactory(factory types.ProxyServiceFactory) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.serviceFactory = factory
}

// SetHandlerFactory는 테스트용 핸들러 팩토리를 설정
func (m *MockContainer) SetHandlerFactory(factory handlers.HandlerFactory) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlerFactory = factory
}

// AsAppContainer는 MockContainer를 app.Container로 변환
// 이는 실제 Container가 필요한 경우를 위한 어댑터
func (m *MockContainer) AsAppContainer() *app.Container {
	// 실제 Container 타입이 필요한 경우를 위한 변환
	// 주의: 이는 제한적인 기능만 제공
	return nil
}