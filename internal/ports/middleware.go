package ports

import (
	"context"
	"time"
)

// MiddlewareOrder defines the standard middleware execution order
type MiddlewareOrder int

const (
	OrderRecovery   MiddlewareOrder = 10 // Error recovery and panic handling
	OrderAuth       MiddlewareOrder = 20 // Authentication
	OrderPermission MiddlewareOrder = 30 // Authorization/Permission checks
	OrderRateLimit  MiddlewareOrder = 40 // Rate limiting
	OrderRequestID  MiddlewareOrder = 50 // Request ID generation
	OrderLogging    MiddlewareOrder = 60 // Access logging
	OrderMetrics    MiddlewareOrder = 70 // Metrics collection
	OrderTracing    MiddlewareOrder = 80 // Distributed tracing
)

// EnhancedHTTPMiddleware extends HTTPMiddleware with additional metadata
type EnhancedHTTPMiddleware interface {
	HTTPMiddleware

	// Order returns the execution order (lower numbers execute first)
	Order() MiddlewareOrder

	// Name returns the middleware name for debugging
	Name() string

	// IsEnabled returns whether this middleware is enabled
	IsEnabled() bool
}

// MiddlewareChain manages a collection of middleware
type MiddlewareChain interface {
	// Add adds middleware to the chain
	Add(middleware EnhancedHTTPMiddleware)

	// Build builds the final handler chain
	Build(finalHandler HTTPHandler) HTTPHandler

	// GetMiddlewares returns all middleware in execution order
	GetMiddlewares() []EnhancedHTTPMiddleware

	// Count returns the number of middleware in the chain
	Count() int
}

// RateLimiter defines rate limiting interface
type RateLimiter interface {
	// Allow checks if a request should be allowed
	Allow(ctx context.Context, key string) (bool, error)

	// AllowN checks if n requests should be allowed
	AllowN(ctx context.Context, key string, n int) (bool, error)

	// Reserve reserves n tokens for a key
	Reserve(ctx context.Context, key string, n int) (*Reservation, error)

	// Reset resets the rate limiter for a key
	Reset(ctx context.Context, key string) error

	// CheckLimit checks if a request should be rate limited
	CheckLimit(ctx context.Context, req *RateLimitRequest) error

	// GetConfig returns the current rate limit configuration
	GetConfig() *RateLimitConfig
}

// Reservation represents a rate limit reservation
type Reservation struct {
	// OK indicates if the reservation was successful
	OK bool

	// Delay is the time to wait before the request can proceed
	Delay time.Duration

	// TimeToAct is the earliest time the action can be taken
	TimeToAct time.Time

	// Cancel cancels the reservation
	Cancel func()
}

// RateLimitRequest represents a rate limiting request
type RateLimitRequest struct {
	// Key is the rate limit key (e.g., IP address, user ID)
	Key string `json:"key"`

	// Resource is the resource being accessed
	Resource string `json:"resource"`

	// Count is the number of requests
	Count int `json:"count"`

	// ClientIP is the client IP address
	ClientIP string `json:"client_ip"`

	// UserID is the authenticated user ID (if available)
	UserID string `json:"user_id,omitempty"`
}

// RateLimitConfig defines rate limiting configuration
type RateLimitConfig struct {
	// RequestsPerSecond is the maximum number of requests per second
	RequestsPerSecond float64 `json:"requests_per_second" yaml:"requests_per_second"`

	// BurstSize is the maximum burst size
	BurstSize int `json:"burst_size" yaml:"burst_size"`

	// KeyGenerator generates rate limit keys from requests
	KeyGenerator func(ctx HTTPContext) string `json:"-" yaml:"-"`

	// SkipSuccessful skips rate limiting for successful requests
	SkipSuccessful bool `json:"skip_successful" yaml:"skip_successful"`

	// TrustedProxies is a list of trusted proxy IPs
	TrustedProxies []string `json:"trusted_proxies" yaml:"trusted_proxies"`
}

