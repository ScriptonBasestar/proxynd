package enterprise

import (
	"github.com/gofiber/fiber/v2"
	"proxynd/internal/domain/enterprise"
	"strconv"
	"time"
)

// AlertsHandler handles alerts and notifications API requests
type AlertsHandler struct{}

// NewAlertsHandler creates a new alerts handler
func NewAlertsHandler() *AlertsHandler {
	return &AlertsHandler{}
}

// ListAlerts handles GET /api/v1/enterprise/alerts
func (h *AlertsHandler) ListAlerts(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))
	status := c.Query("status")
	severity := c.Query("severity")

	alerts := getMockAlerts()

	// Apply filters
	if status != "" {
		filtered := make([]*enterprise.Alert, 0)
		for _, alert := range alerts {
			if string(alert.Status) == status {
				filtered = append(filtered, alert)
			}
		}
		alerts = filtered
	}

	if severity != "" {
		filtered := make([]*enterprise.Alert, 0)
		for _, alert := range alerts {
			if string(alert.Severity) == severity {
				filtered = append(filtered, alert)
			}
		}
		alerts = filtered
	}

	total := int64(len(alerts))
	pagination := enterprise.NewPagination(page, perPage, total)

	return c.JSON(fiber.Map{
		"success":    true,
		"data":       alerts[:min(len(alerts), perPage)],
		"pagination": pagination,
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// GetAlert handles GET /api/v1/enterprise/alerts/:id
func (h *AlertsHandler) GetAlert(c *fiber.Ctx) error {
	id := c.Params("id")

	alerts := getMockAlerts()
	for _, alert := range alerts {
		if alert.ID == id {
			return c.JSON(fiber.Map{
				"success": true,
				"data":    alert,
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
			"code":    "ALERT_NOT_FOUND",
			"message": "Alert not found",
		},
	})
}

// CreateAlertRule handles POST /api/v1/enterprise/alerts/rules
func (h *AlertsHandler) CreateAlertRule(c *fiber.Ctx) error {
	var req enterprise.AlertRuleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
	}

	rule := &enterprise.AlertRule{
		ID:          "rule_" + strconv.FormatInt(time.Now().UnixNano(), 36),
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Enabled:     req.Enabled,
		Severity:    req.Severity,
		Conditions:  req.Conditions,
		Actions:     req.Actions,
		CreatedBy:   "current_user@example.com",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	return c.Status(201).JSON(fiber.Map{
		"success": true,
		"data":    rule,
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// UpdateAlertRule handles PUT /api/v1/enterprise/alerts/rules/:id
func (h *AlertsHandler) UpdateAlertRule(c *fiber.Ctx) error {
	id := c.Params("id")

	var req enterprise.AlertRuleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
	}

	rule := &enterprise.AlertRule{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Enabled:     req.Enabled,
		Severity:    req.Severity,
		Conditions:  req.Conditions,
		Actions:     req.Actions,
		UpdatedAt:   time.Now().UTC(),
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    rule,
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// DeleteAlertRule handles DELETE /api/v1/enterprise/alerts/rules/:id
func (h *AlertsHandler) DeleteAlertRule(c *fiber.Ctx) error {
	id := c.Params("id")

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"id":      id,
			"deleted": true,
		},
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// Mock data generator
func getMockAlerts() []*enterprise.Alert {
	now := time.Now().UTC()
	return []*enterprise.Alert{
		{
			ID:          "alert_001",
			Type:        "security",
			Severity:    enterprise.AlertSeverityCritical,
			Title:       "Critical Vulnerability Detected",
			Description: "CVE-2021-22118 found in spring-core 5.2.0",
			Source:      "vulnerability_scanner",
			TriggeredAt: now.Add(-2 * time.Hour),
			Status:      enterprise.AlertStatusActive,
			Metadata: map[string]interface{}{
				"cve":            "CVE-2021-22118",
				"package":        "spring-core",
				"package_version": "5.2.0",
			},
		},
		{
			ID:          "alert_002",
			Type:        "performance",
			Severity:    enterprise.AlertSeverityHigh,
			Title:       "High Response Time Detected",
			Description: "API response time exceeded 1000ms",
			Source:      "performance_monitor",
			TriggeredAt: now.Add(-1 * time.Hour),
			Status:      enterprise.AlertStatusAcknowledged,
			AckedBy:     "ops@example.com",
			Metadata: map[string]interface{}{
				"endpoint":      "/maven/*",
				"response_time": 1245.5,
			},
		},
		{
			ID:          "alert_003",
			Type:        "capacity",
			Severity:    enterprise.AlertSeverityMedium,
			Title:       "Cache Storage 80% Full",
			Description: "Cache storage usage has reached 80% capacity",
			Source:      "capacity_monitor",
			TriggeredAt: now.Add(-30 * time.Minute),
			Status:      enterprise.AlertStatusActive,
			Metadata: map[string]interface{}{
				"current_usage": "8GB",
				"total_capacity": "10GB",
				"percentage":    80,
			},
		},
	}
}
