package config

import (
	"testing"

	"github.com/go-playground/assert/v2"
)

// TestWebhookConfig_DefaultConfiguration 기본 웹훅 설정 테스트
func TestWebhookConfig_DefaultConfiguration(t *testing.T) {
	config := GetDefaultWebhookConfig()

	// 기본 설정 검증
	assert.Equal(t, config.Enabled, false)
	assert.Equal(t, len(config.Endpoints) > 0, true)

	// 첫 번째 엔드포인트 검증
	endpoint := config.Endpoints[0]
	assert.Equal(t, endpoint.Name, "default")
	assert.Equal(t, endpoint.Method, "POST")
	assert.Equal(t, endpoint.Timeout, "30s")
	assert.Equal(t, endpoint.Format, "json")
	assert.Equal(t, endpoint.Credentials.Type, "none")

	// 속도 제한 설정 검증
	assert.Equal(t, config.RateLimit.Enabled, true)
	assert.Equal(t, config.RateLimit.MaxPerSecond, 10)
	assert.Equal(t, config.RateLimit.MaxPerMinute, 100)
	assert.Equal(t, config.RateLimit.MaxPerHour, 1000)

	// 재시도 설정 검증
	assert.Equal(t, config.Retry.Enabled, true)
	assert.Equal(t, config.Retry.MaxAttempts, 3)
	assert.Equal(t, config.Retry.InitialDelay, "1s")
	assert.Equal(t, config.Retry.BackoffFactor, 2.0)
	assert.Equal(t, len(config.Retry.RetryableStatus) > 0, true)

	// 이벤트 필터 설정 검증
	assert.Equal(t, config.EventFilter.Enabled, true)
	assert.Equal(t, config.EventFilter.DefaultLevel, "INFO")
	assert.Equal(t, config.EventFilter.Deduplication.Enabled, true)

	// 버퍼링 설정 검증
	assert.Equal(t, config.Buffering.Enabled, true)
	assert.Equal(t, config.Buffering.BufferSize, 1000)
	assert.Equal(t, config.Buffering.FlushInterval, "10s")

	// 보안 설정 검증
	assert.Equal(t, config.Security.EnableTLS, true)
	assert.Equal(t, config.Security.VerifySSL, true)
	assert.Equal(t, config.Security.MaxPayloadSize, int64(1048576))

	// 모니터링 설정 검증
	assert.Equal(t, config.Monitoring.Enabled, true)
	assert.Equal(t, config.Monitoring.MetricsInterval, "30s")
	assert.Equal(t, config.Monitoring.EnableHistogram, true)
	assert.Equal(t, len(config.Monitoring.HistogramBuckets) > 0, true)
}

// TestWebhookEndpointConfig 웹훅 엔드포인트 설정 테스트
func TestWebhookEndpointConfig(t *testing.T) {
	endpoint := WebhookEndpointConfig{
		Name:    "test-slack",
		URL:     "https://hooks.slack.com/services/TEST/WEBHOOK",
		Enabled: true,
		Method:  "POST",
		Headers: map[string]string{
			"Content-Type": "application/json",
			"User-Agent":   "ProxyND-Test/1.0",
		},
		Timeout: "30s",
		EventTypes: []string{
			"auth.failure",
			"security.*",
			"cache.error",
		},
		Format: "slack",
		Credentials: WebhookCredentials{
			Type: "none",
		},
		Filters: WebhookEndpointFilters{
			MinLevel:     "WARNING",
			ExcludeTypes: []string{"cache.miss"},
			PackageTypes: []string{"npm", "pip"},
		},
	}

	assert.Equal(t, endpoint.Name, "test-slack")
	assert.Equal(t, endpoint.URL, "https://hooks.slack.com/services/TEST/WEBHOOK")
	assert.Equal(t, endpoint.Enabled, true)
	assert.Equal(t, endpoint.Method, "POST")
	assert.Equal(t, endpoint.Headers["Content-Type"], "application/json")
	assert.Equal(t, endpoint.Headers["User-Agent"], "ProxyND-Test/1.0")
	assert.Equal(t, endpoint.Timeout, "30s")
	assert.Equal(t, len(endpoint.EventTypes), 3)
	assert.Equal(t, endpoint.EventTypes[0], "auth.failure")
	assert.Equal(t, endpoint.Format, "slack")
	assert.Equal(t, endpoint.Credentials.Type, "none")
	assert.Equal(t, endpoint.Filters.MinLevel, "WARNING")
	assert.Equal(t, len(endpoint.Filters.ExcludeTypes), 1)
	assert.Equal(t, endpoint.Filters.ExcludeTypes[0], "cache.miss")
	assert.Equal(t, len(endpoint.Filters.PackageTypes), 2)
}

