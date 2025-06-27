package webhook

import (
	"context"
	"fmt"
	"sync"
	"time"

	"proxynd/alerts"
	"proxynd/configs"
	"proxynd/logging"
)

// EventQueue 이벤트 큐 인터페이스
type EventQueue interface {
	Push(event *alerts.AlertEvent) error
	Pop() (*alerts.AlertEvent, error)
	Size() int
	Clear() error
	Close() error
}

// RetryPolicy 재시도 정책
type RetryPolicy struct {
	MaxAttempts   int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
}

// RateLimiter 속도 제한 인터페이스
type RateLimiter interface {
	Allow() bool
	Wait(ctx context.Context) error
	Tokens() int
	SetLimit(limit int)
}

// WebhookSender 웹훅 전송 핵심 엔진
type WebhookSender struct {
	config      configs.WebhookConfig
	logger      logging.Logger
	queue       EventQueue
	rateLimiter RateLimiter
	adapters    map[string]WebhookAdapter
	workers     []*Worker
	metrics     *SenderMetrics
	stopCh      chan struct{}
	wg          sync.WaitGroup
	mu          sync.RWMutex
	started     bool
}

// Worker 이벤트 처리 워커
type Worker struct {
	id      int
	sender  *WebhookSender
	stopCh  chan struct{}
	eventCh chan *alerts.AlertEvent
	logger  logging.Logger
	retryQ  *RetryQueue
}

// RetryQueue 재시도 큐
type RetryQueue struct {
	items    []*RetryItem
	mu       sync.Mutex
	notifyCh chan struct{}
}

// RetryItem 재시도 항목
type RetryItem struct {
	Event     *alerts.AlertEvent
	Endpoint  string
	Attempt   int
	NextRetry time.Time
	LastError error
	CreatedAt time.Time
}

// SenderMetrics 전송 메트릭
type SenderMetrics struct {
	TotalSent    int64
	TotalFailed  int64
	TotalRetries int64
	QueueSize    int64
	WorkerCount  int64
	mu           sync.RWMutex
}

// NewWebhookSender 새로운 웹훅 전송기 생성
func NewWebhookSender(config configs.WebhookConfig) (*WebhookSender, error) {
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

	sender := &WebhookSender{
		config:      config,
		logger:      logger,
		queue:       queue,
		rateLimiter: rateLimiter,
		adapters:    make(map[string]WebhookAdapter),
		workers:     make([]*Worker, 0),
		metrics:     &SenderMetrics{},
		stopCh:      make(chan struct{}),
	}

	// 기본 어댑터 등록
	sender.RegisterAdapter(NewGenericWebhookAdapter())
	sender.RegisterAdapter(NewSlackWebhookAdapter())
	sender.RegisterAdapter(NewDiscordWebhookAdapter())

	return sender, nil
}

// RegisterAdapter 웹훅 어댑터 등록
func (ws *WebhookSender) RegisterAdapter(adapter WebhookAdapter) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.adapters[adapter.Name()] = adapter
	ws.logger.Info("웹훅 어댑터 등록됨", logging.F("adapter", adapter.Name()))
}

// Start 웹훅 전송기 시작
func (ws *WebhookSender) Start(ctx context.Context) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.started {
		return fmt.Errorf("webhook sender already started")
	}

	if !ws.config.Enabled {
		ws.logger.Info("웹훅 시스템이 비활성화되어 있습니다")
		return nil
	}

	// 워커 생성 및 시작
	workerCount := 5 // 기본 워커 수
	if ws.config.Buffering.Enabled {
		workerCount = 10 // 버퍼링 활성화 시 더 많은 워커
	}

	for i := 0; i < workerCount; i++ {
		worker := &Worker{
			id:      i,
			sender:  ws,
			stopCh:  make(chan struct{}),
			eventCh: make(chan *alerts.AlertEvent, 100),
			logger:  ws.logger,
			retryQ:  NewRetryQueue(),
		}
		ws.workers = append(ws.workers, worker)

		ws.wg.Add(1)
		go worker.Start(ctx)
	}

	// 큐 모니터링 고루틴 시작
	ws.wg.Add(1)
	go ws.queueMonitor(ctx)

	// 재시도 스케줄러 시작
	ws.wg.Add(1)
	go ws.retryScheduler(ctx)

	// 메트릭 수집기 시작
	if ws.config.Monitoring.Enabled {
		ws.wg.Add(1)
		go ws.metricsCollector(ctx)
	}

	ws.started = true
	ws.logger.Info("웹훅 전송기 시작됨",
		logging.F("workers", workerCount),
		logging.F("endpoints", len(ws.config.Endpoints)),
		logging.F("buffering", ws.config.Buffering.Enabled))

	return nil
}

