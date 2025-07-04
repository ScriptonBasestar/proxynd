package types

import (
	"context"
	"io"

	"github.com/gofiber/fiber/v2"
)

// ProxyType represents the type of proxy
type ProxyType string

// Supported proxy types
const (
	ProxyTypeAPT    ProxyType = "apt"
	ProxyTypeMaven  ProxyType = "maven"
	ProxyTypeNPM    ProxyType = "npm"
	ProxyTypeDocker ProxyType = "docker"
	ProxyTypePIP    ProxyType = "pip"
	ProxyTypeYUM    ProxyType = "yum"
	ProxyTypeAPK    ProxyType = "apk"
)

// ProxyRequest represents a standardized proxy request
type ProxyRequest struct {
	Type        ProxyType
	Method      string
	Path        string
	Headers     map[string]string
	QueryParams map[string]string
	Body        io.Reader
	RemoteAddr  string
}

// ProxyResponse represents a standardized proxy response
type ProxyResponse struct {
	StatusCode  int
	Headers     map[string]string
	Body        io.ReadCloser
	ContentType string
	Cached      bool
	CacheKey    string
}

// ProxyHandler defines the interface that all proxy handlers must implement
type ProxyHandler interface {
	// Handle processes a proxy request and returns a response
	Handle(ctx context.Context, req *ProxyRequest) (*ProxyResponse, error)

	// GetType returns the proxy type this handler serves
	GetType() ProxyType

	// ValidateRequest validates if the request is valid for this proxy type
	ValidateRequest(req *ProxyRequest) error

	// IsHealthy checks if the proxy handler is healthy
	IsHealthy(ctx context.Context) bool
}

// ProxyHandlerFactory creates proxy handlers
type ProxyHandlerFactory interface {
	// CreateHandler creates a handler for the specified proxy type
	CreateHandler(proxyType ProxyType) (ProxyHandler, error)

	// GetSupportedTypes returns all supported proxy types
	GetSupportedTypes() []ProxyType

	// RegisterHandler registers a new proxy handler type
	RegisterHandler(proxyType ProxyType, creator func() ProxyHandler) error
}

// ProxyMiddleware defines middleware that can be applied to proxy handlers
type ProxyMiddleware interface {
	// Process wraps a proxy handler with middleware logic
	Process(next ProxyHandler) ProxyHandler
}

// FiberProxyAdapter adapts a ProxyHandler to work with Fiber
type FiberProxyAdapter interface {
	// Adapt converts a ProxyHandler to a Fiber handler
	Adapt(handler ProxyHandler) fiber.Handler
}

// ProxyMetrics defines metrics collection for proxy operations
type ProxyMetrics interface {
	// RecordRequest records a proxy request
	RecordRequest(proxyType ProxyType, method string, path string)

	// RecordResponse records a proxy response
	RecordResponse(proxyType ProxyType, statusCode int, cached bool, duration int64)

	// RecordError records a proxy error
	RecordError(proxyType ProxyType, errorType string)
}

// ProxyConfig represents configuration for a proxy handler
type ProxyConfig interface {
	// GetUpstreams returns configured upstream repositories
	GetUpstreams() []UpstreamConfig

	// GetCacheConfig returns cache configuration
	GetCacheConfig() CacheConfig

	// GetAuthConfig returns authentication configuration
	GetAuthConfig() AuthConfig

	// IsEnabled checks if the proxy is enabled
	IsEnabled() bool

	// Validate validates the configuration
	Validate() error
}

// UpstreamConfig represents an upstream repository configuration
type UpstreamConfig struct {
	Name     string
	URL      string
	Priority int
	Enabled  bool
	Auth     *AuthConfig
	Timeout  int // seconds
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	Enabled      bool
	TTL          int64 // seconds
	MaxSize      int64 // bytes
	CleanupCycle int   // minutes
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Type     string // "basic", "bearer", "oauth2"
	Username string
	Password string
	Token    string
	OAuth2   *OAuth2Config
}

// OAuth2Config represents OAuth2 configuration
type OAuth2Config struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	Scopes       []string
}

// ProxyError represents a proxy-specific error
type ProxyError struct {
	Type       ProxyType
	StatusCode int
	Message    string
	Err        error
}

// Error implements the error interface
func (e *ProxyError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *ProxyError) Unwrap() error {
	return e.Err
}

// NewProxyError creates a new proxy error
func NewProxyError(proxyType ProxyType, statusCode int, message string, err error) *ProxyError {
	return &ProxyError{
		Type:       proxyType,
		StatusCode: statusCode,
		Message:    message,
		Err:        err,
	}
}