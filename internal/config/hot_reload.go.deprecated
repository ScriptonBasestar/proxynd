package config

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"

	"proxynd/logging"
)

// ReloadHandler 리로드 핸들러 인터페이스
type ReloadHandler interface {
	OnConfigReload(oldConfig, newConfig *UnifiedConfig) error
	Name() string
}

// ValidationMode 검증 모드
type ValidationMode int

const (
	ValidationModeDisabled ValidationMode = iota // 검증 비활성화
	ValidationModeWarn                           // 경고만 출력
	ValidationModeStrict                         // 엄격한 검증 (실패 시 리로드 중단)
)

// ConfigChangeEvent 설정 변경 이벤트
type ConfigChangeEvent struct {
	Type      string            `json:"type"`      // file_changed, manual_reload, signal_reload
	FilePath  string            `json:"file_path"` // 변경된 파일 경로
	Timestamp time.Time         `json:"timestamp"` // 변경 시각
	OldHash   string            `json:"old_hash"`  // 이전 파일 해시
	NewHash   string            `json:"new_hash"`  // 새 파일 해시
	Metadata  map[string]string `json:"metadata"`  // 추가 메타데이터
}

// ConfigChangeRecord 설정 변경 기록
type ConfigChangeRecord struct {
	Event            ConfigChangeEvent `json:"event"`
	OldConfig        *UnifiedConfig    `json:"old_config,omitempty"`
	NewConfig        *UnifiedConfig    `json:"new_config,omitempty"`
	Success          bool              `json:"success"`
	Error            string            `json:"error,omitempty"`
	ValidationResult *ValidationResult `json:"validation_result,omitempty"`
	HealthStatus     *HealthStatus     `json:"health_status,omitempty"`
	HandlerResults   map[string]error  `json:"handler_results,omitempty"`
	Duration         time.Duration     `json:"duration"`
	RollbackUsed     bool              `json:"rollback_used"`
}

// HotReloadManager 핫리로드 관리자 (레거시 호환성을 위한 별칭)
type HotReloadManager = UnifiedHotReload

// NewHotReloadManager 새 핫리로드 관리자 생성 (레거시 호환성)
func NewHotReloadManager(configPath string) (*HotReloadManager, error) {
	return NewUnifiedHotReload(configPath, false)
}

// UnifiedHotReload 통합 핫 리로드 시스템
type UnifiedHotReload struct {
	// 설정 로더 (Viper 또는 레거시)
	useViper     bool
	viperLoader  *ViperConfigLoader
	legacyLoader *ConfigLoader

	// 현재 설정
	config   *UnifiedConfig
	configMu sync.RWMutex

	// 리로드 핸들러
	handlers   []ReloadHandler
	handlersMu sync.RWMutex

	// 파일 감시 (레거시 모드용)
	watcher      *fsnotify.Watcher
	debounce     *time.Timer
	debounceMu   sync.Mutex
	debounceTime time.Duration

	// 라이프사이클
	ctx       context.Context
	cancel    context.CancelFunc
	running   bool
	runningMu sync.Mutex

	// 향상된 변경 감지
	validator       *SchemaValidator
	healthMonitor   *ConfigHealthMonitor
	backupManager   *ConfigBackupManager
	logger          logging.Logger
	fileHashes      map[string]string
	fileHashesMu    sync.RWMutex
	changeQueue     chan ConfigChangeEvent
	maxRetries      int
	retryDelay      time.Duration
	validationMode  ValidationMode
	rollbackEnabled bool
	changeHistory   []ConfigChangeRecord
	changeHistoryMu sync.RWMutex
	maxHistorySize  int
}

