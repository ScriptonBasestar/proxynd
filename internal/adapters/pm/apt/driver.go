package apt

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"proxynd/internal/ports"
)

// Driver implements ports.PackageManagerDriver for APT repositories
type Driver struct {
	config      *Config
	httpClient  ports.HTTPClient
	normalizer  ports.PackageNormalizer
	verifier    ports.SignatureVerifier
	errorMapper ports.ErrorMapper
}

// Config represents APT driver configuration
type Config struct {
	Enabled      bool                       `json:"enabled" yaml:"enabled"`
	Mirrors      map[string]*MirrorConfig   `json:"mirrors" yaml:"mirrors"`
	CacheTTL     time.Duration              `json:"cache_ttl" yaml:"cache_ttl"`
	Timeout      time.Duration              `json:"timeout" yaml:"timeout"`
	Distributions map[string]*DistroConfig  `json:"distributions" yaml:"distributions"`
}

// MirrorConfig represents APT mirror configuration
type MirrorConfig struct {
	Name        string           `json:"name" yaml:"name"`
	URL         string           `json:"url" yaml:"url"`
	Enabled     bool             `json:"enabled" yaml:"enabled"`
	Auth        *ports.AuthConfig `json:"auth,omitempty" yaml:"auth,omitempty"`
	Priority    int              `json:"priority" yaml:"priority"`
	Architectures []string       `json:"architectures" yaml:"architectures"`
}

// DistroConfig represents distribution configuration
type DistroConfig struct {
	Name       string   `json:"name" yaml:"name"`
	Codename   string   `json:"codename" yaml:"codename"`
	Components []string `json:"components" yaml:"components"` // main, contrib, non-free
	Mirror     string   `json:"mirror" yaml:"mirror"`
}

// NewDriver creates a new APT driver
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
			CacheTTL: 12 * time.Hour,
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
	return "apt"
}

// IsSupported checks if the path is supported by APT
func (d *Driver) IsSupported(path string) bool {
	// APT paths include:
	// /dists/focal/Release
	// /dists/focal/main/binary-amd64/Packages
	// /pool/main/a/apache2/apache2_2.4.41-4ubuntu3_amd64.deb
	// /ubuntu/dists/focal/Release
	
	patterns := []string{
		`^/dists/[^/]+/Release`,                    // Release files
		`^/dists/[^/]+/Release\.gpg`,              // GPG signatures
		`^/dists/[^/]+/InRelease`,                 // Inline release
		`^/dists/[^/]+/[^/]+/binary-[^/]+/Packages`, // Package lists
		`^/dists/[^/]+/[^/]+/source/Sources`,      // Source lists
		`^/pool/[^/]+/[^/]+/[^/]+/.*\.deb$`,       // Package files
		`^/pool/[^/]+/[^/]+/[^/]+/.*\.dsc$`,       // Source description
		`^/pool/[^/]+/[^/]+/[^/]+/.*\.tar\.(gz|xz|bz2)$`, // Source archives
		`^/[^/]+/dists/`,                          // Mirror prefix
		`^/[^/]+/pool/`,                           // Mirror prefix
	}
	
	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, path); matched {
			return true
		}
	}
	
	return false
}

// NormalizePath normalizes APT package path
func (d *Driver) NormalizePath(path string) (string, error) {
	if d.normalizer != nil {
		return d.normalizer.NormalizePath(d.Type(), path)
	}
	
	// Basic normalization
	normalized := filepath.Clean(path)
	if !strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}
	
	// Remove duplicate slashes
	normalized = regexp.MustCompile(`/+`).ReplaceAllString(normalized, "/")
	
	return normalized, nil
}

// BuildUpstreamURL builds upstream URL for APT request
func (d *Driver) BuildUpstreamURL(req *ports.DriverRequest) (string, error) {
	mirror := d.getMirrorForPath(req.Path)
	
	mirrorConfig, exists := d.config.Mirrors[mirror]
	if !exists {
		return "", fmt.Errorf("mirror '%s' not configured", mirror)
	}
	
	if !mirrorConfig.Enabled {
		return "", fmt.Errorf("mirror '%s' is disabled", mirror)
	}
	
	baseURL := strings.TrimRight(mirrorConfig.URL, "/")
	cleanPath := strings.TrimLeft(req.Path, "/")
	
	// Remove mirror prefix if present in path
	if strings.HasPrefix(cleanPath, mirror+"/") {
		cleanPath = strings.TrimPrefix(cleanPath, mirror+"/")
	}
	
	return fmt.Sprintf("%s/%s", baseURL, cleanPath), nil
}

