package webhook

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"proxynd/alerts"
	"proxynd/configs"
	"proxynd/logging"
)

// Compression format constants
const (
	// CompressionFormatGzip is a const that compression format gzip
	// CompressionFormatNone is a const that compression format none
	CompressionFormatGzip = "gzip"
	CompressionFormatNone = "none"
)

// BatchGroup 배치 그룹
type BatchGroup struct {
	Key       string                        // 그룹 키 (endpoint, type, level 등)
	Events    []*alerts.AlertEvent          // 배치된 이벤트들
	Endpoint  configs.WebhookEndpointConfig // 대상 엔드포인트
	CreatedAt time.Time                     // 생성 시간
	UpdatedAt time.Time                     // 마지막 업데이트 시간
}

// BatchManager 배치 관리자
type BatchManager struct {
	config  configs.WebhookBatchingConfig
	logger  logging.Logger
	groups  map[string]*BatchGroup
	mu      sync.RWMutex
	flushCh chan string // 강제 플러시 채널
	stopCh  chan struct{}
	sender  *WebhookSender
	ticker  *time.Ticker
}

// NewBatchManager 새로운 배치 관리자 생성
func NewBatchManager(config configs.WebhookBatchingConfig, sender *WebhookSender) *BatchManager {
	logger := logging.GetLogger()

	flushInterval, err := time.ParseDuration(config.FlushInterval)
	if err != nil {
		flushInterval = 30 * time.Second
		logger.Warn("잘못된 플러시 간격, 기본값 사용", logging.F("interval", "30s"))
	}

	bm := &BatchManager{
		config:  config,
		logger:  logger,
		groups:  make(map[string]*BatchGroup),
		flushCh: make(chan string, 100),
		stopCh:  make(chan struct{}),
		sender:  sender,
		ticker:  time.NewTicker(flushInterval),
	}

	return bm
}

// Start 배치 관리자 시작
func (bm *BatchManager) Start(ctx context.Context) {
	bm.logger.Info("배치 관리자 시작됨")

	go bm.flushWorker(ctx)
	go bm.periodicFlush(ctx)
}

// Stop 배치 관리자 중지
func (bm *BatchManager) Stop(_ context.Context) error {
	bm.logger.Info("배치 관리자 중지 중...")

	if bm.ticker != nil {
		bm.ticker.Stop()
	}

	close(bm.stopCh)

	// 남은 배치들 모두 플러시
	bm.flushAll()

	bm.logger.Info("배치 관리자 중지됨")
	return nil
}

