package logging

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Context key types to avoid collisions
type contextKey string

const (
	contextKeyRequestID  contextKey = "request_id"
	contextKeyPrincipal  contextKey = "principal" 
	contextKeyManager    contextKey = "manager"
)

// StandardizedMiddlewareConfig defines configuration for standardized logging middleware
type StandardizedMiddlewareConfig struct {
	// Logger to use (can be either zerolog or zap adapter)
	Logger Logger

	// SkipPaths paths to skip logging
	SkipPaths []string

	// SkipStatuses status codes to skip logging
	SkipStatuses []int

	// IncludeBody whether to include request/response body
	IncludeBody bool

	// IncludeHeaders whether to include headers
	IncludeHeaders bool

	// HeadersToLog specific headers to log (empty means all headers)
	HeadersToLog []string

	// CustomFields function to add custom fields
	CustomFields func(*fiber.Ctx) []Field

	// UseStandardizedFields whether to use standardized field names
	UseStandardizedFields bool

	// LogCacheOperations whether to log cache operations
	LogCacheOperations bool

	// LogProxyOperations whether to log proxy operations
	LogProxyOperations bool
}

// DefaultStandardizedMiddlewareConfig provides default configuration for standardized middleware
var DefaultStandardizedMiddlewareConfig = StandardizedMiddlewareConfig{
	SkipPaths: []string{
		"/healthz",
		"/health/live",
		"/metrics",
		"/favicon.ico",
	},
	SkipStatuses:          []int{},
	UseStandardizedFields: true,
	LogCacheOperations:    true,
	LogProxyOperations:    true,
}

// extractRequestContext extracts and sets up request context information
func extractRequestContext(c *fiber.Ctx) (requestID, principal, manager string) {
	// Generate or extract request ID
	requestID = c.Get("X-Request-ID")
	if requestID == "" {
		requestID = c.Get("X-Correlation-ID")
	}
	if requestID == "" {
		requestID = uuid.New().String()
		c.Set("X-Request-ID", requestID)
	}

	// Store request ID in context and locals
	ctx := context.WithValue(c.Context(), contextKeyRequestID, requestID)
	c.SetUserContext(ctx)
	c.Locals(StandardizedFields.RequestID, requestID)

	// Extract user/principal information
	principal = extractPrincipal(c)
	ctx = context.WithValue(ctx, contextKeyPrincipal, principal)
	c.SetUserContext(ctx)
	c.Locals(StandardizedFields.Principal, principal)

	// Extract manager (proxy type)
	manager = extractManager(c)
	ctx = context.WithValue(ctx, contextKeyManager, manager)
	c.SetUserContext(ctx)
	c.Locals(StandardizedFields.Manager, manager)

	return requestID, principal, manager
}

// extractPrincipal extracts principal information from request
func extractPrincipal(c *fiber.Ctx) string {
	if username := c.Locals("username"); username != nil {
		return fmt.Sprintf("%v", username)
	}
	if userID := c.Locals("user_id"); userID != nil {
		return fmt.Sprintf("user:%v", userID)
	}
	if authHeader := c.Get("Authorization"); authHeader != "" {
		return "authenticated"
	}
	return "anonymous"
}

// extractManager extracts manager (proxy type) from request
func extractManager(c *fiber.Ctx) string {
	manager := c.Params("type")
	if manager != "" {
		return manager
	}

	// Try to infer from path
	path := c.Path()
	switch {
	case strings.Contains(path, "/maven"):
		return "maven"
	case strings.Contains(path, "/npm"):
		return "npm"
	case strings.Contains(path, "/apt"):
		return "apt"
	case strings.Contains(path, "/docker"):
		return "docker"
	case strings.Contains(path, "/pip"):
		return "pip"
	case strings.Contains(path, "/yum"):
		return "yum"
	case strings.Contains(path, "/apk"):
		return "apk"
	default:
		return "unknown"
	}
}

