// Package alerts provides alerting and webhook functionality for ProxyND.
// It defines event types, alert levels, and webhook notification structures
// for monitoring and alerting on various system events.
package alerts

import (
	"fmt"
	"time"
)

// WebhookEventType represents the type of event that triggers a webhook notification.
type WebhookEventType string

const (
	// Cache-related events
	EventCacheExpiry     WebhookEventType = "cache.expiry"     // Cache entry expired
	EventCacheMiss       WebhookEventType = "cache.miss"       // Cache miss occurred
	EventCacheEviction   WebhookEventType = "cache.eviction"   // Cache entry evicted
	EventCacheFull       WebhookEventType = "cache.full"       // Cache capacity exceeded
	EventCacheError      WebhookEventType = "cache.error"      // Cache operation error
	EventCacheCleared    WebhookEventType = "cache.cleared"    // Cache cleared
	EventCacheCorruption WebhookEventType = "cache.corruption" // Cache corruption detected

	// Authentication and authorization events
	EventAuthFailure        WebhookEventType = "auth.failure"           // Authentication failed
	EventAuthSuccess        WebhookEventType = "auth.success"           // Authentication succeeded
	EventAuthBlocked        WebhookEventType = "auth.blocked"           // Authentication blocked
	EventAuthRateLimit      WebhookEventType = "auth.rate_limit"        // Authentication rate limited
	EventPermissionDenied   WebhookEventType = "auth.permission_denied" // Permission denied
	EventUnauthorizedAccess WebhookEventType = "auth.unauthorized"      // Unauthorized access attempt

	// Policy violation events
	EventPolicyViolation WebhookEventType = "policy.violation"       // Policy violation detected
	EventPackageBlocked  WebhookEventType = "policy.package_blocked" // Package blocked by policy
	EventSizeExceeded    WebhookEventType = "policy.size_exceeded"   // Size limit exceeded
	EventRateLimited     WebhookEventType = "policy.rate_limited"    // Rate limit exceeded
	EventIPBlocked       WebhookEventType = "policy.ip_blocked"      // IP address blocked
	EventQuotaExceeded   WebhookEventType = "policy.quota_exceeded"  // Quota exceeded

	// Server state change events
	EventServerStarted     WebhookEventType = "server.started"         // Server started
	EventServerStopped     WebhookEventType = "server.stopped"         // Server stopped
	EventServerRestarted   WebhookEventType = "server.restarted"       // Server restarted
	EventHealthCheckFailed WebhookEventType = "server.health_failed"   // Health check failed
	EventHealthCheckPassed WebhookEventType = "server.health_passed"   // Health check passed
	EventConfigReloaded    WebhookEventType = "server.config_reloaded" // Configuration reloaded
	EventConfigError       WebhookEventType = "server.config_error"    // Configuration error

	// Package-related events
	EventPackageDownloaded       WebhookEventType = "package.downloaded"        // Package downloaded
	EventPackageUploaded         WebhookEventType = "package.uploaded"          // Package uploaded
	EventPackageCorrupted        WebhookEventType = "package.corrupted"         // Package corrupted
	EventPackageVerifyFailed     WebhookEventType = "package.verify_failed"     // Package verification failed
	EventPackageSignatureInvalid WebhookEventType = "package.signature_invalid" // 서명 무효
	EventPackageHashMismatch     WebhookEventType = "package.hash_mismatch"     // 해시 불일치

	// 미러 및 프록시 이벤트
	EventMirrorDown      WebhookEventType = "mirror.down"      // 미러 서버 다운
	EventMirrorUp        WebhookEventType = "mirror.up"        // 미러 서버 복구
	EventMirrorSlow      WebhookEventType = "mirror.slow"      // 미러 서버 응답 지연
	EventMirrorError     WebhookEventType = "mirror.error"     // 미러 서버 오류
	EventProxyFallback   WebhookEventType = "proxy.fallback"   // 프록시 폴백
	EventUpstreamTimeout WebhookEventType = "upstream.timeout" // 업스트림 타임아웃

	// 보안 관련 이벤트
	EventSecurityThreat     WebhookEventType = "security.threat"        // 보안 위협
	EventMalwareDetected    WebhookEventType = "security.malware"       // 악성코드 감지
	EventVulnerabilityFound WebhookEventType = "security.vulnerability" // 취약점 발견
	EventAuditLogFull       WebhookEventType = "security.audit_full"    // 감사 로그 가득참
	EventSuspiciousActivity WebhookEventType = "security.suspicious"    // 의심스러운 활동

	// 시스템 리소스 이벤트
	EventDiskFull     WebhookEventType = "system.disk_full"     // 디스크 가득참
	EventDiskLow      WebhookEventType = "system.disk_low"      // 디스크 부족
	EventMemoryHigh   WebhookEventType = "system.memory_high"   // 메모리 사용량 높음
	EventCPUHigh      WebhookEventType = "system.cpu_high"      // CPU 사용량 높음
	EventNetworkError WebhookEventType = "system.network_error" // 네트워크 오류

	// 사용자 및 관리 이벤트
	EventUserCreated     WebhookEventType = "user.created"           // 사용자 생성
	EventUserDeleted     WebhookEventType = "user.deleted"           // 사용자 삭제
	EventUserModified    WebhookEventType = "user.modified"          // 사용자 수정
	EventConfigChanged   WebhookEventType = "admin.config_changed"   // 설정 변경
	EventBackupCompleted WebhookEventType = "admin.backup_completed" // 백업 완료
	EventBackupFailed    WebhookEventType = "admin.backup_failed"    // 백업 실패
)

