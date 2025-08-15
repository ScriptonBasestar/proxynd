package ports

import (
	"context"
	"io"
	"time"
)

// PackageManager defines the main package manager operations
// TODO: Migrate from handlers/proxy/ and internal/services/
type PackageManager interface {
	// GetPackage retrieves a package from repository
	GetPackage(ctx context.Context, req *PackageRequest) (*PackageResponse, error)
	
	// ListPackages lists available packages in repository
	ListPackages(ctx context.Context, repo string) (*PackageListResponse, error)
	
	// GetMetadata retrieves package metadata
	GetMetadata(ctx context.Context, req *MetadataRequest) (*MetadataResponse, error)
	
	// UploadPackage uploads a package to repository
	UploadPackage(ctx context.Context, req *UploadRequest) error
	
	// DeletePackage removes a package from repository
	DeletePackage(ctx context.Context, req *DeleteRequest) error
}

// PackageRequest represents a package retrieval request
type PackageRequest struct {
	Repository string            `json:"repository"`
	Name       string            `json:"name"`
	Version    string            `json:"version"`
	Path       string            `json:"path"`
	Headers    map[string]string `json:"headers"`
	QueryParams map[string]string `json:"query_params"`
}

// PackageResponse represents a package response
type PackageResponse struct {
	Content     io.ReadCloser     `json:"-"`
	ContentType string            `json:"content_type"`
	Size        int64             `json:"size"`
	Headers     map[string]string `json:"headers"`
	Cached      bool              `json:"cached"`
	CacheKey    string            `json:"cache_key"`
}

// PackageListResponse represents package list response
type PackageListResponse struct {
	Packages []PackageInfo `json:"packages"`
	Total    int           `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

// PackageInfo represents package information
type PackageInfo struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Size        int64             `json:"size"`
	Checksum    string            `json:"checksum"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Metadata    map[string]string `json:"metadata"`
}

// MetadataRequest represents metadata request
type MetadataRequest struct {
	Repository string `json:"repository"`
	Name       string `json:"name"`
	Version    string `json:"version"`
}

// MetadataResponse represents metadata response
type MetadataResponse struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Dependencies []string          `json:"dependencies"`
	Metadata     map[string]string `json:"metadata"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// UploadRequest represents package upload request
type UploadRequest struct {
	Repository  string            `json:"repository"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Content     io.ReadCloser     `json:"-"`
	ContentType string            `json:"content_type"`
	Size        int64             `json:"size"`
	Checksum    string            `json:"checksum"`
	Metadata    map[string]string `json:"metadata"`
}

// DeleteRequest represents package deletion request
type DeleteRequest struct {
	Repository string `json:"repository"`
	Name       string `json:"name"`
	Version    string `json:"version"`
}

// TypedPackageManager defines type-specific package manager interface
// TODO: Migrate from handlers/proxy/ specific handlers
type TypedPackageManager interface {
	// GetType returns the package manager type
	GetType() string
	
	// ProcessRequest processes type-specific request
	ProcessRequest(ctx context.Context, req *TypedRequest) (*TypedResponse, error)
	
	// ValidateRequest validates type-specific request
	ValidateRequest(req *TypedRequest) error
}

// TypedRequest represents type-specific request
type TypedRequest struct {
	Type        string            `json:"type"` // npm, maven, apt, docker, etc.
	Operation   string            `json:"operation"`
	Path        string            `json:"path"`
	Headers     map[string]string `json:"headers"`
	QueryParams map[string]string `json:"query_params"`
	Body        []byte            `json:"body"`
}

// TypedResponse represents type-specific response
type TypedResponse struct {
	Content     io.ReadCloser     `json:"-"`
	ContentType string            `json:"content_type"`
	Headers     map[string]string `json:"headers"`
	StatusCode  int               `json:"status_code"`
	Cached      bool              `json:"cached"`
}

// PackageManagerFactory creates package managers
// TODO: Replace handlers/proxy/proxy_factory.go
type PackageManagerFactory interface {
	// CreateManager creates a package manager for given type
	CreateManager(pmType string) (TypedPackageManager, error)
	
	// GetSupportedTypes returns supported package manager types
	GetSupportedTypes() []string
	
	// RegisterManager registers a new package manager type
	RegisterManager(pmType string, creator func() TypedPackageManager) error
}

// RepositoryManager manages repository configurations
// TODO: Migrate from internal/config/
type RepositoryManager interface {
	// GetRepository gets repository configuration
	GetRepository(ctx context.Context, name string) (*RepositoryConfig, error)
	
	// ListRepositories lists all repositories
	ListRepositories(ctx context.Context) ([]*RepositoryConfig, error)
	
	// CreateRepository creates new repository
	CreateRepository(ctx context.Context, config *RepositoryConfig) error
	
	// UpdateRepository updates repository configuration
	UpdateRepository(ctx context.Context, name string, config *RepositoryConfig) error
	
	// DeleteRepository removes repository
	DeleteRepository(ctx context.Context, name string) error
}

// RepositoryConfig represents repository configuration
type RepositoryConfig struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	URL         string            `json:"url"`
	Enabled     bool              `json:"enabled"`
	CachePolicy string            `json:"cache_policy"`
	Auth        *AuthConfig       `json:"auth,omitempty"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Type     string `json:"type"`
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token"`
}