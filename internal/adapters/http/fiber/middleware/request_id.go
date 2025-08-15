package middlewares

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"

	"proxynd/internal/ports"
)

// RequestIDMiddleware implements ports.HTTPMiddleware for request ID generation
type RequestIDMiddleware struct {
	config    *ports.RequestIDConfig
	generator ports.RequestIDGenerator
	logger    ports.Logger
}

// UUIDRequestIDGenerator implements ports.RequestIDGenerator using UUID
type UUIDRequestIDGenerator struct {
	headerName string
}

// NanoIDRequestIDGenerator implements ports.RequestIDGenerator using NanoID
type NanoIDRequestIDGenerator struct {
	headerName string
	alphabet   string
	size       int
}

// NewRequestIDMiddleware creates a new request ID middleware
func NewRequestIDMiddleware(
	config *ports.RequestIDConfig,
	generator ports.RequestIDGenerator,
	logger ports.Logger,
) ports.EnhancedHTTPMiddleware {
	if config == nil {
		config = &ports.RequestIDConfig{
			Enabled:    true,
			HeaderName: "X-Request-ID",
			Generator:  "uuid",
		}
	}
	
	if generator == nil {
		generator = NewUUIDRequestIDGenerator(config.HeaderName)
	}
	
	return &RequestIDMiddleware{
		config:    config,
		generator: generator,
		logger:    logger,
	}
}

// Process implements ports.HTTPMiddleware
func (m *RequestIDMiddleware) Process(next ports.HTTPHandler) ports.HTTPHandler {
	return &RequestIDHandler{
		next:      next,
		config:    m.config,
		generator: m.generator,
		logger:    m.logger,
	}
}

// Order returns the middleware execution order
func (m *RequestIDMiddleware) Order() ports.MiddlewareOrder {
	return ports.OrderRequestID
}

// Name returns the middleware name
func (m *RequestIDMiddleware) Name() string {
	return "request_id"
}

// IsEnabled returns whether the middleware is enabled
func (m *RequestIDMiddleware) IsEnabled() bool {
	return m.config.Enabled
}

// RequestIDHandler handles request ID generation
type RequestIDHandler struct {
	next      ports.HTTPHandler
	config    *ports.RequestIDConfig
	generator ports.RequestIDGenerator
	logger    ports.Logger
}

// Handle processes the request
func (h *RequestIDHandler) Handle(ctx ports.HTTPContext) error {
	// Try to extract existing request ID from headers
	requestID := h.generator.Extract(ctx)
	
	// Generate new request ID if not present
	if requestID == "" {
		requestID = h.generator.Generate()
	}
	
	// Set request ID in context and response headers
	h.generator.SetRequestID(ctx, requestID)
	ctx.SetHeader(h.config.HeaderName, requestID)
	
	// Log request ID if logger is available
	if h.logger != nil {
		h.logger.Debug(ctx.Context(), "Request ID generated",
			ports.Field(&RequestIDField{key: "request_id", value: requestID}),
			ports.Field(&RequestIDField{key: "method", value: ctx.Method()}),
			ports.Field(&RequestIDField{key: "path", value: ctx.Path()}),
		)
	}
	
	return h.next.Handle(ctx)
}

// RequestIDField implements ports.Field for request ID logging
type RequestIDField struct {
	key   string
	value interface{}
}

// Key returns the field key
func (f *RequestIDField) Key() string {
	return f.key
}

// Value returns the field value
func (f *RequestIDField) Value() interface{} {
	return f.value
}

// NewUUIDRequestIDGenerator creates a new UUID-based request ID generator
func NewUUIDRequestIDGenerator(headerName string) ports.RequestIDGenerator {
	if headerName == "" {
		headerName = "X-Request-ID"
	}
	
	return &UUIDRequestIDGenerator{
		headerName: headerName,
	}
}

// Generate generates a new UUID request ID
func (g *UUIDRequestIDGenerator) Generate() string {
	return uuid.New().String()
}

// Extract extracts request ID from HTTP context
func (g *UUIDRequestIDGenerator) Extract(ctx ports.HTTPContext) string {
	// Try to get from header first
	if requestID := ctx.Header(g.headerName); requestID != "" {
		return requestID
	}
	
	// Try to get from context locals
	if requestID, ok := ctx.Get("request_id").(string); ok && requestID != "" {
		return requestID
	}
	
	return ""
}

// SetRequestID sets request ID in HTTP context
func (g *UUIDRequestIDGenerator) SetRequestID(ctx ports.HTTPContext, id string) {
	ctx.Set("request_id", id)
	ctx.SetHeader(g.headerName, id)
}

// NewNanoIDRequestIDGenerator creates a new NanoID-based request ID generator
func NewNanoIDRequestIDGenerator(headerName string, alphabet string, size int) ports.RequestIDGenerator {
	if headerName == "" {
		headerName = "X-Request-ID"
	}
	
	if alphabet == "" {
		alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	}
	
	if size <= 0 {
		size = 21 // Default NanoID size
	}
	
	return &NanoIDRequestIDGenerator{
		headerName: headerName,
		alphabet:   alphabet,
		size:       size,
	}
}

