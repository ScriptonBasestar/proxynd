package config

import (
	"fmt"

	"proxynd/internal/ports"
)

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

// ToPortsConfig converts MiddlewareSettings to ports.MiddlewareConfig
func (m *MiddlewareSettings) ToPortsConfig() *ports.MiddlewareConfig {
	if m == nil {
		return DefaultMiddlewareConfig()
	}

	config := &ports.MiddlewareConfig{}

	// Recovery
	if m.Recovery != nil {
		config.Recovery = &ports.RecoveryConfig{
			Enabled:          m.Recovery.Enabled,
			EnableStackTrace: m.Recovery.EnableStackTrace,
		}
	}

	// Auth
	if m.Auth != nil {
		config.Auth = &ports.MiddlewareAuthConfig{
			Enabled:     m.Auth.Enabled,
			JWTSecret:   m.Auth.JWTSecret,
			SkipPaths:   m.Auth.SkipPaths,
			RequireAuth: m.Auth.RequireAuth,
		}
	}

	// Permission
	if m.Permission != nil {
		config.Permission = &ports.PermissionConfig{
			Enabled:   m.Permission.Enabled,
			SkipPaths: m.Permission.SkipPaths,
		}
	}

	// Rate Limit
	if m.RateLimit != nil {
		config.RateLimit = &ports.RateLimitConfig{
			RequestsPerSecond: m.RateLimit.RequestsPerSecond,
			BurstSize:         m.RateLimit.BurstSize,
			SkipSuccessful:    m.RateLimit.SkipSuccessful,
			TrustedProxies:    m.RateLimit.TrustedProxies,
		}
	}

	// Request ID
	if m.RequestID != nil {
		config.RequestID = &ports.RequestIDConfig{
			Enabled:    m.RequestID.Enabled,
			HeaderName: m.RequestID.HeaderName,
			Generator:  m.RequestID.Generator,
		}
	}

	// Logging
	if m.Logging != nil {
		config.Logging = &ports.LoggingConfig{
			Enabled:     m.Logging.Enabled,
			Format:      m.Logging.Format,
			SkipPaths:   m.Logging.SkipPaths,
			LogBody:     m.Logging.LogBody,
			LogHeaders:  m.Logging.LogHeaders,
			MaxBodySize: m.Logging.MaxBodySize,
		}
	}

	// Metrics
	if m.Metrics != nil {
		config.Metrics = &ports.MetricsConfig{
			Enabled:   m.Metrics.Enabled,
			Path:      m.Metrics.Path,
			SkipPaths: m.Metrics.SkipPaths,
			Namespace: m.Metrics.Namespace,
			Subsystem: m.Metrics.Subsystem,
		}
	}

	// Tracing
	if m.Tracing != nil {
		config.Tracing = &ports.TracingConfig{
			Enabled:       m.Tracing.Enabled,
			ServiceName:   m.Tracing.ServiceName,
			SampleRate:    m.Tracing.SampleRate,
			SkipPaths:     m.Tracing.SkipPaths,
			TraceIDHeader: m.Tracing.TraceIDHeader,
		}
	}

	return config
}

// DefaultMiddlewareConfig returns a default middleware configuration
func DefaultMiddlewareConfig() *ports.MiddlewareConfig {
	return &ports.MiddlewareConfig{
		Recovery: &ports.RecoveryConfig{
			Enabled:          true,
			EnableStackTrace: true,
		},
		Auth: &ports.MiddlewareAuthConfig{
			Enabled:     false,
			SkipPaths:   []string{"/health", "/metrics", "/api/auth/login"},
			RequireAuth: false,
		},
		Permission: &ports.PermissionConfig{
			Enabled:   false,
			SkipPaths: []string{"/health", "/metrics", "/api/auth/login"},
		},
		RateLimit: &ports.RateLimitConfig{
			RequestsPerSecond: 100,
			BurstSize:         20,
			SkipSuccessful:    false,
			TrustedProxies:    []string{"127.0.0.1", "::1"},
		},
		RequestID: &ports.RequestIDConfig{
			Enabled:    true,
			HeaderName: "X-Request-ID",
			Generator:  "uuid",
		},
		Logging: &ports.LoggingConfig{
			Enabled:     true,
			Format:      "json",
			SkipPaths:   []string{"/health", "/metrics"},
			LogBody:     false,
			LogHeaders:  false,
			MaxBodySize: 1024,
		},
		Metrics: &ports.MetricsConfig{
			Enabled:   true,
			Path:      "/metrics",
			SkipPaths: []string{"/health"},
			Namespace: "proxynd",
			Subsystem: "http",
		},
		Tracing: &ports.TracingConfig{
			Enabled:       false,
			ServiceName:   "proxynd",
			SampleRate:    0.1,
			SkipPaths:     []string{"/health", "/metrics"},
			TraceIDHeader: "X-Trace-ID",
		},
	}
}

// DevelopmentMiddlewareConfig returns a development-friendly middleware configuration
func DevelopmentMiddlewareConfig() *ports.MiddlewareConfig {
	config := DefaultMiddlewareConfig()

	// Enable more verbose logging in development
	config.Logging.LogBody = true
	config.Logging.LogHeaders = true
	config.Logging.MaxBodySize = 10240 // 10KB

	// Enable tracing with full sampling in development
	config.Tracing.Enabled = true
	config.Tracing.SampleRate = 1.0

	// More relaxed rate limiting in development
	config.RateLimit.RequestsPerSecond = 1000
	config.RateLimit.BurstSize = 100

	return config
}

// ProductionMiddlewareConfig returns a production-ready middleware configuration
func ProductionMiddlewareConfig() *ports.MiddlewareConfig {
	config := DefaultMiddlewareConfig()

	// Disable stack traces in production
	config.Recovery.EnableStackTrace = false

	// More restrictive logging in production
	config.Logging.LogBody = false
	config.Logging.LogHeaders = false

	// Conservative tracing sampling in production
	config.Tracing.Enabled = true
	config.Tracing.SampleRate = 0.01 // 1% sampling

	// Tighter rate limiting in production
	config.RateLimit.RequestsPerSecond = 50
	config.RateLimit.BurstSize = 10

	return config
}

// ValidateMiddlewareConfig validates middleware configuration
func ValidateMiddlewareConfig(config *ports.MiddlewareConfig) error {
	if config == nil {
		return nil
	}

	// Validate rate limit settings
	if config.RateLimit != nil {
		if config.RateLimit.RequestsPerSecond <= 0 {
			return fmt.Errorf("rate_limit.requests_per_second must be positive")
		}
		if config.RateLimit.BurstSize <= 0 {
			return fmt.Errorf("rate_limit.burst_size must be positive")
		}
	}

	// Validate tracing settings
	if config.Tracing != nil {
		if config.Tracing.SampleRate < 0 || config.Tracing.SampleRate > 1 {
			return fmt.Errorf("tracing.sample_rate must be between 0 and 1")
		}
	}

	// Validate logging settings
	if config.Logging != nil {
		if config.Logging.MaxBodySize < 0 {
			return fmt.Errorf("logging.max_body_size must be non-negative")
		}
	}

	return nil
}
