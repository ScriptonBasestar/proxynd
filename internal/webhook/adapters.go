// Package webhook provides webhook adapter implementations
package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"proxynd/alerts"
	"proxynd/configs"
)

// GenericWebhookAdapter 범용 웹훅 어댑터
type GenericWebhookAdapter struct {
	client *http.Client
}

// NewGenericWebhookAdapter 새로운 범용 어댑터 생성
func NewGenericWebhookAdapter() *GenericWebhookAdapter {
	return &GenericWebhookAdapter{
		client: &http.Client{
			Timeout: time.Second * 30,
		},
	}
}

// Type 어댑터 타입 반환 (인터페이스 호환)
func (gwa *GenericWebhookAdapter) Type() string {
	//nolint:goconst // webhookTypeGeneric 상수 사용보다 직접 문자열이 더 명확
	return "generic"
}

// Name 어댑터 이름 반환 (하위 호환성 - 테스트에서 사용)
func (gwa *GenericWebhookAdapter) Name() string {
	return "generic"
}

// SupportedFormats 지원하는 포맷 목록 반환
func (gwa *GenericWebhookAdapter) SupportedFormats() []string {
	return []string{"json", "text"}
}

// Send 웹훅 전송 (새로운 인터페이스 호환)
func (gwa *GenericWebhookAdapter) Send(ctx context.Context, endpoint string, event *alerts.AlertEvent) error {
	// 기본 엔드포인트 설정 생성
	endpointConfig := configs.WebhookEndpointConfig{
		URL:    endpoint,
		Method: "POST",
		Format: "json",
	}
	return gwa.SendToEndpoint(ctx, event, endpointConfig)
}

// Validate 엔드포인트 검증 (인터페이스 호환)
func (gwa *GenericWebhookAdapter) Validate(endpoint string) error {
	// 기본 URL 검증
	if endpoint == "" {
		return fmt.Errorf("endpoint URL cannot be empty")
	}
	return nil
}

