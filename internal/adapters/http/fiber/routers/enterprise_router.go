package routers

import (
	"github.com/gofiber/fiber/v2"
	"proxynd/internal/adapters/http/fiber/handlers/enterprise"
	enterpriseMiddleware "proxynd/internal/adapters/http/fiber/middleware/enterprise"
)

// SetupEnterpriseRoutes sets up all enterprise API routes
// All enterprise routes require valid enterprise license validation
// enterpriseFeatures parameter can be nil in development mode
func SetupEnterpriseRoutes(app *fiber.App, enterpriseFeatures enterpriseMiddleware.EnterpriseFeatures) {
	// Create handlers
	rbacHandler := enterprise.NewRBACHandler()
	auditHandler := enterprise.NewAuditHandler()
	analyticsHandler := enterprise.NewAnalyticsHandler()
	securityHandler := enterprise.NewSecurityHandler()
	alertsHandler := enterprise.NewAlertsHandler()
	licenseHandler := enterprise.NewLicenseHandler()

	// Create Enterprise API metrics
	enterpriseMetrics := enterpriseMiddleware.NewEnterpriseMetrics()

	// Enterprise API group
	api := app.Group("/api/v1/enterprise")

	// Metrics middleware (must be first to capture all requests)
	api.Use(enterpriseMiddleware.MetricsMiddleware(enterpriseMetrics))

	// License validation middleware
	// enterpriseFeatures is passed from app initialization
	// In development mode, it will be a mock that allows all access
	if enterpriseFeatures == nil {
		// If nil, use dev mode middleware (always allows access)
		licMw := enterpriseMiddleware.NewDevModeLicenseMiddleware()
		api.Use(licMw.RequireEnterprise())
	} else {
		// Production mode: require valid enterprise license
		licMw := enterpriseMiddleware.NewLicenseMiddleware(enterpriseFeatures)
		api.Use(licMw.RequireEnterprise())
	}

	// RBAC Routes (12 endpoints)
	setupRBACRoutes(api, rbacHandler)

	// Audit Routes (8 endpoints)
	setupAuditRoutes(api, auditHandler)

	// Analytics Routes (10 endpoints)
	setupAnalyticsRoutes(api, analyticsHandler)

	// Security Routes (8 endpoints)
	setupSecurityRoutes(api, securityHandler)

	// Alerts Routes (5 endpoints)
	setupAlertsRoutes(api, alertsHandler)

	// License Routes (4 endpoints)
	setupLicenseRoutes(api, licenseHandler)
}

// setupRBACRoutes configures RBAC endpoints
func setupRBACRoutes(api fiber.Router, h *enterprise.RBACHandler) {
	rbac := api.Group("/rbac")

	// Roles Management
	rbac.Get("/roles", h.ListRoles)         // List all roles
	rbac.Get("/roles/:id", h.GetRole)       // Get role details
	rbac.Post("/roles", h.CreateRole)       // Create new role
	rbac.Put("/roles/:id", h.UpdateRole)    // Update role
	rbac.Delete("/roles/:id", h.DeleteRole) // Delete role

	// Permissions Management
	rbac.Get("/permissions", h.ListPermissions)                           // List all permissions
	rbac.Get("/roles/:id/permissions", h.GetRolePermissions)              // Get role permissions
	rbac.Post("/roles/:id/permissions", h.AssignPermissions)              // Assign permissions
	rbac.Delete("/roles/:id/permissions/:permission", h.RevokePermission) // Revoke permission

	// User-Role Assignments
	rbac.Get("/users/:userId/roles", h.GetUserRoles)              // Get user's roles
	rbac.Post("/users/:userId/roles", h.AssignUserRole)           // Assign role to user
	rbac.Delete("/users/:userId/roles/:roleId", h.RemoveUserRole) // Remove role from user
}