// createRequestLogger creates a logger with request context
func createRequestLogger(cfg StandardizedMiddlewareConfig, c *fiber.Ctx, requestID, principal, manager string) Logger {
	var logger Logger
	if cfg.UseStandardizedFields {
		logger = cfg.Logger.WithFields(
			F(StandardizedFields.RequestID, requestID),
			F(StandardizedFields.Principal, principal),
			F(StandardizedFields.Manager, manager),
			F(StandardizedFields.Path, c.Path()),
			F(StandardizedFields.Method, c.Method()),
			F("ip", c.IP()),
			F("user_agent", c.Get("User-Agent")),
		)
	} else {
		// Legacy field names for backward compatibility
		logger = cfg.Logger.WithFields(
			F("request_id", requestID),
			F("method", c.Method()),
			F("path", c.Path()),
			F("ip", c.IP()),
			F("user_agent", c.Get("User-Agent")),
		)
	}

	// Add request headers if enabled
	if cfg.IncludeHeaders {
		headers := extractHeaders(c, cfg.HeadersToLog, true)
		if len(headers) > 0 {
			logger = logger.WithField("request_headers", headers)
		}
	}

	// Add request size
	if c.Request().Header.ContentLength() > 0 {
		logger = logger.WithField(StandardizedFields.BytesIn, c.Request().Header.ContentLength())
	}

	return logger
}

// processResponse processes and logs the response
func processResponse(cfg StandardizedMiddlewareConfig, c *fiber.Ctx, logger Logger, start time.Time, err error) error {
	// Calculate response time
	duration := time.Since(start)
	latencyMs := duration.Milliseconds()

	// Extract cache hit information
	cacheHit := extractCacheHit(c)

	// Create response logger with standardized fields
	respLogger := createResponseLogger(cfg, logger, c, latencyMs, duration, cacheHit)

	// Add additional response information
	respLogger = addResponseMetadata(cfg, respLogger, c, err)

	// Skip logging if status should be skipped
	statusCode := c.Response().StatusCode()
	if shouldSkipStatus(statusCode, cfg.SkipStatuses) {
		return err
	}

	// Log with appropriate level
	logResponse(respLogger, c, statusCode)

	return err
}

// extractCacheHit extracts cache hit information
func extractCacheHit(c *fiber.Ctx) bool {
	if hit := c.Locals("cache_hit"); hit != nil {
		if h, ok := hit.(bool); ok {
			return h
		}
	}
	return false
}

// createResponseLogger creates response logger with standardized fields
func createResponseLogger(
	cfg StandardizedMiddlewareConfig, logger Logger, c *fiber.Ctx,
	latencyMs int64, duration time.Duration, cacheHit bool,
) Logger {
	if cfg.UseStandardizedFields {
		return logger.WithFields(
			F(StandardizedFields.StatusCode, c.Response().StatusCode()),
			F(StandardizedFields.LatencyMs, latencyMs),
			F(StandardizedFields.CacheHit, cacheHit),
			F("duration", duration.String()),
		)
	}
	return logger.WithFields(
		F("status", c.Response().StatusCode()),
		F("duration_ms", latencyMs),
		F("cache_hit", cacheHit),
		F("duration", duration.String()),
	)
}

// addResponseMetadata adds additional response metadata to logger
func addResponseMetadata(cfg StandardizedMiddlewareConfig, respLogger Logger, c *fiber.Ctx, err error) Logger {
	// Add response size
	responseSize := len(c.Response().Body())
	if responseSize > 0 {
		respLogger = respLogger.WithField(StandardizedFields.BytesOut, responseSize)
	}

	// Add upstream information if available
	if upstream := c.Locals("upstream"); upstream != nil {
		respLogger = respLogger.WithField(StandardizedFields.Upstream, upstream)
	}

	// Add package information if available
	if packageName := c.Locals("package_name"); packageName != nil {
		respLogger = respLogger.WithField(StandardizedFields.PackageName, packageName)
	}
	if version := c.Locals("version"); version != nil {
		respLogger = respLogger.WithField(StandardizedFields.Version, version)
	}

	// Add response headers if enabled
	if cfg.IncludeHeaders {
		headers := extractResponseHeaders(c, cfg.HeadersToLog)
		if len(headers) > 0 {
			respLogger = respLogger.WithField("response_headers", headers)
		}
	}

	// Add custom fields
	if cfg.CustomFields != nil {
		fields := cfg.CustomFields(c)
		if len(fields) > 0 {
			respLogger = respLogger.WithFields(fields...)
		}
	}

	// Handle errors
	if err != nil {
		respLogger = respLogger.WithField(StandardizedFields.Error, err.Error())
	}

	return respLogger
}

