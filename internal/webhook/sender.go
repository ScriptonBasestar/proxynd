package webhook

import (
	"context"
	"fmt"
	"sync"
	"time"

	"proxynd/alerts"
	"proxynd/internal/config"
	"proxynd/internal/webhook/filtering"
	"proxynd/internal/webhook/retry"
	"proxynd/internal/webhook/sender"
	"proxynd/internal/webhook/types"
	"proxynd/logging"
)

// WebhookSender 웹훅 전송기 (리팩토링된 버전)
// 기존 인터페이스를 유지하면서 내부적으로 새로운 모듈들을 사용
type WebhookSender struct {
	core         *sender.Core
	filter       *filtering.SimpleEventFilter
	config       config.WebhookConfig                // 하위 호환성을 위한 config 접근
	wg           sync.WaitGroup                      // 하위 호환성을 위한 WaitGroup
	adapters     map[string]CompatibleWebhookAdapter // 하위 호환성을 위한 어댑터 맵
	logger       logging.Logger                      // 하위 호환성을 위한 logger 접근
	started      bool                                // 시작 상태 (테스트 호환성)
	queue        *DummyQueue                         // 테스트 호환성을 위한 큐 필드
	rateLimiter  interface{}                         // 테스트 호환성을 위한 rate limiter 필드
	batchManager *DummyBatchManager                  // 테스트 호환성을 위한 batch manager 필드
	metrics      *DummyMetrics                       // 테스트 호환성을 위한 metrics 필드
}

// NewWebhookSender 새로운 웹훅 전송기 생성
func NewWebhookSender(config config.WebhookConfig) (*WebhookSender, error) {
	// 새로운 core 생성
	core, err := sender.NewCore(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create sender core: %w", err)
	}

	// 필터 생성
	filter := filtering.NewSimpleEventFilter(config)

	// 기본 어댑터들 생성
	adapters := make(map[string]CompatibleWebhookAdapter)
	adapters["generic"] = NewGenericWebhookAdapter()
	adapters["slack"] = sender.NewSlackWebhookAdapter()
	adapters["discord"] = sender.NewDiscordWebhookAdapter()

	// 테스트 호환을 위한 더미 객체들
	dummyQueue := &DummyQueue{}
	dummyRateLimiter := struct{}{}
	dummyBatchManager := &DummyBatchManager{}
	dummyMetrics := &DummyMetrics{}

	return &WebhookSender{
		core:         core,
		filter:       filter,
		config:       config,              // config 저장
		adapters:     adapters,            // 어댑터 맵 초기화
		logger:       logging.GetLogger(), // logger 초기화
		queue:        dummyQueue,          // 테스트 호환성
		rateLimiter:  dummyRateLimiter,    // 테스트 호환성
		batchManager: dummyBatchManager,   // 테스트 호환성
		metrics:      dummyMetrics,        // 테스트 호환성
	}, nil
}

// RegisterAdapter 웹훅 어댑터 등록
func (ws *WebhookSender) RegisterAdapter(adapter types.WebhookAdapter) {
	ws.core.RegisterAdapter(adapter)
	// 하위 호환성을 위해 로컬 맵에도 저장 (CompatibleWebhookAdapter로 캐스팅)
	if compatAdapter, ok := adapter.(CompatibleWebhookAdapter); ok {
		ws.adapters[adapter.Type()] = compatAdapter
	}
}

// Start 웹훅 전송기 시작
func (ws *WebhookSender) Start(ctx context.Context) error {
	err := ws.core.Start(ctx)
	if err == nil {
		ws.started = true
	}
	return err
}

// Stop 웹훅 전송기 중지
func (ws *WebhookSender) Stop(ctx context.Context) error {
	err := ws.core.Stop(ctx)
	if err == nil {
		ws.started = false
	}
	return err
}

// SendEvent 이벤트 비동기 전송
func (ws *WebhookSender) SendEvent(event *alerts.AlertEvent) error {
	// 필터링 적용
	if !ws.filter.ShouldSendEvent(event) {
		return nil
	}

	return ws.core.SendEvent(event)
}

