package webhook

import (
	"context"
	"testing"
	"time"

	"github.com/go-playground/assert/v2"

	"proxynd/alerts"
	"proxynd/configs"
)

// TestNewBatchManager 배치 관리자 생성 테스트
func TestNewBatchManager(t *testing.T) {
	config := configs.WebhookBatchingConfig{
		Enabled:           true,
		MaxSize:           5,
		MaxWaitTime:       "2s",
		FlushInterval:     "10s",
		GroupBy:           "endpoint",
		CompressionFormat: "gzip",
		RetryFailedBatch:  true,
	}

	sender := &WebhookSender{}
	bm := NewBatchManager(config, sender)

	assert.NotEqual(t, bm, nil)
	assert.Equal(t, bm.config.Enabled, true)
	assert.Equal(t, bm.config.MaxSize, 5)
	assert.NotEqual(t, bm.groups, nil)
	assert.NotEqual(t, bm.flushCh, nil)
}

// TestBatchManagerAddEvent 이벤트 추가 테스트
func TestBatchManagerAddEvent(t *testing.T) {
	config := configs.WebhookBatchingConfig{
		Enabled:           true,
		MaxSize:           3,
		MaxWaitTime:       "5s",
		FlushInterval:     "10s",
		GroupBy:           "endpoint",
		CompressionFormat: "none",
		RetryFailedBatch:  false,
	}

	sender := &WebhookSender{
		adapters: make(map[string]CompatibleWebhookAdapter),
	}
	sender.adapters["generic"] = NewGenericWebhookAdapter()

	bm := NewBatchManager(config, sender)

	endpoint := configs.WebhookEndpointConfig{
		Name:    "test-endpoint",
		URL:     "https://example.com/webhook",
		Enabled: true,
		Format:  "json",
	}

	// 첫 번째 이벤트 추가
	event1 := alerts.CreateWebhookEvent(
		alerts.EventCacheExpiry,
		alerts.AlertLevelInfo,
		"테스트 1",
		"테스트 메시지 1",
	)

	bm.AddEvent(event1, endpoint)

	// 배치 그룹이 생성되었는지 확인
	bm.mu.RLock()
	assert.Equal(t, len(bm.groups), 1)

	groupKey := "endpoint:test-endpoint"
	group, exists := bm.groups[groupKey]
	assert.Equal(t, exists, true)
	assert.Equal(t, len(group.Events), 1)
	assert.Equal(t, group.Events[0].ID, event1.ID)
	bm.mu.RUnlock()

	// 두 번째 이벤트 추가
	event2 := alerts.CreateWebhookEvent(
		alerts.EventAuthFailure,
		alerts.AlertLevelWarning,
		"테스트 2",
		"테스트 메시지 2",
	)

	bm.AddEvent(event2, endpoint)

	// 배치에 추가되었는지 확인
	bm.mu.RLock()
	group = bm.groups[groupKey]
	assert.Equal(t, len(group.Events), 2)
	bm.mu.RUnlock()
}

// TestBatchManagerGroupKeyGeneration 그룹 키 생성 테스트
func TestBatchManagerGroupKeyGeneration(t *testing.T) {
	sender := &WebhookSender{}

	event := alerts.CreateWebhookEvent(
		alerts.EventSecurityThreat,
		alerts.AlertLevelError,
		"보안 테스트",
		"보안 테스트 메시지",
	)

	endpoint := configs.WebhookEndpointConfig{
		Name: "security-endpoint",
	}

	// endpoint 그룹화 테스트
	config1 := configs.WebhookBatchingConfig{GroupBy: "endpoint"}
	bm1 := NewBatchManager(config1, sender)
	key1 := bm1.generateGroupKey(event, endpoint)
	assert.Equal(t, key1, "endpoint:security-endpoint")

	// type 그룹화 테스트
	config2 := configs.WebhookBatchingConfig{GroupBy: "type"}
	bm2 := NewBatchManager(config2, sender)
	key2 := bm2.generateGroupKey(event, endpoint)
	assert.Equal(t, key2, "type:security.threat")

	// level 그룹화 테스트
	config3 := configs.WebhookBatchingConfig{GroupBy: "level"}
	bm3 := NewBatchManager(config3, sender)
	key3 := bm3.generateGroupKey(event, endpoint)
	assert.Equal(t, key3, "level:ERROR")

	// source 그룹화 테스트
	config4 := configs.WebhookBatchingConfig{GroupBy: "source"}
	bm4 := NewBatchManager(config4, sender)
	key4 := bm4.generateGroupKey(event, endpoint)
	assert.Equal(t, key4, "source:"+event.Source)
}