// logResponse logs the response with appropriate level based on status code
func logResponse(respLogger Logger, c *fiber.Ctx, statusCode int) {
	message := fmt.Sprintf("%s %s", c.Method(), c.Path())

	switch {
	case statusCode >= 500:
		respLogger.Error(message, F("error_type", "server_error"))
	case statusCode >= 400:
		respLogger.Warn(message, F("error_type", "client_error"))
	case statusCode >= 300:
		respLogger.Info(message, F("redirect", true))
	default:
		respLogger.Info(message)
	}
}

// NewStandardized creates a new standardized logging middleware
func NewStandardized(config ...StandardizedMiddlewareConfig) fiber.Handler {
	cfg := DefaultStandardizedMiddlewareConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	if cfg.Logger == nil {
		cfg.Logger = GetLogger()
	}

	return func(c *fiber.Ctx) error {
		// Extract and setup request context
		requestID, principal, manager := extractRequestContext(c)

		// Skip logging if path should be skipped
		if shouldSkipPath(c.Path(), cfg.SkipPaths) {
			return c.Next()
		}

		// Create request logger and log start
		start := time.Now()
		logger := createRequestLogger(cfg, c, requestID, principal, manager)
		logger.Info("Request started")

		// Execute next handler
		err := c.Next()

		// Process and log response
		return processResponse(cfg, c, logger, start, err)
	}
}

// StandardizedRequestLogger creates a request-specific logger with standardized fields
func StandardizedRequestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Generate or extract request ID
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
			c.Set("X-Request-ID", requestID)
		}

		// Create context with standardized fields
		ctx := context.WithValue(c.Context(), contextKeyRequestID, requestID)

		// Extract principal
		principal := "anonymous"
		if username := c.Locals("username"); username != nil {
			principal = fmt.Sprintf("%v", username)
		}
		ctx = context.WithValue(ctx, contextKeyPrincipal, principal)

		// Extract manager
		manager := c.Params("type")
		if manager == "" {
			manager = "unknown"
		}
		ctx = context.WithValue(ctx, contextKeyManager, manager)

		c.SetUserContext(ctx)

		// Create request-specific logger
		logger := GetLogger().WithFields(
			F(StandardizedFields.RequestID, requestID),
			F(StandardizedFields.Principal, principal),
			F(StandardizedFields.Manager, manager),
			F(StandardizedFields.Path, c.Path()),
			F(StandardizedFields.Method, c.Method()),
			F("ip", c.IP()),
		)

		// Store logger in context
		c.Locals("logger", logger)

		return c.Next()
	}
}

// GetStandardizedRequestLogger extracts standardized logger from request context
func GetStandardizedRequestLogger(c *fiber.Ctx) Logger {
	if logger := c.Locals("logger"); logger != nil {
		if l, ok := logger.(Logger); ok {
			return l
		}
	}
	// Return logger with context if available
	return GetLogger().WithContext(c.Context())
}

// StandardizedErrorLogger logs errors with standardized fields
func StandardizedErrorLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()
		if err != nil {
			logger := GetStandardizedRequestLogger(c)

			standardizedFields := []Field{
				F(StandardizedFields.Error, err.Error()),
				F(StandardizedFields.StatusCode, c.Response().StatusCode()),
			}

			// Add fiber error details if available
			if e, ok := err.(*fiber.Error); ok {
				standardizedFields = append(standardizedFields, F("error_code", e.Code))
			}

			logger.Error("Request failed", standardizedFields...)
		}

		return err
	}
}

// StandardizedRecoveryLogger recovers from panics and logs with standardized fields
func StandardizedRecoveryLogger() fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				logger := GetStandardizedRequestLogger(c)

				// Extract error message
				var msg string
				switch v := r.(type) {
				case error:
					msg = v.Error()
				case string:
					msg = v
				default:
					msg = fmt.Sprintf("%v", v)
				}

				// Log panic with standardized fields and stack trace
				logger.Error("Panic recovered",
					F("panic", msg),
					F("stack", string(debug.Stack())),
					F(StandardizedFields.StatusCode, fiber.StatusInternalServerError),
				)

				// Return 500 error with standardized response
				err = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":      "Internal Server Error",
					"request_id": c.Locals(StandardizedFields.RequestID),
				})
			}
		}()

		return c.Next()
	}
}

