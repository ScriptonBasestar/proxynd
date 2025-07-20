package logging

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// HTTP method constants
const (
	// MethodPOST represents the HTTP POST method
	MethodPOST = "POST"
	// MethodPUT represents the HTTP PUT method
	MethodPUT = "PUT"
	// MethodDELETE represents the HTTP DELETE method
	MethodDELETE = "DELETE"
)

// SecurityLogger handles security-specific logging
type SecurityLogger struct {
	logger    Logger
	audit     *AuditLogger
	component string
}

// SecurityEvent types
const (
	// SecurityEventAuthFailure represents authentication failure events
	SecurityEventAuthFailure = "auth_failure"
	// SecurityEventUnauthorizedAccess represents unauthorized access attempts
	SecurityEventUnauthorizedAccess = "unauthorized_access"
	// SecurityEventSuspiciousActivity represents suspicious user activity
	SecurityEventSuspiciousActivity = "suspicious_activity"
	// SecurityEventBruteForce represents brute force attack attempts
	SecurityEventBruteForce = "brute_force"
	// SecurityEventSQLInjection represents SQL injection attempts
	SecurityEventSQLInjection = "sql_injection"
	// SecurityEventXSS represents XSS attack attempts
	SecurityEventXSS = "xss_attempt"
	// SecurityEventCSRF represents CSRF attack attempts
	SecurityEventCSRF = "csrf_attempt"
	// SecurityEventFileUpload represents suspicious file upload events
	SecurityEventFileUpload = "file_upload"
	// SecurityEventRateLimitExceeded represents rate limit violations
	SecurityEventRateLimitExceeded = "rate_limit_exceeded"
	// SecurityEventIPBlocked represents IP blocking events
	SecurityEventIPBlocked = "ip_blocked"
	// SecurityEventMalwareDetected represents malware detection events
	SecurityEventMalwareDetected = "malware_detected"
)

// ThreatLevel represents the severity of a security event
type ThreatLevel string

const (
	// ThreatLevelLow represents low-severity security threats
	ThreatLevelLow ThreatLevel = "low"
	// ThreatLevelMedium represents medium-severity security threats
	ThreatLevelMedium ThreatLevel = "medium"
	// ThreatLevelHigh represents high-severity security threats
	ThreatLevelHigh ThreatLevel = "high"
	// ThreatLevelCritical represents critical security threats
	ThreatLevelCritical ThreatLevel = "critical"
)

// SecurityEvent represents a security-related event
type SecurityEvent struct {
	Type          string                 `json:"type"`
	ThreatLevel   ThreatLevel            `json:"threat_level"`
	Description   string                 `json:"description"`
	IPAddress     string                 `json:"ip_address"`
	UserAgent     string                 `json:"user_agent"`
	UserID        string                 `json:"user_id,omitempty"`
	SessionID     string                 `json:"session_id,omitempty"`
	RequestID     string                 `json:"request_id,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	Method        string                 `json:"method,omitempty"`
	Path          string                 `json:"path,omitempty"`
	Headers       map[string]string      `json:"headers,omitempty"`
	Payload       string                 `json:"payload,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
}

// NewSecurityLogger creates a new security logger
func NewSecurityLogger(logger Logger, audit *AuditLogger) *SecurityLogger {
	return &SecurityLogger{
		logger:    logger.WithComponent("security"),
		audit:     audit,
		component: "security",
	}
}

// LogSecurityEvent logs a security event
func (s *SecurityLogger) LogSecurityEvent(ctx context.Context, event SecurityEvent) {
	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Extract context information
	if event.RequestID == "" {
		event.RequestID = GetRequestID(ctx)
	}
	if event.CorrelationID == "" {
		event.CorrelationID = GetCorrelationID(ctx)
	}
	if event.UserID == "" {
		event.UserID = GetUserID(ctx)
	}
	if event.SessionID == "" {
		event.SessionID = GetSessionID(ctx)
	}

	fields := []Field{
		String("security_event_type", event.Type),
		String("threat_level", string(event.ThreatLevel)),
		String("description", event.Description),
		String("ip_address", event.IPAddress),
		String("user_agent", event.UserAgent),
	}

	// Add optional fields
	if event.UserID != "" {
		fields = append(fields, String("user_id", event.UserID))
	}
	if event.Method != "" {
		fields = append(fields, String("method", event.Method))
	}
	if event.Path != "" {
		fields = append(fields, String("path", event.Path))
	}
	if event.Payload != "" {
		fields = append(fields, String("payload", event.Payload))
	}
	if event.Headers != nil {
		fields = append(fields, NewField("headers", event.Headers))
	}
	if event.Metadata != nil {
		fields = append(fields, NewField("metadata", event.Metadata))
	}

	// Log based on threat level
	logger := s.logger.WithContext(ctx)
	message := fmt.Sprintf("Security event: %s - %s", event.Type, event.Description)

	switch event.ThreatLevel {
	case ThreatLevelCritical:
		logger.Error(message, fields...)
	case ThreatLevelHigh:
		logger.Error(message, fields...)
	case ThreatLevelMedium:
		logger.Warn(message, fields...)
	default:
		logger.Info(message, fields...)
	}

	// Also log to audit trail
	if s.audit != nil {
		s.audit.LogSecurityEvent(ctx, event.Type, event.Description, "detected", map[string]interface{}{
			"threat_level": event.ThreatLevel,
			"ip_address":   event.IPAddress,
			"user_agent":   event.UserAgent,
			"metadata":     event.Metadata,
		})
	}
}

