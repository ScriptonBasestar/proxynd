package webhook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"proxynd/internal/alerts"
	"proxynd/internal/logging"
)

// DeadLetterQueue Dead Letter Queue for failed webhook events
type DeadLetterQueue struct {
	mu      sync.RWMutex
	dir     string
	maxSize int
	logger  logging.Logger
}

// DeadLetterItem represents a failed webhook event in the DLQ
type DeadLetterItem struct {
	ID          string             `json:"id"`
	Event       *alerts.AlertEvent `json:"event"`
	Endpoint    string             `json:"endpoint"`
	Attempts    int                `json:"attempts"`
	LastAttempt time.Time          `json:"last_attempt"`
	FirstFailed time.Time          `json:"first_failed"`
	Error       string             `json:"error"`
	Metadata    map[string]string  `json:"metadata,omitempty"`
}

// NewDeadLetterQueue creates a new Dead Letter Queue
func NewDeadLetterQueue(dir string, maxSize int) (*DeadLetterQueue, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create DLQ directory: %w", err)
	}

	return &DeadLetterQueue{
		dir:     dir,
		maxSize: maxSize,
		logger:  logging.GetLogger(),
	}, nil
}

// Add adds a failed event to the DLQ
func (dlq *DeadLetterQueue) Add(item DeadLetterItem) error {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	// Generate unique ID if not set
	if item.ID == "" {
		item.ID = fmt.Sprintf("dlq-%d-%s", time.Now().UnixNano(), item.Endpoint)
	}

	// Set first failed time if not set
	if item.FirstFailed.IsZero() {
		item.FirstFailed = time.Now()
	}

	// Check max size
	items, err := dlq.listUnlocked()
	if err != nil {
		return fmt.Errorf("failed to check DLQ size: %w", err)
	}

	if len(items) >= dlq.maxSize {
		dlq.logger.Warn("DLQ at max capacity, oldest item will be removed",
			logging.F("max_size", dlq.maxSize),
			logging.F("current_size", len(items)))

		// Remove oldest item
		if len(items) > 0 {
			if err := dlq.removeUnlocked(items[0].ID); err != nil {
				dlq.logger.Error("Failed to remove oldest DLQ item", logging.F("error", err))
			}
		}
	}

	// Write to file
	filePath := filepath.Join(dlq.dir, item.ID+".json")
	data, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal DLQ item: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write DLQ item: %w", err)
	}

	dlq.logger.Info("Event added to DLQ",
		logging.F("id", item.ID),
		logging.F("endpoint", item.Endpoint),
		logging.F("attempts", item.Attempts),
		logging.F("error", item.Error))

	return nil
}

// List returns all items in the DLQ, sorted by first failed time (oldest first)
func (dlq *DeadLetterQueue) List() ([]DeadLetterItem, error) {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()

	return dlq.listUnlocked()
}

// listUnlocked internal list method without locking (caller must hold lock)
func (dlq *DeadLetterQueue) listUnlocked() ([]DeadLetterItem, error) {
	entries, err := os.ReadDir(dlq.dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read DLQ directory: %w", err)
	}

	var items []DeadLetterItem
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(dlq.dir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			dlq.logger.Warn("Failed to read DLQ item file",
				logging.F("file", entry.Name()),
				logging.F("error", err))
			continue
		}

		var item DeadLetterItem
		if err := json.Unmarshal(data, &item); err != nil {
			dlq.logger.Warn("Failed to unmarshal DLQ item",
				logging.F("file", entry.Name()),
				logging.F("error", err))
			continue
		}

		items = append(items, item)
	}

	// Sort by first failed time (oldest first)
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].FirstFailed.After(items[j].FirstFailed) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	return items, nil
}

// Get retrieves a specific item from the DLQ by ID
func (dlq *DeadLetterQueue) Get(id string) (*DeadLetterItem, error) {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()

	filePath := filepath.Join(dlq.dir, id+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("DLQ item not found: %s", id)
		}
		return nil, fmt.Errorf("failed to read DLQ item: %w", err)
	}

	var item DeadLetterItem
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal DLQ item: %w", err)
	}

	return &item, nil
}

// Remove removes an item from the DLQ
func (dlq *DeadLetterQueue) Remove(id string) error {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	return dlq.removeUnlocked(id)
}

// removeUnlocked internal remove method without locking (caller must hold lock)
func (dlq *DeadLetterQueue) removeUnlocked(id string) error {
	filePath := filepath.Join(dlq.dir, id+".json")
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove DLQ item: %w", err)
	}

	dlq.logger.Info("DLQ item removed", logging.F("id", id))
	return nil
}

// Retry removes an item from DLQ and returns it for retry
// The caller is responsible for re-queuing the event
func (dlq *DeadLetterQueue) Retry(id string) (*DeadLetterItem, error) {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	// Get the item first
	filePath := filepath.Join(dlq.dir, id+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("DLQ item not found: %s", id)
		}
		return nil, fmt.Errorf("failed to read DLQ item: %w", err)
	}

	var item DeadLetterItem
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal DLQ item: %w", err)
	}

	// Remove from DLQ
	if err := dlq.removeUnlocked(id); err != nil {
		return nil, fmt.Errorf("failed to remove item from DLQ: %w", err)
	}

	dlq.logger.Info("DLQ item retrieved for retry",
		logging.F("id", id),
		logging.F("endpoint", item.Endpoint))

	return &item, nil
}

// Purge removes items older than the specified time
func (dlq *DeadLetterQueue) Purge(before time.Time) (int, error) {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	items, err := dlq.listUnlocked()
	if err != nil {
		return 0, fmt.Errorf("failed to list DLQ items: %w", err)
	}

	removed := 0
	for _, item := range items {
		if item.FirstFailed.Before(before) {
			if err := dlq.removeUnlocked(item.ID); err != nil {
				dlq.logger.Error("Failed to purge DLQ item",
					logging.F("id", item.ID),
					logging.F("error", err))
				continue
			}
			removed++
		}
	}

	if removed > 0 {
		dlq.logger.Info("DLQ items purged",
			logging.F("count", removed),
			logging.F("before", before))
	}

	return removed, nil
}

// Size returns the current number of items in the DLQ
func (dlq *DeadLetterQueue) Size() (int, error) {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()

	items, err := dlq.listUnlocked()
	if err != nil {
		return 0, err
	}

	return len(items), nil
}

// Clear removes all items from the DLQ
func (dlq *DeadLetterQueue) Clear() error {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	items, err := dlq.listUnlocked()
	if err != nil {
		return fmt.Errorf("failed to list DLQ items: %w", err)
	}

	for _, item := range items {
		if err := dlq.removeUnlocked(item.ID); err != nil {
			dlq.logger.Error("Failed to clear DLQ item",
				logging.F("id", item.ID),
				logging.F("error", err))
		}
	}

	dlq.logger.Info("DLQ cleared", logging.F("count", len(items)))
	return nil
}

// GetStatistics returns statistics about the DLQ
func (dlq *DeadLetterQueue) GetStatistics() (map[string]interface{}, error) {
	items, err := dlq.List()
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_items": len(items),
		"max_size":    dlq.maxSize,
		"usage_pct":   float64(len(items)) / float64(dlq.maxSize) * 100,
	}

	if len(items) > 0 {
		stats["oldest_item"] = items[0].FirstFailed
		stats["newest_item"] = items[len(items)-1].FirstFailed
	}

	// Count by endpoint
	endpointCounts := make(map[string]int)
	for _, item := range items {
		endpointCounts[item.Endpoint]++
	}
	stats["by_endpoint"] = endpointCounts

	return stats, nil
}