// Generate generates a new NanoID request ID
func (g *NanoIDRequestIDGenerator) Generate() string {
	return g.generateNanoID()
}

// Extract extracts request ID from HTTP context
func (g *NanoIDRequestIDGenerator) Extract(ctx ports.HTTPContext) string {
	// Try to get from header first
	if requestID := ctx.Header(g.headerName); requestID != "" {
		return requestID
	}
	
	// Try to get from context locals
	if requestID, ok := ctx.Get("request_id").(string); ok && requestID != "" {
		return requestID
	}
	
	return ""
}

// SetRequestID sets request ID in HTTP context
func (g *NanoIDRequestIDGenerator) SetRequestID(ctx ports.HTTPContext, id string) {
	ctx.Set("request_id", id)
	ctx.SetHeader(g.headerName, id)
}

// generateNanoID generates a NanoID string
func (g *NanoIDRequestIDGenerator) generateNanoID() string {
	bytes := make([]byte, g.size)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to UUID if random generation fails
		return uuid.New().String()
	}
	
	alphabetLen := len(g.alphabet)
	for i := 0; i < g.size; i++ {
		bytes[i] = g.alphabet[int(bytes[i])%alphabetLen]
	}
	
	return string(bytes)
}

// CustomRequestIDGenerator allows custom request ID generation
type CustomRequestIDGenerator struct {
	headerName string
	generator  func() string
}

// NewCustomRequestIDGenerator creates a new custom request ID generator
func NewCustomRequestIDGenerator(headerName string, generator func() string) ports.RequestIDGenerator {
	if headerName == "" {
		headerName = "X-Request-ID"
	}
	
	if generator == nil {
		// Fallback to UUID generator
		generator = func() string {
			return uuid.New().String()
		}
	}
	
	return &CustomRequestIDGenerator{
		headerName: headerName,
		generator:  generator,
	}
}

// Generate generates a new request ID using the custom function
func (g *CustomRequestIDGenerator) Generate() string {
	return g.generator()
}

// Extract extracts request ID from HTTP context
func (g *CustomRequestIDGenerator) Extract(ctx ports.HTTPContext) string {
	// Try to get from header first
	if requestID := ctx.Header(g.headerName); requestID != "" {
		return requestID
	}
	
	// Try to get from context locals
	if requestID, ok := ctx.Get("request_id").(string); ok && requestID != "" {
		return requestID
	}
	
	return ""
}

// SetRequestID sets request ID in HTTP context
func (g *CustomRequestIDGenerator) SetRequestID(ctx ports.HTTPContext, id string) {
	ctx.Set("request_id", id)
	ctx.SetHeader(g.headerName, id)
}

// HexRequestIDGenerator generates hexadecimal request IDs
type HexRequestIDGenerator struct {
	headerName string
	length     int
}

// NewHexRequestIDGenerator creates a new hex-based request ID generator
func NewHexRequestIDGenerator(headerName string, length int) ports.RequestIDGenerator {
	if headerName == "" {
		headerName = "X-Request-ID"
	}
	
	if length <= 0 {
		length = 16 // Default hex length (32 characters)
	}
	
	return &HexRequestIDGenerator{
		headerName: headerName,
		length:     length,
	}
}

// Generate generates a new hex request ID
func (g *HexRequestIDGenerator) Generate() string {
	bytes := make([]byte, g.length)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to UUID if random generation fails
		return uuid.New().String()
	}
	
	return hex.EncodeToString(bytes)
}

// Extract extracts request ID from HTTP context
func (g *HexRequestIDGenerator) Extract(ctx ports.HTTPContext) string {
	// Try to get from header first
	if requestID := ctx.Header(g.headerName); requestID != "" {
		return requestID
	}
	
	// Try to get from context locals
	if requestID, ok := ctx.Get("request_id").(string); ok && requestID != "" {
		return requestID
	}
	
	return ""
}

// SetRequestID sets request ID in HTTP context
func (g *HexRequestIDGenerator) SetRequestID(ctx ports.HTTPContext, id string) {
	ctx.Set("request_id", id)
	ctx.SetHeader(g.headerName, id)
}

// CreateRequestIDGenerator creates a request ID generator based on configuration
func CreateRequestIDGenerator(config *ports.RequestIDConfig) ports.RequestIDGenerator {
	if config == nil {
		return NewUUIDRequestIDGenerator("")
	}
	
	switch config.Generator {
	case "uuid":
		return NewUUIDRequestIDGenerator(config.HeaderName)
	case "nanoid":
		return NewNanoIDRequestIDGenerator(config.HeaderName, "", 0)
	case "hex":
		return NewHexRequestIDGenerator(config.HeaderName, 0)
	default:
		return NewUUIDRequestIDGenerator(config.HeaderName)
	}
}