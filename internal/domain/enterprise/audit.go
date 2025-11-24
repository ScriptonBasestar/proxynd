package enterprise

import (
	"time"
)

// AuditEvent represents a logged audit event
type AuditEvent struct {
	ID           string                 `json:"id"`
	Timestamp    time.Time              `json:"timestamp"`
	UserID       string                 `json:"user_id"`
	UserEmail    string                 `json:"user_email,omitempty"`
	Action       string                 `json:"action"`        // e.g., "cache.clear", "config.update"
	ResourceType string                 `json:"resource_type"` // e.g., "cache", "config", "role"
	ResourceID   string                 `json:"resource_id"`
	Result       AuditResult            `json:"result"` // success, failure, denied
	Details      map[string]interface{} `json:"details,omitempty"`
	IPAddress    string                 `json:"ip_address"`
	UserAgent    string                 `json:"user_agent,omitempty"`
}

// AuditResult represents the outcome of an audited action
type AuditResult string

const (
	AuditResultSuccess AuditResult = "success"
	AuditResultFailure AuditResult = "failure"
	AuditResultDenied  AuditResult = "denied"
)

// AuditFilter represents search/filter criteria for audit events
type AuditFilter struct {
	UserID       string      `json:"user_id,omitempty"`
	Action       string      `json:"action,omitempty"`
	ResourceType string      `json:"resource_type,omitempty"`
	ResourceID   string      `json:"resource_id,omitempty"`
	Result       AuditResult `json:"result,omitempty"`
	StartTime    time.Time   `json:"start_time,omitempty"`
	EndTime      time.Time   `json:"end_time,omitempty"`
	Page         int         `json:"page"`
	PerPage      int         `json:"per_page"`
}

// AuditStats represents audit statistics
type AuditStats struct {
	TotalEvents    int64                 `json:"total_events"`
	EventsByAction map[string]int64      `json:"events_by_action"`
	EventsByUser   map[string]int64      `json:"events_by_user"`
	EventsByResult map[AuditResult]int64 `json:"events_by_result"`
	RecentDenials  int64                 `json:"recent_denials"`
	TimeRange      string                `json:"time_range"`
	TopUsers       []UserEventCount      `json:"top_users"`
	TopActions     []ActionEventCount    `json:"top_actions"`
}

// UserEventCount represents event count for a user
type UserEventCount struct {
	UserID    string `json:"user_id"`
	UserEmail string `json:"user_email"`
	Count     int64  `json:"count"`
}

// ActionEventCount represents event count for an action
type ActionEventCount struct {
	Action string `json:"action"`
	Count  int64  `json:"count"`
}

// ComplianceReport represents a compliance audit report
type ComplianceReport struct {
	ID           string    `json:"id"`
	GeneratedAt  time.Time `json:"generated_at"`
	ReportType   string    `json:"report_type"` // gdpr, sox, hipaa
	TimeRange    string    `json:"time_range"`
	TotalEvents  int64     `json:"total_events"`
	Violations   int64     `json:"violations"`
	ComplianceOK bool      `json:"compliance_ok"`
	Details      []string  `json:"details"`
}

// NewAuditEvent creates a new audit event
func NewAuditEvent(userID, action, resourceType, resourceID string, result AuditResult) *AuditEvent {
	return &AuditEvent{
		ID:           generateID(),
		Timestamp:    time.Now().UTC(),
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Result:       result,
		Details:      make(map[string]interface{}),
	}
}

// Audit errors
var (
	ErrInvalidFilter = &DomainError{Code: "INVALID_FILTER", Message: "Invalid audit filter criteria"}
	ErrEventNotFound = &DomainError{Code: "EVENT_NOT_FOUND", Message: "Audit event not found"}
	ErrExportFailed  = &DomainError{Code: "EXPORT_FAILED", Message: "Failed to export audit logs"}
)
