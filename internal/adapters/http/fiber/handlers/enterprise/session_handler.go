package enterprise

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/domain/enterprise"
)

// SessionHandler handles session management API requests
type SessionHandler struct{}

// NewSessionHandler creates a new session handler
func NewSessionHandler() *SessionHandler {
	return &SessionHandler{}
}

// ListSessions godoc
// @Summary List user's active sessions
// @Description Get a paginated list of the current user's active sessions
// @Tags Sessions
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Success 200 {object} map[string]interface{} "Paginated list of sessions"
// @Failure 402 {object} map[string]interface{} "Enterprise license required"
// @Router /api/v1/enterprise/sessions [get]
func (h *SessionHandler) ListSessions(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	if perPage > 100 {
		perPage = 100
	}

	// Get current user from context (in production, from JWT claims)
	userID := c.Locals("user_id")
	if userID == nil {
		userID = "current_user"
	}

	// Mock sessions for the user
	sessions := getMockUserSessions(userID.(string))
	total := int64(len(sessions))
	pagination := enterprise.NewPagination(page, perPage, total)

	return c.JSON(fiber.Map{
		"success":    true,
		"data":       sessions,
		"pagination": pagination,
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// GetSession godoc
// @Summary Get session details
// @Description Get detailed information about a specific session
// @Tags Sessions
// @Accept json
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} map[string]interface{} "Session details"
// @Failure 404 {object} map[string]interface{} "Session not found"
// @Failure 402 {object} map[string]interface{} "Enterprise license required"
// @Router /api/v1/enterprise/sessions/{id} [get]
func (h *SessionHandler) GetSession(c *fiber.Ctx) error {
	id := c.Params("id")

	sessions := getMockUserSessions("current_user")
	for _, session := range sessions {
		if session.ID == id {
			return c.JSON(fiber.Map{
				"success": true,
				"data":    session,
				"metadata": fiber.Map{
					"timestamp":  time.Now().UTC(),
					"request_id": c.Locals("requestid"),
				},
			})
		}
	}

	return c.Status(404).JSON(fiber.Map{
		"success": false,
		"error": fiber.Map{
			"code":    "SESSION_NOT_FOUND",
			"message": "Session not found",
		},
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// GetCurrentSession godoc
// @Summary Get current session info
// @Description Get information about the current active session
// @Tags Sessions
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Current session details"
// @Failure 402 {object} map[string]interface{} "Enterprise license required"
// @Router /api/v1/enterprise/sessions/current [get]
func (h *SessionHandler) GetCurrentSession(c *fiber.Ctx) error {
	now := time.Now().UTC()

	session := &enterprise.Session{
		ID:           "sess_current_001",
		UserID:       "user_001",
		UserEmail:    "current@example.com",
		CreatedAt:    now.Add(-2 * time.Hour),
		LastActivity: now,
		ExpiresAt:    now.Add(22 * time.Hour),
		IPAddress:    c.IP(),
		UserAgent:    c.Get("User-Agent"),
		DeviceType:   detectDeviceType(c.Get("User-Agent")),
		Status:       enterprise.SessionStatusActive,
		IsCurrent:    true,
		Location: &enterprise.GeoLocation{
			Country:     "South Korea",
			CountryCode: "KR",
			City:        "Seoul",
		},
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    session,
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// RevokeSession godoc
// @Summary Revoke specific session
// @Description Revoke/logout a specific session by ID
// @Tags Sessions
// @Accept json
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} map[string]interface{} "Session revoked confirmation"
// @Failure 404 {object} map[string]interface{} "Session not found"
// @Failure 402 {object} map[string]interface{} "Enterprise license required"
// @Router /api/v1/enterprise/sessions/{id} [delete]
func (h *SessionHandler) RevokeSession(c *fiber.Ctx) error {
	id := c.Params("id")

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"id":         id,
			"revoked":    true,
			"revoked_at": time.Now().UTC(),
		},
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// RevokeAllSessions godoc
// @Summary Revoke all sessions
// @Description Revoke all active sessions for the current user (logout everywhere)
// @Tags Sessions
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "All sessions revoked confirmation"
// @Failure 402 {object} map[string]interface{} "Enterprise license required"
// @Router /api/v1/enterprise/sessions [delete]
func (h *SessionHandler) RevokeAllSessions(c *fiber.Ctx) error {
	// In production, this would revoke all sessions except the current one
	// unless explicitly requested
	excludeCurrent := c.Query("exclude_current", "true") == "true"

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"revoked_count":   4,
			"exclude_current": excludeCurrent,
			"revoked_at":      time.Now().UTC(),
		},
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// RefreshSession godoc
// @Summary Refresh session
// @Description Refresh the current session's expiration time
// @Tags Sessions
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Session refreshed"
// @Failure 402 {object} map[string]interface{} "Enterprise license required"
// @Router /api/v1/enterprise/sessions/refresh [post]
func (h *SessionHandler) RefreshSession(c *fiber.Ctx) error {
	now := time.Now().UTC()
	config := enterprise.DefaultSessionConfig()

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"session_id":         "sess_current_001",
			"refreshed_at":       now,
			"new_expires_at":     now.Add(config.SessionTimeout),
			"expires_in_seconds": int64(config.SessionTimeout.Seconds()),
		},
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// Admin Endpoints

// AdminListSessions godoc
// @Summary List all active sessions (Admin)
// @Description Get a paginated list of all active sessions across all users
// @Tags Sessions (Admin)
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Param status query string false "Filter by status (active, expired, revoked)"
// @Param device_type query string false "Filter by device type"
// @Success 200 {object} map[string]interface{} "Paginated list of all sessions"
// @Failure 402 {object} map[string]interface{} "Enterprise license required"
// @Failure 403 {object} map[string]interface{} "Admin access required"
// @Router /api/v1/enterprise/admin/sessions [get]
func (h *SessionHandler) AdminListSessions(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))
	status := c.Query("status")
	deviceType := c.Query("device_type")

	if perPage > 100 {
		perPage = 100
	}

	// Mock all sessions
	sessions := getMockAllSessions()

	// Apply filters
	if status != "" {
		filtered := []*enterprise.Session{}
		for _, s := range sessions {
			if string(s.Status) == status {
				filtered = append(filtered, s)
			}
		}
		sessions = filtered
	}

	if deviceType != "" {
		filtered := []*enterprise.Session{}
		for _, s := range sessions {
			if s.DeviceType == deviceType {
				filtered = append(filtered, s)
			}
		}
		sessions = filtered
	}

	total := int64(len(sessions))
	pagination := enterprise.NewPagination(page, perPage, total)

	return c.JSON(fiber.Map{
		"success":    true,
		"data":       sessions,
		"pagination": pagination,
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// AdminGetUserSessions godoc
// @Summary Get user's sessions (Admin)
// @Description Get all sessions for a specific user
// @Tags Sessions (Admin)
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Success 200 {object} map[string]interface{} "Paginated user sessions"
// @Failure 402 {object} map[string]interface{} "Enterprise license required"
// @Failure 403 {object} map[string]interface{} "Admin access required"
// @Router /api/v1/enterprise/admin/sessions/user/{userId} [get]
func (h *SessionHandler) AdminGetUserSessions(c *fiber.Ctx) error {
	userID := c.Params("userId")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	if perPage > 100 {
		perPage = 100
	}

	sessions := getMockUserSessions(userID)
	total := int64(len(sessions))
	pagination := enterprise.NewPagination(page, perPage, total)

	return c.JSON(fiber.Map{
		"success":    true,
		"data":       sessions,
		"user_id":    userID,
		"pagination": pagination,
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// AdminForceLogoutUser godoc
// @Summary Force logout user (Admin)
// @Description Force logout a user by revoking all their sessions
// @Tags Sessions (Admin)
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {object} map[string]interface{} "User logged out confirmation"
// @Failure 402 {object} map[string]interface{} "Enterprise license required"
// @Failure 403 {object} map[string]interface{} "Admin access required"
// @Router /api/v1/enterprise/admin/sessions/user/{userId} [delete]
func (h *SessionHandler) AdminForceLogoutUser(c *fiber.Ctx) error {
	userID := c.Params("userId")
	reason := c.Query("reason", "admin_action")

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"user_id":          userID,
			"sessions_revoked": 3,
			"reason":           reason,
			"revoked_at":       time.Now().UTC(),
			"revoked_by":       "admin@example.com",
		},
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// AdminGetSuspiciousSessions godoc
// @Summary Get suspicious sessions (Admin)
// @Description Get list of sessions flagged as suspicious
// @Tags Sessions (Admin)
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Param risk_level query string false "Filter by risk level (low, medium, high, critical)"
// @Success 200 {object} map[string]interface{} "Paginated suspicious sessions"
// @Failure 402 {object} map[string]interface{} "Enterprise license required"
// @Failure 403 {object} map[string]interface{} "Admin access required"
// @Router /api/v1/enterprise/admin/sessions/suspicious [get]
func (h *SessionHandler) AdminGetSuspiciousSessions(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))
	riskLevel := c.Query("risk_level")

	if perPage > 100 {
		perPage = 100
	}

	suspicious := getMockSuspiciousSessions()

	if riskLevel != "" {
		filtered := []*enterprise.SuspiciousSession{}
		for _, s := range suspicious {
			if s.RiskLevel == riskLevel {
				filtered = append(filtered, s)
			}
		}
		suspicious = filtered
	}

	total := int64(len(suspicious))
	pagination := enterprise.NewPagination(page, perPage, total)

	return c.JSON(fiber.Map{
		"success":    true,
		"data":       suspicious,
		"pagination": pagination,
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// GetSessionStats godoc
// @Summary Get session statistics
// @Description Get statistics about active sessions
// @Tags Sessions (Admin)
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Session statistics"
// @Failure 402 {object} map[string]interface{} "Enterprise license required"
// @Router /api/v1/enterprise/admin/sessions/stats [get]
func (h *SessionHandler) GetSessionStats(c *fiber.Ctx) error {
	stats := &enterprise.SessionStats{
		TotalActiveSessions: 156,
		UniqueUsers:         47,
		SessionsByDevice: map[string]int64{
			"desktop": 89,
			"mobile":  42,
			"tablet":  15,
			"other":   10,
		},
		SessionsByCountry: map[string]int64{
			"KR":    78,
			"US":    45,
			"JP":    18,
			"DE":    10,
			"other": 5,
		},
		AverageSessionAge:    4.5,
		SuspiciousSessions:   3,
		SessionsCreatedToday: 24,
		SessionsExpiredToday: 18,
		SessionsRevokedToday: 2,
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    stats,
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// Helper functions

func detectDeviceType(userAgent string) string {
	// Simple device detection (in production, use a proper UA parser)
	if len(userAgent) == 0 {
		return "unknown"
	}

	// Very basic detection
	switch {
	case contains(userAgent, "Mobile") || contains(userAgent, "Android"):
		return "mobile"
	case contains(userAgent, "iPad") || contains(userAgent, "Tablet"):
		return "tablet"
	default:
		return "desktop"
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func getMockUserSessions(userID string) []*enterprise.Session {
	now := time.Now().UTC()
	return []*enterprise.Session{
		{
			ID:           fmt.Sprintf("sess_%s_001", userID[:4]),
			UserID:       userID,
			UserEmail:    fmt.Sprintf("%s@example.com", userID),
			CreatedAt:    now.Add(-2 * time.Hour),
			LastActivity: now.Add(-5 * time.Minute),
			ExpiresAt:    now.Add(22 * time.Hour),
			IPAddress:    "192.168.1.100",
			UserAgent:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0",
			DeviceType:   "desktop",
			Status:       enterprise.SessionStatusActive,
			IsCurrent:    true,
			Location: &enterprise.GeoLocation{
				Country:     "South Korea",
				CountryCode: "KR",
				City:        "Seoul",
			},
		},
		{
			ID:           fmt.Sprintf("sess_%s_002", userID[:4]),
			UserID:       userID,
			UserEmail:    fmt.Sprintf("%s@example.com", userID),
			CreatedAt:    now.Add(-24 * time.Hour),
			LastActivity: now.Add(-30 * time.Minute),
			ExpiresAt:    now.Add(-1 * time.Hour), // Expired
			IPAddress:    "10.0.0.50",
			UserAgent:    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0)",
			DeviceType:   "mobile",
			Status:       enterprise.SessionStatusActive,
			Location: &enterprise.GeoLocation{
				Country:     "South Korea",
				CountryCode: "KR",
				City:        "Busan",
			},
		},
		{
			ID:           fmt.Sprintf("sess_%s_003", userID[:4]),
			UserID:       userID,
			UserEmail:    fmt.Sprintf("%s@example.com", userID),
			CreatedAt:    now.Add(-48 * time.Hour),
			LastActivity: now.Add(-6 * time.Hour),
			ExpiresAt:    now.Add(18 * time.Hour),
			IPAddress:    "172.16.0.25",
			UserAgent:    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
			DeviceType:   "desktop",
			Status:       enterprise.SessionStatusActive,
			Location: &enterprise.GeoLocation{
				Country:     "Japan",
				CountryCode: "JP",
				City:        "Tokyo",
			},
		},
	}
}

func getMockAllSessions() []*enterprise.Session {
	now := time.Now().UTC()
	users := []string{"user_admin", "user_dev1", "user_dev2", "user_ops"}
	sessions := []*enterprise.Session{}

	for _, userID := range users {
		userSessions := getMockUserSessions(userID)
		sessions = append(sessions, userSessions...)
	}

	// Add some additional varied sessions
	sessions = append(sessions, &enterprise.Session{
		ID:           "sess_guest_001",
		UserID:       "user_guest",
		UserEmail:    "guest@example.com",
		CreatedAt:    now.Add(-1 * time.Hour),
		LastActivity: now.Add(-10 * time.Minute),
		ExpiresAt:    now.Add(23 * time.Hour),
		IPAddress:    "203.0.113.50",
		UserAgent:    "curl/7.64.1",
		DeviceType:   "other",
		Status:       enterprise.SessionStatusActive,
		Location: &enterprise.GeoLocation{
			Country:     "United States",
			CountryCode: "US",
		},
	})

	return sessions
}

func getMockSuspiciousSessions() []*enterprise.SuspiciousSession {
	now := time.Now().UTC()
	return []*enterprise.SuspiciousSession{
		{
			Session: &enterprise.Session{
				ID:           "sess_susp_001",
				UserID:       "user_dev1",
				UserEmail:    "dev1@example.com",
				CreatedAt:    now.Add(-30 * time.Minute),
				LastActivity: now.Add(-5 * time.Minute),
				ExpiresAt:    now.Add(23*time.Hour + 30*time.Minute),
				IPAddress:    "185.220.101.42",
				UserAgent:    "Mozilla/5.0 (compatible; MSIE 10.0)",
				DeviceType:   "desktop",
				Status:       enterprise.SessionStatusActive,
				Location: &enterprise.GeoLocation{
					Country:     "Germany",
					CountryCode: "DE",
					City:        "Frankfurt",
				},
			},
			Reason:     "Login from unusual location (Tor exit node detected)",
			RiskLevel:  "high",
			DetectedAt: now.Add(-25 * time.Minute),
			Indicators: []string{
				"IP address associated with Tor network",
				"Geographic location different from usual",
				"Outdated browser user agent",
			},
			Recommended: "Verify user identity and consider revoking session",
		},
		{
			Session: &enterprise.Session{
				ID:           "sess_susp_002",
				UserID:       "user_admin",
				UserEmail:    "admin@example.com",
				CreatedAt:    now.Add(-2 * time.Hour),
				LastActivity: now.Add(-1 * time.Hour),
				ExpiresAt:    now.Add(22 * time.Hour),
				IPAddress:    "45.33.32.156",
				UserAgent:    "python-requests/2.28.0",
				DeviceType:   "other",
				Status:       enterprise.SessionStatusActive,
				Location: &enterprise.GeoLocation{
					Country:     "United States",
					CountryCode: "US",
				},
			},
			Reason:     "Automated tool detected accessing admin account",
			RiskLevel:  "medium",
			DetectedAt: now.Add(-1*time.Hour - 45*time.Minute),
			Indicators: []string{
				"Non-browser user agent",
				"High request rate from this session",
				"Accessing sensitive admin endpoints",
			},
			Recommended: "Monitor activity and verify if legitimate automation",
		},
		{
			Session: &enterprise.Session{
				ID:           "sess_susp_003",
				UserID:       "user_new",
				UserEmail:    "newuser@example.com",
				CreatedAt:    now.Add(-10 * time.Minute),
				LastActivity: now.Add(-2 * time.Minute),
				ExpiresAt:    now.Add(23*time.Hour + 50*time.Minute),
				IPAddress:    "192.168.1.200",
				UserAgent:    "Mozilla/5.0 (Windows NT 10.0)",
				DeviceType:   "desktop",
				Status:       enterprise.SessionStatusActive,
				Location: &enterprise.GeoLocation{
					Country:     "South Korea",
					CountryCode: "KR",
				},
			},
			Reason:     "Multiple failed login attempts before success",
			RiskLevel:  "low",
			DetectedAt: now.Add(-8 * time.Minute),
			Indicators: []string{
				"5 failed login attempts in last 10 minutes",
				"Password reset requested recently",
			},
			Recommended: "Monitor for unusual activity",
		},
	}
}
