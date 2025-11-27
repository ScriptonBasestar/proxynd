package enterprise

import (
	"time"
)

// SessionStatus represents the status of a session
type SessionStatus string

const (
	SessionStatusActive  SessionStatus = "active"
	SessionStatusExpired SessionStatus = "expired"
	SessionStatusRevoked SessionStatus = "revoked"
)

// Session represents an active user session
type Session struct {
	ID           string            `json:"id"`
	UserID       string            `json:"user_id"`
	UserEmail    string            `json:"user_email,omitempty"`
	TenantID     string            `json:"tenant_id,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	LastActivity time.Time         `json:"last_activity"`
	ExpiresAt    time.Time         `json:"expires_at"`
	IPAddress    string            `json:"ip_address"`
	UserAgent    string            `json:"user_agent"`
	Location     *GeoLocation      `json:"location,omitempty"`
	DeviceType   string            `json:"device_type"`
	Status       SessionStatus     `json:"status"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	IsCurrent    bool              `json:"is_current,omitempty"` // True if this is the requesting session
}

// GeoLocation represents geographic location information
type GeoLocation struct {
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	Region      string  `json:"region,omitempty"`
	City        string  `json:"city,omitempty"`
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
}

// SessionConfig holds session management configuration
type SessionConfig struct {
	MaxConcurrentSessions int           `json:"max_concurrent_sessions"`
	SessionTimeout        time.Duration `json:"session_timeout"`
	IdleTimeout           time.Duration `json:"idle_timeout"`
	RequireReauthAfter    time.Duration `json:"require_reauth_after"`
}

// DefaultSessionConfig returns default session configuration
func DefaultSessionConfig() *SessionConfig {
	return &SessionConfig{
		MaxConcurrentSessions: 5,
		SessionTimeout:        24 * time.Hour,
		IdleTimeout:           30 * time.Minute,
		RequireReauthAfter:    7 * 24 * time.Hour,
	}
}

// SessionStats represents session statistics
type SessionStats struct {
	TotalActiveSessions  int64            `json:"total_active_sessions"`
	UniqueUsers          int64            `json:"unique_users"`
	SessionsByDevice     map[string]int64 `json:"sessions_by_device"`
	SessionsByCountry    map[string]int64 `json:"sessions_by_country"`
	AverageSessionAge    float64          `json:"average_session_age_hours"`
	SuspiciousSessions   int64            `json:"suspicious_sessions"`
	SessionsCreatedToday int64            `json:"sessions_created_today"`
	SessionsExpiredToday int64            `json:"sessions_expired_today"`
	SessionsRevokedToday int64            `json:"sessions_revoked_today"`
}

// SuspiciousSession represents a session flagged as suspicious
type SuspiciousSession struct {
	Session     *Session  `json:"session"`
	Reason      string    `json:"reason"`
	RiskLevel   string    `json:"risk_level"` // low, medium, high, critical
	DetectedAt  time.Time `json:"detected_at"`
	Indicators  []string  `json:"indicators"`
	Recommended string    `json:"recommended_action"`
}

// SessionFilter represents filter criteria for session queries
type SessionFilter struct {
	UserID     string        `json:"user_id,omitempty"`
	TenantID   string        `json:"tenant_id,omitempty"`
	Status     SessionStatus `json:"status,omitempty"`
	DeviceType string        `json:"device_type,omitempty"`
	Country    string        `json:"country,omitempty"`
	StartTime  time.Time     `json:"start_time,omitempty"`
	EndTime    time.Time     `json:"end_time,omitempty"`
	Page       int           `json:"page"`
	PerPage    int           `json:"per_page"`
}

// NewSession creates a new session
func NewSession(userID, userEmail, ipAddress, userAgent, deviceType string) *Session {
	now := time.Now().UTC()
	config := DefaultSessionConfig()

	return &Session{
		ID:           generateID(),
		UserID:       userID,
		UserEmail:    userEmail,
		CreatedAt:    now,
		LastActivity: now,
		ExpiresAt:    now.Add(config.SessionTimeout),
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		DeviceType:   deviceType,
		Status:       SessionStatusActive,
		Metadata:     make(map[string]string),
	}
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().UTC().After(s.ExpiresAt) || s.Status == SessionStatusExpired
}

// IsActive checks if the session is active
func (s *Session) IsActive() bool {
	return s.Status == SessionStatusActive && !s.IsExpired()
}

// Touch updates the last activity time
func (s *Session) Touch() {
	s.LastActivity = time.Now().UTC()
}

// Revoke marks the session as revoked
func (s *Session) Revoke() {
	s.Status = SessionStatusRevoked
}

// Session errors
var (
	ErrSessionNotFound      = &DomainError{Code: "SESSION_NOT_FOUND", Message: "Session not found"}
	ErrSessionExpired       = &DomainError{Code: "SESSION_EXPIRED", Message: "Session has expired"}
	ErrSessionRevoked       = &DomainError{Code: "SESSION_REVOKED", Message: "Session has been revoked"}
	ErrMaxSessionsExceeded  = &DomainError{Code: "MAX_SESSIONS_EXCEEDED", Message: "Maximum concurrent sessions exceeded"}
	ErrInvalidSessionFilter = &DomainError{Code: "INVALID_SESSION_FILTER", Message: "Invalid session filter criteria"}
)
