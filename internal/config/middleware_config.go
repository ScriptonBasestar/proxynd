package config

// MiddlewareSettings represents middleware configuration in the unified config
type MiddlewareSettings struct {
	// Recovery middleware
	Recovery *RecoverySettings `json:"recovery,omitempty" yaml:"recovery,omitempty"`

	// Authentication middleware
	Auth *AuthSettings `json:"auth,omitempty" yaml:"auth,omitempty"`

	// Authorization middleware
	Permission *PermissionSettings `json:"permission,omitempty" yaml:"permission,omitempty"`

	// Rate limiting middleware
	RateLimit *RateLimitSettings `json:"rate_limit,omitempty" yaml:"rate_limit,omitempty"`

	// Request ID middleware
	RequestID *RequestIDSettings `json:"request_id,omitempty" yaml:"request_id,omitempty"`

	// Logging middleware
	Logging *MiddlewareLoggingSettings `json:"logging,omitempty" yaml:"logging,omitempty"`

	// Metrics middleware
	Metrics *MetricsSettings `json:"metrics,omitempty" yaml:"metrics,omitempty"`

	// Tracing middleware
	Tracing *TracingSettings `json:"tracing,omitempty" yaml:"tracing,omitempty"`
}

// RecoverySettings configures panic recovery middleware
type RecoverySettings struct {
	Enabled          bool `json:"enabled" yaml:"enabled" default:"true"`
	EnableStackTrace bool `json:"enable_stack_trace" yaml:"enable_stack_trace" default:"true"`
}

// AuthSettings configures authentication middleware
type AuthSettings struct {
	Enabled     bool     `json:"enabled" yaml:"enabled" default:"false"`
	JWTSecret   string   `json:"jwt_secret" yaml:"jwt_secret"`
	SkipPaths   []string `json:"skip_paths" yaml:"skip_paths"`
	RequireAuth bool     `json:"require_auth" yaml:"require_auth" default:"false"`
}

// PermissionSettings configures authorization middleware
type PermissionSettings struct {
	Enabled   bool     `json:"enabled" yaml:"enabled" default:"false"`
	SkipPaths []string `json:"skip_paths" yaml:"skip_paths"`
}

// RateLimitSettings configures rate limiting middleware
type RateLimitSettings struct {
	RequestsPerSecond float64  `json:"requests_per_second" yaml:"requests_per_second" default:"100"`
	BurstSize         int      `json:"burst_size" yaml:"burst_size" default:"20"`
	SkipSuccessful    bool     `json:"skip_successful" yaml:"skip_successful" default:"false"`
	TrustedProxies    []string `json:"trusted_proxies" yaml:"trusted_proxies"`
}

// RequestIDSettings configures request ID middleware
type RequestIDSettings struct {
	Enabled    bool   `json:"enabled" yaml:"enabled" default:"true"`
	HeaderName string `json:"header_name" yaml:"header_name" default:"X-Request-ID"`
	Generator  string `json:"generator" yaml:"generator" default:"uuid"` // uuid, nanoid, hex
}

// MiddlewareLoggingSettings configures request logging middleware
type MiddlewareLoggingSettings struct {
	Enabled     bool     `json:"enabled" yaml:"enabled" default:"true"`
	Format      string   `json:"format" yaml:"format" default:"json"` // json, text
	SkipPaths   []string `json:"skip_paths" yaml:"skip_paths"`
	LogBody     bool     `json:"log_body" yaml:"log_body" default:"false"`
	LogHeaders  bool     `json:"log_headers" yaml:"log_headers" default:"false"`
	MaxBodySize int64    `json:"max_body_size" yaml:"max_body_size" default:"1024"`
}

// MetricsSettings configures metrics collection middleware
type MetricsSettings struct {
	Enabled   bool     `json:"enabled" yaml:"enabled" default:"true"`
	Path      string   `json:"path" yaml:"path" default:"/metrics"`
	SkipPaths []string `json:"skip_paths" yaml:"skip_paths"`
	Namespace string   `json:"namespace" yaml:"namespace" default:"proxynd"`
	Subsystem string   `json:"subsystem" yaml:"subsystem" default:"http"`
}

// TracingSettings configures distributed tracing middleware
type TracingSettings struct {
	Enabled       bool     `json:"enabled" yaml:"enabled" default:"false"`
	ServiceName   string   `json:"service_name" yaml:"service_name" default:"proxynd"`
	SampleRate    float64  `json:"sample_rate" yaml:"sample_rate" default:"0.1"`
	SkipPaths     []string `json:"skip_paths" yaml:"skip_paths"`
	TraceIDHeader string   `json:"trace_id_header" yaml:"trace_id_header" default:"X-Trace-ID"`
}

// NOTE: HEXAGONAL_MIGRATION - ToPortsConfig() and related conversion functions removed
// These functions created an import cycle (config → ports → config).
// Conversion logic should be in the adapter layer, not in the config package.
// If you need to convert MiddlewareSettings to ports.MiddlewareConfig, implement the
// conversion in internal/adapters/config/ or internal/adapters/http/fiber/middleware/
