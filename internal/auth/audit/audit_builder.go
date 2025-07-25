package audit

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// AuditEventBuilder 감사 이벤트 빌더
type AuditEventBuilder struct {
	service *AuditService
	event   *AuditEvent
}

// NewAuditEventBuilder 새 감사 이벤트 빌더 생성
func NewAuditEventBuilder(service *AuditService, eventType AuditEventType, level AuditLevel, message string) *AuditEventBuilder {
	// 이벤트 ID 생성
	id := generateEventID()

	event := &AuditEvent{
		ID:        id,
		Timestamp: time.Now().UTC(),
		Level:     level,
		EventType: eventType,
		Message:   message,
		Details:   make(map[string]interface{}),
		Tags:      make([]string, 0),
		Source:    "proxynd",
	}

	return &AuditEventBuilder{
		service: service,
		event:   event,
	}
}

// WithUser 사용자 정보 설정
func (b *AuditEventBuilder) WithUser(userID, userEmail string) *AuditEventBuilder {
	b.event.UserID = userID
	b.event.UserEmail = userEmail
	return b
}

// WithSession 세션 정보 설정
func (b *AuditEventBuilder) WithSession(sessionID string) *AuditEventBuilder {
	b.event.SessionID = sessionID
	return b
}

// WithClient 클라이언트 정보 설정
func (b *AuditEventBuilder) WithClient(clientIP, userAgent string) *AuditEventBuilder {
	b.event.ClientIP = clientIP
	b.event.UserAgent = userAgent
	return b
}

// WithRequest 요청 정보 설정
func (b *AuditEventBuilder) WithRequest(requestID string) *AuditEventBuilder {
	b.event.RequestID = requestID
	return b
}

// WithResource 리소스 정보 설정
func (b *AuditEventBuilder) WithResource(resource string) *AuditEventBuilder {
	b.event.Resource = resource
	return b
}

// WithAction 액션 정보 설정
func (b *AuditEventBuilder) WithAction(action string) *AuditEventBuilder {
	b.event.Action = action
	return b
}

// WithResult 결과 정보 설정
func (b *AuditEventBuilder) WithResult(result string) *AuditEventBuilder {
	b.event.Result = result
	return b
}

// WithDetail 상세 정보 추가
func (b *AuditEventBuilder) WithDetail(key string, value interface{}) *AuditEventBuilder {
	if b.event.Details == nil {
		b.event.Details = make(map[string]interface{})
	}
	b.event.Details[key] = value
	return b
}

// WithDetails 여러 상세 정보 추가
func (b *AuditEventBuilder) WithDetails(details map[string]interface{}) *AuditEventBuilder {
	if b.event.Details == nil {
		b.event.Details = make(map[string]interface{})
	}
	for k, v := range details {
		b.event.Details[k] = v
	}
	return b
}

// WithRiskScore 위험도 점수 설정 (0-100)
func (b *AuditEventBuilder) WithRiskScore(score int) *AuditEventBuilder {
	if score < 0 {
		score = 0
	} else if score > 100 {
		score = 100
	}

	b.event.RiskScore = score

	// 위험도에 따른 위협 레벨 자동 설정
	switch {
	case score >= 80:
		b.event.ThreatLevel = "critical"
	case score >= 60:
		b.event.ThreatLevel = "high"
	case score >= 40:
		b.event.ThreatLevel = "medium"
	default:
		b.event.ThreatLevel = "low"
	}

	return b
}

// WithThreatLevel 위협 레벨 설정
func (b *AuditEventBuilder) WithThreatLevel(level string) *AuditEventBuilder {
	validLevels := []string{"low", "medium", "high", "critical"}
	level = strings.ToLower(level)

	for _, validLevel := range validLevels {
		if level == validLevel {
			b.event.ThreatLevel = level
			break
		}
	}

	return b
}

// WithSource 이벤트 소스 설정
func (b *AuditEventBuilder) WithSource(source string) *AuditEventBuilder {
	b.event.Source = source
	return b
}

// WithTag 태그 추가
func (b *AuditEventBuilder) WithTag(tag string) *AuditEventBuilder {
	if tag != "" && !contains(b.event.Tags, tag) {
		b.event.Tags = append(b.event.Tags, tag)
	}
	return b
}

// WithTags 여러 태그 추가
func (b *AuditEventBuilder) WithTags(tags []string) *AuditEventBuilder {
	for _, tag := range tags {
		b.WithTag(tag)
	}
	return b
}

