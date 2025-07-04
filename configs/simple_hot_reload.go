package configs

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

// SimpleHotReload 간소화된 핫 리로드 매니저
type SimpleHotReload struct {
	// 설정 관련
	configPath   string
	configLoader ConfigLoader
	config       atomic.Value // *UnifiedConfig
	
	// 파일 감시
	watcher      *fsnotify.Watcher
	debouncer    *Debouncer
	
	// 리로드 핸들러
	handlers     []func(oldConfig, newConfig interface{}) error
	handlerNames []string
	mu           sync.RWMutex
	
	// 라이프사이클
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// ConfigLoader 설정 로더 인터페이스
type ConfigLoader interface {
	Load() (interface{}, error)
}

// Debouncer 디바운서 구현
type Debouncer struct {
	mu       sync.Mutex
	timer    *time.Timer
	duration time.Duration
}

// NewDebouncer 새 디바운서 생성
func NewDebouncer(duration time.Duration) *Debouncer {
	return &Debouncer{
		duration: duration,
	}
}

// Debounce 디바운스 실행
func (d *Debouncer) Debounce(fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	if d.timer != nil {
		d.timer.Stop()
	}
	
	d.timer = time.AfterFunc(d.duration, fn)
}

// NewSimpleHotReload 새 간소화된 핫 리로드 매니저 생성
func NewSimpleHotReload(configPath string, loader ConfigLoader) (*SimpleHotReload, error) {
	// 초기 설정 로드
	initialConfig, err := loader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load initial config: %w", err)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	hr := &SimpleHotReload{
		configPath:   configPath,
		configLoader: loader,
		debouncer:    NewDebouncer(500 * time.Millisecond),
		handlers:     make([]func(interface{}, interface{}) error, 0),
		handlerNames: make([]string, 0),
		ctx:          ctx,
		cancel:       cancel,
	}
	
	// 초기 설정 저장
	hr.config.Store(initialConfig)
	
	return hr, nil
}

// GetConfig 현재 설정 반환 (thread-safe)
func (hr *SimpleHotReload) GetConfig() interface{} {
	return hr.config.Load()
}

// RegisterHandler 리로드 핸들러 등록
func (hr *SimpleHotReload) RegisterHandler(name string, handler func(oldConfig, newConfig interface{}) error) {
	hr.mu.Lock()
	defer hr.mu.Unlock()
	
	hr.handlers = append(hr.handlers, handler)
	hr.handlerNames = append(hr.handlerNames, name)
	
	log.Printf("[HotReload] Registered handler: %s", name)
}

// Start 핫 리로드 시작
func (hr *SimpleHotReload) Start() error {
	// 파일 감시자 생성
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	hr.watcher = watcher
	
	// 설정 파일 및 디렉토리 감시
	if err := hr.setupWatchers(); err != nil {
		watcher.Close()
		return err
	}
	
	// 시그널 핸들러 설정
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP)
	
	// 감시 고루틴 시작
	hr.wg.Add(1)
	go hr.watchLoop(sigChan)
	
	log.Printf("[HotReload] Started watching: %s", hr.configPath)
	log.Printf("[HotReload] Send SIGHUP to manually reload configuration")
	
	return nil
}

// Stop 핫 리로드 중지
func (hr *SimpleHotReload) Stop() error {
	hr.cancel()
	
	if hr.watcher != nil {
		hr.watcher.Close()
	}
	
	// 모든 고루틴 종료 대기
	hr.wg.Wait()
	
	log.Println("[HotReload] Stopped")
	return nil
}

// setupWatchers 파일 감시 설정
func (hr *SimpleHotReload) setupWatchers() error {
	// 설정 파일 감시
	if err := hr.watcher.Add(hr.configPath); err != nil {
		return fmt.Errorf("failed to watch config file: %w", err)
	}
	
	// 설정 디렉토리도 감시 (새 파일 생성 감지)
	configDir := filepath.Dir(hr.configPath)
	if err := hr.watcher.Add(configDir); err != nil {
		// 디렉토리 감시 실패는 경고만
		log.Printf("[HotReload] Warning: failed to watch config directory: %v", err)
	}
	
	return nil
}

// watchLoop 파일 감시 루프
func (hr *SimpleHotReload) watchLoop(sigChan <-chan os.Signal) {
	defer hr.wg.Done()
	
	for {
		select {
		case <-hr.ctx.Done():
			return
			
		case sig := <-sigChan:
			if sig == syscall.SIGHUP {
				log.Println("[HotReload] SIGHUP received, reloading configuration...")
				hr.reload()
			}
			
		case event, ok := <-hr.watcher.Events:
			if !ok {
				return
			}
			
			// 관련 이벤트만 처리
			if hr.isRelevantEvent(event) {
				log.Printf("[HotReload] File changed: %s", event.Name)
				hr.debouncer.Debounce(hr.reload)
			}
			
		case err, ok := <-hr.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("[HotReload] Watcher error: %v", err)
		}
	}
}

