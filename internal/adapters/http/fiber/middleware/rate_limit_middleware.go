package middlewares

import (
	"fmt"
	"net"
	"strings"
	"time"

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

	// Generate rate limit key
	key := h.generateKey(ctx)

	// Check rate limit
	goCtx := ctx.Context()
	allowed, err := h.rateLimiter.Allow(goCtx, key)
	if err != nil {
		// Log error but allow request to proceed (fail-open)
		if h.logger != nil {
			h.logger.Error(goCtx, "Rate limiter error",
				&RateLimitField{"error", err.Error()},
				&RateLimitField{"key", key})
		}
		return h.next.Handle(ctx)
	}

	// Get config for headers
	cfg := h.rateLimiter.GetConfig()
	limit := int(cfg.RequestsPerSecond)
	resetTime := time.Now().Add(time.Second).Unix()

	if !allowed {
		// Rate limit exceeded
		if h.logger != nil {
			h.logger.Warn(goCtx, "Rate limit exceeded",
				&RateLimitField{"key", key},
				&RateLimitField{"path", ctx.Path()},
				&RateLimitField{"method", ctx.Method()},
				&RateLimitField{"client_ip", h.getClientIP(ctx)})
		}

		// Set rate limit headers
		ctx.SetHeader("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		ctx.SetHeader("X-RateLimit-Remaining", "0")
		ctx.SetHeader("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime))
		ctx.SetHeader("Retry-After", "1")

		// Return 429 Too Many Requests
		ctx.Status(429)
		return ctx.JSON(map[string]interface{}{
			"error":       "Rate limit exceeded",
			"message":     "Too many requests. Please try again later.",
			"retry_after": 1,
			"limit":       limit,
		})
	}

	// Set rate limit headers for successful requests
	// Note: We can't easily get remaining count from token bucket without extra work
	// So we set a reasonable estimate
	ctx.SetHeader("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
	ctx.SetHeader("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime))

	// Continue to next handler
	return h.next.Handle(ctx)
}

// generateKey generates a rate limit key based on configuration
func (h *RateLimitHandler) generateKey(ctx ports.HTTPContext) string {
	// Use custom key generator if provided
	if h.config != nil && h.config.KeyGenerator != nil {
		return h.config.KeyGenerator(ctx)
	}

	// Default: use client IP as key
	return h.getClientIP(ctx)
}

// getClientIP extracts the real client IP considering trusted proxies
func (h *RateLimitHandler) getClientIP(ctx ports.HTTPContext) string {
	clientIP := ctx.ClientIP()

	// Check if current connection is from a trusted proxy
	if h.config != nil && len(h.config.TrustedProxies) > 0 {
		if h.isIPInList(clientIP, h.config.TrustedProxies) {
			// Try X-Forwarded-For header
			if forwarded := ctx.Header("X-Forwarded-For"); forwarded != "" {
				// Get the first IP (original client)
				ips := strings.Split(forwarded, ",")
				if len(ips) > 0 {
					return strings.TrimSpace(ips[0])
				}
			}

			// Try X-Real-IP header
			if realIP := ctx.Header("X-Real-IP"); realIP != "" {
				return realIP
			}
		}
	}

	return clientIP
}

// isIPInList checks if an IP is in a list (supports CIDR notation)
func (h *RateLimitHandler) isIPInList(ip string, list []string) bool {
	clientIP := net.ParseIP(ip)
	if clientIP == nil {
		return false
	}

	for _, entry := range list {
		// Check for CIDR notation
		if strings.Contains(entry, "/") {
			_, ipNet, err := net.ParseCIDR(entry)
			if err == nil && ipNet.Contains(clientIP) {
				return true
			}
		} else {
			// Direct IP comparison
			if entry == ip {
				return true
			}
		}
	}

	return false
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