// FetchPackage fetches package from upstream APT mirror
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
	
	// Add APT-specific headers
	headers["User-Agent"] = "ProxyND/1.0 APT-Proxy"
	
	// APT expects specific Accept headers for different content types
	if strings.Contains(req.Path, "Packages") {
		headers["Accept"] = "text/plain, application/x-gzip"
	} else if strings.Contains(req.Path, "Release") {
		headers["Accept"] = "text/plain"
	}
	
	// Add authentication if configured
	mirror := d.getMirrorForPath(req.Path)
	if mirrorConfig, exists := d.config.Mirrors[mirror]; exists && mirrorConfig.Auth != nil {
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

// ParseMetadata parses APT metadata from content
func (d *Driver) ParseMetadata(content []byte) (*ports.PackageMetadata, error) {
	// TODO: Implement APT metadata parsing
	// This should parse Release files, Packages files, etc.
	// APT metadata format is RFC 822-style fields
	
	return &ports.PackageMetadata{
		Name:      "unknown",
		Version:   "unknown",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// ValidateSignature validates APT package signatures
func (d *Driver) ValidateSignature(content []byte, signature []byte) error {
	if d.verifier != nil {
		return d.verifier.VerifySignature(d.Type(), content, signature)
	}
	
	// TODO: Implement APT-specific signature validation
	// APT uses GPG signatures for Release files
	return nil
}

// GetCacheKey generates cache key for APT request
func (d *Driver) GetCacheKey(req *ports.DriverRequest) string {
	mirror := d.getMirrorForPath(req.Path)
	normalizedPath, _ := d.NormalizePath(req.Path)
	return fmt.Sprintf("apt:%s:%s", mirror, strings.ReplaceAll(normalizedPath, "/", "_"))
}

// ShouldCache determines if response should be cached
func (d *Driver) ShouldCache(resp *ports.DriverResponse) bool {
	// Don't cache error responses
	if resp.StatusCode >= 400 {
		return false
	}
	
	// Cache everything for APT
	return true
}

// GetCacheTTL returns cache TTL for APT response
func (d *Driver) GetCacheTTL(resp *ports.DriverResponse) time.Duration {
	// Package files can be cached longer
	if strings.HasSuffix(resp.ContentType, "application/x-debian-package") {
		return 7 * 24 * time.Hour // 1 week
	}
	
	// Metadata files expire faster
	if strings.Contains(resp.ContentType, "text/plain") {
		return 6 * time.Hour
	}
	
	return d.config.CacheTTL
}

// LoadConfig loads APT driver configuration
func (d *Driver) LoadConfig() error {
	// TODO: Load configuration from file or environment
	// This will integrate with the unified config system
	return nil
}

// ValidateConfig validates APT driver configuration
func (d *Driver) ValidateConfig() error {
	if d.config == nil {
		return fmt.Errorf("apt config is nil")
	}
	
	if !d.config.Enabled {
		return fmt.Errorf("apt driver is disabled")
	}
	
	if len(d.config.Mirrors) == 0 {
		return fmt.Errorf("no apt mirrors configured")
	}
	
	// Validate each mirror
	for name, mirror := range d.config.Mirrors {
		if mirror.URL == "" {
			return fmt.Errorf("mirror '%s' has empty URL", name)
		}
	}
	
	return nil
}

// getMirrorForPath determines which mirror to use for a path
func (d *Driver) getMirrorForPath(path string) string {
	// Check if path starts with a known mirror name
	for name := range d.config.Mirrors {
		if strings.HasPrefix(path, "/"+name+"/") {
			return name
		}
	}
	
	// Default mirror
	return "ubuntu"
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
	case ".deb":
		return "application/x-debian-package"
	case ".dsc":
		return "text/plain"
	case ".gz":
		if strings.Contains(path, "Packages") {
			return "application/x-gzip"
		}
		return "application/gzip"
	case ".xz":
		return "application/x-xz"
	case ".bz2":
		return "application/x-bzip2"
	case ".gpg":
		return "application/pgp-signature"
	default:
		if strings.Contains(path, "Release") || strings.Contains(path, "Packages") || strings.Contains(path, "Sources") {
			return "text/plain"
		}
		return "application/octet-stream"
	}
}