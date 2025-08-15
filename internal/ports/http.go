package ports

import (
	"context"
)

// HTTPServer defines the HTTP server port interface
// TODO: Migrate from fiber.App in internal/app/app.go
type HTTPServer interface {
	// Start starts the HTTP server on given address
	Start(ctx context.Context, addr string) error
	
	// Stop gracefully stops the HTTP server
	Stop(ctx context.Context) error
	
	// RegisterRoute registers a route with handler
	RegisterRoute(method, path string, handler HTTPHandler)
	
	// Use registers middleware
	Use(middleware HTTPMiddleware)
}

// HTTPHandler defines the handler interface for HTTP requests
// TODO: Replace current handler pattern in handlers/
type HTTPHandler interface {
	// Handle processes the HTTP request
	Handle(ctx HTTPContext) error
}

// HTTPMiddleware defines middleware interface
// TODO: Migrate from middlewares/
type HTTPMiddleware interface {
	// Process processes the request through middleware
	Process(next HTTPHandler) HTTPHandler
}

// HTTPContext abstracts HTTP request/response context
// TODO: Replace direct fiber.Ctx usage
type HTTPContext interface {
	// Request operations
	Method() string
	Path() string
	Query(key string) string
	Param(key string) string
	Body() []byte
	Header(key string) string
	
	// Response operations
	Status(code int) HTTPContext
	JSON(obj interface{}) error
	Send(data []byte) error
	SendString(data string) error
	
	// Context operations
	Context() context.Context
	Locals(key string, value ...interface{}) interface{}
}

// HTTPRouter defines routing interface
// TODO: Abstract from fiber routing
type HTTPRouter interface {
	// Group creates a route group
	Group(prefix string) HTTPRouter
	
	// HTTP methods
	GET(path string, handler HTTPHandler)
	POST(path string, handler HTTPHandler)
	PUT(path string, handler HTTPHandler)
	DELETE(path string, handler HTTPHandler)
	PATCH(path string, handler HTTPHandler)
}

// HTTPResponse represents HTTP response structure
type HTTPResponse struct {
	StatusCode int                    `json:"status_code"`
	Headers    map[string]string      `json:"headers"`
	Body       interface{}            `json:"body"`
	Error      string                 `json:"error,omitempty"`
}

// HTTPRequest represents HTTP request structure
type HTTPRequest struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Query   map[string]string `json:"query"`
	Headers map[string]string `json:"headers"`
	Body    []byte            `json:"body"`
}

// ProxyHandler defines proxy-specific handler interface
// TODO: Migrate from handlers/proxy/
type ProxyHandler interface {
	HTTPHandler
	
	// HandleProxy processes proxy requests
	HandleProxy(ctx HTTPContext, proxyType string) error
}

// HealthHandler defines health check handler interface
// TODO: Migrate from health/
type HealthHandler interface {
	HTTPHandler
	
	// CheckHealth performs health check
	CheckHealth(ctx HTTPContext) error
}