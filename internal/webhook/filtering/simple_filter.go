package filtering

import (
	"proxynd/alerts"
	"proxynd/configs"
)

// SimpleEventFilter 간단한 이벤트 필터링 구현체
type SimpleEventFilter struct {
	config configs.WebhookConfig
}

// NewSimpleEventFilter 새로운 간단 이벤트 필터 생성
func NewSimpleEventFilter(config configs.WebhookConfig) *SimpleEventFilter {
	return &SimpleEventFilter{
		config: config,
	}
}

// ShouldSendEvent 이벤트 전송 여부 확인 (기본 구현)
func (sef *SimpleEventFilter) ShouldSendEvent(event *alerts.AlertEvent) bool {
	if event == nil {
		return false
	}

	// 기본 검증
	if event.ID == "" {
		return false
	}

	// 이벤트 필터가 비활성화된 경우 모든 이벤트 허용
	if !sef.config.EventFilter.Enabled {
		return true
	}

	// 레벨 필터 확인
	return sef.MatchesLevelFilter(event)
}

// MatchesLevelFilter 레벨 필터 확인 (기본 구현)
func (sef *SimpleEventFilter) MatchesLevelFilter(event *alerts.AlertEvent) bool {
	if event == nil {
		return false
	}

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

	// 기본 레벨 확인
	minPriority := levelPriority[sef.config.EventFilter.DefaultLevel]
	if minPriority == 0 {
		minPriority = 1
	}

	return eventPriority >= minPriority
}

// MatchesEndpointFilter 엔드포인트별 필터 확인 (기본 구현)
func (sef *SimpleEventFilter) MatchesEndpointFilter(event *alerts.AlertEvent, endpointInterface interface{}) bool {
	endpoint, ok := endpointInterface.(configs.WebhookEndpointConfig)
	if !ok {
		return false
	}

	if event == nil {
		return false
	}

	// 레벨 필터만 확인 (간단한 구현)
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

	return true
}

// GetFilterStats 필터링 통계 반환 (기본 구현)
func (sef *SimpleEventFilter) GetFilterStats() map[string]interface{} {
	return map[string]interface{}{
		"enabled":       sef.config.EventFilter.Enabled,
		"default_level": sef.config.EventFilter.DefaultLevel,
		"type":          "simple",
	}
}
