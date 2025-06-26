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
)

// HotReloadManager 핫리로드 관리자
type HotReloadManager struct {
	configLoader   *ConfigLoader
	config         *UnifiedConfig
	configMutex    sync.RWMutex
	watcher        *fsnotify.Watcher
	reloadHandlers []ReloadHandler
	ctx            context.Context
	cancel         context.CancelFunc
}

// ReloadHandler 리로드 핸들러 인터페이스
type ReloadHandler interface {
	OnConfigReload(oldConfig, newConfig *UnifiedConfig) error
	Name() string
}

// NewHotReloadManager 새 핫리로드 관리자 생성
func NewHotReloadManager(configPath string) (*HotReloadManager, error) {
	loader := NewConfigLoader(configPath)
	
	// 초기 설정 로드
	config, err := loader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load initial config: %w", err)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	manager := &HotReloadManager{
		configLoader:   loader,
		config:         config,
		reloadHandlers: make([]ReloadHandler, 0),
		ctx:            ctx,
		cancel:         cancel,
	}
	
	return manager, nil
}

// RegisterReloadHandler 리로드 핸들러 등록
func (m *HotReloadManager) RegisterReloadHandler(handler ReloadHandler) {
	m.reloadHandlers = append(m.reloadHandlers, handler)
	log.Printf("Registered reload handler: %s", handler.Name())
}

// GetConfig 현재 설정 반환 (thread-safe)
func (m *HotReloadManager) GetConfig() *UnifiedConfig {
	m.configMutex.RLock()
	defer m.configMutex.RUnlock()
	return m.config
}

// Start 핫리로드 시작
func (m *HotReloadManager) Start() error {
	// 파일 감시자 생성
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	m.watcher = watcher
	
	// 설정 파일 감시 추가
	if err := watcher.Add(m.configLoader.configPath); err != nil {
		return fmt.Errorf("failed to watch config file: %w", err)
	}
	
	// 시그널 핸들러 설정
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP)
	
	// 감시 고루틴 시작
	go m.watchLoop(sigChan)
	
	log.Printf("Hot reload manager started. Watching: %s", m.configLoader.configPath)
	log.Println("Send SIGHUP signal or modify config file to reload configuration")
	
	return nil
}

// Stop 핫리로드 중지
func (m *HotReloadManager) Stop() error {
	m.cancel()
	
	if m.watcher != nil {
		return m.watcher.Close()
	}
	
	return nil
}

// watchLoop 감시 루프
func (m *HotReloadManager) watchLoop(sigChan <-chan os.Signal) {
	// 디바운싱을 위한 타이머
	var debounceTimer *time.Timer
	debounce := 500 * time.Millisecond
	
	for {
		select {
		case <-m.ctx.Done():
			return
			
		case sig := <-sigChan:
			if sig == syscall.SIGHUP {
				log.Println("Received SIGHUP signal, reloading configuration...")
				if err := m.reload(); err != nil {
					log.Printf("Failed to reload config: %v", err)
				}
			}
			
		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}
			
			// 파일 쓰기 또는 생성 이벤트만 처리
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				// 디바운싱: 짧은 시간 내 여러 이벤트를 하나로 처리
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				
				debounceTimer = time.AfterFunc(debounce, func() {
					log.Printf("Config file changed: %s, reloading...", event.Name)
					if err := m.reload(); err != nil {
						log.Printf("Failed to reload config: %v", err)
					}
				})
			}
			
		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("File watcher error: %v", err)
		}
	}
}

// reload 설정 리로드
func (m *HotReloadManager) reload() error {
	// 새 설정 로드
	newConfig, err := m.configLoader.Load()
	if err != nil {
		return fmt.Errorf("failed to load new config: %w", err)
	}
	
	// 현재 설정 백업
	m.configMutex.RLock()
	oldConfig := m.config
	m.configMutex.RUnlock()
	
	// 설정 변경 사항 확인
	changes := m.detectChanges(oldConfig, newConfig)
	if len(changes) == 0 {
		log.Println("No configuration changes detected")
		return nil
	}
	
	log.Printf("Detected %d configuration changes:", len(changes))
	for _, change := range changes {
		log.Printf("  - %s: %s -> %s", change.Path, change.OldValue, change.NewValue)
	}
	
	// 핸들러들에게 설정 변경 알림
	for _, handler := range m.reloadHandlers {
		if err := handler.OnConfigReload(oldConfig, newConfig); err != nil {
			log.Printf("Handler %s failed to reload: %v", handler.Name(), err)
			// 하나라도 실패하면 롤백
			return fmt.Errorf("reload handler %s failed: %w", handler.Name(), err)
		}
		log.Printf("Handler %s reloaded successfully", handler.Name())
	}
	
	// 새 설정 적용
	m.configMutex.Lock()
	m.config = newConfig
	m.configMutex.Unlock()
	
	log.Println("Configuration reloaded successfully")
	return nil
}

// ConfigChange 설정 변경 사항
type ConfigChange struct {
	Path     string
	OldValue string
	NewValue string
}

