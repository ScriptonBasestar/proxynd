// Package health provides circuit breaker implementation for auto-recovery
package health

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"proxynd/logging"
)

// CircuitBreakerState 서킷 브레이커 상태
type CircuitBreakerState string

const (
	// StateClosed 정상 상태 - 요청이 통과
	StateClosed CircuitBreakerState = "closed"
	// StateOpen 차단 상태 - 요청이 차단됨
	StateOpen CircuitBreakerState = "open"
	// StateHalfOpen 반열림 상태 - 테스트 요청만 허용
	StateHalfOpen CircuitBreakerState = "half_open"
)

// CircuitBreakerConfig 서킷 브레이커 설정
type CircuitBreakerConfig struct {
	Name               string        `json:"name"`
	FailureThreshold   int           `json:"failure_threshold"`   // 실패 임계값
	SuccessThreshold   int           `json:"success_threshold"`   // 성공 임계값 (half-open에서 closed로)
	Timeout            time.Duration `json:"timeout"`             // open 상태 유지 시간
	MaxRequests        int           `json:"max_requests"`        // half-open 상태에서 최대 요청 수
	ResetTimeout       time.Duration `json:"reset_timeout"`       // 통계 리셋 시간
	EnableMetrics      bool          `json:"enable_metrics"`      // 메트릭 수집 활성화
	EnableNotification bool          `json:"enable_notification"` // 상태 변경 알림 활성화
}

// DefaultCircuitBreakerConfig 기본 서킷 브레이커 설정
func DefaultCircuitBreakerConfig(name string) *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		Name:               name,
		FailureThreshold:   5,
		SuccessThreshold:   3,
		Timeout:            60 * time.Second,
		MaxRequests:        3,
		ResetTimeout:       300 * time.Second,
		EnableMetrics:      true,
		EnableNotification: true,
	}
}

// CircuitBreakerMetrics 서킷 브레이커 메트릭
type CircuitBreakerMetrics struct {
	mutex                sync.RWMutex
	TotalRequests        int64     `json:"total_requests"`
	SuccessfulRequests   int64     `json:"successful_requests"`
	FailedRequests       int64     `json:"failed_requests"`
	RejectedRequests     int64     `json:"rejected_requests"`
	StateChanges         int64     `json:"state_changes"`
	LastStateChange      time.Time `json:"last_state_change"`
	ConsecutiveFailures  int       `json:"consecutive_failures"`
	ConsecutiveSuccesses int       `json:"consecutive_successes"`
}

// CircuitBreaker 서킷 브레이커 구현
type CircuitBreaker struct {
	config        *CircuitBreakerConfig
	state         CircuitBreakerState
	stateExpiry   time.Time
	mutex         sync.RWMutex
	metrics       *CircuitBreakerMetrics
	logger        logging.Logger
	onStateChange func(string, CircuitBreakerState, CircuitBreakerState) // 상태 변경 콜백
}

// NewCircuitBreaker 새 서킷 브레이커 생성
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	if config == nil {
		config = DefaultCircuitBreakerConfig("default")
	}

	cb := &CircuitBreaker{
		config:      config,
		state:       StateClosed,
		stateExpiry: time.Time{},
		metrics: &CircuitBreakerMetrics{
			LastStateChange: time.Now(),
		},
		logger: logging.GetLogger(),
	}

	return cb
}

// SetStateChangeCallback 상태 변경 콜백 설정
func (cb *CircuitBreaker) SetStateChangeCallback(callback func(string, CircuitBreakerState, CircuitBreakerState)) {
	cb.onStateChange = callback
}

// Execute 서킷 브레이커를 통해 함수 실행
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	// 현재 상태 확인
	if !cb.allow() {
		cb.recordRejection()
		return fmt.Errorf("circuit breaker '%s' is open", cb.config.Name)
	}

	// 함수 실행
	err := fn(ctx)
	// 결과 기록
	if err != nil {
		cb.recordFailure()
		return err
	}

	cb.recordSuccess()
	return nil
}

