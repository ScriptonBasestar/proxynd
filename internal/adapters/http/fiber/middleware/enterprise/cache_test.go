package enterprise

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewInMemoryCacheStore tests cache store creation
func TestNewInMemoryCacheStore(t *testing.T) {
	store := NewInMemoryCacheStore()
	assert.NotNil(t, store)
	assert.Equal(t, 0, store.Size())
}

// TestInMemoryCacheStore_SetAndGet tests basic cache operations
func TestInMemoryCacheStore_SetAndGet(t *testing.T) {
	store := NewInMemoryCacheStore()

	entry := &CacheEntry{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       []byte(`{"test":"data"}`),
		CachedAt:   time.Now(),
	}

	// Set entry
	err := store.Set("test-key", entry, 1*time.Minute)
	require.NoError(t, err)

	// Get entry
	retrieved, ok := store.Get("test-key")
	require.True(t, ok)
	assert.Equal(t, entry.StatusCode, retrieved.StatusCode)
	assert.Equal(t, entry.Body, retrieved.Body)
}

// TestInMemoryCacheStore_Expiration tests cache expiration
func TestInMemoryCacheStore_Expiration(t *testing.T) {
	store := NewInMemoryCacheStore()

	entry := &CacheEntry{
		StatusCode: 200,
		Body:       []byte(`{"test":"data"}`),
		CachedAt:   time.Now(),
	}

	// Set with short TTL
	err := store.Set("test-key", entry, 100*time.Millisecond)
	require.NoError(t, err)

	// Should exist immediately
	_, ok := store.Get("test-key")
	assert.True(t, ok)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	_, ok = store.Get("test-key")
	assert.False(t, ok)
}

// TestInMemoryCacheStore_Delete tests cache deletion
func TestInMemoryCacheStore_Delete(t *testing.T) {
	store := NewInMemoryCacheStore()

	entry := &CacheEntry{
		StatusCode: 200,
		Body:       []byte(`{"test":"data"}`),
		CachedAt:   time.Now(),
	}

	_ = store.Set("test-key", entry, 1*time.Minute)
	assert.Equal(t, 1, store.Size())

	err := store.Delete("test-key")
	require.NoError(t, err)

	assert.Equal(t, 0, store.Size())
	_, ok := store.Get("test-key")
	assert.False(t, ok)
}

// TestInMemoryCacheStore_Clear tests clearing all cache
func TestInMemoryCacheStore_Clear(t *testing.T) {
	store := NewInMemoryCacheStore()

	// Add multiple entries
	for i := 0; i < 5; i++ {
		entry := &CacheEntry{
			StatusCode: 200,
			Body:       []byte(`{"test":"data"}`),
			CachedAt:   time.Now(),
		}
	_ = store.Set(string(rune('a'+i)), entry, 1*time.Minute)
	}

	assert.Equal(t, 5, store.Size())

	err := store.Clear()
	require.NoError(t, err)

	assert.Equal(t, 0, store.Size())
}

// TestCacheMiddleware_CacheHit tests cache hit scenario
func TestCacheMiddleware_CacheHit(t *testing.T) {
	app := fiber.New()
	cache := NewCacheMiddleware(CacheConfig{
		TTL: 1 * time.Minute,
	})

	app.Use(cache.Middleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"data": "response"})
	})

	// First request - should miss cache
	req1 := httptest.NewRequest("GET", "/test", nil)
	resp1, err := app.Test(req1)
	require.NoError(t, err)
	defer func() { _ = resp1.Body.Close() }()

	assert.Equal(t, 200, resp1.StatusCode)
	assert.Equal(t, "MISS", resp1.Header.Get("X-Cache"))

	// Second request - should hit cache
	req2 := httptest.NewRequest("GET", "/test", nil)
	resp2, err := app.Test(req2)
	require.NoError(t, err)
	defer func() { _ = resp2.Body.Close() }()

	assert.Equal(t, 200, resp2.StatusCode)
	assert.Equal(t, "HIT", resp2.Header.Get("X-Cache"))
	assert.NotEmpty(t, resp2.Header.Get("X-Cache-Age"))
}

// TestCacheMiddleware_OnlyGET tests that only GET requests are cached
func TestCacheMiddleware_OnlyGET(t *testing.T) {
	app := fiber.New()
	cache := NewCacheMiddleware(CacheConfig{
		TTL: 1 * time.Minute,
	})

	app.Use(cache.Middleware())
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"data": "response"})
	})

	// POST request should not be cached
	req := httptest.NewRequest("POST", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, 200, resp.StatusCode)
	assert.Empty(t, resp.Header.Get("X-Cache"))
}