// Stop 웹훅 전송기 중지
func (ws *WebhookSender) Stop(ctx context.Context) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if !ws.started {
		return nil
	}

	ws.logger.Info("웹훅 전송기 중지 중...")

	// 모든 워커 중지
	for _, worker := range ws.workers {
		close(worker.stopCh)
	}

	// 메인 중지 신호
	close(ws.stopCh)

	// 모든 고루틴이 종료될 때까지 대기 (타임아웃 포함)
	done := make(chan struct{})
	go func() {
		ws.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		ws.logger.Info("모든 웹훅 워커가 정상적으로 종료됨")
	case <-ctx.Done():
		ws.logger.Warn("웹훅 워커 종료 타임아웃")
	}

	// 큐 정리
	if ws.queue != nil {
		ws.queue.Close()
	}

	ws.started = false
	return nil
}

// SendEvent 이벤트 전송 (비동기)
func (ws *WebhookSender) SendEvent(event *alerts.AlertEvent) error {
	if !ws.config.Enabled {
		return nil
	}

	ws.mu.RLock()
	defer ws.mu.RUnlock()

	if !ws.started {
		return fmt.Errorf("webhook sender not started")
	}

	// 이벤트 필터링
	if !ws.shouldSendEvent(event) {
		ws.logger.Debug("이벤트 필터링으로 제외됨",
			logging.F("event_type", event.Type),
			logging.F("level", event.Level))
		return nil
	}

	// 큐에 추가
	if err := ws.queue.Push(event); err != nil {
		ws.logger.Error("이벤트 큐 추가 실패",
			logging.F("error", err.Error()),
			logging.F("event_id", event.ID))
		return fmt.Errorf("failed to queue event: %w", err)
	}

	ws.logger.Debug("이벤트가 큐에 추가됨",
		logging.F("event_id", event.ID),
		logging.F("event_type", event.Type),
		logging.F("queue_size", ws.queue.Size()))

	return nil
}