// LogStandardizedCacheOperation logs cache operations with standardized fields
func LogStandardizedCacheOperation(ctx context.Context, operation string, cacheHit bool, manager, key string) {
	logger := GetLogger().WithContext(ctx)

	fields := []Field{
		F("operation", operation),
		F(StandardizedFields.CacheHit, cacheHit),
		F(StandardizedFields.Manager, manager),
		F("cache_key", key),
	}

	logger.Debug("Cache operation", fields...)
}

// LogStandardizedProxyOperation logs proxy operations with standardized fields
func LogStandardizedProxyOperation(
	ctx context.Context, manager, upstream, packageName, version string,
	bytesIn, bytesOut int64, err error,
) {
	logger := GetLogger().WithContext(ctx)

	fields := []Field{
		F(StandardizedFields.Manager, manager),
		F(StandardizedFields.Upstream, upstream),
		F(StandardizedFields.PackageName, packageName),
		F(StandardizedFields.Version, version),
		F(StandardizedFields.BytesIn, bytesIn),
		F(StandardizedFields.BytesOut, bytesOut),
	}

	if err != nil {
		fields = append(fields, F(StandardizedFields.Error, err.Error()))
		logger.Error("Proxy operation failed", fields...)
	} else {
		logger.Info("Proxy operation completed", fields...)
	}
}

// LogStandardizedHTTPRequest logs HTTP requests with standardized fields
func LogStandardizedHTTPRequest(
	ctx context.Context, method, path string, statusCode int, latencyMs int64, cacheHit bool,
) {
	logger := GetLogger().WithContext(ctx)

	fields := []Field{
		F(StandardizedFields.Method, method),
		F(StandardizedFields.Path, path),
		F(StandardizedFields.StatusCode, statusCode),
		F(StandardizedFields.LatencyMs, latencyMs),
		F(StandardizedFields.CacheHit, cacheHit),
	}

	if statusCode >= 400 {
		logger.Error("HTTP request completed with error", fields...)
	} else {
		logger.Info("HTTP request completed", fields...)
	}
}

// StandardizedFieldsExtractor extracts standardized fields from Fiber context
type StandardizedFieldsExtractor struct{}

// ExtractFromContext extracts standardized fields from context
func (e *StandardizedFieldsExtractor) ExtractFromContext(ctx context.Context) []Field {
	var fields []Field

	if requestID := ctx.Value(StandardizedFields.RequestID); requestID != nil {
		fields = append(fields, F(StandardizedFields.RequestID, requestID))
	}

	if principal := ctx.Value(StandardizedFields.Principal); principal != nil {
		fields = append(fields, F(StandardizedFields.Principal, principal))
	}

	if manager := ctx.Value(StandardizedFields.Manager); manager != nil {
		fields = append(fields, F(StandardizedFields.Manager, manager))
	}

	if traceID := ctx.Value(StandardizedFields.TraceID); traceID != nil {
		fields = append(fields, F(StandardizedFields.TraceID, traceID))
	}

	return fields
}

// ExtractFromFiberContext extracts standardized fields from Fiber context
func (e *StandardizedFieldsExtractor) ExtractFromFiberContext(c *fiber.Ctx) []Field {
	var fields []Field

	if requestID := c.Locals(StandardizedFields.RequestID); requestID != nil {
		fields = append(fields, F(StandardizedFields.RequestID, requestID))
	}

	if principal := c.Locals(StandardizedFields.Principal); principal != nil {
		fields = append(fields, F(StandardizedFields.Principal, principal))
	}

	if manager := c.Locals(StandardizedFields.Manager); manager != nil {
		fields = append(fields, F(StandardizedFields.Manager, manager))
	}

	if cacheHit := c.Locals("cache_hit"); cacheHit != nil {
		fields = append(fields, F(StandardizedFields.CacheHit, cacheHit))
	}

	if upstream := c.Locals("upstream"); upstream != nil {
		fields = append(fields, F(StandardizedFields.Upstream, upstream))
	}

	if packageName := c.Locals("package_name"); packageName != nil {
		fields = append(fields, F(StandardizedFields.PackageName, packageName))
	}

	if version := c.Locals("version"); version != nil {
		fields = append(fields, F(StandardizedFields.Version, version))
	}

	return fields
}

// GlobalStandardizedFieldsExtractor provides a global instance
var GlobalStandardizedFieldsExtractor = &StandardizedFieldsExtractor{}
