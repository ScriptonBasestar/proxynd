package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"proxynd/internal/adapters/pm/common"
	"proxynd/internal/ports"
)

// Driver implements ports.PackageManagerDriver for Docker registries
type Driver struct {
	config      *Config
	httpClient  ports.HTTPClient
	normalizer  ports.PackageNormalizer
	verifier    ports.SignatureVerifier
	errorMapper ports.ErrorMapper
}

// Config represents Docker registry driver configuration
type Config struct {
	Enabled    bool                       `json:"enabled" yaml:"enabled"`
	Registries map[string]*RegistryConfig `json:"registries" yaml:"registries"`
	CacheTTL   time.Duration              `json:"cache_ttl" yaml:"cache_ttl"`
	Timeout    time.Duration              `json:"timeout" yaml:"timeout"`
}

// RegistryConfig represents Docker registry configuration
type RegistryConfig struct {
	Name     string            `json:"name" yaml:"name"`
	URL      string            `json:"url" yaml:"url"`
	Enabled  bool              `json:"enabled" yaml:"enabled"`
	Auth     *ports.AuthConfig `json:"auth,omitempty" yaml:"auth,omitempty"`
	Insecure bool              `json:"insecure" yaml:"insecure"`
	Version  string            `json:"version" yaml:"version"` // v1, v2
}

// Manifest represents Docker manifest structure
type Manifest struct {
	SchemaVersion int                  `json:"schemaVersion"`
	MediaType     string               `json:"mediaType"`
	Config        *ManifestDescriptor  `json:"config"`
	Layers        []ManifestDescriptor `json:"layers"`
}

// ManifestDescriptor represents manifest descriptor
type ManifestDescriptor struct {
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
	Digest    string `json:"digest"`
}

// NewDriver creates a new Docker registry driver
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
	return "registry"
}

// IsSupported checks if the path is supported by Docker registry
func (d *Driver) IsSupported(path string) bool {
	// Docker registry paths include:
	// /v2/
	// /v2/library/ubuntu/manifests/latest
	// /v2/library/ubuntu/blobs/sha256:abcd...
	// /v2/library/ubuntu/tags/list

	patterns := []string{
		`^/v2/?$`,                                  // API version check
		`^/v2/[^/]+/[^/]+/manifests/[^/]+$`,        // Manifests
		`^/v2/[^/]+/[^/]+/blobs/sha256:[a-f0-9]+$`, // Blobs
		`^/v2/[^/]+/[^/]+/tags/list$`,              // Tag lists
		`^/v2/_catalog$`,                           // Catalog
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, path); matched {
			return true
		}
	}

	return false
}

// NormalizePath normalizes Docker registry path
func (d *Driver) NormalizePath(path string) (string, error) {
	if d.normalizer != nil {
		return d.normalizer.NormalizePath(d.Type(), path)
	}

	// Basic normalization
	normalized := strings.TrimSpace(path)
	if !strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}

	// Ensure v2 API prefix
	if normalized == "/" {
		normalized = "/v2/"
	}

	return normalized, nil
}

// BuildUpstreamURL builds upstream URL for Docker registry request
func (d *Driver) BuildUpstreamURL(req *ports.DriverRequest) (string, error) {
	registry := req.Repository
	if registry == "" {
		registry = common.DefaultRepoDockerHub // Default to Docker Hub
	}

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

// FetchPackage fetches package from upstream Docker registry
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

	// Add Docker registry-specific headers
	headers["User-Agent"] = "ProxyND/1.0 Registry-Proxy"

	// Accept appropriate content types based on path
	if strings.Contains(req.Path, "/manifests/") {
		headers["Accept"] = "application/vnd.docker.distribution.manifest.v2+json, " +
			"application/vnd.docker.distribution.manifest.v1+json"
	} else if strings.Contains(req.Path, "/blobs/") {
		headers["Accept"] = "application/octet-stream"
	} else {
		headers["Accept"] = common.MimeApplicationJSON
	}

	// Add authentication if configured
	registry := req.Repository
	if registry == "" {
		registry = "dockerhub"
	}

	if registryConfig, exists := d.config.Registries[registry]; exists && registryConfig.Auth != nil {
		// TODO: Implement Docker registry authentication (Bearer token)
		// This will be handled by the unified HTTP client
		_ = registryConfig // Placeholder to avoid empty branch warning
	}

	// Fetch from upstream
	httpResp, err := d.httpClient.Get(ctx, upstreamURL, headers)
	if err != nil {
		return nil, d.mapError(err)
	}

	// Convert to driver response
	response := &ports.DriverResponse{
		Content:      httpResp.Body,
		ContentType:  getContentType(req.Path, httpResp.Headers),
		Size:         httpResp.Size,
		Headers:      httpResp.Headers,
		StatusCode:   httpResp.StatusCode,
		LastModified: time.Now(), // TODO: Parse from headers
	}

	return response, nil
}

