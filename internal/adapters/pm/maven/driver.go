package maven

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"proxynd/internal/adapters/pm/common"
	"proxynd/internal/ports"
)

// Driver implements ports.PackageManagerDriver for Maven repositories
type Driver struct {
	config      *Config
	httpClient  ports.HTTPClient
	normalizer  ports.PackageNormalizer
	verifier    ports.SignatureVerifier
	errorMapper ports.ErrorMapper
}

// Config represents Maven driver configuration
type Config struct {
	Enabled      bool                         `json:"enabled" yaml:"enabled"`
	Repositories map[string]*RepositoryConfig `json:"repositories" yaml:"repositories"`
	CacheTTL     time.Duration                `json:"cache_ttl" yaml:"cache_ttl"`
	Timeout      time.Duration                `json:"timeout" yaml:"timeout"`
}

// RepositoryConfig represents Maven repository configuration
type RepositoryConfig struct {
	Name           string            `json:"name" yaml:"name"`
	URL            string            `json:"url" yaml:"url"`
	Enabled        bool              `json:"enabled" yaml:"enabled"`
	Auth           *ports.AuthConfig `json:"auth,omitempty" yaml:"auth,omitempty"`
	LayoutType     string            `json:"layout_type" yaml:"layout_type"` // default, legacy
	SnapshotPolicy *SnapshotPolicy   `json:"snapshot_policy,omitempty" yaml:"snapshot_policy,omitempty"`
}

// SnapshotPolicy defines snapshot handling policy
type SnapshotPolicy struct {
	Enabled        bool   `json:"enabled" yaml:"enabled"`
	UpdatePolicy   string `json:"update_policy" yaml:"update_policy"`     // daily, always, never
	ChecksumPolicy string `json:"checksum_policy" yaml:"checksum_policy"` // fail, warn, ignore
}

// NewDriver creates a new Maven driver
func NewDriver(
	config *Config,
	httpClient ports.HTTPClient,
	normalizer ports.PackageNormalizer,
	verifier ports.SignatureVerifier,
	errorMapper ports.ErrorMapper,
) *Driver {
	if config == nil {
		config = &Config{
			Enabled:  false,
			CacheTTL: 24 * time.Hour,
			Timeout:  30 * time.Second,
		}
	}

	return &Driver{
		config:      config,
		httpClient:  httpClient,
		normalizer:  normalizer,
		verifier:    verifier,
		errorMapper: errorMapper,
	}
}

// Type returns the package manager type
func (d *Driver) Type() string {
	return "maven"
}

// IsSupported checks if the path is supported by Maven
func (d *Driver) IsSupported(path string) bool {
	// Maven paths typically follow: /groupId/artifactId/version/artifactId-version.extension
	mavenPattern := regexp.MustCompile(`^/[^/]+(/[^/]+)*/.+\..+$`)
	return mavenPattern.MatchString(path)
}

// NormalizePath normalizes Maven package path
func (d *Driver) NormalizePath(path string) (string, error) {
	if d.normalizer != nil {
		return d.normalizer.NormalizePath(d.Type(), path)
	}

	// Basic normalization - remove duplicate slashes, clean path
	normalized := filepath.Clean(path)
	if !strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}

	return normalized, nil
}

// BuildUpstreamURL builds upstream URL for Maven request
func (d *Driver) BuildUpstreamURL(req *ports.DriverRequest) (string, error) {
	repository := req.Repository
	if repository == "" {
		repository = common.DefaultRepoCentral // Default to Maven Central
	}

	repoConfig, exists := d.config.Repositories[repository]
	if !exists {
		return "", fmt.Errorf("repository '%s' not configured", repository)
	}

	if !repoConfig.Enabled {
		return "", fmt.Errorf("repository '%s' is disabled", repository)
	}

	baseURL := strings.TrimRight(repoConfig.URL, "/")
	cleanPath := strings.TrimLeft(req.Path, "/")

	return fmt.Sprintf("%s/%s", baseURL, cleanPath), nil
}

