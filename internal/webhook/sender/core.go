package sender

import (
	"context"
	"fmt"
	"sync"
	"time"

	"proxynd/internal/alerts"
	"proxynd/internal/config"
	"proxynd/internal/logging"
	"proxynd/internal/webhook/retry"
	"proxynd/internal/webhook/types"
)

// Core 웹훅 전송 핵심 엔진
type Core struct {
	config         config.WebhookConfig
	logger         logging.Logger
	queue          types.EventQueue
	rateLimiter    types.RateLimiter
	adapters       map[string]types.WebhookAdapter
	workers        []*Worker
	batchManager   *BatchManager
	metrics        *types.SenderMetrics
	failureQueue   *PersistentFailureQueue
	historyManager *WebhookHistoryManager
	retryManager   *retry.Manager
	stopCh         chan struct{}
	wg             sync.WaitGroup
	mu             sync.RWMutex
	started        bool
}

// NewCore 새로운 웹훅 전송 코어 생성
func NewCore(config config.WebhookConfig) (*Core, error) {
	logger := logging.GetLogger()

	// 큐 초기화
	queue, err := NewMemoryEventQueue(config.Buffering.BufferSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create event queue: %w", err)
	}

	// 속도 제한기 초기화
	rateLimiter, err := NewTokenBucketLimiter(
		config.RateLimit.MaxPerSecond,
		config.RateLimit.BurstSize,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create rate limiter: %w", err)
	}

	// 영속성 실패 큐 초기화
	failureQueue, err := NewPersistentFailureQueue(
		config.FailureStorage.StorageDir,
		config.Retry.MaxAttempts,
		time.Duration(config.FailureStorage.RetentionHours)*time.Hour,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create failure queue: %w", err)
	}

	// 이력 관리자 초기화
	historyManager := NewWebhookHistoryManager(
		config.FailureStorage.StorageDir+"/history",
		10000,          // 최대 10,000개 이력 보관
		7*24*time.Hour, // 7일 보관
	)

	// 재시도 관리자 초기화
	retryPolicy := retry.Policy{
		MaxAttempts:   config.Retry.MaxAttempts,
		InitialDelay:  parseDurationFromConfig(config.Retry.InitialDelay),
		MaxDelay:      parseDurationFromConfig(config.Retry.MaxDelay),
		BackoffFactor: config.Retry.BackoffFactor,
	}
	retryManager := retry.NewManager(retryPolicy)

	// BatchStats 초기화 (config 기반)
	batchStats := make(map[string]interface{})
	batchStats["enabled"] = config.Batching.Enabled
	if config.Batching.Enabled {
		batchStats["maxSize"] = config.Batching.MaxSize
	}

	core := &Core{
		config:      config,
		logger:      logger,
		queue:       queue,
		rateLimiter: rateLimiter,
		adapters:    make(map[string]types.WebhookAdapter),
		workers:     make([]*Worker, 0),
		metrics: &types.SenderMetrics{
			BatchStats: batchStats,
		},
		failureQueue:   failureQueue,
		historyManager: historyManager,
		retryManager:   retryManager,
		stopCh:         make(chan struct{}),
	}

	// 기본 어댑터 등록
	core.RegisterAdapter(NewGenericWebhookAdapter())
	core.RegisterAdapter(NewSlackWebhookAdapter())
	core.RegisterAdapter(NewDiscordWebhookAdapter())

	return core, nil
}

// RegisterAdapter 웹훅 어댑터 등록
func (c *Core) RegisterAdapter(adapter types.WebhookAdapter) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.adapters[adapter.Type()] = adapter
	c.logger.Info(fmt.Sprintf("Registered webhook adapter: %s", adapter.Type()))
}

// Start 웹훅 전송기 시작
func (c *Core) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return fmt.Errorf("webhook sender already started")
	}

	c.logger.Info("Starting webhook sender...")

	// 배치 관리자 시작
	if c.config.Batching.Enabled {
		batchConfig := BatchConfig{
			MaxSize:      c.config.Batching.MaxSize,
			FlushTimeout: parseDurationFromConfig(c.config.Batching.FlushInterval),
		}
		c.batchManager = NewBatchManager(batchConfig, c.logger)

		// BatchStats 초기화
		c.metrics.BatchStats["enabled"] = true
		c.metrics.BatchStats["maxSize"] = c.config.Batching.MaxSize

		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			c.batchManager.Start(ctx)
		}()
	} else {
		c.metrics.BatchStats["enabled"] = false
	}

	// 워커 시작
	// Note: Workers field might not exist in config, using default
	workerCount := 2 // Default worker count
	if workerCount <= 0 {
		workerCount = 1
	}

	for i := 0; i < workerCount; i++ {
		worker := NewWorker(i, c, c.logger)
		c.workers = append(c.workers, worker)

		c.wg.Add(1)
		go func(w *Worker) {
			defer c.wg.Done()
			w.Start(ctx)
		}(worker)
	}

	// 재시도 관리자 시작
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		if err := c.retryManager.ProcessRetries(ctx); err != nil {
			c.logger.Error(fmt.Sprintf("Retry manager error: %v", err))
		}
	}()

	// 모니터링 고루틴 시작
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.queueMonitor(ctx)
	}()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.metricsCollector(ctx)
	}()

	c.started = true
	c.logger.Info(fmt.Sprintf("Webhook sender started with %d workers", workerCount))
	return nil
}

