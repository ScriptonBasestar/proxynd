package webhook

import (
	"context"
	"fmt"

	"proxynd/alerts"
	"proxynd/configs"
	"proxynd/internal/webhook/filtering"
	"proxynd/internal/webhook/sender"
	"proxynd/internal/webhook/types"
)

// WebhookSender 웹훅 전송기 (리팩토링된 버전)
// 기존 인터페이스를 유지하면서 내부적으로 새로운 모듈들을 사용
type WebhookSender struct {
	core   *sender.Core
	filter *filtering.SimpleEventFilter
}

// NewWebhookSender 새로운 웹훅 전송기 생성
func NewWebhookSender(config configs.WebhookConfig) (*WebhookSender, error) {
	// 새로운 core 생성
	core, err := sender.NewCore(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create sender core: %w", err)
	}

	// 필터 생성
	filter := filtering.NewSimpleEventFilter(config)

	return &WebhookSender{
		core:   core,
		filter: filter,
	}, nil
}

// RegisterAdapter 웹훅 어댑터 등록
func (ws *WebhookSender) RegisterAdapter(adapter types.WebhookAdapter) {
	ws.core.RegisterAdapter(adapter)
}

// Start 웹훅 전송기 시작
func (ws *WebhookSender) Start(ctx context.Context) error {
	return ws.core.Start(ctx)
}

// Stop 웹훅 전송기 중지
func (ws *WebhookSender) Stop(ctx context.Context) error {
	return ws.core.Stop(ctx)
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
	return ws.core.GetHistoryManager()
}

// shouldSendEvent 이벤트 전송 여부 확인 (하위 호환성을 위한 메서드)
func (ws *WebhookSender) shouldSendEvent(event *alerts.AlertEvent) bool {
	return ws.filter.ShouldSendEvent(event)
}

// matchesLevelFilter 레벨 필터 확인 (하위 호환성을 위한 메서드)
func (ws *WebhookSender) matchesLevelFilter(event *alerts.AlertEvent) bool {
	return ws.filter.MatchesLevelFilter(event)
}

// matchesEndpointFilter 엔드포인트별 필터 확인 (하위 호환성을 위한 메서드)
func (ws *WebhookSender) matchesEndpointFilter(event *alerts.AlertEvent, endpoint configs.WebhookEndpointConfig) bool {
	return ws.filter.MatchesEndpointFilter(event, endpoint)
}

// Sender is an alias for WebhookSender to avoid stuttering
type Sender = WebhookSender