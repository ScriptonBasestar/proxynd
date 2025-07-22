package webhook

import (
	"context"
	"time"

	"proxynd/alerts"
)

// SenderInterface 웹훅 전송기 인터페이스
type SenderInterface interface {
	// 생명주기 관리
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	
	// 이벤트 전송
	SendEvent(event *alerts.AlertEvent) error
	SendEventSync(ctx context.Context, event *alerts.AlertEvent) error
	
	// 어댑터 관리
	RegisterAdapter(adapter WebhookAdapter)
	
	// 메트릭 조회
	GetMetrics() *SenderMetrics
	GetHistoryManager() *WebhookHistoryManager
}

// Queue 이벤트 큐 인터페이스
type Queue interface {
	Push(event *alerts.AlertEvent) error
	Pop() (*alerts.AlertEvent, error)
	Size() int
	Clear() error
	Close() error
}

// RetryManager 재시도 관리자 인터페이스
type RetryManager interface {
	// 재시도 처리
	ScheduleRetry(ctx context.Context, event *alerts.AlertEvent, endpoint string, attempt int, lastError error) error
	ProcessRetries(ctx context.Context) error
	
	// 재시도 정책
	CalculateBackoffDelay(attempt int, policy RetryPolicy) time.Duration
	IsRetryableError(err error) bool
}

// EventFilter 이벤트 필터링 인터페이스
type EventFilter interface {
	ShouldSendEvent(event *alerts.AlertEvent) bool
	MatchesLevelFilter(event *alerts.AlertEvent) bool
	MatchesEndpointFilter(event *alerts.AlertEvent, endpoint interface{}) bool
}

// MetricsCollector 메트릭 수집 인터페이스
type MetricsCollector interface {
	IncrementSent()
	IncrementFailed()
	IncrementRetries()
	UpdateQueueSize(size int64)
	UpdateWorkerCount(count int64)
	GetStats() map[string]interface{}
}

// WorkerManager 워커 관리 인터페이스
type WorkerManager interface {
	StartWorkers(ctx context.Context, count int) error
	StopWorkers(ctx context.Context) error
	GetLeastBusyWorker() *Worker
	GetWorkerStats() map[string]interface{}
}