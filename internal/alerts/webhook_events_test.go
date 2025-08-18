package alerts

import (
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
)

// TestCreateWebhookEvent 기본 웹훅 이벤트 생성 테스트
func TestCreateWebhookEvent(t *testing.T) {
	event := CreateWebhookEvent(EventCacheExpiry, AlertLevelWarning, "캐시 만료", "캐시가 만료되었습니다")

	assert.NotEqual(t, event.ID, "")
	assert.Equal(t, event.Level, AlertLevelWarning)
	assert.Equal(t, event.Type, string(EventCacheExpiry))
	assert.Equal(t, event.Title, "캐시 만료")
	assert.Equal(t, event.Message, "캐시가 만료되었습니다")
	assert.Equal(t, event.Source, "proxynd")
	assert.NotEqual(t, event.Metadata, nil)
}

// TestCreateCacheEvent 캐시 이벤트 생성 테스트
func TestCreateCacheEvent(t *testing.T) {
	event := CreateCacheEvent(EventCacheExpiry, "/npm/express", 1024, "express 패키지 캐시가 만료되었습니다")

	assert.Equal(t, event.Type, string(EventCacheExpiry))
	assert.Equal(t, event.Level, AlertLevelInfo)
	assert.Equal(t, event.Metadata["cache_size"], int64(1024))
	assert.Equal(t, event.Metadata["package_path"], "/npm/express")
}

// TestCreateAuthEvent 인증 이벤트 생성 테스트
func TestCreateAuthEvent(t *testing.T) {
	// 성공한 인증 이벤트
	successEvent := CreateAuthEvent(EventAuthSuccess, "testuser", "192.168.1.100", true, "인증 성공")
	assert.Equal(t, successEvent.Type, string(EventAuthSuccess))
	assert.Equal(t, successEvent.Level, AlertLevelInfo)
	assert.Equal(t, successEvent.Metadata["username"], "testuser")
	assert.Equal(t, successEvent.Metadata["client_ip"], "192.168.1.100")
	assert.Equal(t, successEvent.Metadata["success"], true)

	// 실패한 인증 이벤트
	failureEvent := CreateAuthEvent(EventAuthFailure, "baduser", "192.168.1.200", false, "인증 실패")
	assert.Equal(t, failureEvent.Type, string(EventAuthFailure))
	assert.Equal(t, failureEvent.Level, AlertLevelWarning)
	assert.Equal(t, failureEvent.Metadata["success"], false)
}

// TestCreatePolicyEvent 정책 이벤트 생성 테스트
func TestCreatePolicyEvent(t *testing.T) {
	// 낮은 심각도 정책 위반
	lowEvent := CreatePolicyEvent(EventPolicyViolation, "size_limit", "패키지 크기 제한 초과", "low")
	assert.Equal(t, lowEvent.Type, string(EventPolicyViolation))
	assert.Equal(t, lowEvent.Level, AlertLevelWarning)
	assert.Equal(t, lowEvent.Metadata["policy_name"], "size_limit")
	assert.Equal(t, lowEvent.Metadata["severity"], "low")

	// 높은 심각도 정책 위반
	highEvent := CreatePolicyEvent(EventPolicyViolation, "security_policy", "악성 패키지 감지", "critical")
	assert.Equal(t, highEvent.Level, AlertLevelError)
	assert.Equal(t, highEvent.Metadata["severity"], "critical")
}

// TestCreateServerEvent 서버 이벤트 생성 테스트
func TestCreateServerEvent(t *testing.T) {
	details := map[string]interface{}{
		"uptime":       "24h",
		"memory_usage": "75%",
	}

	event := CreateServerEvent(EventServerStarted, "proxynd", "running", details)
	assert.Equal(t, event.Type, string(EventServerStarted))
	assert.Equal(t, event.Level, AlertLevelInfo)
	assert.Equal(t, event.Metadata["service_name"], "proxynd")
	assert.Equal(t, event.Metadata["status"], "running")
	assert.Equal(t, event.Metadata["uptime"], "24h")
	assert.Equal(t, event.Metadata["memory_usage"], "75%")
}