// AddEvent 이벤트를 배치에 추가
func (bm *BatchManager) AddEvent(event *alerts.AlertEvent, endpoint configs.WebhookEndpointConfig) {
	if !bm.config.Enabled {
		// 배치가 비활성화된 경우 바로 전송
		go bm.sendSingleEvent(event, endpoint)
		return
	}

	bm.mu.Lock()
	defer bm.mu.Unlock()

	groupKey := bm.generateGroupKey(event, endpoint)

	group, exists := bm.groups[groupKey]
	if !exists {
		group = &BatchGroup{
			Key:       groupKey,
			Events:    make([]*alerts.AlertEvent, 0, bm.config.MaxSize),
			Endpoint:  endpoint,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		bm.groups[groupKey] = group
	}

	group.Events = append(group.Events, event)
	group.UpdatedAt = time.Now()

	bm.logger.Debug("이벤트가 배치에 추가됨",
		logging.F("group_key", groupKey),
		logging.F("batch_size", len(group.Events)),
		logging.F("event_type", event.Type),
	)

	// 배치 크기 확인
	if len(group.Events) >= bm.config.MaxSize {
		bm.logger.Debug("배치 크기 임계값 도달, 플러시 트리거",
			logging.F("group_key", groupKey),
			logging.F("size", len(group.Events)),
		)
		select {
		case bm.flushCh <- groupKey:
		default:
			bm.logger.Warn("플러시 채널이 가득참", logging.F("group_key", groupKey))
		}
		return
	}

	// 최대 대기 시간 확인
	maxWaitTime, err := time.ParseDuration(bm.config.MaxWaitTime)
	if err != nil {
		maxWaitTime = 5 * time.Second
	}

	if time.Since(group.CreatedAt) >= maxWaitTime {
		bm.logger.Debug("최대 대기 시간 도달, 플러시 트리거",
			logging.F("group_key", groupKey),
			logging.F("wait_time", time.Since(group.CreatedAt)),
		)
		select {
		case bm.flushCh <- groupKey:
		default:
			bm.logger.Warn("플러시 채널이 가득함", logging.F("group_key", groupKey))
		}
	}
}

// generateGroupKey 그룹 키 생성
func (bm *BatchManager) generateGroupKey(event *alerts.AlertEvent, endpoint configs.WebhookEndpointConfig) string {
	switch bm.config.GroupBy {
	case "endpoint":
		return fmt.Sprintf("endpoint:%s", endpoint.Name)
	case "type":
		return fmt.Sprintf("type:%s", event.Type)
	case "level":
		return fmt.Sprintf("level:%s", event.Level)
	case "source":
		return fmt.Sprintf("source:%s", event.Source)
	default:
		return fmt.Sprintf("endpoint:%s", endpoint.Name)
	}
}

// flushWorker 플러시 워커
func (bm *BatchManager) flushWorker(ctx context.Context) {
	for {
		select {
		case groupKey := <-bm.flushCh:
			bm.flushGroup(groupKey)
		case <-bm.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// periodicFlush 주기적 플러시
func (bm *BatchManager) periodicFlush(ctx context.Context) {
	for {
		select {
		case <-bm.ticker.C:
			bm.flushExpiredGroups()
		case <-bm.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// flushGroup 특정 그룹 플러시
func (bm *BatchManager) flushGroup(groupKey string) {
	bm.mu.Lock()
	group, exists := bm.groups[groupKey]
	if !exists || len(group.Events) == 0 {
		bm.mu.Unlock()
		return
	}

	// 그룹을 맵에서 제거
	delete(bm.groups, groupKey)
	bm.mu.Unlock()

	bm.logger.Info("배치 플러시 시작",
		logging.F("group_key", groupKey),
		logging.F("event_count", len(group.Events)),
	)

	err := bm.sendBatch(group)
	if err != nil {
		bm.logger.Error("배치 전송 실패",
			logging.F("group_key", groupKey),
			logging.F("error", err),
		)

		if bm.config.RetryFailedBatch {
			bm.retryBatch(group)
		}
	} else {
		bm.logger.Info("배치 전송 성공",
			logging.F("group_key", groupKey),
			logging.F("event_count", len(group.Events)),
		)
	}
}

// flushExpiredGroups 만료된 그룹들 플러시
func (bm *BatchManager) flushExpiredGroups() {
	bm.mu.RLock()
	var expiredKeys []string

	maxWaitTime, err := time.ParseDuration(bm.config.MaxWaitTime)
	if err != nil {
		maxWaitTime = 5 * time.Second
	}

	now := time.Now()
	for key, group := range bm.groups {
		if now.Sub(group.CreatedAt) >= maxWaitTime {
			expiredKeys = append(expiredKeys, key)
		}
	}
	bm.mu.RUnlock()

	for _, key := range expiredKeys {
		bm.logger.Debug("만료된 배치 플러시", logging.F("group_key", key))
		select {
		case bm.flushCh <- key:
		default:
			bm.logger.Warn("플러시 채널이 가득함", logging.F("group_key", key))
		}
	}
}

// flushAll 모든 그룹 플러시
func (bm *BatchManager) flushAll() {
	bm.mu.RLock()
	var keys []string
	for key := range bm.groups {
		keys = append(keys, key)
	}
	bm.mu.RUnlock()

	bm.logger.Info("모든 배치 플러시", logging.F("group_count", len(keys)))

	for _, key := range keys {
		bm.flushGroup(key)
	}
}

// sendBatch 배치 전송
func (bm *BatchManager) sendBatch(group *BatchGroup) error {
	// 배치 페이로드 생성
	payload := bm.createBatchPayload(group)

	// 압축
	_, err := bm.compressPayload(payload)
	if err != nil {
		return fmt.Errorf("payload compression failed: %w", err)
	}

	// 어댑터를 통해 전송
	adapter, exists := bm.sender.adapters[bm.getAdapterType(group.Endpoint.Format)]
	if !exists {
		adapter = bm.sender.adapters[webhookTypeGeneric]
	}

	// 임시 엔드포인트 설정 (배치용)
	batchEndpoint := group.Endpoint
	batchEndpoint.Headers = bm.setBatchHeaders(batchEndpoint.Headers)

	// 배치 이벤트로 변환하여 전송
	batchEvent := bm.createBatchEvent(group)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	return adapter.Send(ctx, batchEndpoint.URL, batchEvent)
}

// createBatchPayload 배치 페이로드 생성
func (bm *BatchManager) createBatchPayload(group *BatchGroup) map[string]interface{} {
	return map[string]interface{}{
		"batch_id":    fmt.Sprintf("batch_%d_%s", time.Now().Unix(), group.Key),
		"event_count": len(group.Events),
		"created_at":  group.CreatedAt.Unix(),
		"flushed_at":  time.Now().Unix(),
		"group_key":   group.Key,
		"endpoint":    group.Endpoint.Name,
		"events":      group.Events,
		"compression": bm.config.CompressionFormat,
		"batch_type":  "webhook_events",
		"source":      "proxynd-webhook-system",
	}
}

// createBatchEvent 배치 이벤트 생성 (단일 이벤트처럼 보이게)
func (bm *BatchManager) createBatchEvent(group *BatchGroup) *alerts.AlertEvent {
	return &alerts.AlertEvent{
		ID:        fmt.Sprintf("batch_%d_%s", time.Now().Unix(), group.Key),
		Type:      "webhook.batch",
		Level:     alerts.AlertLevelInfo,
		Title:     fmt.Sprintf("Webhook Batch (%d events)", len(group.Events)),
		Message:   fmt.Sprintf("Batch containing %d events", len(group.Events)),
		Source:    "webhook-batch-manager",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"batch_data": bm.createBatchPayload(group),
		},
	}
}

// compressPayload 페이로드 압축
func (bm *BatchManager) compressPayload(payload map[string]interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	switch bm.config.CompressionFormat {
	case CompressionFormatGzip:
		var buf bytes.Buffer
		writer := gzip.NewWriter(&buf)
		_, err := writer.Write(jsonData)
		if err != nil {
			return nil, err
		}
		err = writer.Close()
		if err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	case CompressionFormatNone, "":
		return jsonData, nil
	default:
		bm.logger.Warn("지원하지 않는 압축 포맷", logging.F("format", bm.config.CompressionFormat))
		return jsonData, nil
	}
}

// setBatchHeaders 배치용 헤더 설정
func (bm *BatchManager) setBatchHeaders(headers map[string]string) map[string]string {
	if headers == nil {
		headers = make(map[string]string)
	}

	headers["X-ProxyND-Batch"] = "true"
	headers["X-ProxyND-Batch-Format"] = "webhook_events"

	if bm.config.CompressionFormat == CompressionFormatGzip {
		headers["Content-Encoding"] = "gzip"
	}

	return headers
}

// getAdapterType 어댑터 타입 결정
func (bm *BatchManager) getAdapterType(format string) string {
	if strings.HasPrefix(format, "slack") {
		return "slack"
	}
	if strings.HasPrefix(format, "discord") {
		return "discord"
	}
	return webhookTypeGeneric
}

// retryBatch 실패한 배치 재시도
func (bm *BatchManager) retryBatch(group *BatchGroup) {
	bm.logger.Info("배치 재시도 예약",
		logging.F("group_key", group.Key),
		logging.F("event_count", len(group.Events)),
	)

	// 재시도는 개별 이벤트로 분해하여 처리
	for _, event := range group.Events {
		if err := bm.sender.SendEvent(event); err != nil {
			bm.logger.Warn("Failed to resend individual event",
				logging.F("event_id", event.ID),
				logging.F("error", err))
		}
	}
}

// sendSingleEvent 단일 이벤트 전송 (배치 비활성화 시)
func (bm *BatchManager) sendSingleEvent(event *alerts.AlertEvent, endpoint configs.WebhookEndpointConfig) {
	adapter, exists := bm.sender.adapters[bm.getAdapterType(endpoint.Format)]
	if !exists {
		adapter = bm.sender.adapters[webhookTypeGeneric]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := adapter.Send(ctx, endpoint.URL, event)
	if err != nil {
		bm.logger.Error("단일 이벤트 전송 실패",
			logging.F("event_id", event.ID),
			logging.F("endpoint", endpoint.Name),
			logging.F("error", err),
		)
	}
}

// GetStats 배치 관리자 통계
func (bm *BatchManager) GetStats() map[string]interface{} {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	stats := map[string]interface{}{
		"enabled":       bm.config.Enabled,
		"active_groups": len(bm.groups),
		"total_events":  0,
		"groups":        make(map[string]interface{}),
	}

	totalEvents := 0
	for key, group := range bm.groups {
		eventCount := len(group.Events)
		totalEvents += eventCount

		groupsMap, ok := stats["groups"].(map[string]interface{})
		if !ok {
			bm.logger.Warn("Failed to type assert groups map")
			continue
		}
		groupsMap[key] = map[string]interface{}{
			"event_count": eventCount,
			"created_at":  group.CreatedAt.Unix(),
			"updated_at":  group.UpdatedAt.Unix(),
			"age_seconds": time.Since(group.CreatedAt).Seconds(),
		}
	}

	stats["total_events"] = totalEvents
	return stats
}
