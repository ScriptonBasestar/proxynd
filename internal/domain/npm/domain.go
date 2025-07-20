// Package npm provides NPM domain logic and models
package npm

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strings"

	"proxynd/internal/domain/common"
)

// Domain implements NPM-specific proxy domain logic
type Domain struct {
	packagePattern *regexp.Regexp
	tarballPattern *regexp.Regexp
	scopePattern   *regexp.Regexp
}

// NewDomain creates a new NPM domain instance
func NewDomain() *Domain {
	return &Domain{
		// Pattern for NPM package metadata: /package-name
		packagePattern: regexp.MustCompile(`^/?(@[^/]+/)?([^/]+)/?$`),
		// Pattern for NPM tarballs: /package-name/-/package-name-version.tgz
		tarballPattern: regexp.MustCompile(`^/?(@[^/]+/)?([^/]+)/-/([^/]+)\.tgz$`),
		// Pattern for scoped packages: @scope/package
		scopePattern: regexp.MustCompile(`^@([^/]+)/(.+)$`),
	}
}

// ParseRequest parses and validates an NPM proxy request
func (d *Domain) ParseRequest(_ context.Context, req *common.ProxyRequest) error {
	if req.Method != "GET" && req.Method != "HEAD" {
		return fmt.Errorf("unsupported method for NPM proxy: %s", req.Method)
	}

	// Validate path
	if err := d.ValidatePath(req.Path); err != nil {
		return err
	}

	return nil
}

// BuildUpstreamURL constructs the upstream URL for an NPM request
func (d *Domain) BuildUpstreamURL(repo common.Repository, path string) string {
	// Ensure base URL doesn't end with slash
	baseURL := strings.TrimRight(repo.URL, "/")

	// Ensure path starts with slash
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return baseURL + path
}

// ValidatePath validates if the requested path is valid for NPM
func (d *Domain) ValidatePath(requestPath string) error {
	// Remove leading slash for pattern matching
	cleanPath := strings.TrimPrefix(requestPath, "/")

	// Special NPM registry paths
	if cleanPath == "" || cleanPath == "-/all" || cleanPath == "-/v1/search" {
		return nil
	}

	// Check if it's a package metadata request
	if d.packagePattern.MatchString(cleanPath) {
		return nil
	}

	// Check if it's a tarball request
	if d.tarballPattern.MatchString(cleanPath) {
		return nil
	}

	// Check for special NPM endpoints
	if strings.HasPrefix(cleanPath, "-/") {
		// Various NPM registry endpoints like -/user, -/package, etc.
		return nil
	}

	return fmt.Errorf("invalid NPM path: %s", requestPath)
}

// ExtractMetadata extracts package metadata from the NPM request path
func (d *Domain) ExtractMetadata(requestPath string) (*common.PackageMetadata, error) {
	cleanPath := strings.TrimPrefix(requestPath, "/")

	// Try to match tarball pattern first (more specific)
	if matches := d.tarballPattern.FindStringSubmatch(cleanPath); len(matches) > 0 {
		scope := strings.TrimPrefix(matches[1], "@")
		packageName := matches[2]
		filename := matches[3]

		// Extract version from filename
		// Format: package-name-version.tgz
		version := ""
		if strings.HasSuffix(filename, ".tgz") {
			versionPart := strings.TrimSuffix(filename, ".tgz")
			// Remove package name prefix
			if strings.HasPrefix(versionPart, packageName+"-") {
				version = strings.TrimPrefix(versionPart, packageName+"-")
			}
		}

		fullName := packageName
		if scope != "" {
			fullName = "@" + scope + "/" + packageName
		}

		return &common.PackageMetadata{
			Name:    fullName,
			Version: version,
		}, nil
	}

	// Try to match package pattern
	if matches := d.packagePattern.FindStringSubmatch(cleanPath); len(matches) > 0 {
		scope := strings.TrimPrefix(matches[1], "@")
		packageName := matches[2]

		fullName := packageName
		if scope != "" {
			fullName = "@" + scope + "/" + packageName
		}

		return &common.PackageMetadata{
			Name: fullName,
			// Version not available in metadata requests
		}, nil
	}

	return nil, fmt.Errorf("cannot extract metadata from path: %s", requestPath)
}

// TransformResponse transforms the upstream response if needed
func (d *Domain) TransformResponse(_ context.Context, _ *common.ProxyResponse) error {
	// NPM responses may need transformation to update tarball URLs
	// This would require parsing and modifying the JSON response
	// For now, we'll leave responses as-is
	return nil
}

// GetContentType returns the content type for a given NPM file
func (d *Domain) GetContentType(filename string) string {
	ext := strings.ToLower(path.Ext(filename))

	switch ext {
	case ".json":
		return "application/json"
	case ".tgz":
		return "application/gzip"
	case ".tar":
		return "application/x-tar"
	default:
		// NPM package metadata is JSON even without extension
		if !strings.Contains(filename, ".") {
			return "application/json"
		}
		return "application/octet-stream"
	}
}

// ShouldCache determines if an NPM response should be cached
func (d *Domain) ShouldCache(requestPath string, resp *common.ProxyResponse) bool {
	// Don't cache error responses
	if resp.StatusCode >= 400 {
		return false
	}

	// Don't cache search results
	if strings.Contains(requestPath, "/-/v1/search") {
		return false
	}

	// Don't cache user-specific endpoints
	if strings.Contains(requestPath, "/-/user") {
		return false
	}

	// Cache tarballs (immutable)
	if strings.HasSuffix(requestPath, ".tgz") {
		return true
	}

	// Cache package metadata with short TTL
	// (handled by cache service TTL settings)
	return true
}

// ParsePackageName extracts package name components
func (d *Domain) ParsePackageName(name string) (*PackageNameInfo, error) {
	info := &PackageNameInfo{}

	// Check if it's a scoped package
	if matches := d.scopePattern.FindStringSubmatch(name); len(matches) == 3 {
		info.Scope = matches[1]
		info.Name = matches[2]
		info.FullName = name
	} else {
		info.Name = name
		info.FullName = name
	}

	return info, nil
}

// ParseTarballPath parses an NPM tarball path
func (d *Domain) ParseTarballPath(path string) (*TarballInfo, error) {
	cleanPath := strings.TrimPrefix(path, "/")

	matches := d.tarballPattern.FindStringSubmatch(cleanPath)
	if len(matches) == 0 {
		return nil, fmt.Errorf("invalid tarball path: %s", path)
	}

	scope := strings.TrimPrefix(matches[1], "@")
	packageName := matches[2]
	filename := matches[3]

	// Extract version from filename
	version := ""
	if strings.HasSuffix(filename, ".tgz") {
		versionPart := strings.TrimSuffix(filename, ".tgz")
		if strings.HasPrefix(versionPart, packageName+"-") {
			version = strings.TrimPrefix(versionPart, packageName+"-")
		}
	}

	fullName := packageName
	if scope != "" {
		fullName = "@" + scope + "/" + packageName
	}

	return &TarballInfo{
		Scope:       scope,
		PackageName: packageName,
		FullName:    fullName,
		Version:     version,
		Filename:    filename,
	}, nil
}

// PackageNameInfo represents parsed NPM package name information
type PackageNameInfo struct {
	Scope    string // Without @ prefix
	Name     string // Package name without scope
	FullName string // Full name including scope
}

// TarballInfo represents parsed NPM tarball information
type TarballInfo struct {
	Scope       string
	PackageName string
	FullName    string
	Version     string
	Filename    string
}

// Ensure Domain implements the ProxyDomain interface
var _ common.ProxyDomain = (*Domain)(nil)
