package yum

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

// Driver implements ports.PackageManagerDriver for YUM repositories
type Driver struct {
	config      *Config
	httpClient  ports.HTTPClient
	normalizer  ports.PackageNormalizer
	verifier    ports.SignatureVerifier
	errorMapper ports.ErrorMapper
}

// Config represents YUM driver configuration
type Config struct {
	Enabled      bool                         `json:"enabled" yaml:"enabled"`
	Repositories map[string]*RepositoryConfig `json:"repositories" yaml:"repositories"`
	CacheTTL     time.Duration                `json:"cache_ttl" yaml:"cache_ttl"`
	Timeout      time.Duration                `json:"timeout" yaml:"timeout"`
}

// RepositoryConfig represents YUM repository configuration
type RepositoryConfig struct {
	Name       string            `json:"name" yaml:"name"`
	BaseURL    string            `json:"baseurl" yaml:"baseurl"`
	MirrorList string            `json:"mirrorlist" yaml:"mirrorlist"`
	Enabled    bool              `json:"enabled" yaml:"enabled"`
	GPGCheck   bool              `json:"gpgcheck" yaml:"gpgcheck"`
	GPGKey     string            `json:"gpgkey" yaml:"gpgkey"`
	Auth       *ports.AuthConfig `json:"auth,omitempty" yaml:"auth,omitempty"`
	Priority   int               `json:"priority" yaml:"priority"`
}

// NewDriver creates a new YUM driver
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
			CacheTTL: 6 * time.Hour,
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
	return "yum"
}

// IsSupported checks if the path is supported by YUM
func (d *Driver) IsSupported(path string) bool {
	// YUM paths include:
	// /repodata/repomd.xml
	// /repodata/primary.xml.gz
	// /Packages/package-1.0-1.el7.x86_64.rpm

	patterns := []string{
		`^/repodata/.*\.xml(\.gz|\.bz2|\.xz)?$`, // Repository metadata
		`^/repodata/repomd\.xml$`,               // Repository metadata descriptor
		`^/Packages/.*\.rpm$`,                   // RPM packages
		`^/.*\.rpm$`,                            // RPM packages (alternative layout)
		`^/.*\.drpm$`,                           // Delta RPMs
		`^/.*\.src\.rpm$`,                       // Source RPMs
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, path); matched {
			return true
		}
	}

	return false
}

// NormalizePath normalizes YUM package path
func (d *Driver) NormalizePath(path string) (string, error) {
	if d.normalizer != nil {
		return d.normalizer.NormalizePath(d.Type(), path)
	}

	// Basic normalization
	normalized := filepath.Clean(path)
	if !strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}

	return normalized, nil
}

// BuildUpstreamURL builds upstream URL for YUM request
func (d *Driver) BuildUpstreamURL(req *ports.DriverRequest) (string, error) {
	repository := req.Repository
	if repository == "" {
		repository = common.DefaultRepoCentos // Default repository
	}

	repoConfig, exists := d.config.Repositories[repository]
	if !exists {
		return "", fmt.Errorf("repository '%s' not configured", repository)
	}

	if !repoConfig.Enabled {
		return "", fmt.Errorf("repository '%s' is disabled", repository)
	}

	baseURL := repoConfig.BaseURL
	if baseURL == "" && repoConfig.MirrorList != "" {
		// TODO: Resolve mirror list to get actual URLs
		return "", fmt.Errorf("mirror list resolution not implemented")
	}

	if baseURL == "" {
		return "", fmt.Errorf("no base URL configured for repository '%s'", repository)
	}

	baseURL = strings.TrimRight(baseURL, "/")
	cleanPath := strings.TrimLeft(req.Path, "/")

	return fmt.Sprintf("%s/%s", baseURL, cleanPath), nil
}

// FetchPackage fetches package from upstream YUM repository
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

	// Add YUM-specific headers
	headers["User-Agent"] = "ProxyND/1.0 YUM-Proxy"

	// Accept appropriate content types
	if strings.Contains(req.Path, ".xml") {
		headers["Accept"] = "application/xml, text/xml"
	} else if strings.Contains(req.Path, ".rpm") {
		headers["Accept"] = common.MimeApplicationXRpm
	}

	// Add authentication if configured
	repository := req.Repository
	if repository == "" {
		repository = "centos"
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

// ParseMetadata parses YUM metadata from content
func (d *Driver) ParseMetadata(content []byte) (*ports.PackageMetadata, error) {
	// TODO: Implement YUM metadata parsing
	// This should parse repomd.xml, primary.xml files
	return &ports.PackageMetadata{
		Name:      "unknown",
		Version:   "unknown",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// ValidateSignature validates YUM package signatures
func (d *Driver) ValidateSignature(content []byte, signature []byte) error {
	if d.verifier != nil {
		return d.verifier.VerifySignature(d.Type(), content, signature)
	}

	// TODO: Implement YUM-specific signature validation
	// YUM uses GPG signatures for packages and repository metadata
	return nil
}

// GetCacheKey generates cache key for YUM request
func (d *Driver) GetCacheKey(req *ports.DriverRequest) string {
	repository := req.Repository
	if repository == "" {
		repository = "centos"
	}

	normalizedPath, _ := d.NormalizePath(req.Path)
	return fmt.Sprintf("yum:%s:%s", repository, strings.ReplaceAll(normalizedPath, "/", "_"))
}

// ShouldCache determines if response should be cached
func (d *Driver) ShouldCache(resp *ports.DriverResponse) bool {
	// Don't cache error responses
	if resp.StatusCode >= 400 {
		return false
	}

	return true
}

// GetCacheTTL returns cache TTL for YUM response
func (d *Driver) GetCacheTTL(resp *ports.DriverResponse) time.Duration {
	// RPM packages can be cached longer
	if strings.Contains(resp.ContentType, "application/x-rpm") {
		return 7 * 24 * time.Hour // 1 week
	}

	// Metadata files expire faster
	if strings.Contains(resp.ContentType, "xml") {
		return 6 * time.Hour
	}

	return d.config.CacheTTL
}

// LoadConfig loads YUM driver configuration
func (d *Driver) LoadConfig() error {
	// TODO: Load configuration from file or environment
	return nil
}

// ValidateConfig validates YUM driver configuration
func (d *Driver) ValidateConfig() error {
	if d.config == nil {
		return fmt.Errorf("yum config is nil")
	}

	if !d.config.Enabled {
		return fmt.Errorf("yum driver is disabled")
	}

	if len(d.config.Repositories) == 0 {
		return fmt.Errorf("no yum repositories configured")
	}

	// Validate each repository
	for name, repo := range d.config.Repositories {
		if repo.BaseURL == "" && repo.MirrorList == "" {
			return fmt.Errorf("repository '%s' has neither baseurl nor mirrorlist", name)
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
	case ".rpm":
		return "application/x-rpm"
	case ".drpm":
		return "application/x-rpm"
	case ".xml":
		return "application/xml"
	case ".gz":
		if strings.Contains(path, ".xml.gz") {
			return "application/xml"
		}
		return "application/gzip"
	case ".bz2":
		return "application/x-bzip2"
	case ".xz":
		return "application/x-xz"
	default:
		return "application/octet-stream"
	}
}
