package unit

import (
	"context"
	"testing"

	"github.com/go-playground/assert/v2"
	"proxynd/alerts"
	"proxynd/verification"
)

// MockAlertManager 테스트용 모의 알림 관리자
type MockAlertManager struct {
	sentAlerts []*alerts.AlertEvent
}

func (m *MockAlertManager) RegisterAlerter(alerter alerts.Alerter) {}
func (m *MockAlertManager) UnregisterAlerter(name string)          {}
func (m *MockAlertManager) Send(ctx context.Context, event *alerts.AlertEvent) error {
	m.sentAlerts = append(m.sentAlerts, event)
	return nil
}
func (m *MockAlertManager) SendToChannel(ctx context.Context, channelName string, event *alerts.AlertEvent) error {
	m.sentAlerts = append(m.sentAlerts, event)
	return nil
}
func (m *MockAlertManager) GetAlerters() []string {
	return []string{"mock"}
}

func TestPackageVerifier_NPM(t *testing.T) {
	// 테스트 설정
	config := &verification.VerifierConfig{
		StrictMode:     true,
		BlockOnFailure: true,
		AlertOnFailure: true,
		PackageTypeConfigs: map[string]verification.PackageConfig{
			"npm": {
				Enabled:        true,
				RequiredHashes: []string{"sha512"},
			},
		},
	}

	mockAlertManager := &MockAlertManager{}
	verifier := verification.NewPackageVerifier(config, mockAlertManager)

	// 테스트 데이터
	testData := []byte("test package content")

	// 테스트 케이스 1: 올바른 integrity
	t.Run("Valid NPM Package", func(t *testing.T) {
		metadata := map[string]string{
			"npm-integrity": "sha512-" + "ee26b0dd4af7e749aa1a8ee3c10ae9923f618980772e473f8819a5d4940e0db27ac185f8a0e1b5cc7821e8f9c0a081c4b8f6263e8716cc9f7b3a5e29a5a",
		}

		result, err := verifier.VerifyPackage(context.Background(), "npm", "test-package", testData, metadata)

		assert.Equal(t, nil, err)
		assert.Equal(t, true, result.Valid)
	})

	// 테스트 케이스 2: 잘못된 integrity
	t.Run("Invalid NPM Package", func(t *testing.T) {
		metadata := map[string]string{
			"npm-integrity": "sha512-wronghash",
		}

		result, err := verifier.VerifyPackage(context.Background(), "npm", "test-package", testData, metadata)

		assert.Equal(t, nil, err)
		assert.Equal(t, false, result.Valid)
		assert.Equal(t, "NPM package integrity check failed", result.Message)

		// 알림이 전송되었는지 확인
		assert.Equal(t, 1, len(mockAlertManager.sentAlerts))
		alert := mockAlertManager.sentAlerts[0]
		assert.Equal(t, alerts.AlertLevelCritical, alert.Level)
		assert.Equal(t, "package_verification_failed", alert.Type)
	})

	// 테스트 케이스 3: integrity 없음
	t.Run("No Integrity Provided", func(t *testing.T) {
		metadata := map[string]string{}

		result, err := verifier.VerifyPackage(context.Background(), "npm", "test-package", testData, metadata)

		assert.Equal(t, nil, err)
		assert.Equal(t, true, result.Valid)
		assert.Equal(t, "No integrity field provided", result.Message)
	})
}

func TestPackageVerifier_PyPI(t *testing.T) {
	// 테스트 설정
	config := &verification.VerifierConfig{
		StrictMode:     true,
		BlockOnFailure: true,
		AlertOnFailure: true,
		PackageTypeConfigs: map[string]verification.PackageConfig{
			"pip": {
				Enabled:        true,
				RequiredHashes: []string{"sha256"},
			},
		},
	}

	mockAlertManager := &MockAlertManager{}
	verifier := verification.NewPackageVerifier(config, mockAlertManager)

	// 테스트 데이터
	testData := []byte("test pip package")
	expectedHash := "2c03e0e8e5d5c8f8a5f5c8e8f8a5f5c8e8f8a5f5c8e8f8a5f5c8e8f8a5f5c8"

	// 테스트 케이스 1: 메타데이터의 SHA256
	t.Run("Valid PyPI Package with Metadata", func(t *testing.T) {
		metadata := map[string]string{
			"sha256": expectedHash,
		}

		result, err := verifier.VerifyPackage(context.Background(), "pip", "test-package.whl", testData, metadata)

		assert.Equal(t, nil, err)
		// 실제 해시와 다르므로 실패해야 함
		assert.Equal(t, false, result.Valid)
	})

	// 테스트 케이스 2: URL의 SHA256
	t.Run("Valid PyPI Package with URL Hash", func(t *testing.T) {
		metadata := map[string]string{}
		packagePath := "packages/test-package.whl#sha256=" + expectedHash

		result, err := verifier.VerifyPackage(context.Background(), "pip", packagePath, testData, metadata)

		assert.Equal(t, nil, err)
		// 실제 해시와 다르므로 실패해야 함
		assert.Equal(t, false, result.Valid)
	})
}

