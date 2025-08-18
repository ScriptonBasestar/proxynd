package apk

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

// Driver implements ports.PackageManagerDriver for APK repositories
type Driver struct {
	config      *Config
	httpClient  ports.HTTPClient
	normalizer  ports.PackageNormalizer
	verifier    ports.SignatureVerifier
	errorMapper ports.ErrorMapper
}

// Config represents APK driver configuration
type Config struct {
	Enabled      bool                         `json:"enabled" yaml:"enabled"`
	Repositories map[string]*RepositoryConfig `json:"repositories" yaml:"repositories"`
	CacheTTL     time.Duration                `json:"cache_ttl" yaml:"cache_ttl"`
	Timeout      time.Duration                `json:"timeout" yaml:"timeout"`
}

// RepositoryConfig represents APK repository configuration
type RepositoryConfig struct {
	Name         string            `json:"name" yaml:"name"`
	URL          string            `json:"url" yaml:"url"`
	Enabled      bool              `json:"enabled" yaml:"enabled"`
	Auth         *ports.AuthConfig `json:"auth,omitempty" yaml:"auth,omitempty"`
	Branch       string            `json:"branch" yaml:"branch"`             // main, community, testing
	Architecture string            `json:"architecture" yaml:"architecture"` // x86_64, aarch64, etc.
	SigningKey   string            `json:"signing_key" yaml:"signing_key"`
}

// NewDriver creates a new APK driver
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
	return "apk"
}

// IsSupported checks if the path is supported by APK
func (d *Driver) IsSupported(path string) bool {
	// APK paths include:
	// /alpine/v3.16/main/x86_64/APKINDEX.tar.gz
	// /alpine/v3.16/main/x86_64/package-1.0-r0.apk
	// /alpine/edge/testing/x86_64/APKINDEX.tar.gz

	patterns := []string{
		`^/alpine/v[\d.]+/[^/]+/[^/]+/APKINDEX\.tar\.gz$`, // Index files
		`^/alpine/v[\d.]+/[^/]+/[^/]+/.*\.apk$`,           // Package files
		`^/alpine/edge/[^/]+/[^/]+/APKINDEX\.tar\.gz$`,    // Edge index files
		`^/alpine/edge/[^/]+/[^/]+/.*\.apk$`,              // Edge package files
		`^/[^/]+/[^/]+/[^/]+/APKINDEX\.tar\.gz$`,          // Alternative layout
		`^/[^/]+/[^/]+/[^/]+/.*\.apk$`,                    // Alternative layout
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, path); matched {
			return true
		}
	}

	return false
}

// NormalizePath normalizes APK package path
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

// BuildUpstreamURL builds upstream URL for APK request
func (d *Driver) BuildUpstreamURL(req *ports.DriverRequest) (string, error) {
	repository := req.Repository
	if repository == "" {
		repository = common.DefaultRepoAlpine // Default repository
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

// FetchPackage fetches package from upstream APK repository
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

	// Add APK-specific headers
	headers["User-Agent"] = "ProxyND/1.0 APK-Proxy"

	// Accept appropriate content types
	if strings.Contains(req.Path, "APKINDEX") {
		headers["Accept"] = "application/gzip, application/x-tar"
	} else if strings.Contains(req.Path, ".apk") {
		headers["Accept"] = "application/vnd.alpine.apk"
	}

	// Add authentication if configured
	repository := req.Repository
	if repository == "" {
		repository = "alpine"
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

// ParseMetadata parses APK metadata from content
func (d *Driver) ParseMetadata(content []byte) (*ports.PackageMetadata, error) {
	// TODO: Implement APK metadata parsing
	// This should parse APKINDEX files
	return &ports.PackageMetadata{
		Name:      "unknown",
		Version:   "unknown",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// ValidateSignature validates APK package signatures
func (d *Driver) ValidateSignature(content []byte, signature []byte) error {
	if d.verifier != nil {
		return d.verifier.VerifySignature(d.Type(), content, signature)
	}

	// TODO: Implement APK-specific signature validation
	// APK uses RSA signatures embedded in the packages
	return nil
}

// GetCacheKey generates cache key for APK request
func (d *Driver) GetCacheKey(req *ports.DriverRequest) string {
	repository := req.Repository
	if repository == "" {
		repository = "alpine"
	}

	normalizedPath, _ := d.NormalizePath(req.Path)
	return fmt.Sprintf("apk:%s:%s", repository, strings.ReplaceAll(normalizedPath, "/", "_"))
}

// ShouldCache determines if response should be cached
func (d *Driver) ShouldCache(resp *ports.DriverResponse) bool {
	// Don't cache error responses
	if resp.StatusCode >= 400 {
		return false
	}

	return true
}

// GetCacheTTL returns cache TTL for APK response
func (d *Driver) GetCacheTTL(resp *ports.DriverResponse) time.Duration {
	// Package files can be cached longer
	if strings.Contains(resp.ContentType, "application/vnd.alpine.apk") {
		return 7 * 24 * time.Hour // 1 week
	}

	// Index files expire faster
	if strings.Contains(resp.ContentType, "application/gzip") {
		return 6 * time.Hour
	}

	return d.config.CacheTTL
}

// LoadConfig loads APK driver configuration
func (d *Driver) LoadConfig() error {
	// TODO: Load configuration from file or environment
	return nil
}

// ValidateConfig validates APK driver configuration
func (d *Driver) ValidateConfig() error {
	if d.config == nil {
		return fmt.Errorf("apk config is nil")
	}

	if !d.config.Enabled {
		return fmt.Errorf("apk driver is disabled")
	}

	if len(d.config.Repositories) == 0 {
		return fmt.Errorf("no apk repositories configured")
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
	if strings.Contains(path, "APKINDEX.tar.gz") {
		return "application/gzip"
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".apk":
		return "application/vnd.alpine.apk"
	case ".gz":
		return "application/gzip"
	case ".tar":
		return "application/x-tar"
	default:
		return "application/octet-stream"
	}
}