// TestCreatePackageEvent 패키지 이벤트 생성 테스트
func TestCreatePackageEvent(t *testing.T) {
	packageInfo := &PackageInfo{
		Type:         "npm",
		Name:         "express",
		Version:      "4.18.0",
		Path:         "/npm/express",
		ExpectedHash: "abc123",
		ActualHash:   "def456",
		RemoteURL:    "https://registry.npmjs.org",
	}

	// 정상 다운로드 이벤트
	downloadEvent := CreatePackageEvent(EventPackageDownloaded, packageInfo, "패키지 다운로드 완료")
	assert.Equal(t, downloadEvent.Type, string(EventPackageDownloaded))
	assert.Equal(t, downloadEvent.Level, AlertLevelInfo)
	assert.Equal(t, downloadEvent.PackageInfo.Name, "express")

	// 패키지 손상 이벤트
	corruptedEvent := CreatePackageEvent(EventPackageCorrupted, packageInfo, "패키지 해시 불일치")
	assert.Equal(t, corruptedEvent.Type, string(EventPackageCorrupted))
	assert.Equal(t, corruptedEvent.Level, AlertLevelError)
}

// TestCreateSecurityEvent 보안 이벤트 생성 테스트
func TestCreateSecurityEvent(t *testing.T) {
	details := map[string]interface{}{
		"source_ip":   "192.168.1.100",
		"attack_type": "brute_force",
	}

	// 일반 보안 위협
	normalEvent := CreateSecurityEvent(EventSecurityThreat, "medium", "무차별 대입 공격 감지", details)
	assert.Equal(t, normalEvent.Type, string(EventSecurityThreat))
	assert.Equal(t, normalEvent.Level, AlertLevelWarning)
	assert.Equal(t, normalEvent.Metadata["threat_level"], "medium")
	assert.Equal(t, normalEvent.Metadata["source_ip"], "192.168.1.100")

	// 치명적 보안 위협
	criticalEvent := CreateSecurityEvent(EventMalwareDetected, "critical", "악성코드 감지", details)
	assert.Equal(t, criticalEvent.Level, AlertLevelCritical)
	assert.Equal(t, criticalEvent.Metadata["threat_level"], "critical")
}

// TestCreateSystemEvent 시스템 이벤트 생성 테스트
func TestCreateSystemEvent(t *testing.T) {
	// 디스크 사용량 경고
	diskEvent := CreateSystemEvent(EventDiskLow, "disk", 80.5, 75.0, "%")
	assert.Equal(t, diskEvent.Type, string(EventDiskLow))
	assert.Equal(t, diskEvent.Level, AlertLevelWarning)
	assert.Equal(t, diskEvent.Metadata["resource_type"], "disk")
	assert.Equal(t, diskEvent.Metadata["current_value"], 80.5)
	assert.Equal(t, diskEvent.Metadata["threshold"], 75.0)
	assert.Equal(t, diskEvent.Metadata["unit"], "%")

	// 치명적 디스크 사용량
	criticalDiskEvent := CreateSystemEvent(EventDiskFull, "disk", 95.0, 60.0, "%")
	assert.Equal(t, criticalDiskEvent.Level, AlertLevelCritical) // 95.0 >= 60.0 * 1.5 (90.0)
}

// TestIsWebhookEventType 이벤트 타입 유효성 검사 테스트
func TestIsWebhookEventType(t *testing.T) {
	// 유효한 이벤트 타입들
	validTypes := []string{
		string(EventCacheExpiry),
		string(EventAuthFailure),
		string(EventPolicyViolation),
		string(EventServerStarted),
		string(EventPackageDownloaded),
		string(EventMirrorDown),
		string(EventSecurityThreat),
		string(EventDiskFull),
		string(EventUserCreated),
	}

	for _, eventType := range validTypes {
		assert.Equal(t, IsWebhookEventType(eventType), true)
	}

	// 무효한 이벤트 타입들
	invalidTypes := []string{
		"invalid.type",
		"cache.unknown",
		"auth.invalid",
		"",
		"random_string",
	}

	for _, eventType := range invalidTypes {
		assert.Equal(t, IsWebhookEventType(eventType), false)
	}
}