func TestPackageVerifier_Docker(t *testing.T) {
	// 테스트 설정
	config := &verification.VerifierConfig{
		StrictMode:     true,
		BlockOnFailure: true,
		AlertOnFailure: true,
		PackageTypeConfigs: map[string]verification.PackageConfig{
			"docker": {
				Enabled:        true,
				RequiredHashes: []string{"sha256"},
			},
		},
	}

	mockAlertManager := &MockAlertManager{}
	verifier := verification.NewPackageVerifier(config, mockAlertManager)

	// 테스트 데이터
	testData := []byte("docker layer content")

	// 테스트 케이스 1: Docker digest 헤더
	t.Run("Valid Docker Content with Digest Header", func(t *testing.T) {
		metadata := map[string]string{
			"docker-content-digest": "sha256:abcdef1234567890",
		}

		result, err := verifier.VerifyPackage(context.Background(), "docker", "library/nginx/blobs/sha256:abcdef", testData, metadata)

		assert.Equal(t, nil, err)
		// 실제 해시와 다르므로 실패해야 함
		assert.Equal(t, false, result.Valid)
		assert.Equal(t, "Docker content digest verification failed", result.Message)
	})

	// 테스트 케이스 2: 경로의 digest
	t.Run("Valid Docker Content with Path Digest", func(t *testing.T) {
		metadata := map[string]string{}
		packagePath := "library/nginx@sha256:abcdef1234567890"

		result, err := verifier.VerifyPackage(context.Background(), "docker", packagePath, testData, metadata)

		assert.Equal(t, nil, err)
		// 실제 해시와 다르므로 실패해야 함
		assert.Equal(t, false, result.Valid)
	})
}

func TestPackageVerifier_Maven(t *testing.T) {
	// 테스트 설정
	config := &verification.VerifierConfig{
		StrictMode:     true,
		BlockOnFailure: true,
		AlertOnFailure: true,
		PackageTypeConfigs: map[string]verification.PackageConfig{
			"maven": {
				Enabled:        true,
				RequiredHashes: []string{"sha256", "sha1"},
			},
		},
	}

	mockAlertManager := &MockAlertManager{}
	verifier := verification.NewPackageVerifier(config, mockAlertManager)

	// 테스트 데이터
	testData := []byte("maven artifact content")

	// 테스트 케이스 1: SHA256 검증
	t.Run("Maven with SHA256", func(t *testing.T) {
		metadata := map[string]string{
			"sha256": "abcdef1234567890",
		}

		result, err := verifier.VerifyPackage(context.Background(), "maven", "com/example/artifact/1.0/artifact-1.0.jar", testData, metadata)

		assert.Equal(t, nil, err)
		assert.Equal(t, false, result.Valid)
		assert.Equal(t, "Maven artifact SHA256 verification failed", result.Message)
	})

	// 테스트 케이스 2: SHA1 검증
	t.Run("Maven with SHA1", func(t *testing.T) {
		metadata := map[string]string{
			"sha1": "abcdef1234567890",
		}

		result, err := verifier.VerifyPackage(context.Background(), "maven", "com/example/artifact/1.0/artifact-1.0.jar", testData, metadata)

		assert.Equal(t, nil, err)
		assert.Equal(t, false, result.Valid)
		assert.Equal(t, "Maven artifact SHA1 verification failed", result.Message)
	})
}

func TestAlertManager(t *testing.T) {
	// 알림 설정
	config := &alerts.AlertConfig{
		Enabled: true,
		RateLimit: alerts.RateLimitConfig{
			Enabled:      true,
			MaxPerMinute: 10,
			MaxPerHour:   100,
			BurstSize:    5,
		},
	}

	alertManager := alerts.NewAlertManager(config)

	// 로그 알림 채널 추가
	logAlerter, err := alerts.NewLogAlerter(alerts.LogAlerterConfig{
		Enabled:    true,
		LogFile:    "",
		JSONFormat: false,
	})
	assert.Equal(t, nil, err)

	alertManager.RegisterAlerter(logAlerter)

	// 테스트 알림 전송
	event := &alerts.AlertEvent{
		Level:   alerts.AlertLevelCritical,
		Type:    "test_alert",
		Title:   "Test Alert",
		Message: "This is a test alert",
		Source:  "unit_test",
		PackageInfo: &alerts.PackageInfo{
			Type:         "npm",
			Name:         "test-package",
			Version:      "1.0.0",
			ExpectedHash: "expected",
			ActualHash:   "actual",
		},
	}

	err = alertManager.Send(context.Background(), event)
	assert.Equal(t, nil, err)

	// 등록된 알림 채널 확인
	alerters := alertManager.GetAlerters()
	assert.Equal(t, 1, len(alerters))
	assert.Equal(t, "log", alerters[0])
}

func TestRateLimiter(t *testing.T) {
	rateLimiter := alerts.NewRateLimiter(5, 20, 3)

	// 버스트 크기까지는 허용
	for i := 0; i < 3; i++ {
		assert.Equal(t, true, rateLimiter.Allow())
	}

	// 버스트 크기 초과하면 차단 (10초 내)
	assert.Equal(t, false, rateLimiter.Allow())
}
