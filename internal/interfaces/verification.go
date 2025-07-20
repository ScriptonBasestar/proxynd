package interfaces

import (
	"context"
)

// PackageVerifier defines the interface for package verification
type PackageVerifier interface {
	// VerifyPackage verifies a package based on its type
	VerifyPackage(
		ctx context.Context, packageType string, packagePath string,
		data []byte, metadata map[string]string,
	) (*VerificationResult, error)

	// SetStrictMode enables or disables strict verification mode
	SetStrictMode(strict bool)

	// IsStrictMode returns whether strict mode is enabled
	IsStrictMode() bool

	// RegisterVerifier registers a custom verifier for a package type
	RegisterVerifier(packageType string, verifier TypedPackageVerifier) error

	// GetSupportedTypes returns list of supported package types
	GetSupportedTypes() []string
}

// TypedPackageVerifier defines the interface for type-specific package verifiers
type TypedPackageVerifier interface {
	// Verify performs the verification
	Verify(ctx context.Context, packagePath string, data []byte, metadata map[string]string) (*VerificationResult, error)

	// Type returns the package type this verifier handles
	Type() string
}

// VerificationResult represents the result of package verification
type VerificationResult struct {
	Valid       bool                   `json:"valid"`
	PackageType string                 `json:"package_type"`
	PackageName string                 `json:"package_name"`
	Version     string                 `json:"version"`
	Issues      []VerificationIssue    `json:"issues,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Timestamp   int64                  `json:"timestamp"`
}

// VerificationIssue represents an issue found during verification
type VerificationIssue struct {
	Severity   IssueSeverity `json:"severity"`
	Code       string        `json:"code"`
	Message    string        `json:"message"`
	Details    string        `json:"details,omitempty"`
	Suggestion string        `json:"suggestion,omitempty"`
}

// IssueSeverity represents the severity of a verification issue
type IssueSeverity string

const (
	// SeverityError indicates a critical verification error
	SeverityError IssueSeverity = "error"
	// SeverityWarning indicates a verification warning
	SeverityWarning IssueSeverity = "warning"
	// SeverityInfo indicates informational verification details
	SeverityInfo IssueSeverity = "info"
)
