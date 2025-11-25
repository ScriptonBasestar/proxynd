package filtering

import (
	"proxynd/internal/alerts"
	"proxynd/internal/config"
)

// SimpleEventFilter 간단한 이벤트 필터링 구현체
type SimpleEventFilter struct {
	config config.WebhookConfig
}

// NewSimpleEventFilter 새로운 간단 이벤트 필터 생성
func NewSimpleEventFilter(config config.WebhookConfig) *SimpleEventFilter {
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

	// 레벨 오버라이드 확인
	levelPriority := map[string]int{
		"INFO": 1, "WARNING": 2, "ERROR": 3, "CRITICAL": 4,
	}

	eventPriority := levelPriority[string(event.Level)]
	if eventPriority == 0 {
		eventPriority = 1 // 기본값
	}

	// 레벨 오버라이드 매칭 확인
	for pattern, overrideLevel := range sef.config.EventFilter.LevelOverrides {
		if sef.matchesPattern(string(event.Type), pattern) {
			minPriority := levelPriority[overrideLevel]
			if minPriority == 0 {
				minPriority = 1
			}
			return eventPriority >= minPriority
		}
	}

	// 기본 레벨 필터 확인
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
	endpoint, ok := endpointInterface.(config.WebhookEndpointConfig)
	if !ok {
		return false
	}

	if event == nil {
		return false
	}

	// 이벤트 타입 필터 확인
	if len(endpoint.EventTypes) > 0 {
		eventTypeMatched := false
		for _, pattern := range endpoint.EventTypes {
			if sef.matchesPattern(string(event.Type), pattern) {
				eventTypeMatched = true
				break
			}
		}
		if !eventTypeMatched {
			return false
		}
	}

	// 레벨 필터 확인
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

// matchesPattern 패턴 매칭 헬퍼
func (sef *SimpleEventFilter) matchesPattern(eventType, pattern string) bool {
	// 전체 와일드카드
	if pattern == "*" {
		return true
	}
	// 정확한 매칭
	if pattern == eventType {
		return true
	}
	// 접미사 와일드카드 패턴 (예: "security.*")
	if len(pattern) > 2 && pattern[len(pattern)-2:] == ".*" {
		prefix := pattern[:len(pattern)-2]
		if len(eventType) > len(prefix) && eventType[:len(prefix)] == prefix && eventType[len(prefix)] == '.' {
			return true
		}
	}
	return false
}

// GetFilterStats 필터링 통계 반환 (기본 구현)
func (sef *SimpleEventFilter) GetFilterStats() map[string]interface{} {
	return map[string]interface{}{
		"enabled":       sef.config.EventFilter.Enabled,
		"default_level": sef.config.EventFilter.DefaultLevel,
		"type":          "simple",
	}
}
