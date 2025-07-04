package common

import (
	"context"
	"io"
)

// PackageMetadata represents metadata about a package
type PackageMetadata struct {
	Name         string
	Version      string
	Architecture string
	Description  string
	Size         int64
	Checksum     string
	Dependencies []string
}

// Repository represents an upstream repository
type Repository struct {
	Name      string
	URL       string
	Enabled   bool
	Priority  int
	AuthType  string
	Username  string
	Password  string
	Token     string
}

// ProxyRequest represents a request to proxy
type ProxyRequest struct {
	Method      string
	Path        string
	QueryParams map[string]string
	Headers     map[string]string
	Body        io.Reader
}

// ProxyResponse represents a response from proxy
type ProxyResponse struct {
	StatusCode  int
	Headers     map[string]string
	Body        io.ReadCloser
	ContentType string
	Size        int64
}

// ProxyDomain defines the interface for proxy domain logic
type ProxyDomain interface {
	// ParseRequest parses and validates a proxy request
	ParseRequest(ctx context.Context, req *ProxyRequest) error
	
	// BuildUpstreamURL constructs the upstream URL for a request
	BuildUpstreamURL(repo Repository, path string) string
	
	// ValidatePath validates if the requested path is valid for this proxy type
	ValidatePath(path string) error
	
	// ExtractMetadata extracts package metadata from the request path
	ExtractMetadata(path string) (*PackageMetadata, error)
	
	// TransformResponse transforms the upstream response if needed
	TransformResponse(ctx context.Context, resp *ProxyResponse) error
	
	// GetContentType returns the content type for a given file
	GetContentType(filename string) string
	
	// ShouldCache determines if a response should be cached
	ShouldCache(path string, resp *ProxyResponse) bool
}

// CacheKeyBuilder builds cache keys for different proxy types
type CacheKeyBuilder interface {
	// BuildKey builds a cache key from request parameters
	BuildKey(proxyType, path string, params map[string]string) string
	
	// ParseKey parses a cache key back to its components
	ParseKey(key string) (proxyType, path string, params map[string]string, err error)
}

// PathParser parses package paths for different proxy types
type PathParser interface {
	// Parse parses a path and extracts relevant information
	Parse(path string) (map[string]string, error)
	
	// IsValid checks if a path is valid
	IsValid(path string) bool
}