// TestWebhookCredentials 웹훅 인증 정보 테스트
func TestWebhookCredentials(t *testing.T) {
	// Basic 인증 테스트
	basicCreds := WebhookCredentials{
		Type:     "basic",
		Username: "testuser",
		Password: "testpass",
	}
	assert.Equal(t, basicCreds.Type, "basic")
	assert.Equal(t, basicCreds.Username, "testuser")
	assert.Equal(t, basicCreds.Password, "testpass")

	// Bearer 토큰 테스트
	bearerCreds := WebhookCredentials{
		Type:  "bearer",
		Token: "bearer-token-123",
	}
	assert.Equal(t, bearerCreds.Type, "bearer")
	assert.Equal(t, bearerCreds.Token, "bearer-token-123")

	// HMAC 서명 테스트
	hmacCreds := WebhookCredentials{
		Type:      "hmac",
		Secret:    "hmac-secret-key",
		Algorithm: "sha256",
	}
	assert.Equal(t, hmacCreds.Type, "hmac")
	assert.Equal(t, hmacCreds.Secret, "hmac-secret-key")
	assert.Equal(t, hmacCreds.Algorithm, "sha256")

	// OAuth2 테스트
	oauthCreds := WebhookCredentials{
		Type:         "oauth2",
		ClientID:     "client-123",
		ClientSecret: "secret-456",
		TokenURL:     "https://auth.example.com/token",
		Scope:        "webhook:write",
	}
	assert.Equal(t, oauthCreds.Type, "oauth2")
	assert.Equal(t, oauthCreds.ClientID, "client-123")
	assert.Equal(t, oauthCreds.ClientSecret, "secret-456")
	assert.Equal(t, oauthCreds.TokenURL, "https://auth.example.com/token")
	assert.Equal(t, oauthCreds.Scope, "webhook:write")
}

// TestWebhookRateLimitConfig 속도 제한 설정 테스트
func TestWebhookRateLimitConfig(t *testing.T) {
	rateLimit := WebhookRateLimitConfig{
		Enabled:         true,
		MaxPerSecond:    5,
		MaxPerMinute:    50,
		MaxPerHour:      500,
		BurstSize:       10,
		WindowDuration:  "30s",
		BackoffStrategy: "linear",
		MaxBackoffDelay: "2m",
	}

	assert.Equal(t, rateLimit.Enabled, true)
	assert.Equal(t, rateLimit.MaxPerSecond, 5)
	assert.Equal(t, rateLimit.MaxPerMinute, 50)
	assert.Equal(t, rateLimit.MaxPerHour, 500)
	assert.Equal(t, rateLimit.BurstSize, 10)
	assert.Equal(t, rateLimit.WindowDuration, "30s")
	assert.Equal(t, rateLimit.BackoffStrategy, "linear")
	assert.Equal(t, rateLimit.MaxBackoffDelay, "2m")
}

