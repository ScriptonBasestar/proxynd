package fiber

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"proxynd/internal/adapters/http/fiber/handlers"
	"proxynd/internal/ports"
	"proxynd/internal/usecase"
)

// Server implements HTTP server using Fiber framework
// TODO: Migrate from internal/app/app.go and routers/
type Server struct {
	app           *fiber.App
	proxyService  *usecase.ProxyService
	cacheService  *usecase.CacheStrategyService
	healthService *usecase.HealthService
	logger        ports.Logger
	metrics       ports.MetricsCollector
	config        *ServerConfig
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port           string        `json:"port"`
	ReadTimeout    time.Duration `json:"read_timeout"`
	WriteTimeout   time.Duration `json:"write_timeout"`
	IdleTimeout    time.Duration `json:"idle_timeout"`
	EnableCORS     bool          `json:"enable_cors"`
	EnableCompress bool          `json:"enable_compress"`
	EnableSecurity bool          `json:"enable_security"`
	EnableLimiter  bool          `json:"enable_limiter"`
	TrustedProxies []string      `json:"trusted_proxies"`
}

// NewServer creates a new Fiber HTTP server
func NewServer(
	proxyService *usecase.ProxyService,
	cacheService *usecase.CacheStrategyService,
	healthService *usecase.HealthService,
	logger ports.Logger,
	metrics ports.MetricsCollector,
	config *ServerConfig,
) *Server {
	app := fiber.New(fiber.Config{
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
		// TODO: Add more Fiber configuration
	})

	server := &Server{
		app:           app,
		proxyService:  proxyService,
		cacheService:  cacheService,
		healthService: healthService,
		logger:        logger,
		metrics:       metrics,
		config:        config,
	}

	server.setupMiddleware()
	server.setupRoutes()

	return server
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context, addr string) error {
	// TODO: Implement graceful startup
	if addr == "" {
		addr = fmt.Sprintf(":%s", s.config.Port)
	}

	s.logger.Info(ctx, "Starting HTTP server",
		NewField("address", addr),
		NewField("config", s.config),
	)

	return s.app.Listen(addr)
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	// TODO: Implement graceful shutdown
	s.logger.Info(ctx, "Stopping HTTP server")
	return s.app.Shutdown()
}

// RegisterRoute registers a route with handler
func (s *Server) RegisterRoute(method, path string, handler ports.HTTPHandler) {
	// TODO: Implement route registration with port abstraction
	switch method {
	case "GET":
		s.app.Get(path, s.adaptHandler(handler))
	case "POST":
		s.app.Post(path, s.adaptHandler(handler))
	case "PUT":
		s.app.Put(path, s.adaptHandler(handler))
	case "DELETE":
		s.app.Delete(path, s.adaptHandler(handler))
	case "PATCH":
		s.app.Patch(path, s.adaptHandler(handler))
	}
}

// Use registers middleware
func (s *Server) Use(middleware ports.HTTPMiddleware) {
	// TODO: Implement middleware registration with port abstraction
	s.app.Use(s.adaptMiddleware(middleware))
}

// setupMiddleware configures default middleware
func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.app.Use(recover.New())

	// Logger middleware
	if s.logger != nil {
		s.app.Use(logger.New())
	}

	// Security middleware
	if s.config.EnableSecurity {
		s.app.Use(helmet.New())
	}

	// CORS middleware
	if s.config.EnableCORS {
		s.app.Use(cors.New())
	}

	// Compression middleware
	if s.config.EnableCompress {
		s.app.Use(compress.New())
	}

	// Rate limiting middleware
	if s.config.EnableLimiter {
		s.app.Use(limiter.New(limiter.Config{
			Max:        100,
			Expiration: time.Minute,
		}))
	}

	// Custom metrics middleware
	s.app.Use(s.metricsMiddleware())
}

// setupRoutes configures application routes
// TODO: Migrate from routers/ package
func (s *Server) setupRoutes() {
	// Health routes
	health := s.app.Group("/health")
	health.Get("/", s.handleHealth)
	health.Get("/ready", s.handleReadiness)
	health.Get("/live", s.handleLiveness)

	// API routes
	api := s.app.Group("/api")
	v1 := api.Group("/v1")

	// Proxy routes - TODO: Migrate from routers/proxy_router.go
	proxy := v1.Group("/proxy")
	proxy.All("/*", s.handleProxy)

	// Cache routes
	cache := v1.Group("/cache")
	cache.Get("/stats", s.handleCacheStats)
	cache.Delete("/", s.handleCacheInvalidate)

	// Admin routes
	admin := v1.Group("/admin")
	admin.Get("/metrics", s.handleMetrics)
	admin.Get("/status", s.handleStatus)
}

// handleHealth handles health check requests
func (s *Server) handleHealth(c *fiber.Ctx) error {
	ctx := &FiberContext{ctx: c}

	// Use the health handler from new architecture
	handler := handlers.NewHealthHandler(handlers.NewBaseHandler(
		s.proxyService,
		s.cacheService,
		s.healthService,
		s.logger,
		s.metrics,
	))

	return handler.CheckHealth(ctx)
}

