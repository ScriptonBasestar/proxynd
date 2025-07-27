package factory

import (
	"fmt"
	"sync"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/config"
	"proxynd/internal/adapters/http"
	"proxynd/logging"
)

// HandlerAdapterFactory HTTP 어댑터 기반 핸들러 팩토리
type HandlerAdapterFactory struct {
	adapters         map[string]ProxyHandlerAdapter
	healthCheckMap   map[string]HealthStatus
	lifecycleManager *HandlerLifecycleManager
	mu               sync.RWMutex
	logger           logging.Logger
}

// ProxyHandlerAdapter 프록시 핸들러 어댑터 인터페이스
type ProxyHandlerAdapter interface {
	Handle(c *fiber.Ctx) error
	GetProxyType() string
	HealthCheck() error
	Initialize() error
	Shutdown() error
}

// HealthStatus 핸들러 헬스 상태
type HealthStatus struct {
	IsHealthy   bool   `json:"is_healthy"`
	LastCheck   int64  `json:"last_check"`
	ErrorCount  int    `json:"error_count"`
	LastError   string `json:"last_error,omitempty"`
	Initialized bool   `json:"initialized"`
}

// HandlerLifecycleManager 핸들러 생명주기 관리자
type HandlerLifecycleManager struct {
	initOrder    []string
	shutdownHooks map[string]func() error
	initialized  map[string]bool
	mu           sync.RWMutex
	logger       logging.Logger
}

// NewHandlerAdapterFactory 새로운 핸들러 어댑터 팩토리 생성
func NewHandlerAdapterFactory() *HandlerAdapterFactory {
	factory := &HandlerAdapterFactory{
		adapters:       make(map[string]ProxyHandlerAdapter),
		healthCheckMap: make(map[string]HealthStatus),
		logger:         logging.GetLogger(),
		lifecycleManager: &HandlerLifecycleManager{
			initOrder:     make([]string, 0),
			shutdownHooks: make(map[string]func() error),
			initialized:   make(map[string]bool),
			logger:        logging.GetLogger(),
		},
	}
	
	// 모든 프록시 타입에 대한 어댑터 등록
	factory.registerAllAdapters()
	
	return factory
}

// registerAllAdapters 모든 프록시 타입 어댑터 등록
func (f *HandlerAdapterFactory) registerAllAdapters() {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	// 초기화 순서 정의 (의존성 순서)
	f.lifecycleManager.initOrder = []string{
		"maven", "apt", "npm", "pip", "docker", "yum", "apk",
	}
	
	// 각 프록시 타입별 어댑터 팩토리 함수 등록
	adapterFactories := map[string]func() (ProxyHandlerAdapter, error){
		"maven":  f.createMavenAdapter,
		"apt":    f.createAptAdapter,
		"npm":    f.createNpmAdapter,
		"pip":    f.createPipAdapter,
		"docker": f.createDockerAdapter,
		"yum":    f.createYumAdapter,
		"apk":    f.createApkAdapter,
	}
	
	// 어댑터 생성 및 등록
	for proxyType, factory := range adapterFactories {
		adapter, err := factory()
		if err != nil {
			f.logger.Warn("Failed to create adapter",
				logging.F("proxy_type", proxyType),
				logging.F("error", err.Error()))
			
			// 헬스 상태를 unhealthy로 설정
			f.healthCheckMap[proxyType] = HealthStatus{
				IsHealthy:   false,
				LastError:   err.Error(),
				Initialized: false,
			}
			continue
		}
		
		f.adapters[proxyType] = adapter
		f.healthCheckMap[proxyType] = HealthStatus{
			IsHealthy:   true,
			Initialized: false, // 아직 초기화되지 않음
		}
		
		f.logger.Info("Handler adapter registered",
			logging.F("proxy_type", proxyType))
	}
}

// GetAdapter 프록시 타입에 해당하는 어댑터 반환
func (f *HandlerAdapterFactory) GetAdapter(proxyType string) (ProxyHandlerAdapter, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	adapter, exists := f.adapters[proxyType]
	if !exists {
		return nil, fmt.Errorf("unknown proxy type: %s", proxyType)
	}
	
	// 초기화 확인
	if !f.lifecycleManager.initialized[proxyType] {
		f.mu.RUnlock()
		if err := f.initializeAdapter(proxyType); err != nil {
			f.mu.RLock()
			return nil, fmt.Errorf("failed to initialize adapter %s: %w", proxyType, err)
		}
		f.mu.RLock()
	}
	
	return adapter, nil
}

