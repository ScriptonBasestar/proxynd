package apt

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strings"

	"proxynd/internal/domain/common"
)

// Domain implements APT-specific proxy domain logic
type Domain struct {
	packagePattern *regexp.Regexp
	metadataFiles  map[string]bool
}

// NewDomain creates a new APT domain instance
func NewDomain() *Domain {
	return &Domain{
		// Pattern for .deb packages: pool/component/p/package/package_version_arch.deb
		packagePattern: regexp.MustCompile(`pool/[^/]+/[^/]+/[^/]+/([^/]+)_([^_]+)_([^.]+)\.deb$`),
		metadataFiles: map[string]bool{
			"Release":        true,
			"Release.gpg":    true,
			"InRelease":      true,
			"Packages":       true,
			"Packages.gz":    true,
			"Packages.xz":    true,
			"Packages.bz2":   true,
			"Sources":        true,
			"Sources.gz":     true,
			"Sources.xz":     true,
			"Sources.bz2":    true,
			"Contents":       true,
			"Contents.gz":    true,
			"Translation":    true,
			"Translation.gz": true,
		},
	}
}

// ParseRequest parses and validates an APT proxy request
func (d *Domain) ParseRequest(ctx context.Context, req *common.ProxyRequest) error {
	if req.Method != "GET" && req.Method != "HEAD" {
		return fmt.Errorf("unsupported method for APT proxy: %s", req.Method)
	}

	// Validate path
	if err := d.ValidatePath(req.Path); err != nil {
		return err
	}

	return nil
}

// BuildUpstreamURL constructs the upstream URL for an APT request
func (d *Domain) BuildUpstreamURL(repo common.Repository, path string) string {
	// Ensure base URL doesn't end with slash
	baseURL := strings.TrimRight(repo.URL, "/")

	// Ensure path starts with slash
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return baseURL + path
}

// ValidatePath validates if the requested path is valid for APT
func (d *Domain) ValidatePath(requestPath string) error {
	// Remove leading slash for pattern matching
	cleanPath := strings.TrimPrefix(requestPath, "/")

	// Check if it's a package file
	if strings.Contains(cleanPath, "/pool/") && strings.HasSuffix(cleanPath, ".deb") {
		return nil
	}

	// Check if it's a metadata file
	if d.isMetadataFile(cleanPath) {
		return nil
	}

	// Check for GPG keys
	if strings.HasSuffix(cleanPath, ".gpg") || strings.HasSuffix(cleanPath, ".key") {
		return nil
	}

	// Check for by-hash access
	if strings.Contains(cleanPath, "/by-hash/") {
		return nil
	}

	return fmt.Errorf("invalid APT path: %s", requestPath)
}

// ExtractMetadata extracts package metadata from the APT request path
func (d *Domain) ExtractMetadata(requestPath string) (*common.PackageMetadata, error) {
	cleanPath := strings.TrimPrefix(requestPath, "/")

	// Try to match package pattern
	matches := d.packagePattern.FindStringSubmatch(cleanPath)
	if len(matches) != 4 {
		return nil, fmt.Errorf("cannot extract metadata from path: %s", requestPath)
	}

	packageName := matches[1]
	version := matches[2]
	architecture := matches[3]

	return &common.PackageMetadata{
		Name:         packageName,
		Version:      version,
		Architecture: architecture,
	}, nil
}

// TransformResponse transforms the upstream response if needed
func (d *Domain) TransformResponse(ctx context.Context, resp *common.ProxyResponse) error {
	// APT responses typically don't need transformation
	// This could be extended to modify repository URLs in Release files
	return nil
}

// GetContentType returns the content type for a given APT file
func (d *Domain) GetContentType(filename string) string {
	ext := strings.ToLower(path.Ext(filename))
	base := path.Base(filename)

	// Check specific files
	switch base {
	case "Release", "InRelease", "Packages", "Sources", "Contents":
		return "text/plain"
	}

	// Check by extension
	switch ext {
	case ".deb":
		return "application/vnd.debian.binary-package"
	case ".gz":
		return "application/gzip"
	case ".xz":
		return "application/x-xz"
	case ".bz2":
		return "application/x-bzip2"
	case ".gpg", ".key":
		return "application/pgp-signature"
	default:
		return "application/octet-stream"
	}
}

// ShouldCache determines if an APT response should be cached
func (d *Domain) ShouldCache(requestPath string, resp *common.ProxyResponse) bool {
	// Don't cache error responses
	if resp.StatusCode >= 400 {
		return false
	}

	// Don't cache InRelease files (they change frequently)
	if strings.HasSuffix(requestPath, "InRelease") {
		return false
	}

	// Don't cache Release files
	if strings.HasSuffix(requestPath, "Release") && !strings.HasSuffix(requestPath, ".gpg") {
		return false
	}

	// Cache packages and other files
	return true
}

// isMetadataFile checks if a path is an APT metadata file
func (d *Domain) isMetadataFile(path string) bool {
	// Check if path contains dists/ (metadata files are in dists/)
	if !strings.Contains(path, "/dists/") {
		return false
	}

	// Get the filename
	filename := path
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		filename = path[idx+1:]
	}

	// Check against known metadata files
	return d.metadataFiles[filename]
}

// ParsePackagePath parses an APT package path into its components
func (d *Domain) ParsePackagePath(pkgPath string) (*PackageInfo, error) {
	cleanPath := strings.TrimPrefix(pkgPath, "/")

	matches := d.packagePattern.FindStringSubmatch(cleanPath)
	if len(matches) != 4 {
		return nil, fmt.Errorf("invalid package path: %s", pkgPath)
	}

	// Extract component from path
	// Format: pool/component/...
	parts := strings.Split(cleanPath, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid package path format: %s", pkgPath)
	}

	component := parts[1]

	return &PackageInfo{
		Name:         matches[1],
		Version:      matches[2],
		Architecture: matches[3],
		Component:    component,
		Filename:     path.Base(cleanPath),
	}, nil
}

// ParseDistribution extracts distribution information from path
func (d *Domain) ParseDistribution(path string) (*DistInfo, error) {
	// Format: dists/distribution/...
	if !strings.Contains(path, "/dists/") {
		return nil, fmt.Errorf("not a distribution path: %s", path)
	}

	parts := strings.Split(path, "/")
	distsIdx := -1

	// Find "dists" in path
	for i, part := range parts {
		if part == "dists" {
			distsIdx = i
			break
		}
	}

	if distsIdx == -1 || distsIdx+1 >= len(parts) {
		return nil, fmt.Errorf("invalid distribution path: %s", path)
	}

	distribution := parts[distsIdx+1]

	// Extract component if present
	component := ""
	if distsIdx+2 < len(parts) {
		component = parts[distsIdx+2]
	}

	return &DistInfo{
		Distribution: distribution,
		Component:    component,
	}, nil
}

// PackageInfo represents parsed APT package information
type PackageInfo struct {
	Name         string
	Version      string
	Architecture string
	Component    string
	Filename     string
}

// DistInfo represents APT distribution information
type DistInfo struct {
	Distribution string
	Component    string
}

// Ensure Domain implements the ProxyDomain interface
var _ common.ProxyDomain = (*Domain)(nil)