// detectChanges 설정 변경 사항 감지
func (m *HotReloadManager) detectChanges(oldConfig, newConfig *UnifiedConfig) []ConfigChange {
	changes := []ConfigChange{}
	
	// 간단한 변경 감지 (주요 필드만)
	if oldConfig.Server.Port != newConfig.Server.Port {
		changes = append(changes, ConfigChange{
			Path:     "server.port",
			OldValue: fmt.Sprintf("%d", oldConfig.Server.Port),
			NewValue: fmt.Sprintf("%d", newConfig.Server.Port),
		})
	}
	
	if oldConfig.Logging.Level != newConfig.Logging.Level {
		changes = append(changes, ConfigChange{
			Path:     "logging.level",
			OldValue: oldConfig.Logging.Level,
			NewValue: newConfig.Logging.Level,
		})
	}
	
	if oldConfig.Cache.Backend != newConfig.Cache.Backend {
		changes = append(changes, ConfigChange{
			Path:     "cache.backend",
			OldValue: oldConfig.Cache.Backend,
			NewValue: newConfig.Cache.Backend,
		})
	}
	
	if oldConfig.Metrics.Enabled != newConfig.Metrics.Enabled {
		changes = append(changes, ConfigChange{
			Path:     "metrics.enabled",
			OldValue: fmt.Sprintf("%v", oldConfig.Metrics.Enabled),
			NewValue: fmt.Sprintf("%v", newConfig.Metrics.Enabled),
		})
	}
	
	// TODO: 더 상세한 변경 감지 구현
	
	return changes
}

// 예제 리로드 핸들러들

// LoggingReloadHandler 로깅 리로드 핸들러
type LoggingReloadHandler struct {
	logger interface{}
}

func (h *LoggingReloadHandler) OnConfigReload(oldConfig, newConfig *UnifiedConfig) error {
	if oldConfig.Logging.Level != newConfig.Logging.Level {
		// 로그 레벨 변경
		log.Printf("Changing log level from %s to %s", oldConfig.Logging.Level, newConfig.Logging.Level)
		// TODO: 실제 로거의 레벨 변경
	}
	
	if oldConfig.Logging.Format != newConfig.Logging.Format {
		// 로그 포맷 변경
		log.Printf("Changing log format from %s to %s", oldConfig.Logging.Format, newConfig.Logging.Format)
		// TODO: 실제 로거의 포맷 변경
	}
	
	return nil
}

func (h *LoggingReloadHandler) Name() string {
	return "LoggingReloadHandler"
}

// CacheReloadHandler 캐시 리로드 핸들러
type CacheReloadHandler struct {
	cacheManager interface{}
}

func (h *CacheReloadHandler) OnConfigReload(oldConfig, newConfig *UnifiedConfig) error {
	if oldConfig.Cache.Backend != newConfig.Cache.Backend {
		// 캐시 백엔드 변경은 재시작 필요
		return fmt.Errorf("cache backend change requires restart")
	}
	
	if oldConfig.Cache.TTL != newConfig.Cache.TTL {
		// TTL 변경
		log.Printf("Updating cache TTL from %s to %s", oldConfig.Cache.TTL, newConfig.Cache.TTL)
		// TODO: 캐시 매니저의 TTL 업데이트
	}
	
	return nil
}

func (h *CacheReloadHandler) Name() string {
	return "CacheReloadHandler"
}

// MetricsReloadHandler 메트릭 리로드 핸들러
type MetricsReloadHandler struct {
	metricsServer interface{}
}

func (h *MetricsReloadHandler) OnConfigReload(oldConfig, newConfig *UnifiedConfig) error {
	if oldConfig.Metrics.Enabled != newConfig.Metrics.Enabled {
		if newConfig.Metrics.Enabled {
			log.Println("Enabling metrics endpoint")
			// TODO: 메트릭 서버 시작
		} else {
			log.Println("Disabling metrics endpoint")
			// TODO: 메트릭 서버 중지
		}
	}
	
	return nil
}

func (h *MetricsReloadHandler) Name() string {
	return "MetricsReloadHandler"
}

// SecurityReloadHandler 보안 리로드 핸들러
type SecurityReloadHandler struct {
	authManager interface{}
}

func (h *SecurityReloadHandler) OnConfigReload(oldConfig, newConfig *UnifiedConfig) error {
	// 사용자 파일 리로드
	if oldConfig.Security.Authentication.BasicAuth.UsersFile != newConfig.Security.Authentication.BasicAuth.UsersFile {
		log.Printf("Reloading users from: %s", newConfig.Security.Authentication.BasicAuth.UsersFile)
		// TODO: 사용자 파일 리로드
	}
	
	// IP 화이트리스트 업데이트
	if oldConfig.Security.AccessControl.IPWhitelist.Enabled != newConfig.Security.AccessControl.IPWhitelist.Enabled {
		if newConfig.Security.AccessControl.IPWhitelist.Enabled {
			log.Println("Enabling IP whitelist")
		} else {
			log.Println("Disabling IP whitelist")
		}
		// TODO: IP 화이트리스트 업데이트
	}
	
	return nil
}

func (h *SecurityReloadHandler) Name() string {
	return "SecurityReloadHandler"
}