// isRelevantEvent 처리해야 할 이벤트인지 확인
func (hr *SimpleHotReload) isRelevantEvent(event fsnotify.Event) bool {
	// 설정 파일이 아닌 경우 무시
	if !hr.isConfigFile(event.Name) {
		return false
	}
	
	// Write, Create, Rename 이벤트만 처리
	relevantOps := fsnotify.Write | fsnotify.Create | fsnotify.Rename
	return event.Op&relevantOps != 0
}

// isConfigFile 설정 파일인지 확인
func (hr *SimpleHotReload) isConfigFile(path string) bool {
	// 절대 경로로 변환
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	
	configAbs, err := filepath.Abs(hr.configPath)
	if err != nil {
		return false
	}
	
	// 같은 파일이거나 관련 파일인지 확인
	if absPath == configAbs {
		return true
	}
	
	// 환경별 설정 파일 체크 (예: config.production.yaml)
	base := filepath.Base(configAbs)
	ext := filepath.Ext(base)
	nameWithoutExt := base[:len(base)-len(ext)]
	
	return filepath.Base(absPath) == base || 
		   filepath.Base(absPath) == nameWithoutExt+"."+os.Getenv("PROXYND_ENV")+ext
}

// reload 설정 리로드
func (hr *SimpleHotReload) reload() {
	// 새 설정 로드
	newConfig, err := hr.configLoader.Load()
	if err != nil {
		log.Printf("[HotReload] Failed to load config: %v", err)
		return
	}
	
	// 현재 설정
	oldConfig := hr.config.Load()
	
	// 핸들러 실행
	hr.mu.RLock()
	handlers := make([]func(interface{}, interface{}) error, len(hr.handlers))
	handlerNames := make([]string, len(hr.handlerNames))
	copy(handlers, hr.handlers)
	copy(handlerNames, hr.handlerNames)
	hr.mu.RUnlock()
	
	// 모든 핸들러 실행
	success := true
	for i, handler := range handlers {
		if err := handler(oldConfig, newConfig); err != nil {
			log.Printf("[HotReload] Handler '%s' failed: %v", handlerNames[i], err)
			success = false
		}
	}
	
	// 모든 핸들러가 성공한 경우에만 설정 업데이트
	if success {
		hr.config.Store(newConfig)
		log.Println("[HotReload] Configuration reloaded successfully")
	} else {
		log.Println("[HotReload] Configuration reload failed, keeping old config")
	}
}

// SimpleReloadHandler 간단한 리로드 핸들러 예제
type SimpleReloadHandler struct {
	name string
	fn   func(oldConfig, newConfig interface{}) error
}

// NewSimpleReloadHandler 새 간단한 리로드 핸들러 생성
func NewSimpleReloadHandler(name string, fn func(oldConfig, newConfig interface{}) error) *SimpleReloadHandler {
	return &SimpleReloadHandler{
		name: name,
		fn:   fn,
	}
}

// Handle 핸들러 실행
func (h *SimpleReloadHandler) Handle(oldConfig, newConfig interface{}) error {
	return h.fn(oldConfig, newConfig)
}

// Name 핸들러 이름 반환
func (h *SimpleReloadHandler) Name() string {
	return h.name
}

// 유틸리티 함수들

// MustStartHotReload 핫 리로드를 시작하고 실패 시 패닉
func MustStartHotReload(configPath string, loader ConfigLoader) *SimpleHotReload {
	hr, err := NewSimpleHotReload(configPath, loader)
	if err != nil {
		panic(fmt.Sprintf("failed to create hot reload: %v", err))
	}
	
	if err := hr.Start(); err != nil {
		panic(fmt.Sprintf("failed to start hot reload: %v", err))
	}
	
	return hr
}

// ConfigChangeDetector 설정 변경 감지 헬퍼
type ConfigChangeDetector struct {
	detectFuncs map[string]func(old, new interface{}) bool
}

// NewConfigChangeDetector 새 설정 변경 감지기 생성
func NewConfigChangeDetector() *ConfigChangeDetector {
	return &ConfigChangeDetector{
		detectFuncs: make(map[string]func(old, new interface{}) bool),
	}
}

// Register 변경 감지 함수 등록
func (d *ConfigChangeDetector) Register(name string, detectFunc func(old, new interface{}) bool) {
	d.detectFuncs[name] = detectFunc
}

// DetectChanges 변경 사항 감지
func (d *ConfigChangeDetector) DetectChanges(oldConfig, newConfig interface{}) []string {
	var changes []string
	
	for name, detectFunc := range d.detectFuncs {
		if detectFunc(oldConfig, newConfig) {
			changes = append(changes, name)
		}
	}
	
	return changes
}