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

// WebhookSenderSafe is a thread-safe implementation of webhook sender
type WebhookSenderSafe struct {
	config         configs.WebhookConfig
	logger         logging.Logger
	queue          EventQueue
	rateLimiter    RateLimiter
	adapters       map[string]WebhookAdapter
	batchManager   *BatchManagerSafe
	metrics        *SenderMetricsSafe
	failureQueue   *PersistentFailureQueue
	historyManager *WebhookHistoryManager
	
	// Thread-safe worker management
	workerPool     *WorkerPoolSafe
	queueMonitor   *QueueMonitorSafe
	
	// Lifecycle management
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	
	// State management with atomic operations
	started        int32 // 0 = stopped, 1 = started
	stopping       int32 // 0 = running, 1 = stopping
}

// NewWebhookSenderSafe creates a new thread-safe webhook sender
func NewWebhookSenderSafe(config configs.WebhookConfig, queue EventQueue) *WebhookSenderSafe {
	ctx, cancel := context.WithCancel(context.Background())
	
	sender := &WebhookSenderSafe{
		config:         config,
		logger:         logging.GetLogger(),
		queue:          queue,
		rateLimiter:    NewRateLimiter(config.RateLimit),
		adapters:       make(map[string]WebhookAdapter),
		metrics:        NewSenderMetricsSafe(),
		failureQueue:   NewPersistentFailureQueue(config.Retry.FailureDir),
		historyManager: NewWebhookHistoryManager(config.History),
		ctx:            ctx,
		cancel:         cancel,
	}
	
	// Initialize adapters
	sender.initializeAdapters()
	
	// Create worker pool
	sender.workerPool = NewWorkerPoolSafe(sender, config.WorkerCount)
	
	// Create batch manager
	sender.batchManager = NewBatchManagerSafe(config.Batching, sender.logger)
	
	return sender
}

// Start starts the webhook sender
func (ws *WebhookSenderSafe) Start() error {
	// Check if already started
	if !atomic.CompareAndSwapInt32(&ws.started, 0, 1) {
		return ErrAlreadyStarted
	}
	
	ws.logger.Info("웹훅 발송자 시작 중")
	
	// Start worker pool
	if err := ws.workerPool.ScaleUp(ws.config.WorkerCount); err != nil {
		atomic.StoreInt32(&ws.started, 0)
		return err
	}
	
	// Get workers for queue monitor
	workers := ws.workerPool.GetWorkers()
	ws.queueMonitor = NewQueueMonitorSafe(ws, workers)
	
	// Start queue monitor
	ws.wg.Add(1)
	go ws.queueMonitor.Start(ws.ctx, &ws.wg)
	
	// Start batch manager
	ws.wg.Add(1)
	go ws.batchManager.Start(ws.ctx, &ws.wg)
	
	// Start metrics collector
	ws.wg.Add(1)
	go ws.metricsCollector(&ws.wg)
	
	// Start health monitor
	ws.wg.Add(1)
	go ws.healthMonitor(&ws.wg)
	
	ws.logger.Info("웹훅 발송자 시작됨",
		logging.F("workers", ws.config.WorkerCount),
		logging.F("rate_limit", ws.config.RateLimit.RequestsPerSecond))
	
	return nil
}

// Stop gracefully stops the webhook sender
func (ws *WebhookSenderSafe) Stop() error {
	// Check if already stopping
	if !atomic.CompareAndSwapInt32(&ws.stopping, 0, 1) {
		return ErrAlreadyStopping
	}
	
	// Check if started
	if atomic.LoadInt32(&ws.started) == 0 {
		return ErrNotStarted
	}
	
	ws.logger.Info("웹훅 발송자 종료 중")
	
	// Cancel context to signal all goroutines
	ws.cancel()
	
	// Wait for all goroutines with timeout
	done := make(chan struct{})
	go func() {
		ws.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		ws.logger.Info("웹훅 발송자 정상 종료됨")
	case <-time.After(30 * time.Second):
		ws.logger.Warn("웹훅 발송자 종료 시간 초과")
		return ErrShutdownTimeout
	}
	
	// Update state
	atomic.StoreInt32(&ws.started, 0)
	atomic.StoreInt32(&ws.stopping, 0)
	
	return nil
}

// Send sends an alert event (thread-safe)
func (ws *WebhookSenderSafe) Send(event *alerts.AlertEvent) error {
	// Check if running
	if atomic.LoadInt32(&ws.started) == 0 {
		return ErrNotStarted
	}
	
	if atomic.LoadInt32(&ws.stopping) == 1 {
		return ErrShuttingDown
	}
	
	// Validate event
	if event == nil || event.ID == "" {
		return ErrInvalidEvent
	}
	
	// Apply rate limiting
	if !ws.rateLimiter.Allow() {
		ws.metrics.IncrementRateLimited()
		return ErrRateLimited
	}
	
	// Add to queue with timeout
	ctx, cancel := context.WithTimeout(ws.ctx, 5*time.Second)
	defer cancel()
	
	select {
	case ws.queue.Send() <- event:
		ws.metrics.IncrementQueued()
		return nil
	case <-ctx.Done():
		ws.metrics.IncrementDropped()
		return ErrQueueTimeout
	}
}

