package alerts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"proxynd/pkg/httpclient"
)

// WebhookAlerter 웹훅 기반 알림 구현
type WebhookAlerter struct {
	enabled       bool
	url           string
	method        string
	headers       map[string]string
	timeout       time.Duration
	retryCount    int
	retryInterval time.Duration
	client        *http.Client
}

// WebhookAlerterConfig 웹훅 알림 설정
type WebhookAlerterConfig struct {
	Enabled       bool              `json:"enabled"`
	URL           string            `json:"url"`
	Method        string            `json:"method"`
	Headers       map[string]string `json:"headers"`
	Timeout       string            `json:"timeout"`
	RetryCount    int               `json:"retry_count"`
	RetryInterval string            `json:"retry_interval"`
}

// NewWebhookAlerter 새 웹훅 알림 생성
func NewWebhookAlerter(config WebhookAlerterConfig) (*WebhookAlerter, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("webhook URL is required")
	}

	// 기본값 설정
	if config.Method == "" {
		config.Method = "POST"
	}
	if config.RetryCount <= 0 {
		config.RetryCount = 3
	}

	// 타임아웃 파싱
	timeout := 30 * time.Second
	if config.Timeout != "" {
		if d, err := time.ParseDuration(config.Timeout); err == nil {
			timeout = d
		}
	}

	// 재시도 간격 파싱
	retryInterval := 5 * time.Second
	if config.RetryInterval != "" {
		if d, err := time.ParseDuration(config.RetryInterval); err == nil {
			retryInterval = d
		}
	}

	// HTTP 클라이언트 설정
	clientConfig := httpclient.Config{
		Timeout:          timeout,
		ConnectTimeout:   10 * time.Second,
		KeepAliveTimeout: 30 * time.Second,
		MaxIdleConns:     10,
	}

	wa := &WebhookAlerter{
		enabled:       config.Enabled,
		url:           config.URL,
		method:        config.Method,
		headers:       config.Headers,
		timeout:       timeout,
		retryCount:    config.RetryCount,
		retryInterval: retryInterval,
		client:        httpclient.New(clientConfig),
	}

	return wa, nil
}

// Send 알림 전송
func (wa *WebhookAlerter) Send(ctx context.Context, event *AlertEvent) error {
	// 이벤트를 웹훅 페이로드로 변환
	payload, err := wa.createPayload(event)
	if err != nil {
		return fmt.Errorf("failed to create payload: %w", err)
	}

	// 재시도 로직과 함께 전송
	var lastErr error
	for i := 0; i <= wa.retryCount; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wa.retryInterval):
			}
		}

		if err := wa.sendRequest(ctx, payload); err != nil {
			lastErr = err
			continue
		}

		return nil
	}

	return fmt.Errorf("failed after %d retries: %w", wa.retryCount, lastErr)
}

// createPayload 웹훅 페이로드 생성
func (wa *WebhookAlerter) createPayload(event *AlertEvent) ([]byte, error) {
	// 웹훅 페이로드 구조
	payload := map[string]interface{}{
		"id":        event.ID,
		"level":     event.Level,
		"type":      event.Type,
		"title":     event.Title,
		"message":   event.Message,
		"source":    event.Source,
		"timestamp": event.Timestamp.Unix(),
		"metadata":  event.Metadata,
	}

	// 패키지 정보 추가
	if event.PackageInfo != nil {
		payload["package"] = map[string]interface{}{
			"type":          event.PackageInfo.Type,
			"name":          event.PackageInfo.Name,
			"version":       event.PackageInfo.Version,
			"path":          event.PackageInfo.Path,
			"expected_hash": event.PackageInfo.ExpectedHash,
			"actual_hash":   event.PackageInfo.ActualHash,
			"remote_url":    event.PackageInfo.RemoteURL,
		}
	}

	return json.Marshal(payload)
}

// sendRequest HTTP 요청 전송
func (wa *WebhookAlerter) sendRequest(ctx context.Context, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, wa.method, wa.url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 헤더 설정
	req.Header.Set("Content-Type", "application/json")
	for k, v := range wa.headers {
		req.Header.Set(k, v)
	}

	// 요청 전송
	resp, err := wa.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 응답 상태 확인
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// SendBatch 여러 알림 일괄 전송
func (wa *WebhookAlerter) SendBatch(ctx context.Context, events []*AlertEvent) error {
	// 배치 페이로드 생성
	batchPayload := map[string]interface{}{
		"batch_id":    fmt.Sprintf("batch_%d", time.Now().Unix()),
		"event_count": len(events),
		"events":      events,
	}

	payload, err := json.Marshal(batchPayload)
	if err != nil {
		return fmt.Errorf("failed to create batch payload: %w", err)
	}

	// 재시도 로직과 함께 전송
	var lastErr error
	for i := 0; i <= wa.retryCount; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wa.retryInterval):
			}
		}

		if err := wa.sendRequest(ctx, payload); err != nil {
			lastErr = err
			continue
		}

		return nil
	}

	return fmt.Errorf("failed after %d retries: %w", wa.retryCount, lastErr)
}

// IsEnabled 알림 채널 활성화 여부
func (wa *WebhookAlerter) IsEnabled() bool {
	return wa.enabled
}

// Name 알림 채널 이름
func (wa *WebhookAlerter) Name() string {
	return "webhook"
}