// RequestIDGenerator generates unique request IDs
type RequestIDGenerator interface {
	// Generate generates a new request ID
	Generate() string

	// Extract extracts request ID from HTTP context
	Extract(ctx HTTPContext) string

	// SetRequestID sets request ID in HTTP context
	SetRequestID(ctx HTTPContext, id string)
}

// MiddlewareAuthService extends AuthService with middleware-specific methods
type MiddlewareAuthService interface {
	AuthService

	// IsEnabled returns whether authentication is enabled
	IsEnabled() bool
}

// PermissionService defines authorization service interface
type PermissionService interface {
	// CheckPermission checks if a user has permission for a resource
	CheckPermission(ctx context.Context, userID, resource, action string) (bool, error)

	// CheckRole checks if a user has a specific role
	CheckRole(ctx context.Context, userID, role string) (bool, error)

	// GetUserPermissions returns all permissions for a user
	GetUserPermissions(ctx context.Context, userID string) ([]MiddlewarePermission, error)

	// IsEnabled returns whether authorization is enabled
	IsEnabled() bool
}

// UserInfo represents authenticated user information
type UserInfo struct {
	ID       string                 `json:"id"`
	Username string                 `json:"username"`
	Email    string                 `json:"email"`
	Roles    []string               `json:"roles"`
	Claims   map[string]interface{} `json:"claims"`
}

// JWTClaims represents JWT token claims
type JWTClaims struct {
	UserID    string                 `json:"user_id"`
	Username  string                 `json:"username"`
	Roles     []string               `json:"roles"`
	ExpiresAt time.Time              `json:"expires_at"`
	IssuedAt  time.Time              `json:"issued_at"`
	Claims    map[string]interface{} `json:"claims"`
}