// WebhookEventContext 웹훅 이벤트 컨텍스트 정보
type WebhookEventContext struct {
	// 요청 정보
	RequestID     string            `json:"request_id,omitempty"`
	UserAgent     string            `json:"user_agent,omitempty"`
	ClientIP      string            `json:"client_ip,omitempty"`
	RequestPath   string            `json:"request_path,omitempty"`
	RequestMethod string            `json:"request_method,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`

	// 사용자 정보
	Username string `json:"username,omitempty"`
	UserID   string `json:"user_id,omitempty"`
	UserRole string `json:"user_role,omitempty"`

	// 서버 정보
	ServerName string `json:"server_name,omitempty"`
	ServerAddr string `json:"server_addr,omitempty"`
	Version    string `json:"version,omitempty"`

	// 성능 메트릭
	ResponseTime  time.Duration `json:"response_time,omitempty"`
	RequestSize   int64         `json:"request_size,omitempty"`
	ResponseSize  int64         `json:"response_size,omitempty"`
	CacheHitRatio float64       `json:"cache_hit_ratio,omitempty"`
}

// CreateWebhookEvent 웹훅 이벤트 생성 헬퍼 함수
func CreateWebhookEvent(eventType WebhookEventType, level AlertLevel, title, message string) *AlertEvent {
	return &AlertEvent{
		ID:        generateEventID(),
		Level:     level,
		Type:      string(eventType),
		Title:     title,
		Message:   message,
		Source:    "proxynd",
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
}

// CreateCacheEvent 캐시 관련 이벤트 생성
func CreateCacheEvent(eventType WebhookEventType, packagePath string, cacheSize int64, message string) *AlertEvent {
	event := CreateWebhookEvent(eventType, AlertLevelInfo, string(eventType), message)
	event.Metadata["cache_size"] = cacheSize
	event.Metadata["package_path"] = packagePath
	return event
}

// CreateAuthEvent 인증 관련 이벤트 생성
func CreateAuthEvent(eventType WebhookEventType, username, clientIP string, success bool, message string) *AlertEvent {
	level := AlertLevelInfo
	if !success {
		level = AlertLevelWarning
	}

	event := CreateWebhookEvent(eventType, level, string(eventType), message)
	event.Metadata["username"] = username
	event.Metadata["client_ip"] = clientIP
	event.Metadata["success"] = success
	return event
}

// CreatePolicyEvent 정책 위반 이벤트 생성
func CreatePolicyEvent(eventType WebhookEventType, policyName, violationDetail string, severity string) *AlertEvent {
	level := AlertLevelWarning
	if severity == "high" || severity == "critical" {
		level = AlertLevelError
	}

	event := CreateWebhookEvent(eventType, level, string(eventType), violationDetail)
	event.Metadata["policy_name"] = policyName
	event.Metadata["severity"] = severity
	return event
}

// CreateServerEvent 서버 상태 이벤트 생성
func CreateServerEvent(eventType WebhookEventType, serviceName string, status string, details map[string]interface{}) *AlertEvent {
	level := AlertLevelInfo
	if status == "error" || status == "failed" {
		level = AlertLevelError
	} else if status == "warning" {
		level = AlertLevelWarning
	}

	event := CreateWebhookEvent(eventType, level, string(eventType), fmt.Sprintf("Service %s status: %s", serviceName, status))
	event.Metadata["service_name"] = serviceName
	event.Metadata["status"] = status
	for k, v := range details {
		event.Metadata[k] = v
	}
	return event
}

// CreatePackageEvent 패키지 관련 이벤트 생성
func CreatePackageEvent(eventType WebhookEventType, packageInfo *PackageInfo, message string) *AlertEvent {
	level := AlertLevelInfo
	if eventType == EventPackageCorrupted || eventType == EventPackageVerifyFailed ||
		eventType == EventPackageSignatureInvalid || eventType == EventPackageHashMismatch {
		level = AlertLevelError
	}

	event := CreateWebhookEvent(eventType, level, string(eventType), message)
	event.PackageInfo = packageInfo
	return event
}

// CreateSecurityEvent 보안 관련 이벤트 생성
func CreateSecurityEvent(eventType WebhookEventType, threatLevel string, description string, details map[string]interface{}) *AlertEvent {
	level := AlertLevelWarning
	if threatLevel == "high" || threatLevel == "critical" {
		level = AlertLevelCritical
	}

	event := CreateWebhookEvent(eventType, level, string(eventType), description)
	event.Metadata["threat_level"] = threatLevel
	for k, v := range details {
		event.Metadata[k] = v
	}
	return event
}

// CreateSystemEvent 시스템 리소스 이벤트 생성
func CreateSystemEvent(eventType WebhookEventType, resourceType string, currentValue, threshold float64, unit string) *AlertEvent {
	level := AlertLevelWarning
	if currentValue >= threshold*1.5 {
		level = AlertLevelCritical
	}

	message := fmt.Sprintf("%s usage: %.2f%s (threshold: %.2f%s)", resourceType, currentValue, unit, threshold, unit)
	event := CreateWebhookEvent(eventType, level, string(eventType), message)
	event.Metadata["resource_type"] = resourceType
	event.Metadata["current_value"] = currentValue
	event.Metadata["threshold"] = threshold
	event.Metadata["unit"] = unit
	return event
}

// generateEventID 이벤트 ID 생성 (간단한 타임스탬프 기반)
func generateEventID() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}

