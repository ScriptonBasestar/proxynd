// Package health provides auto-recovery mechanisms for system resilience
package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"proxynd/internal/logging"
)

// RecoveryStrategy 복구 전략 타입
type RecoveryStrategy string

const (
	// StrategyRestart 서비스 재시작
	StrategyRestart RecoveryStrategy = "restart"
	// StrategyReset 상태 리셋
	StrategyReset RecoveryStrategy = "reset"
	// StrategyFallback 대체 서비스로 전환
	StrategyFallback RecoveryStrategy = "fallback"
	// StrategyCircuitBreaker 서킷 브레이커 활성화
	StrategyCircuitBreaker RecoveryStrategy = "circuit_breaker"
	// StrategyCustom 사용자 정의 복구
	StrategyCustom RecoveryStrategy = "custom"
)

// RecoveryAction 복구 액션 인터페이스
type RecoveryAction interface {
	Execute(ctx context.Context, trigger *RecoveryTrigger) error
	Name() string
	Description() string
}

// RecoveryTrigger 복구 트리거 정보
type RecoveryTrigger struct {
	ComponentName   string                 `json:"component_name"`
	TriggerTime     time.Time              `json:"trigger_time"`
	FailureReason   string                 `json:"failure_reason"`
	FailureCount    int                    `json:"failure_count"`
	LastFailureTime time.Time              `json:"last_failure_time"`
	HealthStatus    Status                 `json:"health_status"`
	Details         map[string]interface{} `json:"details"`
	AttemptCount    int                    `json:"attempt_count"`
}

// RecoveryConfig 복구 설정
type RecoveryConfig struct {
	ComponentName    string           `json:"component_name"`
	FailureThreshold int              `json:"failure_threshold"`  // 복구 트리거 실패 횟수
	CheckInterval    time.Duration    `json:"check_interval"`     // 상태 확인 간격
	RecoveryTimeout  time.Duration    `json:"recovery_timeout"`   // 복구 작업 타임아웃
	MaxRetryAttempts int              `json:"max_retry_attempts"` // 최대 재시도 횟수
	RetryBackoff     time.Duration    `json:"retry_backoff"`      // 재시도 간격
	Strategy         RecoveryStrategy `json:"strategy"`           // 복구 전략
	Enabled          bool             `json:"enabled"`            // 자동 복구 활성화
	CooldownPeriod   time.Duration    `json:"cooldown_period"`    // 복구 후 대기 시간
	NotifyOnRecovery bool             `json:"notify_on_recovery"` // 복구 완료 알림
	NotifyOnFailure  bool             `json:"notify_on_failure"`  // 복구 실패 알림
}

// DefaultRecoveryConfig 기본 복구 설정
func DefaultRecoveryConfig(componentName string) *RecoveryConfig {
	return &RecoveryConfig{
		ComponentName:    componentName,
		FailureThreshold: 3,
		CheckInterval:    30 * time.Second,
		RecoveryTimeout:  60 * time.Second,
		MaxRetryAttempts: 3,
		RetryBackoff:     10 * time.Second,
		Strategy:         StrategyRestart,
		Enabled:          true,
		CooldownPeriod:   5 * time.Minute,
		NotifyOnRecovery: true,
		NotifyOnFailure:  true,
	}
}

// RecoveryHistory 복구 이력
type RecoveryHistory struct {
	Timestamp     time.Time        `json:"timestamp"`
	ComponentName string           `json:"component_name"`
	Strategy      RecoveryStrategy `json:"strategy"`
	Success       bool             `json:"success"`
	Duration      time.Duration    `json:"duration"`
	Error         string           `json:"error,omitempty"`
	AttemptCount  int              `json:"attempt_count"`
}

// AutoRecoveryService 자동 복구 서비스
type AutoRecoveryService struct {
	configs        map[string]*RecoveryConfig
	actions        map[RecoveryStrategy]RecoveryAction
	healthService  *HealthService
	triggers       map[string]*RecoveryTrigger
	history        []*RecoveryHistory
	mutex          sync.RWMutex
	logger         logging.Logger
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	maxHistorySize int
	onRecovery     func(string, bool, time.Duration) // 복구 완료 콜백
}

// NewAutoRecoveryService 새 자동 복구 서비스 생성
func NewAutoRecoveryService(healthService *HealthService) *AutoRecoveryService {
	ctx, cancel := context.WithCancel(context.Background())

	service := &AutoRecoveryService{
		configs:        make(map[string]*RecoveryConfig),
		actions:        make(map[RecoveryStrategy]RecoveryAction),
		healthService:  healthService,
		triggers:       make(map[string]*RecoveryTrigger),
		history:        make([]*RecoveryHistory, 0),
		logger:         logging.GetLogger(),
		ctx:            ctx,
		cancel:         cancel,
		maxHistorySize: 1000,
	}

	// 기본 복구 액션 등록
	service.registerDefaultActions()

	return service
}