// TestCacheMiddleware_TTL tests cache TTL
func TestCacheMiddleware_TTL(t *testing.T) {
	app := fiber.New()
	cache := NewCacheMiddleware(CacheConfig{
		TTL: 100 * time.Millisecond,
	})

	app.Use(cache.Middleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"timestamp": time.Now().UnixNano()})
	})

	// First request
	req1 := httptest.NewRequest("GET", "/test", nil)
	resp1, err := app.Test(req1)
	require.NoError(t, err)
	defer func() { _ = resp1.Body.Close() }()

	body1, _ := io.ReadAll(resp1.Body)

	// Immediate second request should hit cache
	req2 := httptest.NewRequest("GET", "/test", nil)
	resp2, err := app.Test(req2)
	require.NoError(t, err)
	defer func() { _ = resp2.Body.Close() }()

	body2, _ := io.ReadAll(resp2.Body)
	assert.Equal(t, "HIT", resp2.Header.Get("X-Cache"))
	assert.Equal(t, body1, body2, "Cached response should be identical")

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Third request should miss cache (expired)
	req3 := httptest.NewRequest("GET", "/test", nil)
	resp3, err := app.Test(req3)
	require.NoError(t, err)
	defer func() { _ = resp3.Body.Close() }()

	assert.Equal(t, "MISS", resp3.Header.Get("X-Cache"))
}

// TestCacheMiddleware_DifferentPaths tests caching different paths
func TestCacheMiddleware_DifferentPaths(t *testing.T) {
	app := fiber.New()
	cache := NewCacheMiddleware(CacheConfig{
		TTL: 1 * time.Minute,
	})

	app.Use(cache.Middleware())
	app.Get("/path1", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"path": "1"})
	})
	app.Get("/path2", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"path": "2"})
	})

	// Request path1
	req1 := httptest.NewRequest("GET", "/path1", nil)
	resp1, err := app.Test(req1)
	require.NoError(t, err)
	defer func() { _ = resp1.Body.Close() }()
	assert.Equal(t, "MISS", resp1.Header.Get("X-Cache"))

	// Request path2
	req2 := httptest.NewRequest("GET", "/path2", nil)
	resp2, err := app.Test(req2)
	require.NoError(t, err)
	defer func() { _ = resp2.Body.Close() }()
	assert.Equal(t, "MISS", resp2.Header.Get("X-Cache"))

	// Request path1 again - should hit cache
	req3 := httptest.NewRequest("GET", "/path1", nil)
	resp3, err := app.Test(req3)
	require.NoError(t, err)
	defer func() { _ = resp3.Body.Close() }()
	assert.Equal(t, "HIT", resp3.Header.Get("X-Cache"))
}

// TestCacheMiddleware_QueryParams tests caching with different query parameters
func TestCacheMiddleware_QueryParams(t *testing.T) {
	app := fiber.New()
	cache := NewCacheMiddleware(CacheConfig{
		TTL: 1 * time.Minute,
	})

	app.Use(cache.Middleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		page := c.Query("page", "1")
		return c.JSON(fiber.Map{"page": page})
	})

	// Request with page=1
	req1 := httptest.NewRequest("GET", "/test?page=1", nil)
	resp1, err := app.Test(req1)
	require.NoError(t, err)
	defer func() { _ = resp1.Body.Close() }()
	assert.Equal(t, "MISS", resp1.Header.Get("X-Cache"))

	// Request with page=2 - different query, should miss
	req2 := httptest.NewRequest("GET", "/test?page=2", nil)
	resp2, err := app.Test(req2)
	require.NoError(t, err)
	defer func() { _ = resp2.Body.Close() }()
	assert.Equal(t, "MISS", resp2.Header.Get("X-Cache"))

	// Request with page=1 again - should hit cache
	req3 := httptest.NewRequest("GET", "/test?page=1", nil)
	resp3, err := app.Test(req3)
	require.NoError(t, err)
	defer func() { _ = resp3.Body.Close() }()
	assert.Equal(t, "HIT", resp3.Header.Get("X-Cache"))
}

// TestCacheMiddleware_Skip tests conditional skipping
func TestCacheMiddleware_Skip(t *testing.T) {
	app := fiber.New()
	cache := NewCacheMiddleware(CacheConfig{
		TTL: 1 * time.Minute,
		Skip: func(c *fiber.Ctx) bool {
			// Skip caching for admin users
			return c.Get("X-Admin") == "true"
		},
	})

	app.Use(cache.Middleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"timestamp": time.Now().UnixNano()})
	})

	// Admin request - should skip cache
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.Header.Set("X-Admin", "true")
	resp1, err := app.Test(req1)
	require.NoError(t, err)
	defer func() { _ = resp1.Body.Close() }()

	assert.Empty(t, resp1.Header.Get("X-Cache"))

	// Regular request - should use cache
	req2 := httptest.NewRequest("GET", "/test", nil)
	resp2, err := app.Test(req2)
	require.NoError(t, err)
	defer func() { _ = resp2.Body.Close() }()

	assert.Equal(t, "MISS", resp2.Header.Get("X-Cache"))
}

