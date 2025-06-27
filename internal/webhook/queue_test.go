package webhook

import (
	"testing"

	"github.com/go-playground/assert/v2"
	"proxynd/alerts"
)

// TestMemoryEventQueue 메모리 이벤트 큐 테스트
func TestMemoryEventQueue(t *testing.T) {
	queue, err := NewMemoryEventQueue(3)
	assert.Equal(t, err, nil)
	assert.NotEqual(t, queue, nil)
	assert.Equal(t, queue.Size(), 0)

	// 이벤트 추가
	event1 := alerts.CreateWebhookEvent(
		alerts.EventCacheExpiry,
		alerts.AlertLevelInfo,
		"테스트 1",
		"테스트 메시지 1",
	)

	err = queue.Push(event1)
	assert.Equal(t, err, nil)
	assert.Equal(t, queue.Size(), 1)

	// 이벤트 제거
	retrievedEvent, err := queue.Pop()
	assert.Equal(t, err, nil)
	assert.Equal(t, retrievedEvent.ID, event1.ID)
	assert.Equal(t, queue.Size(), 0)

	// 빈 큐에서 Pop 시도
	_, err = queue.Pop()
	assert.NotEqual(t, err, nil)
}

// TestMemoryEventQueueOverflow 큐 오버플로우 테스트
func TestMemoryEventQueueOverflow(t *testing.T) {
	queue, err := NewMemoryEventQueue(2) // 최대 2개 이벤트
	assert.Equal(t, err, nil)

	// 3개 이벤트 추가 (오버플로우 발생)
	for i := 1; i <= 3; i++ {
		event := alerts.CreateWebhookEvent(
			alerts.EventCacheExpiry,
			alerts.AlertLevelInfo,
			"테스트",
			"테스트 메시지",
		)
		event.ID = string(rune('A' + i - 1)) // A, B, C

		err = queue.Push(event)
		assert.Equal(t, err, nil)
	}

	// 큐 크기는 최대 크기로 제한됨
	assert.Equal(t, queue.Size(), 2)

	// 첫 번째 이벤트(A)는 제거되고 B, C만 남아있어야 함
	event, err := queue.Pop()
	assert.Equal(t, err, nil)
	assert.Equal(t, event.ID, "B")

	event, err = queue.Pop()
	assert.Equal(t, err, nil)
	assert.Equal(t, event.ID, "C")

	assert.Equal(t, queue.Size(), 0)
}

// TestMemoryEventQueueClear 큐 정리 테스트
func TestMemoryEventQueueClear(t *testing.T) {
	queue, err := NewMemoryEventQueue(10)
	assert.Equal(t, err, nil)

	// 여러 이벤트 추가
	for i := 0; i < 5; i++ {
		event := alerts.CreateWebhookEvent(
			alerts.EventCacheExpiry,
			alerts.AlertLevelInfo,
			"테스트",
			"테스트 메시지",
		)
		queue.Push(event)
	}

	assert.Equal(t, queue.Size(), 5)

	// 큐 정리
	err = queue.Clear()
	assert.Equal(t, err, nil)
	assert.Equal(t, queue.Size(), 0)
}

// TestMemoryEventQueueClose 큐 닫기 테스트
func TestMemoryEventQueueClose(t *testing.T) {
	queue, err := NewMemoryEventQueue(10)
	assert.Equal(t, err, nil)

	// 이벤트 추가
	event := alerts.CreateWebhookEvent(
		alerts.EventCacheExpiry,
		alerts.AlertLevelInfo,
		"테스트",
		"테스트 메시지",
	)
	queue.Push(event)

	assert.Equal(t, queue.Size(), 1)

	// 큐 닫기
	err = queue.Close()
	assert.Equal(t, err, nil)

	// 닫힌 후에는 큐 작업 불가
	assert.Equal(t, queue.events, ([]*alerts.AlertEvent)(nil))
}

// TestPersistentEventQueue 영속성 큐 테스트 (기본적인 테스트)
func TestPersistentEventQueue(t *testing.T) {
	queue, err := NewPersistentEventQueue("/tmp/test_queue", 10)
	assert.Equal(t, err, nil)
	assert.NotEqual(t, queue, nil)

	// 기본 동작 테스트 (메모리 큐와 동일)
	event := alerts.CreateWebhookEvent(
		alerts.EventCacheExpiry,
		alerts.AlertLevelInfo,
		"테스트",
		"테스트 메시지",
	)

	err = queue.Push(event)
	assert.Equal(t, err, nil)
	assert.Equal(t, queue.Size(), 1)

	retrievedEvent, err := queue.Pop()
	assert.Equal(t, err, nil)
	assert.Equal(t, retrievedEvent.ID, event.ID)
	assert.Equal(t, queue.Size(), 0)
}
