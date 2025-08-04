package webhook

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-playground/assert/v2"

	"proxynd/alerts"
	configpkg "proxynd/internal/config"
	"proxynd/internal/webhook/retry"
)

// RetryPolicy is an alias for retry.Policy for test compatibility
type RetryPolicy = retry.Policy

// TestNewWebhookSender 웹훅 전송기 생성 테스트
func TestNewWebhookSender(t *testing.T) {
	webhookConfig := configpkg.GetDefaultWebhookConfig()
	webhookConfig.Enabled = true

	sender, err := NewWebhookSender(webhookConfig)
	assert.Equal(t, err, nil)
	assert.NotEqual(t, sender, nil)
	assert.Equal(t, sender.config.Enabled, true)
	assert.Equal(t, len(sender.adapters), 3) // generic, slack, discord adapters
	assert.NotEqual(t, sender.queue, nil)
	assert.NotEqual(t, sender.rateLimiter, nil)
	assert.NotEqual(t, sender.batchManager, nil)
}

// TestWebhookSenderStartStop 시작/중지 테스트
func TestWebhookSenderStartStop(t *testing.T) {
	webhookConfig := configpkg.GetDefaultWebhookConfig()
	webhookConfig.Enabled = true

	sender, err := NewWebhookSender(webhookConfig)
	assert.Equal(t, err, nil)

	ctx := context.Background()

	// 시작
	err = sender.Start(ctx)
	assert.Equal(t, err, nil)
	assert.Equal(t, sender.started, true)

	// 이미 시작된 상태에서 다시 시작 시도
	err = sender.Start(ctx)
	assert.NotEqual(t, err, nil)

	// 중지
	stopCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	err = sender.Stop(stopCtx)
	assert.Equal(t, err, nil)
	assert.Equal(t, sender.started, false)
}

// TestSendEvent 이벤트 전송 테스트
func TestSendEvent(t *testing.T) {
	webhookConfig := configpkg.GetDefaultWebhookConfig()
	webhookConfig.Enabled = true
	webhookConfig.Batching.Enabled = false // 배치 비활성화하여 큐 테스트

	sender, err := NewWebhookSender(webhookConfig)
	assert.Equal(t, err, nil)

	ctx := context.Background()
	err = sender.Start(ctx)
	assert.Equal(t, err, nil)
	defer func() { _ = sender.Stop(ctx) }()

	// 테스트 이벤트 생성
	event := alerts.CreateWebhookEvent(
		alerts.EventCacheExpiry,
		alerts.AlertLevelInfo,
		"캐시 만료",
		"테스트 캐시가 만료되었습니다",
	)

	// 이벤트 전송
	err = sender.SendEvent(event)
	assert.Equal(t, err, nil)

	// 큐에 추가되었는지 확인
	assert.Equal(t, sender.queue.Size() > 0, true)
}

// TestEventFiltering 이벤트 필터링 테스트
func TestEventFiltering(t *testing.T) {
	webhookConfig := configpkg.GetDefaultWebhookConfig()
	webhookConfig.Enabled = true
	webhookConfig.EventFilter.Enabled = true
	webhookConfig.EventFilter.DefaultLevel = "WARNING"

	sender, err := NewWebhookSender(webhookConfig)
	assert.Equal(t, err, nil)

	// INFO 레벨 이벤트 (필터링되어야 함)
	infoEvent := alerts.CreateWebhookEvent(
		alerts.EventCacheExpiry,
		alerts.AlertLevelInfo,
		"정보",
		"정보 메시지",
	)

	shouldSend := sender.shouldSendEvent(infoEvent)
	assert.Equal(t, shouldSend, false)

	// WARNING 레벨 이벤트 (전송되어야 함)
	warningEvent := alerts.CreateWebhookEvent(
		alerts.EventAuthFailure,
		alerts.AlertLevelWarning,
		"경고",
		"경고 메시지",
	)

	shouldSend = sender.shouldSendEvent(warningEvent)
	assert.Equal(t, shouldSend, true)
}