// Convenience methods for common security events

// LogAuthenticationFailure logs authentication failures
func (s *SecurityLogger) LogAuthenticationFailure(
	ctx context.Context,
	c *fiber.Ctx,
	reason string,
	metadata map[string]interface{},
) {
	event := SecurityEvent{
		Type:        SecurityEventAuthFailure,
		ThreatLevel: ThreatLevelMedium,
		Description: fmt.Sprintf("Authentication failed: %s", reason),
		IPAddress:   c.IP(),
		UserAgent:   c.Get("User-Agent"),
		Method:      c.Method(),
		Path:        c.Path(),
		Metadata:    metadata,
	}
	s.LogSecurityEvent(ctx, event)
}

// LogUnauthorizedAccess logs unauthorized access attempts
func (s *SecurityLogger) LogUnauthorizedAccess(
	ctx context.Context,
	c *fiber.Ctx,
	resource string,
	metadata map[string]interface{},
) {
	event := SecurityEvent{
		Type:        SecurityEventUnauthorizedAccess,
		ThreatLevel: ThreatLevelHigh,
		Description: fmt.Sprintf("Unauthorized access attempt to: %s", resource),
		IPAddress:   c.IP(),
		UserAgent:   c.Get("User-Agent"),
		Method:      c.Method(),
		Path:        c.Path(),
		Metadata:    metadata,
	}
	s.LogSecurityEvent(ctx, event)
}

// LogSuspiciousActivity logs suspicious behavior
func (s *SecurityLogger) LogSuspiciousActivity(
	ctx context.Context,
	c *fiber.Ctx,
	activity string,
	threatLevel ThreatLevel,
	metadata map[string]interface{},
) {
	event := SecurityEvent{
		Type:        SecurityEventSuspiciousActivity,
		ThreatLevel: threatLevel,
		Description: activity,
		IPAddress:   c.IP(),
		UserAgent:   c.Get("User-Agent"),
		Method:      c.Method(),
		Path:        c.Path(),
		Metadata:    metadata,
	}
	s.LogSecurityEvent(ctx, event)
}

// LogBruteForceAttempt logs brute force attacks
func (s *SecurityLogger) LogBruteForceAttempt(
	ctx context.Context,
	c *fiber.Ctx,
	target string,
	attemptCount int,
	metadata map[string]interface{},
) {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["attempt_count"] = attemptCount
	metadata["target"] = target

	event := SecurityEvent{
		Type:        SecurityEventBruteForce,
		ThreatLevel: ThreatLevelHigh,
		Description: fmt.Sprintf("Brute force attack detected against %s (%d attempts)", target, attemptCount),
		IPAddress:   c.IP(),
		UserAgent:   c.Get("User-Agent"),
		Method:      c.Method(),
		Path:        c.Path(),
		Metadata:    metadata,
	}
	s.LogSecurityEvent(ctx, event)
}

// LogInjectionAttempt logs SQL injection attempts
func (s *SecurityLogger) LogInjectionAttempt(
	ctx context.Context,
	c *fiber.Ctx,
	injectionType, payload string,
	metadata map[string]interface{},
) {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["injection_type"] = injectionType

	event := SecurityEvent{
		Type:        SecurityEventSQLInjection,
		ThreatLevel: ThreatLevelHigh,
		Description: fmt.Sprintf("%s injection attempt detected", injectionType),
		IPAddress:   c.IP(),
		UserAgent:   c.Get("User-Agent"),
		Method:      c.Method(),
		Path:        c.Path(),
		Payload:     payload,
		Metadata:    metadata,
	}
	s.LogSecurityEvent(ctx, event)
}

