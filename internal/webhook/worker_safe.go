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

// WorkerSafe is a thread-safe implementation of webhook worker
type WorkerSafe struct {
	id       int
	sender   *WebhookSender
	stopCh   chan struct{}
	eventCh  chan *alerts.AlertEvent
	logger   logging.Logger
	retryQ   *RetryQueueSafe
	
	// Atomic counters for thread-safe metrics
	processedCount int64
	errorCount     int64
}

// NewWorkerSafe creates a new thread-safe worker
func NewWorkerSafe(id int, sender *WebhookSender, queueSize int) *WorkerSafe {
	return &WorkerSafe{
		id:      id,
		sender:  sender,
		stopCh:  make(chan struct{}),
		eventCh: make(chan *alerts.AlertEvent, queueSize),
		logger:  sender.logger,
		retryQ:  NewRetryQueueSafe(),
	}
}

// Start starts the worker with proper context handling
func (w *WorkerSafe) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	w.logger.Info("웹훅 워커 시작됨", logging.F("worker_id", w.id))

	// Start retry processor in a separate goroutine
	retryCtx, retryCancel := context.WithCancel(ctx)
	retryWg := &sync.WaitGroup{}
	retryWg.Add(1)
	go w.processRetryLoop(retryCtx, retryWg)

	for {
		select {
		case event := <-w.eventCh:
			w.processEvent(ctx, event)
			atomic.AddInt64(&w.processedCount, 1)
			
		case <-w.stopCh:
			w.logger.Info("웹훅 워커 중지 신호 받음", logging.F("worker_id", w.id))
			retryCancel()
			retryWg.Wait()
			return
			
		case <-ctx.Done():
			w.logger.Info("웹훅 워커 컨텍스트 종료", logging.F("worker_id", w.id))
			retryCancel()
			retryWg.Wait()
			return
		}
	}
}

// GetQueueSize returns the current queue size (thread-safe)
func (w *WorkerSafe) GetQueueSize() int {
	return len(w.eventCh)
}

// GetMetrics returns worker metrics (thread-safe)
func (w *WorkerSafe) GetMetrics() WorkerMetrics {
	return WorkerMetrics{
		ProcessedCount: atomic.LoadInt64(&w.processedCount),
		ErrorCount:     atomic.LoadInt64(&w.errorCount),
		QueueSize:      len(w.eventCh),
		RetryQueueSize: w.retryQ.Size(),
	}
}

// Stop gracefully stops the worker
func (w *WorkerSafe) Stop() {
	close(w.stopCh)
}

// processEvent processes a single event
func (w *WorkerSafe) processEvent(ctx context.Context, event *alerts.AlertEvent) {
	w.logger.Debug("이벤트 처리 시작",
		logging.F("worker_id", w.id),
		logging.F("event_id", event.ID),
		logging.F("event_type", event.Type))

	// Process all enabled endpoints
	for _, endpoint := range w.sender.config.Endpoints {
		if !endpoint.Enabled {
			continue
		}

		// Check endpoint filter
		if !w.sender.matchesEndpointFilter(event, endpoint) {
			w.logger.Debug("엔드포인트 필터로 제외됨",
				logging.F("endpoint", endpoint.Name),
				logging.F("event_type", event.Type))
			continue
		}

		// Attempt to send
		if err := w.sender.sendToEndpoint(ctx, event, endpoint); err != nil {
			atomic.AddInt64(&w.errorCount, 1)
			w.logger.Error("엔드포인트 전송 실패",
				logging.F("worker_id", w.id),
				logging.F("endpoint", endpoint.Name),
				logging.F("event_id", event.ID),
				logging.F("error", err.Error()))

			// Add to retry queue if retryable
			if w.sender.isRetryableError(err) {
				retryItem := &RetryItem{
					Event:     event,
					Endpoint:  endpoint.Name,
					Attempt:   1,
					NextRetry: time.Now().Add(time.Second * 5),
					LastError: err,
					CreatedAt: time.Now(),
				}
				w.retryQ.Add(retryItem)
			}
		} else {
			w.logger.Debug("엔드포인트 전송 성공",
				logging.F("worker_id", w.id),
				logging.F("endpoint", endpoint.Name),
				logging.F("event_id", event.ID))
		}
	}
}

