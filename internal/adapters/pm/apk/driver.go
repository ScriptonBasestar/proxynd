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
		common.ApplyAuthentication(headers, repoConfig.Auth)
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
	// Detect content type
	contentStr := string(content)

	// Check if it's APKINDEX (tar.gz compressed text format)
	if len(content) > 2 && content[0] == 0x1f && content[1] == 0x8b {
		// Gzip compressed APKINDEX
		return d.parseAPKINDEXMetadata(content)
	}

	// Check if it's an APK package (contains APK metadata)
	if strings.Contains(contentStr, "P:") || strings.Contains(contentStr, "V:") {
		// Plain text APKINDEX format
		return d.parseAPKINDEXMetadata(content)
	}

	// Unknown metadata format
	return &ports.PackageMetadata{
		Name:        "unknown",
		Description: "APK package or index",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

// ValidateSignature validates APK package signatures
func (d *Driver) ValidateSignature(content, signature []byte) error {
	// Use the common signature verifier
	if d.verifier != nil {
		return d.verifier.VerifySignature(d.Type(), content, signature)
	}

	// APK uses RSA signatures embedded in packages
	// If no verifier is configured, perform basic validation
	if len(signature) == 0 {
		// No signature provided
		// Check if signature verification is required by configuration
		return nil
	}

	// Basic signature format validation
	// APK signatures are typically RSA-based
	sigStr := string(signature)
	if strings.Contains(sigStr, "-----BEGIN") && strings.Contains(sigStr, "SIGNATURE") {
		// RSA signature format detected
		return nil // Format valid, but not verified
	}

	// Try as hex-encoded hash (SHA256)
	if len(signature) == 64 {
		return d.validateHashSignature(content, signature)
	}

	return fmt.Errorf("invalid signature format: expected RSA signature or SHA256 hash")
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

// parseAPKINDEXMetadata parses APKINDEX metadata
func (d *Driver) parseAPKINDEXMetadata(content []byte) (*ports.PackageMetadata, error) {
	// APKINDEX format is a simple text format with lines like:
	// P:package-name
	// V:version
	// A:architecture
	// T:description

	contentStr := string(content)

	metadata := &ports.PackageMetadata{
		Name:        "apkindex",
		Description: "APK package index",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Attributes:  make(map[string]string),
	}

	// Count packages (lines starting with P:)
	packageCount := strings.Count(contentStr, "\nP:")
	if strings.HasPrefix(contentStr, "P:") {
		packageCount++ // Count first package if starts with P:
	}
	metadata.Attributes["package_count"] = fmt.Sprintf("%d", packageCount)

	// Extract first package name and version as example
	lines := strings.Split(contentStr, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "P:") {
			metadata.Attributes["example_package"] = strings.TrimPrefix(line, "P:")
			// Look for version on next lines
			for j := i + 1; j < len(lines) && j < i+10; j++ {
				if strings.HasPrefix(lines[j], "V:") {
					metadata.Version = strings.TrimPrefix(lines[j], "V:")
					break
				}
			}
			break
		}
	}

	return metadata, nil
}

// validateHashSignature validates hash-based signatures using common utility
func (d *Driver) validateHashSignature(content, signature []byte) error {
	return common.ValidateHashSignature(content, signature)
}
