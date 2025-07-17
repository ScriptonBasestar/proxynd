package plugins

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"proxynd/logging"
)

var (
	ErrManagerNotStarted = errors.New("plugin manager not started")
	ErrManagerStopped    = errors.New("plugin manager stopped")
)

// PluginManager 플러그인 관리자
type PluginManager struct {
	registry     PluginRegistry
	groupManager GroupManager
	plugins      map[string]Plugin
	handlers     map[string]PackageHandler
	mu           sync.RWMutex
	started      bool
	stopped      bool
	logger       logging.Logger
}

// PluginManagerConfig 플러그인 매니저 설정
type PluginManagerConfig struct {
	AutoRegisterBuiltins bool          `json:"auto_register_builtins" yaml:"auto_register_builtins"`
	HealthCheckInterval  time.Duration `json:"health_check_interval" yaml:"health_check_interval"`
	EnableHotReload      bool          `json:"enable_hot_reload" yaml:"enable_hot_reload"`
}

// NewPluginManager 새로운 플러그인 매니저 생성
func NewPluginManager(config PluginManagerConfig) *PluginManager {
	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 5 * time.Minute
	}

	return &PluginManager{
		registry:     NewPluginRegistry(),
		groupManager: NewGroupManager(GroupManagerConfig{}),
		plugins:      make(map[string]Plugin),
		handlers:     make(map[string]PackageHandler),
		logger:       logging.GetLogger(),
	}
}

// Start 플러그인 매니저 시작
func (pm *PluginManager) Start(ctx context.Context, config PluginManagerConfig) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.started {
		return nil
	}

	pm.logger.Info("Starting plugin manager")

	// 내장 플러그인 자동 등록
	if config.AutoRegisterBuiltins {
		if err := pm.registerBuiltinPlugins(); err != nil {
			return fmt.Errorf("failed to register builtin plugins: %w", err)
		}
	}

	// 헬스체크 고루틴 시작
	if config.HealthCheckInterval > 0 {
		go pm.startHealthCheckLoop(ctx, config.HealthCheckInterval)
	}

	pm.started = true
	pm.logger.Info("Plugin manager started successfully")

	return nil
}

// Stop 플러그인 매니저 중지
func (pm *PluginManager) Stop(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.stopped {
		return nil
	}

	pm.logger.Info("Stopping plugin manager")

	// 모든 핸들러 종료
	for packageType, handler := range pm.handlers {
		if err := handler.Shutdown(ctx); err != nil {
			pm.logger.Error("Failed to shutdown handler",
				logging.F("type", packageType),
				logging.F("error", err))
		}
	}

	// 모든 플러그인 언로드
	for packageType, plugin := range pm.plugins {
		if err := plugin.Unload(); err != nil {
			pm.logger.Error("Failed to unload plugin",
				logging.F("type", packageType),
				logging.F("error", err))
		}
	}

	// 레지스트리 종료
	if err := pm.registry.(*DefaultPluginRegistry).Shutdown(ctx); err != nil {
		pm.logger.Error("Failed to shutdown plugin registry", logging.F("error", err))
	}

	pm.stopped = true
	pm.logger.Info("Plugin manager stopped")

	return nil
}

// RegisterPlugin 플러그인 등록
func (pm *PluginManager) RegisterPlugin(plugin Plugin) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.stopped {
		return ErrManagerStopped
	}

	metadata := plugin.GetMetadata()
	packageType := metadata.Type

	// 이미 등록된 플러그인인지 확인
	if _, exists := pm.plugins[packageType]; exists {
		return fmt.Errorf("plugin %s already registered", packageType)
	}

	// 플러그인 등록
	if registryImpl, ok := pm.registry.(*DefaultPluginRegistry); ok {
		if err := registryImpl.RegisterPlugin(plugin); err != nil {
			return fmt.Errorf("failed to register plugin in registry: %w", err)
		}
	}

	// 핸들러 생성 및 저장
	handler := plugin.CreateHandler()
	pm.plugins[packageType] = plugin
	pm.handlers[packageType] = handler

	pm.logger.Info("Plugin registered successfully",
		logging.F("type", packageType),
		logging.F("name", metadata.Name),
		logging.F("version", metadata.Version))

	return nil
}

// UnregisterPlugin 플러그인 등록 해제
func (pm *PluginManager) UnregisterPlugin(packageType string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.stopped {
		return ErrManagerStopped
	}

	plugin, exists := pm.plugins[packageType]
	if !exists {
		return fmt.Errorf("plugin %s not found", packageType)
	}

	// 핸들러 종료
	if handler, exists := pm.handlers[packageType]; exists {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := handler.Shutdown(ctx); err != nil {
			pm.logger.Error("Failed to shutdown handler",
				logging.F("type", packageType),
				logging.F("error", err))
		}
	}

	// 레지스트리에서 제거
	if err := pm.registry.Unregister(packageType); err != nil {
		pm.logger.Error("Failed to unregister from registry",
			logging.F("type", packageType),
			logging.F("error", err))
	}

	// 플러그인 언로드
	if err := plugin.Unload(); err != nil {
		pm.logger.Error("Failed to unload plugin",
			logging.F("type", packageType),
			logging.F("error", err))
	}

	delete(pm.plugins, packageType)
	delete(pm.handlers, packageType)

	pm.logger.Info("Plugin unregistered successfully",
		logging.F("type", packageType))

	return nil
}

