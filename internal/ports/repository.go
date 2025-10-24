package ports

import (
	"context"
	"io"
	"time"
)

// RepositoryMode defines the operation mode of a repository
type RepositoryMode string

const (
	// ModeProxy forwards requests to upstream registry (기존 기능)
	ModeProxy RepositoryMode = "proxy"

	// ModeHosted serves packages from local storage (신규 기능)
	ModeHosted RepositoryMode = "hosted"

	// ModeHybrid tries hosted first, then falls back to proxy
	ModeHybrid RepositoryMode = "hybrid"
)

// HostedRepositoryConfig extends RepositoryConfig with hosted-specific settings
// Note: RepositoryConfig is defined in pm.go
type HostedRepositoryConfig struct {
	// Basic config (same as RepositoryConfig in pm.go)
	Name    string         `json:"name" yaml:"name"`
	Type    string         `json:"type" yaml:"type"` // ansible, npm, pypi, etc.
	Mode    RepositoryMode `json:"mode" yaml:"mode"`
	Enabled bool           `json:"enabled" yaml:"enabled"`

	// Proxy mode settings
	Upstream string `json:"upstream,omitempty" yaml:"upstream,omitempty"`

	// Hosted mode settings
	AllowUpload    bool   `json:"allow_upload,omitempty" yaml:"allow_upload,omitempty"`
	AuthRequired   bool   `json:"auth_required,omitempty" yaml:"auth_required,omitempty"`
	StorageBackend string `json:"storage_backend,omitempty" yaml:"storage_backend,omitempty"`

	// Hybrid mode settings
	HostedPriority bool `json:"hosted_priority,omitempty" yaml:"hosted_priority,omitempty"`

	Metadata  map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time         `json:"updated_at" yaml:"updated_at"`
}

// HostedRepository defines operations for hosted (private) repositories
type HostedRepository interface {
	// StorePackage stores a package in the hosted repository
	StorePackage(ctx context.Context, req *StorePackageRequest) (*StorePackageResponse, error)

	// GetPackage retrieves a package from the hosted repository
	GetPackage(ctx context.Context, req *GetPackageRequest) (*GetPackageResponse, error)

	// DeletePackage removes a package from the hosted repository
	DeletePackage(ctx context.Context, req *DeletePackageRequest) error

	// ListPackages lists packages in the hosted repository
	ListPackages(ctx context.Context, req *ListPackagesRequest) (*ListPackagesResponse, error)

	// GetMetadata retrieves package metadata without content
	GetMetadata(ctx context.Context, req *GetMetadataRequest) (*PackageMetadata, error)
}

// StorePackageRequest represents a package storage request
type StorePackageRequest struct {
	Repository  string            `json:"repository"`
	PackageType string            `json:"package_type"` // ansible, pypi, npm, etc.
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Content     io.Reader         `json:"-"`
	ContentType string            `json:"content_type"`
	Size        int64             `json:"size"`
	SHA256      string            `json:"sha256"` // Pre-calculated or empty
	Metadata    map[string]string `json:"metadata"`
	UploadedBy  string            `json:"uploaded_by"`
}

// StorePackageResponse represents a package storage response
type StorePackageResponse struct {
	SHA256      string    `json:"sha256"`
	Size        int64     `json:"size"`
	StoredAt    time.Time `json:"stored_at"`
	PackagePath string    `json:"package_path"` // Virtual path for retrieval
}

// GetPackageRequest represents a package retrieval request
type GetPackageRequest struct {
	Repository  string `json:"repository"`
	PackageType string `json:"package_type"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Path        string `json:"path"` // Alternative: full path
}

// GetPackageResponse represents a package retrieval response
type GetPackageResponse struct {
	Content     io.ReadCloser    `json:"-"`
	ContentType string           `json:"content_type"`
	Size        int64            `json:"size"`
	SHA256      string           `json:"sha256"`
	Metadata    *PackageMetadata `json:"metadata"`
}

// DeletePackageRequest represents a package deletion request
type DeletePackageRequest struct {
	Repository  string `json:"repository"`
	PackageType string `json:"package_type"`
	Name        string `json:"name"`
	Version     string `json:"version"`
}

// ListPackagesRequest represents a package listing request
type ListPackagesRequest struct {
	Repository  string            `json:"repository"`
	PackageType string            `json:"package_type"`
	Filters     map[string]string `json:"filters"`
	Page        int               `json:"page"`
	PageSize    int               `json:"page_size"`
	OrderBy     string            `json:"order_by"`
}

// ListPackagesResponse represents a package listing response
type ListPackagesResponse struct {
	Packages   []*PackageMetadata `json:"packages"`
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalPages int                `json:"total_pages"`
}

// GetMetadataRequest represents a metadata retrieval request
type GetMetadataRequest struct {
	Repository  string `json:"repository"`
	PackageType string `json:"package_type"`
	Name        string `json:"name"`
	Version     string `json:"version"` // Optional: all versions if empty
}

// HostedRepositoryManager manages hosted repository configurations
// Note: Basic RepositoryManager is defined in pm.go
type HostedRepositoryManager interface {
	// GetHostedRepository retrieves hosted repository configuration
	GetHostedRepository(ctx context.Context, name string) (*HostedRepositoryConfig, error)

	// ListHostedRepositories lists all hosted repositories
	ListHostedRepositories(ctx context.Context, filters map[string]string) ([]*HostedRepositoryConfig, error)

	// CreateHostedRepository creates a new hosted repository
	CreateHostedRepository(ctx context.Context, config *HostedRepositoryConfig) error

	// UpdateHostedRepository updates hosted repository configuration
	UpdateHostedRepository(ctx context.Context, name string, config *HostedRepositoryConfig) error

	// DeleteHostedRepository removes a hosted repository
	DeleteHostedRepository(ctx context.Context, name string) error
}
