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
	Repository  string            `json:"repository"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Path        string            `json:"path"`
	Headers     map[string]string `json:"headers"`
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

// ProxyService defines the high-level proxy service interface
// Orchestrates package manager drivers with caching and configuration
type ProxyService interface {
	// HandleRequest processes a high-level proxy request
	HandleRequest(ctx context.Context, req *ProxyRequest) (*ProxyResponse, error)

	// GetDriver returns the underlying package manager driver
	GetDriver() PackageManagerDriver

	// GetCache returns the cache service
	GetCache() CacheService

	// GetType returns the proxy service type
	GetType() string

	// IsEnabled returns if the service is enabled
	IsEnabled() bool
}

// PackageManagerDriver defines the contract for package manager implementations
// Each package manager (maven/npm/apt/etc.) must implement this interface
type PackageManagerDriver interface {
	// Basic information
	Type() string
	IsSupported(path string) bool

	// Request processing
	NormalizePath(path string) (string, error)
	BuildUpstreamURL(req *DriverRequest) (string, error)
	FetchPackage(ctx context.Context, req *DriverRequest) (*DriverResponse, error)

	// Metadata handling
	ParseMetadata(content []byte) (*PackageMetadata, error)
	ValidateSignature(content, signature []byte) error

	// Caching behavior
	GetCacheKey(req *DriverRequest) string
	ShouldCache(resp *DriverResponse) bool
	GetCacheTTL(resp *DriverResponse) time.Duration

	// Configuration
	LoadConfig() error
	ValidateConfig() error
}

// DriverRequest represents a driver-level request
type DriverRequest struct {
	Path           string            `json:"path"`
	Method         string            `json:"method"`
	Headers        map[string]string `json:"headers"`
	QueryParams    map[string]string `json:"query_params"`
	RemoteAddr     string            `json:"remote_addr"`
	UserAgent      string            `json:"user_agent"`
	Repository     string            `json:"repository"`
	Authentication *AuthConfig       `json:"auth,omitempty"`
}

// DriverResponse represents a driver-level response
type DriverResponse struct {
	Content      io.ReadCloser     `json:"-"`
	ContentType  string            `json:"content_type"`
	Size         int64             `json:"size"`
	Headers      map[string]string `json:"headers"`
	StatusCode   int               `json:"status_code"`
	Checksum     string            `json:"checksum"`
	LastModified time.Time         `json:"last_modified"`
	Metadata     *PackageMetadata  `json:"metadata,omitempty"`
}

// PackageMetadata represents package metadata
type PackageMetadata struct {
	Namespace    string            `json:"namespace,omitempty"` // For Ansible collections
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	License      string            `json:"license"`
	Author       string            `json:"author"`
	Homepage     string            `json:"homepage"`
	Dependencies []string          `json:"dependencies"`
	Keywords     []string          `json:"keywords"`
	Deprecated   bool              `json:"deprecated,omitempty"` // For Ansible collections
	Attributes   map[string]string `json:"attributes"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// ProxyRequest represents a high-level proxy request
type ProxyRequest struct {
	Type        string            `json:"type"`         // maven, npm, apt, etc.
	Path        string            `json:"path"`         // request path
	Method      string            `json:"method"`       // HTTP method
	Headers     map[string]string `json:"headers"`      // HTTP headers
	QueryParams map[string]string `json:"query_params"` // query parameters
	RemoteAddr  string            `json:"remote_addr"`  // client IP
	UserAgent   string            `json:"user_agent"`   // user agent
	Repository  string            `json:"repository"`   // target repository
}

// ProxyResponse represents a high-level proxy response
type ProxyResponse struct {
	Content     io.ReadCloser     `json:"-"`
	ContentType string            `json:"content_type"`
	Size        int64             `json:"size"`
	Headers     map[string]string `json:"headers"`
	StatusCode  int               `json:"status_code"`
	Cached      bool              `json:"cached"`
	CacheKey    string            `json:"cache_key"`
	TTL         time.Duration     `json:"ttl"`
	Metadata    *PackageMetadata  `json:"metadata,omitempty"`
}

