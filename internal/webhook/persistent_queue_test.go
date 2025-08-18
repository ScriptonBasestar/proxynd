package webhook

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"proxynd/internal/alerts"
)

func TestPersistentFailureQueue_AddAndRetrieve(t *testing.T) {
	// 임시 저장소 디렉토리 생성
	tempDir := filepath.Join(os.TempDir(), "webhook-test", "failures")
	defer func() { _ = os.RemoveAll(tempDir) }()

	// 큐 생성
	queue, err := NewPersistentFailureQueue(tempDir, 3, time.Hour*24)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer func() { _ = queue.Close() }()

	// 테스트 이벤트 생성
	event := &alerts.AlertEvent{
		ID:        "test-event-1",
		Type:      "cache.miss",
		Level:     "WARNING",
		Message:   "Cache miss detected",
		Timestamp: time.Now(),
	}

	// 실패 이벤트 추가
	err = queue.AddFailedEvent(event, "test-endpoint",
		errors.New("connection timeout"))
	if err != nil {
		t.Fatalf("Failed to add event: %v", err)
	}

	// 재시도 가능한 항목 확인
	retryableItems := queue.GetRetryableItems()
	if len(retryableItems) != 0 {
		t.Errorf("Expected 0 retryable items, got %d", len(retryableItems))
	}

	// 통계 확인
	stats := queue.GetStats()
	if stats["total_failed_items"] != 1 {
		t.Errorf("Expected 1 total failed items, got %v", stats["total_failed_items"])
	}
	if stats["pending_retries"] != 1 {
		t.Errorf("Expected 1 pending retries, got %v", stats["pending_retries"])
	}
}

func TestPersistentFailureQueue_RetryLogic(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "webhook-test", "retries")
	defer func() { _ = os.RemoveAll(tempDir) }()

	queue, err := NewPersistentFailureQueue(tempDir, 2, time.Hour*24)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer func() { _ = queue.Close() }()

	event := &alerts.AlertEvent{
		ID:        "test-event-2",
		Type:      "auth.failure",
		Level:     "ERROR",
		Message:   "Authentication failed",
		Timestamp: time.Now(),
	}

	// 실패 이벤트 추가
	err = queue.AddFailedEvent(event, "auth-endpoint",
		errors.New("401 unauthorized"))
	if err != nil {
		t.Fatalf("Failed to add event: %v", err)
	}

	// 첫 번째 재시도 시도 시뮬레이션 (실패)
	items := queue.GetFailedItems()
	var itemID string
	for id := range items {
		itemID = id
		break
	}

	err = queue.UpdateRetryAttempt(itemID, false,
		errors.New("502 bad gateway"))
	if err != nil {
		t.Fatalf("Failed to update retry attempt: %v", err)
	}

	// 두 번째 재시도 시도 시뮬레이션 (성공)
	err = queue.UpdateRetryAttempt(itemID, true, nil)
	if err != nil {
		t.Fatalf("Failed to update retry attempt: %v", err)
	}

	// 항목이 제거되었는지 확인
	items = queue.GetFailedItems()
	if len(items) != 0 {
		t.Errorf("Expected 0 items after successful retry, got %d", len(items))
	}
}

func TestPersistentFailureQueue_Persistence(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "webhook-test", "persistence")
	defer func() { _ = os.RemoveAll(tempDir) }()

	// 첫 번째 큐 인스턴스
	queue1, err := NewPersistentFailureQueue(tempDir, 3, time.Hour*24)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}

	event := &alerts.AlertEvent{
		ID:        "persistent-event",
		Type:      "system.error",
		Level:     "CRITICAL",
		Message:   "System error occurred",
		Timestamp: time.Now(),
	}

	err = queue1.AddFailedEvent(event, "system-endpoint",
		errors.New("system failure"))
	if err != nil {
		t.Fatalf("Failed to add event: %v", err)
	}

	_ = queue1.Close()

	// 두 번째 큐 인스턴스 (재시작 시뮬레이션)
	queue2, err := NewPersistentFailureQueue(tempDir, 3, time.Hour*24)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer func() { _ = queue2.Close() }()

	// 기존 데이터가 로드되었는지 확인
	items := queue2.GetFailedItems()
	if len(items) != 1 {
		t.Fatalf("Expected 1 item after reload, got %d", len(items))
	}

	// 이벤트 데이터 검증
	for _, item := range items {
		if item.Event.ID != "persistent-event" {
			t.Errorf("Expected event ID persistent-event, got %s", item.Event.ID)
		}
		if item.Endpoint != "system-endpoint" {
			t.Errorf("Expected endpoint system-endpoint, got %s", item.Endpoint)
		}
		break
	}
}

func TestPersistentFailureQueue_ExpiredItemsCleanup(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "webhook-test", "cleanup")
	defer func() { _ = os.RemoveAll(tempDir) }()

	// 짧은 TTL로 큐 생성
	queue, err := NewPersistentFailureQueue(tempDir, 3, time.Millisecond*100)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer func() { _ = queue.Close() }()

	event := &alerts.AlertEvent{
		ID:        "expiring-event",
		Type:      "test.event",
		Level:     "INFO",
		Message:   "Test event",
		Timestamp: time.Now(),
	}

	err = queue.AddFailedEvent(event, "test-endpoint",
		errors.New("test error"))
	if err != nil {
		t.Fatalf("Failed to add event: %v", err)
	}

	// TTL 만료 대기
	time.Sleep(time.Millisecond * 200)

	// 만료된 항목 정리
	removedCount := queue.RemoveExpiredItems()
	if removedCount != 1 {
		t.Errorf("Expected 1 removed item, got %d", removedCount)
	}

	// 항목이 제거되었는지 확인
	items := queue.GetFailedItems()
	if len(items) != 0 {
		t.Errorf("Expected 0 items after cleanup, got %d", len(items))
	}
}
