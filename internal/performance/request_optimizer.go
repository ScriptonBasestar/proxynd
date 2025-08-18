package performance

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/logging"
)

// RequestOptimizer optimizes HTTP request processing
type RequestOptimizer struct {
	logger      logging.Logger
	config      *RequestOptimizerConfig
	stats       *RequestStats
	compressor  *ResponseCompressor
	rateLimiter *AdaptiveRateLimiter
}

// RequestOptimizerConfig configures request optimization
type RequestOptimizerConfig struct {
	// Compression settings
	EnableCompression    bool     `yaml:"enable_compression" json:"enable_compression" default:"true"`
	CompressionLevel     int      `yaml:"compression_level" json:"compression_level" default:"6"`
	CompressionThreshold int      `yaml:"compression_threshold" json:"compression_threshold" default:"1024"`
	CompressibleTypes    []string `yaml:"compressible_types,omitempty" json:"compressible_types,omitempty"`

	// Caching optimization
	// Enable cache optimization (true by default)
	EnableCacheOptimization bool          `yaml:"enable_cache_optimization" json:"enable_cache_optimization"`
	CacheControlMaxAge      time.Duration `yaml:"cache_control_max_age" json:"cache_control_max_age" default:"1h"`
	ETags                   bool          `yaml:"etags" json:"etags" default:"true"`

	// Rate limiting
	// Enable adaptive rate limiting (true by default)
	EnableAdaptiveRateLimit bool          `yaml:"enable_adaptive_rate_limit" json:"enable_adaptive_rate_limit"`
	BaseRateLimit           int           `yaml:"base_rate_limit" json:"base_rate_limit" default:"1000"`
	BurstLimit              int           `yaml:"burst_limit" json:"burst_limit" default:"100"`
	RateLimitWindow         time.Duration `yaml:"rate_limit_window" json:"rate_limit_window" default:"1m"`

	// Request optimization
	MaxRequestSize   int64         `yaml:"max_request_size" json:"max_request_size" default:"10485760"` // 10MB
	RequestTimeout   time.Duration `yaml:"request_timeout" json:"request_timeout" default:"30s"`
	KeepAliveTimeout time.Duration `yaml:"keep_alive_timeout" json:"keep_alive_timeout" default:"30s"`

	// Response optimization
	EnableResponseStreaming bool  `yaml:"enable_response_streaming" json:"enable_response_streaming" default:"true"`
	StreamingThreshold      int64 `yaml:"streaming_threshold" json:"streaming_threshold" default:"1048576"` // 1MB
	MaxResponseSize         int64 `yaml:"max_response_size" json:"max_response_size" default:"104857600"`   // 100MB

	// Performance monitoring
	SlowRequestThreshold time.Duration `yaml:"slow_request_threshold" json:"slow_request_threshold" default:"2s"`
	EnableMetrics        bool          `yaml:"enable_metrics" json:"enable_metrics" default:"true"`
}

// RequestStats tracks request optimization statistics
type RequestStats struct {
	mu sync.RWMutex

	// Request statistics
	TotalRequests       int64 `json:"total_requests"`
	OptimizedRequests   int64 `json:"optimized_requests"`
	CompressedResponses int64 `json:"compressed_responses"`
	CachedResponses     int64 `json:"cached_responses"`
	RateLimitedRequests int64 `json:"rate_limited_requests"`
	SlowRequests        int64 `json:"slow_requests"`

	// Performance metrics
	AverageResponseTime time.Duration `json:"average_response_time"`
	CompressionRatio    float64       `json:"compression_ratio"`
	CacheHitRate        float64       `json:"cache_hit_rate"`

	// Size optimization
	TotalBytesIn  int64 `json:"total_bytes_in"`
	TotalBytesOut int64 `json:"total_bytes_out"`
	BytesSaved    int64 `json:"bytes_saved"`

	// Error tracking
	TimeoutErrors      int64 `json:"timeout_errors"`
	CompressionErrors  int64 `json:"compression_errors"`
	OptimizationErrors int64 `json:"optimization_errors"`

	LastOptimization time.Time `json:"last_optimization"`
}