// Call 서킷 브레이커를 통해 함수 호출 (반환값 포함)
func (cb *CircuitBreaker) Call(ctx context.Context, fn func(context.Context) (interface{}, error)) (interface{}, error) { //nolint:lll
	// 현재 상태 확인
	if !cb.allow() {
		cb.recordRejection()
		return nil, fmt.Errorf("circuit breaker '%s' is open", cb.config.Name)
	}

	// 함수 실행
	result, err := fn(ctx)
	// 결과 기록
	if err != nil {
		cb.recordFailure()
		return nil, err
	}

	cb.recordSuccess()
	return result, nil
}

// allow 요청 허용 여부 확인
func (cb *CircuitBreaker) allow() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		return true

	case StateOpen:
		// 타임아웃 확인
		if now.After(cb.stateExpiry) {
			cb.setState(StateHalfOpen)
			cb.stateExpiry = time.Time{}
			return true
		}
		return false

	case StateHalfOpen:
		// 최대 요청 수 확인
		return cb.metrics.TotalRequests < int64(cb.config.MaxRequests)

	default:
		return false
	}
}

// recordSuccess 성공 기록
func (cb *CircuitBreaker) recordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.metrics.mutex.Lock()
	cb.metrics.TotalRequests++
	cb.metrics.SuccessfulRequests++
	cb.metrics.ConsecutiveSuccesses++
	cb.metrics.ConsecutiveFailures = 0
	cb.metrics.mutex.Unlock()

	// 상태 전환 체크
	if cb.state == StateHalfOpen {
		if cb.metrics.ConsecutiveSuccesses >= cb.config.SuccessThreshold {
			cb.setState(StateClosed)
			cb.resetMetrics()
		}
	}
}

// recordFailure 실패 기록
func (cb *CircuitBreaker) recordFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.metrics.mutex.Lock()
	cb.metrics.TotalRequests++
	cb.metrics.FailedRequests++
	cb.metrics.ConsecutiveFailures++
	cb.metrics.ConsecutiveSuccesses = 0
	cb.metrics.mutex.Unlock()

	// 상태 전환 체크
	if cb.state == StateClosed || cb.state == StateHalfOpen {
		if cb.metrics.ConsecutiveFailures >= cb.config.FailureThreshold {
			cb.setState(StateOpen)
			cb.stateExpiry = time.Now().Add(cb.config.Timeout)
		}
	}
}

// recordRejection 거부 기록
func (cb *CircuitBreaker) recordRejection() {
	cb.metrics.mutex.Lock()
	cb.metrics.RejectedRequests++
	cb.metrics.mutex.Unlock()
}

// setState 상태 변경
func (cb *CircuitBreaker) setState(newState CircuitBreakerState) {
	oldState := cb.state
	cb.state = newState

	cb.metrics.mutex.Lock()
	cb.metrics.StateChanges++
	cb.metrics.LastStateChange = time.Now()
	cb.metrics.mutex.Unlock()

	// 로깅
	cb.logger.Info("Circuit breaker state changed",
		logging.F("name", cb.config.Name),
		logging.F("old_state", string(oldState)),
		logging.F("new_state", string(newState)),
		logging.F("consecutive_failures", cb.metrics.ConsecutiveFailures),
		logging.F("consecutive_successes", cb.metrics.ConsecutiveSuccesses))

	// 콜백 호출
	if cb.onStateChange != nil {
		go cb.onStateChange(cb.config.Name, oldState, newState)
	}
}

// resetMetrics 메트릭 리셋
func (cb *CircuitBreaker) resetMetrics() {
	cb.metrics.mutex.Lock()
	defer cb.metrics.mutex.Unlock()

	cb.metrics.ConsecutiveFailures = 0
	cb.metrics.ConsecutiveSuccesses = 0
	// 총 요청 수와 상태 변경 기록은 유지
}

// GetState 현재 상태 반환
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// GetMetrics 메트릭 반환
func (cb *CircuitBreaker) GetMetrics() *CircuitBreakerMetrics {
	cb.metrics.mutex.RLock()
	defer cb.metrics.mutex.RUnlock()

	// 복사본 반환
	return &CircuitBreakerMetrics{
		TotalRequests:        cb.metrics.TotalRequests,
		SuccessfulRequests:   cb.metrics.SuccessfulRequests,
		FailedRequests:       cb.metrics.FailedRequests,
		RejectedRequests:     cb.metrics.RejectedRequests,
		StateChanges:         cb.metrics.StateChanges,
		LastStateChange:      cb.metrics.LastStateChange,
		ConsecutiveFailures:  cb.metrics.ConsecutiveFailures,
		ConsecutiveSuccesses: cb.metrics.ConsecutiveSuccesses,
	}
}

