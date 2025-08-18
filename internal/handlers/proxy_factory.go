package handlers

import (
	"fmt"
	"sync"

	"proxynd/internal/container"
	"proxynd/internal/logging"
)

// ProxyHandlerFactoryImpl 프록시 핸들러 팩토리 구현체
type ProxyHandlerFactoryImpl struct {
	container container.ContainerProvider
	handlers  map[string]func() BaseProxyHandler
	instances map[string]*BaseProxyHandlerImpl
	mutex     sync.RWMutex
	logger    logging.Logger
}

// NewProxyHandlerFactory 새로운 프록시 핸들러 팩토리 생성
func NewProxyHandlerFactory(container container.ContainerProvider) *ProxyHandlerFactoryImpl {
	factory := &ProxyHandlerFactoryImpl{
		container: container,
		handlers:  make(map[string]func() BaseProxyHandler),
		instances: make(map[string]*BaseProxyHandlerImpl),
		logger:    logging.GetLogger(),
	}

	// 기본 핸들러들 등록
	factory.registerDefaultHandlers()

	return factory
}

// registerDefaultHandlers 기본 프록시 핸들러들 등록
func (f *ProxyHandlerFactoryImpl) registerDefaultHandlers() {
	// 기본 핸들러는 외부에서 등록하도록 변경 (import cycle 방지)
	f.logger.Info("Proxy handler factory initialized")
}

// Register 프록시 핸들러 등록
func (f *ProxyHandlerFactoryImpl) Register(proxyType string, creator func() BaseProxyHandler) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.handlers[proxyType] = creator

	f.logger.Debug("Proxy handler registered",
		logging.F("proxy_type", proxyType),
	)
}

// Unregister 프록시 핸들러 등록 해제
func (f *ProxyHandlerFactoryImpl) Unregister(proxyType string) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	delete(f.handlers, proxyType)
	delete(f.instances, proxyType)

	f.logger.Debug("Proxy handler unregistered",
		logging.F("proxy_type", proxyType),
	)
}

// Create 프록시 핸들러 생성
func (f *ProxyHandlerFactoryImpl) Create(proxyType string) (*BaseProxyHandlerImpl, error) {
	f.mutex.RLock()
	creator, exists := f.handlers[proxyType]
	f.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("지원하지 않는 프록시 타입: %s", proxyType)
	}

	// 핸들러 생성
	handler := creator()
	baseHandler := NewBaseProxyHandlerImpl(f.container, handler)

	f.logger.Debug("Proxy handler created",
		logging.F("proxy_type", proxyType),
		logging.F("handler_name", baseHandler.Name()),
	)

	return baseHandler, nil
}

// CreateSingleton 싱글톤 프록시 핸들러 생성/반환
func (f *ProxyHandlerFactoryImpl) CreateSingleton(proxyType string) (*BaseProxyHandlerImpl, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	// 이미 생성된 인스턴스가 있으면 반환
	if instance, exists := f.instances[proxyType]; exists {
		f.logger.Debug("Returning existing proxy handler instance",
			logging.F("proxy_type", proxyType),
		)
		return instance, nil
	}

	// 새 인스턴스 생성
	creator, exists := f.handlers[proxyType]
	if !exists {
		return nil, fmt.Errorf("지원하지 않는 프록시 타입: %s", proxyType)
	}

	handler := creator()
	baseHandler := NewBaseProxyHandlerImpl(f.container, handler)

	// 인스턴스 캐시
	f.instances[proxyType] = baseHandler

	f.logger.Debug("Singleton proxy handler created",
		logging.F("proxy_type", proxyType),
		logging.F("handler_name", baseHandler.Name()),
	)

	return baseHandler, nil
}

// GetSupportedTypes 지원하는 프록시 타입 목록 반환
func (f *ProxyHandlerFactoryImpl) GetSupportedTypes() []string {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	types := make([]string, 0, len(f.handlers))
	for proxyType := range f.handlers {
		types = append(types, proxyType)
	}

	return types
}

// IsSupported 프록시 타입 지원 여부 확인
func (f *ProxyHandlerFactoryImpl) IsSupported(proxyType string) bool {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	_, exists := f.handlers[proxyType]
	return exists
}

// GetHandlerInfo 핸들러 정보 반환
func (f *ProxyHandlerFactoryImpl) GetHandlerInfo(proxyType string) (map[string]interface{}, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	creator, exists := f.handlers[proxyType]
	if !exists {
		return nil, fmt.Errorf("지원하지 않는 프록시 타입: %s", proxyType)
	}

	// 임시 핸들러 생성하여 정보 수집
	handler := creator()

	info := map[string]interface{}{
		"type":    handler.Type(),
		"enabled": handler.IsEnabled(),
	}

	// 캐시 지원 여부 확인
	if _, ok := handler.(CacheableProxyHandler); ok {
		info["supports_cache"] = true
	} else {
		info["supports_cache"] = false
	}

	// 인증 지원 여부 확인
	if _, ok := handler.(AuthenticatedProxyHandler); ok {
		info["supports_auth"] = true
	} else {
		info["supports_auth"] = false
	}

	// 메트릭 지원 여부 확인
	if _, ok := handler.(MetricsAwareProxyHandler); ok {
		info["supports_metrics"] = true
	} else {
		info["supports_metrics"] = false
	}

	// 헬스체크 지원 여부 확인
	if _, ok := handler.(Healthable); ok {
		info["supports_health_check"] = true
	} else {
		info["supports_health_check"] = false
	}

	return info, nil
}

// HealthCheck 모든 활성화된 핸들러의 헬스체크 수행
func (f *ProxyHandlerFactoryImpl) HealthCheck() map[string]error {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	results := make(map[string]error)

	for proxyType, creator := range f.handlers {
		handler := creator()

		// 활성화되지 않은 핸들러는 스킵
		if !handler.IsEnabled() {
			continue
		}

		// 헬스체크 지원하는 핸들러만 체크
		if healthableHandler, ok := handler.(Healthable); ok {
			if err := healthableHandler.HealthCheck(); err != nil {
				results[proxyType] = err
				f.logger.Error("Proxy handler health check failed",
					logging.F("proxy_type", proxyType),
					logging.F("error", err),
				)
			} else {
				results[proxyType] = nil
				f.logger.Debug("Proxy handler health check passed",
					logging.F("proxy_type", proxyType),
				)
			}
		}
	}

	return results
}

// Shutdown 팩토리 및 모든 인스턴스 정리
func (f *ProxyHandlerFactoryImpl) Shutdown() {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	// 모든 인스턴스 정리
	for proxyType := range f.instances {
		f.logger.Debug("Shutting down proxy handler instance",
			logging.F("proxy_type", proxyType),
		)
	}

	// 인스턴스 맵 초기화
	f.instances = make(map[string]*BaseProxyHandlerImpl)

	f.logger.Info("Proxy handler factory shutdown completed")
}

// GetStatistics 팩토리 통계 정보 반환
func (f *ProxyHandlerFactoryImpl) GetStatistics() map[string]interface{} {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	stats := map[string]interface{}{
		"total_registered_types": len(f.handlers),
		"active_instances":       len(f.instances),
		"supported_types":        f.GetSupportedTypes(),
	}

	// 각 타입별 상태 정보
	typeStats := make(map[string]interface{})
	for proxyType, creator := range f.handlers {
		handler := creator()
		typeStats[proxyType] = map[string]interface{}{
			"enabled":      handler.IsEnabled(),
			"has_instance": f.instances[proxyType] != nil,
		}
	}
	stats["type_details"] = typeStats

	return stats
}