// RegisterRecoveryConfig 복구 설정 등록
func (ars *AutoRecoveryService) RegisterRecoveryConfig(config *RecoveryConfig) {
	ars.mutex.Lock()
	defer ars.mutex.Unlock()

	ars.configs[config.ComponentName] = config

	ars.logger.Info("Recovery config registered",
		logging.F("component", config.ComponentName),
		logging.F("strategy", string(config.Strategy)),
		logging.F("failure_threshold", config.FailureThreshold))
}

// RegisterRecoveryAction 복구 액션 등록
func (ars *AutoRecoveryService) RegisterRecoveryAction(strategy RecoveryStrategy, action RecoveryAction) {
	ars.mutex.Lock()
	defer ars.mutex.Unlock()

	ars.actions[strategy] = action

	ars.logger.Info("Recovery action registered",
		logging.F("strategy", string(strategy)),
		logging.F("action", action.Name()))
}

// SetRecoveryCallback 복구 완료 콜백 설정
func (ars *AutoRecoveryService) SetRecoveryCallback(callback func(string, bool, time.Duration)) {
	ars.onRecovery = callback
}

// Start 자동 복구 서비스 시작
func (ars *AutoRecoveryService) Start() {
	ars.wg.Add(1)
	go ars.monitorHealth()

	ars.logger.Info("Auto-recovery service started",
		logging.F("configs_count", len(ars.configs)))
}

// Stop 자동 복구 서비스 정지
func (ars *AutoRecoveryService) Stop() {
	ars.cancel()
	ars.wg.Wait()

	ars.logger.Info("Auto-recovery service stopped")
}

// monitorHealth 헬스 상태 모니터링
func (ars *AutoRecoveryService) monitorHealth() {
	defer ars.wg.Done()

	ticker := time.NewTicker(10 * time.Second) // 기본 체크 간격
	defer ticker.Stop()

	for {
		select {
		case <-ars.ctx.Done():
			return
		case <-ticker.C:
			ars.checkAndRecover()
		}
	}
}

// checkAndRecover 상태 확인 및 복구 수행
func (ars *AutoRecoveryService) checkAndRecover() {
	if ars.healthService == nil {
		return
	}

	// 전체 헬스 상태 조회
	overallStatus, results := ars.healthService.GetStatus()

	ars.mutex.Lock()
	configs := make(map[string]*RecoveryConfig)
	for k, v := range ars.configs {
		configs[k] = v
	}
	ars.mutex.Unlock()

	// 각 구성 요소별 상태 확인
	for componentName, config := range configs {
		if !config.Enabled {
			continue
		}

		// 해당 구성 요소 결과 찾기
		var checkResult *CheckResult
		for _, result := range results {
			if result.Name == componentName {
				checkResult = result
				break
			}
		}

		if checkResult == nil {
			continue
		}

		// 실패 상태 확인
		switch checkResult.Status {
		case StatusUnhealthy:
			ars.handleFailure(componentName, checkResult, config)
		case StatusHealthy:
			ars.handleRecovery(componentName)
		}
	}

	// 전체 상태가 불량할 때의 처리
	if overallStatus == StatusUnhealthy {
		ars.handleSystemFailure(results)
	}
}

// handleFailure 실패 처리
func (ars *AutoRecoveryService) handleFailure(componentName string, result *CheckResult, config *RecoveryConfig) {
	ars.mutex.Lock()
	defer ars.mutex.Unlock()

	now := time.Now()

	// 트리거 정보 업데이트
	trigger, exists := ars.triggers[componentName]
	if !exists {
		trigger = &RecoveryTrigger{
			ComponentName: componentName,
			TriggerTime:   now,
			Details:       make(map[string]interface{}),
		}
		ars.triggers[componentName] = trigger
	}

	trigger.FailureCount++
	trigger.LastFailureTime = now
	trigger.FailureReason = result.Message
	trigger.HealthStatus = result.Status
	trigger.Details = result.Details

	// 실패 임계값 확인
	if trigger.FailureCount >= config.FailureThreshold {
		// 쿨다운 기간 확인
		if trigger.AttemptCount > 0 {
			lastAttempt := trigger.TriggerTime
			if now.Sub(lastAttempt) < config.CooldownPeriod {
				return // 쿨다운 기간 중
			}
		}

		// 최대 재시도 횟수 확인
		if trigger.AttemptCount >= config.MaxRetryAttempts {
			ars.logger.Warn("Max recovery attempts reached",
				logging.F("component", componentName),
				logging.F("attempts", trigger.AttemptCount))
			return
		}

		// 복구 시도
		go ars.performRecovery(componentName, trigger, config)
	}
}

