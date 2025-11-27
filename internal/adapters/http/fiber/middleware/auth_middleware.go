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

	// Extract token from request
	token := h.extractToken(ctx)
	if token == "" {
		// No authentication provided
		if h.config.RequireAuth {
			return ctx.Status(401).JSON(map[string]string{
				"error": "Unauthorized: No authentication token provided",
			})
		}
		// Auth is optional, continue without user context
		return h.next.Handle(ctx)
	}

	// Validate token using auth service
	claims, err := h.authService.ValidateToken(ctx.Context(), token)
	if err != nil {
		h.logger.Warn(ctx.Context(), "Token validation failed",
			&AuthField{key: "error", value: err.Error()},
			&AuthField{key: "path", value: ctx.Path()},
		)
		return ctx.Status(401).JSON(map[string]string{
			"error":   "Unauthorized: Invalid or expired token",
			"details": err.Error(),
		})
	}

	// Store user information in context for downstream handlers
	ctx.Locals("user_id", claims.UserID)
	ctx.Locals("user_email", claims.Email)
	ctx.Locals("user_role", claims.Role)
	ctx.Locals("authenticated", true)

	// Log successful authentication
	h.logger.Debug(ctx.Context(), "User authenticated",
		&AuthField{key: "user_id", value: claims.UserID},
		&AuthField{key: "path", value: ctx.Path()},
	)

	return h.next.Handle(ctx)
}

// extractToken extracts authentication token from request
// Supports: Bearer token, API key, Basic auth
func (h *AuthHandler) extractToken(ctx ports.HTTPContext) string {
	// 1. Check Authorization header for Bearer token
	authHeaderRaw := ctx.Get("Authorization")
	if authHeaderRaw != nil {
		authHeader, ok := authHeaderRaw.(string)
		if ok && authHeader != "" {
			// Bearer token format: "Bearer <token>"
			if len(authHeader) > 7 && strings.ToLower(authHeader[:7]) == "bearer " {
				return authHeader[7:]
			}
			// Also support direct token without "Bearer " prefix
			return authHeader
		}
	}

	// 2. Check X-API-Key header
	apiKeyRaw := ctx.Get("X-API-Key")
	if apiKeyRaw != nil {
		apiKey, ok := apiKeyRaw.(string)
		if ok && apiKey != "" {
			return apiKey
		}
	}

	// 3. Check query parameter for API key (less secure, but sometimes needed)
	apiKeyQuery := ctx.Query("api_key")
	if apiKeyQuery != "" {
		return apiKeyQuery
	}

	return ""
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
