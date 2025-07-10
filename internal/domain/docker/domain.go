package docker

import (
	"context"
	"fmt"
	"strings"

	"proxynd/internal/domain/common"
)

// Domain implements Docker registry-specific proxy domain logic
type Domain struct{}

// NewDomain creates a new Docker domain instance
func NewDomain() *Domain {
	return &Domain{}
}

// ParseRequest parses and validates a Docker proxy request
func (d *Domain) ParseRequest(ctx context.Context, req *common.ProxyRequest) error {
	// Docker registry supports GET, HEAD, PUT, POST, DELETE
	validMethods := map[string]bool{
		"GET":    true,
		"HEAD":   true,
		"PUT":    true,
		"POST":   true,
		"DELETE": true,
	}

	if !validMethods[req.Method] {
		return fmt.Errorf("unsupported method for Docker proxy: %s", req.Method)
	}

	return d.ValidatePath(req.Path)
}

// BuildUpstreamURL constructs the upstream URL for a Docker request
func (d *Domain) BuildUpstreamURL(repo common.Repository, path string) string {
	baseURL := strings.TrimRight(repo.URL, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return baseURL + path
}

// ValidatePath validates if the requested path is valid for Docker
func (d *Domain) ValidatePath(path string) error {
	// Docker registry API v2 paths
	if strings.HasPrefix(path, "/v2/") {
		return nil
	}

	// Legacy v1 API (deprecated but may still be used)
	if strings.HasPrefix(path, "/v1/") {
		return nil
	}

	return fmt.Errorf("invalid Docker registry path: %s", path)
}

// ExtractMetadata extracts package metadata from the Docker request path
func (d *Domain) ExtractMetadata(path string) (*common.PackageMetadata, error) {
	// Docker paths: /v2/library/ubuntu/manifests/latest
	// Extract image name and tag/digest
	if strings.HasPrefix(path, "/v2/") {
		parts := strings.Split(strings.TrimPrefix(path, "/v2/"), "/")
		if len(parts) >= 3 && parts[len(parts)-2] == "manifests" {
			imageName := strings.Join(parts[:len(parts)-2], "/")
			reference := parts[len(parts)-1]

			return &common.PackageMetadata{
				Name:    imageName,
				Version: reference,
			}, nil
		}
	}

	return nil, fmt.Errorf("cannot extract metadata from path: %s", path)
}

// TransformResponse transforms the upstream response if needed
func (d *Domain) TransformResponse(ctx context.Context, resp *common.ProxyResponse) error {
	// Docker responses typically don't need transformation
	return nil
}

// GetContentType returns the content type for a given Docker registry response
func (d *Domain) GetContentType(filename string) string {
	// Docker uses specific content types
	if strings.Contains(filename, "manifest") {
		return "application/vnd.docker.distribution.manifest.v2+json"
	}
	if strings.Contains(filename, "config") {
		return "application/vnd.docker.container.image.v1+json"
	}
	return "application/octet-stream"
}

// ShouldCache determines if a Docker response should be cached
func (d *Domain) ShouldCache(path string, resp *common.ProxyResponse) bool {
	// Don't cache error responses
	if resp.StatusCode >= 400 {
		return false
	}

	// Cache blobs (immutable)
	if strings.Contains(path, "/blobs/") {
		return true
	}

	// Don't cache manifests tagged as "latest"
	if strings.Contains(path, "/manifests/latest") {
		return false
	}

	// Cache other manifests
	if strings.Contains(path, "/manifests/") {
		return true
	}

	return true
}

// Ensure Domain implements the ProxyDomain interface
var _ common.ProxyDomain = (*Domain)(nil)