// GetStatus 상태 정보 반환
func (cb *CircuitBreaker) GetStatus() map[string]interface{} {
	cb.mutex.RLock()
	state := cb.state
	stateExpiry := cb.stateExpiry
	cb.mutex.RUnlock()

	metrics := cb.GetMetrics()

	status := map[string]interface{}{
		"name":         cb.config.Name,
		"state":        string(state),
		"state_expiry": stateExpiry,
		"config":       cb.config,
		"metrics":      metrics,
	}

	// 상태별 추가 정보
	switch state {
	case StateOpen:
		if !stateExpiry.IsZero() {
			status["time_until_half_open"] = time.Until(stateExpiry).Seconds()
		}
	case StateHalfOpen:
		status["remaining_test_requests"] = cb.config.MaxRequests - int(metrics.TotalRequests)
	}

	return status
}

// Reset 서킷 브레이커 리셋
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	oldState := cb.state
	cb.state = StateClosed
	cb.stateExpiry = time.Time{}

	cb.metrics.mutex.Lock()
	cb.metrics.TotalRequests = 0
	cb.metrics.SuccessfulRequests = 0
	cb.metrics.FailedRequests = 0
	cb.metrics.RejectedRequests = 0
	cb.metrics.ConsecutiveFailures = 0
	cb.metrics.ConsecutiveSuccesses = 0
	cb.metrics.StateChanges++
	cb.metrics.LastStateChange = time.Now()
	cb.metrics.mutex.Unlock()

	cb.logger.Info("Circuit breaker reset",
		logging.F("name", cb.config.Name),
		logging.F("old_state", string(oldState)))

	// 콜백 호출
	if cb.onStateChange != nil {
		go cb.onStateChange(cb.config.Name, oldState, StateClosed)
	}
}

// CircuitBreakerManager 서킷 브레이커 관리자
type CircuitBreakerManager struct {
	breakers map[string]*CircuitBreaker
	mutex    sync.RWMutex
	logger   logging.Logger
}

// NewCircuitBreakerManager 새 서킷 브레이커 관리자 생성
func NewCircuitBreakerManager() *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreaker),
		logger:   logging.GetLogger(),
	}
}

// GetOrCreate 서킷 브레이커 조회 또는 생성
func (cbm *CircuitBreakerManager) GetOrCreate(name string, config *CircuitBreakerConfig) *CircuitBreaker {
	cbm.mutex.Lock()
	defer cbm.mutex.Unlock()

	if cb, exists := cbm.breakers[name]; exists {
		return cb
	}

	if config == nil {
		config = DefaultCircuitBreakerConfig(name)
	} else {
		config.Name = name
	}

	cb := NewCircuitBreaker(config)
	cbm.breakers[name] = cb

	cbm.logger.Info("Circuit breaker created",
		logging.F("name", name),
		logging.F("failure_threshold", config.FailureThreshold),
		logging.F("timeout", config.Timeout))

	return cb
}

// Get 서킷 브레이커 조회
func (cbm *CircuitBreakerManager) Get(name string) (*CircuitBreaker, error) {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	if cb, exists := cbm.breakers[name]; exists {
		return cb, nil
	}

	return nil, fmt.Errorf("circuit breaker '%s' not found", name)
}

// List 모든 서킷 브레이커 목록 반환
func (cbm *CircuitBreakerManager) List() map[string]*CircuitBreaker {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	result := make(map[string]*CircuitBreaker)
	for name, cb := range cbm.breakers {
		result[name] = cb
	}

	return result
}

// GetStatus 모든 서킷 브레이커 상태 반환
func (cbm *CircuitBreakerManager) GetStatus() map[string]interface{} {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	status := make(map[string]interface{})
	for name, cb := range cbm.breakers {
		status[name] = cb.GetStatus()
	}

	return status
}