// FetchPackage fetches package from upstream Maven repository
func (d *Driver) FetchPackage(ctx context.Context, req *ports.DriverRequest) (*ports.DriverResponse, error) {
	upstreamURL, err := d.BuildUpstreamURL(req)
	if err != nil {
		return nil, d.mapError(err)
	}

	// Prepare headers
	headers := make(map[string]string)
	for k, v := range req.Headers {
		headers[k] = v
	}

	// Add Maven-specific headers
	headers["User-Agent"] = "ProxyND/1.0 Maven-Proxy"
	headers["Accept"] = "*/*"

	// Add authentication if configured
	repository := req.Repository
	if repository == "" {
		repository = "central"
	}

	if repoConfig, exists := d.config.Repositories[repository]; exists && repoConfig.Auth != nil {
		// TODO: Implement authentication headers
		// This will be handled by the unified HTTP client
		_ = repoConfig // Placeholder to avoid empty branch warning
	}

	// Fetch from upstream
	httpResp, err := d.httpClient.Get(ctx, upstreamURL, headers)
	if err != nil {
		return nil, d.mapError(err)
	}

	// Convert to driver response
	response := &ports.DriverResponse{
		Content:      httpResp.Body,
		ContentType:  getContentType(req.Path),
		Size:         httpResp.Size,
		Headers:      httpResp.Headers,
		StatusCode:   httpResp.StatusCode,
		LastModified: time.Now(), // TODO: Parse from headers
	}

	return response, nil
}

// ParseMetadata parses Maven metadata from content
func (d *Driver) ParseMetadata(content []byte) (*ports.PackageMetadata, error) {
	// TODO: Implement Maven POM parsing
	// This should parse maven-metadata.xml or pom.xml files
	return &ports.PackageMetadata{
		Name:      "unknown",
		Version:   "unknown",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// ValidateSignature validates Maven package signatures
func (d *Driver) ValidateSignature(content, signature []byte) error {
	if d.verifier != nil {
		return d.verifier.VerifySignature(d.Type(), content, signature)
	}

	// TODO: Implement Maven-specific signature validation
	// Maven uses MD5, SHA1, SHA256, SHA512, PGP signatures
	return nil
}

// GetCacheKey generates cache key for Maven request
func (d *Driver) GetCacheKey(req *ports.DriverRequest) string {
	repository := req.Repository
	if repository == "" {
		repository = "central"
	}

	normalizedPath, _ := d.NormalizePath(req.Path)
	return fmt.Sprintf("maven:%s:%s", repository, strings.ReplaceAll(normalizedPath, "/", "_"))
}

// ShouldCache determines if response should be cached
func (d *Driver) ShouldCache(resp *ports.DriverResponse) bool {
	// Don't cache error responses
	if resp.StatusCode >= 400 {
		return false
	}

	// Don't cache snapshot artifacts (they change frequently)
	if strings.Contains(resp.Headers["Content-Type"], "SNAPSHOT") {
		return false
	}

	return true
}

// GetCacheTTL returns cache TTL for Maven response
func (d *Driver) GetCacheTTL(resp *ports.DriverResponse) time.Duration {
	// Snapshot artifacts have shorter TTL
	if strings.Contains(resp.Headers["Content-Type"], "SNAPSHOT") {
		return 1 * time.Hour
	}

	// Release artifacts can be cached longer
	return d.config.CacheTTL
}

// LoadConfig loads Maven driver configuration
func (d *Driver) LoadConfig() error {
	// TODO: Load configuration from file or environment
	// This will integrate with the unified config system
	return nil
}

// ValidateConfig validates Maven driver configuration
func (d *Driver) ValidateConfig() error {
	if d.config == nil {
		return fmt.Errorf("maven config is nil")
	}

	if !d.config.Enabled {
		return fmt.Errorf("maven driver is disabled")
	}

	if len(d.config.Repositories) == 0 {
		return fmt.Errorf("no maven repositories configured")
	}

	// Validate each repository
	for name, repo := range d.config.Repositories {
		if repo.URL == "" {
			return fmt.Errorf("repository '%s' has empty URL", name)
		}
	}

	return nil
}

// mapError maps errors using the error mapper
func (d *Driver) mapError(err error) error {
	if d.errorMapper != nil {
		return d.errorMapper.MapError(d.Type(), err)
	}
	return err
}

// getContentType returns content type based on file extension
func getContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".pom":
		return "application/xml"
	case ".jar":
		return common.MimeApplicationJavaArchive
	case ".war":
		return common.MimeApplicationJavaArchive
	case ".ear":
		return common.MimeApplicationJavaArchive
	case ".xml":
		return "application/xml"
	case ".md5":
		return common.MimeTextPlain
	case ".sha1":
		return common.MimeTextPlain
	case ".sha256":
		return common.MimeTextPlain
	case ".sha512":
		return common.MimeTextPlain
	case ".asc":
		return "application/pgp-signature"
	default:
		return "application/octet-stream"
	}
}