// TestLevelOverrides 레벨 오버라이드 테스트
func TestLevelOverrides(t *testing.T) {
	webhookConfig := configpkg.GetDefaultWebhookConfig()
	webhookConfig.Enabled = true
	webhookConfig.EventFilter.Enabled = true
	webhookConfig.EventFilter.DefaultLevel = "ERROR"
	webhookConfig.EventFilter.LevelOverrides = map[string]string{
		"security.*": "INFO", // 보안 이벤트는 INFO부터 허용
	}

	sender, err := NewWebhookSender(webhookConfig)
	assert.Equal(t, err, nil)

	// 일반 INFO 이벤트 (필터링되어야 함)
	infoEvent := alerts.CreateWebhookEvent(
		alerts.EventCacheExpiry,
		alerts.AlertLevelInfo,
		"정보",
		"정보 메시지",
	)

	shouldSend := sender.shouldSendEvent(infoEvent)
	assert.Equal(t, shouldSend, false)

	// 보안 INFO 이벤트 (오버라이드로 허용되어야 함)
	securityEvent := alerts.CreateWebhookEvent(
		alerts.EventSecurityThreat,
		alerts.AlertLevelInfo,
		"보안 위협",
		"보안 위협 감지",
	)

	shouldSend = sender.shouldSendEvent(securityEvent)
	assert.Equal(t, shouldSend, true)
}

// TestEndpointFiltering 엔드포인트별 필터링 테스트
func TestEndpointFiltering(t *testing.T) {
	webhookConfig := configpkg.GetDefaultWebhookConfig()
	endpoint := configpkg.WebhookEndpointConfig{
		Name:       "security-only",
		Enabled:    true,
		EventTypes: []string{"security.*"},
		Filters: configpkg.WebhookEndpointFilters{
			MinLevel: "WARNING",
		},
	}

	sender, err := NewWebhookSender(webhookConfig)
	assert.Equal(t, err, nil)

	// 보안 WARNING 이벤트 (매칭되어야 함)
	securityEvent := alerts.CreateWebhookEvent(
		alerts.EventSecurityThreat,
		alerts.AlertLevelWarning,
		"보안 위협",
		"보안 위협 감지",
	)

	matches := sender.matchesEndpointFilter(securityEvent, endpoint)
	assert.Equal(t, matches, true)

	// 캐시 이벤트 (매칭되지 않아야 함)
	cacheEvent := alerts.CreateWebhookEvent(
		alerts.EventCacheExpiry,
		alerts.AlertLevelWarning,
		"캐시 만료",
		"캐시가 만료되었습니다",
	)

	matches = sender.matchesEndpointFilter(cacheEvent, endpoint)
	assert.Equal(t, matches, false)

	// 보안 INFO 이벤트 (레벨이 낮아서 매칭되지 않아야 함)
	lowSecurityEvent := alerts.CreateWebhookEvent(
		alerts.EventSecurityThreat,
		alerts.AlertLevelInfo,
		"보안 정보",
		"보안 정보",
	)

	matches = sender.matchesEndpointFilter(lowSecurityEvent, endpoint)
	assert.Equal(t, matches, false)
}

// TestPatternMatching 패턴 매칭 테스트
func TestPatternMatching(t *testing.T) {
	sender := &WebhookSender{}

	// 와일드카드 매칭
	assert.Equal(t, sender.matchesPattern("security.threat", "security.*"), true)
	assert.Equal(t, sender.matchesPattern("security.malware", "security.*"), true)
	assert.Equal(t, sender.matchesPattern("cache.expiry", "security.*"), false)

	// 전체 매칭
	assert.Equal(t, sender.matchesPattern("anything", "*"), true)

	// 정확한 매칭
	assert.Equal(t, sender.matchesPattern("auth.failure", "auth.failure"), true)
	assert.Equal(t, sender.matchesPattern("auth.success", "auth.failure"), false)
}

// TestAdapterRegistration 어댑터 등록 테스트
func TestAdapterRegistration(t *testing.T) {
	webhookConfig := configpkg.GetDefaultWebhookConfig()
	sender, err := NewWebhookSender(webhookConfig)
	assert.Equal(t, err, nil)

	// 초기 어댑터 확인 (generic, slack, discord)
	assert.Equal(t, len(sender.adapters), 3)

	// 어댑터들이 등록되어 있는지 확인
	assert.NotEqual(t, sender.adapters["generic"], nil)
	assert.NotEqual(t, sender.adapters["slack"], nil)
	assert.NotEqual(t, sender.adapters["discord"], nil)

	assert.Equal(t, sender.adapters["generic"].Name(), "generic")
	assert.Equal(t, sender.adapters["slack"].Name(), "slack")
	assert.Equal(t, sender.adapters["discord"].Name(), "discord")
}

