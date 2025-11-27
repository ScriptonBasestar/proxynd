package aws

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSSMCache(t *testing.T) {
	t.Run("creates cache with custom TTL", func(t *testing.T) {
		client := &AWSClient{}
		cache := NewSSMCache(client, 10*time.Minute)

		assert.NotNil(t, cache)
		assert.Equal(t, 10*time.Minute, cache.defaultTTL)
		assert.NotNil(t, cache.cache)
	})

	t.Run("uses default TTL when zero provided", func(t *testing.T) {
		client := &AWSClient{}
		cache := NewSSMCache(client, 0)

		assert.NotNil(t, cache)
		assert.Equal(t, 5*time.Minute, cache.defaultTTL)
	})
}

func TestSSMCache_GetParameter(t *testing.T) {
	t.Run("cache stores and retrieves values", func(t *testing.T) {
		client := &AWSClient{}
		cache := NewSSMCache(client, 1*time.Hour)

		// Manually populate cache for testing
		cache.setCached("test-param", "test-value", false)

		cached := cache.getCached("test-param")
		require.NotNil(t, cached, "Expected cached value, got nil")
		assert.Equal(t, "test-value", cached.Value)
		assert.False(t, cached.Encrypted)
	})

	t.Run("cache expires after TTL", func(t *testing.T) {
		client := &AWSClient{}
		shortCache := NewSSMCache(client, 10*time.Millisecond)
		shortCache.setCached("expire-test", "value", false)

		// Should be in cache immediately
		cached := shortCache.getCached("expire-test")
		assert.NotNil(t, cached, "Expected value in cache immediately")

		// Wait for expiration
		time.Sleep(20 * time.Millisecond)

		// Should be expired now
		cached = shortCache.getCached("expire-test")
		assert.Nil(t, cached, "Expected cache to expire")
	})

	t.Run("returns nil for non-existent parameter", func(t *testing.T) {
		client := &AWSClient{}
		cache := NewSSMCache(client, 1*time.Hour)

		cached := cache.getCached("non-existent")
		assert.Nil(t, cached)
	})
}

