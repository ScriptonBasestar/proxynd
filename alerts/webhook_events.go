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
	// EventCacheExpiry represents cache entry expiration event
	EventCacheExpiry WebhookEventType = "cache.expiry"
	// EventCacheMiss represents cache miss occurrence event
	EventCacheMiss WebhookEventType = "cache.miss"
	// EventCacheEviction represents cache entry eviction event
	EventCacheEviction WebhookEventType = "cache.eviction"
	// EventCacheFull represents cache capacity exceeded event
	EventCacheFull WebhookEventType = "cache.full"
	// EventCacheError represents cache operation error event
	EventCacheError WebhookEventType = "cache.error"
	// EventCacheCleared represents cache cleared event
	EventCacheCleared WebhookEventType = "cache.cleared"
	// EventCacheCorruption represents cache corruption detected event
	EventCacheCorruption WebhookEventType = "cache.corruption"

	// EventAuthFailure represents authentication failure event
	EventAuthFailure WebhookEventType = "auth.failure"
	// EventAuthSuccess represents authentication success event
	EventAuthSuccess WebhookEventType = "auth.success"
	// EventAuthBlocked represents authentication blocked event
	EventAuthBlocked WebhookEventType = "auth.blocked"
	// EventAuthRateLimit represents authentication rate limit event
	EventAuthRateLimit WebhookEventType = "auth.rate_limit"
	// EventPermissionDenied represents permission denied event
	EventPermissionDenied WebhookEventType = "auth.permission_denied"
	// EventUnauthorizedAccess represents unauthorized access attempt event
	EventUnauthorizedAccess WebhookEventType = "auth.unauthorized"

	// EventPolicyViolation represents policy violation detected event
	EventPolicyViolation WebhookEventType = "policy.violation"
	// EventPackageBlocked represents package blocked by policy event
	EventPackageBlocked WebhookEventType = "policy.package_blocked"
	// EventSizeExceeded represents size limit exceeded event
	EventSizeExceeded WebhookEventType = "policy.size_exceeded"
	// EventRateLimited represents rate limit exceeded event
	EventRateLimited WebhookEventType = "policy.rate_limited"
	// EventIPBlocked represents IP address blocked event
	EventIPBlocked WebhookEventType = "policy.ip_blocked"
	// EventQuotaExceeded represents quota exceeded event
	EventQuotaExceeded WebhookEventType = "policy.quota_exceeded"

	// EventServerStarted represents server started event
	EventServerStarted WebhookEventType = "server.started"
	// EventServerStopped represents server stopped event
	EventServerStopped WebhookEventType = "server.stopped"
	// EventServerRestarted represents server restarted event
	EventServerRestarted WebhookEventType = "server.restarted"
	// EventHealthCheckFailed represents health check failed event
	EventHealthCheckFailed WebhookEventType = "server.health_failed"
	// EventHealthCheckPassed represents health check passed event
	EventHealthCheckPassed WebhookEventType = "server.health_passed"
	// EventConfigReloaded represents configuration reloaded event
	EventConfigReloaded WebhookEventType = "server.config_reloaded"
	// EventConfigError represents configuration error event
	EventConfigError WebhookEventType = "server.config_error"

	// EventPackageDownloaded represents package downloaded event
	EventPackageDownloaded WebhookEventType = "package.downloaded"
	// EventPackageUploaded represents package uploaded event
	EventPackageUploaded WebhookEventType = "package.uploaded"
	// EventPackageCorrupted represents package corrupted event
	EventPackageCorrupted WebhookEventType = "package.corrupted"
	// EventPackageVerifyFailed represents package verification failed event
	EventPackageVerifyFailed WebhookEventType = "package.verify_failed"
	// EventPackageSignatureInvalid represents package signature invalid event
	EventPackageSignatureInvalid WebhookEventType = "package.signature_invalid"
	// EventPackageHashMismatch represents package hash mismatch event
	EventPackageHashMismatch WebhookEventType = "package.hash_mismatch"

	// EventMirrorDown represents mirror server down event
	EventMirrorDown WebhookEventType = "mirror.down"
	// EventMirrorUp represents mirror server recovery event
	EventMirrorUp WebhookEventType = "mirror.up"
	// EventMirrorSlow represents mirror server slow response event
	EventMirrorSlow WebhookEventType = "mirror.slow"
	// EventMirrorError represents mirror server error event
	EventMirrorError WebhookEventType = "mirror.error"
	// EventProxyFallback represents proxy fallback event
	EventProxyFallback WebhookEventType = "proxy.fallback"
	// EventUpstreamTimeout represents upstream timeout event
	EventUpstreamTimeout WebhookEventType = "upstream.timeout"

	// EventSecurityThreat represents security threat event
	EventSecurityThreat WebhookEventType = "security.threat"
	// EventMalwareDetected represents malware detected event
	EventMalwareDetected WebhookEventType = "security.malware"
	// EventVulnerabilityFound represents vulnerability found event
	EventVulnerabilityFound WebhookEventType = "security.vulnerability"
	// EventAuditLogFull represents audit log full event
	EventAuditLogFull WebhookEventType = "security.audit_full"
	// EventSuspiciousActivity represents suspicious activity event
	EventSuspiciousActivity WebhookEventType = "security.suspicious"

	// EventDiskFull represents disk full event
	EventDiskFull WebhookEventType = "system.disk_full"
	// EventDiskLow represents disk low event
	EventDiskLow WebhookEventType = "system.disk_low"
	// EventMemoryHigh represents memory high usage event
	EventMemoryHigh WebhookEventType = "system.memory_high"
	// EventCPUHigh represents CPU high usage event
	EventCPUHigh WebhookEventType = "system.cpu_high"
	// EventNetworkError represents network error event
	EventNetworkError WebhookEventType = "system.network_error"

	// EventUserCreated represents user created event
	EventUserCreated WebhookEventType = "user.created"
	// EventUserDeleted represents user deleted event
	EventUserDeleted WebhookEventType = "user.deleted"
	// EventUserModified represents user modified event
	EventUserModified WebhookEventType = "user.modified"
	// EventConfigChanged represents configuration changed event
	EventConfigChanged WebhookEventType = "admin.config_changed"
	// EventBackupCompleted represents backup completed event
	EventBackupCompleted WebhookEventType = "admin.backup_completed"
	// EventBackupFailed represents backup failed event
	EventBackupFailed WebhookEventType = "admin.backup_failed"
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
func CreateServerEvent(
	eventType WebhookEventType,
	serviceName string,
	status string,
	details map[string]interface{},
) *AlertEvent {
	level := AlertLevelInfo
	switch status {
	case "error", "failed":
		level = AlertLevelError
	case "warning":
		level = AlertLevelWarning
	}

	message := fmt.Sprintf("Service %s status: %s", serviceName, status)
	event := CreateWebhookEvent(eventType, level, string(eventType), message)
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
func CreateSecurityEvent(
	eventType WebhookEventType,
	threatLevel string,
	description string,
	details map[string]interface{},
) *AlertEvent {
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
func CreateSystemEvent(
	eventType WebhookEventType,
	resourceType string,
	currentValue, threshold float64,
	unit string,
) *AlertEvent {
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
