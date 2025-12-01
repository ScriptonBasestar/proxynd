package enterprise

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// LicenseHandler handles license management API requests
type LicenseHandler struct{}

// NewLicenseHandler creates a new license handler
func NewLicenseHandler() *LicenseHandler {
	return &LicenseHandler{}
}

// GetLicenseInfo godoc
// @Summary Get license information
// @Description Get detailed information about the current enterprise license
// @Tags License
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "License information"
// @Failure 402 {object} map[string]interface{} "Enterprise license required"
// @Router /api/v1/enterprise/license/info [get]
func (h *LicenseHandler) GetLicenseInfo(c *fiber.Ctx) error {
	info := fiber.Map{
		"license_type":   "enterprise",
		"company":        "Example Corp",
		"issued_to":      "admin@example.com",
		"issued_at":      time.Now().Add(-90 * 24 * time.Hour).UTC(),
		"expires_at":     time.Now().Add(275 * 24 * time.Hour).UTC(),
		"max_servers":    10,
		"active_servers": 3,
		"features_enabled": []string{
			"rbac",
			"audit_log",
			"advanced_analytics",
			"vulnerability_scanning",
			"multi_datacenter",
		},
		"is_valid":       true,
		"is_trial":       false,
		"days_remaining": 275,
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    info,
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// ValidateLicense godoc
// @Summary Validate license key
// @Description Validate and activate a license key
// @Tags License
// @Accept json
// @Produce json
// @Param license body object{license_key=string} true "License key to validate"
// @Success 200 {object} map[string]interface{} "Validation result"
// @Failure 400 {object} map[string]interface{} "Invalid license key"
// @Failure 402 {object} map[string]interface{} "Enterprise license required"
// @Router /api/v1/enterprise/license/validate [post]
func (h *LicenseHandler) ValidateLicense(c *fiber.Ctx) error {
	var req struct {
		LicenseKey string `json:"license_key"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
	}

	// Mock validation
	result := fiber.Map{
		"valid":        true,
		"license_type": "enterprise",
		"features": []string{
			"rbac",
			"audit_log",
			"advanced_analytics",
		},
		"expires_at": time.Now().Add(365 * 24 * time.Hour).UTC(),
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// ListFeatures godoc
// @Summary List available features
// @Description Get a list of all available enterprise features and their enablement status
// @Tags License
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Feature list with enablement status"
// @Failure 402 {object} map[string]interface{} "Enterprise license required"
// @Router /api/v1/enterprise/license/features [get]
func (h *LicenseHandler) ListFeatures(c *fiber.Ctx) error {
	features := []fiber.Map{
		{
			"id":          "rbac",
			"name":        "Role-Based Access Control",
			"description": "Advanced user permissions and role management",
			"category":    "authentication",
			"enabled":     true,
		},
		{
			"id":          "audit_log",
			"name":        "Audit Logging",
			"description": "Comprehensive audit trail for compliance",
			"category":    "compliance",
			"enabled":     true,
		},
		{
			"id":          "advanced_analytics",
			"name":        "Advanced Analytics",
			"description": "Detailed usage analytics and custom reports",
			"category":    "analytics",
			"enabled":     true,
		},
		{
			"id":          "vulnerability_scanning",
			"name":        "Vulnerability Scanning",
			"description": "Automated security vulnerability detection",
			"category":    "security",
			"enabled":     true,
		},
		{
			"id":          "multi_datacenter",
			"name":        "Multi-Datacenter Replication",
			"description": "Replicate cache across multiple data centers",
			"category":    "infrastructure",
			"enabled":     true,
		},
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    features,
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// GetUsageMetrics godoc
// @Summary Get license usage metrics
// @Description Get current license usage metrics and limits
// @Tags License
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "License usage metrics"
// @Failure 402 {object} map[string]interface{} "Enterprise license required"
// @Router /api/v1/enterprise/license/usage [get]
func (h *LicenseHandler) GetUsageMetrics(c *fiber.Ctx) error {
	usage := fiber.Map{
		"license_type":             "enterprise",
		"max_servers":              10,
		"active_servers":           3,
		"max_users":                100,
		"active_users":             47,
		"storage_limit_gb":         1000,
		"storage_used_gb":          234,
		"bandwidth_limit_gb_month": 10000,
		"bandwidth_used_gb_month":  3456,
		"features_in_use": []string{
			"rbac",
			"audit_log",
			"advanced_analytics",
		},
		"period_start": time.Now().Add(-30 * 24 * time.Hour).UTC(),
		"period_end":   time.Now().UTC(),
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    usage,
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}