func TestSSMCache_GetParameters(t *testing.T) {
	t.Run("returns cached parameters when available", func(t *testing.T) {
		client := &AWSClient{}
		cache := NewSSMCache(client, 1*time.Hour)

		// Prepopulate cache
		cache.setCached("param1", "value1", false)
		cache.setCached("param2", "value2", false)

		// Request the cached parameters
		// Note: This test is simplified. In reality, GetParameters would call the client for missing params
		cached1 := cache.getCached("param1")
		cached2 := cache.getCached("param2")

		assert.NotNil(t, cached1)
		assert.NotNil(t, cached2)
		assert.Equal(t, "value1", cached1.Value)
		assert.Equal(t, "value2", cached2.Value)
	})

	t.Run("handles empty parameter list", func(t *testing.T) {
		client := &AWSClient{}
		cache := NewSSMCache(client, 1*time.Hour)

		ctx := context.Background()
		result, err := cache.GetParameters(ctx, []string{}, false)

		assert.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestSSMCache_Invalidate(t *testing.T) {
	t.Run("invalidate removes from cache", func(t *testing.T) {
		client := &AWSClient{}
		cache := NewSSMCache(client, 1*time.Hour)
		cache.setCached("invalidate-test", "value", false)

		cached := cache.getCached("invalidate-test")
		assert.NotNil(t, cached, "Expected value in cache")

		cache.Invalidate("invalidate-test")

		cached = cache.getCached("invalidate-test")
		assert.Nil(t, cached, "Expected value to be invalidated")
	})

	t.Run("invalidate non-existent parameter does not error", func(t *testing.T) {
		client := &AWSClient{}
		cache := NewSSMCache(client, 1*time.Hour)

		// Should not panic or error
		cache.Invalidate("non-existent")
	})
}

func TestSSMCache_InvalidateAll(t *testing.T) {
	t.Run("invalidate all clears entire cache", func(t *testing.T) {
		client := &AWSClient{}
		cache := NewSSMCache(client, 1*time.Hour)
		cache.setCached("param1", "value1", false)
		cache.setCached("param2", "value2", false)
		cache.setCached("param3", "value3", false)

		stats := cache.GetStats()
		assert.Equal(t, 3, stats["total_entries"])

		cache.InvalidateAll()

		stats = cache.GetStats()
		assert.Equal(t, 0, stats["total_entries"])
	})

	t.Run("invalidate all on empty cache", func(t *testing.T) {
		client := &AWSClient{}
		cache := NewSSMCache(client, 1*time.Hour)

		// Should not panic
		cache.InvalidateAll()

		stats := cache.GetStats()
		assert.Equal(t, 0, stats["total_entries"])
	})
}

func TestSSMCache_CleanupExpired(t *testing.T) {
	t.Run("removes expired entries", func(t *testing.T) {
		client := &AWSClient{}
		cache := NewSSMCache(client, 10*time.Millisecond)

		// Add some entries
		cache.setCached("param1", "value1", false)
		cache.setCached("param2", "value2", false)

		stats := cache.GetStats()
		assert.Equal(t, 2, stats["total_entries"])

		// Wait for expiration
		time.Sleep(20 * time.Millisecond)

		// Cleanup
		cache.CleanupExpired()

		stats = cache.GetStats()
		assert.Equal(t, 0, stats["total_entries"])
	})

	t.Run("keeps valid entries", func(t *testing.T) {
		client := &AWSClient{}
		cache := NewSSMCache(client, 1*time.Hour)

		cache.setCached("long-lived", "value", false)

		// Manually add expired entry
		cache.mutex.Lock()
		cache.cache["expired"] = &CachedParameter{
			Value:      "expired-value",
			ExpiresAt:  time.Now().Add(-1 * time.Hour),
			Encrypted:  false,
			LastUpdate: time.Now(),
		}
		cache.mutex.Unlock()

		stats := cache.GetStats()
		assert.Equal(t, 2, stats["total_entries"])

		cache.CleanupExpired()

		stats = cache.GetStats()
		assert.Equal(t, 1, stats["total_entries"])

		// Verify valid entry still exists
		cached := cache.getCached("long-lived")
		assert.NotNil(t, cached)
		assert.Equal(t, "value", cached.Value)
	})
}

func TestSSMCache_GetStats(t *testing.T) {
	t.Run("returns accurate statistics", func(t *testing.T) {
		client := &AWSClient{}
		cache := NewSSMCache(client, 1*time.Hour)

		// Add valid entries
		cache.setCached("valid1", "value1", false)
		cache.setCached("valid2", "value2", false)

		// Manually add an expired entry
		cache.mutex.Lock()
		cache.cache["expired"] = &CachedParameter{
			Value:      "expired-value",
			ExpiresAt:  time.Now().Add(-1 * time.Hour), // Already expired
			Encrypted:  false,
			LastUpdate: time.Now(),
		}
		cache.mutex.Unlock()

		stats := cache.GetStats()

		assert.Equal(t, 3, stats["total_entries"])
		assert.Equal(t, 1, stats["expired_entries"])
		assert.Equal(t, 2, stats["valid_entries"])
		assert.Equal(t, "1h0m0s", stats["default_ttl"])
	})

	t.Run("empty cache stats", func(t *testing.T) {
		client := &AWSClient{}
		cache := NewSSMCache(client, 5*time.Minute)

		stats := cache.GetStats()

		assert.Equal(t, 0, stats["total_entries"])
		assert.Equal(t, 0, stats["expired_entries"])
		assert.Equal(t, 0, stats["valid_entries"])
		assert.Equal(t, "5m0s", stats["default_ttl"])
	})
}

func TestSSMCache_StartAutoCleanup(t *testing.T) {
	t.Run("auto cleanup removes expired entries", func(t *testing.T) {
		client := &AWSClient{}
		cache := NewSSMCache(client, 20*time.Millisecond)

		// Add entries
		cache.setCached("param1", "value1", false)
		cache.setCached("param2", "value2", false)

		// Start auto cleanup with short interval
		cache.StartAutoCleanup(50 * time.Millisecond)

		// Wait for entries to expire and cleanup to run
		time.Sleep(100 * time.Millisecond)

		stats := cache.GetStats()
		assert.Equal(t, 0, stats["total_entries"], "Auto cleanup should have removed expired entries")
	})

	t.Run("uses default interval when zero provided", func(t *testing.T) {
		client := &AWSClient{}
		cache := NewSSMCache(client, 1*time.Hour)

		// Should not panic with zero interval (uses default)
		cache.StartAutoCleanup(0)

		// Cleanup goroutine should be running
		time.Sleep(10 * time.Millisecond)
	})
}

func TestSSMCache_SetCached(t *testing.T) {
	t.Run("stores parameter with correct expiration", func(t *testing.T) {
		client := &AWSClient{}
		cache := NewSSMCache(client, 1*time.Hour)

		before := time.Now()
		cache.setCached("test", "value", true)
		after := time.Now()

		cached := cache.getCached("test")
		require.NotNil(t, cached)

		assert.Equal(t, "value", cached.Value)
		assert.True(t, cached.Encrypted)
		assert.True(t, cached.ExpiresAt.After(before.Add(1*time.Hour)))
		assert.True(t, cached.ExpiresAt.Before(after.Add(1*time.Hour+time.Second)))
		assert.True(t, cached.LastUpdate.After(before))
		assert.True(t, cached.LastUpdate.Before(after.Add(time.Second)))
	})
}

func TestSSMCache_Concurrency(t *testing.T) {
	t.Run("concurrent reads and writes are safe", func(t *testing.T) {
		client := &AWSClient{}
		cache := NewSSMCache(client, 1*time.Hour)

		// Pre-populate
		cache.setCached("param1", "value1", false)

		done := make(chan bool)

		// Concurrent readers
		for i := 0; i < 10; i++ {
			go func() {
				for j := 0; j < 100; j++ {
					_ = cache.getCached("param1")
				}
				done <- true
			}()
		}

		// Concurrent writers
		for i := 0; i < 10; i++ {
			go func(id int) {
				for j := 0; j < 100; j++ {
					cache.setCached("param1", "value", false)
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 20; i++ {
			<-done
		}

		// Verify cache is still functional
		cached := cache.getCached("param1")
		assert.NotNil(t, cached)
	})
}

// Benchmark tests
func BenchmarkSSMCache_GetCached(b *testing.B) {
	client := &AWSClient{}
	cache := NewSSMCache(client, 1*time.Hour)
	cache.setCached("bench-param", "bench-value", false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cache.getCached("bench-param")
	}
}

func BenchmarkSSMCache_SetCached(b *testing.B) {
	client := &AWSClient{}
	cache := NewSSMCache(client, 1*time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.setCached("bench-param", "bench-value", false)
	}
}

func BenchmarkSSMCache_ConcurrentReads(b *testing.B) {
	client := &AWSClient{}
	cache := NewSSMCache(client, 1*time.Hour)
	cache.setCached("bench-param", "bench-value", false)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = cache.getCached("bench-param")
		}
	})
}