// IsWebhookEventType 유효한 웹훅 이벤트 타입인지 확인
func IsWebhookEventType(eventType string) bool {
	validTypes := []WebhookEventType{
		// 캐시 관련
		EventCacheExpiry, EventCacheMiss, EventCacheEviction, EventCacheFull,
		EventCacheError, EventCacheCleared, EventCacheCorruption,

		// 인증 및 권한 관련
		EventAuthFailure, EventAuthSuccess, EventAuthBlocked, EventAuthRateLimit,
		EventPermissionDenied, EventUnauthorizedAccess,

		// 정책 위반 관련
		EventPolicyViolation, EventPackageBlocked, EventSizeExceeded,
		EventRateLimited, EventIPBlocked, EventQuotaExceeded,

		// 서버 상태 변경
		EventServerStarted, EventServerStopped, EventServerRestarted,
		EventHealthCheckFailed, EventHealthCheckPassed, EventConfigReloaded,
		EventConfigError,

		// 패키지 관련
		EventPackageDownloaded, EventPackageUploaded, EventPackageCorrupted,
		EventPackageVerifyFailed, EventPackageSignatureInvalid, EventPackageHashMismatch,

		// 미러 및 프록시
		EventMirrorDown, EventMirrorUp, EventMirrorSlow, EventMirrorError,
		EventProxyFallback, EventUpstreamTimeout,

		// 보안 관련
		EventSecurityThreat, EventMalwareDetected, EventVulnerabilityFound,
		EventAuditLogFull, EventSuspiciousActivity,

		// 시스템 리소스
		EventDiskFull, EventDiskLow, EventMemoryHigh, EventCPUHigh,
		EventNetworkError,

		// 사용자 및 관리
		EventUserCreated, EventUserDeleted, EventUserModified,
		EventConfigChanged, EventBackupCompleted, EventBackupFailed,
	}

	for _, validType := range validTypes {
		if string(validType) == eventType {
			return true
		}
	}
	return false
}

// GetEventTypesByCategory 카테고리별 이벤트 타입 목록 반환
func GetEventTypesByCategory() map[string][]WebhookEventType {
	return map[string][]WebhookEventType{
		"cache": {
			EventCacheExpiry, EventCacheMiss, EventCacheEviction, EventCacheFull,
			EventCacheError, EventCacheCleared, EventCacheCorruption,
		},
		"auth": {
			EventAuthFailure, EventAuthSuccess, EventAuthBlocked, EventAuthRateLimit,
			EventPermissionDenied, EventUnauthorizedAccess,
		},
		"policy": {
			EventPolicyViolation, EventPackageBlocked, EventSizeExceeded,
			EventRateLimited, EventIPBlocked, EventQuotaExceeded,
		},
		"server": {
			EventServerStarted, EventServerStopped, EventServerRestarted,
			EventHealthCheckFailed, EventHealthCheckPassed, EventConfigReloaded,
			EventConfigError,
		},
		"package": {
			EventPackageDownloaded, EventPackageUploaded, EventPackageCorrupted,
			EventPackageVerifyFailed, EventPackageSignatureInvalid, EventPackageHashMismatch,
		},
		"mirror": {
			EventMirrorDown, EventMirrorUp, EventMirrorSlow, EventMirrorError,
			EventProxyFallback, EventUpstreamTimeout,
		},
		"security": {
			EventSecurityThreat, EventMalwareDetected, EventVulnerabilityFound,
			EventAuditLogFull, EventSuspiciousActivity,
		},
		"system": {
			EventDiskFull, EventDiskLow, EventMemoryHigh, EventCPUHigh,
			EventNetworkError,
		},
		"admin": {
			EventUserCreated, EventUserDeleted, EventUserModified,
			EventConfigChanged, EventBackupCompleted, EventBackupFailed,
		},
	}
}
