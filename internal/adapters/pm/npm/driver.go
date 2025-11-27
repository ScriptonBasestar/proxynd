package npm

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"proxynd/internal/ports"
)

// Driver implements ports.PackageManagerDriver for NPM registries
type Driver struct {
	config      *Config
	httpClient  ports.HTTPClient
	normalizer  ports.PackageNormalizer
	verifier    ports.SignatureVerifier
	errorMapper ports.ErrorMapper
}

// Config represents NPM driver configuration
type Config struct {
	Enabled    bool                       `json:"enabled" yaml:"enabled"`
	Registries map[string]*RegistryConfig `json:"registries" yaml:"registries"`
	CacheTTL   time.Duration              `json:"cache_ttl" yaml:"cache_ttl"`
	Timeout    time.Duration              `json:"timeout" yaml:"timeout"`
	Scopes     map[string]string          `json:"scopes" yaml:"scopes"` // scope -> registry mapping
}

// RegistryConfig represents NPM registry configuration
type RegistryConfig struct {
	Name       string            `json:"name" yaml:"name"`
	URL        string            `json:"url" yaml:"url"`
	Enabled    bool              `json:"enabled" yaml:"enabled"`
	Auth       *ports.AuthConfig `json:"auth,omitempty" yaml:"auth,omitempty"`
	Token      string            `json:"token,omitempty" yaml:"token,omitempty"`
	AlwaysAuth bool              `json:"always_auth" yaml:"always_auth"`
}

// PackageInfo represents NPM package.json structure
type PackageInfo struct {
	Name             string            `json:"name"`
	Version          string            `json:"version"`
	Description      string            `json:"description"`
	Keywords         []string          `json:"keywords"`
	Homepage         string            `json:"homepage"`
	License          string            `json:"license"`
	Author           interface{}       `json:"author"` // can be string or object
	Repository       interface{}       `json:"repository"`
	Dependencies     map[string]string `json:"dependencies"`
	DevDependencies  map[string]string `json:"devDependencies"`
	PeerDependencies map[string]string `json:"peerDependencies"`
	Dist             *DistInfo         `json:"dist"`
}

// DistInfo represents NPM distribution info
type DistInfo struct {
	Tarball      string `json:"tarball"`
	Shasum       string `json:"shasum"`
	Integrity    string `json:"integrity"`
	FileCount    int    `json:"fileCount"`
	UnpackedSize int64  `json:"unpackedSize"`
}

// NewDriver creates a new NPM driver
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
			Scopes:   make(map[string]string),
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
	return "npm"
}

// IsSupported checks if the path is supported by NPM
func (d *Driver) IsSupported(path string) bool {
	// NPM paths can be:
	// /@scope/package-name (scoped packages)
	// /package-name (regular packages)
	// /package-name/-/package-name-version.tgz (tarball)
	// /package-name/version (specific version)

	patterns := []string{
		`^/@[^/]+/[^/]+$`,                    // scoped package
		`^/[^@][^/]*$`,                       // regular package
		`^/[^/]+/-/[^/]+-[^/]+\.tgz$`,        // tarball
		`^/@[^/]+/[^/]+/-/[^/]+-[^/]+\.tgz$`, // scoped tarball
		`^/[^/]+/[^/]+$`,                     // package/version
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, path); matched {
			return true
		}
	}

	return false
}

// NormalizePath normalizes NPM package path
func (d *Driver) NormalizePath(path string) (string, error) {
	if d.normalizer != nil {
		return d.normalizer.NormalizePath(d.Type(), path)
	}

	// Basic normalization
	normalized := strings.TrimSpace(path)
	if !strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}

	// Handle double encoding of scoped packages
	normalized = strings.ReplaceAll(normalized, "%40", "@")
	normalized = strings.ReplaceAll(normalized, "%2F", "/")

	return normalized, nil
}

// BuildUpstreamURL builds upstream URL for NPM request
func (d *Driver) BuildUpstreamURL(req *ports.DriverRequest) (string, error) {
	registry := d.getRegistryForPath(req.Path)

	registryConfig, exists := d.config.Registries[registry]
	if !exists {
		return "", fmt.Errorf("registry '%s' not configured", registry)
	}

	if !registryConfig.Enabled {
		return "", fmt.Errorf("registry '%s' is disabled", registry)
	}

	baseURL := strings.TrimRight(registryConfig.URL, "/")
	cleanPath := strings.TrimLeft(req.Path, "/")

	return fmt.Sprintf("%s/%s", baseURL, cleanPath), nil
}

