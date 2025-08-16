package middlewares

import (
	"proxynd/internal/ports"
)

// BaseMiddleware provides common middleware functionality
// TODO: Migrate from middlewares/ package
type BaseMiddleware struct {
	logger  ports.Logger
	metrics ports.MetricsCollector
}

// NewBaseMiddleware creates a new base middleware
func NewBaseMiddleware(logger ports.Logger, metrics ports.MetricsCollector) *BaseMiddleware {
	return &BaseMiddleware{
		logger:  logger,
		metrics: metrics,
	}
}

// Process implements ports.HTTPMiddleware interface
func (m *BaseMiddleware) Process(next ports.HTTPHandler) ports.HTTPHandler {
	return &MiddlewareHandler{
		next:   next,
		logger: m.logger,
	}
}

// MiddlewareHandler wraps an HTTP handler with middleware
type MiddlewareHandler struct {
	next   ports.HTTPHandler
	logger ports.Logger
}

// Handle processes the request through middleware
func (mh *MiddlewareHandler) Handle(ctx ports.HTTPContext) error {
	// TODO: Implement base middleware processing
	return mh.next.Handle(ctx)
}

// TODO: AuthMiddleware implementation moved to auth_middleware.go

// TODO: AuthMiddleware methods moved to auth_middleware.go

// TODO: HEXAGONAL_MIGRATION - SecurityMiddleware implementation moved to security.go

// TODO: LoggingMiddleware implementation moved to logging.go

// TODO: LoggingMiddlewareHandler implementation moved to logging.go

// TODO: MetricsMiddleware implementation moved to metrics.go

// TODO: RateLimitMiddleware implementation moved to rate_limit_middleware.go

// CORSMiddleware handles CORS headers
// TODO: Add CORS support
type CORSMiddleware struct {
	*BaseMiddleware
}

// NewCORSMiddleware creates a new CORS middleware
func NewCORSMiddleware(base *BaseMiddleware) *CORSMiddleware {
	return &CORSMiddleware{
		BaseMiddleware: base,
	}
}

// Process implements CORS middleware
func (m *CORSMiddleware) Process(next ports.HTTPHandler) ports.HTTPHandler {
	return &CORSMiddlewareHandler{
		next: next,
	}
}

// CORSMiddlewareHandler handles CORS
type CORSMiddlewareHandler struct {
	next ports.HTTPHandler
}

// Handle processes CORS headers
func (h *CORSMiddlewareHandler) Handle(ctx ports.HTTPContext) error {
	// TODO: Implement CORS logic
	return h.next.Handle(ctx)
}

// TODO: HEXAGONAL_MIGRATION - ErrorRecoveryMiddleware implementation moved to error_recovery.go

// NewField creates a new log field
func NewField(key string, value interface{}) ports.Field {
	return &LogField{key: key, value: value}
}

// LogField implements ports.Field
type LogField struct {
	key   string
	value interface{}
}

// Key returns field key
func (lf *LogField) Key() string {
	return lf.key
}

// Value returns field value
func (lf *LogField) Value() interface{} {
	return lf.value
}
