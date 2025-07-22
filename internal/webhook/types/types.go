package types

import (
	"context"

	"proxynd/alerts"
)

// EventQueue 이벤트 큐 인터페이스
type EventQueue interface {
	Push(event *alerts.AlertEvent) error
	Pop() (*alerts.AlertEvent, error)
	Size() int
	Clear() error
	Close() error
}

// RateLimiter 속도 제한 인터페이스
type RateLimiter interface {
	Allow() bool
	Wait(ctx context.Context) error
	Tokens() int
	SetLimit(limit int)
}

// WebhookAdapter 웹훅 어댑터 인터페이스
type WebhookAdapter interface {
	Type() string
	Send(ctx context.Context, endpoint string, event *alerts.AlertEvent) error
	Validate(endpoint string) error
}

// SenderMetrics 전송 메트릭
type SenderMetrics struct {
	TotalSent    int64
	TotalFailed  int64
	TotalRetries int64
	QueueSize    int64
	WorkerCount  int64
	BatchStats   map[string]interface{} // 배치 통계
}

// IncrementSent 전송 성공 카운트 증가
func (sm *SenderMetrics) IncrementSent() {
	sm.TotalSent++
}

// IncrementFailed 전송 실패 카운트 증가
func (sm *SenderMetrics) IncrementFailed() {
	sm.TotalFailed++
}

// IncrementRetries 재시도 카운트 증가
func (sm *SenderMetrics) IncrementRetries() {
	sm.TotalRetries++
}

// UpdateQueueSize 큐 크기 업데이트
func (sm *SenderMetrics) UpdateQueueSize(size int64) {
	sm.QueueSize = size
}

// UpdateWorkerCount 워커 수 업데이트
func (sm *SenderMetrics) UpdateWorkerCount(count int64) {
	sm.WorkerCount = count
}