// handleReadiness handles readiness check requests
func (s *Server) handleReadiness(c *fiber.Ctx) error {
	// TODO: Implement readiness check handler
	return c.JSON(fiber.Map{
		"ready":     true,
		"timestamp": time.Now(),
	})
}

// handleLiveness handles liveness check requests
func (s *Server) handleLiveness(c *fiber.Ctx) error {
	// TODO: Implement liveness check handler
	return c.JSON(fiber.Map{
		"alive":     true,
		"timestamp": time.Now(),
	})
}

// handleProxy handles proxy requests
func (s *Server) handleProxy(c *fiber.Ctx) error {
	// TODO: Implement proxy request handling
	return c.Status(501).JSON(fiber.Map{
		"error": "proxy handler not implemented",
	})
}

// handleCacheStats handles cache statistics requests
func (s *Server) handleCacheStats(c *fiber.Ctx) error {
	// TODO: Implement cache stats handler
	return c.JSON(fiber.Map{
		"cache_stats": "not implemented",
	})
}

// handleCacheInvalidate handles cache invalidation requests
func (s *Server) handleCacheInvalidate(c *fiber.Ctx) error {
	// TODO: Implement cache invalidation handler
	return c.JSON(fiber.Map{
		"message": "cache invalidation not implemented",
	})
}

// handleMetrics handles metrics requests
func (s *Server) handleMetrics(c *fiber.Ctx) error {
	// TODO: Implement metrics handler
	return c.JSON(fiber.Map{
		"metrics": "not implemented",
	})
}

// handleStatus handles status requests
func (s *Server) handleStatus(c *fiber.Ctx) error {
	// TODO: Implement status handler
	return c.JSON(fiber.Map{
		"status":  "running",
		"version": "dev",
	})
}

// adaptHandler adapts port handler to Fiber handler
func (s *Server) adaptHandler(handler ports.HTTPHandler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// TODO: Implement handler adaptation
		ctx := &FiberContext{ctx: c}
		return handler.Handle(ctx)
	}
}

// adaptMiddleware adapts port middleware to Fiber middleware
func (s *Server) adaptMiddleware(middleware ports.HTTPMiddleware) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// TODO: Implement middleware adaptation
		return c.Next()
	}
}

// metricsMiddleware creates metrics collection middleware
func (s *Server) metricsMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Process request
		err := c.Next()

		// Record metrics
		if s.metrics != nil {
			duration := time.Since(start)
			labels := map[string]string{
				"method": c.Method(),
				"path":   c.Path(),
				"status": fmt.Sprintf("%d", c.Response().StatusCode()),
			}

			s.metrics.IncCounter("http_requests_total", labels)
			s.metrics.ObserveHistogram("http_request_duration_seconds", duration.Seconds(), labels)
		}

		return err
	}
}

// FiberContext implements ports.HTTPContext for Fiber
type FiberContext struct {
	ctx *fiber.Ctx
}

// Method returns HTTP method
func (fc *FiberContext) Method() string {
	return fc.ctx.Method()
}

// Path returns request path
func (fc *FiberContext) Path() string {
	return fc.ctx.Path()
}

// Query returns query parameter
func (fc *FiberContext) Query(key string) string {
	return fc.ctx.Query(key)
}

// Param returns path parameter
func (fc *FiberContext) Param(key string) string {
	return fc.ctx.Params(key)
}

// Body returns request body
func (fc *FiberContext) Body() []byte {
	return fc.ctx.Body()
}

// Header returns request header
func (fc *FiberContext) Header(key string) string {
	return fc.ctx.Get(key)
}

// SetHeader sets response header
func (fc *FiberContext) SetHeader(key, value string) {
	fc.ctx.Set(key, value)
}

// ClientIP returns client IP address
func (fc *FiberContext) ClientIP() string {
	return fc.ctx.IP()
}

// UserAgent returns user agent header
func (fc *FiberContext) UserAgent() string {
	return fc.ctx.Get("User-Agent")
}

// Status sets response status code
func (fc *FiberContext) Status(code int) ports.HTTPContext {
	fc.ctx.Status(code)
	return fc
}

// JSON sends JSON response
func (fc *FiberContext) JSON(obj interface{}) error {
	return fc.ctx.JSON(obj)
}

// SendJSON sends JSON response with status
func (fc *FiberContext) SendJSON(status int, obj interface{}) error {
	return fc.ctx.Status(status).JSON(obj)
}

// Send sends byte response
func (fc *FiberContext) Send(data []byte) error {
	return fc.ctx.Send(data)
}

// SendString sends string response
func (fc *FiberContext) SendString(data string) error {
	return fc.ctx.SendString(data)
}

// Context returns request context
func (fc *FiberContext) Context() context.Context {
	return fc.ctx.Context()
}

// Locals gets/sets local values
func (fc *FiberContext) Locals(key string, value ...interface{}) interface{} {
	if len(value) > 0 {
		fc.ctx.Locals(key, value[0])
		return value[0]
	}
	return fc.ctx.Locals(key)
}

// Get gets value from context locals
func (fc *FiberContext) Get(key string) interface{} {
	return fc.ctx.Locals(key)
}

// Set sets value in context locals
func (fc *FiberContext) Set(key string, value interface{}) {
	fc.ctx.Locals(key, value)
}

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
