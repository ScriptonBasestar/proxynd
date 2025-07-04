package proxy

//go:generate mockery --name ProxyService --output ./mocks --outpkg mocks --filename proxy_service.go
//go:generate mockery --name CacheService --output ./mocks --outpkg mocks --filename cache_service.go
//go:generate mockery --name ConfigService --output ./mocks --outpkg mocks --filename config_service.go
//go:generate mockery --name UpstreamClient --output ./mocks --outpkg mocks --filename upstream_client.go

import (
	"context"
	"io"
)

// ProxyRequest represents a proxy request
type ProxyRequest struct {
	Path       string
	ProxyType  string
	Method     string
	Headers    map[string]string
	RemoteAddr string
}

// ProxyResponse represents a proxy response
type ProxyResponse struct {
	Body        io.ReadCloser
	StatusCode  int
	Headers     map[string]string
	ContentType string
	FileName    string
	Cached      bool
}

// ProxyService defines the interface for proxy services
type ProxyService interface {
	// HandleRequest processes a proxy request
	HandleRequest(ctx context.Context, req ProxyRequest) (*ProxyResponse, error)
	
	// ValidateRequest validates if the request is valid
	ValidateRequest(req ProxyRequest) error
	
	// GetProxyType returns the type of proxy (apt, maven, npm, etc.)
	GetProxyType() string
}

// CacheService defines the interface for cache operations
type CacheService interface {
	// Get retrieves content from cache
	Get(ctx context.Context, key string) (io.ReadCloser, bool, error)
	
	// Put stores content in cache
	Put(ctx context.Context, key string, content io.Reader) error
	
	// Exists checks if a key exists in cache
	Exists(ctx context.Context, key string) (bool, error)
	
	// Delete removes content from cache
	Delete(ctx context.Context, key string) error
}

// ConfigService defines the interface for configuration operations
type ConfigService interface {
	// GetProxyConfig returns configuration for a specific proxy type
	GetProxyConfig(ctx context.Context, proxyType string) (interface{}, error)
	
	// GetGlobalConfig returns global configuration
	GetGlobalConfig(ctx context.Context) (interface{}, error)
	
	// ReloadConfig reloads configuration from disk
	ReloadConfig(ctx context.Context) error
}

// UpstreamClient defines the interface for upstream requests
type UpstreamClient interface {
	// Fetch retrieves content from upstream
	Fetch(ctx context.Context, url string, headers map[string]string) (*ProxyResponse, error)
}