// SendBatch sends multiple events as a batch (thread-safe)
func (ws *WebhookSenderSafe) SendBatch(events []*alerts.AlertEvent, endpoint configs.WebhookEndpointConfig) error {
	// Check if running
	if atomic.LoadInt32(&ws.started) == 0 {
		return ErrNotStarted
	}
	
	// Add to batch manager
	for _, event := range events {
		if event != nil && event.ID != "" {
			ws.batchManager.AddEvent(event, endpoint)
		}
	}
	
	return nil
}

// GetMetrics returns current metrics (thread-safe)
func (ws *WebhookSenderSafe) GetMetrics() SenderMetrics {
	poolMetrics := ws.workerPool.GetMetrics()
	senderMetrics := ws.metrics.GetMetrics()
	
	return SenderMetrics{
		TotalSent:    senderMetrics.TotalSent,
		TotalFailed:  senderMetrics.TotalFailed,
		TotalRetries: senderMetrics.TotalRetries,
		QueueSize:    int64(poolMetrics.TotalQueueSize),
		WorkerCount:  int64(poolMetrics.WorkerCount),
		BatchStats:   ws.batchManager.GetStats(),
	}
}

// metricsCollector periodically collects and logs metrics
func (ws *WebhookSenderSafe) metricsCollector(wg *sync.WaitGroup) {
	defer wg.Done()
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			metrics := ws.GetMetrics()
			ws.logger.Info("웹훅 발송자 메트릭",
				logging.F("total_sent", metrics.TotalSent),
				logging.F("total_failed", metrics.TotalFailed),
				logging.F("queue_size", metrics.QueueSize),
				logging.F("worker_count", metrics.WorkerCount))
				
		case <-ws.ctx.Done():
			return
		}
	}
}

// healthMonitor monitors system health and adjusts workers
func (ws *WebhookSenderSafe) healthMonitor(wg *sync.WaitGroup) {
	defer wg.Done()
	
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			poolMetrics := ws.workerPool.GetMetrics()
			
			// Auto-scale based on queue size
			if poolMetrics.AvgQueueSize > 80 {
				// Scale up if queues are getting full
				if err := ws.workerPool.ScaleUp(2); err != nil {
					ws.logger.Warn("워커 확장 실패", logging.F("error", err))
				}
			} else if poolMetrics.AvgQueueSize < 20 && poolMetrics.WorkerCount > ws.config.WorkerCount {
				// Scale down if queues are mostly empty
				if err := ws.workerPool.ScaleDown(1); err != nil {
					ws.logger.Warn("워커 축소 실패", logging.F("error", err))
				}
			}
			
			// Check error rate
			if poolMetrics.ErrorRate > 0.1 { // More than 10% errors
				ws.logger.Warn("높은 오류율 감지",
					logging.F("error_rate", poolMetrics.ErrorRate),
					logging.F("total_errors", poolMetrics.TotalErrors))
			}
			
		case <-ws.ctx.Done():
			return
		}
	}
}

// initializeAdapters initializes webhook adapters
func (ws *WebhookSenderSafe) initializeAdapters() {
	// Initialize default HTTP adapter
	ws.adapters["http"] = NewHTTPAdapter(ws.config.HTTP)
	ws.adapters["https"] = ws.adapters["http"]
	
	// Initialize other adapters based on config
	if ws.config.Slack.Enabled {
		ws.adapters["slack"] = NewSlackAdapter(ws.config.Slack)
	}
	
	if ws.config.Discord.Enabled {
		ws.adapters["discord"] = NewDiscordAdapter(ws.config.Discord)
	}
}

// SenderMetricsSafe is a thread-safe metrics collector
type SenderMetricsSafe struct {
	totalSent        int64
	totalFailed      int64
	totalRetries     int64
	totalQueued      int64
	totalDropped     int64
	totalRateLimited int64
}

// NewSenderMetricsSafe creates new thread-safe metrics
func NewSenderMetricsSafe() *SenderMetricsSafe {
	return &SenderMetricsSafe{}
}

// IncrementSent increments sent counter
func (m *SenderMetricsSafe) IncrementSent() {
	atomic.AddInt64(&m.totalSent, 1)
}

// IncrementFailed increments failed counter
func (m *SenderMetricsSafe) IncrementFailed() {
	atomic.AddInt64(&m.totalFailed, 1)
}

// IncrementRetries increments retries counter
func (m *SenderMetricsSafe) IncrementRetries() {
	atomic.AddInt64(&m.totalRetries, 1)
}

// IncrementQueued increments queued counter
func (m *SenderMetricsSafe) IncrementQueued() {
	atomic.AddInt64(&m.totalQueued, 1)
}

// IncrementDropped increments dropped counter
func (m *SenderMetricsSafe) IncrementDropped() {
	atomic.AddInt64(&m.totalDropped, 1)
}

// IncrementRateLimited increments rate limited counter
func (m *SenderMetricsSafe) IncrementRateLimited() {
	atomic.AddInt64(&m.totalRateLimited, 1)
}

// GetMetrics returns current metrics
func (m *SenderMetricsSafe) GetMetrics() SenderMetricsSafe {
	return SenderMetricsSafe{
		totalSent:        atomic.LoadInt64(&m.totalSent),
		totalFailed:      atomic.LoadInt64(&m.totalFailed),
		totalRetries:     atomic.LoadInt64(&m.totalRetries),
		totalQueued:      atomic.LoadInt64(&m.totalQueued),
		totalDropped:     atomic.LoadInt64(&m.totalDropped),
		totalRateLimited: atomic.LoadInt64(&m.totalRateLimited),
	}
}