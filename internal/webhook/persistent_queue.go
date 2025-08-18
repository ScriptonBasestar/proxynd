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

// PersistentFailureQueue 실패한 웹훅 이벤트의 영속성 저장소
type PersistentFailureQueue struct {
	storageDir string
	logger     logging.Logger
	mu         sync.RWMutex
	items      map[string]*FailedWebhookItem
	maxRetries int
	retryTTL   time.Duration
}

// FailedWebhookItem 실패한 웹훅 항목
type FailedWebhookItem struct {
	ID         string             `json:"id"`
	Event      *alerts.AlertEvent `json:"event"`
	Endpoint   string             `json:"endpoint"`
	Attempts   int                `json:"attempts"`
	MaxRetries int                `json:"max_retries"`
	NextRetry  time.Time          `json:"next_retry"`
	LastError  string             `json:"last_error"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

// NewPersistentFailureQueue 새로운 영속성 실패 큐 생성
func NewPersistentFailureQueue(storageDir string, maxRetries int,
	retryTTL time.Duration,
) (*PersistentFailureQueue, error) {
	logger := logging.GetLogger()

	// 저장 디렉토리 생성
	if err := os.MkdirAll(storageDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	queue := &PersistentFailureQueue{
		storageDir: storageDir,
		logger:     logger,
		items:      make(map[string]*FailedWebhookItem),
		maxRetries: maxRetries,
		retryTTL:   retryTTL,
	}

	// 기존 실패 항목 로드
	if err := queue.loadFromDisk(); err != nil {
		logger.Error("기존 실패 항목 로드 실패", logging.F(fieldError, err))
	}

	return queue, nil
}

// AddFailedEvent 실패한 이벤트 추가
func (pq *PersistentFailureQueue) AddFailedEvent(event *alerts.AlertEvent, endpoint string, lastError error) error {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	itemID := fmt.Sprintf("%s_%s_%d", event.ID, endpoint, time.Now().Unix())

	item := &FailedWebhookItem{
		ID:         itemID,
		Event:      event,
		Endpoint:   endpoint,
		Attempts:   1,
		MaxRetries: pq.maxRetries,
		NextRetry:  time.Now().Add(time.Minute), // 첫 재시도는 1분 후
		LastError:  lastError.Error(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	pq.items[itemID] = item

	// 디스크에 저장
	if err := pq.saveToDisk(item); err != nil {
		pq.logger.Error("실패 항목 저장 오류",
			logging.F(fieldItemID, itemID),
			logging.F(fieldError, err))
		return err
	}

	pq.logger.Info("실패한 웹훅 이벤트 저장됨",
		logging.F(fieldItemID, itemID),
		logging.F("event_id", event.ID),
		logging.F("endpoint", endpoint),
		logging.F(fieldNextRetry, item.NextRetry))

	return nil
}

// GetRetryableItems 재시도 가능한 항목들 반환
func (pq *PersistentFailureQueue) GetRetryableItems() []*FailedWebhookItem {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	now := time.Now()
	var retryableItems []*FailedWebhookItem

	for _, item := range pq.items {
		// 재시도 시간이 되었고 최대 재시도 횟수를 초과하지 않은 항목
		if now.After(item.NextRetry) && item.Attempts < item.MaxRetries {
			retryableItems = append(retryableItems, item)
		}
	}

	return retryableItems
}

// UpdateRetryAttempt 재시도 시도 업데이트
func (pq *PersistentFailureQueue) UpdateRetryAttempt(itemID string, success bool, lastError error) error {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	item, exists := pq.items[itemID]
	if !exists {
		return fmt.Errorf("item not found: %s", itemID)
	}

	if success {
		// 성공 시 항목 제거
		delete(pq.items, itemID)
		pq.removeFromDisk(itemID)
		pq.logger.Info("웹훅 재시도 성공, 항목 제거됨",
			logging.F(fieldItemID, itemID),
			logging.F(fieldAttempts, item.Attempts))
	} else {
		// 실패 시 재시도 정보 업데이트
		item.Attempts++
		item.UpdatedAt = time.Now()

		if lastError != nil {
			item.LastError = lastError.Error()
		}

		// 지수 백오프로 다음 재시도 시간 계산
		backoffDelay := time.Duration(item.Attempts) * time.Minute * time.Duration(item.Attempts)
		if backoffDelay > time.Hour {
			backoffDelay = time.Hour // 최대 1시간
		}
		item.NextRetry = time.Now().Add(backoffDelay)

		// 최대 재시도 횟수 초과 시 만료 처리
		if item.Attempts >= item.MaxRetries {
			pq.logger.Warn("웹훅 최대 재시도 횟수 초과",
				logging.F(fieldItemID, itemID),
				logging.F(fieldAttempts, item.Attempts),
				logging.F("max_retries", item.MaxRetries))
			// 항목은 유지하되 재시도하지 않음 (수동 검토를 위해)
		} else {
			pq.logger.Info("웹훅 재시도 실패, 다음 시도 예약됨",
				logging.F(fieldItemID, itemID),
				logging.F(fieldAttempts, item.Attempts),
				logging.F(fieldNextRetry, item.NextRetry))
		}

		// 디스크에 업데이트
		if err := pq.saveToDisk(item); err != nil {
			pq.logger.Error("실패 항목 업데이트 저장 오류",
				logging.F(fieldItemID, itemID),
				logging.F(fieldError, err))
			return err
		}
	}

	return nil
}

// GetFailedItems 모든 실패 항목 반환 (관리 목적)
func (pq *PersistentFailureQueue) GetFailedItems() map[string]*FailedWebhookItem {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	// 복사본 반환
	items := make(map[string]*FailedWebhookItem)
	for id, item := range pq.items {
		items[id] = item
	}
	return items
}

// RemoveExpiredItems TTL을 초과한 항목들 제거
func (pq *PersistentFailureQueue) RemoveExpiredItems() int {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	now := time.Now()
	expiredCount := 0

	for id, item := range pq.items {
		if now.Sub(item.CreatedAt) > pq.retryTTL {
			delete(pq.items, id)
			pq.removeFromDisk(id)
			expiredCount++
			pq.logger.Info("만료된 실패 항목 제거됨",
				logging.F(fieldItemID, id),
				logging.F("age", now.Sub(item.CreatedAt)))
		}
	}

	if expiredCount > 0 {
		pq.logger.Info("만료된 실패 항목들 정리 완료",
			logging.F("removed_count", expiredCount))
	}

	return expiredCount
}

// GetStats 통계 정보 반환
func (pq *PersistentFailureQueue) GetStats() map[string]interface{} {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	now := time.Now()
	pendingRetries := 0
	maxRetriesExceeded := 0
	oldestItem := now

	for _, item := range pq.items {
		if item.Attempts < item.MaxRetries {
			pendingRetries++
		} else {
			maxRetriesExceeded++
		}

		if item.CreatedAt.Before(oldestItem) {
			oldestItem = item.CreatedAt
		}
	}

	return map[string]interface{}{
		"total_failed_items":      len(pq.items),
		"pending_retries":         pendingRetries,
		"max_retries_exceeded":    maxRetriesExceeded,
		"oldest_item_age_seconds": int(now.Sub(oldestItem).Seconds()),
	}
}

// loadFromDisk 디스크에서 실패 항목들 로드
func (pq *PersistentFailureQueue) loadFromDisk() error {
	files, err := filepath.Glob(filepath.Join(pq.storageDir, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to scan storage directory: %w", err)
	}

	loadedCount := 0
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			pq.logger.Error("실패 항목 파일 읽기 오류",
				logging.F(fieldFile, file),
				logging.F(fieldError, err))
			continue
		}

		var item FailedWebhookItem
		if err := json.Unmarshal(data, &item); err != nil {
			pq.logger.Error("실패 항목 JSON 파싱 오류",
				logging.F(fieldFile, file),
				logging.F(fieldError, err))
			continue
		}

		pq.items[item.ID] = &item
		loadedCount++
	}

	if loadedCount > 0 {
		pq.logger.Info("기존 실패 항목들 로드 완료",
			logging.F("loaded_count", loadedCount))
	}

	return nil
}

// saveToDisk 실패 항목을 디스크에 저장
func (pq *PersistentFailureQueue) saveToDisk(item *FailedWebhookItem) error {
	data, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	filePath := filepath.Join(pq.storageDir, fmt.Sprintf("%s.json", item.ID))
	return os.WriteFile(filePath, data, 0o644)
}

// removeFromDisk 디스크에서 실패 항목 제거
func (pq *PersistentFailureQueue) removeFromDisk(itemID string) {
	filePath := filepath.Join(pq.storageDir, fmt.Sprintf("%s.json", itemID))
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		pq.logger.Error("실패 항목 파일 삭제 오류",
			logging.F(fieldFile, filePath),
			logging.F(fieldError, err))
	}
}

// Close 큐 정리
func (pq *PersistentFailureQueue) Close() error {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	pq.logger.Info("영속성 실패 큐 종료",
		logging.F("remaining_items", len(pq.items)))

	// 메모리 정리
	pq.items = nil
	return nil
}
