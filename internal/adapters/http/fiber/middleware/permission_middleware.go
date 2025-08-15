package middlewares

import (
	"strings"

	"proxynd/internal/ports"
)

// PermissionMiddleware implements ports.EnhancedHTTPMiddleware for authorization
type PermissionMiddleware struct {
	config      *ports.PermissionConfig
	permService ports.PermissionService
	logger      ports.Logger
}

// NewPermissionMiddleware creates a new permission middleware
func NewPermissionMiddleware(
	config *ports.PermissionConfig,
	permService ports.PermissionService,
	logger ports.Logger,
) ports.EnhancedHTTPMiddleware {
	if config == nil {
		config = &ports.PermissionConfig{
			Enabled:   false, // Disabled by default
			SkipPaths: []string{"/health", "/metrics", "/api/auth/login"},
		}
	}
	
	return &PermissionMiddleware{
		config:      config,
		permService: permService,
		logger:      logger,
	}
}

// Process implements ports.EnhancedHTTPMiddleware
func (m *PermissionMiddleware) Process(next ports.HTTPHandler) ports.HTTPHandler {
	return &PermissionHandler{
		next:        next,
		config:      m.config,
		permService: m.permService,
		logger:      m.logger,
	}
}

// Order returns the middleware execution order
func (m *PermissionMiddleware) Order() ports.MiddlewareOrder {
	return ports.OrderPermission
}

// Name returns the middleware name
func (m *PermissionMiddleware) Name() string {
	return "permission"
}

// IsEnabled returns whether the middleware is enabled
func (m *PermissionMiddleware) IsEnabled() bool {
	return m.config.Enabled && m.permService != nil && m.permService.IsEnabled()
}

// PermissionHandler handles authorization
type PermissionHandler struct {
	next        ports.HTTPHandler
	config      *ports.PermissionConfig
	permService ports.PermissionService
	logger      ports.Logger
}

// Handle processes authorization
func (h *PermissionHandler) Handle(ctx ports.HTTPContext) error {
	// Skip authorization for configured paths
	if h.shouldSkipPath(ctx.Path()) {
		return h.next.Handle(ctx)
	}
	
	// Skip if authorization is not enabled
	if !h.config.Enabled || h.permService == nil || !h.permService.IsEnabled() {
		return h.next.Handle(ctx)
	}
	
	// TODO: Implement actual authorization logic
	// For now, just pass through
	return h.next.Handle(ctx)
}

// shouldSkipPath checks if the path should be skipped for authorization
func (h *PermissionHandler) shouldSkipPath(path string) bool {
	for _, skipPath := range h.config.SkipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}

// PermissionField implements ports.Field for permission logging
type PermissionField struct {
	key   string
	value interface{}
}

// Key returns the field key
func (f *PermissionField) Key() string {
	return f.key
}

// Value returns the field value
func (f *PermissionField) Value() interface{} {
	return f.value
}