// Reset 모든 서킷 브레이커 리셋
func (cbm *CircuitBreakerManager) Reset() {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	for _, cb := range cbm.breakers {
		cb.Reset()
	}

	cbm.logger.Info("All circuit breakers reset")
}

// Remove 서킷 브레이커 제거
func (cbm *CircuitBreakerManager) Remove(name string) {
	cbm.mutex.Lock()
	defer cbm.mutex.Unlock()

	delete(cbm.breakers, name)
	cbm.logger.Info("Circuit breaker removed", logging.F("name", name))
}

// 전역 서킷 브레이커 관리자
var globalCircuitBreakerManager = NewCircuitBreakerManager()

// GetCircuitBreaker 전역 서킷 브레이커 조회 또는 생성
func GetCircuitBreaker(name string, config *CircuitBreakerConfig) *CircuitBreaker {
	return globalCircuitBreakerManager.GetOrCreate(name, config)
}

// GetCircuitBreakerManager 전역 서킷 브레이커 관리자 반환
func GetCircuitBreakerManager() *CircuitBreakerManager {
	return globalCircuitBreakerManager
}

// CircuitBreakerHealthChecker 서킷 브레이커 상태를 모니터링하는 헬스 체커
type CircuitBreakerHealthChecker struct {
	manager *CircuitBreakerManager
}

// NewCircuitBreakerHealthChecker 새 서킷 브레이커 헬스 체커 생성
func NewCircuitBreakerHealthChecker(manager *CircuitBreakerManager) *CircuitBreakerHealthChecker {
	if manager == nil {
		manager = globalCircuitBreakerManager
	}
	return &CircuitBreakerHealthChecker{manager: manager}
}

// Name 체커 이름 반환
func (cbhc *CircuitBreakerHealthChecker) Name() string {
	return "circuit_breakers"
}

// Check 서킷 브레이커들의 상태 확인
func (cbhc *CircuitBreakerHealthChecker) Check(_ context.Context) *CheckResult {
	start := time.Now()
	result := &CheckResult{
		Name:        cbhc.Name(),
		Status:      StatusHealthy,
		LastChecked: time.Now(),
		Details:     make(map[string]interface{}),
	}

	breakers := cbhc.manager.List()
	if len(breakers) == 0 {
		result.Message = "No circuit breakers configured"
		result.Duration = time.Since(start)
		return result
	}

	openCount := 0
	halfOpenCount := 0
	closedCount := 0
	breakerDetails := make(map[string]interface{})

	for name, cb := range breakers {
		state := cb.GetState()
		metrics := cb.GetMetrics()

		breakerDetails[name] = map[string]interface{}{
			"state":                string(state),
			"consecutive_failures": metrics.ConsecutiveFailures,
			"total_requests":       metrics.TotalRequests,
			"success_rate":         float64(metrics.SuccessfulRequests) / float64(max(metrics.TotalRequests, 1)) * 100,
		}

		switch state {
		case StateOpen:
			openCount++
		case StateHalfOpen:
			halfOpenCount++
		case StateClosed:
			closedCount++
		}
	}

	result.Details["total_breakers"] = len(breakers)
	result.Details["open_breakers"] = openCount
	result.Details["half_open_breakers"] = halfOpenCount
	result.Details["closed_breakers"] = closedCount
	result.Details["breakers"] = breakerDetails

	// 전체 상태 결정
	if openCount > 0 {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("%d circuit breaker(s) are open", openCount)
	} else if halfOpenCount > 0 {
		result.Status = StatusDegraded
		result.Message = fmt.Sprintf("%d circuit breaker(s) are half-open", halfOpenCount)
	} else {
		result.Message = fmt.Sprintf("All %d circuit breaker(s) are closed", closedCount)
	}

	result.Duration = time.Since(start)
	return result
}

// max 헬퍼 함수
func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// ErrCircuitBreakerOpen 서킷 브레이커가 열려있을 때 반환되는 에러
var ErrCircuitBreakerOpen = errors.New("circuit breaker is open")

// IsCircuitBreakerError 서킷 브레이커 에러인지 확인
func IsCircuitBreakerError(err error) bool {
	return errors.Is(err, ErrCircuitBreakerOpen)
}