// setupAuditRoutes configures audit logging endpoints
func setupAuditRoutes(api fiber.Router, h *enterprise.AuditHandler) {
	audit := api.Group("/audit")

	audit.Get("/events", h.ListEvents)                                     // List audit events
	audit.Get("/events/search", h.SearchEvents)                            // Advanced search (must be before :id)
	audit.Get("/events/:id", h.GetEvent)                                   // Get audit event details
	audit.Get("/users/:userId/events", h.GetUserEvents)                    // Get user's audit trail
	audit.Get("/resources/:resourceType/:resourceId", h.GetResourceEvents) // Get resource audit history
	audit.Post("/export", h.ExportEvents)                                  // Export audit logs
	audit.Get("/stats", h.GetStats)                                        // Audit statistics
	audit.Get("/compliance/report", h.GetComplianceReport)                 // Compliance report
}

// setupAnalyticsRoutes configures analytics and reporting endpoints
func setupAnalyticsRoutes(api fiber.Router, h *enterprise.AnalyticsHandler) {
	analytics := api.Group("/analytics")

	// Metrics & Statistics
	analytics.Get("/overview", h.GetOverview)                // Dashboard overview
	analytics.Get("/usage", h.GetUsageStats)                 // Usage statistics
	analytics.Get("/performance", h.GetPerformanceMetrics)   // Performance metrics
	analytics.Get("/cache-efficiency", h.GetCacheEfficiency) // Cache efficiency

	// Custom Reports
	analytics.Get("/reports", h.ListReports)         // List saved reports
	analytics.Post("/reports", h.CreateReport)       // Create custom report
	analytics.Get("/reports/:id", h.GetReport)       // Get report data
	analytics.Delete("/reports/:id", h.DeleteReport) // Delete report

	// Cost Analysis
	analytics.Get("/costs", h.GetCostAnalysis) // Cost breakdown
	analytics.Get("/trends", h.GetTrends)      // Trend analysis
}

// setupSecurityRoutes configures security scanning endpoints
func setupSecurityRoutes(api fiber.Router, h *enterprise.SecurityHandler) {
	security := api.Group("/security")

	// Vulnerability Scanning
	security.Get("/vulnerabilities", h.ListVulnerabilities)  // List vulnerabilities
	security.Get("/vulnerabilities/:id", h.GetVulnerability) // Get vulnerability details
	security.Post("/scan", h.TriggerScan)                    // Trigger manual scan
	security.Get("/scan/:jobId", h.GetScanJob)               // Get scan job status

	// License Scanning
	security.Get("/licenses", h.ListLicenses)                     // List detected licenses
	security.Get("/licenses/violations", h.ListLicenseViolations) // License violations

	// Malware Detection
	security.Get("/malware/alerts", h.ListMalwareAlerts)           // List malware alerts
	security.Get("/malware/quarantine", h.ListQuarantinedPackages) // Quarantined packages
}

// setupAlertsRoutes configures alerts and notifications endpoints
func setupAlertsRoutes(api fiber.Router, h *enterprise.AlertsHandler) {
	alerts := api.Group("/alerts")

	alerts.Get("", h.ListAlerts)                   // List active alerts
	alerts.Get("/:id", h.GetAlert)                 // Get alert details
	alerts.Post("/rules", h.CreateAlertRule)       // Create alert rule
	alerts.Put("/rules/:id", h.UpdateAlertRule)    // Update alert rule
	alerts.Delete("/rules/:id", h.DeleteAlertRule) // Delete alert rule
}

// setupLicenseRoutes configures license management endpoints
func setupLicenseRoutes(api fiber.Router, h *enterprise.LicenseHandler) {
	license := api.Group("/license")

	license.Get("/info", h.GetLicenseInfo)       // Get license information
	license.Post("/validate", h.ValidateLicense) // Validate license key
	license.Get("/features", h.ListFeatures)     // List available features
	license.Get("/usage", h.GetUsageMetrics)     // License usage metrics
}
