package handlers

import (
	"fmt"
	"sync"

	"proxynd/internal/app"
	"proxynd/logging"
)

// HandlerCreator 핸들러 생성자 함수 타입
type HandlerCreator func(*app.Container) Handler

// HandlerFactoryImpl 핸들러 팩토리 구현체
type HandlerFactoryImpl struct {
	container *app.Container
	creators  map[string]HandlerCreator
	mu        sync.RWMutex
	logger    logging.Logger
}

// NewHandlerFactory 새로운 핸들러 팩토리 생성
func NewHandlerFactory(container *app.Container) *HandlerFactoryImpl {
	return &HandlerFactoryImpl{
		container: container,
		creators:  make(map[string]HandlerCreator),
		logger:    logging.GetLogger(),
	}
}

// Register 핸들러 타입과 생성자 등록
func (f *HandlerFactoryImpl) Register(handlerType string, creator HandlerCreator) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.creators[handlerType] = creator
	f.logger.Info("Handler registered",
		logging.F("type", handlerType),
		logging.F("total_types", len(f.creators)),
	)
}

// Create 핸들러 생성
func (f *HandlerFactoryImpl) Create(handlerType string) (Handler, error) {
	f.mu.RLock()
	creator, exists := f.creators[handlerType]
	f.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unknown handler type: %s", handlerType)
	}

	handler := creator(f.container)

	// 로그 기록 (handler가 nil이 아닌 경우만)
	if handler != nil {
		f.logger.Debug("Handler created",
			logging.F("type", handlerType),
			logging.F("handler_name", handler.Name()),
		)
	} else {
		f.logger.Warn("Handler created but is nil",
			logging.F("type", handlerType),
		)
	}

	return handler, nil
}

// GetSupportedTypes 지원되는 핸들러 타입 목록 반환
func (f *HandlerFactoryImpl) GetSupportedTypes() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	types := make([]string, 0, len(f.creators))
	for handlerType := range f.creators {
		types = append(types, handlerType)
	}
	return types
}

// IsSupported 특정 핸들러 타입이 지원되는지 확인
func (f *HandlerFactoryImpl) IsSupported(handlerType string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	_, exists := f.creators[handlerType]
	return exists
}

// Unregister 핸들러 타입 등록 해제
func (f *HandlerFactoryImpl) Unregister(handlerType string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.creators[handlerType]; exists {
		delete(f.creators, handlerType)
		f.logger.Info("Handler unregistered",
			logging.F("type", handlerType),
			logging.F("total_types", len(f.creators)),
		)
		return true
	}
	return false
}

// Clear 모든 핸들러 타입 등록 해제
func (f *HandlerFactoryImpl) Clear() {
	f.mu.Lock()
	defer f.mu.Unlock()

	count := len(f.creators)
	f.creators = make(map[string]HandlerCreator)

	f.logger.Info("All handlers cleared",
		logging.F("cleared_count", count),
	)
}

// GetStats 팩토리 통계 반환
func (f *HandlerFactoryImpl) GetStats() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return map[string]interface{}{
		"total_types":      len(f.creators),
		"supported_types":  f.GetSupportedTypes(),
		"container_status": f.container != nil,
	}
}

// DefaultHandlerFactory 기본 핸들러 팩토리 (싱글톤)
var (
	defaultFactory     *HandlerFactoryImpl
	defaultFactoryOnce sync.Once
)

// GetDefaultFactory 기본 핸들러 팩토리 반환
func GetDefaultFactory(container *app.Container) *HandlerFactoryImpl {
	defaultFactoryOnce.Do(func() {
		defaultFactory = NewHandlerFactory(container)
	})
	return defaultFactory
}

// RegisterDefaultHandlers 기본 핸들러들 등록
func RegisterDefaultHandlers(factory *HandlerFactoryImpl) {
	// APT 핸들러 등록 - proxy 패키지에서 import 필요
	// 실제 구현은 proxy 패키지에서 등록
	factory.Register("apt", func(container *app.Container) Handler {
		return NewBaseHandler(container, "apt-handler", "apt")
	})

	// Maven 핸들러 등록
	factory.Register("maven", func(container *app.Container) Handler {
		return NewBaseHandler(container, "maven-handler", "maven")
	})

	// NPM 핸들러 등록
	factory.Register("npm", func(container *app.Container) Handler {
		return NewBaseHandler(container, "npm-handler", "npm")
	})

	// Docker 핸들러 등록
	factory.Register("docker", func(container *app.Container) Handler {
		return NewBaseHandler(container, "docker-handler", "docker")
	})

	// PIP 핸들러 등록
	factory.Register("pip", func(container *app.Container) Handler {
		return NewBaseHandler(container, "pip-handler", "pip")
	})

	// YUM 핸들러 등록
	factory.Register("yum", func(container *app.Container) Handler {
		return NewBaseHandler(container, "yum-handler", "yum")
	})

	// APK 핸들러 등록
	factory.Register("apk", func(container *app.Container) Handler {
		return NewBaseHandler(container, "apk-handler", "apk")
	})
}