// CacheService defines caching operations for proxy services
type CacheService interface {
	// Get retrieves content from cache
	Get(ctx context.Context, key string) (io.ReadCloser, *ProxyCacheMetadata, error)

	// Put stores content in cache with TTL
	Put(ctx context.Context, key string, content io.Reader, ttl time.Duration) error

	// Exists checks if a key exists in cache
	Exists(ctx context.Context, key string) (bool, error)

	// Delete removes content from cache
	Delete(ctx context.Context, key string) error

	// GetMetadata returns cache metadata without content
	GetMetadata(ctx context.Context, key string) (*ProxyCacheMetadata, error)
}

// ProxyCacheMetadata represents proxy-specific cache metadata
type ProxyCacheMetadata struct {
	Key        string    `json:"key"`
	Size       int64     `json:"size"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	AccessedAt time.Time `json:"accessed_at"`
	HitCount   int64     `json:"hit_count"`
}

// HTTPClient defines a unified HTTP client interface
// TODO: Migrate from fiber.Agent and various http.Client instances
type HTTPClient interface {
	// Get performs a GET request
	Get(ctx context.Context, url string, headers map[string]string) (*ProxyHTTPResponse, error)

	// Post performs a POST request
	Post(ctx context.Context, url string, body io.Reader, headers map[string]string) (*ProxyHTTPResponse, error)

	// Put performs a PUT request
	Put(ctx context.Context, url string, body io.Reader, headers map[string]string) (*ProxyHTTPResponse, error)

	// Delete performs a DELETE request
	Delete(ctx context.Context, url string, headers map[string]string) (*ProxyHTTPResponse, error)

	// Head performs a HEAD request
	Head(ctx context.Context, url string, headers map[string]string) (*ProxyHTTPResponse, error)
}

// ProxyHTTPResponse represents HTTP response for proxy operations
type ProxyHTTPResponse struct {
	Body       io.ReadCloser     `json:"-"`
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Size       int64             `json:"size"`
}

// PackageNormalizer defines path normalization interface
// TODO: Implement in internal/adapters/pm/common/normalizer.go
type PackageNormalizer interface {
	// NormalizePath normalizes package path for the specific package manager
	NormalizePath(pmType, path string) (string, error)

	// ValidatePath validates if the path is valid for the package manager
	ValidatePath(pmType, path string) error

	// ExtractComponents extracts package components from path
	ExtractComponents(pmType, path string) (*PathComponents, error)
}

// PathComponents represents extracted path components
type PathComponents struct {
	PackageName string            `json:"package_name"`
	Version     string            `json:"version"`
	Classifier  string            `json:"classifier"`
	Extension   string            `json:"extension"`
	Scope       string            `json:"scope"`
	Attributes  map[string]string `json:"attributes"`
}

// SignatureVerifier defines signature verification interface
// TODO: Implement in internal/adapters/pm/common/signature.go
type SignatureVerifier interface {
	// VerifySignature verifies package signature
	VerifySignature(pmType string, content, signature []byte) error

	// GenerateSignature generates signature for content
	GenerateSignature(pmType string, content []byte) ([]byte, error)

	// SupportedSignatureTypes returns supported signature types
	SupportedSignatureTypes(pmType string) []string
}

// ErrorMapper defines error mapping interface
// TODO: Implement in internal/adapters/pm/common/errors.go
type ErrorMapper interface {
	// MapError maps driver-specific errors to standard proxy errors
	MapError(pmType string, err error) error

	// IsRetryableError checks if error is retryable
	IsRetryableError(err error) bool

	// IsNotFoundError checks if error represents "not found"
	IsNotFoundError(err error) bool

	// IsForbiddenError checks if error represents "forbidden"
	IsForbiddenError(err error) bool
}