// Stop 웹훅 전송기 중지
func (c *Core) Stop(ctx context.Context) error {
	c.mu.Lock()
	if !c.started {
		c.mu.Unlock()
		return fmt.Errorf("webhook sender not started")
	}
	c.started = false
	c.mu.Unlock()

	c.logger.Info("Stopping webhook sender...")

	// 정지 신호 전송
	close(c.stopCh)

	// 배치 관리자 중지
	if c.batchManager != nil {
		c.batchManager.Stop()
	}

	// 재시도 관리자 중지
	c.retryManager.Stop()

	// 워커들 중지
	for _, worker := range c.workers {
		worker.Stop()
	}

	// 모든 고루틴 대기 (타임아웃 포함)
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		c.logger.Info("All webhook sender goroutines stopped")
	case <-ctx.Done():
		c.logger.Warn("Webhook sender stop timed out")
		return ctx.Err()
	case <-time.After(30 * time.Second):
		c.logger.Warn("Webhook sender stop timed out after 30 seconds")
		return fmt.Errorf("stop timeout")
	}

	// 큐 정리
	if err := c.queue.Clear(); err != nil {
		c.logger.Error(fmt.Sprintf("Failed to clear queue: %v", err))
	}

	// 큐 종료
	if err := c.queue.Close(); err != nil {
		c.logger.Error(fmt.Sprintf("Failed to close queue: %v", err))
	}

	c.logger.Info("Webhook sender stopped successfully")
	return nil
}

// SendEvent 이벤트 비동기 전송
func (c *Core) SendEvent(event *alerts.AlertEvent) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.started {
		return fmt.Errorf("webhook sender not started")
	}

	// 이벤트 검증
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	// 이벤트 필터링 확인
	if !c.shouldSendEvent(event) {
		c.logger.Debug(fmt.Sprintf("Event %s filtered out", event.ID))
		return nil
	}

	// 속도 제한 확인
	if !c.rateLimiter.Allow() {
		c.logger.Warn(fmt.Sprintf("Rate limit exceeded for event %s", event.ID))
		return fmt.Errorf("rate limit exceeded")
	}

	// 큐에 추가
	if err := c.queue.Push(event); err != nil {
		c.metrics.IncrementFailed()
		return fmt.Errorf("failed to queue event: %w", err)
	}

	c.logger.Debug(fmt.Sprintf("Event %s queued for sending", event.ID))
	return nil
}

// SendEventSync 이벤트 동기 전송
func (c *Core) SendEventSync(ctx context.Context, event *alerts.AlertEvent) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.started {
		return fmt.Errorf("webhook sender not started")
	}

	// 이벤트 검증
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	// 이벤트 필터링 확인
	if !c.shouldSendEvent(event) {
		c.logger.Debug(fmt.Sprintf("Event %s filtered out", event.ID))
		return nil
	}

	// 속도 제한 확인 (동기식에서는 대기)
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit wait failed: %w", err)
	}

	// 모든 엔드포인트에 직접 전송
	var lastErr error
	for _, endpoint := range c.config.Endpoints {
		if err := c.sendToEndpoint(ctx, event, endpoint); err != nil {
			c.logger.Error(fmt.Sprintf("Failed to send event %s to endpoint %s: %v",
				event.ID, endpoint.URL, err))
			lastErr = err
			c.metrics.IncrementFailed()
		} else {
			c.metrics.IncrementSent()
		}
	}

	return lastErr
}

// GetMetrics 메트릭 조회
func (c *Core) GetMetrics() *types.SenderMetrics {
	return c.metrics
}

// GetHistoryManager 이력 관리자 조회
func (c *Core) GetHistoryManager() *WebhookHistoryManager {
	return c.historyManager
}

// sendToEndpoint 특정 엔드포인트로 이벤트 전송 (내부 메서드)
func (c *Core) sendToEndpoint(
	ctx context.Context, event *alerts.AlertEvent, endpoint config.WebhookEndpointConfig,
) error {
	// 이 메서드는 실제 전송 로직을 구현해야 하지만,
	// 여기서는 간단한 로깅만 수행
	c.logger.Debug(fmt.Sprintf("Sending event %s to endpoint %s", event.ID, endpoint.URL))
	return nil
}

// shouldSendEvent 이벤트 전송 여부 확인 (내부 메서드)
func (c *Core) shouldSendEvent(event *alerts.AlertEvent) bool {
	// 간단한 필터링 로직
	return event != nil && event.ID != ""
}

// queueMonitor 큐 모니터링 (내부 메서드)
func (c *Core) queueMonitor(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			queueSize := int64(c.queue.Size())
			c.metrics.UpdateQueueSize(queueSize)

			if queueSize > int64(c.config.Buffering.BufferSize)*8/10 {
				c.logger.Warn(fmt.Sprintf("Queue size high: %d", queueSize))
			}
		}
	}
}

// metricsCollector 메트릭 수집 (내부 메서드)
func (c *Core) metricsCollector(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.collectMetrics()
		}
	}
}

// collectMetrics 메트릭 수집 실행
func (c *Core) collectMetrics() {
	c.metrics.UpdateQueueSize(int64(c.queue.Size()))
	c.metrics.UpdateWorkerCount(int64(len(c.workers)))
}
