package handlers

import (
	"fmt"
	"sync"

	"proxynd/internal/logging"
)

// TestHandlerFactory 테스트용 핸들러 팩토리
type TestHandlerFactory struct {
	container interface{} // 일반적인 컨테이너 인터페이스
	creators  map[string]func() TestHandler
	mu        sync.RWMutex
	logger    logging.Logger
}

// TestHandler 테스트용 핸들러 인터페이스
type TestHandler interface {
	Name() string
	Type() string
	Handle() error
}

// NewTestHandlerFactory 새로운 테스트 핸들러 팩토리 생성
func NewTestHandlerFactory(container interface{}) *TestHandlerFactory {
	return &TestHandlerFactory{
		container: container,
		creators:  make(map[string]func() TestHandler),
		logger:    logging.GetLogger(),
	}
}

// Register 핸들러 등록
func (f *TestHandlerFactory) Register(proxyType string, creator func() TestHandler) error {
	if proxyType == "" {
		return fmt.Errorf("proxy type cannot be empty")
	}
	if creator == nil {
		return fmt.Errorf("creator function cannot be nil")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.creators[proxyType] = creator
	f.logger.Info("Test handler registered", logging.F("type", proxyType))
	return nil
}

// Create 핸들러 생성
func (f *TestHandlerFactory) Create(proxyType string) (TestHandler, error) {
	f.mu.RLock()
	creator, exists := f.creators[proxyType]
	f.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unsupported proxy type: %s", proxyType)
	}

	handler := creator()
	f.logger.Info("Test handler created", logging.F("type", proxyType))
	return handler, nil
}

// GetSupportedTypes 지원하는 프록시 타입 목록 반환
func (f *TestHandlerFactory) GetSupportedTypes() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	types := make([]string, 0, len(f.creators))
	for proxyType := range f.creators {
		types = append(types, proxyType)
	}
	return types
}

// IsSupported 핸들러 지원 여부 확인
func (f *TestHandlerFactory) IsSupported(proxyType string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	_, exists := f.creators[proxyType]
	return exists
}

// Count 등록된 핸들러 개수 반환
func (f *TestHandlerFactory) Count() int {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return len(f.creators)
}