// TestWebhookRetryConfig 재시도 설정 테스트
func TestWebhookRetryConfig(t *testing.T) {
	retry := WebhookRetryConfig{
		Enabled:       true,
		MaxAttempts:   5,
		InitialDelay:  "2s",
		MaxDelay:      "60s",
		BackoffFactor: 1.5,
		RetryableStatus: []int{
			500, 502, 503, 504, 408, 429,
		},
		DeadLetterQueue: "/var/log/webhook-failures.log",
		PersistFailures: true,
	}

	assert.Equal(t, retry.Enabled, true)
	assert.Equal(t, retry.MaxAttempts, 5)
	assert.Equal(t, retry.InitialDelay, "2s")
	assert.Equal(t, retry.MaxDelay, "60s")
	assert.Equal(t, retry.BackoffFactor, 1.5)
	assert.Equal(t, len(retry.RetryableStatus), 6)
	assert.Equal(t, retry.RetryableStatus[0], 500)
	assert.Equal(t, retry.DeadLetterQueue, "/var/log/webhook-failures.log")
	assert.Equal(t, retry.PersistFailures, true)
}

// TestWebhookEventFilter 이벤트 필터 설정 테스트
func TestWebhookEventFilter(t *testing.T) {
	eventFilter := WebhookEventFilter{
		Enabled:      true,
		DefaultLevel: "WARNING",
		LevelOverrides: map[string]string{
			"security.*":   "CRITICAL",
			"cache.miss":   "DEBUG",
			"auth.failure": "ERROR",
		},
		Aggregation: WebhookAggregationConfig{
			Enabled:    true,
			WindowSize: "10m",
			MaxEvents:  200,
			GroupBy:    []string{"type", "source"},
			Strategies: map[string]string{
				"cache.*": "count",
				"auth.*":  "sample",
			},
			FlushTriggers: []string{"time", "count"},
		},
		Deduplication: WebhookDeduplicationConfig{
			Enabled:    true,
			WindowSize: "2m",
			KeyFields:  []string{"type", "message", "source"},
			Strategy:   "merge",
			MaxCount:   3,
		},
	}

	assert.Equal(t, eventFilter.Enabled, true)
	assert.Equal(t, eventFilter.DefaultLevel, "WARNING")
	assert.Equal(t, len(eventFilter.LevelOverrides), 3)
	assert.Equal(t, eventFilter.LevelOverrides["security.*"], "CRITICAL")
	assert.Equal(t, eventFilter.LevelOverrides["cache.miss"], "DEBUG")
	assert.Equal(t, eventFilter.LevelOverrides["auth.failure"], "ERROR")

	// 집계 설정 검증
	assert.Equal(t, eventFilter.Aggregation.Enabled, true)
	assert.Equal(t, eventFilter.Aggregation.WindowSize, "10m")
	assert.Equal(t, eventFilter.Aggregation.MaxEvents, 200)
	assert.Equal(t, len(eventFilter.Aggregation.GroupBy), 2)
	assert.Equal(t, eventFilter.Aggregation.GroupBy[0], "type")
	assert.Equal(t, len(eventFilter.Aggregation.Strategies), 2)
	assert.Equal(t, eventFilter.Aggregation.Strategies["cache.*"], "count")

	// 중복 제거 설정 검증
	assert.Equal(t, eventFilter.Deduplication.Enabled, true)
	assert.Equal(t, eventFilter.Deduplication.WindowSize, "2m")
	assert.Equal(t, len(eventFilter.Deduplication.KeyFields), 3)
	assert.Equal(t, eventFilter.Deduplication.Strategy, "merge")
	assert.Equal(t, eventFilter.Deduplication.MaxCount, 3)
}

