package middlewares

import (
	"strings"

	"proxynd/internal/ports"
)

// AuthMiddleware implements ports.EnhancedHTTPMiddleware for authentication
type AuthMiddleware struct {
	config      *ports.MiddlewareAuthConfig
	authService ports.MiddlewareAuthService
	logger      ports.Logger
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(
	config *ports.MiddlewareAuthConfig,
	authService ports.MiddlewareAuthService,
	logger ports.Logger,
) ports.EnhancedHTTPMiddleware {
	if config == nil {
		config = &ports.MiddlewareAuthConfig{
			Enabled:     false, // Disabled by default
			SkipPaths:   []string{"/health", "/metrics", "/api/auth/login"},
			RequireAuth: false,
		}
	}

	return &AuthMiddleware{
		config:      config,
		authService: authService,
		logger:      logger,
	}
}

// Process implements ports.EnhancedHTTPMiddleware
func (m *AuthMiddleware) Process(next ports.HTTPHandler) ports.HTTPHandler {
	return &AuthHandler{
		next:        next,
		config:      m.config,
		authService: m.authService,
		logger:      m.logger,
	}
}

// Order returns the middleware execution order
func (m *AuthMiddleware) Order() ports.MiddlewareOrder {
	return ports.OrderAuth
}

// Name returns the middleware name
func (m *AuthMiddleware) Name() string {
	return "auth"
}

// IsEnabled returns whether the middleware is enabled
func (m *AuthMiddleware) IsEnabled() bool {
	return m.config.Enabled && m.authService != nil && m.authService.IsEnabled()
}

// AuthHandler handles authentication
type AuthHandler struct {
	next        ports.HTTPHandler
	config      *ports.MiddlewareAuthConfig
	authService ports.MiddlewareAuthService
	logger      ports.Logger
}

// Handle processes authentication
func (h *AuthHandler) Handle(ctx ports.HTTPContext) error {
	// Skip authentication for configured paths
	if h.shouldSkipPath(ctx.Path()) {
		return h.next.Handle(ctx)
	}

	// Skip if authentication is not enabled
	if !h.config.Enabled || h.authService == nil || !h.authService.IsEnabled() {
		return h.next.Handle(ctx)
	}

	// TODO: Implement actual authentication logic
	// For now, just pass through
	return h.next.Handle(ctx)
}

// shouldSkipPath checks if the path should be skipped for authentication
func (h *AuthHandler) shouldSkipPath(path string) bool {
	for _, skipPath := range h.config.SkipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}

// AuthField implements ports.Field for auth logging
type AuthField struct {
	key   string
	value interface{}
}

// Key returns the field key
func (f *AuthField) Key() string {
	return f.key
}

// Value returns the field value
func (f *AuthField) Value() interface{} {
	return f.value
}
