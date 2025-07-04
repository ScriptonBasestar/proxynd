package configs

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/spf13/viper"
)

// ViperHotReloadManager Viper 기반 핫리로드 관리자
type ViperHotReloadManager struct {
	loader         *ViperConfigLoader
	config         *UnifiedConfig
	configMutex    sync.RWMutex
	reloadHandlers []ReloadHandler
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewViperHotReloadManager 새 Viper 핫리로드 관리자 생성
func NewViperHotReloadManager(configPath string) (*ViperHotReloadManager, error) {
	loader := NewViperConfigLoader()
	loader.SetConfigPath(configPath)

	// 초기 설정 로드
	config, err := loader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load initial config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	manager := &ViperHotReloadManager{
		loader:         loader,
		config:         config,
		reloadHandlers: make([]ReloadHandler, 0),
		ctx:            ctx,
		cancel:         cancel,
	}

	return manager, nil
}

// RegisterReloadHandler 리로드 핸들러 등록
func (m *ViperHotReloadManager) RegisterReloadHandler(handler ReloadHandler) {
	m.reloadHandlers = append(m.reloadHandlers, handler)
	log.Printf("Registered reload handler: %s", handler.Name())
}

// GetConfig 현재 설정 반환 (thread-safe)
func (m *ViperHotReloadManager) GetConfig() *UnifiedConfig {
	m.configMutex.RLock()
	defer m.configMutex.RUnlock()
	return m.config
}

// GetViper 내부 Viper 인스턴스 반환
func (m *ViperHotReloadManager) GetViper() *viper.Viper {
	return m.loader.GetViper()
}

// Start 핫리로드 시작
func (m *ViperHotReloadManager) Start() error {
	// Viper의 자동 설정 감시 시작
	m.loader.WatchConfig(func(newConfig *UnifiedConfig) {
		// 현재 설정 백업
		m.configMutex.RLock()
		oldConfig := m.config
		m.configMutex.RUnlock()

		// 변경 사항 로그
		log.Println("Configuration file changed, reloading...")

		// 핸들러들에게 설정 변경 알림
		allSuccess := true
		for _, handler := range m.reloadHandlers {
			if err := handler.OnConfigReload(oldConfig, newConfig); err != nil {
				log.Printf("Handler %s failed to reload: %v", handler.Name(), err)
				allSuccess = false
			} else {
				log.Printf("Handler %s reloaded successfully", handler.Name())
			}
		}

		// 모든 핸들러가 성공한 경우에만 새 설정 적용
		if allSuccess {
			m.configMutex.Lock()
			m.config = newConfig
			m.configMutex.Unlock()
			log.Println("Configuration reloaded successfully")
		} else {
			log.Println("Some handlers failed, configuration not updated")
		}
	})

	// 시그널 핸들러 설정 (SIGHUP으로 수동 리로드)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP)

	// 시그널 처리 고루틴
	go func() {
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
			}
		}
	}()

	log.Println("Hot reload manager started with Viper")
	log.Println("Configuration will be automatically reloaded on file changes")
	log.Println("Send SIGHUP signal to manually reload configuration")

	return nil
}

// Stop 핫리로드 중지
func (m *ViperHotReloadManager) Stop() error {
	m.cancel()
	return nil
}

// reload 수동 설정 리로드
func (m *ViperHotReloadManager) reload() error {
	// 새 설정 로드
	newConfig, err := m.loader.Load()
	if err != nil {
		return fmt.Errorf("failed to load new config: %w", err)
	}

	// 현재 설정 백업
	m.configMutex.RLock()
	oldConfig := m.config
	m.configMutex.RUnlock()

	// 핸들러들에게 설정 변경 알림
	for _, handler := range m.reloadHandlers {
		if err := handler.OnConfigReload(oldConfig, newConfig); err != nil {
			log.Printf("Handler %s failed to reload: %v", handler.Name(), err)
			return fmt.Errorf("reload handler %s failed: %w", handler.Name(), err)
		}
		log.Printf("Handler %s reloaded successfully", handler.Name())
	}

	// 새 설정 적용
	m.configMutex.Lock()
	m.config = newConfig
	m.configMutex.Unlock()

	log.Println("Configuration manually reloaded successfully")
	return nil
}

