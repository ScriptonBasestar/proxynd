package middlewares

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"proxynd/internal/ports"
)

// RecoveryMiddleware implements ports.EnhancedHTTPMiddleware for panic recovery
type RecoveryMiddleware struct {
	config *ports.RecoveryConfig
	logger ports.Logger
}

// NewRecoveryMiddleware creates a new recovery middleware
func NewRecoveryMiddleware(
	config *ports.RecoveryConfig,
	logger ports.Logger,
) ports.EnhancedHTTPMiddleware {
	if config == nil {
		config = &ports.RecoveryConfig{
			Enabled:          true,
			EnableStackTrace: true,
		}
	}
	
	return &RecoveryMiddleware{
		config: config,
		logger: logger,
	}
}

// Process implements ports.EnhancedHTTPMiddleware
func (m *RecoveryMiddleware) Process(next ports.HTTPHandler) ports.HTTPHandler {
	return &RecoveryHandler{
		next:   next,
		config: m.config,
		logger: m.logger,
	}
}

// Order returns the middleware execution order
func (m *RecoveryMiddleware) Order() ports.MiddlewareOrder {
	return ports.OrderRecovery
}

// Name returns the middleware name
func (m *RecoveryMiddleware) Name() string {
	return "recovery"
}

// IsEnabled returns whether the middleware is enabled
func (m *RecoveryMiddleware) IsEnabled() bool {
	return m.config.Enabled
}

// RecoveryHandler handles panic recovery
type RecoveryHandler struct {
	next   ports.HTTPHandler
	config *ports.RecoveryConfig
	logger ports.Logger
}

// Handle processes the request with panic recovery
func (h *RecoveryHandler) Handle(ctx ports.HTTPContext) error {
	if !h.config.Enabled {
		return h.next.Handle(ctx)
	}
	
	defer func() {
		if r := recover(); r != nil {
			h.handlePanic(ctx, r)
		}
	}()
	
	return h.next.Handle(ctx)
}

// handlePanic handles recovered panics
func (h *RecoveryHandler) handlePanic(ctx ports.HTTPContext, recovered interface{}) {
	// Create error message
	errMsg := fmt.Sprintf("Panic recovered: %v", recovered)
	
	// Get stack trace if enabled
	var stackTrace string
	if h.config.EnableStackTrace {
		stackTrace = string(debug.Stack())
	}
	
	// Log the panic
	if h.logger != nil {
		fields := []ports.Field{
			&RecoveryField{key: "panic_value", value: recovered},
			&RecoveryField{key: "method", value: ctx.Method()},
			&RecoveryField{key: "path", value: ctx.Path()},
			&RecoveryField{key: "client_ip", value: ctx.ClientIP()},
			&RecoveryField{key: "user_agent", value: ctx.UserAgent()},
		}
		
		if h.config.EnableStackTrace && stackTrace != "" {
			fields = append(fields, &RecoveryField{key: "stack_trace", value: stackTrace})
		}
		
		// Add request ID if available
		if requestID := ctx.Get("request_id"); requestID != nil {
			fields = append(fields, &RecoveryField{key: "request_id", value: requestID})
		}
		
		h.logger.Error(ctx.Context(), "Panic recovered in HTTP handler", fields...)
	}
	
	// Send error response
	errorResponse := map[string]interface{}{
		"error":   "Internal Server Error",
		"message": "An unexpected error occurred",
	}
	
	// Add debug info in development
	if h.config.EnableStackTrace {
		errorResponse["debug"] = map[string]interface{}{
			"panic": recovered,
			"stack": stackTrace,
		}
	}
	
	// Set status and send JSON response
	if err := ctx.SendJSON(500, errorResponse); err != nil {
		// If JSON response fails, try string response
		ctx.SendString(500, "Internal Server Error")
	}
}

// RecoveryField implements ports.Field for recovery logging
type RecoveryField struct {
	key   string
	value interface{}
}

// Key returns the field key
func (f *RecoveryField) Key() string {
	return f.key
}

// Value returns the field value
func (f *RecoveryField) Value() interface{} {
	return f.value
}

// GetGoroutineID returns the current goroutine ID
func GetGoroutineID() uint64 {
	buf := make([]byte, 64)
	n := runtime.Stack(buf, false)
	return parseGoroutineID(buf[:n])
}

// parseGoroutineID parses goroutine ID from stack trace
func parseGoroutineID(stack []byte) uint64 {
	// Simple parsing of "goroutine N [running]:"
	// This is a simplified implementation
	for i, b := range stack {
		if b == ' ' {
			// Found space, look for number
			var id uint64
			for j := i + 1; j < len(stack) && stack[j] >= '0' && stack[j] <= '9'; j++ {
				id = id*10 + uint64(stack[j]-'0')
			}
			return id
		}
	}
	return 0
}

// RecoveryStats tracks recovery statistics
type RecoveryStats struct {
	TotalPanics   uint64 `json:"total_panics"`
	RecentPanics  uint64 `json:"recent_panics"`
	LastPanicTime string `json:"last_panic_time"`
}

// GetRecoveryStats returns recovery statistics
func GetRecoveryStats() *RecoveryStats {
	// This would be implemented with actual statistics collection
	return &RecoveryStats{
		TotalPanics:   0,
		RecentPanics:  0,
		LastPanicTime: "",
	}
}