// InitializeAll 모든 어댑터 초기화
func (f *HandlerAdapterFactory) InitializeAll() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	// 정의된 순서대로 초기화
	for _, proxyType := range f.lifecycleManager.initOrder {
		if err := f.initializeAdapterUnsafe(proxyType); err != nil {
			f.logger.Error("Failed to initialize adapter",
				logging.F("proxy_type", proxyType),
				logging.F("error", err.Error()))
			return err
		}
	}
	
	f.logger.Info("All adapters initialized successfully",
		logging.F("count", len(f.lifecycleManager.initOrder)))
	
	return nil
}

// initializeAdapter 특정 어댑터 초기화 (thread-safe)
func (f *HandlerAdapterFactory) initializeAdapter(proxyType string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.initializeAdapterUnsafe(proxyType)
}

// initializeAdapterUnsafe 특정 어댑터 초기화 (unsafe, 이미 lock된 상태에서 호출)
func (f *HandlerAdapterFactory) initializeAdapterUnsafe(proxyType string) error {
	if f.lifecycleManager.initialized[proxyType] {
		return nil // 이미 초기화됨
	}
	
	adapter, exists := f.adapters[proxyType]
	if !exists {
		return fmt.Errorf("adapter not found: %s", proxyType)
	}
	
	if err := adapter.Initialize(); err != nil {
		f.healthCheckMap[proxyType] = HealthStatus{
			IsHealthy:   false,
			LastError:   err.Error(),
			Initialized: false,
		}
		return err
	}
	
	f.lifecycleManager.initialized[proxyType] = true
	f.healthCheckMap[proxyType] = HealthStatus{
		IsHealthy:   true,
		Initialized: true,
	}
	
	f.logger.Debug("Adapter initialized",
		logging.F("proxy_type", proxyType))
	
	return nil
}

// HealthCheck 모든 어댑터 헬스체크 수행
func (f *HandlerAdapterFactory) HealthCheck() map[string]HealthStatus {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	for proxyType, adapter := range f.adapters {
		if !f.lifecycleManager.initialized[proxyType] {
			continue // 초기화되지 않은 어댑터는 스킵
		}
		
		err := adapter.HealthCheck()
		status := f.healthCheckMap[proxyType]
		
		if err != nil {
			status.IsHealthy = false
			status.LastError = err.Error()
			status.ErrorCount++
		} else {
			status.IsHealthy = true
			status.LastError = ""
		}
		
		f.healthCheckMap[proxyType] = status
	}
	
	// 헬스 상태 복사본 반환
	result := make(map[string]HealthStatus)
	for k, v := range f.healthCheckMap {
		result[k] = v
	}
	
	return result
}

// Shutdown 모든 어댑터 정리
func (f *HandlerAdapterFactory) Shutdown() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	// 역순으로 정리
	for i := len(f.lifecycleManager.initOrder) - 1; i >= 0; i-- {
		proxyType := f.lifecycleManager.initOrder[i]
		
		if adapter, exists := f.adapters[proxyType]; exists {
			if err := adapter.Shutdown(); err != nil {
				f.logger.Error("Failed to shutdown adapter",
					logging.F("proxy_type", proxyType),
					logging.F("error", err.Error()))
			}
		}
		
		f.lifecycleManager.initialized[proxyType] = false
	}
	
	f.logger.Info("All adapters shutdown completed")
	return nil
}

// GetStats 팩토리 통계 반환
func (f *HandlerAdapterFactory) GetStats() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	stats := map[string]interface{}{
		"total_adapters":    len(f.adapters),
		"initialized_count": f.getInitializedCount(),
		"healthy_count":     f.getHealthyCount(),
		"supported_types":   f.getSupportedTypes(),
		"health_status":     f.healthCheckMap,
	}
	
	return stats
}

// 어댑터 생성 팩토리 함수들

