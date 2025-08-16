package middlewares

import (
	"proxynd/internal/ports"
)

// RateLimitMiddleware implements ports.EnhancedHTTPMiddleware for rate limiting
type RateLimitMiddleware struct {
	config      *ports.RateLimitConfig
	rateLimiter ports.RateLimiter
	logger      ports.Logger
}

// NewRateLimitMiddleware creates a new rate limiting middleware
func NewRateLimitMiddleware(
	config *ports.RateLimitConfig,
	rateLimiter ports.RateLimiter,
	logger ports.Logger,
) ports.EnhancedHTTPMiddleware {
	if config == nil {
		config = &ports.RateLimitConfig{
			RequestsPerSecond: 100,
			BurstSize:         20,
			SkipSuccessful:    false,
			TrustedProxies:    []string{"127.0.0.1", "::1"},
		}
	}

	return &RateLimitMiddleware{
		config:      config,
		rateLimiter: rateLimiter,
		logger:      logger,
	}
}

// Process implements ports.EnhancedHTTPMiddleware
func (m *RateLimitMiddleware) Process(next ports.HTTPHandler) ports.HTTPHandler {
	return &RateLimitHandler{
		next:        next,
		config:      m.config,
		rateLimiter: m.rateLimiter,
		logger:      m.logger,
	}
}

// Order returns the middleware execution order
func (m *RateLimitMiddleware) Order() ports.MiddlewareOrder {
	return ports.OrderRateLimit
}

// Name returns the middleware name
func (m *RateLimitMiddleware) Name() string {
	return "rate_limit"
}

// IsEnabled returns whether the middleware is enabled
func (m *RateLimitMiddleware) IsEnabled() bool {
	return m.rateLimiter != nil
}

// RateLimitHandler handles rate limiting
type RateLimitHandler struct {
	next        ports.HTTPHandler
	config      *ports.RateLimitConfig
	rateLimiter ports.RateLimiter
	logger      ports.Logger
}

// Handle processes rate limiting
func (h *RateLimitHandler) Handle(ctx ports.HTTPContext) error {
	// Skip if rate limiter is not available
	if h.rateLimiter == nil {
		return h.next.Handle(ctx)
	}

	// TODO: Implement actual rate limiting logic
	// For now, just pass through
	return h.next.Handle(ctx)
}

// RateLimitField implements ports.Field for rate limit logging
type RateLimitField struct {
	key   string
	value interface{}
}

// Key returns the field key
func (f *RateLimitField) Key() string {
	return f.key
}

// Value returns the field value
func (f *RateLimitField) Value() interface{} {
	return f.value
}
