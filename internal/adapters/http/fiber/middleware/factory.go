package middlewares

import (
	"context"
	"fmt"
	"sort"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/ports"
	zapAdapter "proxynd/internal/adapters/observability/zap"
	prometheusAdapter "proxynd/internal/adapters/observability/prometheus"
	otelAdapter "proxynd/internal/adapters/observability/opentelemetry"
)

// MiddlewareFactory creates and manages middleware chains
type MiddlewareFactory struct {
	config      *ports.MiddlewareConfig
	logger      ports.Logger
	metrics     ports.MetricsCollector
	tracer      ports.TraceService
	rateLimiter ports.RateLimiter
	authService ports.MiddlewareAuthService
	permService ports.PermissionService
	reqIDGen    ports.RequestIDGenerator
}

// MiddlewareChain implements ports.MiddlewareChain
type MiddlewareChain struct {
	middlewares []ports.EnhancedHTTPMiddleware
}

// NewMiddlewareFactory creates a new middleware factory
func NewMiddlewareFactory(config *ports.MiddlewareConfig) (*MiddlewareFactory, error) {
	factory := &MiddlewareFactory{
		config: config,
	}
	
	// Initialize observability components
	if err := factory.initializeObservability(); err != nil {
		return nil, fmt.Errorf("failed to initialize observability: %w", err)
	}
	
	// Initialize other services
	if err := factory.initializeServices(); err != nil {
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}
	
	return factory, nil
}

// initializeObservability initializes observability components
func (f *MiddlewareFactory) initializeObservability() error {
	// Initialize logger
	if f.config.Logging != nil && f.config.Logging.Enabled {
		loggerConfig := &zapAdapter.LoggerConfig{
			Level:  "info",
			Format: f.config.Logging.Format,
		}
		
		logger, err := zapAdapter.NewLogger(loggerConfig)
		if err != nil {
			return fmt.Errorf("failed to create logger: %w", err)
		}
		f.logger = logger
	}
	
	// Initialize metrics collector
	if f.config.Metrics != nil && f.config.Metrics.Enabled {
		metricsConfig := &prometheusAdapter.MetricsConfig{
			Namespace: f.config.Metrics.Namespace,
			Subsystem: f.config.Metrics.Subsystem,
		}
		
		f.metrics = prometheusAdapter.NewMetricsCollector(metricsConfig)
	}
	
	// Initialize tracer
	if f.config.Tracing != nil && f.config.Tracing.Enabled {
		tracingConfig := &otelAdapter.TracingConfig{
			Enabled:     f.config.Tracing.Enabled,
			ServiceName: f.config.Tracing.ServiceName,
			SampleRate:  f.config.Tracing.SampleRate,
		}
		
		f.tracer = otelAdapter.NewTraceService(tracingConfig)
	}
	
	return nil
}

// initializeServices initializes other services
func (f *MiddlewareFactory) initializeServices() error {
	// Initialize request ID generator
	if f.config.RequestID != nil && f.config.RequestID.Enabled {
		f.reqIDGen = NewUUIDRequestIDGenerator(f.config.RequestID.HeaderName)
	}
	
	// TODO: Initialize other services when available
	// f.authService = ...
	// f.permService = ...
	// f.rateLimiter = ...
	
	return nil
}

// BuildMiddlewareChain builds the complete middleware chain
func (f *MiddlewareFactory) BuildMiddlewareChain() ports.MiddlewareChain {
	chain := &MiddlewareChain{
		middlewares: make([]ports.HTTPMiddleware, 0),
	}
	
	// Add middleware in order based on configuration
	if f.config.Recovery != nil && f.config.Recovery.Enabled {
		chain.Add(f.createRecoveryMiddleware())
	}
	
	if f.config.Auth != nil && f.config.Auth.Enabled && f.authService != nil {
		chain.Add(f.createAuthMiddleware())
	}
	
	if f.config.Permission != nil && f.config.Permission.Enabled && f.permService != nil {
		chain.Add(f.createPermissionMiddleware())
	}
	
	if f.config.RateLimit != nil && f.rateLimiter != nil {
		chain.Add(f.createRateLimitMiddleware())
	}
	
	if f.config.RequestID != nil && f.config.RequestID.Enabled {
		chain.Add(f.createRequestIDMiddleware())
	}
	
	if f.config.Logging != nil && f.config.Logging.Enabled {
		chain.Add(f.createLoggingMiddleware())
	}
	
	if f.config.Metrics != nil && f.config.Metrics.Enabled {
		chain.Add(f.createMetricsMiddleware())
	}
	
	if f.config.Tracing != nil && f.config.Tracing.Enabled {
		chain.Add(f.createTracingMiddleware())
	}
	
	return chain
}