// TestGetEventTypesByCategory 카테고리별 이벤트 타입 조회 테스트
func TestGetEventTypesByCategory(t *testing.T) {
	categories := GetEventTypesByCategory()

	// 모든 카테고리가 존재하는지 확인
	expectedCategories := []string{
		"cache", "auth", "policy", "server", "package",
		"mirror", "security", "system", "admin",
	}

	for _, category := range expectedCategories {
		_, exists := categories[category]
		assert.Equal(t, exists, true)
	}

	// 캐시 카테고리에 올바른 이벤트들이 포함되어 있는지 확인
	cacheEvents := categories["cache"]
	assert.Equal(t, len(cacheEvents) > 0, true)

	// EventCacheExpiry가 캐시 카테고리에 포함되어 있는지 확인
	found := false
	for _, event := range cacheEvents {
		if event == EventCacheExpiry {
			found = true
			break
		}
	}
	assert.Equal(t, found, true)

	// 인증 카테고리 확인
	authEvents := categories["auth"]
	assert.Equal(t, len(authEvents) > 0, true)

	authFound := false
	for _, event := range authEvents {
		if event == EventAuthFailure {
			authFound = true
			break
		}
	}
	assert.Equal(t, authFound, true)
}

// TestEventIDGeneration 이벤트 ID 생성 테스트
func TestEventIDGeneration(t *testing.T) {
	event1 := CreateWebhookEvent(EventCacheExpiry, AlertLevelInfo, "테스트 1", "메시지 1")
	time.Sleep(time.Millisecond) // ID 중복 방지
	event2 := CreateWebhookEvent(EventAuthFailure, AlertLevelWarning, "테스트 2", "메시지 2")

	// 이벤트 ID가 다른지 확인
	assert.NotEqual(t, event1.ID, event2.ID)

	// 이벤트 ID가 특정 형식을 따르는지 확인
	assert.Equal(t, len(event1.ID) > 4, true) // "evt_" prefix + timestamp
	assert.Equal(t, event1.ID[:4], "evt_")
	assert.Equal(t, event2.ID[:4], "evt_")
}

// TestWebhookEventContext 웹훅 이벤트 컨텍스트 구조체 테스트
func TestWebhookEventContext(t *testing.T) {
	context := &WebhookEventContext{
		RequestID:     "req_123456",
		UserAgent:     "npm/8.0.0",
		ClientIP:      "192.168.1.100",
		RequestPath:   "/npm/express",
		RequestMethod: "GET",
		Headers: map[string]string{
			"Authorization": "Bearer token123",
			"Accept":        "application/json",
		},
		Username:      "testuser",
		UserID:        "user_123",
		UserRole:      "developer",
		ServerName:    "proxynd-01",
		ServerAddr:    "10.0.0.1:8080",
		Version:       "1.0.0",
		ResponseTime:  150 * time.Millisecond,
		RequestSize:   1024,
		ResponseSize:  2048,
		CacheHitRatio: 0.85,
	}

	assert.Equal(t, context.RequestID, "req_123456")
	assert.Equal(t, context.UserAgent, "npm/8.0.0")
	assert.Equal(t, context.ClientIP, "192.168.1.100")
	assert.Equal(t, context.RequestPath, "/npm/express")
	assert.Equal(t, context.RequestMethod, "GET")
	assert.Equal(t, context.Headers["Authorization"], "Bearer token123")
	assert.Equal(t, context.Username, "testuser")
	assert.Equal(t, context.UserID, "user_123")
	assert.Equal(t, context.UserRole, "developer")
	assert.Equal(t, context.ServerName, "proxynd-01")
	assert.Equal(t, context.ServerAddr, "10.0.0.1:8080")
	assert.Equal(t, context.Version, "1.0.0")
	assert.Equal(t, context.ResponseTime, 150*time.Millisecond)
	assert.Equal(t, context.RequestSize, int64(1024))
	assert.Equal(t, context.ResponseSize, int64(2048))
	assert.Equal(t, context.CacheHitRatio, 0.85)
}