// WithError 에러 정보 추가
func (b *AuditEventBuilder) WithError(err error) *AuditEventBuilder {
	if err != nil {
		b.WithDetail("error", err.Error())
		b.WithTag("error")
	}
	return b
}

// WithDuration 처리 시간 추가
func (b *AuditEventBuilder) WithDuration(duration time.Duration) *AuditEventBuilder {
	b.WithDetail("duration_ms", duration.Milliseconds())
	return b
}

// WithIP IP 주소 정보 추가 (클라이언트 IP와 별도)
func (b *AuditEventBuilder) WithIP(ipType, ip string) *AuditEventBuilder {
	b.WithDetail(fmt.Sprintf("%s_ip", ipType), ip)
	return b
}

// WithUserAgent User-Agent 정보 추가 (상세)
func (b *AuditEventBuilder) WithUserAgentDetails(userAgent string) *AuditEventBuilder {
	b.event.UserAgent = userAgent

	// User-Agent 분석 (간단한 버전)
	if userAgent != "" {
		b.WithDetail("user_agent_full", userAgent)

		// 봇 감지
		botKeywords := []string{"bot", "crawler", "spider", "scraper", "scanner"}
		for _, keyword := range botKeywords {
			if strings.Contains(strings.ToLower(userAgent), keyword) {
				b.WithTag("bot")
				b.WithRiskScore(30)
				break
			}
		}

		// 의심스러운 User-Agent 감지
		suspiciousKeywords := []string{"curl", "wget", "python", "go-http", "postman"}
		for _, keyword := range suspiciousKeywords {
			if strings.Contains(strings.ToLower(userAgent), keyword) {
				b.WithTag("automated")
				break
			}
		}
	}

	return b
}

// WithHTTPMethod HTTP 메서드 추가
func (b *AuditEventBuilder) WithHTTPMethod(method string) *AuditEventBuilder {
	b.WithDetail("http_method", method)

	// 위험한 메서드 감지
	dangerousMethods := []string{"DELETE", "PUT", "PATCH"}
	for _, dangerous := range dangerousMethods {
		if strings.ToUpper(method) == dangerous {
			b.WithTag("write_operation")
			break
		}
	}

	return b
}

// WithStatusCode HTTP 상태 코드 추가
func (b *AuditEventBuilder) WithStatusCode(code int) *AuditEventBuilder {
	b.WithDetail("status_code", code)

	// 에러 상태 코드 감지
	if code >= 400 {
		b.WithTag("error_response")
		if code >= 500 {
			b.WithTag("server_error")
		} else {
			b.WithTag("client_error")
		}
	}

	return b
}

// WithFileInfo 파일 정보 추가
func (b *AuditEventBuilder) WithFileInfo(filename string, size int64, checksum string) *AuditEventBuilder {
	b.WithDetail("filename", filename)
	b.WithDetail("file_size", size)
	if checksum != "" {
		b.WithDetail("checksum", checksum)
	}
	return b
}

// WithPackageInfo 패키지 정보 추가
func (b *AuditEventBuilder) WithPackageInfo(packageType, packageName, version string) *AuditEventBuilder {
	b.WithDetail("package_type", packageType)
	b.WithDetail("package_name", packageName)
	b.WithDetail("package_version", version)
	b.WithTag("package_operation")
	return b
}

// WithGeoLocation 지리적 위치 정보 추가
func (b *AuditEventBuilder) WithGeoLocation(country, city string, lat, lon float64) *AuditEventBuilder {
	if country != "" {
		b.WithDetail("country", country)
	}
	if city != "" {
		b.WithDetail("city", city)
	}
	if lat != 0 || lon != 0 {
		b.WithDetail("latitude", lat)
		b.WithDetail("longitude", lon)
	}
	return b
}

// WithSecurityFlags 보안 플래그 추가
func (b *AuditEventBuilder) WithSecurityFlags(flags []string) *AuditEventBuilder {
	for _, flag := range flags {
		b.WithTag(fmt.Sprintf("security_%s", flag))
	}

	// 보안 이벤트는 위험도 상승
	if len(flags) > 0 {
		currentRisk := b.event.RiskScore
		if currentRisk < 50 {
			b.WithRiskScore(50 + len(flags)*10)
		}
	}

	return b
}

// Success 성공 이벤트로 표시
func (b *AuditEventBuilder) Success() *AuditEventBuilder {
	b.WithResult("success")
	b.WithTag("success")
	return b
}