// handleRecovery 복구 처리
func (ars *AutoRecoveryService) handleRecovery(componentName string) {
	ars.mutex.Lock()
	defer ars.mutex.Unlock()

	// 트리거 정보 리셋
	if trigger, exists := ars.triggers[componentName]; exists {
		if trigger.FailureCount > 0 {
			ars.logger.Info("Component recovered naturally",
				logging.F("component", componentName),
				logging.F("failure_count", trigger.FailureCount))
		}
		delete(ars.triggers, componentName)
	}
}

// handleSystemFailure 시스템 전체 실패 처리
func (ars *AutoRecoveryService) handleSystemFailure(results map[string]*CheckResult) {
	// 시스템 전체적인 복구 로직
	unhealthyCount := 0
	for _, result := range results {
		if result.Status == StatusUnhealthy {
			unhealthyCount++
		}
	}

	if unhealthyCount > len(results)/2 {
		ars.logger.Warn("System-wide health issues detected",
			logging.F("unhealthy_components", unhealthyCount),
			logging.F("total_components", len(results)))

		// 시스템 레벨 복구 액션 수행
		// 예: 서킷 브레이커 활성화, 로드 밸런싱 조정 등
	}
}

// performRecovery 복구 수행
func (ars *AutoRecoveryService) performRecovery(componentName string, trigger *RecoveryTrigger, config *RecoveryConfig) { //nolint:lll
	start := time.Now()
	trigger.AttemptCount++

	ars.logger.Info("Starting recovery attempt",
		logging.F("component", componentName),
		logging.F("strategy", string(config.Strategy)),
		logging.F("attempt", trigger.AttemptCount))

	// 복구 액션 조회
	ars.mutex.RLock()
	action, exists := ars.actions[config.Strategy]
	ars.mutex.RUnlock()

	if !exists {
		ars.logger.Error("Recovery action not found",
			logging.F("component", componentName),
			logging.F("strategy", string(config.Strategy)))
		return
	}

	// 복구 실행 (타임아웃 포함)
	ctx, cancel := context.WithTimeout(ars.ctx, config.RecoveryTimeout)
	defer cancel()

	err := action.Execute(ctx, trigger)
	duration := time.Since(start)

	// 결과 기록
	history := &RecoveryHistory{
		Timestamp:     start,
		ComponentName: componentName,
		Strategy:      config.Strategy,
		Success:       err == nil,
		Duration:      duration,
		AttemptCount:  trigger.AttemptCount,
	}

	if err != nil {
		history.Error = err.Error()
		ars.logger.Error("Recovery attempt failed",
			logging.F("component", componentName),
			logging.F("strategy", string(config.Strategy)),
			logging.F("attempt", trigger.AttemptCount),
			logging.F("duration", duration),
			logging.F("error", err))
	} else {
		ars.logger.Info("Recovery attempt succeeded",
			logging.F("component", componentName),
			logging.F("strategy", string(config.Strategy)),
			logging.F("attempt", trigger.AttemptCount),
			logging.F("duration", duration))

		// 성공 시 트리거 리셋
		ars.mutex.Lock()
		delete(ars.triggers, componentName)
		ars.mutex.Unlock()
	}

	// 이력 추가
	ars.addHistory(history)

	// 콜백 호출
	if ars.onRecovery != nil {
		go ars.onRecovery(componentName, err == nil, duration)
	}
}

// addHistory 이력 추가
func (ars *AutoRecoveryService) addHistory(history *RecoveryHistory) {
	ars.mutex.Lock()
	defer ars.mutex.Unlock()

	ars.history = append(ars.history, history)

	// 이력 크기 제한
	if len(ars.history) > ars.maxHistorySize {
		copy(ars.history, ars.history[1:])
		ars.history = ars.history[:ars.maxHistorySize]
	}
}

// GetRecoveryHistory 복구 이력 조회
func (ars *AutoRecoveryService) GetRecoveryHistory(limit int) []*RecoveryHistory {
	ars.mutex.RLock()
	defer ars.mutex.RUnlock()

	if limit <= 0 || limit > len(ars.history) {
		limit = len(ars.history)
	}

	// 최근 이력부터 반환
	start := len(ars.history) - limit
	result := make([]*RecoveryHistory, limit)
	copy(result, ars.history[start:])

	return result
}