// LogRateLimitExceeded logs rate limiting violations
func (s *SecurityLogger) LogRateLimitExceeded(
	ctx context.Context,
	c *fiber.Ctx,
	limit int,
	window time.Duration,
	metadata map[string]interface{},
) {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["rate_limit"] = limit
	metadata["window"] = window.String()

	event := SecurityEvent{
		Type:        SecurityEventRateLimitExceeded,
		ThreatLevel: ThreatLevelMedium,
		Description: fmt.Sprintf("Rate limit exceeded: %d requests in %s", limit, window),
		IPAddress:   c.IP(),
		UserAgent:   c.Get("User-Agent"),
		Method:      c.Method(),
		Path:        c.Path(),
		Metadata:    metadata,
	}
	s.LogSecurityEvent(ctx, event)
}

// LogMalwareDetection logs malware detection
func (s *SecurityLogger) LogMalwareDetection(
	ctx context.Context,
	c *fiber.Ctx,
	fileName, malwareType string,
	metadata map[string]interface{},
) {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["file_name"] = fileName
	metadata["malware_type"] = malwareType

	event := SecurityEvent{
		Type:        SecurityEventMalwareDetected,
		ThreatLevel: ThreatLevelCritical,
		Description: fmt.Sprintf("Malware detected in file: %s (type: %s)", fileName, malwareType),
		IPAddress:   c.IP(),
		UserAgent:   c.Get("User-Agent"),
		Method:      c.Method(),
		Path:        c.Path(),
		Metadata:    metadata,
	}
	s.LogSecurityEvent(ctx, event)
}

// SecurityAnalyzer provides security analysis utilities
type SecurityAnalyzer struct {
	logger *SecurityLogger
}

// NewSecurityAnalyzer creates a new security analyzer
func NewSecurityAnalyzer(logger *SecurityLogger) *SecurityAnalyzer {
	return &SecurityAnalyzer{
		logger: logger,
	}
}

// AnalyzeRequest analyzes incoming requests for security threats
func (sa *SecurityAnalyzer) AnalyzeRequest(ctx context.Context, c *fiber.Ctx) {
	// Check for SQL injection patterns
	sa.checkSQLInjection(ctx, c)

	// Check for XSS attempts
	sa.checkXSS(ctx, c)

	// Check for suspicious user agents
	sa.checkUserAgent(ctx, c)

	// Check for suspicious IP addresses
	sa.checkIPAddress(ctx, c)

	// Check for file upload security
	if c.Method() == MethodPOST && strings.Contains(c.Get("Content-Type"), "multipart/form-data") {
		sa.checkFileUpload(ctx, c)
	}
}

// checkSQLInjection checks for SQL injection patterns
func (sa *SecurityAnalyzer) checkSQLInjection(ctx context.Context, c *fiber.Ctx) {
	sqlPatterns := []string{
		"union", "select", "insert", "update", "delete", "drop", "create", "alter",
		"exec", "execute", "sp_", "xp_", "script", "javascript", "vbscript",
		"--", "/*", "*/", "@@", "char(", "nchar(", "varchar(", "nvarchar(",
		"waitfor", "delay", "benchmark", "sleep(", "pg_sleep",
	}

	// Check query parameters
	queries := c.Queries()
	for key, value := range queries {
		lowerValue := strings.ToLower(value)
		for _, pattern := range sqlPatterns {
			if strings.Contains(lowerValue, pattern) {
				sa.logger.LogInjectionAttempt(ctx, c, "SQL", value, map[string]interface{}{
					"parameter": key,
					"pattern":   pattern,
				})
				break
			}
		}
	}

	// Check request body if present
	if len(c.Body()) > 0 {
		body := strings.ToLower(string(c.Body()))
		for _, pattern := range sqlPatterns {
			if strings.Contains(body, pattern) {
				sa.logger.LogInjectionAttempt(ctx, c, "SQL", string(c.Body()), map[string]interface{}{
					"location": "body",
					"pattern":  pattern,
				})
				break
			}
		}
	}
}

// checkXSS checks for XSS patterns
func (sa *SecurityAnalyzer) checkXSS(ctx context.Context, c *fiber.Ctx) {
	xssPatterns := []string{
		"<script", "</script>", "javascript:", "onload=", "onerror=",
		"onclick=", "onmouseover=", "onfocus=", "onblur=", "onchange=",
		"eval(", "setTimeout(", "setInterval(", "document.cookie",
		"document.write", "innerHTML", "document.location",
	}

	// Check query parameters
	queries := c.Queries()
	for key, value := range queries {
		lowerValue := strings.ToLower(value)
		for _, pattern := range xssPatterns {
			if strings.Contains(lowerValue, pattern) {
				sa.logger.LogSuspiciousActivity(ctx, c,
					fmt.Sprintf("XSS attempt detected in parameter: %s", key),
					ThreatLevelHigh,
					map[string]interface{}{
						"parameter": key,
						"pattern":   pattern,
						"value":     value,
					})
				break
			}
		}
	}
}

