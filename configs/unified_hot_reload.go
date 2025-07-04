package configs

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// UnifiedHotReload 통합 핫 리로드 시스템
type UnifiedHotReload struct {
	// 설정 로더 (Viper 또는 레거시)
	useViper     bool
	viperLoader  *ViperConfigLoader
	legacyLoader *ConfigLoader
	
	// 현재 설정
	config       *UnifiedConfig
	configMu     sync.RWMutex
	
	// 리로드 핸들러
	handlers     []ReloadHandler
	handlersMu   sync.RWMutex
	
	// 파일 감시 (레거시 모드용)
	watcher      *fsnotify.Watcher
	debounce     *time.Timer
	debounceMu   sync.Mutex
	debounceTime time.Duration
	
	// 라이프사이클
	ctx          context.Context
	cancel       context.CancelFunc
	running      bool
	runningMu    sync.Mutex
}

// NewUnifiedHotReload 새 통합 핫 리로드 생성
func NewUnifiedHotReload(configPath string, useViper bool) (*UnifiedHotReload, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	uhr := &UnifiedHotReload{
		useViper:     useViper,
		handlers:     make([]ReloadHandler, 0),
		debounceTime: 500 * time.Millisecond,
		ctx:          ctx,
		cancel:       cancel,
	}
	
	// 로더 초기화 및 초기 설정 로드
	var initialConfig *UnifiedConfig
	var err error
	
	if useViper {
		uhr.viperLoader = NewViperConfigLoader()
		uhr.viperLoader.SetConfigPath(configPath)
		initialConfig, err = uhr.viperLoader.Load()
	} else {
		uhr.legacyLoader = NewConfigLoader(configPath)
		initialConfig, err = uhr.legacyLoader.Load()
	}
	
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to load initial config: %w", err)
	}
	
	uhr.config = initialConfig
	return uhr, nil
}

// GetConfig 현재 설정 반환 (thread-safe)
func (uhr *UnifiedHotReload) GetConfig() *UnifiedConfig {
	uhr.configMu.RLock()
	defer uhr.configMu.RUnlock()
	return uhr.config
}

// RegisterHandler 리로드 핸들러 등록
func (uhr *UnifiedHotReload) RegisterHandler(handler ReloadHandler) {
	uhr.handlersMu.Lock()
	defer uhr.handlersMu.Unlock()
	
	uhr.handlers = append(uhr.handlers, handler)
	log.Printf("[UnifiedHotReload] Registered handler: %s", handler.Name())
}

// Start 핫 리로드 시작
func (uhr *UnifiedHotReload) Start() error {
	uhr.runningMu.Lock()
	defer uhr.runningMu.Unlock()
	
	if uhr.running {
		return fmt.Errorf("hot reload already running")
	}
	
	if uhr.useViper {
		// Viper 자동 감시 설정
		uhr.setupViperWatch()
	} else {
		// 레거시 파일 감시 설정
		if err := uhr.setupLegacyWatch(); err != nil {
			return err
		}
	}
	
	// SIGHUP 시그널 핸들링
	uhr.setupSignalHandling()
	
	uhr.running = true
	log.Println("[UnifiedHotReload] Started")
	
	return nil
}

// Stop 핫 리로드 중지
func (uhr *UnifiedHotReload) Stop() error {
	uhr.runningMu.Lock()
	defer uhr.runningMu.Unlock()
	
	if !uhr.running {
		return nil
	}
	
	uhr.cancel()
	
	if uhr.watcher != nil {
		uhr.watcher.Close()
	}
	
	uhr.running = false
	log.Println("[UnifiedHotReload] Stopped")
	
	return nil
}

// setupViperWatch Viper 자동 감시 설정
func (uhr *UnifiedHotReload) setupViperWatch() {
	uhr.viperLoader.WatchConfig(func(newConfig *UnifiedConfig) {
		uhr.handleConfigChange(newConfig)
	})
	
	log.Println("[UnifiedHotReload] Viper watch enabled")
}

// setupLegacyWatch 레거시 파일 감시 설정
func (uhr *UnifiedHotReload) setupLegacyWatch() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	uhr.watcher = watcher
	
	// 설정 파일 감시
	if err := watcher.Add(uhr.legacyLoader.configPath); err != nil {
		watcher.Close()
		return fmt.Errorf("failed to watch config file: %w", err)
	}
	
	// 감시 고루틴 시작
	go uhr.legacyWatchLoop()
	
	log.Printf("[UnifiedHotReload] Legacy watch enabled: %s", uhr.legacyLoader.configPath)
	return nil
}

// legacyWatchLoop 레거시 파일 감시 루프
func (uhr *UnifiedHotReload) legacyWatchLoop() {
	for {
		select {
		case <-uhr.ctx.Done():
			return
			
		case event, ok := <-uhr.watcher.Events:
			if !ok {
				return
			}
			
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				uhr.debounceReload()
			}
			
		case err, ok := <-uhr.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("[UnifiedHotReload] Watch error: %v", err)
		}
	}
}

// debounceReload 디바운스된 리로드
func (uhr *UnifiedHotReload) debounceReload() {
	uhr.debounceMu.Lock()
	defer uhr.debounceMu.Unlock()
	
	if uhr.debounce != nil {
		uhr.debounce.Stop()
	}
	
	uhr.debounce = time.AfterFunc(uhr.debounceTime, func() {
		log.Println("[UnifiedHotReload] Config file changed, reloading...")
		uhr.reloadLegacy()
	})
}

// reloadLegacy 레거시 설정 리로드
func (uhr *UnifiedHotReload) reloadLegacy() {
	newConfig, err := uhr.legacyLoader.Load()
	if err != nil {
		log.Printf("[UnifiedHotReload] Failed to reload config: %v", err)
		return
	}
	
	uhr.handleConfigChange(newConfig)
}

