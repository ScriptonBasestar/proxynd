package usecase

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"proxynd/internal/ports"
)

func TestReadThroughStrategy(t *testing.T) {
	strategy := NewReadThroughStrategy()

	// Test ShouldCache - should always return true
	req := &ports.CacheRequest{
		Key:         "test-key",
		PackageType: "maven",
	}
	assert.True(t, strategy.ShouldCache(req))

	// Test GetTTL
	ttl := strategy.GetTTL(req)
	assert.Equal(t, time.Hour*24, ttl)

	// Test GetBackend
	backend := strategy.GetBackend(req)
	assert.Equal(t, "filesystem", backend)

	// Test GenerateKey
	key := strategy.GenerateKey(req)
	assert.Equal(t, "test-key", key)

	// Test GetEvictionPolicy
	policy := strategy.GetEvictionPolicy()
	assert.NotNil(t, policy)
}

func TestWriteThroughStrategy(t *testing.T) {
	ttl := time.Hour * 48
	backend := "s3"
	strategy := NewWriteThroughStrategy(ttl, backend)

	// Test ShouldCache - should always return true
	req := &ports.CacheRequest{
		Key:         "test-key",
		PackageType: "docker",
	}
	assert.True(t, strategy.ShouldCache(req))

	// Test GetTTL
	resultTTL := strategy.GetTTL(req)
	assert.Equal(t, ttl, resultTTL)

	// Test GetBackend
	resultBackend := strategy.GetBackend(req)
	assert.Equal(t, backend, resultBackend)

	// Test GenerateKey
	key := strategy.GenerateKey(req)
	assert.Equal(t, "test-key", key)

	// Test GetEvictionPolicy
	policy := strategy.GetEvictionPolicy()
	assert.NotNil(t, policy)
}

func TestStaleWhileRevalidateStrategy(t *testing.T) {
	ttl := time.Hour * 12
	staleWindow := time.Hour * 2
	strategy := NewStaleWhileRevalidateStrategy(ttl, staleWindow)

	// Test ShouldCache
	req := &ports.CacheRequest{
		Key:         "test-key",
		PackageType: "npm",
	}
	assert.True(t, strategy.ShouldCache(req))

	// Test GetTTL
	resultTTL := strategy.GetTTL(req)
	assert.Equal(t, ttl, resultTTL)

	// Test GetBackend
	backend := strategy.GetBackend(req)
	assert.Equal(t, "filesystem", backend)

	// Test GenerateKey
	key := strategy.GenerateKey(req)
	assert.Equal(t, "test-key", key)

	// Test GetEvictionPolicy
	policy := strategy.GetEvictionPolicy()
	assert.NotNil(t, policy)

	// Test GetStaleWindow
	resultStaleWindow := strategy.GetStaleWindow()
	assert.Equal(t, staleWindow, resultStaleWindow)
}

func TestLRUEvictionPolicy(t *testing.T) {
	policy := NewLRUEvictionPolicy()

	// Test ShouldEvict - cache over capacity
	stats := ports.CacheStats{
		Size: 2 * 1024 * 1024 * 1024, // 2GB, over 1GB limit
	}
	metadata := ports.CacheMetadata{
		LastAccessed: time.Now().Add(-time.Hour), // Recent access
	}
	assert.True(t, policy.ShouldEvict(metadata, stats))

	// Test ShouldEvict - old item
	stats = ports.CacheStats{
		Size: 512 * 1024 * 1024, // 512MB, under limit
	}
	metadata = ports.CacheMetadata{
		LastAccessed: time.Now().Add(-time.Hour * 24 * 8), // 8 days old
	}
	assert.True(t, policy.ShouldEvict(metadata, stats))

	// Test ShouldEvict - recent and under capacity
	metadata = ports.CacheMetadata{
		LastAccessed: time.Now().Add(-time.Hour), // Recent
	}
	assert.False(t, policy.ShouldEvict(metadata, stats))

	// Test GetPriority - old item with low access count
	metadata = ports.CacheMetadata{
		LastAccessed: time.Now().Add(-time.Hour * 24 * 3), // 3 days old
		AccessCount:  2,                                    // Low access
	}
	priority := policy.GetPriority(metadata)
	assert.Greater(t, priority, 0)
	assert.Greater(t, priority, 50) // Should have high priority due to low access

	// Test GetPriority - recent item with high access count
	metadata = ports.CacheMetadata{
		LastAccessed: time.Now().Add(-time.Hour * 2), // Recent
		AccessCount:  100,                             // High access
	}
	priority = policy.GetPriority(metadata)
	assert.Greater(t, priority, 0)
	assert.Less(t, priority, 50) // Should have lower priority
}

