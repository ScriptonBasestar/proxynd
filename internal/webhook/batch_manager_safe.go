package webhook

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"proxynd/alerts"
	"proxynd/configs"
	"proxynd/logging"
)

// BatchManagerSafe is a thread-safe batch manager implementation
type BatchManagerSafe struct {
	config  configs.WebhookBatchingConfig
	logger  logging.Logger
	
	// Thread-safe group management
	groups     map[string]*BatchGroupSafe
	groupsMu   sync.RWMutex
	
	// Lifecycle management
	ctx        context.Context
	cancel     context.CancelFunc
	
	// Metrics
	totalBatches int64
	totalEvents  int64
	failedBatches int64
}

// BatchGroupSafe is a thread-safe batch group
type BatchGroupSafe struct {
	endpoint   configs.WebhookEndpointConfig
	events     []*alerts.AlertEvent
	mu         sync.Mutex
	lastFlush  time.Time
	flushTimer *time.Timer
	manager    *BatchManagerSafe
}

// NewBatchManagerSafe creates a new thread-safe batch manager
func NewBatchManagerSafe(config configs.WebhookBatchingConfig, logger logging.Logger) *BatchManagerSafe {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &BatchManagerSafe{
		config:   config,
		logger:   logger,
		groups:   make(map[string]*BatchGroupSafe),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start starts the batch manager
func (bm *BatchManagerSafe) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	
	if !bm.config.Enabled {
		bm.logger.Info("배치 매니저 비활성화됨")
		return
	}
	
	bm.logger.Info("배치 매니저 시작됨",
		logging.F("max_size", bm.config.MaxBatchSize),
		logging.F("max_wait", bm.config.MaxWaitTime))
	
	// Periodic cleanup of expired groups
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			bm.cleanupExpiredGroups()
			
		case <-ctx.Done():
			bm.logger.Info("배치 매니저 종료 중")
			bm.flushAllGroups()
			return
		}
	}
}

// Stop stops the batch manager
func (bm *BatchManagerSafe) Stop() error {
	bm.cancel()
	return nil
}

// AddEvent adds an event to the appropriate batch group
func (bm *BatchManagerSafe) AddEvent(event *alerts.AlertEvent, endpoint configs.WebhookEndpointConfig) {
	if !bm.config.Enabled {
		// If batching is disabled, send immediately
		// This would be handled by the sender directly
		return
	}
	
	groupKey := bm.getGroupKey(event, endpoint)
	group := bm.getOrCreateGroup(groupKey, endpoint)
	
	group.AddEvent(event)
	atomic.AddInt64(&bm.totalEvents, 1)
}

// getGroupKey generates a unique key for batch grouping
func (bm *BatchManagerSafe) getGroupKey(event *alerts.AlertEvent, endpoint configs.WebhookEndpointConfig) string {
	// Group by endpoint and event type
	return endpoint.Name + ":" + event.Type
}

// getOrCreateGroup gets or creates a batch group
func (bm *BatchManagerSafe) getOrCreateGroup(key string, endpoint configs.WebhookEndpointConfig) *BatchGroupSafe {
	// Try read lock first
	bm.groupsMu.RLock()
	group, exists := bm.groups[key]
	bm.groupsMu.RUnlock()
	
	if exists {
		return group
	}
	
	// Need write lock to create new group
	bm.groupsMu.Lock()
	defer bm.groupsMu.Unlock()
	
	// Double-check after acquiring write lock
	if group, exists := bm.groups[key]; exists {
		return group
	}
	
	// Create new group
	group = &BatchGroupSafe{
		endpoint:  endpoint,
		events:    make([]*alerts.AlertEvent, 0, bm.config.MaxBatchSize),
		lastFlush: time.Now(),
		manager:   bm,
	}
	
	bm.groups[key] = group
	
	// Start flush timer
	group.startFlushTimer()
	
	return group
}

// AddEvent adds an event to the batch group
func (bg *BatchGroupSafe) AddEvent(event *alerts.AlertEvent) {
	bg.mu.Lock()
	defer bg.mu.Unlock()
	
	bg.events = append(bg.events, event)
	
	// Check if batch is full
	if len(bg.events) >= bg.manager.config.MaxBatchSize {
		bg.flushLocked()
	}
}

// startFlushTimer starts the flush timer for the group
func (bg *BatchGroupSafe) startFlushTimer() {
	bg.mu.Lock()
	defer bg.mu.Unlock()
	
	if bg.flushTimer != nil {
		bg.flushTimer.Stop()
	}
	
	bg.flushTimer = time.AfterFunc(bg.manager.config.MaxWaitTime, func() {
		bg.Flush()
	})
}

// Flush flushes the batch group
func (bg *BatchGroupSafe) Flush() {
	bg.mu.Lock()
	defer bg.mu.Unlock()
	
	bg.flushLocked()
}