// BuildFiberHandlers builds Fiber-compatible handlers
func (f *MiddlewareFactory) BuildFiberHandlers() []fiber.Handler {
	chain := f.BuildMiddlewareChain()
	middlewares := chain.GetMiddlewares()
	
	handlers := make([]fiber.Handler, len(middlewares))
	for i, middleware := range middlewares {
		handlers[i] = f.adaptToFiberHandler(middleware)
	}
	
	return handlers
}

// adaptToFiberHandler adapts ports.EnhancedHTTPMiddleware to fiber.Handler
func (f *MiddlewareFactory) adaptToFiberHandler(middleware ports.EnhancedHTTPMiddleware) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Create HTTP context adapter
		ctx := NewFiberHTTPContext(c)
		
		// Create final handler that calls c.Next()
		finalHandler := &FiberNextHandler{}
		
		// Process through middleware
		handler := middleware.Process(finalHandler)
		return handler.Handle(ctx)
	}
}

// createRecoveryMiddleware creates recovery middleware
func (f *MiddlewareFactory) createRecoveryMiddleware() ports.EnhancedHTTPMiddleware {
	return NewRecoveryMiddleware(f.config.Recovery, f.logger)
}

// createAuthMiddleware creates authentication middleware
func (f *MiddlewareFactory) createAuthMiddleware() ports.EnhancedHTTPMiddleware {
	return NewAuthMiddleware(f.config.Auth, f.authService, f.logger)
}

// createPermissionMiddleware creates permission middleware
func (f *MiddlewareFactory) createPermissionMiddleware() ports.EnhancedHTTPMiddleware {
	return NewPermissionMiddleware(f.config.Permission, f.permService, f.logger)
}

// createRateLimitMiddleware creates rate limiting middleware
func (f *MiddlewareFactory) createRateLimitMiddleware() ports.EnhancedHTTPMiddleware {
	return NewRateLimitMiddleware(f.config.RateLimit, f.rateLimiter, f.logger)
}

// createRequestIDMiddleware creates request ID middleware
func (f *MiddlewareFactory) createRequestIDMiddleware() ports.EnhancedHTTPMiddleware {
	return NewRequestIDMiddleware(f.config.RequestID, f.reqIDGen, f.logger)
}

// createLoggingMiddleware creates logging middleware
func (f *MiddlewareFactory) createLoggingMiddleware() ports.EnhancedHTTPMiddleware {
	return NewLoggingMiddleware(f.config.Logging, f.logger)
}

// createMetricsMiddleware creates metrics middleware
func (f *MiddlewareFactory) createMetricsMiddleware() ports.EnhancedHTTPMiddleware {
	return NewMetricsMiddleware(f.config.Metrics, f.metrics, f.logger)
}

// createTracingMiddleware creates tracing middleware
func (f *MiddlewareFactory) createTracingMiddleware() ports.EnhancedHTTPMiddleware {
	return NewTracingMiddleware(f.config.Tracing, f.tracer, f.logger)
}

// Add adds middleware to the chain
func (c *MiddlewareChain) Add(middleware ports.EnhancedHTTPMiddleware) {
	c.middlewares = append(c.middlewares, middleware)
	
	// Sort middlewares by order
	sort.Slice(c.middlewares, func(i, j int) bool {
		return c.middlewares[i].Order() < c.middlewares[j].Order()
	})
}