// 기존 핸들러들을 Viper와 호환되도록 업데이트

// ViperLoggingReloadHandler Viper 호환 로깅 리로드 핸들러
type ViperLoggingReloadHandler struct {
	loggerManager interface{}
}

func NewViperLoggingReloadHandler(loggerManager interface{}) *ViperLoggingReloadHandler {
	return &ViperLoggingReloadHandler{
		loggerManager: loggerManager,
	}
}

func (h *ViperLoggingReloadHandler) OnConfigReload(oldConfig, newConfig *UnifiedConfig) error {
	// 로그 레벨 변경 확인
	if oldConfig.Logging.Level != newConfig.Logging.Level {
		log.Printf("Updating log level: %s -> %s", oldConfig.Logging.Level, newConfig.Logging.Level)
		// TODO: 실제 로거 레벨 업데이트
	}

	// 로그 포맷 변경 확인
	if oldConfig.Logging.Format != newConfig.Logging.Format {
		log.Printf("Updating log format: %s -> %s", oldConfig.Logging.Format, newConfig.Logging.Format)
		// TODO: 실제 로거 포맷 업데이트
	}

	// 로그 출력 변경 확인
	if oldConfig.Logging.Output != newConfig.Logging.Output {
		log.Printf("Updating log output: %s -> %s", oldConfig.Logging.Output, newConfig.Logging.Output)
		// TODO: 실제 로거 출력 업데이트
	}

	return nil
}

func (h *ViperLoggingReloadHandler) Name() string {
	return "ViperLoggingReloadHandler"
}

// ViperCacheReloadHandler Viper 호환 캐시 리로드 핸들러
type ViperCacheReloadHandler struct {
	cacheManager interface{}
}

func NewViperCacheReloadHandler(cacheManager interface{}) *ViperCacheReloadHandler {
	return &ViperCacheReloadHandler{
		cacheManager: cacheManager,
	}
}

func (h *ViperCacheReloadHandler) OnConfigReload(oldConfig, newConfig *UnifiedConfig) error {
	// 캐시 백엔드 변경은 재시작 필요
	if oldConfig.Cache.Backend != newConfig.Cache.Backend {
		return fmt.Errorf("cache backend change from %s to %s requires restart", 
			oldConfig.Cache.Backend, newConfig.Cache.Backend)
	}

	// TTL 변경
	if oldConfig.Cache.TTL != newConfig.Cache.TTL {
		log.Printf("Updating cache TTL: %s -> %s", oldConfig.Cache.TTL, newConfig.Cache.TTL)
		// TODO: 캐시 매니저 TTL 업데이트
	}

	// 최대 크기 변경
	if oldConfig.Cache.MaxSize != newConfig.Cache.MaxSize {
		log.Printf("Updating cache max size: %s -> %s", oldConfig.Cache.MaxSize, newConfig.Cache.MaxSize)
		// TODO: 캐시 매니저 크기 제한 업데이트
	}

	return nil
}

func (h *ViperCacheReloadHandler) Name() string {
	return "ViperCacheReloadHandler"
}

// ViperMetricsReloadHandler Viper 호환 메트릭 리로드 핸들러
type ViperMetricsReloadHandler struct {
	metricsServer interface{}
}

func NewViperMetricsReloadHandler(metricsServer interface{}) *ViperMetricsReloadHandler {
	return &ViperMetricsReloadHandler{
		metricsServer: metricsServer,
	}
}