// SendEventSync 이벤트 동기 전송
func (ws *WebhookSender) SendEventSync(ctx context.Context, event *alerts.AlertEvent) error {
	if !ws.config.Enabled {
		return nil
	}

	// 이벤트 필터링
	if !ws.shouldSendEvent(event) {
		return nil
	}

	// 각 엔드포인트로 직접 전송
	var errors []error
	for _, endpoint := range ws.config.Endpoints {
		if !endpoint.Enabled {
			continue
		}

		if !ws.matchesEndpointFilter(event, endpoint) {
			continue
		}

		if err := ws.sendToEndpoint(ctx, event, endpoint); err != nil {
			errors = append(errors, fmt.Errorf("endpoint %s: %w", endpoint.Name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("some endpoints failed: %v", errors)
	}

	return nil
}

// shouldSendEvent 이벤트 전송 여부 판단
func (ws *WebhookSender) shouldSendEvent(event *alerts.AlertEvent) bool {
	if !ws.config.EventFilter.Enabled {
		return true
	}

	// 레벨 필터링
	if !ws.matchesLevelFilter(event) {
		return false
	}

	// 이벤트 규칙 적용
	for _, rule := range ws.config.EventFilter.EventRules {
		if !rule.Enabled {
			continue
		}

		if ws.matchesEventRule(event, rule) {
			// 규칙에 따른 액션 수행
			for _, action := range rule.Actions {
				switch action {
				case "deny":
					return false
				case "allow":
					return true
				}
			}
		}
	}

	return true
}

// matchesLevelFilter 레벨 필터 확인
func (ws *WebhookSender) matchesLevelFilter(event *alerts.AlertEvent) bool {
	levelPriority := map[string]int{
		"INFO":     1,
		"WARNING":  2,
		"ERROR":    3,
		"CRITICAL": 4,
	}

	eventPriority := levelPriority[string(event.Level)]
	if eventPriority == 0 {
		eventPriority = 1 // 기본값
	}

	// 이벤트 타입별 레벨 오버라이드 확인
	for pattern, level := range ws.config.EventFilter.LevelOverrides {
		if ws.matchesPattern(event.Type, pattern) {
			minPriority := levelPriority[level]
			return eventPriority >= minPriority
		}
	}

	// 기본 레벨 확인
	minPriority := levelPriority[ws.config.EventFilter.DefaultLevel]
	if minPriority == 0 {
		minPriority = 1
	}

	return eventPriority >= minPriority
}

// matchesEndpointFilter 엔드포인트별 필터 확인
func (ws *WebhookSender) matchesEndpointFilter(event *alerts.AlertEvent, endpoint configs.WebhookEndpointConfig) bool {
	// 이벤트 타입 필터
	if len(endpoint.EventTypes) > 0 {
		matched := false
		for _, eventType := range endpoint.EventTypes {
			if eventType == "*" || ws.matchesPattern(event.Type, eventType) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 레벨 필터
	if endpoint.Filters.MinLevel != "" {
		levelPriority := map[string]int{
			"INFO": 1, "WARNING": 2, "ERROR": 3, "CRITICAL": 4,
		}
		eventPriority := levelPriority[string(event.Level)]
		minPriority := levelPriority[endpoint.Filters.MinLevel]
		if eventPriority < minPriority {
			return false
		}
	}

	// 제외 타입 필터
	for _, excludeType := range endpoint.Filters.ExcludeTypes {
		if ws.matchesPattern(event.Type, excludeType) {
			return false
		}
	}

	// 포함 타입 필터
	if len(endpoint.Filters.IncludeTypes) > 0 {
		matched := false
		for _, includeType := range endpoint.Filters.IncludeTypes {
			if ws.matchesPattern(event.Type, includeType) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

// matchesEventRule 이벤트 규칙 매칭
func (ws *WebhookSender) matchesEventRule(event *alerts.AlertEvent, rule configs.WebhookEventRule) bool {
	// 기본적인 조건 매칭 로직 (향후 확장 가능)
	if typeCondition, exists := rule.Conditions["type"]; exists {
		if typeStr, ok := typeCondition.(string); ok {
			if !ws.matchesPattern(event.Type, typeStr) {
				return false
			}
		}
	}

	if levelCondition, exists := rule.Conditions["level"]; exists {
		if levelStr, ok := levelCondition.(string); ok {
			if string(event.Level) != levelStr {
				return false
			}
		}
	}

	return true
}

// matchesPattern 패턴 매칭 (와일드카드 지원)
func (ws *WebhookSender) matchesPattern(value, pattern string) bool {
	if pattern == "*" {
		return true
	}

	// 간단한 와일드카드 매칭 (예: "cache.*")
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(value) >= len(prefix) && value[:len(prefix)] == prefix
	}

	return value == pattern
}

// sendToEndpoint 특정 엔드포인트로 전송
func (ws *WebhookSender) sendToEndpoint(ctx context.Context, event *alerts.AlertEvent, endpoint configs.WebhookEndpointConfig) error {
	// 어댑터 선택
	adapterName := "generic"
	if endpoint.Format == "slack" {
		adapterName = "slack"
	} else if endpoint.Format == "discord" {
		adapterName = "discord"
	}

	adapter, exists := ws.adapters[adapterName]
	if !exists {
		adapter = ws.adapters["generic"] // 폴백
	}

	// 재시도 정책 적용
	retryPolicy := RetryPolicy{
		MaxAttempts:   ws.config.Retry.MaxAttempts,
		InitialDelay:  time.Second,
		MaxDelay:      time.Minute * 5,
		BackoffFactor: ws.config.Retry.BackoffFactor,
	}

	// 재시도 로직과 함께 전송
	var lastErr error
	for attempt := 1; attempt <= retryPolicy.MaxAttempts; attempt++ {
		// 속도 제한 확인
		if err := ws.rateLimiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limit wait failed: %w", err)
		}

		// 전송 시도
		if err := adapter.Send(ctx, event, endpoint); err != nil {
			lastErr = err
			ws.metrics.incrementFailed()

			// 재시도 가능한 오류인지 확인
			if !ws.isRetryableError(err) {
				return fmt.Errorf("non-retryable error: %w", err)
			}

			if attempt < retryPolicy.MaxAttempts {
				delay := ws.calculateBackoffDelay(attempt, retryPolicy)
				ws.logger.Warn("웹훅 전송 실패, 재시도 예정",
					logging.F("endpoint", endpoint.Name),
					logging.F("attempt", attempt),
					logging.F("delay", delay),
					logging.F("error", err.Error()))

				select {
				case <-time.After(delay):
					ws.metrics.incrementRetries()
					continue
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		} else {
			// 성공
			ws.metrics.incrementSent()
			ws.logger.Debug("웹훅 전송 성공",
				logging.F("endpoint", endpoint.Name),
				logging.F("event_id", event.ID))
			return nil
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", retryPolicy.MaxAttempts, lastErr)
}

// calculateBackoffDelay 백오프 지연 시간 계산
func (ws *WebhookSender) calculateBackoffDelay(attempt int, policy RetryPolicy) time.Duration {
	delay := time.Duration(float64(policy.InitialDelay) *
		pow(policy.BackoffFactor, float64(attempt-1)))

	if delay > policy.MaxDelay {
		delay = policy.MaxDelay
	}

	return delay
}

// pow 간단한 거듭제곱 함수
func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

// isRetryableError 재시도 가능한 오류인지 확인
func (ws *WebhookSender) isRetryableError(err error) bool {
	// HTTP 상태 코드나 오류 타입을 기반으로 재시도 가능 여부 판단
	// 구현은 실제 어댑터에서 더 정교하게 처리
	return true
}

// GetMetrics 메트릭 반환
func (ws *WebhookSender) GetMetrics() *SenderMetrics {
	ws.metrics.mu.RLock()
	defer ws.metrics.mu.RUnlock()

	return &SenderMetrics{
		TotalSent:    ws.metrics.TotalSent,
		TotalFailed:  ws.metrics.TotalFailed,
		TotalRetries: ws.metrics.TotalRetries,
		QueueSize:    int64(ws.queue.Size()),
		WorkerCount:  int64(len(ws.workers)),
	}
}

// queueMonitor 큐 모니터링 고루틴
func (ws *WebhookSender) queueMonitor(ctx context.Context) {
	defer ws.wg.Done()

	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 큐에서 이벤트를 가져와서 워커에게 분배
			for {
				event, err := ws.queue.Pop()
				if err != nil {
					break // 큐가 비어있음
				}

				// 가장 적게 일하는 워커 찾기
				worker := ws.getLeastBusyWorker()
				if worker != nil {
					select {
					case worker.eventCh <- event:
						// 성공적으로 워커에게 전달
					default:
						// 워커가 바쁘면 다시 큐에 넣기
						ws.queue.Push(event)
					}
				}
			}
		case <-ws.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// getLeastBusyWorker 가장 적게 일하는 워커 반환
func (ws *WebhookSender) getLeastBusyWorker() *Worker {
	var leastBusy *Worker
	minLoad := int(^uint(0) >> 1) // max int

	for _, worker := range ws.workers {
		load := len(worker.eventCh)
		if load < minLoad {
			minLoad = load
			leastBusy = worker
		}
	}

	return leastBusy
}

// retryScheduler 재시도 스케줄러
func (ws *WebhookSender) retryScheduler(ctx context.Context) {
	defer ws.wg.Done()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 각 워커의 재시도 큐 처리
			for _, worker := range ws.workers {
				worker.retryQ.processRetries(ctx, ws)
			}
		case <-ws.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// metricsCollector 메트릭 수집기
func (ws *WebhookSender) metricsCollector(ctx context.Context) {
	defer ws.wg.Done()

	interval, _ := time.ParseDuration(ws.config.Monitoring.MetricsInterval)
	if interval == 0 {
		interval = time.Second * 30
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ws.collectMetrics()
		case <-ws.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// collectMetrics 메트릭 수집
func (ws *WebhookSender) collectMetrics() {
	metrics := ws.GetMetrics()

	ws.logger.Debug("웹훅 메트릭 수집",
		logging.F("total_sent", metrics.TotalSent),
		logging.F("total_failed", metrics.TotalFailed),
		logging.F("total_retries", metrics.TotalRetries),
		logging.F("queue_size", metrics.QueueSize),
		logging.F("worker_count", metrics.WorkerCount))
}

// SenderMetrics 메서드들
func (sm *SenderMetrics) incrementSent() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.TotalSent++
}

func (sm *SenderMetrics) incrementFailed() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.TotalFailed++
}

func (sm *SenderMetrics) incrementRetries() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.TotalRetries++
}