func (f *HandlerAdapterFactory) createMavenAdapter() (ProxyHandlerAdapter, error) {
	config := config.MavenProxyConfig{}
	if !config.ConfigExists() {
		return nil, fmt.Errorf("maven config not found")
	}
	if err := config.ReadConfig(); err != nil {
		return nil, fmt.Errorf("failed to read maven config: %w", err)
	}
	
	adapter := http.NewMavenBrowserAdapter(config, f.logger)
	return &mavenAdapterWrapper{adapter: adapter}, nil
}

func (f *HandlerAdapterFactory) createAptAdapter() (ProxyHandlerAdapter, error) {
	config := config.AptProxyConfig{}
	if !config.ConfigExists() {
		return nil, fmt.Errorf("apt config not found")
	}
	if err := config.ReadConfig(); err != nil {
		return nil, fmt.Errorf("failed to read apt config: %w", err)
	}
	
	adapter := http.NewAPTHandlerAdapter(config, f.logger)
	return &aptAdapterWrapper{adapter: adapter}, nil
}

func (f *HandlerAdapterFactory) createNpmAdapter() (ProxyHandlerAdapter, error) {
	config := config.NpmProxyConfig{}
	if !config.ConfigExists() {
		return nil, fmt.Errorf("npm config not found")
	}
	if err := config.ReadConfig(); err != nil {
		return nil, fmt.Errorf("failed to read npm config: %w", err)
	}
	
	adapter := http.NewNPMHandlerAdapter(config, f.logger)
	return &npmAdapterWrapper{adapter: adapter}, nil
}

func (f *HandlerAdapterFactory) createPipAdapter() (ProxyHandlerAdapter, error) {
	config := config.PipProxyConfig{}
	if !config.ConfigExists() {
		return nil, fmt.Errorf("pip config not found")
	}
	if err := config.ReadConfig(); err != nil {
		return nil, fmt.Errorf("failed to read pip config: %w", err)
	}
	
	adapter := http.NewPIPHandlerAdapter(config, f.logger)
	return &pipAdapterWrapper{adapter: adapter}, nil
}

func (f *HandlerAdapterFactory) createDockerAdapter() (ProxyHandlerAdapter, error) {
	config := config.DockerProxyConfig{}
	if !config.ConfigExists() {
		return nil, fmt.Errorf("docker config not found")
	}
	if err := config.ReadConfig(); err != nil {
		return nil, fmt.Errorf("failed to read docker config: %w", err)
	}
	
	adapter := http.NewDockerHandlerAdapter(config, f.logger)
	return &dockerAdapterWrapper{adapter: adapter}, nil
}

func (f *HandlerAdapterFactory) createYumAdapter() (ProxyHandlerAdapter, error) {
	config := config.YumProxyConfig{}
	if !config.ConfigExists() {
		return nil, fmt.Errorf("yum config not found")
	}
	if err := config.ReadConfig(); err != nil {
		return nil, fmt.Errorf("failed to read yum config: %w", err)
	}
	
	adapter := http.NewYumHandlerAdapter(config, f.logger)
	return &yumAdapterWrapper{adapter: adapter}, nil
}

func (f *HandlerAdapterFactory) createApkAdapter() (ProxyHandlerAdapter, error) {
	config := config.ApkProxyConfig{}
	if !config.ConfigExists() {
		return nil, fmt.Errorf("apk config not found")
	}
	if err := config.ReadConfig(); err != nil {
		return nil, fmt.Errorf("failed to read apk config: %w", err)
	}
	
	adapter := http.NewApkHandlerAdapter(config, f.logger)
	return &apkAdapterWrapper{adapter: adapter}, nil
}

// 헬퍼 메서드들

func (f *HandlerAdapterFactory) getInitializedCount() int {
	count := 0
	for _, initialized := range f.lifecycleManager.initialized {
		if initialized {
			count++
		}
	}
	return count
}

func (f *HandlerAdapterFactory) getHealthyCount() int {
	count := 0
	for _, status := range f.healthCheckMap {
		if status.IsHealthy {
			count++
		}
	}
	return count
}

func (f *HandlerAdapterFactory) getSupportedTypes() []string {
	types := make([]string, 0, len(f.adapters))
	for proxyType := range f.adapters {
		types = append(types, proxyType)
	}
	return types
}

// 어댑터 래퍼 구현들 (각 어댑터를 ProxyHandlerAdapter 인터페이스로 래핑)