// flushLocked flushes the group (must be called with lock held)
func (bg *BatchGroupSafe) flushLocked() {
	if len(bg.events) == 0 {
		return
	}
	
	// Stop timer
	if bg.flushTimer != nil {
		bg.flushTimer.Stop()
		bg.flushTimer = nil
	}
	
	// Copy events for sending
	eventsCopy := make([]*alerts.AlertEvent, len(bg.events))
	copy(eventsCopy, bg.events)
	
	// Clear events
	bg.events = bg.events[:0]
	bg.lastFlush = time.Now()
	
	// Send batch asynchronously
	go bg.sendBatch(eventsCopy)
}

// sendBatch sends a batch of events
func (bg *BatchGroupSafe) sendBatch(events []*alerts.AlertEvent) {
	atomic.AddInt64(&bg.manager.totalBatches, 1)
	
	bg.manager.logger.Debug("배치 전송 중",
		logging.F("endpoint", bg.endpoint.Name),
		logging.F("size", len(events)))
	
	// Create batch payload
	payload := map[string]interface{}{
		"events":     events,
		"batch_size": len(events),
		"timestamp":  time.Now().Unix(),
	}
	
	// Send via appropriate adapter
	// This would be implemented based on endpoint type
	if err := bg.sendViaAdapter(payload); err != nil {
		atomic.AddInt64(&bg.manager.failedBatches, 1)
		bg.manager.logger.Error("배치 전송 실패",
			logging.F("endpoint", bg.endpoint.Name),
			logging.F("size", len(events)),
			logging.F("error", err))
	} else {
		bg.manager.logger.Info("배치 전송 성공",
			logging.F("endpoint", bg.endpoint.Name),
			logging.F("size", len(events)))
	}
	
	// Restart timer for next batch
	bg.startFlushTimer()
}

// sendViaAdapter sends the batch via the appropriate adapter
func (bg *BatchGroupSafe) sendViaAdapter(payload interface{}) error {
	// This would be implemented to use the actual webhook adapters
	// For now, it's a placeholder
	return nil
}

// flushAllGroups flushes all batch groups
func (bm *BatchManagerSafe) flushAllGroups() {
	bm.groupsMu.Lock()
	groups := make([]*BatchGroupSafe, 0, len(bm.groups))
	for _, group := range bm.groups {
		groups = append(groups, group)
	}
	bm.groupsMu.Unlock()
	
	// Flush all groups
	var wg sync.WaitGroup
	for _, group := range groups {
		wg.Add(1)
		go func(g *BatchGroupSafe) {
			defer wg.Done()
			g.Flush()
		}(group)
	}
	
	// Wait with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		bm.logger.Info("모든 배치 그룹 플러시 완료")
	case <-time.After(10 * time.Second):
		bm.logger.Warn("배치 그룹 플러시 시간 초과")
	}
}

// cleanupExpiredGroups removes groups that haven't been used recently
func (bm *BatchManagerSafe) cleanupExpiredGroups() {
	bm.groupsMu.Lock()
	defer bm.groupsMu.Unlock()
	
	now := time.Now()
	expiredKeys := make([]string, 0)
	
	for key, group := range bm.groups {
		group.mu.Lock()
		isEmpty := len(group.events) == 0
		lastFlush := group.lastFlush
		group.mu.Unlock()
		
		// Remove groups that are empty and haven't been used for 5 minutes
		if isEmpty && now.Sub(lastFlush) > 5*time.Minute {
			expiredKeys = append(expiredKeys, key)
		}
	}
	
	// Remove expired groups
	for _, key := range expiredKeys {
		if group, exists := bm.groups[key]; exists {
			group.mu.Lock()
			if group.flushTimer != nil {
				group.flushTimer.Stop()
			}
			group.mu.Unlock()
			delete(bm.groups, key)
		}
	}
	
	if len(expiredKeys) > 0 {
		bm.logger.Debug("만료된 배치 그룹 정리됨",
			logging.F("count", len(expiredKeys)))
	}
}

// GetStats returns batch manager statistics
func (bm *BatchManagerSafe) GetStats() map[string]interface{} {
	bm.groupsMu.RLock()
	activeGroups := len(bm.groups)
	
	groupStats := make(map[string]interface{})
	for key, group := range bm.groups {
		group.mu.Lock()
		stats := map[string]interface{}{
			"event_count":   len(group.events),
			"last_flush":    group.lastFlush.Unix(),
			"endpoint":      group.endpoint.Name,
		}
		group.mu.Unlock()
		groupStats[key] = stats
	}
	bm.groupsMu.RUnlock()
	
	return map[string]interface{}{
		"enabled":        bm.config.Enabled,
		"active_groups":  activeGroups,
		"total_batches":  atomic.LoadInt64(&bm.totalBatches),
		"total_events":   atomic.LoadInt64(&bm.totalEvents),
		"failed_batches": atomic.LoadInt64(&bm.failedBatches),
		"group_stats":    groupStats,
	}
}