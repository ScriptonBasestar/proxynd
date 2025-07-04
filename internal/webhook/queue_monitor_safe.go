package webhook

import (
	"context"
	"sync"
	"sync/atomic"

	"proxynd/alerts"
	"proxynd/logging"
)

// QueueMonitorSafe is a thread-safe implementation of queue monitoring
type QueueMonitorSafe struct {
	sender      *WebhookSenderSafe
	workers     []*WorkerSafe
	workerIndex int32 // Atomic counter for round-robin
	mu          sync.RWMutex
	logger      logging.Logger
}

// NewQueueMonitorSafe creates a new thread-safe queue monitor
func NewQueueMonitorSafe(sender *WebhookSenderSafe, workers []*WorkerSafe) *QueueMonitorSafe {
	return &QueueMonitorSafe{
		sender:  sender,
		workers: workers,
		logger:  sender.logger,
	}
}

// Start starts monitoring the queue
func (qm *QueueMonitorSafe) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	qm.logger.Info("큐 모니터 시작됨")

	for {
		select {
		case event := <-qm.sender.queue.Receive():
			if event == nil {
				continue
			}
			
			// Route event to worker
			if err := qm.routeEvent(event); err != nil {
				qm.logger.Error("이벤트 라우팅 실패",
					logging.F("event_id", event.ID),
					logging.F("error", err.Error()))
			}
			
		case <-ctx.Done():
			qm.logger.Info("큐 모니터 종료")
			return
		}
	}
}

// routeEvent routes an event to the least busy worker
func (qm *QueueMonitorSafe) routeEvent(event *alerts.AlertEvent) error {
	worker := qm.selectWorker()
	if worker == nil {
		return ErrNoAvailableWorkers
	}

	// Non-blocking send to worker
	select {
	case worker.eventCh <- event:
		qm.logger.Debug("이벤트 라우팅됨",
			logging.F("event_id", event.ID),
			logging.F("worker_id", worker.id))
		return nil
	default:
		// If worker queue is full, try next worker
		return qm.routeEventWithRetry(event, 3)
	}
}

// routeEventWithRetry attempts to route event with retries
func (qm *QueueMonitorSafe) routeEventWithRetry(event *alerts.AlertEvent, maxRetries int) error {
	for i := 0; i < maxRetries; i++ {
		worker := qm.selectWorker()
		if worker == nil {
			continue
		}

		select {
		case worker.eventCh <- event:
			return nil
		default:
			// Try next worker
			continue
		}
	}
	
	return ErrAllWorkersAreBusy
}

// selectWorker selects a worker using multiple strategies
func (qm *QueueMonitorSafe) selectWorker() *WorkerSafe {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	if len(qm.workers) == 0 {
		return nil
	}

	// Strategy 1: Try least busy worker
	if worker := qm.getLeastBusyWorker(); worker != nil {
		return worker
	}

	// Strategy 2: Round-robin fallback
	return qm.getRoundRobinWorker()
}

// getLeastBusyWorker returns the worker with smallest queue
func (qm *QueueMonitorSafe) getLeastBusyWorker() *WorkerSafe {
	var leastBusy *WorkerSafe
	minQueueSize := int(^uint(0) >> 1) // Max int

	for _, worker := range qm.workers {
		queueSize := worker.GetQueueSize()
		if queueSize < minQueueSize {
			minQueueSize = queueSize
			leastBusy = worker
		}
	}

	// Only return if queue is not full
	if leastBusy != nil && minQueueSize < cap(leastBusy.eventCh) {
		return leastBusy
	}

	return nil
}

// getRoundRobinWorker returns next worker in round-robin order
func (qm *QueueMonitorSafe) getRoundRobinWorker() *WorkerSafe {
	index := atomic.AddInt32(&qm.workerIndex, 1)
	workerIdx := int(index-1) % len(qm.workers)
	return qm.workers[workerIdx]
}

// GetMetrics returns queue monitor metrics
func (qm *QueueMonitorSafe) GetMetrics() QueueMonitorMetrics {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	metrics := QueueMonitorMetrics{
		WorkerCount:   len(qm.workers),
		WorkerMetrics: make([]WorkerMetrics, 0, len(qm.workers)),
	}

	for _, worker := range qm.workers {
		metrics.WorkerMetrics = append(metrics.WorkerMetrics, worker.GetMetrics())
	}

	return metrics
}

