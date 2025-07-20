package plugins

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"proxynd/logging"
)

var (
	// ErrHandlerNotFound is returned when a handler is not found
	ErrHandlerNotFound = errors.New("handler not found")
	// ErrHandlerExists is returned when a handler already exists
	ErrHandlerExists = errors.New("handler already exists")
	// ErrInvalidHandler is returned when a handler is invalid
	ErrInvalidHandler = errors.New("invalid handler")
	// ErrRegistryClosed is returned when the registry is closed
	ErrRegistryClosed = errors.New("registry is closed")
	// ErrCircularDependency is returned when circular dependency is detected
	ErrCircularDependency = errors.New("circular dependency detected")
)

// DefaultPluginRegistry provides a default plugin registry implementation
type DefaultPluginRegistry struct {
	handlers map[string]PackageHandler
	plugins  map[string]Plugin
	metadata map[string]PluginMetadata
	mu       sync.RWMutex
	closed   bool
	logger   logging.Logger
}

// NewPluginRegistry 새로운 플러그인 레지스트리 생성
func NewPluginRegistry() PluginRegistry {
	return &DefaultPluginRegistry{
		handlers: make(map[string]PackageHandler),
		plugins:  make(map[string]Plugin),
		metadata: make(map[string]PluginMetadata),
		logger:   logging.GetLogger(),
	}
}

// Register 핸들러 등록
func (r *DefaultPluginRegistry) Register(handler PackageHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return ErrRegistryClosed
	}

	if handler == nil {
		return ErrInvalidHandler
	}

	packageType := handler.GetType()
	if packageType == "" {
		return fmt.Errorf("handler type cannot be empty")
	}

	// 이미 등록된 핸들러인지 확인
	if _, exists := r.handlers[packageType]; exists {
		return fmt.Errorf("handler for type %s: %w", packageType, ErrHandlerExists)
	}

	// 핸들러 설정 검증
	if err := r.validateHandler(handler); err != nil {
		return fmt.Errorf("handler validation failed: %w", err)
	}

	r.handlers[packageType] = handler
	r.logger.Info("Handler registered successfully",
		logging.F("type", packageType),
		logging.F("name", handler.GetName()),
		logging.F("version", handler.GetVersion()))

	return nil
}

// Unregister 핸들러 등록 해제
func (r *DefaultPluginRegistry) Unregister(packageType string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return ErrRegistryClosed
	}

	handler, exists := r.handlers[packageType]
	if !exists {
		return fmt.Errorf("handler for type %s: %w", packageType, ErrHandlerNotFound)
	}

	// 핸들러 종료
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := handler.Shutdown(ctx); err != nil {
		r.logger.Error("Failed to shutdown handler",
			logging.F("type", packageType),
			logging.F("error", err))
	}

	delete(r.handlers, packageType)
	delete(r.plugins, packageType)
	delete(r.metadata, packageType)

	r.logger.Info("Handler unregistered successfully",
		logging.F("type", packageType))

	return nil
}

// GetHandler 핸들러 조회
func (r *DefaultPluginRegistry) GetHandler(packageType string) (PackageHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handler, exists := r.handlers[packageType]
	return handler, exists
}

// ListHandlers 등록된 핸들러 목록 반환
func (r *DefaultPluginRegistry) ListHandlers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handlers := make([]string, 0, len(r.handlers))
	for packageType := range r.handlers {
		handlers = append(handlers, packageType)
	}

	return handlers
}

// GetHandlersByMode 모드별 핸들러 조회
func (r *DefaultPluginRegistry) GetHandlersByMode(mode OperationMode) []PackageHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handlers := make([]PackageHandler, 0)
	for _, handler := range r.handlers {
		if handler.SupportsMode(mode) {
			handlers = append(handlers, handler)
		}
	}

	return handlers
}

// IsRegistered 핸들러 등록 여부 확인
func (r *DefaultPluginRegistry) IsRegistered(packageType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.handlers[packageType]
	return exists
}

// GetHandlerCount 등록된 핸들러 수 반환
func (r *DefaultPluginRegistry) GetHandlerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.handlers)
}

// RegisterPlugin 플러그인 등록
func (r *DefaultPluginRegistry) RegisterPlugin(plugin Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return ErrRegistryClosed
	}

	if plugin == nil {
		return ErrInvalidHandler
	}

	metadata := plugin.GetMetadata()
	packageType := metadata.Type

	// 이미 등록된 플러그인인지 확인
	if _, exists := r.plugins[packageType]; exists {
		return fmt.Errorf("plugin for type %s: %w", packageType, ErrHandlerExists)
	}

	// 의존성 확인
	if err := r.checkDependencies(plugin); err != nil {
		return fmt.Errorf("dependency check failed: %w", err)
	}

	// 플러그인 로드
	if err := plugin.Load(); err != nil {
		return fmt.Errorf("failed to load plugin: %w", err)
	}

	// 핸들러 생성 및 등록
	handler := plugin.CreateHandler()
	if err := r.Register(handler); err != nil {
		// 플러그인 언로드
		_ = plugin.Unload()
		return fmt.Errorf("failed to register handler: %w", err)
	}

	r.plugins[packageType] = plugin
	r.metadata[packageType] = metadata

	r.logger.Info("Plugin registered successfully",
		logging.F("type", packageType),
		logging.F("name", metadata.Name),
		logging.F("version", metadata.Version))

	return nil
}

