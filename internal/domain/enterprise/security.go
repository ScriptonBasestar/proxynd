package enterprise

import (
	"time"
)

// Vulnerability represents a detected security vulnerability
type Vulnerability struct {
	ID             string       `json:"id"`
	PackageName    string       `json:"package_name"`
	PackageVersion string       `json:"package_version"`
	PackageManager string       `json:"package_manager"`
	CVE            string       `json:"cve"`
	Severity       VulnSeverity `json:"severity"`
	Description    string       `json:"description"`
	PublishedDate  time.Time    `json:"published_date"`
	FixedVersion   string       `json:"fixed_version,omitempty"`
	DetectedAt     time.Time    `json:"detected_at"`
	Status         VulnStatus   `json:"status"`
}

// VulnSeverity represents vulnerability severity levels
type VulnSeverity string

const (
	VulnSeverityCritical VulnSeverity = "critical"
	VulnSeverityHigh     VulnSeverity = "high"
	VulnSeverityMedium   VulnSeverity = "medium"
	VulnSeverityLow      VulnSeverity = "low"
	VulnSeverityInfo     VulnSeverity = "info"
)

// VulnStatus represents vulnerability remediation status
type VulnStatus string

const (
	VulnStatusOpen      VulnStatus = "open"
	VulnStatusMitigated VulnStatus = "mitigated"
	VulnStatusFixed     VulnStatus = "fixed"
	VulnStatusIgnored   VulnStatus = "ignored"
)

// PackageLicense represents detected license in a package
type PackageLicense struct {
	ID             string    `json:"id"`
	PackageName    string    `json:"package_name"`
	PackageVersion string    `json:"package_version"`
	PackageManager string    `json:"package_manager"`
	LicenseType    string    `json:"license_type"` // MIT, GPL-3.0, Apache-2.0, etc.
	IsApproved     bool      `json:"is_approved"`
	DetectedAt     time.Time `json:"detected_at"`
}

// LicenseViolation represents a license policy violation
type LicenseViolation struct {
	ID             string    `json:"id"`
	PackageName    string    `json:"package_name"`
	PackageVersion string    `json:"package_version"`
	PackageManager string    `json:"package_manager"`
	LicenseType    string    `json:"license_type"`
	ViolationType  string    `json:"violation_type"` // blocked, restricted
	PolicyRule     string    `json:"policy_rule"`
	DetectedAt     time.Time `json:"detected_at"`
	Status         string    `json:"status"` // open, resolved, waived
}

// MalwareAlert represents a malware detection alert
type MalwareAlert struct {
	ID             string       `json:"id"`
	PackageName    string       `json:"package_name"`
	PackageVersion string       `json:"package_version"`
	PackageManager string       `json:"package_manager"`
	ThreatType     string       `json:"threat_type"` // trojan, backdoor, malicious_script
	Severity       VulnSeverity `json:"severity"`
	Description    string       `json:"description"`
	DetectedAt     time.Time    `json:"detected_at"`
	Quarantined    bool         `json:"quarantined"`
	ScanEngine     string       `json:"scan_engine"` // clamav, virustotal
}

// QuarantinedPackage represents a quarantined package
type QuarantinedPackage struct {
	ID             string    `json:"id"`
	PackageName    string    `json:"package_name"`
	PackageVersion string    `json:"package_version"`
	PackageManager string    `json:"package_manager"`
	Reason         string    `json:"reason"`
	QuarantinedAt  time.Time `json:"quarantined_at"`
	QuarantinedBy  string    `json:"quarantined_by"`
	CanRestore     bool      `json:"can_restore"`
}

// ScanJob represents a security scan job
type ScanJob struct {
	ID              string     `json:"id"`
	Type            string     `json:"type"` // vulnerability, license, malware
	Status          ScanStatus `json:"status"`
	StartedAt       time.Time  `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	PackagesScanned int        `json:"packages_scanned"`
	IssuesFound     int        `json:"issues_found"`
	Error           string     `json:"error,omitempty"`
}

// ScanStatus represents scan job status
type ScanStatus string

const (
	ScanStatusPending   ScanStatus = "pending"
	ScanStatusRunning   ScanStatus = "running"
	ScanStatusCompleted ScanStatus = "completed"
	ScanStatusFailed    ScanStatus = "failed"
)

// ScanRequest represents a request to start a security scan
type ScanRequest struct {
	Type           string   `json:"type" validate:"required,oneof=vulnerability license malware"`
	PackageManager string   `json:"package_manager,omitempty"`
	PackageNames   []string `json:"package_names,omitempty"`
	FullScan       bool     `json:"full_scan"`
}

// Security errors
var (
	ErrVulnNotFound    = &DomainError{Code: "VULN_NOT_FOUND", Message: "Vulnerability not found"}
	ErrInvalidScanType = &DomainError{Code: "INVALID_SCAN_TYPE", Message: "Invalid scan type specified"}
	ErrScanJobNotFound = &DomainError{Code: "SCAN_JOB_NOT_FOUND", Message: "Scan job not found"}
)
