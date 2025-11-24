package enterprise

import (
	"time"
)

// Alert represents a system alert
type Alert struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // security, performance, capacity, compliance
	Severity    AlertSeverity          `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Source      string                 `json:"source"` // vulnerability_scanner, performance_monitor, etc.
	TriggeredAt time.Time              `json:"triggered_at"`
	Status      AlertStatus            `json:"status"`
	AckedBy     string                 `json:"acked_by,omitempty"`
	AckedAt     *time.Time             `json:"acked_at,omitempty"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AlertSeverity represents alert severity levels
type AlertSeverity string

const (
	AlertSeverityCritical AlertSeverity = "critical"
	AlertSeverityHigh     AlertSeverity = "high"
	AlertSeverityMedium   AlertSeverity = "medium"
	AlertSeverityLow      AlertSeverity = "low"
	AlertSeverityInfo     AlertSeverity = "info"
)

// AlertStatus represents alert status
type AlertStatus string

const (
	AlertStatusActive       AlertStatus = "active"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
	AlertStatusResolved     AlertStatus = "resolved"
	AlertStatusIgnored      AlertStatus = "ignored"
)

// AlertRule represents an alert rule configuration
type AlertRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"` // threshold, anomaly, pattern
	Enabled     bool                   `json:"enabled"`
	Severity    AlertSeverity          `json:"severity"`
	Conditions  map[string]interface{} `json:"conditions"` // rule-specific conditions
	Actions     []AlertAction          `json:"actions"`    // email, webhook, slack
	CreatedBy   string                 `json:"created_by"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// AlertAction represents an action to take when alert is triggered
type AlertAction struct {
	Type   string                 `json:"type"` // email, webhook, slack
	Config map[string]interface{} `json:"config"`
}

// AlertRuleRequest represents a request to create/update an alert rule
type AlertRuleRequest struct {
	Name        string                 `json:"name" validate:"required,min=3,max=100"`
	Description string                 `json:"description" validate:"max=500"`
	Type        string                 `json:"type" validate:"required,oneof=threshold anomaly pattern"`
	Enabled     bool                   `json:"enabled"`
	Severity    AlertSeverity          `json:"severity" validate:"required"`
	Conditions  map[string]interface{} `json:"conditions" validate:"required"`
	Actions     []AlertAction          `json:"actions" validate:"required,min=1"`
}

// Alert errors
var (
	ErrAlertNotFound     = &DomainError{Code: "ALERT_NOT_FOUND", Message: "Alert not found"}
	ErrAlertRuleNotFound = &DomainError{Code: "ALERT_RULE_NOT_FOUND", Message: "Alert rule not found"}
	ErrInvalidConditions = &DomainError{Code: "INVALID_CONDITIONS", Message: "Invalid alert rule conditions"}
)