// SendToEndpoint 웹훅 전송 (기존 구현, 하위 호환성용)
func (gwa *GenericWebhookAdapter) SendToEndpoint(ctx context.Context, event *alerts.AlertEvent,
	endpoint configs.WebhookEndpointConfig) error {
	// 메시지 포맷팅
	payload, err := gwa.FormatMessage(event, endpoint.Format)
	if err != nil {
		return fmt.Errorf("failed to format message: %w", err)
	}

	// JSON 직렬화
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// HTTP 요청 생성
	req, err := http.NewRequestWithContext(ctx, endpoint.Method, endpoint.URL, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 헤더 설정
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ProxyND-Webhook/1.0")

	// 추가 헤더 설정
	for key, value := range endpoint.Headers {
		req.Header.Set(key, value)
	}

	// 인증 설정
	if err := gwa.setAuthentication(req, endpoint.Credentials); err != nil {
		return fmt.Errorf("failed to set authentication: %w", err)
	}

	// 요청 전송
	resp, err := gwa.client.Do(req)
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

// FormatMessage 메시지 포맷팅
func (gwa *GenericWebhookAdapter) FormatMessage(event *alerts.AlertEvent, format string) (interface{}, error) {
	switch format {
	case "json":
		return gwa.formatJSON(event), nil
	case "text":
		return gwa.formatText(event), nil
	default:
		return gwa.formatJSON(event), nil
	}
}

// formatJSON JSON 포맷으로 변환
func (gwa *GenericWebhookAdapter) formatJSON(event *alerts.AlertEvent) map[string]interface{} {
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

	return payload
}

// formatText 텍스트 포맷으로 변환
func (gwa *GenericWebhookAdapter) formatText(event *alerts.AlertEvent) map[string]interface{} {
	message := fmt.Sprintf("[%s] %s: %s", event.Level, event.Type, event.Message)

	return map[string]interface{}{
		"text": message,
		"details": map[string]interface{}{
			"id":        event.ID,
			"timestamp": event.Timestamp.Format(time.RFC3339),
			"source":    event.Source,
			"metadata":  event.Metadata,
		},
	}
}

// setAuthentication 인증 설정
func (gwa *GenericWebhookAdapter) setAuthentication(req *http.Request, creds configs.WebhookCredentials) error {
	switch creds.Type {
	case "none":
		// 인증 없음
		return nil

	case "basic":
		if creds.Username == "" || creds.Password == "" {
			return fmt.Errorf("basic auth requires username and password")
		}
		req.SetBasicAuth(creds.Username, creds.Password)

	case "bearer":
		if creds.Token == "" {
			return fmt.Errorf("bearer auth requires token")
		}
		req.Header.Set("Authorization", "Bearer "+creds.Token)

	case "hmac":
		// HMAC 서명은 별도 구현 필요
		return gwa.setHMACSignature(req, creds)

	case "oauth2":
		// OAuth2는 별도 구현 필요
		return gwa.setOAuth2Token(req, creds)

	default:
		return fmt.Errorf("unsupported auth type: %s", creds.Type)
	}

	// 추가 헤더 설정
	for key, value := range creds.ExtraHeaders {
		req.Header.Set(key, value)
	}

	return nil
}

// setHMACSignature HMAC 서명 설정
func (gwa *GenericWebhookAdapter) setHMACSignature(req *http.Request, creds configs.WebhookCredentials) error {
	if creds.Secret == "" {
		return fmt.Errorf("HMAC auth requires secret")
	}

	// HMAC 서명 구현 (실제로는 crypto/hmac 사용)
	// 여기서는 간단한 구현
	signature := fmt.Sprintf("hmac-sha256=%s", creds.Secret) // 실제로는 요청 본문을 해시
	req.Header.Set("X-Hub-Signature-256", signature)

	return nil
}

// setOAuth2Token OAuth2 토큰 설정
func (gwa *GenericWebhookAdapter) setOAuth2Token(req *http.Request, creds configs.WebhookCredentials) error {
	// OAuth2 토큰 획득 및 설정 (실제로는 OAuth2 라이브러리 사용)
	if creds.ClientID == "" || creds.ClientSecret == "" || creds.TokenURL == "" {
		return fmt.Errorf("OAuth2 auth requires client_id, client_secret, and token_url")
	}

	// 간단한 구현 (실제로는 토큰 캐싱 및 갱신 로직 필요)
	token := "oauth2_access_token" // 실제로는 토큰 서버에서 획득
	req.Header.Set("Authorization", "Bearer "+token)

	return nil
}

// WebhookAdapter 인터페이스 정의
type WebhookAdapter interface {
	Name() string
	SupportedFormats() []string
	Send(ctx context.Context, event *alerts.AlertEvent, endpoint configs.WebhookEndpointConfig) error
	FormatMessage(event *alerts.AlertEvent, format string) (interface{}, error)
}

// SlackWebhookAdapter Slack 전용 웹훅 어댑터
type SlackWebhookAdapter struct {
	client *http.Client
}

// NewSlackWebhookAdapter 새로운 Slack 어댑터 생성
func NewSlackWebhookAdapter() *SlackWebhookAdapter {
	return &SlackWebhookAdapter{
		client: &http.Client{
			Timeout: time.Second * 30,
		},
	}
}

// Name 어댑터 이름 반환
func (swa *SlackWebhookAdapter) Name() string {
	return webhookTypeSlack
}

// SupportedFormats 지원하는 포맷 목록 반환
func (swa *SlackWebhookAdapter) SupportedFormats() []string {
	return []string{webhookTypeSlack, "slack-blocks", "slack-text"}
}

// Send Slack 웹훅 전송
func (swa *SlackWebhookAdapter) Send(ctx context.Context, event *alerts.AlertEvent,
	endpoint configs.WebhookEndpointConfig) error {
	// Slack 메시지 포맷팅
	payload, err := swa.FormatMessage(event, endpoint.Format)
	if err != nil {
		return fmt.Errorf("failed to format Slack message: %w", err)
	}

	// JSON 직렬화
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %w", err)
	}

	// HTTP 요청 생성
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint.URL, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Slack request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ProxyND-Webhook/1.0")

	// 요청 전송
	resp, err := swa.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Slack request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// FormatMessage Slack 메시지 포맷팅
func (swa *SlackWebhookAdapter) FormatMessage(event *alerts.AlertEvent, format string) (interface{}, error) {
	switch format {
	case webhookTypeSlack, "":
		return swa.formatAttachment(event), nil
	case "slack-blocks":
		return swa.formatBlocks(event), nil
	case "slack-text":
		return swa.formatText(event), nil
	default:
		return swa.formatAttachment(event), nil
	}
}

// formatAttachment Slack 어태치먼트 포맷 (레거시)
func (swa *SlackWebhookAdapter) formatAttachment(event *alerts.AlertEvent) map[string]interface{} {
	// 레벨에 따른 색상 결정
	color := swa.getSlackColorForLevel(event.Level)

	// 이모지 추가
	emoji := swa.getSlackEmojiForType(event.Type)

	attachment := map[string]interface{}{
		"color":     color,
		"title":     fmt.Sprintf("%s %s", emoji, event.Title),
		"text":      event.Message,
		"timestamp": event.Timestamp.Unix(),
		"fields": []map[string]interface{}{
			{
				"title": "Event Type",
				"value": event.Type,
				"short": true,
			},
			{
				"title": "Level",
				"value": string(event.Level),
				"short": true,
			},
			{
				"title": "Source",
				"value": event.Source,
				"short": true,
			},
		},
	}

	// 패키지 정보 추가
	if event.PackageInfo != nil {
		fields, ok := attachment["fields"].([]map[string]interface{})
		if !ok {
			fields = make([]map[string]interface{}, 0)
		}
		fields = append(fields, map[string]interface{}{
			"title": "Package",
			"value": fmt.Sprintf("%s/%s@%s", event.PackageInfo.Type, event.PackageInfo.Name, event.PackageInfo.Version),
			"short": true,
		})
		attachment["fields"] = fields
	}

	return map[string]interface{}{
		"text":        "🚨 ProxyND Alert",
		"attachments": []interface{}{attachment},
	}
}

// formatBlocks Slack 블록 킷 포맷 (신규)
func (swa *SlackWebhookAdapter) formatBlocks(event *alerts.AlertEvent) map[string]interface{} {
	emoji := swa.getSlackEmojiForType(event.Type)

	blocks := []interface{}{
		map[string]interface{}{
			"type": "header",
			"text": map[string]interface{}{
				"type": "plain_text",
				"text": fmt.Sprintf("%s ProxyND Alert", emoji),
			},
		},
		map[string]interface{}{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*%s*\n%s", event.Title, event.Message),
			},
		},
	}

	return map[string]interface{}{
		"blocks": blocks,
	}
}

// formatText 단순 텍스트 포맷
func (swa *SlackWebhookAdapter) formatText(event *alerts.AlertEvent) map[string]interface{} {
	emoji := swa.getSlackEmojiForType(event.Type)

	text := fmt.Sprintf("%s *ProxyND Alert*\n\n", emoji)
	text += fmt.Sprintf("*%s*\n", event.Title)
	text += fmt.Sprintf("%s\n\n", event.Message)

	return map[string]interface{}{
		"text": text,
	}
}

// getSlackColorForLevel 레벨에 따른 색상 반환
func (swa *SlackWebhookAdapter) getSlackColorForLevel(level alerts.AlertLevel) string {
	switch level {
	case alerts.AlertLevelInfo:
		return "good"
	case alerts.AlertLevelWarning:
		return "warning"
	case alerts.AlertLevelError:
		return "danger"
	case alerts.AlertLevelCritical:
		return "#ff0000"
	default:
		return "#cccccc"
	}
}

// getSlackEmojiForType 이벤트 타입에 따른 이모지 반환
func (swa *SlackWebhookAdapter) getSlackEmojiForType(eventType string) string {
	switch {
	case strings.HasPrefix(eventType, "auth."):
		return "🔒"
	case strings.HasPrefix(eventType, "cache."):
		return "💾"
	case strings.HasPrefix(eventType, "server."):
		return "🚨"
	case strings.HasPrefix(eventType, "security."):
		return "🛡️"
	case strings.HasPrefix(eventType, "package."):
		return "📦"
	default:
		return "ℹ️"
	}
}

// DiscordWebhookAdapter Discord 전용 웹훅 어댑터
type DiscordWebhookAdapter struct {
	client *http.Client
}

// NewDiscordWebhookAdapter 새로운 Discord 어댑터 생성
func NewDiscordWebhookAdapter() *DiscordWebhookAdapter {
	return &DiscordWebhookAdapter{
		client: &http.Client{
			Timeout: time.Second * 30,
		},
	}
}

// Name 어댑터 이름 반환
func (dwa *DiscordWebhookAdapter) Name() string {
	return webhookTypeDiscord
}

// SupportedFormats 지원하는 포맷 목록 반환
func (dwa *DiscordWebhookAdapter) SupportedFormats() []string {
	return []string{"discord", "discord-embed", "discord-text"}
}

// Send Discord 웹훅 전송
func (dwa *DiscordWebhookAdapter) Send(ctx context.Context, event *alerts.AlertEvent,
	endpoint configs.WebhookEndpointConfig) error {
	// Discord 메시지 포맷팅
	payload, err := dwa.FormatMessage(event, endpoint.Format)
	if err != nil {
		return fmt.Errorf("failed to format Discord message: %w", err)
	}

	// JSON 직렬화
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Discord payload: %w", err)
	}

	// HTTP 요청 생성
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint.URL, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Discord request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ProxyND-Webhook/1.0")

	// 요청 전송
	resp, err := dwa.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Discord request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Discord는 204 No Content를 반환
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("discord webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// FormatMessage Discord 메시지 포맷팅
func (dwa *DiscordWebhookAdapter) FormatMessage(event *alerts.AlertEvent, format string) (interface{}, error) {
	switch format {
	case webhookTypeDiscord, "discord-embed", "":
		return dwa.formatEmbed(event), nil
	case "discord-text":
		return dwa.formatDiscordText(event), nil
	default:
		return dwa.formatEmbed(event), nil
	}
}

// formatEmbed Discord 임베드 포맷
func (dwa *DiscordWebhookAdapter) formatEmbed(event *alerts.AlertEvent) map[string]interface{} {
	// 색상 결정
	color := dwa.getDiscordColorForLevel(event.Level)

	// 이모지 추가
	emoji := dwa.getDiscordEmojiForType(event.Type)

	embed := map[string]interface{}{
		"title":       fmt.Sprintf("%s %s", emoji, event.Title),
		"description": event.Message,
		"color":       color,
		"timestamp":   event.Timestamp.Format(time.RFC3339),
		"footer": map[string]interface{}{
			"text": fmt.Sprintf("ProxyND Alert • ID: %s", event.ID),
		},
	}

	return map[string]interface{}{
		"embeds": []interface{}{embed},
	}
}

// formatDiscordText 단순 텍스트 포맷
func (dwa *DiscordWebhookAdapter) formatDiscordText(event *alerts.AlertEvent) map[string]interface{} {
	emoji := dwa.getDiscordEmojiForType(event.Type)

	content := fmt.Sprintf("%s **ProxyND Alert**\n\n", emoji)
	content += fmt.Sprintf("**%s**\n", event.Title)
	content += event.Message

	return map[string]interface{}{
		"content": content,
	}
}

// getDiscordColorForLevel 레벨에 따른 색상 반환 (Discord 임베드용 십진수 색상)
func (dwa *DiscordWebhookAdapter) getDiscordColorForLevel(level alerts.AlertLevel) int {
	switch level {
	case alerts.AlertLevelInfo:
		return 0x00ff00 // 녹색
	case alerts.AlertLevelWarning:
		return 0xffff00 // 노란색
	case alerts.AlertLevelError:
		return 0xff6600 // 주황색
	case alerts.AlertLevelCritical:
		return 0xff0000 // 빨간색
	default:
		return 0x808080 // 회색
	}
}

// getDiscordEmojiForType 이벤트 타입에 따른 이모지 반환
func (dwa *DiscordWebhookAdapter) getDiscordEmojiForType(eventType string) string {
	switch {
	case strings.HasPrefix(eventType, "auth."):
		return "🔒"
	case strings.HasPrefix(eventType, "cache."):
		return "💾"
	case strings.HasPrefix(eventType, "server."):
		return "🚨"
	case strings.HasPrefix(eventType, "security."):
		return "🛡️"
	case strings.HasPrefix(eventType, "package."):
		return "📦"
	default:
		return "ℹ️"
	}
}
