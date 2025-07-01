package webhook

import (
	"testing"

	"proxynd/configs"
)

func TestWebhookTester_ValidateEndpointConfig(t *testing.T) {
	tester := &WebhookTester{}

	// 유효한 설정 테스트
	validEndpoint := configs.WebhookEndpointConfig{
		Name:    "test-endpoint",
		URL:     "https://example.com/webhook",
		Enabled: true,
		Method:  "POST",
		Format:  "json",
		Timeout: "30s",
		Credentials: configs.WebhookCredentials{
			Type: "none",
		},
	}

	errors := tester.ValidateEndpointConfig(validEndpoint)
	if len(errors) != 0 {
		t.Errorf("Expected no validation errors for valid config, got %v", errors)
	}

	// 잘못된 설정 테스트 - URL 누락
	invalidEndpoint := configs.WebhookEndpointConfig{
		Name:   "invalid-endpoint",
		Method: "POST",
	}

	errors = tester.ValidateEndpointConfig(invalidEndpoint)
	if len(errors) == 0 {
		t.Error("Expected validation errors for missing URL")
	}

	// URL 누락 오류 확인
	found := false
	for _, err := range errors {
		if err == "URL is required" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'URL is required' error")
	}
}

func TestWebhookTester_ValidateEndpointConfig_HTTPMethod(t *testing.T) {
	tester := &WebhookTester{}

	// 잘못된 HTTP 메서드
	endpoint := configs.WebhookEndpointConfig{
		Name:   "test-endpoint",
		URL:    "https://example.com/webhook",
		Method: "INVALID",
	}

	errors := tester.ValidateEndpointConfig(endpoint)
	if len(errors) == 0 {
		t.Error("Expected validation error for invalid HTTP method")
	}

	found := false
	for _, err := range errors {
		if err == "invalid HTTP method: INVALID" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'invalid HTTP method' error")
	}
}

func TestWebhookTester_ValidateEndpointConfig_Format(t *testing.T) {
	tester := &WebhookTester{}

	// 잘못된 포맷
	endpoint := configs.WebhookEndpointConfig{
		Name:   "test-endpoint",
		URL:    "https://example.com/webhook",
		Method: "POST",
		Format: "invalid-format",
	}

	errors := tester.ValidateEndpointConfig(endpoint)
	if len(errors) == 0 {
		t.Error("Expected validation error for invalid format")
	}

	found := false
	for _, err := range errors {
		if err == "unsupported format: invalid-format" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'unsupported format' error")
	}
}

func TestWebhookTester_ValidateEndpointConfig_Timeout(t *testing.T) {
	tester := &WebhookTester{}

	// 잘못된 타임아웃 형식
	endpoint := configs.WebhookEndpointConfig{
		Name:    "test-endpoint",
		URL:     "https://example.com/webhook",
		Method:  "POST",
		Timeout: "invalid-timeout",
	}

	errors := tester.ValidateEndpointConfig(endpoint)
	if len(errors) == 0 {
		t.Error("Expected validation error for invalid timeout")
	}

	found := false
	for _, err := range errors {
		if err == "invalid timeout format: invalid-timeout" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'invalid timeout format' error")
	}
}

func TestWebhookTester_ValidateEndpointConfig_BasicAuth(t *testing.T) {
	tester := &WebhookTester{}

	// Basic 인증 - 사용자명 누락
	endpoint := configs.WebhookEndpointConfig{
		Name:   "test-endpoint",
		URL:    "https://example.com/webhook",
		Method: "POST",
		Credentials: configs.WebhookCredentials{
			Type:     "basic",
			Password: "password",
		},
	}

	errors := tester.ValidateEndpointConfig(endpoint)
	if len(errors) == 0 {
		t.Error("Expected validation error for missing username in basic auth")
	}

	found := false
	for _, err := range errors {
		if err == "basic auth requires username and password" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'basic auth requires username and password' error")
	}
}

func TestWebhookTester_ValidateEndpointConfig_BearerAuth(t *testing.T) {
	tester := &WebhookTester{}

	// Bearer 인증 - 토큰 누락
	endpoint := configs.WebhookEndpointConfig{
		Name:   "test-endpoint",
		URL:    "https://example.com/webhook",
		Method: "POST",
		Credentials: configs.WebhookCredentials{
			Type: "bearer",
		},
	}

	errors := tester.ValidateEndpointConfig(endpoint)
	if len(errors) == 0 {
		t.Error("Expected validation error for missing token in bearer auth")
	}

	found := false
	for _, err := range errors {
		if err == "bearer auth requires token" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'bearer auth requires token' error")
	}
}

func TestWebhookTester_ValidateEndpointConfig_AuthType(t *testing.T) {
	tester := &WebhookTester{}

	// 잘못된 인증 타입
	endpoint := configs.WebhookEndpointConfig{
		Name:   "test-endpoint",
		URL:    "https://example.com/webhook",
		Method: "POST",
		Credentials: configs.WebhookCredentials{
			Type: "invalid-auth",
		},
	}

	errors := tester.ValidateEndpointConfig(endpoint)
	if len(errors) == 0 {
		t.Error("Expected validation error for invalid auth type")
	}

	found := false
	for _, err := range errors {
		if err == "unsupported auth type: invalid-auth" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'unsupported auth type' error")
	}
}

func TestWebhookTester_TestSingleEndpoint_EndpointNotFound(t *testing.T) {
	config := configs.GetDefaultWebhookConfig()
	config.Endpoints = []configs.WebhookEndpointConfig{} // 빈 엔드포인트 목록

	tester := NewWebhookTester(config, nil)

	result, err := tester.TestSingleEndpoint("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent endpoint")
	}
	if result != nil {
		t.Error("Expected nil result for nonexistent endpoint")
	}
	if err.Error() != "endpoint not found: nonexistent" {
		t.Errorf("Expected 'endpoint not found' error, got %v", err)
	}
}

func TestWebhookTester_TestSingleEndpoint_DisabledEndpoint(t *testing.T) {
	config := configs.GetDefaultWebhookConfig()
	config.Endpoints = []configs.WebhookEndpointConfig{
		{
			Name:    "disabled-endpoint",
			URL:     "https://example.com/webhook",
			Enabled: false,
		},
	}

	tester := NewWebhookTester(config, nil)

	result, err := tester.TestSingleEndpoint("disabled-endpoint")
	if err != nil {
		t.Errorf("Expected no error for disabled endpoint, got %v", err)
	}
	if result == nil {
		t.Error("Expected result for disabled endpoint")
	}
	if result.Success {
		t.Error("Expected failure for disabled endpoint")
	}
	if result.ErrorMessage != "endpoint is disabled" {
		t.Errorf("Expected 'endpoint is disabled' error, got %s", result.ErrorMessage)
	}
}

func TestWebhookTester_GetTestHistory(t *testing.T) {
	config := configs.GetDefaultWebhookConfig()
	tester := NewWebhookTester(config, nil)

	// 현재는 빈 슬라이스 반환
	history, err := tester.GetTestHistory("test-endpoint", 10)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(history) != 0 {
		t.Errorf("Expected empty history, got %d items", len(history))
	}
}

func TestNewWebhookTester(t *testing.T) {
	config := configs.GetDefaultWebhookConfig()
	sender := &WebhookSender{} // 더미 sender

	tester := NewWebhookTester(config, sender)
	if tester == nil {
		t.Error("Expected non-nil tester")
	}
	if tester.config.Enabled != config.Enabled {
		t.Error("Expected config to be set correctly")
	}
	if tester.sender != sender {
		t.Error("Expected sender to be set correctly")
	}
}
