package logging

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// AuditEvent represents an audit log event
type AuditEvent struct {
	Timestamp     time.Time              `json:"timestamp"`
	EventType     string                 `json:"event_type"`
	Action        string                 `json:"action"`
	Resource      string                 `json:"resource"`
	ResourceID    string                 `json:"resource_id,omitempty"`
	UserID        string                 `json:"user_id,omitempty"`
	SessionID     string                 `json:"session_id,omitempty"`
	IPAddress     string                 `json:"ip_address,omitempty"`
	UserAgent     string                 `json:"user_agent,omitempty"`
	RequestID     string                 `json:"request_id,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	Result        string                 `json:"result"` // success, failure, unauthorized
	ErrorMessage  string                 `json:"error_message,omitempty"`
	Duration      time.Duration          `json:"duration,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// AuditLogger handles audit logging
type AuditLogger struct {
	logger    Logger
	component string
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logger Logger) *AuditLogger {
	return &AuditLogger{
		logger:    logger.WithComponent("audit"),
		component: "audit",
	}
}

// LogEvent logs an audit event
func (a *AuditLogger) LogEvent(ctx context.Context, event AuditEvent) {
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

	// Convert event to JSON for structured logging
	eventJSON, _ := json.Marshal(event)

	fields := []Field{
		String("event_type", event.EventType),
		String("action", event.Action),
		String("resource", event.Resource),
		String("result", event.Result),
		String("audit_event", string(eventJSON)),
	}

	// Add optional fields
	if event.ResourceID != "" {
		fields = append(fields, String("resource_id", event.ResourceID))
	}
	if event.UserID != "" {
		fields = append(fields, String("user_id", event.UserID))
	}
	if event.IPAddress != "" {
		fields = append(fields, String("ip_address", event.IPAddress))
	}
	if event.ErrorMessage != "" {
		fields = append(fields, String("error_message", event.ErrorMessage))
	}
	if event.Duration > 0 {
		fields = append(fields, Duration("duration", event.Duration))
	}

	// Add metadata
	if event.Metadata != nil {
		fields = append(fields, NewField("metadata", event.Metadata))
	}

	message := "Audit event: " + event.Action

	// Log based on result
	switch event.Result {
	case "failure", "error":
		a.logger.WithContext(ctx).Error(message, fields...)
	case "unauthorized", "forbidden":
		a.logger.WithContext(ctx).Warn(message, fields...)
	default:
		a.logger.WithContext(ctx).Info(message, fields...)
	}
}

// Convenience methods for common audit events

// LogAuthentication logs authentication events
func (a *AuditLogger) LogAuthentication(ctx context.Context, userID, method, result string, metadata map[string]interface{}) {
	event := AuditEvent{
		EventType:  "authentication",
		Action:     "authenticate",
		Resource:   "user",
		ResourceID: userID,
		Result:     result,
		Metadata:   metadata,
	}

	if metadata == nil {
		event.Metadata = make(map[string]interface{})
	}
	event.Metadata["auth_method"] = method

	a.LogEvent(ctx, event)
}

// LogAuthorization logs authorization events
func (a *AuditLogger) LogAuthorization(ctx context.Context, userID, resource, action, result string, metadata map[string]interface{}) {
	event := AuditEvent{
		EventType: "authorization",
		Action:    action,
		Resource:  resource,
		Result:    result,
		Metadata:  metadata,
	}

	a.LogEvent(ctx, event)
}

// LogFileAccess logs file access events
func (a *AuditLogger) LogFileAccess(ctx context.Context, filePath, action, result string, metadata map[string]interface{}) {
	event := AuditEvent{
		EventType:  "file_access",
		Action:     action,
		Resource:   "file",
		ResourceID: filePath,
		Result:     result,
		Metadata:   metadata,
	}

	a.LogEvent(ctx, event)
}

// LogConfigurationChange logs configuration change events
func (a *AuditLogger) LogConfigurationChange(ctx context.Context, configType, action, result string, changes map[string]interface{}) {
	event := AuditEvent{
		EventType:  "configuration_change",
		Action:     action,
		Resource:   "configuration",
		ResourceID: configType,
		Result:     result,
		Metadata: map[string]interface{}{
			"changes": changes,
		},
	}

	a.LogEvent(ctx, event)
}

// LogAdminAction logs administrative actions
func (a *AuditLogger) LogAdminAction(ctx context.Context, action, resource, result string, metadata map[string]interface{}) {
	event := AuditEvent{
		EventType: "admin_action",
		Action:    action,
		Resource:  resource,
		Result:    result,
		Metadata:  metadata,
	}

	a.LogEvent(ctx, event)
}

// LogSecurityEvent logs security-related events
func (a *AuditLogger) LogSecurityEvent(ctx context.Context, eventType, action, result string, metadata map[string]interface{}) {
	event := AuditEvent{
		EventType: "security_event",
		Action:    action,
		Resource:  eventType,
		Result:    result,
		Metadata:  metadata,
	}

	a.LogEvent(ctx, event)
}

// AuditMiddleware creates middleware for automatic audit logging
func AuditMiddleware(auditLogger *AuditLogger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Extract request information
		userID := GetUserID(c.UserContext())
		if userID == "" {
			userID = c.Get("X-User-ID")
		}

		// Process request
		err := c.Next()

		// Log audit event for specific endpoints
		if shouldAudit(c.Method(), c.Path()) {
			duration := time.Since(start)
			result := "success"
			errorMessage := ""

			if err != nil {
				result = "failure"
				errorMessage = err.Error()
			} else if c.Response().StatusCode() >= 400 {
				result = "failure"
			}

			event := AuditEvent{
				EventType:    "api_call",
				Action:       getActionFromRequest(c.Method(), c.Path()),
				Resource:     getResourceFromPath(c.Path()),
				UserID:       userID,
				IPAddress:    c.IP(),
				UserAgent:    c.Get("User-Agent"),
				Result:       result,
				ErrorMessage: errorMessage,
				Duration:     duration,
				Metadata: map[string]interface{}{
					"method":      c.Method(),
					"path":        c.Path(),
					"status_code": c.Response().StatusCode(),
					"query":       c.Queries(),
				},
			}

			auditLogger.LogEvent(c.UserContext(), event)
		}

		return err
	}
}

// shouldAudit determines if a request should be audited
func shouldAudit(method, path string) bool {
	// Skip health checks and metrics
	skipPaths := []string{
		"/health",
		"/metrics",
		"/favicon.ico",
	}

	for _, skipPath := range skipPaths {
		if path == skipPath {
			return false
		}
	}

	// Audit all POST, PUT, DELETE requests
	if method == "POST" || method == "PUT" || method == "DELETE" {
		return true
	}

	// Audit specific GET endpoints (admin, auth, etc.)
	auditPaths := []string{
		"/admin",
		"/auth",
		"/api/admin",
		"/api/auth",
	}

	for _, auditPath := range auditPaths {
		if len(path) >= len(auditPath) && path[:len(auditPath)] == auditPath {
			return true
		}
	}

	return false
}

// getActionFromRequest determines the action from HTTP method and path
func getActionFromRequest(method, path string) string {
	switch method {
	case "GET":
		return "read"
	case "POST":
		return "create"
	case "PUT", "PATCH":
		return "update"
	case "DELETE":
		return "delete"
	default:
		return method
	}
}

// getResourceFromPath extracts resource type from path
func getResourceFromPath(path string) string {
	// Simple resource extraction - can be enhanced based on API structure
	if len(path) > 1 {
		parts := strings.Split(path[1:], "/")
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return "unknown"
}