// TestBatchManagerMaxSizeFlush 최대 크기 플러시 테스트
func TestBatchManagerMaxSizeFlush(t *testing.T) {
	config := configs.WebhookBatchingConfig{
		Enabled:           true,
		MaxSize:           2, // 작은 크기로 설정
		MaxWaitTime:       "10s",
		FlushInterval:     "30s",
		GroupBy:           "endpoint",
		CompressionFormat: "none",
		RetryFailedBatch:  false,
	}

	sender := &WebhookSender{
		adapters: make(map[string]CompatibleWebhookAdapter),
	}
	sender.adapters["generic"] = NewGenericWebhookAdapter()

	bm := NewBatchManager(config, sender)

	endpoint := configs.WebhookEndpointConfig{
		Name:    "test-endpoint",
		URL:     "https://example.com/webhook",
		Enabled: true,
		Format:  "json",
	}

	// 첫 번째 이벤트 추가
	event1 := alerts.CreateWebhookEvent(
		alerts.EventCacheExpiry,
		alerts.AlertLevelInfo,
		"테스트 1",
		"테스트 메시지 1",
	)
	bm.AddEvent(event1, endpoint)

	// 두 번째 이벤트 추가 (최대 크기 도달)
	event2 := alerts.CreateWebhookEvent(
		alerts.EventAuthFailure,
		alerts.AlertLevelWarning,
		"테스트 2",
		"테스트 메시지 2",
	)
	bm.AddEvent(event2, endpoint)

	// 플러시 채널에 신호가 전송되었는지 확인
	select {
	case groupKey := <-bm.flushCh:
		assert.Equal(t, groupKey, "endpoint:test-endpoint")
	case <-time.After(time.Second):
		t.Fatal("플러시 신호가 전송되지 않았습니다")
	}
}

// TestBatchManagerCreateBatchPayload 배치 페이로드 생성 테스트
func TestBatchManagerCreateBatchPayload(t *testing.T) {
	config := configs.WebhookBatchingConfig{
		GroupBy: "endpoint",
	}

	sender := &WebhookSender{}
	bm := NewBatchManager(config, sender)

	endpoint := configs.WebhookEndpointConfig{
		Name: "test-endpoint",
	}

	events := []*alerts.AlertEvent{
		alerts.CreateWebhookEvent(
			alerts.EventCacheExpiry,
			alerts.AlertLevelInfo,
			"테스트 1",
			"테스트 메시지 1",
		),
		alerts.CreateWebhookEvent(
			alerts.EventAuthFailure,
			alerts.AlertLevelWarning,
			"테스트 2",
			"테스트 메시지 2",
		),
	}

	group := &BatchGroup{
		Key:       "endpoint:test-endpoint",
		Events:    events,
		Endpoint:  endpoint,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	payload := bm.createBatchPayload(group)

	assert.NotEqual(t, payload["batch_id"], nil)
	assert.Equal(t, payload["event_count"], 2)
	assert.Equal(t, payload["group_key"], "endpoint:test-endpoint")
	assert.Equal(t, payload["endpoint"], "test-endpoint")
	assert.Equal(t, payload["events"], events)
	assert.Equal(t, payload["batch_type"], "webhook_events")
	assert.Equal(t, payload["source"], "proxynd-webhook-system")
}

// TestBatchManagerGetStats 통계 반환 테스트
func TestBatchManagerGetStats(t *testing.T) {
	config := configs.WebhookBatchingConfig{
		Enabled: true,
		GroupBy: "endpoint",
	}

	sender := &WebhookSender{}
	bm := NewBatchManager(config, sender)

	// 초기 통계
	stats := bm.GetStats()
	assert.Equal(t, stats["enabled"], true)
	assert.Equal(t, stats["active_groups"], 0)
	assert.Equal(t, stats["total_events"], 0)

	// 이벤트 추가
	endpoint := configs.WebhookEndpointConfig{
		Name: "test-endpoint",
	}

	event := alerts.CreateWebhookEvent(
		alerts.EventCacheExpiry,
		alerts.AlertLevelInfo,
		"테스트",
		"테스트 메시지",
	)

	bm.AddEvent(event, endpoint)

	// 통계 업데이트 확인
	stats = bm.GetStats()
	assert.Equal(t, stats["active_groups"], 1)
	assert.Equal(t, stats["total_events"], 1)

	groups, ok := stats["groups"].(map[string]interface{})
	assert.Equal(t, ok, true)
	assert.Equal(t, len(groups), 1)

	groupStats, exists := groups["endpoint:test-endpoint"]
	assert.Equal(t, exists, true)

	groupData, ok := groupStats.(map[string]interface{})
	assert.Equal(t, ok, true)
	assert.Equal(t, groupData["event_count"], 1)
}

// TestBatchManagerDisabled 배치 비활성화 테스트
func TestBatchManagerDisabled(t *testing.T) {
	config := configs.WebhookBatchingConfig{
		Enabled: false,
	}

	sender := &WebhookSender{
		adapters: make(map[string]CompatibleWebhookAdapter),
	}
	sender.adapters["generic"] = NewGenericWebhookAdapter()

	bm := NewBatchManager(config, sender)

	endpoint := configs.WebhookEndpointConfig{
		Name:    "test-endpoint",
		URL:     "https://example.com/webhook",
		Enabled: true,
		Format:  "json",
	}

	event := alerts.CreateWebhookEvent(
		alerts.EventCacheExpiry,
		alerts.AlertLevelInfo,
		"테스트",
		"테스트 메시지",
	)

	// 배치가 비활성화된 경우 그룹이 생성되지 않아야 함
	bm.AddEvent(event, endpoint)

	bm.mu.RLock()
	assert.Equal(t, len(bm.groups), 0)
	bm.mu.RUnlock()
}

// TestBatchManagerStartStop 시작/중지 테스트
func TestBatchManagerStartStop(t *testing.T) {
	config := configs.WebhookBatchingConfig{
		Enabled:       true,
		FlushInterval: "100ms", // 빠른 테스트를 위해 짧게 설정
	}

	sender := &WebhookSender{}
	bm := NewBatchManager(config, sender)

	ctx := context.Background()

	// 시작
	bm.Start(ctx)

	// 잠시 실행
	time.Sleep(200 * time.Millisecond)

	// 중지
	err := bm.Stop(ctx)
	assert.Equal(t, err, nil)
}