// GetHandler 핸들러 조회
func (pm *PluginManager) GetHandler(packageType string) (PackageHandler, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	handler, exists := pm.handlers[packageType]
	return handler, exists
}

// ListHandlers 등록된 핸들러 목록
func (pm *PluginManager) ListHandlers() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	handlers := make([]string, 0, len(pm.handlers))
	for packageType := range pm.handlers {
		handlers = append(handlers, packageType)
	}

	return handlers
}

// GetHandlersByMode 모드별 핸들러 조회
func (pm *PluginManager) GetHandlersByMode(mode OperationMode) []PackageHandler {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	handlers := make([]PackageHandler, 0)
	for _, handler := range pm.handlers {
		if handler.SupportsMode(mode) {
			handlers = append(handlers, handler)
		}
	}

	return handlers
}

// HandleRequest 요청을 적절한 핸들러로 라우팅
func (pm *PluginManager) HandleRequest(ctx *fiber.Ctx, packageType string, mode OperationMode) error {
	if !pm.started {
		return ErrManagerNotStarted
	}

	if pm.stopped {
		return ErrManagerStopped
	}

	handler, exists := pm.GetHandler(packageType)
	if !exists {
		return fiber.NewError(fiber.StatusNotFound,
			fmt.Sprintf("Handler for package type '%s' not found", packageType))
	}

	if !handler.SupportsMode(mode) {
		return fiber.NewError(fiber.StatusBadRequest,
			fmt.Sprintf("Handler for '%s' does not support %s mode", packageType, mode))
	}

	return handler.HandleRequest(ctx, mode)
}

// GetPluginMetadata 플러그인 메타데이터 조회
func (pm *PluginManager) GetPluginMetadata(packageType string) (PluginMetadata, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugin, exists := pm.plugins[packageType]
	if !exists {
		return PluginMetadata{}, false
	}

	return plugin.GetMetadata(), true
}

// HealthCheck 전체 헬스체크
func (pm *PluginManager) HealthCheck(ctx context.Context) map[string]error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	results := make(map[string]error)

	for packageType, handler := range pm.handlers {
		if err := handler.HealthCheck(ctx); err != nil {
			results[packageType] = err
		} else {
			results[packageType] = nil
		}
	}

	return results
}

// GetStatistics 플러그인 매니저 통계
func (pm *PluginManager) GetStatistics() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_plugins"] = len(pm.plugins)
	stats["total_handlers"] = len(pm.handlers)
	stats["started"] = pm.started
	stats["stopped"] = pm.stopped

	// 모드별 핸들러 수
	proxyHandlers := pm.getHandlersByModeInternal(ProxyMode)
	mirrorHandlers := pm.getHandlersByModeInternal(MirrorMode)

	stats["proxy_handlers"] = len(proxyHandlers)
	stats["mirror_handlers"] = len(mirrorHandlers)

	// 핸들러별 상태
	handlerStatus := make(map[string]string)
	for packageType, handler := range pm.handlers {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := handler.HealthCheck(ctx); err != nil {
			handlerStatus[packageType] = "unhealthy"
		} else {
			handlerStatus[packageType] = "healthy"
		}
		cancel()
	}
	stats["handler_status"] = handlerStatus

	return stats
}

// registerBuiltinPlugins 내장 플러그인 등록
func (pm *PluginManager) registerBuiltinPlugins() error {
	// 내장 플러그인들은 별도로 등록하도록 변경
	// import cycle을 피하기 위해 외부에서 RegisterBuiltinPlugin을 호출
	pm.logger.Info("Builtin plugins registration deferred to external initialization")
	return nil
}

// RegisterBuiltinPlugin 내장 플러그인 등록 (외부에서 호출)
func (pm *PluginManager) RegisterBuiltinPlugin(plugin Plugin) error {
	return pm.RegisterPlugin(plugin)
}

// startHealthCheckLoop 헬스체크 루프 시작
func (pm *PluginManager) startHealthCheckLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	pm.logger.Info("Health check loop started",
		logging.F("interval", interval))

	for {
		select {
		case <-ctx.Done():
			pm.logger.Info("Health check loop stopped due to context cancellation")
			return
		case <-ticker.C:
			pm.performHealthCheck(ctx)
		}
	}
}

// performHealthCheck 헬스체크 수행
func (pm *PluginManager) performHealthCheck(ctx context.Context) {
	results := pm.HealthCheck(ctx)

	healthyCount := 0
	for packageType, err := range results {
		if err == nil {
			healthyCount++
		} else {
			pm.logger.Warn("Handler health check failed",
				logging.F("type", packageType),
				logging.F("error", err))
		}
	}

	pm.logger.Debug("Health check completed",
		logging.F("healthy_handlers", healthyCount),
		logging.F("total_handlers", len(results)))
}

// getHandlersByModeInternal 모드별 핸들러 조회 (내부용)
func (pm *PluginManager) getHandlersByModeInternal(mode OperationMode) []PackageHandler {
	handlers := make([]PackageHandler, 0)
	for _, handler := range pm.handlers {
		if handler.SupportsMode(mode) {
			handlers = append(handlers, handler)
		}
	}
	return handlers
}