func (h *ViperMetricsReloadHandler) OnConfigReload(oldConfig, newConfig *UnifiedConfig) error {
	// 메트릭 활성화 상태 변경
	if oldConfig.Metrics.Enabled != newConfig.Metrics.Enabled {
		if newConfig.Metrics.Enabled {
			log.Println("Enabling metrics endpoint")
			// TODO: 메트릭 서버 시작
		} else {
			log.Println("Disabling metrics endpoint")
			// TODO: 메트릭 서버 중지
		}
	}

	// 메트릭 포트 변경 (별도 포트 사용 시)
	if oldConfig.Metrics.Port != newConfig.Metrics.Port && newConfig.Metrics.Port > 0 {
		log.Printf("Metrics port change requires restart: %d -> %d", 
			oldConfig.Metrics.Port, newConfig.Metrics.Port)
		// 포트 변경은 일반적으로 재시작 필요
	}

	return nil
}

func (h *ViperMetricsReloadHandler) Name() string {
	return "ViperMetricsReloadHandler"
}

// ViperSecurityReloadHandler Viper 호환 보안 리로드 핸들러
type ViperSecurityReloadHandler struct {
	authManager interface{}
}

func NewViperSecurityReloadHandler(authManager interface{}) *ViperSecurityReloadHandler {
	return &ViperSecurityReloadHandler{
		authManager: authManager,
	}
}

func (h *ViperSecurityReloadHandler) OnConfigReload(oldConfig, newConfig *UnifiedConfig) error {
	// 기본 인증 상태 변경
	oldBasicAuth := oldConfig.Security.Authentication.BasicAuth
	newBasicAuth := newConfig.Security.Authentication.BasicAuth
	
	if oldBasicAuth.Enabled != newBasicAuth.Enabled {
		if newBasicAuth.Enabled {
			log.Println("Enabling basic authentication")
		} else {
			log.Println("Disabling basic authentication")
		}
		// TODO: 인증 매니저 업데이트
	}

	// 사용자 파일 변경
	if oldBasicAuth.UsersFile != newBasicAuth.UsersFile {
		log.Printf("Reloading users from: %s", newBasicAuth.UsersFile)
		// TODO: 사용자 파일 리로드
	}

	// IP 화이트리스트 변경
	oldIPWhitelist := oldConfig.Security.AccessControl.IPWhitelist
	newIPWhitelist := newConfig.Security.AccessControl.IPWhitelist
	
	if oldIPWhitelist.Enabled != newIPWhitelist.Enabled {
		if newIPWhitelist.Enabled {
			log.Println("Enabling IP whitelist")
		} else {
			log.Println("Disabling IP whitelist")
		}
		// TODO: IP 필터 업데이트
	}

	return nil
}

func (h *ViperSecurityReloadHandler) Name() string {
	return "ViperSecurityReloadHandler"
}

// ViperRegistryReloadHandler Viper 호환 레지스트리 리로드 핸들러
type ViperRegistryReloadHandler struct {
	registryManager interface{}
}

func NewViperRegistryReloadHandler(registryManager interface{}) *ViperRegistryReloadHandler {
	return &ViperRegistryReloadHandler{
		registryManager: registryManager,
	}
}

func (h *ViperRegistryReloadHandler) OnConfigReload(oldConfig, newConfig *UnifiedConfig) error {
	// NPM 레지스트리 변경
	if oldConfig.Registries.NPM.Enabled != newConfig.Registries.NPM.Enabled {
		if newConfig.Registries.NPM.Enabled {
			log.Println("Enabling NPM registry")
		} else {
			log.Println("Disabling NPM registry")
		}
	}

	// PyPI 레지스트리 변경
	if oldConfig.Registries.PyPI.Enabled != newConfig.Registries.PyPI.Enabled {
		if newConfig.Registries.PyPI.Enabled {
			log.Println("Enabling PyPI registry")
		} else {
			log.Println("Disabling PyPI registry")
		}
	}

	// Docker 레지스트리 변경
	if oldConfig.Registries.Docker.Enabled != newConfig.Registries.Docker.Enabled {
		if newConfig.Registries.Docker.Enabled {
			log.Println("Enabling Docker registry")
		} else {
			log.Println("Disabling Docker registry")
		}
	}

	// TODO: 실제 레지스트리 매니저 업데이트

	return nil
}

func (h *ViperRegistryReloadHandler) Name() string {
	return "ViperRegistryReloadHandler"
}