// QueueMonitorMetrics represents queue monitor metrics
type QueueMonitorMetrics struct {
	WorkerCount   int
	WorkerMetrics []WorkerMetrics
}

// WorkerPoolSafe manages a pool of workers with thread-safe operations
type WorkerPoolSafe struct {
	workers    []*WorkerSafe
	mu         sync.RWMutex
	maxWorkers int
	sender     *WebhookSenderSafe
	logger     logging.Logger
}

// NewWorkerPoolSafe creates a new thread-safe worker pool
func NewWorkerPoolSafe(sender *WebhookSenderSafe, maxWorkers int) *WorkerPoolSafe {
	return &WorkerPoolSafe{
		workers:    make([]*WorkerSafe, 0, maxWorkers),
		maxWorkers: maxWorkers,
		sender:     sender,
		logger:     sender.logger,
	}
}

// ScaleUp adds more workers if needed
func (wp *WorkerPoolSafe) ScaleUp(count int) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	currentCount := len(wp.workers)
	if currentCount+count > wp.maxWorkers {
		count = wp.maxWorkers - currentCount
	}

	if count <= 0 {
		return ErrMaxWorkersReached
	}

	ctx := context.Background()
	wg := &sync.WaitGroup{}

	for i := 0; i < count; i++ {
		worker := NewWorkerSafe(currentCount+i, wp.sender, 100)
		wp.workers = append(wp.workers, worker)
		
		wg.Add(1)
		go worker.Start(ctx, wg)
	}

	wp.logger.Info("워커 풀 확장됨",
		logging.F("added", count),
		logging.F("total", len(wp.workers)))

	return nil
}

// ScaleDown removes workers gracefully
func (wp *WorkerPoolSafe) ScaleDown(count int) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	currentCount := len(wp.workers)
	if count > currentCount {
		count = currentCount
	}

	if count <= 0 {
		return nil
	}

	// Stop workers from the end
	for i := 0; i < count; i++ {
		idx := currentCount - 1 - i
		worker := wp.workers[idx]
		worker.Stop()
	}

	// Remove stopped workers
	wp.workers = wp.workers[:currentCount-count]

	wp.logger.Info("워커 풀 축소됨",
		logging.F("removed", count),
		logging.F("total", len(wp.workers)))

	return nil
}

// GetWorkers returns a copy of current workers
func (wp *WorkerPoolSafe) GetWorkers() []*WorkerSafe {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	workers := make([]*WorkerSafe, len(wp.workers))
	copy(workers, wp.workers)
	return workers
}

// GetMetrics returns worker pool metrics
func (wp *WorkerPoolSafe) GetMetrics() WorkerPoolMetrics {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	var totalProcessed, totalErrors int64
	var totalQueueSize, totalRetrySize int

	for _, worker := range wp.workers {
		metrics := worker.GetMetrics()
		totalProcessed += metrics.ProcessedCount
		totalErrors += metrics.ErrorCount
		totalQueueSize += metrics.QueueSize
		totalRetrySize += metrics.RetryQueueSize
	}

	return WorkerPoolMetrics{
		WorkerCount:      len(wp.workers),
		TotalProcessed:   totalProcessed,
		TotalErrors:      totalErrors,
		TotalQueueSize:   totalQueueSize,
		TotalRetrySize:   totalRetrySize,
		MaxWorkers:       wp.maxWorkers,
		AvgQueueSize:     float64(totalQueueSize) / float64(len(wp.workers)),
		AvgRetrySize:     float64(totalRetrySize) / float64(len(wp.workers)),
		ErrorRate:        float64(totalErrors) / float64(totalProcessed+1), // +1 to avoid division by zero
	}
}

// WorkerPoolMetrics represents worker pool performance metrics
type WorkerPoolMetrics struct {
	WorkerCount    int
	TotalProcessed int64
	TotalErrors    int64
	TotalQueueSize int
	TotalRetrySize int
	MaxWorkers     int
	AvgQueueSize   float64
	AvgRetrySize   float64
	ErrorRate      float64
}