// GetRecoveryStatus 복구 상태 조회
func (ars *AutoRecoveryService) GetRecoveryStatus() map[string]interface{} {
	ars.mutex.RLock()
	defer ars.mutex.RUnlock()

	status := map[string]interface{}{
		"configs_count":   len(ars.configs),
		"actions_count":   len(ars.actions),
		"active_triggers": len(ars.triggers),
		"history_count":   len(ars.history),
	}

	// 활성 트리거 정보
	if len(ars.triggers) > 0 {
		triggers := make(map[string]interface{})
		for name, trigger := range ars.triggers {
			triggers[name] = map[string]interface{}{
				"failure_count":  trigger.FailureCount,
				"last_failure":   trigger.LastFailureTime,
				"attempt_count":  trigger.AttemptCount,
				"failure_reason": trigger.FailureReason,
			}
		}
		status["triggers"] = triggers
	}

	// 최근 복구 이력
	recentHistory := ars.history
	if len(recentHistory) > 10 {
		recentHistory = recentHistory[len(recentHistory)-10:]
	}
	status["recent_history"] = recentHistory

	return status
}

// TriggerManualRecovery 수동 복구 트리거
func (ars *AutoRecoveryService) TriggerManualRecovery(componentName string, strategy RecoveryStrategy) error {
	ars.mutex.RLock()
	config, configExists := ars.configs[componentName]
	_, actionExists := ars.actions[strategy]
	ars.mutex.RUnlock()

	if !configExists {
		return fmt.Errorf("recovery config not found for component: %s", componentName)
	}

	if !actionExists {
		return fmt.Errorf("recovery action not found for strategy: %s", strategy)
	}

	// 수동 트리거 생성
	trigger := &RecoveryTrigger{
		ComponentName:   componentName,
		TriggerTime:     time.Now(),
		FailureReason:   "manual_trigger",
		FailureCount:    config.FailureThreshold, // 임계값 이상으로 설정
		LastFailureTime: time.Now(),
		HealthStatus:    StatusUnhealthy,
		Details:         map[string]interface{}{"manual": true},
		AttemptCount:    0,
	}

	// 복구 실행
	go ars.performRecovery(componentName, trigger, config)

	ars.logger.Info("Manual recovery triggered",
		logging.F("component", componentName),
		logging.F("strategy", string(strategy)))

	return nil
}

// registerDefaultActions 기본 복구 액션 등록
func (ars *AutoRecoveryService) registerDefaultActions() {
	// 리스타트 액션
	ars.RegisterRecoveryAction(StrategyRestart, &RestartAction{})

	// 리셋 액션
	ars.RegisterRecoveryAction(StrategyReset, &ResetAction{})

	// 서킷 브레이커 액션
	ars.RegisterRecoveryAction(StrategyCircuitBreaker, &CircuitBreakerAction{})
}

// 기본 복구 액션 구현들

// RestartAction 재시작 복구 액션
type RestartAction struct{}

func (ra *RestartAction) Name() string {
	return "restart"
}

func (ra *RestartAction) Description() string {
	return "Restart the component service"
}

func (ra *RestartAction) Execute(ctx context.Context, trigger *RecoveryTrigger) error {
	// 실제 재시작 로직은 구체적인 구현에 따라 달라짐
	// 여기서는 시뮬레이션
	time.Sleep(2 * time.Second) // 재시작 시뮬레이션
	return nil
}

// ResetAction 리셋 복구 액션
type ResetAction struct{}

func (ra *ResetAction) Name() string {
	return "reset"
}

func (ra *ResetAction) Description() string {
	return "Reset component state"
}

func (ra *ResetAction) Execute(ctx context.Context, trigger *RecoveryTrigger) error {
	// 상태 리셋 로직
	time.Sleep(1 * time.Second) // 리셋 시뮬레이션
	return nil
}

// CircuitBreakerAction 서킷 브레이커 액션
type CircuitBreakerAction struct{}

func (cba *CircuitBreakerAction) Name() string {
	return "circuit_breaker"
}

func (cba *CircuitBreakerAction) Description() string {
	return "Activate circuit breaker for component"
}

func (cba *CircuitBreakerAction) Execute(ctx context.Context, trigger *RecoveryTrigger) error {
	// 서킷 브레이커 활성화
	cbManager := GetCircuitBreakerManager()
	cb := cbManager.GetOrCreate(trigger.ComponentName, nil)

	// 서킷 브레이커를 강제로 열기
	cb.Reset() // 먼저 리셋 후

	// 실패 시뮬레이션으로 서킷 브레이커 열기
	for i := 0; i < 10; i++ {
		_ = cb.Execute(ctx, func(context.Context) error {
			return fmt.Errorf("simulated failure to open circuit breaker")
		})
	}

	return nil
}

// 전역 자동 복구 서비스
var globalAutoRecoveryService *AutoRecoveryService

// InitGlobalAutoRecovery 전역 자동 복구 서비스 초기화
func InitGlobalAutoRecovery(healthService *HealthService) {
	globalAutoRecoveryService = NewAutoRecoveryService(healthService)
}

// GetGlobalAutoRecovery 전역 자동 복구 서비스 반환
func GetGlobalAutoRecovery() *AutoRecoveryService {
	return globalAutoRecoveryService
}