// UnregisterPlugin 플러그인 등록 해제
func (r *DefaultPluginRegistry) UnregisterPlugin(packageType string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return ErrRegistryClosed
	}

	plugin, exists := r.plugins[packageType]
	if !exists {
		return fmt.Errorf("plugin for type %s: %w", packageType, ErrHandlerNotFound)
	}

	// 핸들러 등록 해제
	if err := r.unregisterHandler(packageType); err != nil {
		return err
	}

	// 플러그인 언로드
	if err := plugin.Unload(); err != nil {
		r.logger.Error("Failed to unload plugin",
			logging.F("type", packageType),
			logging.F("error", err))
	}

	delete(r.plugins, packageType)
	delete(r.metadata, packageType)

	r.logger.Info("Plugin unregistered successfully",
		logging.F("type", packageType))

	return nil
}

// GetPluginMetadata 플러그인 메타데이터 조회
func (r *DefaultPluginRegistry) GetPluginMetadata(packageType string) (PluginMetadata, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadata, exists := r.metadata[packageType]
	return metadata, exists
}

// Shutdown 레지스트리 종료
func (r *DefaultPluginRegistry) Shutdown(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}

	r.closed = true

	// 모든 핸들러 종료
	for packageType, handler := range r.handlers {
		if err := handler.Shutdown(ctx); err != nil {
			r.logger.Error("Failed to shutdown handler during registry shutdown",
				logging.F("type", packageType),
				logging.F("error", err))
		}
	}

	// 모든 플러그인 언로드
	for packageType, plugin := range r.plugins {
		if err := plugin.Unload(); err != nil {
			r.logger.Error("Failed to unload plugin during registry shutdown",
				logging.F("type", packageType),
				logging.F("error", err))
		}
	}

	r.logger.Info("Plugin registry shutdown completed")
	return nil
}

// validateHandler 핸들러 유효성 검증
func (r *DefaultPluginRegistry) validateHandler(handler PackageHandler) error {
	// 기본 정보 검증
	if handler.GetType() == "" {
		return errors.New("handler type cannot be empty")
	}
	if handler.GetName() == "" {
		return errors.New("handler name cannot be empty")
	}
	if handler.GetVersion() == "" {
		return errors.New("handler version cannot be empty")
	}

	// 최소한 하나의 모드는 지원해야 함
	if !handler.SupportsMode(ProxyMode) && !handler.SupportsMode(MirrorMode) {
		return errors.New("handler must support at least one operation mode")
	}

	// 헬스체크 테스트
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := handler.HealthCheck(ctx); err != nil {
		return fmt.Errorf("handler health check failed: %w", err)
	}

	return nil
}

// checkDependencies 의존성 확인
func (r *DefaultPluginRegistry) checkDependencies(plugin Plugin) error {
	dependencies := plugin.GetDependencies()
	if len(dependencies) == 0 {
		return nil
	}

	// 순환 의존성 검사
	visited := make(map[string]bool)
	recursion := make(map[string]bool)

	var checkCircular func(string) error
	checkCircular = func(packageType string) error {
		if recursion[packageType] {
			return ErrCircularDependency
		}
		if visited[packageType] {
			return nil
		}

		visited[packageType] = true
		recursion[packageType] = true

		// 의존성 플러그인들 확인
		if dependentPlugin, exists := r.plugins[packageType]; exists {
			for _, dep := range dependentPlugin.GetDependencies() {
				if err := checkCircular(dep); err != nil {
					return err
				}
			}
		}

		recursion[packageType] = false
		return nil
	}

	metadata := plugin.GetMetadata()
	return checkCircular(metadata.Type)
}

// unregisterHandler 핸들러 등록 해제 (내부용)
func (r *DefaultPluginRegistry) unregisterHandler(packageType string) error {
	handler, exists := r.handlers[packageType]
	if !exists {
		return fmt.Errorf("handler for type %s: %w", packageType, ErrHandlerNotFound)
	}

	// 핸들러 종료
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := handler.Shutdown(ctx); err != nil {
		r.logger.Error("Failed to shutdown handler",
			logging.F("type", packageType),
			logging.F("error", err))
	}

	delete(r.handlers, packageType)
	return nil
}
