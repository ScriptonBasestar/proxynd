package pypi

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"proxynd/internal/ports"
)

// Driver implements ports.PackageManagerDriver for PyPI repositories
type Driver struct {
	config      *Config
	httpClient  ports.HTTPClient
	normalizer  ports.PackageNormalizer
	verifier    ports.SignatureVerifier
	errorMapper ports.ErrorMapper
}

// Config represents PyPI driver configuration
type Config struct {
	Enabled    bool                      `json:"enabled" yaml:"enabled"`
	Indexes    map[string]*IndexConfig   `json:"indexes" yaml:"indexes"`
	CacheTTL   time.Duration             `json:"cache_ttl" yaml:"cache_ttl"`
	Timeout    time.Duration             `json:"timeout" yaml:"timeout"`
}

// IndexConfig represents PyPI index configuration
type IndexConfig struct {
	Name     string           `json:"name" yaml:"name"`
	URL      string           `json:"url" yaml:"url"`
	Enabled  bool             `json:"enabled" yaml:"enabled"`
	Auth     *ports.AuthConfig `json:"auth,omitempty" yaml:"auth,omitempty"`
	TrustedHost bool          `json:"trusted_host" yaml:"trusted_host"`
}

// NewDriver creates a new PyPI driver
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
	return "pypi"
}

// IsSupported checks if the path is supported by PyPI
func (d *Driver) IsSupported(path string) bool {
	// PyPI paths include:
	// /simple/package-name/
	// /packages/source/p/package/package-1.0.tar.gz
	// /packages/any/py2.py3/package/package-1.0-py2.py3-none-any.whl
	
	patterns := []string{
		`^/simple/[^/]+/?$`,                       // Simple API
		`^/pypi/[^/]+/?$`,                         // PyPI API
		`^/pypi/[^/]+/json/?$`,                    // JSON API
		`^/packages/.*\.(tar\.gz|zip|whl|egg)$`,   // Package files
	}
	
	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, path); matched {
			return true
		}
	}
	
	return false
}

// NormalizePath normalizes PyPI package path
func (d *Driver) NormalizePath(path string) (string, error) {
	if d.normalizer != nil {
		return d.normalizer.NormalizePath(d.Type(), path)
	}
	
	// Basic normalization
	normalized := strings.ToLower(strings.TrimSpace(path))
	if !strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}
	
	// PyPI package names are case-insensitive and normalize _ to -
	if strings.HasPrefix(normalized, "/simple/") {
		parts := strings.Split(normalized, "/")
		if len(parts) >= 3 {
			packageName := parts[2]
			packageName = strings.ReplaceAll(packageName, "_", "-")
			packageName = strings.ReplaceAll(packageName, ".", "-")
			parts[2] = packageName
			normalized = strings.Join(parts, "/")
		}
	}
	
	return normalized, nil
}

// BuildUpstreamURL builds upstream URL for PyPI request
func (d *Driver) BuildUpstreamURL(req *ports.DriverRequest) (string, error) {
	index := req.Repository
	if index == "" {
		index = "pypi" // Default to PyPI
	}
	
	indexConfig, exists := d.config.Indexes[index]
	if !exists {
		return "", fmt.Errorf("index '%s' not configured", index)
	}
	
	if !indexConfig.Enabled {
		return "", fmt.Errorf("index '%s' is disabled", index)
	}
	
	baseURL := strings.TrimRight(indexConfig.URL, "/")
	cleanPath := strings.TrimLeft(req.Path, "/")
	
	return fmt.Sprintf("%s/%s", baseURL, cleanPath), nil
}

// FetchPackage fetches package from upstream PyPI index
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
	
	// Add PyPI-specific headers
	headers["User-Agent"] = "ProxyND/1.0 PyPI-Proxy"
	
	// Accept appropriate content types
	if strings.Contains(req.Path, "/simple/") {
		headers["Accept"] = "text/html, application/vnd.pypi.simple.v1+html"
	} else if strings.Contains(req.Path, "/json") {
		headers["Accept"] = "application/json"
	}
	
	// Add authentication if configured
	index := req.Repository
	if index == "" {
		index = "pypi"
	}
	
	if indexConfig, exists := d.config.Indexes[index]; exists && indexConfig.Auth != nil {
		// TODO: Implement authentication headers
		// This will be handled by the unified HTTP client
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

// ParseMetadata parses PyPI metadata from content
func (d *Driver) ParseMetadata(content []byte) (*ports.PackageMetadata, error) {
	// TODO: Implement PyPI metadata parsing
	// This should parse setup.py, PKG-INFO, METADATA files
	return &ports.PackageMetadata{
		Name:      "unknown",
		Version:   "unknown",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// ValidateSignature validates PyPI package signatures
func (d *Driver) ValidateSignature(content []byte, signature []byte) error {
	if d.verifier != nil {
		return d.verifier.VerifySignature(d.Type(), content, signature)
	}
	
	// TODO: Implement PyPI-specific signature validation
	// PyPI uses GPG signatures and hash verification
	return nil
}

// GetCacheKey generates cache key for PyPI request
func (d *Driver) GetCacheKey(req *ports.DriverRequest) string {
	index := req.Repository
	if index == "" {
		index = "pypi"
	}
	
	normalizedPath, _ := d.NormalizePath(req.Path)
	return fmt.Sprintf("pypi:%s:%s", index, strings.ReplaceAll(normalizedPath, "/", "_"))
}

// ShouldCache determines if response should be cached
func (d *Driver) ShouldCache(resp *ports.DriverResponse) bool {
	// Don't cache error responses
	if resp.StatusCode >= 400 {
		return false
	}
	
	return true
}

// GetCacheTTL returns cache TTL for PyPI response
func (d *Driver) GetCacheTTL(resp *ports.DriverResponse) time.Duration {
	// Package files can be cached longer
	if strings.Contains(resp.ContentType, "application/") {
		return 24 * time.Hour
	}
	
	// Simple API and JSON responses expire faster
	return d.config.CacheTTL
}

// LoadConfig loads PyPI driver configuration
func (d *Driver) LoadConfig() error {
	// TODO: Load configuration from file or environment
	return nil
}

// ValidateConfig validates PyPI driver configuration
func (d *Driver) ValidateConfig() error {
	if d.config == nil {
		return fmt.Errorf("pypi config is nil")
	}
	
	if !d.config.Enabled {
		return fmt.Errorf("pypi driver is disabled")
	}
	
	if len(d.config.Indexes) == 0 {
		return fmt.Errorf("no pypi indexes configured")
	}
	
	// Validate each index
	for name, index := range d.config.Indexes {
		if index.URL == "" {
			return fmt.Errorf("index '%s' has empty URL", name)
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

// getContentType returns content type based on path
func getContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	
	switch ext {
	case ".whl":
		return "application/zip"
	case ".gz":
		return "application/gzip"
	case ".zip":
		return "application/zip"
	case ".egg":
		return "application/zip"
	default:
		if strings.Contains(path, "/simple/") {
			return "text/html"
		} else if strings.Contains(path, "/json") {
			return "application/json"
		}
		return "application/octet-stream"
	}
}