// ParseMetadata parses Docker registry metadata from content
func (d *Driver) ParseMetadata(content []byte) (*ports.PackageMetadata, error) {
	var manifest Manifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &ports.PackageMetadata{
		Name:      "docker-image",
		Version:   "unknown",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Attributes: map[string]string{
			"schema_version": fmt.Sprintf("%d", manifest.SchemaVersion),
			"media_type":     manifest.MediaType,
		},
	}, nil
}

// ValidateSignature validates Docker registry signatures
func (d *Driver) ValidateSignature(content, signature []byte) error {
	if d.verifier != nil {
		return d.verifier.VerifySignature(d.Type(), content, signature)
	}

	// TODO: Implement Docker-specific signature validation
	// Docker uses content trust and notary for signing
	return nil
}

// GetCacheKey generates cache key for Docker registry request
func (d *Driver) GetCacheKey(req *ports.DriverRequest) string {
	registry := req.Repository
	if registry == "" {
		registry = "dockerhub"
	}

	normalizedPath, _ := d.NormalizePath(req.Path)
	return fmt.Sprintf("registry:%s:%s", registry, strings.ReplaceAll(normalizedPath, "/", "_"))
}

// ShouldCache determines if response should be cached
func (d *Driver) ShouldCache(resp *ports.DriverResponse) bool {
	// Don't cache error responses
	if resp.StatusCode >= 400 {
		return false
	}

	// Don't cache API version checks
	if strings.Contains(resp.Headers["Content-Type"], "application/json") &&
		resp.Size < 100 {
		return false
	}

	return true
}

// GetCacheTTL returns cache TTL for Docker registry response
func (d *Driver) GetCacheTTL(resp *ports.DriverResponse) time.Duration {
	// Blobs are immutable and can be cached longer
	if strings.Contains(resp.ContentType, "application/octet-stream") {
		return 7 * 24 * time.Hour // 1 week
	}

	// Manifests and tag lists expire faster
	if strings.Contains(resp.ContentType, "application/json") {
		return 6 * time.Hour
	}

	return d.config.CacheTTL
}

// LoadConfig loads Docker registry driver configuration
func (d *Driver) LoadConfig() error {
	// TODO: Load configuration from file or environment
	return nil
}

// ValidateConfig validates Docker registry driver configuration
func (d *Driver) ValidateConfig() error {
	if d.config == nil {
		return fmt.Errorf("registry config is nil")
	}

	if !d.config.Enabled {
		return fmt.Errorf("registry driver is disabled")
	}

	if len(d.config.Registries) == 0 {
		return fmt.Errorf("no registries configured")
	}

	// Validate each registry
	for name, registry := range d.config.Registries {
		if registry.URL == "" {
			return fmt.Errorf("registry '%s' has empty URL", name)
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

// getContentType returns content type based on path and response headers
func getContentType(path string, headers map[string]string) string {
	// Use response content type if available
	if ct, exists := headers["Content-Type"]; exists && ct != "" {
		return ct
	}

	// Determine based on path
	if strings.Contains(path, "/manifests/") {
		return "application/vnd.docker.distribution.manifest.v2+json"
	} else if strings.Contains(path, "/blobs/") {
		return "application/octet-stream"
	} else if strings.Contains(path, "/tags/list") || strings.Contains(path, "/_catalog") {
		return "application/json"
	}

	return "application/json"
}
