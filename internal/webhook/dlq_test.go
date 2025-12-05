package webhook

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"proxynd/internal/alerts"
)

func TestDeadLetterQueue_AddAndList(t *testing.T) {
	// Create temporary directory for DLQ
	tempDir := filepath.Join(os.TempDir(), "test_dlq")
	defer func() { _ = os.RemoveAll(tempDir) }()

	dlq, err := NewDeadLetterQueue(tempDir, 100)
	if err != nil {
		t.Fatalf("Failed to create DLQ: %v", err)
	}

	// Create test event
	event := &alerts.AlertEvent{
		ID:        "test-event-1",
		Type:      "webhook.failure",
		Level:     "ERROR",
		Message:   "Test webhook failure",
		Source:    "test",
		Timestamp: time.Now(),
	}

	// Add to DLQ
	item := DeadLetterItem{
		Event:       event,
		Endpoint:    "test-endpoint",
		Attempts:    3,
		LastAttempt: time.Now(),
		FirstFailed: time.Now(),
		Error:       "connection timeout",
		Metadata: map[string]string{
			"reason": "max_retries_exceeded",
		},
	}

	err = dlq.Add(item)
	if err != nil {
		t.Fatalf("Failed to add item to DLQ: %v", err)
	}

	// List items
	items, err := dlq.List()
	if err != nil {
		t.Fatalf("Failed to list DLQ items: %v", err)
	}

	if len(items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(items))
	}

	if items[0].Event.ID != event.ID {
		t.Errorf("Expected event ID %s, got %s", event.ID, items[0].Event.ID)
	}

	if items[0].Endpoint != "test-endpoint" {
		t.Errorf("Expected endpoint 'test-endpoint', got %s", items[0].Endpoint)
	}
}

func TestDeadLetterQueue_GetAndRemove(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "test_dlq_get_remove")
	defer func() { _ = os.RemoveAll(tempDir) }()

	dlq, err := NewDeadLetterQueue(tempDir, 100)
	if err != nil {
		t.Fatalf("Failed to create DLQ: %v", err)
	}

	// Add item
	event := &alerts.AlertEvent{
		ID:      "test-event-2",
		Type:    "test",
		Level:   "ERROR",
		Message: "Test",
	}

	item := DeadLetterItem{
		Event:    event,
		Endpoint: "test-endpoint",
		Attempts: 5,
		Error:    "test error",
	}

	err = dlq.Add(item)
	if err != nil {
		t.Fatalf("Failed to add item: %v", err)
	}

	// Get the item
	items, _ := dlq.List()
	if len(items) == 0 {
		t.Fatal("No items in DLQ")
	}

	itemID := items[0].ID
	retrieved, err := dlq.Get(itemID)
	if err != nil {
		t.Fatalf("Failed to get item: %v", err)
	}

	if retrieved.Event.ID != event.ID {
		t.Errorf("Expected event ID %s, got %s", event.ID, retrieved.Event.ID)
	}

	// Remove the item
	err = dlq.Remove(itemID)
	if err != nil {
		t.Fatalf("Failed to remove item: %v", err)
	}

	// Verify removal
	items, _ = dlq.List()
	if len(items) != 0 {
		t.Errorf("Expected 0 items after removal, got %d", len(items))
	}
}

func TestDeadLetterQueue_Retry(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "test_dlq_retry")
	defer func() { _ = os.RemoveAll(tempDir) }()

	dlq, err := NewDeadLetterQueue(tempDir, 100)
	if err != nil {
		t.Fatalf("Failed to create DLQ: %v", err)
	}

	// Add item
	event := &alerts.AlertEvent{
		ID:      "test-event-3",
		Type:    "test",
		Level:   "ERROR",
		Message: "Test retry",
	}

	item := DeadLetterItem{
		Event:    event,
		Endpoint: "test-endpoint",
		Attempts: 3,
		Error:    "test error",
	}

	err = dlq.Add(item)
	if err != nil {
		t.Fatalf("Failed to add item: %v", err)
	}

	// Get item ID
	items, _ := dlq.List()
	itemID := items[0].ID

	// Retry the item
	retried, err := dlq.Retry(itemID)
	if err != nil {
		t.Fatalf("Failed to retry item: %v", err)
	}

	if retried.Event.ID != event.ID {
		t.Errorf("Expected event ID %s, got %s", event.ID, retried.Event.ID)
	}

	// Verify item was removed from DLQ
	items, _ = dlq.List()
	if len(items) != 0 {
		t.Errorf("Expected 0 items after retry, got %d", len(items))
	}
}