// Failed 실패 이벤트로 표시
func (b *AuditEventBuilder) Failed(reason string) *AuditEventBuilder {
	b.WithResult("failed")
	b.WithTag("failed")
	if reason != "" {
		b.WithDetail("failure_reason", reason)
	}

	// 실패 이벤트는 위험도 상승
	if b.event.RiskScore < 40 {
		b.WithRiskScore(40)
	}

	return b
}

// Blocked 차단된 이벤트로 표시
func (b *AuditEventBuilder) Blocked(reason string) *AuditEventBuilder {
	b.WithResult("blocked")
	b.WithTag("blocked")
	if reason != "" {
		b.WithDetail("block_reason", reason)
	}

	// 차단 이벤트는 높은 위험도
	if b.event.RiskScore < 70 {
		b.WithRiskScore(70)
	}

	return b
}

// Log 이벤트를 최종적으로 로깅
func (b *AuditEventBuilder) Log() {
	if b.service != nil {
		b.service.writeEvent(b.event)
	}
}

// LogWithCallback 콜백과 함께 이벤트 로깅
func (b *AuditEventBuilder) LogWithCallback(callback func(*AuditEvent)) {
	if callback != nil {
		callback(b.event)
	}
	b.Log()
}

// Build 이벤트 객체 반환 (로깅하지 않음)
func (b *AuditEventBuilder) Build() *AuditEvent {
	// 이벤트 복사본 반환
	eventCopy := *b.event

	// 슬라이스 복사
	if len(b.event.Tags) > 0 {
		eventCopy.Tags = make([]string, len(b.event.Tags))
		copy(eventCopy.Tags, b.event.Tags)
	}

	// 맵 복사
	if len(b.event.Details) > 0 {
		eventCopy.Details = make(map[string]interface{})
		for k, v := range b.event.Details {
			eventCopy.Details[k] = v
		}
	}

	return &eventCopy
}

// helper functions

// generateEventID 이벤트 ID 생성
func generateEventID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// fallback to timestamp-based ID
		return fmt.Sprintf("evt_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("evt_%s", hex.EncodeToString(bytes))
}

// contains 슬라이스에 요소가 포함되어 있는지 확인
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Convenience methods for common audit patterns

// AuthenticationEvent 인증 이벤트 생성
func (s *AuditService) AuthenticationEvent(eventType AuditEventType, userID, userEmail, clientIP string, success bool) *AuditEventBuilder {
	level := LevelInfo
	if !success {
		level = LevelWarning
	}

	message := fmt.Sprintf("Authentication %s", eventType)
	if !success {
		message += " failed"
	}

	builder := NewAuditEventBuilder(s, eventType, level, message).
		WithUser(userID, userEmail).
		WithClient(clientIP, "").
		WithTag("authentication")

	if success {
		builder.Success()
	} else {
		builder.Failed("authentication_failed").WithRiskScore(60)
	}

	return builder
}

// SecurityEvent 보안 이벤트 생성
func (s *AuditService) SecurityEvent(eventType AuditEventType, message, clientIP string, riskScore int) *AuditEventBuilder {
	return NewAuditEventBuilder(s, eventType, LevelSecurity, message).
		WithClient(clientIP, "").
		WithTag("security").
		WithRiskScore(riskScore)
}

// AccessEvent 접근 이벤트 생성
func (s *AuditService) AccessEvent(resource, action, userID, clientIP string, allowed bool) *AuditEventBuilder {
	eventType := EventAuthzPermission
	level := LevelInfo
	message := fmt.Sprintf("Access to %s (%s)", resource, action)

	if !allowed {
		eventType = EventAuthzDenied
		level = LevelWarning
		message += " denied"
	} else {
		message += " granted"
	}

	builder := NewAuditEventBuilder(s, eventType, level, message).
		WithUser(userID, "").
		WithClient(clientIP, "").
		WithResource(resource).
		WithAction(action).
		WithTag("access_control")

	if allowed {
		builder.Success()
	} else {
		builder.Failed("access_denied").WithRiskScore(50)
	}

	return builder
}

// OperationEvent 운영 이벤트 생성
func (s *AuditService) OperationEvent(eventType AuditEventType, operation, userID string, success bool) *AuditEventBuilder {
	level := LevelInfo
	if !success {
		level = LevelWarning
	}

	message := fmt.Sprintf("Operation: %s", operation)
	if !success {
		message += " failed"
	}

	builder := NewAuditEventBuilder(s, eventType, level, message).
		WithUser(userID, "").
		WithAction(operation).
		WithTag("operation")

	if success {
		builder.Success()
	} else {
		builder.Failed("operation_failed")
	}

	return builder
}