// Build builds the final handler chain
func (c *MiddlewareChain) Build(finalHandler ports.HTTPHandler) ports.HTTPHandler {
	if len(c.middlewares) == 0 {
		return finalHandler
	}
	
	// Build the chain from the end to the beginning
	handler := finalHandler
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		handler = c.middlewares[i].Process(handler)
	}
	
	return handler
}

// GetMiddlewares returns all middleware in execution order
func (c *MiddlewareChain) GetMiddlewares() []ports.EnhancedHTTPMiddleware {
	return c.middlewares
}

// Count returns the number of middleware in the chain
func (c *MiddlewareChain) Count() int {
	return len(c.middlewares)
}

// FiberHTTPContext adapts fiber.Ctx to ports.HTTPContext
type FiberHTTPContext struct {
	ctx *fiber.Ctx
}

// NewFiberHTTPContext creates a new Fiber HTTP context adapter
func NewFiberHTTPContext(c *fiber.Ctx) ports.HTTPContext {
	return &FiberHTTPContext{ctx: c}
}

// Context returns the request context
func (f *FiberHTTPContext) Context() context.Context {
	return f.ctx.Context()
}

// Method returns the HTTP method
func (f *FiberHTTPContext) Method() string {
	return f.ctx.Method()
}

// Path returns the request path
func (f *FiberHTTPContext) Path() string {
	return f.ctx.Path()
}

// Query returns query parameter value
func (f *FiberHTTPContext) Query(key string) string {
	return f.ctx.Query(key)
}

// Header returns header value
func (f *FiberHTTPContext) Header(key string) string {
	return f.ctx.Get(key)
}

// SetHeader sets header value
func (f *FiberHTTPContext) SetHeader(key, value string) {
	f.ctx.Set(key, value)
}

// Body returns request body
func (f *FiberHTTPContext) Body() []byte {
	return f.ctx.Body()
}

// JSON sends JSON response (implements HTTPContext interface)
func (f *FiberHTTPContext) JSON(obj interface{}) error {
	return f.ctx.JSON(obj)
}

// ParseJSON parses JSON body into v
func (f *FiberHTTPContext) ParseJSON(v interface{}) error {
	return f.ctx.BodyParser(v)
}

// SendJSON sends JSON response with status
func (f *FiberHTTPContext) SendJSON(status int, v interface{}) error {
	return f.ctx.Status(status).JSON(v)
}

// SendString sends string response with status
func (f *FiberHTTPContext) SendString(status int, s string) error {
	return f.ctx.Status(status).SendString(s)
}

// Status sets response status
func (f *FiberHTTPContext) Status(code int) ports.HTTPContext {
	f.ctx.Status(code)
	return f
}

// Get gets value from context locals
func (f *FiberHTTPContext) Get(key string) interface{} {
	return f.ctx.Locals(key)
}

// Set sets value in context locals
func (f *FiberHTTPContext) Set(key string, value interface{}) {
	f.ctx.Locals(key, value)
}

// ClientIP returns client IP address
func (f *FiberHTTPContext) ClientIP() string {
	return f.ctx.IP()
}

// UserAgent returns user agent
func (f *FiberHTTPContext) UserAgent() string {
	return f.ctx.Get("User-Agent")
}

// Param returns path parameter value
func (f *FiberHTTPContext) Param(key string) string {
	return f.ctx.Params(key)
}

// Send sends raw data
func (f *FiberHTTPContext) Send(data []byte) error {
	return f.ctx.Send(data)
}

// SendString sends string data
func (f *FiberHTTPContext) SendString(data string) error {
	return f.ctx.SendString(data)
}

// Locals gets/sets local values
func (f *FiberHTTPContext) Locals(key string, value ...interface{}) interface{} {
	if len(value) > 0 {
		f.ctx.Locals(key, value[0])
		return value[0]
	}
	return f.ctx.Locals(key)
}

// FiberNextHandler calls fiber's Next() method
type FiberNextHandler struct{}

// Handle calls the next handler in the chain
func (f *FiberNextHandler) Handle(ctx ports.HTTPContext) error {
	if fiberCtx, ok := ctx.(*FiberHTTPContext); ok {
		return fiberCtx.ctx.Next()
	}
	return fmt.Errorf("invalid context type")
}