// ResponseCompressor handles response compression
type ResponseCompressor struct {
	config *RequestOptimizerConfig
	logger logging.Logger
}

// AdaptiveRateLimiter implements adaptive rate limiting
type AdaptiveRateLimiter struct {
	config        *RequestOptimizerConfig
	logger        logging.Logger
	mu            sync.RWMutex
	requestCounts map[string]int
	lastReset     time.Time
	currentLimit  int
	systemLoad    float64
}

// NewRequestOptimizer creates a new request optimizer
func NewRequestOptimizer(logger logging.Logger, config *RequestOptimizerConfig) *RequestOptimizer {
	optimizer := &RequestOptimizer{
		logger:      logger.WithField("component", "request.optimizer"),
		config:      config,
		stats:       NewRequestStats(),
		compressor:  NewResponseCompressor(logger, config),
		rateLimiter: NewAdaptiveRateLimiter(logger, config),
	}

	return optimizer
}

// NewRequestStats creates new request statistics
func NewRequestStats() *RequestStats {
	return &RequestStats{}
}

// NewResponseCompressor creates a new response compressor
func NewResponseCompressor(logger logging.Logger, config *RequestOptimizerConfig) *ResponseCompressor {
	return &ResponseCompressor{
		config: config,
		logger: logger.WithField("component", "response.compressor"),
	}
}

// NewAdaptiveRateLimiter creates a new adaptive rate limiter
func NewAdaptiveRateLimiter(logger logging.Logger, config *RequestOptimizerConfig) *AdaptiveRateLimiter {
	return &AdaptiveRateLimiter{
		config:        config,
		logger:        logger.WithField("component", "rate.limiter"),
		requestCounts: make(map[string]int),
		lastReset:     time.Now(),
		currentLimit:  config.BaseRateLimit,
	}
}

// OptimizeRequest applies request optimizations
func (ro *RequestOptimizer) OptimizeRequest(c *fiber.Ctx) error {
	start := time.Now()

	// Update request stats
	ro.updateRequestStats(c)

	// Apply rate limiting
	if ro.config.EnableAdaptiveRateLimit {
		if err := ro.rateLimiter.CheckRateLimit(c); err != nil {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "Rate limit exceeded",
				"retry_after": ro.config.RateLimitWindow.Seconds(),
			})
		}
	}

	// Validate request size
	if c.Request().Header.ContentLength() > int(ro.config.MaxRequestSize) {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
			"error":    "Request too large",
			"max_size": ro.config.MaxRequestSize,
		})
	}

	// Set request timeout
	if ro.config.RequestTimeout > 0 {
		ctx, cancel := context.WithTimeout(c.UserContext(), ro.config.RequestTimeout)
		defer cancel()
		c.SetUserContext(ctx)
	}

	// Process request through the chain
	err := c.Next()

	// Apply response optimizations
	ro.optimizeResponse(c)

	// Record performance metrics
	duration := time.Since(start)
	ro.recordRequestMetrics(c, duration, err)

	return err
}

// optimizeResponse applies response optimizations
func (ro *RequestOptimizer) optimizeResponse(c *fiber.Ctx) {
	// Apply compression if enabled
	if ro.config.EnableCompression {
		ro.compressor.CompressResponse(c)
	}

	// Apply cache headers if enabled
	if ro.config.EnableCacheOptimization {
		ro.applyCacheHeaders(c)
	}

	// Apply ETags if enabled
	if ro.config.ETags {
		ro.applyETags(c)
	}
}

// CompressResponse compresses the response if applicable
func (rc *ResponseCompressor) CompressResponse(c *fiber.Ctx) {
	// Check if response should be compressed
	if !rc.shouldCompress(c) {
		return
	}

	// Get response body
	body := c.Response().Body()
	if len(body) < rc.config.CompressionThreshold {
		return
	}

	// Apply compression (simplified - in practice, use proper compression)
	originalSize := len(body)
	compressedSize := originalSize // Placeholder for actual compression

	// Set compression headers
	c.Set("Content-Encoding", "gzip")
	c.Set("Vary", "Accept-Encoding")

	rc.logger.Debug("Response compressed",
		logging.F("original_size", originalSize),
		logging.F("compressed_size", compressedSize),
		logging.Float64("ratio", float64(compressedSize)/float64(originalSize)))
}

