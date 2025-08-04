package handlers

import (
	"fmt"
	"sync"

	"proxynd/internal/container"
	"proxynd/logging"
)

// StandardProxyHandlerFactory 표준 프록시 핸들러 팩토리 구현체
type StandardProxyHandlerFactory struct {
	mu       sync.RWMutex
	handlers map[string]func(container.ContainerProvider) (ContainerProxyHandler, error)
	logger   logging.Logger
}

// NewStandardProxyHandlerFactory 새로운 표준 프록시 핸들러 팩토리 생성
func NewStandardProxyHandlerFactory() *StandardProxyHandlerFactory {
	return &StandardProxyHandlerFactory{
		handlers: make(map[string]func(container.ContainerProvider) (ContainerProxyHandler, error)),
		logger:   logging.GetLogger(),
	}
}

// CreateHandler 특정 프록시 타입 핸들러 생성
func (f *StandardProxyHandlerFactory) CreateHandler(proxyType string, provider container.ContainerProvider) (ContainerProxyHandler, error) {
	f.mu.RLock()
	createFn, exists := f.handlers[proxyType]
	f.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unsupported proxy type: %s", proxyType)
	}

	handler, err := createFn(provider)
	if err != nil {
		f.logger.Error("Failed to create handler",
			logging.F("type", proxyType),
			logging.F("error", err),
		)
		return nil, fmt.Errorf("failed to create %s handler: %w", proxyType, err)
	}

	// Container 설정
	handler.SetContainer(provider)

	// 설정 로드
	if err := handler.LoadConfig(); err != nil {
		f.logger.Error("Failed to load config for handler",
			logging.F("type", proxyType),
			logging.F("error", err),
		)
		return nil, fmt.Errorf("failed to load config for %s handler: %w", proxyType, err)
	}

	f.logger.Info("Handler created successfully",
		logging.F("type", proxyType),
		logging.F("name", handler.Name()),
	)

	return handler, nil
}

// SupportedTypes 지원하는 프록시 타입 목록 반환
func (f *StandardProxyHandlerFactory) SupportedTypes() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	types := make([]string, 0, len(f.handlers))
	for proxyType := range f.handlers {
		types = append(types, proxyType)
	}
	return types
}

// RegisterHandler 프록시 타입별 핸들러 등록
func (f *StandardProxyHandlerFactory) RegisterHandler(proxyType string, createFn func(container.ContainerProvider) (ContainerProxyHandler, error)) error {
	if proxyType == "" {
		return fmt.Errorf("proxy type cannot be empty")
	}

	if createFn == nil {
		return fmt.Errorf("create function cannot be nil")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	// 중복 등록 체크
	if _, exists := f.handlers[proxyType]; exists {
		f.logger.Warn("Handler already registered, overriding",
			logging.F("type", proxyType),
		)
	}

	f.handlers[proxyType] = createFn

	f.logger.Info("Handler registered",
		logging.F("type", proxyType),
	)

	return nil
}

// UnregisterHandler 핸들러 등록 해제
func (f *StandardProxyHandlerFactory) UnregisterHandler(proxyType string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.handlers[proxyType]; exists {
		delete(f.handlers, proxyType)
		f.logger.Info("Handler unregistered",
			logging.F("type", proxyType),
		)
	}
}

// IsRegistered 핸들러 등록 여부 확인
func (f *StandardProxyHandlerFactory) IsRegistered(proxyType string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	_, exists := f.handlers[proxyType]
	return exists
}

// Clear 모든 핸들러 등록 해제
func (f *StandardProxyHandlerFactory) Clear() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.handlers = make(map[string]func(container.ContainerProvider) (ContainerProxyHandler, error))
	f.logger.Info("All handlers cleared")
}

// Count 등록된 핸들러 개수 반환
func (f *StandardProxyHandlerFactory) Count() int {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return len(f.handlers)
}

// unified_handler.go에서 필요한 추가 메서드들

// IsSupported 프록시 타입 지원 여부 확인
func (f *StandardProxyHandlerFactory) IsSupported(proxyType string) bool {
	return f.IsRegistered(proxyType)
}

// GetSupportedTypes SupportedTypes의 별칭
func (f *StandardProxyHandlerFactory) GetSupportedTypes() []string {
	return f.SupportedTypes()
}

// CreateSingleton 싱글톤 핸들러 생성 (현재는 일반 생성과 동일)
func (f *StandardProxyHandlerFactory) CreateSingleton(proxyType string) (ContainerProxyHandler, error) {
	// TODO: Container에서 provider 가져오기
	// 임시로 nil을 전달하지만 실제로는 container provider가 필요
	return f.CreateHandler(proxyType, nil)
}

// GetHandlerInfo 핸들러 정보 반환
func (f *StandardProxyHandlerFactory) GetHandlerInfo(proxyType string) map[string]interface{} {
	info := make(map[string]interface{})
	info["type"] = proxyType
	info["supported"] = f.IsSupported(proxyType)
	return info
}

// HealthCheck 팩토리 건강 상태 확인
func (f *StandardProxyHandlerFactory) HealthCheck() error {
	f.mu.RLock()
	count := len(f.handlers)
	f.mu.RUnlock()

	if count == 0 {
		return fmt.Errorf("no handlers registered")
	}
	return nil
}

// GetStatistics 팩토리 통계 정보 반환
func (f *StandardProxyHandlerFactory) GetStatistics() map[string]interface{} {
	stats := make(map[string]interface{})
	stats["registered_count"] = f.Count()
	stats["supported_types"] = f.SupportedTypes()
	return stats
}

// Register 단순한 핸들러 등록 (unified_handler.go용)
func (f *StandardProxyHandlerFactory) Register(proxyType string, creator func() ContainerProxyHandler) {
	// creator를 ContainerProvider를 받는 함수로 래핑
	_ = f.RegisterHandler(proxyType, func(provider container.ContainerProvider) (ContainerProxyHandler, error) {
		handler := creator()
		return handler, nil
	})
}

// Unregister 핸들러 등록 해제 (unified_handler.go용)
func (f *StandardProxyHandlerFactory) Unregister(proxyType string) {
	f.UnregisterHandler(proxyType)
}

// Shutdown 팩토리 종료
func (f *StandardProxyHandlerFactory) Shutdown() error {
	f.Clear()
	f.logger.Info("ProxyHandlerFactory shutdown completed")
	return nil
}