func TestTTLEvictionPolicy(t *testing.T) {
	ttl := time.Hour * 24
	policy := NewTTLEvictionPolicy(ttl)

	// Test ShouldEvict - expired item
	metadata := ports.CacheMetadata{
		CachedAt: time.Now().Add(-time.Hour * 25), // Cached 25 hours ago, TTL is 24h
	}
	stats := ports.CacheStats{}
	assert.True(t, policy.ShouldEvict(metadata, stats))

	// Test ShouldEvict - not expired
	metadata = ports.CacheMetadata{
		CachedAt: time.Now().Add(-time.Hour * 12), // Cached 12 hours ago, TTL is 24h
	}
	assert.False(t, policy.ShouldEvict(metadata, stats))

	// Test GetPriority - expired item
	metadata = ports.CacheMetadata{
		CachedAt: time.Now().Add(-time.Hour * 25), // Expired
	}
	priority := policy.GetPriority(metadata)
	assert.Equal(t, 1000, priority) // Highest priority

	// Test GetPriority - expires soon
	metadata = ports.CacheMetadata{
		CachedAt: time.Now().Add(-time.Hour * 23).Add(-time.Minute * 30), // Expires in 30 minutes
	}
	priority = policy.GetPriority(metadata)
	assert.Equal(t, 100, priority)

	// Test GetPriority - expires in a few hours
	metadata = ports.CacheMetadata{
		CachedAt: time.Now().Add(-time.Hour * 20), // Expires in 4 hours
	}
	priority = policy.GetPriority(metadata)
	assert.Equal(t, 50, priority)

	// Test GetPriority - fresh item
	metadata = ports.CacheMetadata{
		CachedAt: time.Now().Add(-time.Hour * 2), // Expires in 22 hours
	}
	priority = policy.GetPriority(metadata)
	assert.Equal(t, 10, priority) // Low priority
}

func TestEvictionPolicyIntegration(t *testing.T) {
	// Test that strategies use eviction policies correctly
	lruStrategy := NewReadThroughStrategy()
	policy := lruStrategy.GetEvictionPolicy()
	
	// Should be LRU policy
	assert.IsType(t, &LRUEvictionPolicy{}, policy)
	
	// Test with sample data
	metadata := ports.CacheMetadata{
		LastAccessed: time.Now().Add(-time.Hour * 24 * 10), // 10 days old
		AccessCount:  1,                                     // Rarely accessed
	}
	stats := ports.CacheStats{
		Size: 512 * 1024 * 1024, // Under capacity
	}
	
	assert.True(t, policy.ShouldEvict(metadata, stats))
	priority := policy.GetPriority(metadata)
	assert.Greater(t, priority, 150) // High priority due to age and low access
}

func TestStrategyDefaults(t *testing.T) {
	// Test that all strategies have sensible defaults
	strategies := []ports.CacheStrategy{
		NewReadThroughStrategy(),
		NewWriteThroughStrategy(time.Hour*24, "filesystem"),
		NewStaleWhileRevalidateStrategy(time.Hour*12, time.Hour*1),
	}

	req := &ports.CacheRequest{
		Key:         "test-key",
		PackageType: "test",
	}

	for i, strategy := range strategies {
		t.Run(fmt.Sprintf("strategy_%d", i), func(t *testing.T) {
			// All strategies should cache by default
			assert.True(t, strategy.ShouldCache(req))
			
			// All should have positive TTL
			ttl := strategy.GetTTL(req)
			assert.Greater(t, ttl, time.Duration(0))
			
			// All should have a backend
			backend := strategy.GetBackend(req)
			assert.NotEmpty(t, backend)
			
			// All should generate keys
			key := strategy.GenerateKey(req)
			assert.Equal(t, "test-key", key)
			
			// All should have eviction policies
			policy := strategy.GetEvictionPolicy()
			assert.NotNil(t, policy)
		})
	}
}