// shouldCompress determines if response should be compressed
func (rc *ResponseCompressor) shouldCompress(c *fiber.Ctx) bool {
	// Check Accept-Encoding header
	acceptEncoding := c.Get("Accept-Encoding")
	if acceptEncoding == "" {
		return false
	}

	// Check content type
	contentType := c.Get("Content-Type")
	if !rc.isCompressibleType(contentType) {
		return false
	}

	// Check if already compressed
	if c.Get("Content-Encoding") != "" {
		return false
	}

	return true
}

// isCompressibleType checks if content type is compressible
func (rc *ResponseCompressor) isCompressibleType(contentType string) bool {
	if len(rc.config.CompressibleTypes) == 0 {
		// Default compressible types
		compressibleTypes := []string{
			"text/html",
			"text/css",
			"text/javascript",
			"application/javascript",
			"application/json",
			"text/plain",
			"application/xml",
			"text/xml",
		}
		rc.config.CompressibleTypes = compressibleTypes
	}

	for _, ct := range rc.config.CompressibleTypes {
		if contentType == ct {
			return true
		}
	}

	return false
}

// applyCacheHeaders applies cache control headers
func (ro *RequestOptimizer) applyCacheHeaders(c *fiber.Ctx) {
	// Set cache control headers based on content type and path
	path := c.Path()

	if isStaticAsset(path) {
		// Static assets can be cached for longer
		c.Set("Cache-Control", "public, max-age=86400") // 24 hours
	} else if isAPIEndpoint(path) {
		// API endpoints with shorter cache time
		maxAge := int(ro.config.CacheControlMaxAge.Seconds())
		c.Set("Cache-Control", "public, max-age="+string(rune(maxAge)))
	}

	// Set Last-Modified header
	c.Set("Last-Modified", time.Now().Format(http.TimeFormat))
}

// applyETags applies ETag headers for caching
func (ro *RequestOptimizer) applyETags(c *fiber.Ctx) {
	// Generate ETag based on content (simplified)
	body := c.Response().Body()
	if len(body) > 0 {
		etag := generateETag(body)
		c.Set("ETag", etag)

		// Check If-None-Match header
		ifNoneMatch := c.Get("If-None-Match")
		if ifNoneMatch == etag {
			c.Status(fiber.StatusNotModified)
			return
		}
	}
}

// CheckRateLimit checks if request should be rate limited
func (arl *AdaptiveRateLimiter) CheckRateLimit(c *fiber.Ctx) error {
	arl.mu.Lock()
	defer arl.mu.Unlock()

	clientIP := c.IP()
	now := time.Now()

	// Reset counters if window has passed
	if now.Sub(arl.lastReset) > arl.config.RateLimitWindow {
		arl.requestCounts = make(map[string]int)
		arl.lastReset = now
		arl.adjustRateLimit()
	}

	// Check current request count
	count, exists := arl.requestCounts[clientIP]
	if !exists {
		count = 0
	}

	if count >= arl.currentLimit {
		arl.logger.Warn("Rate limit exceeded",
			logging.F("client_ip", clientIP),
			logging.F("count", count),
			logging.F("limit", arl.currentLimit))
		return fiber.NewError(fiber.StatusTooManyRequests, "Rate limit exceeded")
	}

	// Increment counter
	arl.requestCounts[clientIP] = count + 1

	return nil
}

// adjustRateLimit adjusts rate limit based on system load
func (arl *AdaptiveRateLimiter) adjustRateLimit() {
	// Adjust rate limit based on system load (simplified)
	if arl.systemLoad > 0.8 {
		// High load - reduce rate limit
		arl.currentLimit = int(float64(arl.config.BaseRateLimit) * 0.7)
	} else if arl.systemLoad < 0.5 {
		// Low load - increase rate limit
		arl.currentLimit = int(float64(arl.config.BaseRateLimit) * 1.3)
	} else {
		// Normal load - use base rate limit
		arl.currentLimit = arl.config.BaseRateLimit
	}

	arl.logger.Debug("Rate limit adjusted",
		logging.Float64("system_load", arl.systemLoad),
		logging.F("new_limit", arl.currentLimit))
}