// processRetryLoop continuously processes retry items
func (w *WorkerSafe) processRetryLoop(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			w.processRetries(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// processRetries processes pending retry items
func (w *WorkerSafe) processRetries(ctx context.Context) {
	items := w.retryQ.GetReadyItems()
	
	for _, item := range items {
		// Check max attempts
		if item.Attempt >= w.sender.config.Retry.MaxAttempts {
			w.sender.logger.Warn("재시도 한도 초과로 포기",
				logging.F("event_id", item.Event.ID),
				logging.F("endpoint", item.Endpoint),
				logging.F("attempts", item.Attempt))
			
			w.sender.handlePermanentFailure(item)
			continue
		}
		
		// Find endpoint
		var endpoint *configs.WebhookEndpointConfig
		for _, ep := range w.sender.config.Endpoints {
			if ep.Name == item.Endpoint {
				endpoint = &ep
				break
			}
		}
		
		if endpoint == nil {
			w.sender.logger.Warn("엔드포인트를 찾을 수 없음",
				logging.F("endpoint", item.Endpoint))
			continue
		}
		
		// Retry send
		if err := w.sender.sendToEndpoint(ctx, item.Event, *endpoint); err != nil {
			// Update retry item
			item.Attempt++
			item.LastError = err
			item.NextRetry = time.Now().Add(w.calculateBackoffDelay(item.Attempt))
			
			// Re-add to queue
			w.retryQ.Add(item)
			
			w.sender.logger.Warn("재시도 실패",
				logging.F("event_id", item.Event.ID),
				logging.F("endpoint", item.Endpoint),
				logging.F("attempt", item.Attempt),
				logging.F("next_retry", item.NextRetry),
				logging.F("error", err.Error()))
		} else {
			// Success
			w.sender.logger.Info("재시도 성공",
				logging.F("event_id", item.Event.ID),
				logging.F("endpoint", item.Endpoint),
				logging.F("attempt", item.Attempt))
		}
	}
}

// calculateBackoffDelay calculates exponential backoff delay
func (w *WorkerSafe) calculateBackoffDelay(attempt int) time.Duration {
	delay := time.Second * time.Duration(attempt*attempt)
	maxDelay := time.Minute * 5
	if delay > maxDelay {
		delay = maxDelay
	}
	return delay
}

// WorkerMetrics represents worker performance metrics
type WorkerMetrics struct {
	ProcessedCount int64
	ErrorCount     int64
	QueueSize      int
	RetryQueueSize int
}

// RetryQueueSafe is a thread-safe retry queue implementation
type RetryQueueSafe struct {
	items    []*RetryItem
	mu       sync.Mutex
	notifyCh chan struct{}
}

// NewRetryQueueSafe creates a new thread-safe retry queue
func NewRetryQueueSafe() *RetryQueueSafe {
	return &RetryQueueSafe{
		items:    make([]*RetryItem, 0),
		notifyCh: make(chan struct{}, 1),
	}
}

// Add adds an item to the retry queue (thread-safe)
func (rq *RetryQueueSafe) Add(item *RetryItem) {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	rq.items = append(rq.items, item)

	// Non-blocking notification
	select {
	case rq.notifyCh <- struct{}{}:
	default:
	}
}

// GetReadyItems returns items ready for retry (thread-safe)
func (rq *RetryQueueSafe) GetReadyItems() []*RetryItem {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	now := time.Now()
	readyItems := make([]*RetryItem, 0)
	remainingItems := make([]*RetryItem, 0)

	for _, item := range rq.items {
		if now.After(item.NextRetry) || now.Equal(item.NextRetry) {
			// Deep copy the item to avoid race conditions
			copiedItem := &RetryItem{
				Event:     item.Event,
				Endpoint:  item.Endpoint,
				Attempt:   item.Attempt,
				NextRetry: item.NextRetry,
				LastError: item.LastError,
				CreatedAt: item.CreatedAt,
			}
			readyItems = append(readyItems, copiedItem)
		} else {
			remainingItems = append(remainingItems, item)
		}
	}

	rq.items = remainingItems
	return readyItems
}

// Size returns the current queue size (thread-safe)
func (rq *RetryQueueSafe) Size() int {
	rq.mu.Lock()
	defer rq.mu.Unlock()
	return len(rq.items)
}

// Clear removes all items from the queue (thread-safe)
func (rq *RetryQueueSafe) Clear() {
	rq.mu.Lock()
	defer rq.mu.Unlock()
	rq.items = rq.items[:0]
}

// GetAllItems returns a copy of all items (thread-safe)
func (rq *RetryQueueSafe) GetAllItems() []*RetryItem {
	rq.mu.Lock()
	defer rq.mu.Unlock()
	
	// Deep copy to prevent race conditions
	items := make([]*RetryItem, len(rq.items))
	copy(items, rq.items)
	return items
}