// handleConfigChange 설정 변경 처리
func (uhr *UnifiedHotReload) handleConfigChange(newConfig *UnifiedConfig) {
	// 현재 설정
	uhr.configMu.RLock()
	oldConfig := uhr.config
	uhr.configMu.RUnlock()
	
	// 핸들러 복사 (잠금 최소화)
	uhr.handlersMu.RLock()
	handlers := make([]ReloadHandler, len(uhr.handlers))
	copy(handlers, uhr.handlers)
	uhr.handlersMu.RUnlock()
	
	// 모든 핸들러 실행
	allSuccess := true
	for _, handler := range handlers {
		if err := handler.OnConfigReload(oldConfig, newConfig); err != nil {
			log.Printf("[UnifiedHotReload] Handler %s failed: %v", handler.Name(), err)
			allSuccess = false
		} else {
			log.Printf("[UnifiedHotReload] Handler %s succeeded", handler.Name())
		}
	}
	
	// 성공 시 설정 업데이트
	if allSuccess {
		uhr.configMu.Lock()
		uhr.config = newConfig
		uhr.configMu.Unlock()
		log.Println("[UnifiedHotReload] Configuration updated successfully")
	} else {
		log.Println("[UnifiedHotReload] Some handlers failed, keeping old configuration")
	}
}

// setupSignalHandling SIGHUP 시그널 핸들링 설정
func (uhr *UnifiedHotReload) setupSignalHandling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP)
	
	go func() {
		for {
			select {
			case <-uhr.ctx.Done():
				return
			case <-sigChan:
				log.Println("[UnifiedHotReload] SIGHUP received, reloading...")
				uhr.manualReload()
			}
		}
	}()
}

// manualReload 수동 리로드
func (uhr *UnifiedHotReload) manualReload() {
	var newConfig *UnifiedConfig
	var err error
	
	if uhr.useViper {
		newConfig, err = uhr.viperLoader.Load()
	} else {
		newConfig, err = uhr.legacyLoader.Load()
	}
	
	if err != nil {
		log.Printf("[UnifiedHotReload] Manual reload failed: %v", err)
		return
	}
	
	uhr.handleConfigChange(newConfig)
}

// GetViper Viper 인스턴스 반환 (Viper 모드인 경우)
func (uhr *UnifiedHotReload) GetViper() *viper.Viper {
	if uhr.useViper && uhr.viperLoader != nil {
		return uhr.viperLoader.GetViper()
	}
	return nil
}

// SetDebounceTime 디바운스 시간 설정
func (uhr *UnifiedHotReload) SetDebounceTime(duration time.Duration) {
	uhr.debounceTime = duration
}

// IsRunning 실행 중인지 확인
func (uhr *UnifiedHotReload) IsRunning() bool {
	uhr.runningMu.Lock()
	defer uhr.runningMu.Unlock()
	return uhr.running
}

// 간소화된 핸들러 생성 헬퍼

// MakeLoggingHandler 로깅 핸들러 생성
func MakeLoggingHandler() ReloadHandler {
	return &genericReloadHandler{
		name: "LoggingHandler",
		fn: func(old, new *UnifiedConfig) error {
			if old.Logging.Level != new.Logging.Level {
				log.Printf("[Logging] Level changed: %s -> %s", old.Logging.Level, new.Logging.Level)
			}
			if old.Logging.Format != new.Logging.Format {
				log.Printf("[Logging] Format changed: %s -> %s", old.Logging.Format, new.Logging.Format)
			}
			return nil
		},
	}
}

// MakeCacheHandler 캐시 핸들러 생성
func MakeCacheHandler() ReloadHandler {
	return &genericReloadHandler{
		name: "CacheHandler",
		fn: func(old, new *UnifiedConfig) error {
			if old.Cache.Backend != new.Cache.Backend {
				return fmt.Errorf("cache backend change requires restart: %s -> %s", 
					old.Cache.Backend, new.Cache.Backend)
			}
			if old.Cache.TTL != new.Cache.TTL {
				log.Printf("[Cache] TTL changed: %s -> %s", old.Cache.TTL, new.Cache.TTL)
			}
			return nil
		},
	}
}

// MakeMetricsHandler 메트릭 핸들러 생성
func MakeMetricsHandler() ReloadHandler {
	return &genericReloadHandler{
		name: "MetricsHandler",
		fn: func(old, new *UnifiedConfig) error {
			if old.Metrics.Enabled != new.Metrics.Enabled {
				if new.Metrics.Enabled {
					log.Println("[Metrics] Enabling metrics")
				} else {
					log.Println("[Metrics] Disabling metrics")
				}
			}
			return nil
		},
	}
}

// genericReloadHandler 범용 리로드 핸들러
type genericReloadHandler struct {
	name string
	fn   func(old, new *UnifiedConfig) error
}

func (h *genericReloadHandler) OnConfigReload(oldConfig, newConfig *UnifiedConfig) error {
	return h.fn(oldConfig, newConfig)
}

func (h *genericReloadHandler) Name() string {
	return h.name
}

// QuickSetupHotReload 빠른 핫 리로드 설정
func QuickSetupHotReload(configPath string, useViper bool) (*UnifiedHotReload, error) {
	hr, err := NewUnifiedHotReload(configPath, useViper)
	if err != nil {
		return nil, err
	}
	
	// 기본 핸들러 등록
	hr.RegisterHandler(MakeLoggingHandler())
	hr.RegisterHandler(MakeCacheHandler())
	hr.RegisterHandler(MakeMetricsHandler())
	
	// 시작
	if err := hr.Start(); err != nil {
		return nil, err
	}
	
	return hr, nil
}