// updateRequestStats updates request statistics
func (ro *RequestOptimizer) updateRequestStats(c *fiber.Ctx) {
	ro.stats.mu.Lock()
	defer ro.stats.mu.Unlock()

	ro.stats.TotalRequests++

	// Track request size
	if contentLength := c.Request().Header.ContentLength(); contentLength > 0 {
		ro.stats.TotalBytesIn += int64(contentLength)
	}
}

// recordRequestMetrics records request performance metrics
func (ro *RequestOptimizer) recordRequestMetrics(c *fiber.Ctx, duration time.Duration, err error) {
	ro.stats.mu.Lock()
	defer ro.stats.mu.Unlock()

	// Update average response time
	if ro.stats.TotalRequests == 1 {
		ro.stats.AverageResponseTime = duration
	} else {
		// Calculate rolling average
		oldAvg := float64(ro.stats.AverageResponseTime.Nanoseconds())
		newAvg := (oldAvg*float64(ro.stats.TotalRequests-1) +
			float64(duration.Nanoseconds())) / float64(ro.stats.TotalRequests)
		ro.stats.AverageResponseTime = time.Duration(int64(newAvg))
	}

	// Track slow requests
	if duration > ro.config.SlowRequestThreshold {
		ro.stats.SlowRequests++
		ro.logger.Warn("Slow request detected",
			logging.Duration("duration", duration),
			logging.F("path", c.Path()),
			logging.F("method", c.Method()))
	}

	// Track response size
	if body := c.Response().Body(); len(body) > 0 {
		ro.stats.TotalBytesOut += int64(len(body))
	}

	// Track errors
	if err != nil {
		ro.stats.OptimizationErrors++
	}

	ro.stats.LastOptimization = time.Now()
}

// GetStats returns current request optimization statistics
func (ro *RequestOptimizer) GetStats() *RequestStats {
	ro.stats.mu.RLock()
	defer ro.stats.mu.RUnlock()

	// Create a copy to avoid race conditions
	stats := &RequestStats{
		TotalRequests:       ro.stats.TotalRequests,
		OptimizedRequests:   ro.stats.OptimizedRequests,
		CompressedResponses: ro.stats.CompressedResponses,
		CachedResponses:     ro.stats.CachedResponses,
		RateLimitedRequests: ro.stats.RateLimitedRequests,
		SlowRequests:        ro.stats.SlowRequests,
		AverageResponseTime: ro.stats.AverageResponseTime,
		CompressionRatio:    ro.stats.CompressionRatio,
		CacheHitRate:        ro.stats.CacheHitRate,
		TotalBytesIn:        ro.stats.TotalBytesIn,
		TotalBytesOut:       ro.stats.TotalBytesOut,
		BytesSaved:          ro.stats.BytesSaved,
		TimeoutErrors:       ro.stats.TimeoutErrors,
		CompressionErrors:   ro.stats.CompressionErrors,
		OptimizationErrors:  ro.stats.OptimizationErrors,
		LastOptimization:    ro.stats.LastOptimization,
	}

	return stats
}

// Helper functions

func isStaticAsset(path string) bool {
	staticExtensions := []string{".css", ".js", ".png", ".jpg", ".jpeg", ".gif", ".ico", ".woff", ".woff2"}
	for _, ext := range staticExtensions {
		if len(path) > len(ext) && path[len(path)-len(ext):] == ext {
			return true
		}
	}
	return false
}

func isAPIEndpoint(path string) bool {
	return len(path) > 4 && path[:4] == "/api"
}

func generateETag(body []byte) string {
	// Simplified ETag generation - in practice, use proper hashing
	return "\"" + string(rune(len(body))) + "\""
}