// APIKeyInfo represents API key information
type APIKeyInfo struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	UserID      string     `json:"user_id"`
	Permissions []string   `json:"permissions"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// MiddlewarePermission represents a permission for middleware
type MiddlewarePermission struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
	Effect   string `json:"effect"` // allow, deny
}

// SignatureService defines package signature verification interface
type SignatureService interface {
	// Verify verifies package signature
	Verify(ctx context.Context, req *SignatureRequest) error

	// IsEnabled returns whether signature verification is enabled
	IsEnabled() bool
}

// SignatureRequest represents a signature verification request
type SignatureRequest struct {
	// PackagePath is the path to the package to verify
	PackagePath string `json:"package_path"`

	// SignaturePath is the path to the signature file
	SignaturePath string `json:"signature_path"`

	// PublicKeyPath is the path to the public key
	PublicKeyPath string `json:"public_key_path"`

	// Algorithm is the signature algorithm (gpg, rsa, etc.)
	Algorithm string `json:"algorithm"`

	// PackageType is the type of package (apt, yum, etc.)
	PackageType string `json:"package_type"`
}

// TraceService defines distributed tracing service interface
type TraceService interface {
	// StartSpan starts a new trace span
	StartSpan(ctx context.Context, operationName string) (TraceSpan, context.Context)

	// StartChildSpan starts a child span
	StartChildSpan(ctx context.Context, parent TraceSpan, operationName string) (TraceSpan, context.Context)

	// InjectHeaders injects trace headers into HTTP request
	InjectHeaders(span TraceSpan, headers map[string]string)

	// ExtractHeaders extracts trace context from HTTP headers
	ExtractHeaders(headers map[string]string) (TraceContext, error)

	// IsEnabled returns whether tracing is enabled
	IsEnabled() bool
}

// TraceSpan represents a trace span
type TraceSpan interface {
	// SetTag sets a tag on the span
	SetTag(key string, value interface{})

	// SetError marks the span as having an error
	SetError(err error)

	// LogEvent logs an event on the span
	LogEvent(event string, fields map[string]interface{})

	// Finish finishes the span
	Finish()

	// Context returns the span context
	Context() TraceContext
}

// TraceContext represents trace context
type TraceContext interface {
	// TraceID returns the trace ID
	TraceID() string

	// SpanID returns the span ID
	SpanID() string

	// IsSampled returns whether the trace is sampled
	IsSampled() bool

	// BaggageItem returns a baggage item
	BaggageItem(key string) string

	// SetBaggageItem sets a baggage item
	SetBaggageItem(key, value string)
}

// MiddlewareConfig defines configuration for middleware chain
type MiddlewareConfig struct {
	// Recovery middleware config
	Recovery *RecoveryConfig `json:"recovery,omitempty" yaml:"recovery,omitempty"`

	// Auth middleware config
	Auth *MiddlewareAuthConfig `json:"auth,omitempty" yaml:"auth,omitempty"`

	// Permission middleware config
	Permission *PermissionConfig `json:"permission,omitempty" yaml:"permission,omitempty"`

	// Rate limit middleware config
	RateLimit *RateLimitConfig `json:"rate_limit,omitempty" yaml:"rate_limit,omitempty"`

	// Request ID middleware config
	RequestID *RequestIDConfig `json:"request_id,omitempty" yaml:"request_id,omitempty"`

	// Logging middleware config
	Logging *LoggingConfig `json:"logging,omitempty" yaml:"logging,omitempty"`

	// Metrics middleware config
	Metrics *MetricsConfig `json:"metrics,omitempty" yaml:"metrics,omitempty"`

	// Tracing middleware config
	Tracing *TracingConfig `json:"tracing,omitempty" yaml:"tracing,omitempty"`
}

// RecoveryConfig defines recovery middleware configuration
type RecoveryConfig struct {
	Enabled          bool `json:"enabled" yaml:"enabled"`
	EnableStackTrace bool `json:"enable_stack_trace" yaml:"enable_stack_trace"`
}

// MiddlewareAuthConfig defines authentication middleware configuration
type MiddlewareAuthConfig struct {
	Enabled     bool     `json:"enabled" yaml:"enabled"`
	JWTSecret   string   `json:"jwt_secret" yaml:"jwt_secret"`
	SkipPaths   []string `json:"skip_paths" yaml:"skip_paths"`
	RequireAuth bool     `json:"require_auth" yaml:"require_auth"`
}

// PermissionConfig defines permission middleware configuration
type PermissionConfig struct {
	Enabled   bool     `json:"enabled" yaml:"enabled"`
	SkipPaths []string `json:"skip_paths" yaml:"skip_paths"`
}

// RequestIDConfig defines request ID middleware configuration
type RequestIDConfig struct {
	Enabled    bool   `json:"enabled" yaml:"enabled"`
	HeaderName string `json:"header_name" yaml:"header_name"`
	Generator  string `json:"generator" yaml:"generator"` // uuid, nanoid, custom
}

// LoggingConfig defines logging middleware configuration
type LoggingConfig struct {
	Enabled     bool     `json:"enabled" yaml:"enabled"`
	Format      string   `json:"format" yaml:"format"` // json, text
	SkipPaths   []string `json:"skip_paths" yaml:"skip_paths"`
	LogBody     bool     `json:"log_body" yaml:"log_body"`
	LogHeaders  bool     `json:"log_headers" yaml:"log_headers"`
	MaxBodySize int64    `json:"max_body_size" yaml:"max_body_size"`
}

// MetricsConfig defines metrics middleware configuration
type MetricsConfig struct {
	Enabled   bool     `json:"enabled" yaml:"enabled"`
	Path      string   `json:"path" yaml:"path"`
	SkipPaths []string `json:"skip_paths" yaml:"skip_paths"`
	Namespace string   `json:"namespace" yaml:"namespace"`
	Subsystem string   `json:"subsystem" yaml:"subsystem"`
}

// TracingConfig defines tracing middleware configuration
type TracingConfig struct {
	Enabled       bool     `json:"enabled" yaml:"enabled"`
	ServiceName   string   `json:"service_name" yaml:"service_name"`
	SampleRate    float64  `json:"sample_rate" yaml:"sample_rate"`
	SkipPaths     []string `json:"skip_paths" yaml:"skip_paths"`
	TraceIDHeader string   `json:"trace_id_header" yaml:"trace_id_header"`
}
