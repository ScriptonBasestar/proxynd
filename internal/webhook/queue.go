package webhook

import (
	"fmt"
	"sync"

	"proxynd/internal/alerts"
)

// MemoryEventQueue 메모리 기반 이벤트 큐
type MemoryEventQueue struct {
	events   []*alerts.AlertEvent
	mu       sync.RWMutex
	maxSize  int
	notifyCh chan struct{}
}

// NewMemoryEventQueue 새로운 메모리 이벤트 큐 생성
func NewMemoryEventQueue(maxSize int) (*MemoryEventQueue, error) {
	if maxSize <= 0 {
		maxSize = 1000 // 기본값
	}

	return &MemoryEventQueue{
		events:   make([]*alerts.AlertEvent, 0, maxSize),
		maxSize:  maxSize,
		notifyCh: make(chan struct{}, 1),
	}, nil
}

// Push 이벤트를 큐에 추가
func (q *MemoryEventQueue) Push(event *alerts.AlertEvent) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// 큐가 가득 찬 경우
	if len(q.events) >= q.maxSize {
		// 오래된 이벤트 제거 (FIFO)
		q.events = q.events[1:]
	}

	q.events = append(q.events, event)

	// 알림 전송 (non-blocking)
	select {
	case q.notifyCh <- struct{}{}:
	default:
	}

	return nil
}

// Pop 큐에서 이벤트 제거 및 반환
func (q *MemoryEventQueue) Pop() (*alerts.AlertEvent, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.events) == 0 {
		return nil, fmt.Errorf("queue is empty")
	}

	event := q.events[0]
	q.events = q.events[1:]

	return event, nil
}

// Size 큐 크기 반환
func (q *MemoryEventQueue) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.events)
}

// Clear 큐 정리
func (q *MemoryEventQueue) Clear() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.events = q.events[:0]
	return nil
}

// Close 큐 닫기
func (q *MemoryEventQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.events = nil
	close(q.notifyCh)
	return nil
}

// NotifyCh 알림 채널 반환
func (q *MemoryEventQueue) NotifyCh() <-chan struct{} {
	return q.notifyCh
}

// PersistentEventQueue 영속성 이벤트 큐 (향후 확장용)
type PersistentEventQueue struct {
	filePath string
	memory   *MemoryEventQueue
}

// NewPersistentEventQueue 새로운 영속성 이벤트 큐 생성
func NewPersistentEventQueue(filePath string, maxSize int) (*PersistentEventQueue, error) {
	memQueue, err := NewMemoryEventQueue(maxSize)
	if err != nil {
		return nil, err
	}

	queue := &PersistentEventQueue{
		filePath: filePath,
		memory:   memQueue,
	}

	// 파일에서 이벤트 로드 (구현 생략)
	// queue.loadFromFile()

	return queue, nil
}

// Push 이벤트를 큐에 추가 (영속성)
func (q *PersistentEventQueue) Push(event *alerts.AlertEvent) error {
	// 메모리 큐에 추가
	if err := q.memory.Push(event); err != nil {
		return err
	}

	// 파일에 저장 (구현 생략)
	// return q.saveToFile(event)
	return nil
}

// Pop 큐에서 이벤트 제거 및 반환 (영속성)
func (q *PersistentEventQueue) Pop() (*alerts.AlertEvent, error) {
	event, err := q.memory.Pop()
	if err != nil {
		return nil, err
	}

	// 파일에서 제거 (구현 생략)
	// q.removeFromFile(event.ID)

	return event, nil
}

// Size 큐 크기 반환
func (q *PersistentEventQueue) Size() int {
	return q.memory.Size()
}

// Clear 큐 정리
func (q *PersistentEventQueue) Clear() error {
	// 파일 정리 (구현 생략)
	// q.clearFile()
	return q.memory.Clear()
}

// Close 큐 닫기
func (q *PersistentEventQueue) Close() error {
	return q.memory.Close()
}