// TestWebhookBufferingConfig 버퍼링 설정 테스트
func TestWebhookBufferingConfig(t *testing.T) {
	buffering := WebhookBufferingConfig{
		Enabled:        true,
		BufferSize:     2000,
		FlushInterval:  "5s",
		FlushThreshold: 200,
		BatchSize:      25,
		PersistBuffer:  true,
		BufferPath:     "/custom/buffer/path",
	}

	assert.Equal(t, buffering.Enabled, true)
	assert.Equal(t, buffering.BufferSize, 2000)
	assert.Equal(t, buffering.FlushInterval, "5s")
	assert.Equal(t, buffering.FlushThreshold, 200)
	assert.Equal(t, buffering.BatchSize, 25)
	assert.Equal(t, buffering.PersistBuffer, true)
	assert.Equal(t, buffering.BufferPath, "/custom/buffer/path")
}

// TestWebhookSecurityConfig 보안 설정 테스트
func TestWebhookSecurityConfig(t *testing.T) {
	security := WebhookSecurityConfig{
		EnableTLS:        true,
		VerifySSL:        false,
		CACertPath:       "/etc/ssl/certs/ca.pem",
		ClientCertPath:   "/etc/ssl/certs/client.pem",
		ClientKeyPath:    "/etc/ssl/private/client.key",
		AllowedHosts:     []string{"trusted.example.com", "*.internal.company.com"},
		BlockedHosts:     []string{"blocked.example.com"},
		MaxPayloadSize:   2097152, // 2MB
		EncryptPayload:   true,
		EncryptionKey:    "encryption-key-123",
		SignPayload:      true,
		SigningKey:       "signing-key-456",
		SigningAlgorithm: "sha512",
	}

	assert.Equal(t, security.EnableTLS, true)
	assert.Equal(t, security.VerifySSL, false)
	assert.Equal(t, security.CACertPath, "/etc/ssl/certs/ca.pem")
	assert.Equal(t, security.ClientCertPath, "/etc/ssl/certs/client.pem")
	assert.Equal(t, security.ClientKeyPath, "/etc/ssl/private/client.key")
	assert.Equal(t, len(security.AllowedHosts), 2)
	assert.Equal(t, security.AllowedHosts[0], "trusted.example.com")
	assert.Equal(t, len(security.BlockedHosts), 1)
	assert.Equal(t, security.BlockedHosts[0], "blocked.example.com")
	assert.Equal(t, security.MaxPayloadSize, int64(2097152))
	assert.Equal(t, security.EncryptPayload, true)
	assert.Equal(t, security.EncryptionKey, "encryption-key-123")
	assert.Equal(t, security.SignPayload, true)
	assert.Equal(t, security.SigningKey, "signing-key-456")
	assert.Equal(t, security.SigningAlgorithm, "sha512")
}

// TestWebhookMonitoringConfig 모니터링 설정 테스트
func TestWebhookMonitoringConfig(t *testing.T) {
	monitoring := WebhookMonitoringConfig{
		Enabled:         true,
		MetricsInterval: "15s",
		EnableHistogram: true,
		HistogramBuckets: []float64{
			0.01, 0.1, 1.0, 10.0,
		},
		LogSuccessful:       true,
		LogFailed:           true,
		LogLevel:            "DEBUG",
		StatsRetention:      "48h",
		HealthCheckURL:      "https://health.example.com/webhook",
		HealthCheckInterval: "10m",
		AlertOnFailureRate:  0.05,
	}

	assert.Equal(t, monitoring.Enabled, true)
	assert.Equal(t, monitoring.MetricsInterval, "15s")
	assert.Equal(t, monitoring.EnableHistogram, true)
	assert.Equal(t, len(monitoring.HistogramBuckets), 4)
	assert.Equal(t, monitoring.HistogramBuckets[0], 0.01)
	assert.Equal(t, monitoring.LogSuccessful, true)
	assert.Equal(t, monitoring.LogFailed, true)
	assert.Equal(t, monitoring.LogLevel, "DEBUG")
	assert.Equal(t, monitoring.StatsRetention, "48h")
	assert.Equal(t, monitoring.HealthCheckURL, "https://health.example.com/webhook")
	assert.Equal(t, monitoring.HealthCheckInterval, "10m")
	assert.Equal(t, monitoring.AlertOnFailureRate, 0.05)
}