func TestDeadLetterQueue_MaxSize(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "test_dlq_maxsize")
	defer func() { _ = os.RemoveAll(tempDir) }()

	maxSize := 5
	dlq, err := NewDeadLetterQueue(tempDir, maxSize)
	if err != nil {
		t.Fatalf("Failed to create DLQ: %v", err)
	}

	// Add more items than max size
	for i := 0; i < maxSize+2; i++ {
		event := &alerts.AlertEvent{
			ID:      "event-" + string(rune('0'+i)),
			Type:    "test",
			Level:   "ERROR",
			Message: "Test",
		}

		item := DeadLetterItem{
			Event:    event,
			Endpoint: "test",
			Attempts: 1,
			Error:    "test",
		}

		// Sleep briefly to ensure different timestamps
		time.Sleep(10 * time.Millisecond)

		err = dlq.Add(item)
		if err != nil {
			t.Fatalf("Failed to add item %d: %v", i, err)
		}
	}

	// Verify max size is enforced
	items, err := dlq.List()
	if err != nil {
		t.Fatalf("Failed to list items: %v", err)
	}

	if len(items) > maxSize {
		t.Errorf("Expected at most %d items, got %d", maxSize, len(items))
	}
}

func TestDeadLetterQueue_Statistics(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "test_dlq_stats")
	defer func() { _ = os.RemoveAll(tempDir) }()

	dlq, err := NewDeadLetterQueue(tempDir, 100)
	if err != nil {
		t.Fatalf("Failed to create DLQ: %v", err)
	}

	// Add multiple items for different endpoints
	endpoints := []string{"endpoint-1", "endpoint-2", "endpoint-1"}
	for i, endpoint := range endpoints {
		event := &alerts.AlertEvent{
			ID:      "event-" + string(rune('0'+i)),
			Type:    "test",
			Level:   "ERROR",
			Message: "Test",
		}

		item := DeadLetterItem{
			Event:    event,
			Endpoint: endpoint,
			Attempts: 1,
			Error:    "test",
		}

		err = dlq.Add(item)
		if err != nil {
			t.Fatalf("Failed to add item: %v", err)
		}
	}

	// Get statistics
	stats, err := dlq.GetStatistics()
	if err != nil {
		t.Fatalf("Failed to get statistics: %v", err)
	}

	totalItems, ok := stats["total_items"].(int)
	if !ok || totalItems != 3 {
		t.Errorf("Expected total_items=3, got %v", stats["total_items"])
	}

	byEndpoint, ok := stats["by_endpoint"].(map[string]int)
	if !ok {
		t.Fatal("Expected by_endpoint map in statistics")
	}

	if byEndpoint["endpoint-1"] != 2 {
		t.Errorf("Expected 2 items for endpoint-1, got %d", byEndpoint["endpoint-1"])
	}

	if byEndpoint["endpoint-2"] != 1 {
		t.Errorf("Expected 1 item for endpoint-2, got %d", byEndpoint["endpoint-2"])
	}
}

func TestDeadLetterQueue_Purge(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "test_dlq_purge")
	defer func() { _ = os.RemoveAll(tempDir) }()

	dlq, err := NewDeadLetterQueue(tempDir, 100)
	if err != nil {
		t.Fatalf("Failed to create DLQ: %v", err)
	}

	// Add old item
	oldTime := time.Now().Add(-48 * time.Hour)
	oldItem := DeadLetterItem{
		Event: &alerts.AlertEvent{
			ID:      "old-event",
			Type:    "test",
			Level:   "ERROR",
			Message: "Old event",
		},
		Endpoint:    "test",
		Attempts:    1,
		FirstFailed: oldTime,
		Error:       "test",
	}

	err = dlq.Add(oldItem)
	if err != nil {
		t.Fatalf("Failed to add old item: %v", err)
	}

	// Add recent item
	recentItem := DeadLetterItem{
		Event: &alerts.AlertEvent{
			ID:      "recent-event",
			Type:    "test",
			Level:   "ERROR",
			Message: "Recent event",
		},
		Endpoint:    "test",
		Attempts:    1,
		FirstFailed: time.Now(),
		Error:       "test",
	}

	err = dlq.Add(recentItem)
	if err != nil {
		t.Fatalf("Failed to add recent item: %v", err)
	}

	// Purge items older than 24 hours
	purgeTime := time.Now().Add(-24 * time.Hour)
	removed, err := dlq.Purge(purgeTime)
	if err != nil {
		t.Fatalf("Failed to purge: %v", err)
	}

	if removed != 1 {
		t.Errorf("Expected 1 item removed, got %d", removed)
	}

	// Verify only recent item remains
	items, _ := dlq.List()
	if len(items) != 1 {
		t.Errorf("Expected 1 item after purge, got %d", len(items))
	}

	if len(items) > 0 && items[0].Event.ID != "recent-event" {
		t.Errorf("Expected recent-event to remain, got %s", items[0].Event.ID)
	}
}
