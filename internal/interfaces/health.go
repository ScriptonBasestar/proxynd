package interfaces

import (
	"context"
	"time"
)

// HealthService defines the interface for health monitoring
type HealthService interface {
	// RegisterChecker registers a health checker
	RegisterChecker(name string, checker HealthChecker)

	// UnregisterChecker removes a health checker
	UnregisterChecker(name string)

	// Start begins periodic health checks
	Start(ctx context.Context)

	// Stop halts health checking
	Stop()

	// GetStatus returns current health status
	GetStatus() HealthStatus

	// GetDetailedStatus returns detailed status for all checkers
	GetDetailedStatus() map[string]CheckResult

	// GetUptime returns service uptime
	GetUptime() time.Duration

	// ForceCheck forces an immediate health check
	ForceCheck(ctx context.Context) HealthStatus
}

// HealthChecker defines the interface for individual health checks
type HealthChecker interface {
	// Check performs the health check
	Check(ctx context.Context) error

	// Name returns the name of the checker
	Name() string

	// Type returns the type of check (critical, warning, info)
	Type() CheckType
}

// HealthStatus represents overall health status
type HealthStatus struct {
	Status      Status                 `json:"status"`
	Checks      map[string]CheckResult `json:"checks"`
	Uptime      string                 `json:"uptime"`
	LastChecked time.Time              `json:"last_checked"`
}

// CheckResult represents the result of a single health check
type CheckResult struct {
	Status    Status        `json:"status"`
	Message   string        `json:"message,omitempty"`
	Error     string        `json:"error,omitempty"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
}

// Status represents health status
type Status string

const (
	// StatusHealthy indicates the component is functioning normally
	StatusHealthy Status = "healthy"
	// StatusDegraded indicates the component has issues but is functional
	StatusDegraded Status = "degraded"
	// StatusUnhealthy indicates the component is not functioning
	StatusUnhealthy Status = "unhealthy"
)

// CheckType represents the type of health check
type CheckType string

const (
	// CheckTypeCritical indicates a critical health check
	CheckTypeCritical CheckType = "critical"
	// CheckTypeWarning indicates a warning-level health check
	CheckTypeWarning CheckType = "warning"
	// CheckTypeInfo indicates an informational health check
	CheckTypeInfo CheckType = "info"
)
