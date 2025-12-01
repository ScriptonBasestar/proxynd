package pypi

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"proxynd/internal/adapters/pm/common"
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
	Enabled  bool                    `json:"enabled" yaml:"enabled"`
	Indexes  map[string]*IndexConfig `json:"indexes" yaml:"indexes"`
	CacheTTL time.Duration           `json:"cache_ttl" yaml:"cache_ttl"`
	Timeout  time.Duration           `json:"timeout" yaml:"timeout"`
}

// IndexConfig represents PyPI index configuration
type IndexConfig struct {
	Name        string            `json:"name" yaml:"name"`
	URL         string            `json:"url" yaml:"url"`
	Enabled     bool              `json:"enabled" yaml:"enabled"`
	Auth        *ports.AuthConfig `json:"auth,omitempty" yaml:"auth,omitempty"`
	TrustedHost bool              `json:"trusted_host" yaml:"trusted_host"`
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
	return common.PMTypePypi
}

// IsSupported checks if the path is supported by PyPI
func (d *Driver) IsSupported(path string) bool {
	// PyPI paths include:
	// /simple/package-name/
	// /packages/source/p/package/package-1.0.tar.gz
	// /packages/any/py2.py3/package/package-1.0-py2.py3-none-any.whl

	patterns := []string{
		`^/simple/[^/]+/?$`,                     // Simple API
		`^/pypi/[^/]+/?$`,                       // PyPI API
		`^/pypi/[^/]+/json/?$`,                  // JSON API
		`^/packages/.*\.(tar\.gz|zip|whl|egg)$`, // Package files
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
		index = common.DefaultRepoPyPI // Default to PyPI
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
		if err := common.ApplyAuthentication(headers, indexConfig.Auth); err != nil {
			return nil, fmt.Errorf("failed to apply authentication: %w", err)
		}
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
	// Detect content type
	contentStr := string(content)

	// Check if it's JSON API response
	if strings.HasPrefix(strings.TrimSpace(contentStr), "{") {
		return d.parseJSONMetadata(content)
	}

	// Check if it's PKG-INFO or METADATA format (RFC 822-style)
	if strings.Contains(contentStr, "Metadata-Version:") && strings.Contains(contentStr, "Name:") {
		return d.parsePKGInfoMetadata(content)
	}

	// Check if it's a wheel file (ZIP archive)
	if len(content) > 4 && string(content[0:2]) == "PK" {
		return &ports.PackageMetadata{
			Name:        "python-wheel",
			Description: "Python wheel package",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}, nil
	}

	// Unknown metadata format
	return &ports.PackageMetadata{
		Name:        "unknown",
		Description: "PyPI metadata",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

// ValidateSignature validates PyPI package signatures
func (d *Driver) ValidateSignature(content, signature []byte) error {
	// Use the common signature verifier
	if d.verifier != nil {
		return d.verifier.VerifySignature(d.Type(), content, signature)
	}

	// PyPI uses GPG signatures and hash verification
	// If no verifier is configured, perform basic validation
	if len(signature) == 0 {
		// No signature provided
		// Check if signature verification is required by configuration
		return nil
	}

	// Basic GPG signature format validation
	// GPG signatures typically start with specific markers
	sigStr := string(signature)
	if !strings.Contains(sigStr, "BEGIN PGP SIGNATURE") {
		// Not a PGP signature format, try as hex-encoded hash
		if len(signature) == 64 { // SHA256 hex
			// Hash-based signature validation
			return d.validateHashSignature(content, signature)
		}
		return fmt.Errorf("invalid signature format: expected PGP signature or hash")
	}

	// PGP signature found, but no verifier configured
	// Return nil to indicate signature format is valid but not verified
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
		return common.MimeApplicationZip
	case ".gz":
		return "application/gzip"
	case ".zip":
		return common.MimeApplicationZip
	case ".egg":
		return common.MimeApplicationZip
	default:
		if strings.Contains(path, "/simple/") {
			return "text/html"
		} else if strings.Contains(path, "/json") {
			return "application/json"
		}
		return "application/octet-stream"
	}
}

// parseJSONMetadata parses PyPI JSON API metadata
func (d *Driver) parseJSONMetadata(content []byte) (*ports.PackageMetadata, error) {
	// PyPI JSON API format:
	// {
	//   "info": {
	//     "name": "package-name",
	//     "version": "1.0.0",
	//     "summary": "Package description"
	//   }
	// }

	var data struct {
		Info struct {
			Name        string   `json:"name"`
			Version     string   `json:"version"`
			Summary     string   `json:"summary"`
			Author      string   `json:"author"`
			License     string   `json:"license"`
			Homepage    string   `json:"home_page"`
			Keywords    string   `json:"keywords"`
			Description string   `json:"description"`
			Requires    []string `json:"requires_dist"`
		} `json:"info"`
	}

	if err := json.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON metadata: %w", err)
	}

	metadata := &ports.PackageMetadata{
		Name:        data.Info.Name,
		Version:     data.Info.Version,
		Description: data.Info.Summary,
		License:     data.Info.License,
		Author:      data.Info.Author,
		Homepage:    data.Info.Homepage,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Attributes:  make(map[string]string),
	}

	// Parse keywords
	if data.Info.Keywords != "" {
		metadata.Keywords = strings.Split(data.Info.Keywords, ",")
		for i := range metadata.Keywords {
			metadata.Keywords[i] = strings.TrimSpace(metadata.Keywords[i])
		}
	}

	// Add dependencies
	if len(data.Info.Requires) > 0 {
		metadata.Dependencies = data.Info.Requires
	}

	return metadata, nil
}

// parsePKGInfoMetadata parses PKG-INFO/METADATA format
func (d *Driver) parsePKGInfoMetadata(content []byte) (*ports.PackageMetadata, error) {
	// PKG-INFO format is RFC 822-style with fields like:
	// Metadata-Version: 2.1
	// Name: package-name
	// Version: 1.0.0

	contentStr := string(content)

	metadata := &ports.PackageMetadata{
		Name:        "unknown",
		Description: "Python package metadata",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Attributes:  make(map[string]string),
	}

	// Parse key-value pairs
	lines := strings.Split(contentStr, "\n")
	for _, line := range lines {
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				switch key {
				case "Name":
					metadata.Name = value
				case "Version":
					metadata.Version = value
				case "Summary":
					metadata.Description = value
				case "Author":
					metadata.Author = value
				case "License":
					metadata.License = value
				case "Home-page":
					metadata.Homepage = value
				case "Keywords":
					metadata.Keywords = strings.Split(value, ",")
					for i := range metadata.Keywords {
						metadata.Keywords[i] = strings.TrimSpace(metadata.Keywords[i])
					}
				case "Metadata-Version":
					metadata.Attributes["metadata_version"] = value
				}
			}
		}
	}

	return metadata, nil
}

// validateHashSignature validates hash-based signatures using common utility
func (d *Driver) validateHashSignature(content, signature []byte) error {
	return common.ValidateHashSignature(content, signature)
}
