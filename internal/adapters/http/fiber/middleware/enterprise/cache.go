// Package enterprise provides enterprise middleware components
package enterprise

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// CacheEntry represents a cached response
type CacheEntry struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
	CachedAt   time.Time
	ExpiresAt  time.Time
}

// CacheStore defines the interface for cache storage
type CacheStore interface {
	Get(key string) (*CacheEntry, bool)
	Set(key string, entry *CacheEntry, ttl time.Duration) error
	Delete(key string) error
	Clear() error
	Size() int
}

// InMemoryCacheStore implements CacheStore using in-memory map
type InMemoryCacheStore struct {
	data sync.Map
	mu   sync.RWMutex
}

// NewInMemoryCacheStore creates a new in-memory cache store
func NewInMemoryCacheStore() *InMemoryCacheStore {
	store := &InMemoryCacheStore{}

	// Start cleanup goroutine
	go store.cleanupExpired()

	return store
}

// Get retrieves a cache entry
func (s *InMemoryCacheStore) Get(key string) (*CacheEntry, bool) {
	val, ok := s.data.Load(key)
	if !ok {
		return nil, false
	}

	entry := val.(*CacheEntry)

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		s.data.Delete(key)
		return nil, false
	}

	return entry, true
}

// Set stores a cache entry
func (s *InMemoryCacheStore) Set(key string, entry *CacheEntry, ttl time.Duration) error {
	entry.ExpiresAt = time.Now().Add(ttl)
	s.data.Store(key, entry)
	return nil
}

// Delete removes a cache entry
func (s *InMemoryCacheStore) Delete(key string) error {
	s.data.Delete(key)
	return nil
}

// Clear removes all cache entries
func (s *InMemoryCacheStore) Clear() error {
	s.data.Range(func(key, value interface{}) bool {
		s.data.Delete(key)
		return true
	})
	return nil
}

// Size returns the number of cached entries
func (s *InMemoryCacheStore) Size() int {
	count := 0
	s.data.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// cleanupExpired periodically removes expired entries
func (s *InMemoryCacheStore) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		s.data.Range(func(key, value interface{}) bool {
			entry := value.(*CacheEntry)
			if now.After(entry.ExpiresAt) {
				s.data.Delete(key)
			}
			return true
		})
	}
}

// CacheConfig defines cache middleware configuration
type CacheConfig struct {
	// TTL is the time-to-live for cached entries
	TTL time.Duration

	// Store is the cache storage backend
	Store CacheStore

	// KeyGenerator generates cache keys from requests
	// Default: method + path + query
	KeyGenerator func(c *fiber.Ctx) string

	// ShouldCache determines if a request should be cached
	// Default: cache only GET requests with 200 status
	ShouldCache func(c *fiber.Ctx) bool

	// Skip allows conditional skipping of caching
	Skip func(c *fiber.Ctx) bool
}

// CacheMiddleware provides response caching
type CacheMiddleware struct {
	config CacheConfig
}

// NewCacheMiddleware creates a new cache middleware
func NewCacheMiddleware(config CacheConfig) *CacheMiddleware {
	// Set defaults
	if config.TTL == 0 {
		config.TTL = 5 * time.Minute
	}
	if config.Store == nil {
		config.Store = NewInMemoryCacheStore()
	}
	if config.KeyGenerator == nil {
		config.KeyGenerator = defaultKeyGenerator
	}
	if config.ShouldCache == nil {
		config.ShouldCache = defaultShouldCache
	}

	return &CacheMiddleware{
		config: config,
	}
}