// checkUserAgent checks for suspicious user agents
func (sa *SecurityAnalyzer) checkUserAgent(ctx context.Context, c *fiber.Ctx) {
	userAgent := strings.ToLower(c.Get("User-Agent"))

	suspiciousAgents := []string{
		"sqlmap", "havij", "nmap", "nikto", "dirb", "dirbuster",
		"wfuzz", "burp", "zap", "w3af", "acunetix", "netsparker",
		"wget", "curl", "python-requests", "go-http-client",
	}

	for _, agent := range suspiciousAgents {
		if strings.Contains(userAgent, agent) {
			sa.logger.LogSuspiciousActivity(ctx, c,
				fmt.Sprintf("Suspicious user agent detected: %s", agent),
				ThreatLevelMedium,
				map[string]interface{}{
					"user_agent": c.Get("User-Agent"),
					"pattern":    agent,
				})
			break
		}
	}
}

// checkIPAddress checks for suspicious IP addresses
func (sa *SecurityAnalyzer) checkIPAddress(ctx context.Context, c *fiber.Ctx) {
	ip := c.IP()

	// Check for private IP addresses accessing from internet
	if isPrivateIP(ip) && c.Get("X-Forwarded-For") != "" {
		sa.logger.LogSuspiciousActivity(ctx, c,
			"Private IP address with X-Forwarded-For header",
			ThreatLevelMedium,
			map[string]interface{}{
				"ip_address":      ip,
				"x_forwarded_for": c.Get("X-Forwarded-For"),
			})
	}

	// Check for known bad IP ranges (this would typically come from threat intelligence)
	// This is a simplified example
	knownBadRanges := []string{
		"10.0.0.0/8",     // Private range - shouldn't access from internet
		"172.16.0.0/12",  // Private range
		"192.168.0.0/16", // Private range
	}

	for _, badRange := range knownBadRanges {
		if ipInRange(ip, badRange) {
			sa.logger.LogSuspiciousActivity(ctx, c,
				"Request from suspicious IP range",
				ThreatLevelLow,
				map[string]interface{}{
					"ip_address": ip,
					"range":      badRange,
				})
			break
		}
	}
}

// checkFileUpload checks file uploads for security issues
func (sa *SecurityAnalyzer) checkFileUpload(ctx context.Context, c *fiber.Ctx) {
	form, err := c.MultipartForm()
	if err != nil {
		return
	}

	for fieldName, files := range form.File {
		for _, file := range files {
			// Check file extension
			if isExecutableFile(file.Filename) {
				sa.logger.LogSuspiciousActivity(ctx, c,
					fmt.Sprintf("Executable file upload attempt: %s", file.Filename),
					ThreatLevelHigh,
					map[string]interface{}{
						"filename": file.Filename,
						"field":    fieldName,
						"size":     file.Size,
					})
			}

			// Check file size
			maxSize := int64(10 * 1024 * 1024) // 10MB
			if file.Size > maxSize {
				sa.logger.LogSuspiciousActivity(ctx, c,
					fmt.Sprintf("Large file upload: %s (%d bytes)", file.Filename, file.Size),
					ThreatLevelMedium,
					map[string]interface{}{
						"filename": file.Filename,
						"size":     file.Size,
						"max_size": maxSize,
					})
			}
		}
	}
}

// Helper functions

// isPrivateIP checks if an IP address is in private range
func isPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}

	for _, rangeStr := range privateRanges {
		_, cidr, err := net.ParseCIDR(rangeStr)
		if err != nil {
			continue
		}
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// ipInRange checks if an IP is in a CIDR range
func ipInRange(ipStr, cidrStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	_, cidr, err := net.ParseCIDR(cidrStr)
	if err != nil {
		return false
	}

	return cidr.Contains(ip)
}

// isExecutableFile checks if a file extension is executable
func isExecutableFile(filename string) bool {
	execExtensions := []string{
		".exe", ".bat", ".cmd", ".com", ".pif", ".scr", ".vbs", ".js",
		".jar", ".app", ".deb", ".rpm", ".run", ".bin", ".sh", ".py",
		".pl", ".php", ".asp", ".jsp", ".war", ".ear",
	}

	filename = strings.ToLower(filename)
	for _, ext := range execExtensions {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}
	return false
}