// FetchPackage fetches package from upstream NPM registry
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

	// Add NPM-specific headers
	headers["User-Agent"] = "ProxyND/1.0 NPM-Proxy"
	headers["Accept"] = "application/json"

	// Add authentication if configured
	registry := d.getRegistryForPath(req.Path)
	if registryConfig, exists := d.config.Registries[registry]; exists {
		if registryConfig.Token != "" {
			headers["Authorization"] = fmt.Sprintf("Bearer %s", registryConfig.Token)
		} else if registryConfig.Auth != nil {
			auth := registryConfig.Auth
			switch auth.Type {
			case "basic":
				// Basic Authentication
				if auth.Username != "" && auth.Password != "" {
					headers["Authorization"] = fmt.Sprintf("Basic %s", encodeBasicAuth(auth.Username, auth.Password))
				}
			case "bearer", "token":
				// Bearer Token Authentication
				if auth.Token != "" {
					headers["Authorization"] = fmt.Sprintf("Bearer %s", auth.Token)
				}
			case "digest":
				// Digest Authentication - add WWW-Authenticate response handling
				// Note: Full digest auth requires challenge-response, implemented in HTTP client
				if auth.Username != "" {
					headers["X-Auth-Username"] = auth.Username
				}
			}
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

// ParseMetadata parses NPM metadata from content
func (d *Driver) ParseMetadata(content []byte) (*ports.PackageMetadata, error) {
	var packageInfo PackageInfo
	if err := json.Unmarshal(content, &packageInfo); err != nil {
		return nil, fmt.Errorf("failed to parse package.json: %w", err)
	}

	// Extract author string
	var author string
	switch a := packageInfo.Author.(type) {
	case string:
		author = a
	case map[string]interface{}:
		if name, ok := a["name"].(string); ok {
			author = name
		}
	}

	// Extract dependencies
	var dependencies []string
	for name, version := range packageInfo.Dependencies {
		dependencies = append(dependencies, fmt.Sprintf("%s@%s", name, version))
	}

	return &ports.PackageMetadata{
		Name:         packageInfo.Name,
		Version:      packageInfo.Version,
		Description:  packageInfo.Description,
		License:      packageInfo.License,
		Author:       author,
		Homepage:     packageInfo.Homepage,
		Dependencies: dependencies,
		Keywords:     packageInfo.Keywords,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, nil
}

// ValidateSignature validates NPM package signatures
func (d *Driver) ValidateSignature(content, signature []byte) error {
	// Use the common signature verifier
	if d.verifier != nil {
		return d.verifier.VerifySignature(d.Type(), content, signature)
	}

	// NPM uses SHA1, SHA512, and SRI integrity hashes
	// If no verifier is configured, perform basic validation
	if len(signature) == 0 {
		// No signature provided
		return nil
	}

	sigStr := strings.TrimSpace(string(signature))

	// Check if it's SRI (Subresource Integrity) format
	// Format: sha512-base64hash or sha384-base64hash or sha256-base64hash
	if strings.HasPrefix(sigStr, "sha512-") || strings.HasPrefix(sigStr, "sha384-") || strings.HasPrefix(sigStr, "sha256-") {
		return d.validateSRIIntegrity(content, sigStr)
	}

	// Try hash-based signature validation (SHA1, SHA256, SHA512 hex)
	return d.validateHashSignature(content, signature)
}

// GetCacheKey generates cache key for NPM request
func (d *Driver) GetCacheKey(req *ports.DriverRequest) string {
	registry := d.getRegistryForPath(req.Path)
	normalizedPath, _ := d.NormalizePath(req.Path)
	return fmt.Sprintf("npm:%s:%s", registry, strings.ReplaceAll(normalizedPath, "/", "_"))
}

// ShouldCache determines if response should be cached
func (d *Driver) ShouldCache(resp *ports.DriverResponse) bool {
	// Don't cache error responses
	if resp.StatusCode >= 400 {
		return false
	}

	// Cache package metadata and tarballs
	return true
}

// GetCacheTTL returns cache TTL for NPM response
func (d *Driver) GetCacheTTL(resp *ports.DriverResponse) time.Duration {
	// Tarballs can be cached longer (they don't change)
	if strings.Contains(resp.ContentType, "application/octet-stream") {
		return 24 * time.Hour
	}

	// Package metadata expires faster
	return d.config.CacheTTL
}

// LoadConfig loads NPM driver configuration
func (d *Driver) LoadConfig() error {
	// TODO: Load configuration from file or environment
	// This will integrate with the unified config system
	return nil
}

// ValidateConfig validates NPM driver configuration
func (d *Driver) ValidateConfig() error {
	if d.config == nil {
		return fmt.Errorf("npm config is nil")
	}

	if !d.config.Enabled {
		return fmt.Errorf("npm driver is disabled")
	}

	if len(d.config.Registries) == 0 {
		return fmt.Errorf("no npm registries configured")
	}

	// Validate each registry
	for name, registry := range d.config.Registries {
		if registry.URL == "" {
			return fmt.Errorf("registry '%s' has empty URL", name)
		}
	}

	return nil
}

// getRegistryForPath determines which registry to use for a path
func (d *Driver) getRegistryForPath(path string) string {
	// Extract scope from scoped packages
	if strings.HasPrefix(path, "/@") {
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			scope := parts[1] // @scope
			if registry, exists := d.config.Scopes[scope]; exists {
				return registry
			}
		}
	}

	// Default registry
	return "npmjs"
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
	if strings.HasSuffix(path, ".tgz") || strings.HasSuffix(path, ".tar.gz") {
		return "application/octet-stream"
	}

	// Package metadata
	return "application/json"
}

// validateSRIIntegrity validates SRI (Subresource Integrity) format
func (d *Driver) validateSRIIntegrity(content []byte, integrity string) error {
	// SRI format: algorithm-base64hash
	// Example: sha512-xxxxx or sha256-xxxxx

	parts := strings.SplitN(integrity, "-", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid SRI format: %s", integrity)
	}

	algorithm := parts[0]
	expectedHash := parts[1]

	var computed string
	switch algorithm {
	case "sha256":
		hasher := sha256.New()
		hasher.Write(content)
		computed = base64.StdEncoding.EncodeToString(hasher.Sum(nil))
	case "sha384":
		hasher := sha512.New384()
		hasher.Write(content)
		computed = base64.StdEncoding.EncodeToString(hasher.Sum(nil))
	case "sha512":
		hasher := sha512.New()
		hasher.Write(content)
		computed = base64.StdEncoding.EncodeToString(hasher.Sum(nil))
	default:
		return fmt.Errorf("unsupported SRI algorithm: %s", algorithm)
	}

	if computed != expectedHash {
		return fmt.Errorf("%s SRI integrity mismatch", algorithm)
	}

	return nil
}

// validateHashSignature validates hash-based signatures (SHA1, SHA256, SHA512 hex)
func (d *Driver) validateHashSignature(content, signature []byte) error {
	sigStr := strings.ToLower(strings.TrimSpace(string(signature)))

	// Try SHA1 (40 hex characters) - NPM shasum
	if len(sigStr) == 40 {
		hasher := sha1.New()
		hasher.Write(content)
		computed := hex.EncodeToString(hasher.Sum(nil))
		if computed == sigStr {
			return nil
		}
		return fmt.Errorf("SHA1 hash mismatch")
	}

	// Try SHA256 (64 hex characters)
	if len(sigStr) == 64 {
		hasher := sha256.New()
		hasher.Write(content)
		computed := hex.EncodeToString(hasher.Sum(nil))
		if computed == sigStr {
			return nil
		}
		return fmt.Errorf("SHA256 hash mismatch")
	}

	// Try SHA512 (128 hex characters)
	if len(sigStr) == 128 {
		hasher := sha512.New()
		hasher.Write(content)
		computed := hex.EncodeToString(hasher.Sum(nil))
		if computed == sigStr {
			return nil
		}
		return fmt.Errorf("SHA512 hash mismatch")
	}

	return fmt.Errorf("unknown hash signature length: %d", len(sigStr))
}

// encodeBasicAuth encodes username and password for HTTP Basic Authentication
func encodeBasicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