// Middleware returns the cache middleware handler
func (cm *CacheMiddleware) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if caching should be skipped
		if cm.config.Skip != nil && cm.config.Skip(c) {
			return c.Next()
		}

		// Only cache GET requests by default
		if c.Method() != "GET" {
			return c.Next()
		}

		// Generate cache key
		key := cm.config.KeyGenerator(c)

		// Try to get from cache
		if entry, ok := cm.config.Store.Get(key); ok {
			// Cache hit
			c.Set("X-Cache", "HIT")
			c.Set("X-Cache-Age", time.Since(entry.CachedAt).String())

			// Restore headers
			for name, value := range entry.Headers {
				c.Set(name, value)
			}

			return c.Status(entry.StatusCode).Send(entry.Body)
		}

		// Cache miss - set header and continue
		c.Set("X-Cache", "MISS")

		// Intercept response to cache it
		err := c.Next()
		if err != nil {
			return err
		}

		// Check if response should be cached
		if !cm.config.ShouldCache(c) {
			return nil
		}

		// Create cache entry
		entry := &CacheEntry{
			StatusCode: c.Response().StatusCode(),
			Headers:    make(map[string]string),
			Body:       c.Response().Body(),
			CachedAt:   time.Now(),
		}

		// Copy important headers
		importantHeaders := []string{
			"Content-Type",
			"Content-Encoding",
			"Cache-Control",
			"ETag",
		}
		for _, header := range importantHeaders {
			if value := c.Get(header); value != "" {
				entry.Headers[header] = value
			}
		}

		// Store in cache
		_ = cm.config.Store.Set(key, entry, cm.config.TTL)

		return nil
	}
}

// Invalidate removes cache entries matching a pattern
func (cm *CacheMiddleware) Invalidate(pattern string) error {
	// For now, just clear all cache
	// In production, implement pattern matching
	return cm.config.Store.Clear()
}

// GetStore returns the underlying cache store
func (cm *CacheMiddleware) GetStore() CacheStore {
	return cm.config.Store
}

// defaultKeyGenerator generates a cache key from request
func defaultKeyGenerator(c *fiber.Ctx) string {
	// Use method + path + query
	data := c.Method() + ":" + c.Path()
	if queryString := c.Request().URI().QueryString(); len(queryString) > 0 {
		data += "?" + string(queryString)
	}

	// Hash for shorter keys
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16]) // Use first 16 bytes
}

// defaultShouldCache determines if response should be cached
func defaultShouldCache(c *fiber.Ctx) bool {
	status := c.Response().StatusCode()

	// Only cache successful responses
	if status != 200 {
		return false
	}

	// Don't cache if Content-Type is not JSON
	contentType := c.Get("Content-Type")
	if contentType != "application/json" && contentType != "application/json; charset=utf-8" {
		return false
	}

	return true
}

// CacheForEndpoint creates a cache middleware for specific endpoints
func CacheForEndpoint(ttl time.Duration) *CacheMiddleware {
	return NewCacheMiddleware(CacheConfig{
		TTL: ttl,
	})
}

// AnalyticsCacheMiddleware creates cache middleware optimized for analytics endpoints
func AnalyticsCacheMiddleware() *CacheMiddleware {
	return NewCacheMiddleware(CacheConfig{
		TTL: 2 * time.Minute, // Analytics data changes less frequently
		KeyGenerator: func(c *fiber.Ctx) string {
			// Include user context if available
			userID := ""
			if uid := c.Locals("user_id"); uid != nil {
				if id, ok := uid.(string); ok {
					userID = id
				}
			}

			data := c.Method() + ":" + c.Path() + ":user:" + userID
			if queryString := c.Request().URI().QueryString(); len(queryString) > 0 {
				data += "?" + string(queryString)
			}

			hash := sha256.Sum256([]byte(data))
			return "analytics:" + hex.EncodeToString(hash[:16])
		},
	})
}

// StatsResponse represents cached stats response
type StatsResponse struct {
	Success  bool                   `json:"success"`
	Data     interface{}            `json:"data"`
	Metadata map[string]interface{} `json:"metadata"`
	Cached   bool                   `json:"cached,omitempty"`
	CacheAge string                 `json:"cache_age,omitempty"`
}

// CacheStats returns cache statistics
func (cm *CacheMiddleware) CacheStats() map[string]interface{} {
	size := cm.config.Store.Size()

	return map[string]interface{}{
		"entries": size,
		"ttl":     cm.config.TTL.String(),
	}
}