// TestCacheMiddleware_OnlySuccess tests caching only successful responses
func TestCacheMiddleware_OnlySuccess(t *testing.T) {
	app := fiber.New()
	cache := NewCacheMiddleware(CacheConfig{
		TTL: 1 * time.Minute,
	})

	app.Use(cache.Middleware())
	app.Get("/error", func(c *fiber.Ctx) error {
		return c.Status(500).JSON(fiber.Map{"error": "server error"})
	})

	// Error response should not be cached
	req1 := httptest.NewRequest("GET", "/error", nil)
	resp1, err := app.Test(req1)
	require.NoError(t, err)
	defer func() { _ = resp1.Body.Close() }()

	assert.Equal(t, 500, resp1.StatusCode)
	assert.Equal(t, "MISS", resp1.Header.Get("X-Cache"))

	// Second request should also miss (error not cached)
	req2 := httptest.NewRequest("GET", "/error", nil)
	resp2, err := app.Test(req2)
	require.NoError(t, err)
	defer func() { _ = resp2.Body.Close() }()

	assert.Equal(t, "MISS", resp2.Header.Get("X-Cache"))
}

// TestCacheForEndpoint tests endpoint-specific cache creation
func TestCacheForEndpoint(t *testing.T) {
	cache := CacheForEndpoint(30 * time.Second)

	assert.NotNil(t, cache)
	assert.Equal(t, 30*time.Second, cache.config.TTL)
}

// TestAnalyticsCacheMiddleware tests analytics-optimized cache
func TestAnalyticsCacheMiddleware(t *testing.T) {
	cache := AnalyticsCacheMiddleware()

	assert.NotNil(t, cache)
	assert.Equal(t, 2*time.Minute, cache.config.TTL)
}

// TestCacheStats tests cache statistics
func TestCacheStats(t *testing.T) {
	store := NewInMemoryCacheStore()
	cache := NewCacheMiddleware(CacheConfig{
		TTL:   1 * time.Minute,
		Store: store,
	})

	// Add some entries
	for i := 0; i < 3; i++ {
		entry := &CacheEntry{
			StatusCode: 200,
			Body:       []byte(`{"test":"data"}`),
			CachedAt:   time.Now(),
		}
	_ = store.Set(string(rune('a'+i)), entry, 1*time.Minute)
	}

	stats := cache.CacheStats()
	assert.Equal(t, 3, stats["entries"])
	assert.Equal(t, "1m0s", stats["ttl"])
}

// TestCacheMiddleware_Integration tests full integration scenario
func TestCacheMiddleware_Integration(t *testing.T) {
	app := fiber.New()
	cache := AnalyticsCacheMiddleware()

	app.Use(cache.Middleware())
	app.Get("/analytics/overview", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"total_requests": 12345,
			"timestamp":      time.Now().Unix(),
		})
	})

	// First request - cache miss
	req1 := httptest.NewRequest("GET", "/analytics/overview", nil)
	resp1, err := app.Test(req1)
	require.NoError(t, err)
	defer func() { _ = resp1.Body.Close() }()

	assert.Equal(t, 200, resp1.StatusCode)
	assert.Equal(t, "MISS", resp1.Header.Get("X-Cache"))

	body1, _ := io.ReadAll(resp1.Body)
	var data1 map[string]interface{}
	json.Unmarshal(body1, &data1)

	// Second request - cache hit (same data)
	req2 := httptest.NewRequest("GET", "/analytics/overview", nil)
	resp2, err := app.Test(req2)
	require.NoError(t, err)
	defer func() { _ = resp2.Body.Close() }()

	assert.Equal(t, 200, resp2.StatusCode)
	assert.Equal(t, "HIT", resp2.Header.Get("X-Cache"))

	body2, _ := io.ReadAll(resp2.Body)
	var data2 map[string]interface{}
	json.Unmarshal(body2, &data2)

	// Timestamp should be the same (cached)
	assert.Equal(t, data1["timestamp"], data2["timestamp"])

	// Verify cache stats
	stats := cache.CacheStats()
	assert.Equal(t, 1, stats["entries"])
}
