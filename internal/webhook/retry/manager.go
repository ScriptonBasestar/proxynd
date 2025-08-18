package retry

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"proxynd/internal/alerts"
	"proxynd/internal/logging"
)

// Policy 재시도 정책
type Policy struct {
	MaxAttempts   int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
}

// DefaultPolicy 기본 재시도 정책
func DefaultPolicy() Policy {
	return Policy{
		MaxAttempts:   3,
		InitialDelay:  time.Second,
		MaxDelay:      time.Minute,
		BackoffFactor: 2.0,
	}
}

// Item 재시도 항목
type Item struct {
	Event     *alerts.AlertEvent
	Endpoint  string
	Attempt   int
	NextRetry time.Time
	LastError error
	CreatedAt time.Time
}

// Queue 재시도 큐
type Queue struct {
	items    []*Item
	mu       sync.Mutex
	notifyCh chan struct{}
}

// NewQueue 새로운 재시도 큐 생성
func NewQueue() *Queue {
	return &Queue{
		items:    make([]*Item, 0),
		notifyCh: make(chan struct{}, 1),
	}
}

// Push 재시도 항목 추가
func (rq *Queue) Push(item *Item) {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	rq.items = append(rq.items, item)

	// 비동기 알림
	select {
	case rq.notifyCh <- struct{}{}:
	default:
	}
}

// PopReady 준비된 재시도 항목 조회
func (rq *Queue) PopReady() *Item {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	now := time.Now()
	for i, item := range rq.items {
		if now.After(item.NextRetry) || now.Equal(item.NextRetry) {
			// 항목 제거
			rq.items = append(rq.items[:i], rq.items[i+1:]...)
			return item
		}
	}

	return nil
}

// Size 큐 크기 반환
func (rq *Queue) Size() int {
	rq.mu.Lock()
	defer rq.mu.Unlock()
	return len(rq.items)
}

// NotifyCh 알림 채널 반환
func (rq *Queue) NotifyCh() <-chan struct{} {
	return rq.notifyCh
}

// Manager 재시도 관리자
type Manager struct {
	policy Policy
	queue  *Queue
	logger logging.Logger
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewManager 새로운 재시도 관리자 생성
func NewManager(policy Policy) *Manager {
	return &Manager{
		policy: policy,
		queue:  NewQueue(),
		logger: logging.GetLogger(),
		stopCh: make(chan struct{}),
	}
}

// ScheduleRetry 재시도 스케줄링
func (rm *Manager) ScheduleRetry(
	ctx context.Context, event *alerts.AlertEvent, endpoint string, attempt int, lastError error,
) error {
	if attempt >= rm.policy.MaxAttempts {
		rm.logger.Warn(fmt.Sprintf("Max retry attempts reached for event %s to endpoint %s", event.ID, endpoint))
		return fmt.Errorf("max retry attempts (%d) reached", rm.policy.MaxAttempts)
	}

	delay := rm.CalculateBackoffDelay(attempt, rm.policy)
	nextRetry := time.Now().Add(delay)

	item := &Item{
		Event:     event,
		Endpoint:  endpoint,
		Attempt:   attempt + 1,
		NextRetry: nextRetry,
		LastError: lastError,
		CreatedAt: time.Now(),
	}

	rm.queue.Push(item)

	rm.logger.Debug(fmt.Sprintf("Scheduled retry %d for event %s to endpoint %s after %v",
		item.Attempt, event.ID, endpoint, delay))

	return nil
}

// ProcessRetries 재시도 처리
func (rm *Manager) ProcessRetries(ctx context.Context) error {
	rm.wg.Add(1)
	defer rm.wg.Done()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-rm.stopCh:
			return nil
		case <-ticker.C:
			rm.processReadyRetries(ctx)
		case <-rm.queue.NotifyCh():
			rm.processReadyRetries(ctx)
		}
	}
}

// processReadyRetries 준비된 재시도 처리
func (rm *Manager) processReadyRetries(ctx context.Context) {
	for {
		item := rm.queue.PopReady()
		if item == nil {
			break
		}

		// 여기서는 실제 재시도 로직을 호출해야 하지만,
		// 이는 sender에서 구현되어야 하므로 로깅만 수행
		rm.logger.Debug(fmt.Sprintf("Processing retry %d for event %s to endpoint %s",
			item.Attempt, item.Event.ID, item.Endpoint))
	}
}

// CalculateBackoffDelay 백오프 지연 시간 계산
func (rm *Manager) CalculateBackoffDelay(attempt int, policy Policy) time.Duration {
	if attempt <= 0 {
		return policy.InitialDelay
	}

	// 지수 백오프 계산
	delay := float64(policy.InitialDelay) * math.Pow(policy.BackoffFactor, float64(attempt-1))

	// 최대 지연 시간 제한
	if delay > float64(policy.MaxDelay) {
		delay = float64(policy.MaxDelay)
	}

	return time.Duration(delay)
}

// IsRetryableError 재시도 가능한 오류인지 확인
func (rm *Manager) IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// 일반적인 재시도 가능한 오류 패턴
	errorMsg := err.Error()
	retryablePatterns := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"temporary failure",
		"service unavailable",
		"internal server error",
		"bad gateway",
		"gateway timeout",
	}

	for _, pattern := range retryablePatterns {
		if contains(errorMsg, pattern) {
			return true
		}
	}

	return false
}

// Stop 재시도 관리자 중지
func (rm *Manager) Stop() {
	close(rm.stopCh)
	rm.wg.Wait()
}

// GetStats 재시도 통계 반환
func (rm *Manager) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"queue_size":     rm.queue.Size(),
		"max_attempts":   rm.policy.MaxAttempts,
		"initial_delay":  rm.policy.InitialDelay.String(),
		"max_delay":      rm.policy.MaxDelay.String(),
		"backoff_factor": rm.policy.BackoffFactor,
	}
}

// contains 문자열 포함 여부 확인 (간단한 구현)
func contains(str, substr string) bool {
	return len(str) >= len(substr) && str[:len(substr)] == substr ||
		len(str) > len(substr) && str[len(str)-len(substr):] == substr ||
		findSubstring(str, substr)
}

// findSubstring 부분 문자열 검색
func findSubstring(str, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(str) < len(substr) {
		return false
	}

	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