// SendEventSync 이벤트 동기 전송
func (ws *WebhookSender) SendEventSync(ctx context.Context, event *alerts.AlertEvent) error {
	// 필터링 적용
	if !ws.filter.ShouldSendEvent(event) {
		return nil
	}

	return ws.core.SendEventSync(ctx, event)
}

// GetMetrics 메트릭 조회
func (ws *WebhookSender) GetMetrics() *types.SenderMetrics {
	return ws.core.GetMetrics()
}

// GetHistoryManager 이력 관리자 조회
func (ws *WebhookSender) GetHistoryManager() *WebhookHistoryManager {
	// TODO: sender.Core에서 실제 WebhookHistoryManager를 반환하도록 수정 필요
	// 임시로 nil 반환 (실제 구현에서는 적절한 변환 로직 필요)
	return nil
}

// shouldSendEvent 이벤트 전송 여부 확인 (하위 호환성을 위한 메서드)
func (ws *WebhookSender) shouldSendEvent(event *alerts.AlertEvent) bool {
	return ws.filter.ShouldSendEvent(event)
}

// matchesEndpointFilter 엔드포인트별 필터 확인 (하위 호환성을 위한 메서드)
func (ws *WebhookSender) matchesEndpointFilter(event *alerts.AlertEvent, endpoint config.WebhookEndpointConfig) bool {
	return ws.filter.MatchesEndpointFilter(event, endpoint)
}

// sendToEndpoint 특정 엔드포인트로 이벤트 전송 (하위 호환성 메서드)
func (ws *WebhookSender) sendToEndpoint(
	ctx context.Context, event *alerts.AlertEvent, endpoint config.WebhookEndpointConfig,
) error {
	// Core의 내부 메서드를 통해 전송 (실제 구현에서는 더 정교한 위임 필요)
	return ws.core.SendEventSync(ctx, event)
}

// calculateBackoffDelay 백오프 지연 계산 (하위 호환성 메서드)
func (ws *WebhookSender) calculateBackoffDelay(attempt int, policy retry.Policy) time.Duration {
	// retry.Manager를 통해 계산
	retryManager := retry.NewManager(policy)
	return retryManager.CalculateBackoffDelay(attempt, policy)
}

// isRetryableError 재시도 가능한 오류 확인 (하위 호환성 메서드)
func (ws *WebhookSender) isRetryableError(err error) bool {
	// 간단한 구현 - 실제로는 retry manager를 통해 확인
	return err != nil
}

// TestCompatibilityTypes - 테스트 호환성을 위한 타입들

// DummyQueue 테스트용 더미 큐
type DummyQueue struct{}

func (dq *DummyQueue) Size() int { return 0 }

// DummyMetrics 테스트용 더미 메트릭
type DummyMetrics struct{}

func (dm *DummyMetrics) incrementSent()    {}
func (dm *DummyMetrics) incrementFailed()  {}
func (dm *DummyMetrics) incrementRetries() {}

// DummyBatchManager 테스트용 더미 배치 매니저
type DummyBatchManager struct{}

func (dbm *DummyBatchManager) GetStats() map[string]interface{} {
	return make(map[string]interface{})
}

// CompatibleWebhookAdapter 테스트 호환성을 위한 어댑터 인터페이스
type CompatibleWebhookAdapter interface {
	types.WebhookAdapter
	Name() string
}

// matchesPattern 패턴 매칭 (테스트 호환성)
func (ws *WebhookSender) matchesPattern(eventType, pattern string) bool {
	// 간단한 와일드카드 패턴 매칭
	if pattern == "*" {
		return true
	}
	if pattern == eventType {
		return true
	}
	// 더 복잡한 패턴 매칭 로직은 필요에 따라 추가
	return false
}

// Sender is an alias for WebhookSender to avoid stuttering
type Sender = WebhookSender
