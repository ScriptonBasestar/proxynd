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

// ParseMetadata parses Maven metadata from content
func (d *Driver) ParseMetadata(content []byte) (*ports.PackageMetadata, error) {
	// Detect content type
	contentStr := string(content)

	// Check if it's maven-metadata.xml
	if strings.Contains(contentStr, "<metadata") && strings.Contains(contentStr, "<groupId>") {
		return d.parseMavenMetadataXML(content)
	}

	// Check if it's pom.xml
	if strings.Contains(contentStr, "<project") && strings.Contains(contentStr, "<artifactId>") {
		return d.parsePOMXML(content)
	}

	// Check if it's a JAR file (ZIP archive)
	if len(content) > 4 && string(content[0:2]) == "PK" {
		return &ports.PackageMetadata{
			Name:        "java-archive",
			Description: "Java JAR/WAR/EAR package",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}, nil
	}

	// Unknown metadata format
	return &ports.PackageMetadata{
		Name:        "unknown",
		Description: "Maven metadata",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

// ValidateSignature validates Maven package signatures
func (d *Driver) ValidateSignature(content, signature []byte) error {
	// Use the common signature verifier
	if d.verifier != nil {
		return d.verifier.VerifySignature(d.Type(), content, signature)
	}

	// Maven uses MD5, SHA1, SHA256, SHA512, PGP signatures
	// If no verifier is configured, perform basic validation
	if len(signature) == 0 {
		// No signature provided
		return nil
	}

	// Basic PGP signature format validation
	sigStr := string(signature)
	if strings.Contains(sigStr, "BEGIN PGP SIGNATURE") {
		// PGP signature found, but no verifier configured
		// Return nil to indicate signature format is valid but not verified
		return nil
	}

	// Try hash-based signature validation (MD5, SHA1, SHA256, SHA512)
	return d.validateHashSignature(content, signature)
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

// parseMavenMetadataXML parses maven-metadata.xml
func (d *Driver) parseMavenMetadataXML(content []byte) (*ports.PackageMetadata, error) {
	// maven-metadata.xml format:
	// <metadata>
	//   <groupId>...</groupId>
	//   <artifactId>...</artifactId>
	//   <version>...</version>
	// </metadata>

	contentStr := string(content)

	metadata := &ports.PackageMetadata{
		Name:        "maven-metadata",
		Description: "Maven repository metadata",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Attributes:  make(map[string]string),
	}

	// Extract groupId
	if start := strings.Index(contentStr, "<groupId>"); start != -1 {
		start += 9
		if end := strings.Index(contentStr[start:], "</groupId>"); end != -1 {
			metadata.Attributes["group_id"] = contentStr[start : start+end]
		}
	}

	// Extract artifactId
	if start := strings.Index(contentStr, "<artifactId>"); start != -1 {
		start += 12
		if end := strings.Index(contentStr[start:], "</artifactId>"); end != -1 {
			metadata.Name = contentStr[start : start+end]
		}
	}

	// Extract version (latest or release)
	if start := strings.Index(contentStr, "<release>"); start != -1 {
		start += 9
		if end := strings.Index(contentStr[start:], "</release>"); end != -1 {
			metadata.Version = contentStr[start : start+end]
		}
	} else if start := strings.Index(contentStr, "<latest>"); start != -1 {
		start += 8
		if end := strings.Index(contentStr[start:], "</latest>"); end != -1 {
			metadata.Version = contentStr[start : start+end]
		}
	}

	return metadata, nil
}

// parsePOMXML parses pom.xml
func (d *Driver) parsePOMXML(content []byte) (*ports.PackageMetadata, error) {
	// pom.xml format:
	// <project>
	//   <groupId>...</groupId>
	//   <artifactId>...</artifactId>
	//   <version>...</version>
	//   <description>...</description>
	// </project>

	contentStr := string(content)

	metadata := &ports.PackageMetadata{
		Name:        "unknown",
		Description: "Maven POM",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Attributes:  make(map[string]string),
	}

	// Extract groupId
	if start := strings.Index(contentStr, "<groupId>"); start != -1 {
		start += 9
		if end := strings.Index(contentStr[start:], "</groupId>"); end != -1 {
			metadata.Attributes["group_id"] = contentStr[start : start+end]
		}
	}

	// Extract artifactId
	if start := strings.Index(contentStr, "<artifactId>"); start != -1 {
		start += 12
		if end := strings.Index(contentStr[start:], "</artifactId>"); end != -1 {
			metadata.Name = contentStr[start : start+end]
		}
	}

	// Extract version
	if start := strings.Index(contentStr, "<version>"); start != -1 {
		start += 9
		if end := strings.Index(contentStr[start:], "</version>"); end != -1 {
			metadata.Version = contentStr[start : start+end]
		}
	}

	// Extract description
	if start := strings.Index(contentStr, "<description>"); start != -1 {
		start += 13
		if end := strings.Index(contentStr[start:], "</description>"); end != -1 {
			metadata.Description = contentStr[start : start+end]
		}
	}

	return metadata, nil
}

// validateHashSignature validates hash-based signatures using common utility
func (d *Driver) validateHashSignature(content, signature []byte) error {
	return common.ValidateHashSignature(content, signature)
}