// TestMetrics 메트릭 테스트
func TestMetrics(t *testing.T) {
	webhookConfig := configpkg.GetDefaultWebhookConfig()
	sender, err := NewWebhookSender(webhookConfig)
	assert.Equal(t, err, nil)

	// 초기 메트릭
	metrics := sender.GetMetrics()
	assert.Equal(t, metrics.TotalSent, int64(0))
	assert.Equal(t, metrics.TotalFailed, int64(0))
	assert.Equal(t, metrics.TotalRetries, int64(0))

	// 메트릭 증가
	sender.metrics.incrementSent()
	sender.metrics.incrementFailed()
	sender.metrics.incrementRetries()

	metrics = sender.GetMetrics()
	assert.Equal(t, metrics.TotalSent, int64(1))
	assert.Equal(t, metrics.TotalFailed, int64(1))
	assert.Equal(t, metrics.TotalRetries, int64(1))
}

// TestBackoffDelay 백오프 지연 계산 테스트
func TestBackoffDelay(t *testing.T) {
	sender := &WebhookSender{}

	policy := RetryPolicy{
		InitialDelay:  time.Second,
		MaxDelay:      time.Minute,
		BackoffFactor: 2.0,
	}

	// 첫 번째 재시도
	delay1 := sender.calculateBackoffDelay(1, policy)
	assert.Equal(t, delay1, time.Second)

	// 두 번째 재시도
	delay2 := sender.calculateBackoffDelay(2, policy)
	assert.Equal(t, delay2, time.Second*2)

	// 세 번째 재시도
	delay3 := sender.calculateBackoffDelay(3, policy)
	assert.Equal(t, delay3, time.Second*4)

	// 최대 지연 시간 확인
	delay10 := sender.calculateBackoffDelay(10, policy)
	assert.Equal(t, delay10, time.Minute) // 최대 지연 시간으로 제한
}

// TestWebhookSenderBatching 배치 전송 테스트
func TestWebhookSenderBatching(t *testing.T) {
	webhookConfig := configpkg.GetDefaultWebhookConfig()
	webhookConfig.Enabled = true
	webhookConfig.Batching.Enabled = true
	webhookConfig.Batching.MaxSize = 3
	webhookConfig.Batching.MaxWaitTime = "1s"

	sender, err := NewWebhookSender(webhookConfig)
	assert.Equal(t, err, nil)

	ctx := context.Background()
	err = sender.Start(ctx)
	assert.Equal(t, err, nil)
	defer func() { _ = sender.Stop(ctx) }()

	// 테스트 이벤트들 전송
	for i := 0; i < 5; i++ {
		event := alerts.CreateWebhookEvent(
			alerts.EventCacheExpiry,
			alerts.AlertLevelInfo,
			fmt.Sprintf("테스트 %d", i+1),
			fmt.Sprintf("테스트 메시지 %d", i+1),
		)

		err = sender.SendEvent(event)
		assert.Equal(t, err, nil)
	}

	// 배치 관리자 통계 확인
	stats := sender.batchManager.GetStats()
	assert.Equal(t, stats["enabled"], true)

	// 잠시 대기하여 배치 처리 시간 확보
	time.Sleep(100 * time.Millisecond)
}

// TestWebhookSenderGetMetricsWithBatch 배치 통계가 포함된 메트릭 테스트
func TestWebhookSenderGetMetricsWithBatch(t *testing.T) {
	webhookConfig := configpkg.GetDefaultWebhookConfig()
	webhookConfig.Enabled = true
	webhookConfig.Batching.Enabled = true

	sender, err := NewWebhookSender(webhookConfig)
	assert.Equal(t, err, nil)

	metrics := sender.GetMetrics()
	assert.NotEqual(t, metrics.BatchStats, nil)
	assert.Equal(t, metrics.BatchStats["enabled"], true)
}

// TestWebhookSenderBatchingDisabled 배치 비활성화 테스트
func TestWebhookSenderBatchingDisabled(t *testing.T) {
	webhookConfig := configpkg.GetDefaultWebhookConfig()
	webhookConfig.Enabled = true
	webhookConfig.Batching.Enabled = false

	sender, err := NewWebhookSender(webhookConfig)
	assert.Equal(t, err, nil)

	metrics := sender.GetMetrics()
	assert.NotEqual(t, metrics.BatchStats, nil)
	assert.Equal(t, metrics.BatchStats["enabled"], false)
}