// NewUnifiedHotReload 새 통합 핫 리로드 생성
func NewUnifiedHotReload(configPath string, useViper bool) (*UnifiedHotReload, error) {
	ctx, cancel := context.WithCancel(context.Background())
	logger := logging.GetLogger()

	uhr := &UnifiedHotReload{
		useViper:        useViper,
		handlers:        make([]ReloadHandler, 0),
		debounceTime:    500 * time.Millisecond,
		ctx:             ctx,
		cancel:          cancel,
		logger:          logger,
		fileHashes:      make(map[string]string),
		changeQueue:     make(chan ConfigChangeEvent, 100),
		maxRetries:      3,
		retryDelay:      1 * time.Second,
		validationMode:  ValidationModeWarn,
		rollbackEnabled: true,
		changeHistory:   make([]ConfigChangeRecord, 0),
		maxHistorySize:  50,
	}

	// 향상된 컴포넌트 초기화
	uhr.validator = NewSchemaValidator()
	uhr.healthMonitor = NewConfigHealthMonitor(logger, uhr.validator)
	uhr.backupManager = &ConfigBackupManager{
		backupDir:          filepath.Join(filepath.Dir(configPath), "backups"),
		maxBackups:         20,
		compressionEnabled: true,
		retentionPeriod:    30 * 24 * time.Hour, // 30일
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

	// 초기 파일 해시 계산
	if err := uhr.updateFileHash(configPath); err != nil {
		uhr.logger.Warn("Failed to calculate initial file hash", logging.F("error", err))
	}

	// 변경 이벤트 처리 고루틴 시작
	go uhr.processChangeEvents()

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

// RegisterReloadHandler 리로드 핸들러 등록 (레거시 호환성)
func (uhr *UnifiedHotReload) RegisterReloadHandler(handler ReloadHandler) {
	uhr.RegisterHandler(handler)
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
		if err := uhr.watcher.Close(); err != nil {
			log.Printf("[UnifiedHotReload] Failed to close watcher: %v", err)
		}
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
		if closeErr := watcher.Close(); closeErr != nil {
			log.Printf("[UnifiedHotReload] Failed to close watcher after add error: %v", closeErr)
		}
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
		fn: func(old, newVal *UnifiedConfig) error {
			if old.Logging.Level != newVal.Logging.Level {
				log.Printf("[Logging] Level changed: %s -> %s", old.Logging.Level, newVal.Logging.Level)
			}
			if old.Logging.Format != newVal.Logging.Format {
				log.Printf("[Logging] Format changed: %s -> %s", old.Logging.Format, newVal.Logging.Format)
			}
			return nil
		},
	}
}

// MakeCacheHandler 캐시 핸들러 생성
func MakeCacheHandler() ReloadHandler {
	return &genericReloadHandler{
		name: "CacheHandler",
		fn: func(old, newVal *UnifiedConfig) error {
			if old.Cache.Backend != newVal.Cache.Backend {
				return fmt.Errorf("cache backend change requires restart: %s -> %s",
					old.Cache.Backend, newVal.Cache.Backend)
			}
			if old.Cache.TTL != newVal.Cache.TTL {
				log.Printf("[Cache] TTL changed: %s -> %s", old.Cache.TTL, newVal.Cache.TTL)
			}
			return nil
		},
	}
}

// MakeMetricsHandler 메트릭 핸들러 생성
func MakeMetricsHandler() ReloadHandler {
	return &genericReloadHandler{
		name: "MetricsHandler",
		fn: func(old, newVal *UnifiedConfig) error {
			if old.Metrics.Enabled != newVal.Metrics.Enabled {
				if newVal.Metrics.Enabled {
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
	fn   func(old, newVal *UnifiedConfig) error
}

// OnConfigReload handles configuration reload events using the stored function
func (h *genericReloadHandler) OnConfigReload(oldConfig, newConfig *UnifiedConfig) error {
	return h.fn(oldConfig, newConfig)
}

// Name returns the name of the reload handler
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

// 레거시 리로드 핸들러들 (호환성)

// LoggingReloadHandler 로깅 리로드 핸들러
type LoggingReloadHandler struct {
	_ interface{} // reserved for future logger implementation
}

// OnConfigReload handles logging configuration changes during reload
func (h *LoggingReloadHandler) OnConfigReload(oldConfig, newConfig *UnifiedConfig) error {
	if oldConfig.Logging.Level != newConfig.Logging.Level {
		log.Printf("Changing log level from %s to %s", oldConfig.Logging.Level, newConfig.Logging.Level)
		// TODO: 실제 로거의 레벨 변경
	}
	if oldConfig.Logging.Format != newConfig.Logging.Format {
		log.Printf("Changing log format from %s to %s", oldConfig.Logging.Format, newConfig.Logging.Format)
		// TODO: 실제 로거의 포맷 변경
	}
	return nil
}

// Name returns the name of the logging reload handler
func (h *LoggingReloadHandler) Name() string {
	return "LoggingReloadHandler"
}

// CacheReloadHandler 캐시 리로드 핸들러
type CacheReloadHandler struct {
	_ interface{} // reserved for future cache manager implementation
}

// OnConfigReload handles cache configuration changes during reload
func (h *CacheReloadHandler) OnConfigReload(oldConfig, newConfig *UnifiedConfig) error {
	if oldConfig.Cache.Backend != newConfig.Cache.Backend {
		return fmt.Errorf("cache backend change requires restart")
	}
	if oldConfig.Cache.TTL != newConfig.Cache.TTL {
		log.Printf("Updating cache TTL from %s to %s", oldConfig.Cache.TTL, newConfig.Cache.TTL)
		// TODO: 캐시 매니저의 TTL 업데이트
	}
	return nil
}

// Name returns the name of the cache reload handler
func (h *CacheReloadHandler) Name() string {
	return "CacheReloadHandler"
}

// MetricsReloadHandler 메트릭 리로드 핸들러
type MetricsReloadHandler struct {
	_ interface{} // reserved for future metrics server implementation
}

// OnConfigReload handles metrics configuration changes during reload
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

// Name returns the name of the metrics reload handler
func (h *MetricsReloadHandler) Name() string {
	return "MetricsReloadHandler"
}

// SecurityReloadHandler 보안 리로드 핸들러
type SecurityReloadHandler struct {
	_ interface{} // reserved for future auth manager implementation
}

// OnConfigReload handles security configuration changes during reload
func (h *SecurityReloadHandler) OnConfigReload(oldConfig, newConfig *UnifiedConfig) error {
	// Check if BasicAuth configuration changed
	if oldConfig.Security.Authentication.BasicAuth != nil && newConfig.Security.Authentication.BasicAuth != nil {
		if oldConfig.Security.Authentication.BasicAuth.Realm != newConfig.Security.Authentication.BasicAuth.Realm {
			log.Printf("BasicAuth realm changed to: %s", newConfig.Security.Authentication.BasicAuth.Realm)
		}
		// TODO: Reload users if configuration changed
	}
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

// Name returns the name of the security reload handler
func (h *SecurityReloadHandler) Name() string {
	return "SecurityReloadHandler"
}

// Enhanced Hot Reload Methods

// updateFileHash 파일 해시 업데이트
func (uhr *UnifiedHotReload) updateFileHash(filePath string) error {
	hash, err := uhr.calculateFileHash(filePath)
	if err != nil {
		return err
	}

	uhr.fileHashesMu.Lock()
	uhr.fileHashes[filePath] = hash
	uhr.fileHashesMu.Unlock()

	return nil
}

// calculateFileHash 파일 해시 계산
func (uhr *UnifiedHotReload) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	hasher := md5.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// processChangeEvents 변경 이벤트 처리
func (uhr *UnifiedHotReload) processChangeEvents() {
	for {
		select {
		case <-uhr.ctx.Done():
			return
		case event := <-uhr.changeQueue:
			uhr.handleChangeEvent(event)
		}
	}
}

// handleChangeEvent 개별 변경 이벤트 처리
func (uhr *UnifiedHotReload) handleChangeEvent(event ConfigChangeEvent) {
	startTime := time.Now()
	record := ConfigChangeRecord{
		Event:          event,
		HandlerResults: make(map[string]error),
	}

	uhr.logger.Info("Processing config change event",
		logging.F("type", event.Type),
		logging.F("file", event.FilePath))

	// 현재 설정 백업
	uhr.configMu.RLock()
	oldConfig := uhr.config
	uhr.configMu.RUnlock()
	record.OldConfig = oldConfig

	// 백업 생성
	if uhr.backupManager != nil {
		if err := uhr.backupManager.CreateBackup(oldConfig); err != nil {
			uhr.logger.Error("Failed to create config backup", logging.F("error", err))
		}
	}

	// 새 설정 로드 시도
	newConfig, err := uhr.loadConfigWithRetry()
	if err != nil {
		record.Success = false
		record.Error = err.Error()
		record.Duration = time.Since(startTime)
		uhr.addToHistory(record)
		uhr.logger.Error("Failed to load new config", logging.F("error", err))
		return
	}

	record.NewConfig = newConfig

	// 설정 검증
	if uhr.validator != nil {
		validationResult := uhr.validator.Validate(newConfig)
		record.ValidationResult = validationResult

		if !validationResult.Valid {
			switch uhr.validationMode {
			case ValidationModeStrict:
				record.Success = false
				record.Error = "Validation failed in strict mode"
				record.Duration = time.Since(startTime)
				uhr.addToHistory(record)
				uhr.logger.Error("Config validation failed, aborting reload",
					logging.F("errors", len(validationResult.Errors)))
				return
			case ValidationModeWarn:
				uhr.logger.Warn("Config validation warnings",
					logging.F("errors", len(validationResult.Errors)),
					logging.F("warnings", len(validationResult.Warnings)))
			}
		}
	}

	// 헬스 체크
	if uhr.healthMonitor != nil {
		healthStatus := uhr.healthMonitor.CheckHealth(newConfig)
		record.HealthStatus = healthStatus

		if healthStatus.Overall == "unhealthy" && uhr.validationMode == ValidationModeStrict {
			record.Success = false
			record.Error = "Health check failed in strict mode"
			record.Duration = time.Since(startTime)
			uhr.addToHistory(record)
			uhr.logger.Error("Config health check failed, aborting reload",
				logging.F("score", healthStatus.Score))
			return
		}
	}

	// 핸들러 실행
	success := uhr.executeHandlers(oldConfig, newConfig, record.HandlerResults)
	if !success && uhr.rollbackEnabled {
		uhr.logger.Warn("Some handlers failed, attempting rollback")
		if err := uhr.performRollback(oldConfig); err != nil {
			uhr.logger.Error("Rollback failed", logging.F("error", err))
		} else {
			record.RollbackUsed = true
			uhr.logger.Info("Rollback completed successfully")
		}
	}

	// 설정 업데이트
	if success || !uhr.rollbackEnabled {
		uhr.configMu.Lock()
		uhr.config = newConfig
		uhr.configMu.Unlock()
		record.Success = true
		uhr.logger.Info("Configuration updated successfully")
	} else {
		record.Success = false
		record.Error = "Handler execution failed"
	}

	record.Duration = time.Since(startTime)
	uhr.addToHistory(record)
}

// loadConfigWithRetry 재시도를 포함한 설정 로드
func (uhr *UnifiedHotReload) loadConfigWithRetry() (*UnifiedConfig, error) {
	var lastError error

	for attempt := 1; attempt <= uhr.maxRetries; attempt++ {
		var config *UnifiedConfig
		var err error

		if uhr.useViper {
			config, err = uhr.viperLoader.Load()
		} else {
			config, err = uhr.legacyLoader.Load()
		}

		if err == nil {
			return config, nil
		}

		lastError = err
		uhr.logger.Warn("Config load attempt failed",
			logging.F("attempt", attempt),
			logging.F("max_attempts", uhr.maxRetries),
			logging.F("error", err))

		if attempt < uhr.maxRetries {
			time.Sleep(uhr.retryDelay)
		}
	}

	return nil, fmt.Errorf("failed to load config after %d attempts: %w", uhr.maxRetries, lastError)
}

// executeHandlers 핸들러 실행
func (uhr *UnifiedHotReload) executeHandlers(oldConfig, newConfig *UnifiedConfig, results map[string]error) bool {
	uhr.handlersMu.RLock()
	handlers := make([]ReloadHandler, len(uhr.handlers))
	copy(handlers, uhr.handlers)
	uhr.handlersMu.RUnlock()

	allSuccess := true
	for _, handler := range handlers {
		if err := handler.OnConfigReload(oldConfig, newConfig); err != nil {
			results[handler.Name()] = err
			allSuccess = false
			uhr.logger.Error("Reload handler failed",
				logging.F("handler", handler.Name()),
				logging.F("error", err))
		} else {
			results[handler.Name()] = nil
			uhr.logger.Debug("Reload handler succeeded", logging.F("handler", handler.Name()))
		}
	}

	return allSuccess
}

// performRollback 롤백 수행
func (uhr *UnifiedHotReload) performRollback(oldConfig *UnifiedConfig) error {
	uhr.logger.Info("Performing configuration rollback")

	// 설정 복원
	uhr.configMu.Lock()
	uhr.config = oldConfig
	uhr.configMu.Unlock()

	// 핸들러에 롤백 알림
	uhr.handlersMu.RLock()
	handlers := make([]ReloadHandler, len(uhr.handlers))
	copy(handlers, uhr.handlers)
	uhr.handlersMu.RUnlock()

	for _, handler := range handlers {
		if err := handler.OnConfigReload(uhr.config, oldConfig); err != nil {
			uhr.logger.Error("Rollback handler failed",
				logging.F("handler", handler.Name()),
				logging.F("error", err))
		}
	}

	return nil
}

// addToHistory 변경 히스토리에 추가
func (uhr *UnifiedHotReload) addToHistory(record ConfigChangeRecord) {
	uhr.changeHistoryMu.Lock()
	defer uhr.changeHistoryMu.Unlock()

	uhr.changeHistory = append(uhr.changeHistory, record)

	// 최대 크기 초과 시 오래된 기록 제거
	if len(uhr.changeHistory) > uhr.maxHistorySize {
		uhr.changeHistory = uhr.changeHistory[1:]
	}
}

// GetChangeHistory 변경 히스토리 반환
func (uhr *UnifiedHotReload) GetChangeHistory() []ConfigChangeRecord {
	uhr.changeHistoryMu.RLock()
	defer uhr.changeHistoryMu.RUnlock()

	history := make([]ConfigChangeRecord, len(uhr.changeHistory))
	copy(history, uhr.changeHistory)
	return history
}

// SetValidationMode 검증 모드 설정
func (uhr *UnifiedHotReload) SetValidationMode(mode ValidationMode) {
	uhr.validationMode = mode
	uhr.logger.Info("Validation mode changed", logging.F("mode", mode))
}

// SetRollbackEnabled 롤백 기능 활성화/비활성화
func (uhr *UnifiedHotReload) SetRollbackEnabled(enabled bool) {
	uhr.rollbackEnabled = enabled
	uhr.logger.Info("Rollback feature toggled", logging.F("enabled", enabled))
}

// TriggerManualReload 수동 리로드 트리거
func (uhr *UnifiedHotReload) TriggerManualReload() error {
	event := ConfigChangeEvent{
		Type:      "manual_reload",
		Timestamp: time.Now(),
		Metadata:  map[string]string{"trigger": "manual"},
	}

	select {
	case uhr.changeQueue <- event:
		uhr.logger.Info("Manual reload triggered")
		return nil
	default:
		return fmt.Errorf("change queue is full")
	}
}

// GetHealthStatus 현재 헬스 상태 반환
func (uhr *UnifiedHotReload) GetHealthStatus() *HealthStatus {
	if uhr.healthMonitor != nil {
		return uhr.healthMonitor.GetHealthStatus()
	}
	return nil
}

// GetValidationResult 현재 설정의 검증 결과 반환
func (uhr *UnifiedHotReload) GetValidationResult() *ValidationResult {
	if uhr.validator != nil {
		uhr.configMu.RLock()
		config := uhr.config
		uhr.configMu.RUnlock()
		return uhr.validator.Validate(config)
	}
	return nil
}

// Enhanced file watching with hash comparison
//
//nolint:unused // 향후 고급 리로드 기능을 위해 유지
func (uhr *UnifiedHotReload) debounceReloadEnhanced(filePath string) {
	uhr.debounceMu.Lock()
	defer uhr.debounceMu.Unlock()

	if uhr.debounce != nil {
		uhr.debounce.Stop()
	}

	uhr.debounce = time.AfterFunc(uhr.debounceTime, func() {
		// 파일 해시 비교
		newHash, err := uhr.calculateFileHash(filePath)
		if err != nil {
			uhr.logger.Error("Failed to calculate file hash", logging.F("file", filePath), logging.F("error", err))
			return
		}

		uhr.fileHashesMu.RLock()
		oldHash, exists := uhr.fileHashes[filePath]
		uhr.fileHashesMu.RUnlock()

		if exists && oldHash == newHash {
			uhr.logger.Debug("File hash unchanged, skipping reload", logging.F("file", filePath))
			return
		}

		// 해시가 변경되었으므로 리로드 이벤트 생성
		event := ConfigChangeEvent{
			Type:      "file_changed",
			FilePath:  filePath,
			Timestamp: time.Now(),
			OldHash:   oldHash,
			NewHash:   newHash,
			Metadata:  map[string]string{"extension": filepath.Ext(filePath)},
		}

		// 파일 해시 업데이트
		uhr.fileHashesMu.Lock()
		uhr.fileHashes[filePath] = newHash
		uhr.fileHashesMu.Unlock()

		select {
		case uhr.changeQueue <- event:
			uhr.logger.Info("Config file change detected",
				logging.F("file", filePath),
				logging.F("old_hash", oldHash[:8]),
				logging.F("new_hash", newHash[:8]))
		default:
			uhr.logger.Warn("Change queue is full, dropping event", logging.F("file", filePath))
		}
	})
}
