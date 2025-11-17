package enterprise

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"proxynd/internal/domain/enterprise"
	"strconv"
	"time"
)

// SecurityHandler handles security scanning API requests
type SecurityHandler struct{}

// NewSecurityHandler creates a new security handler
func NewSecurityHandler() *SecurityHandler {
	return &SecurityHandler{}
}

// ListVulnerabilities handles GET /api/v1/enterprise/security/vulnerabilities
func (h *SecurityHandler) ListVulnerabilities(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))
	severity := c.Query("severity")

	vulns := getMockVulnerabilities()

	// Filter by severity if provided
	if severity != "" {
		filtered := make([]*enterprise.Vulnerability, 0)
		for _, v := range vulns {
			if string(v.Severity) == severity {
				filtered = append(filtered, v)
			}
		}
		vulns = filtered
	}

	total := int64(len(vulns))
	pagination := enterprise.NewPagination(page, perPage, total)

	return c.JSON(fiber.Map{
		"success":    true,
		"data":       vulns[:min(len(vulns), perPage)],
		"pagination": pagination,
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// GetVulnerability handles GET /api/v1/enterprise/security/vulnerabilities/:id
func (h *SecurityHandler) GetVulnerability(c *fiber.Ctx) error {
	id := c.Params("id")

	vulns := getMockVulnerabilities()
	for _, vuln := range vulns {
		if vuln.ID == id {
			return c.JSON(fiber.Map{
				"success": true,
				"data":    vuln,
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
			"code":    "VULN_NOT_FOUND",
			"message": "Vulnerability not found",
		},
	})
}

// TriggerScan handles POST /api/v1/enterprise/security/scan
func (h *SecurityHandler) TriggerScan(c *fiber.Ctx) error {
	var req enterprise.ScanRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
	}

	job := &enterprise.ScanJob{
		ID:              fmt.Sprintf("scan_%d", time.Now().Unix()),
		Type:            req.Type,
		Status:          enterprise.ScanStatusPending,
		StartedAt:       time.Now().UTC(),
		PackagesScanned: 0,
		IssuesFound:     0,
	}

	return c.Status(202).JSON(fiber.Map{
		"success": true,
		"data":    job,
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// GetScanJob handles GET /api/v1/enterprise/security/scan/:jobId
func (h *SecurityHandler) GetScanJob(c *fiber.Ctx) error {
	jobID := c.Params("jobId")

	completedAt := time.Now().UTC()
	job := &enterprise.ScanJob{
		ID:              jobID,
		Type:            "vulnerability",
		Status:          enterprise.ScanStatusCompleted,
		StartedAt:       time.Now().Add(-30 * time.Minute).UTC(),
		CompletedAt:     &completedAt,
		PackagesScanned: 1247,
		IssuesFound:     23,
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    job,
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// ListLicenses handles GET /api/v1/enterprise/security/licenses
func (h *SecurityHandler) ListLicenses(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	licenses := getMockLicenses()
	total := int64(len(licenses))
	pagination := enterprise.NewPagination(page, perPage, total)

	return c.JSON(fiber.Map{
		"success":    true,
		"data":       licenses[:min(len(licenses), perPage)],
		"pagination": pagination,
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// ListLicenseViolations handles GET /api/v1/enterprise/security/licenses/violations
func (h *SecurityHandler) ListLicenseViolations(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	violations := getMockLicenseViolations()
	total := int64(len(violations))
	pagination := enterprise.NewPagination(page, perPage, total)

	return c.JSON(fiber.Map{
		"success":    true,
		"data":       violations[:min(len(violations), perPage)],
		"pagination": pagination,
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// ListMalwareAlerts handles GET /api/v1/enterprise/security/malware/alerts
func (h *SecurityHandler) ListMalwareAlerts(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	alerts := getMockMalwareAlerts()
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

// ListQuarantinedPackages handles GET /api/v1/enterprise/security/malware/quarantine
func (h *SecurityHandler) ListQuarantinedPackages(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	packages := getMockQuarantinedPackages()
	total := int64(len(packages))
	pagination := enterprise.NewPagination(page, perPage, total)

	return c.JSON(fiber.Map{
		"success":    true,
		"data":       packages[:min(len(packages), perPage)],
		"pagination": pagination,
		"metadata": fiber.Map{
			"timestamp":  time.Now().UTC(),
			"request_id": c.Locals("requestid"),
		},
	})
}

// Mock data generators
func getMockVulnerabilities() []*enterprise.Vulnerability {
	now := time.Now().UTC()
	return []*enterprise.Vulnerability{
		{
			ID:             "vuln_001",
			PackageName:    "lodash",
			PackageVersion: "4.17.15",
			PackageManager: "npm",
			CVE:            "CVE-2020-8203",
			Severity:       enterprise.VulnSeverityHigh,
			Description:    "Prototype Pollution vulnerability",
			PublishedDate:  now.Add(-180 * 24 * time.Hour),
			FixedVersion:   "4.17.19",
			DetectedAt:     now.Add(-2 * 24 * time.Hour),
			Status:         enterprise.VulnStatusOpen,
		},
		{
			ID:             "vuln_002",
			PackageName:    "spring-core",
			PackageVersion: "5.2.0",
			PackageManager: "maven",
			CVE:            "CVE-2021-22118",
			Severity:       enterprise.VulnSeverityCritical,
			Description:    "Denial of Service vulnerability",
			PublishedDate:  now.Add(-90 * 24 * time.Hour),
			FixedVersion:   "5.2.9",
			DetectedAt:     now.Add(-1 * 24 * time.Hour),
			Status:         enterprise.VulnStatusOpen,
		},
		{
			ID:             "vuln_003",
			PackageName:    "requests",
			PackageVersion: "2.25.0",
			PackageManager: "pypi",
			CVE:            "CVE-2021-33503",
			Severity:       enterprise.VulnSeverityMedium,
			Description:    "Header injection vulnerability",
			PublishedDate:  now.Add(-60 * 24 * time.Hour),
			FixedVersion:   "2.25.1",
			DetectedAt:     now.Add(-5 * time.Hour),
			Status:         enterprise.VulnStatusMitigated,
		},
	}
}

func getMockLicenses() []*enterprise.PackageLicense {
	now := time.Now().UTC()
	return []*enterprise.PackageLicense{
		{
			ID:             "lic_001",
			PackageName:    "react",
			PackageVersion: "18.2.0",
			PackageManager: "npm",
			LicenseType:    "MIT",
			IsApproved:     true,
			DetectedAt:     now.Add(-10 * 24 * time.Hour),
		},
		{
			ID:             "lic_002",
			PackageName:    "spring-boot",
			PackageVersion: "2.7.0",
			PackageManager: "maven",
			LicenseType:    "Apache-2.0",
			IsApproved:     true,
			DetectedAt:     now.Add(-5 * 24 * time.Hour),
		},
	}
}

func getMockLicenseViolations() []*enterprise.LicenseViolation {
	now := time.Now().UTC()
	return []*enterprise.LicenseViolation{
		{
			ID:             "viol_001",
			PackageName:    "some-gpl-package",
			PackageVersion: "1.0.0",
			PackageManager: "npm",
			LicenseType:    "GPL-3.0",
			ViolationType:  "blocked",
			PolicyRule:     "No GPL licenses allowed",
			DetectedAt:     now.Add(-2 * 24 * time.Hour),
			Status:         "open",
		},
	}
}

func getMockMalwareAlerts() []*enterprise.MalwareAlert {
	now := time.Now().UTC()
	return []*enterprise.MalwareAlert{
		{
			ID:             "mal_001",
			PackageName:    "suspicious-package",
			PackageVersion: "1.0.0",
			PackageManager: "npm",
			ThreatType:     "backdoor",
			Severity:       enterprise.VulnSeverityCritical,
			Description:    "Package contains suspicious code execution",
			DetectedAt:     now.Add(-1 * time.Hour),
			Quarantined:    true,
			ScanEngine:     "clamav",
		},
	}
}

func getMockQuarantinedPackages() []*enterprise.QuarantinedPackage {
	now := time.Now().UTC()
	return []*enterprise.QuarantinedPackage{
		{
			ID:             "quar_001",
			PackageName:    "suspicious-package",
			PackageVersion: "1.0.0",
			PackageManager: "npm",
			Reason:         "Malware detected: backdoor",
			QuarantinedAt:  now.Add(-1 * time.Hour),
			QuarantinedBy:  "security_scanner",
			CanRestore:     false,
		},
	}
}
