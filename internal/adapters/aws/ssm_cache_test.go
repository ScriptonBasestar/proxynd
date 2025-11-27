package aws

import (
	"context"
	"testing"
	"time"
)

// MockAWSClient is a mock implementation of AWSClient for testing
type MockAWSClient struct {
	getParameterFunc  func(ctx context.Context, name string, withDecryption bool) (string, error)
	getParametersFunc func(ctx context.Context, names []string, withDecryption bool) (map[string]string, error)
	getByPathFunc     func(ctx context.Context, path string, withDecryption bool, recursive bool) (map[string]string, error)
}

func (m *MockAWSClient) GetParameter(ctx context.Context, name string, withDecryption bool) (string, error) {
	if m.getParameterFunc != nil {
		return m.getParameterFunc(ctx, name, withDecryption)
	}
	return "", nil
}

func (m *MockAWSClient) GetParameters(ctx context.Context, names []string, withDecryption bool) (map[string]string, error) {
	if m.getParametersFunc != nil {
		return m.getParametersFunc(ctx, names, withDecryption)
	}
	return make(map[string]string), nil
}

func (m *MockAWSClient) GetParametersByPath(ctx context.Context, path string, withDecryption bool, recursive bool) (map[string]string, error) {
	if m.getByPathFunc != nil {
		return m.getByPathFunc(ctx, path, withDecryption, recursive)
	}
	return make(map[string]string), nil
}

func TestSSMCache_GetParameter(t *testing.T) {
	// Create mock client that returns a known value
	mockClient := &AWSClient{}
	cache := NewSSMCache(mockClient, 1*time.Second)

	// Test that cache misses fetch from SSM
	// Note: This test requires a real SSM client or more sophisticated mocking
	// For now, we'll test the cache mechanisms

	t.Run("cache stores and retrieves values", func(t *testing.T) {
		// Manually populate cache for testing
		cache.setCached("test-param", "test-value", false)

		cached := cache.getCached("test-param")
		if cached == nil {
			t.Fatal("Expected cached value, got nil")
		}

		if cached.Value != "test-value" {
			t.Errorf("Expected 'test-value', got '%s'", cached.Value)
		}
	})

	t.Run("cache expires after TTL", func(t *testing.T) {
		shortCache := NewSSMCache(mockClient, 10*time.Millisecond)
		shortCache.setCached("expire-test", "value", false)

		// Should be in cache immediately
		if cached := shortCache.getCached("expire-test"); cached == nil {
			t.Error("Expected value in cache immediately")
		}

		// Wait for expiration
		time.Sleep(20 * time.Millisecond)

		// Should be expired now
		if cached := shortCache.getCached("expire-test"); cached != nil {
			t.Error("Expected cache to expire, but value still present")
		}
	})

	t.Run("invalidate removes from cache", func(t *testing.T) {
		cache := NewSSMCache(mockClient, 1*time.Hour)
		cache.setCached("invalidate-test", "value", false)

		if cached := cache.getCached("invalidate-test"); cached == nil {
			t.Error("Expected value in cache")
		}

		cache.Invalidate("invalidate-test")

		if cached := cache.getCached("invalidate-test"); cached != nil {
			t.Error("Expected value to be invalidated")
		}
	})

	t.Run("invalidate all clears cache", func(t *testing.T) {
		cache := NewSSMCache(mockClient, 1*time.Hour)
		cache.setCached("param1", "value1", false)
		cache.setCached("param2", "value2", false)
		cache.setCached("param3", "value3", false)

		stats := cache.GetStats()
		if stats["total_entries"].(int) != 3 {
			t.Errorf("Expected 3 entries, got %d", stats["total_entries"])
		}

		cache.InvalidateAll()

		stats = cache.GetStats()
		if stats["total_entries"].(int) != 0 {
			t.Errorf("Expected 0 entries after InvalidateAll, got %d", stats["total_entries"])
		}
	})
}

func TestSSMCache_CleanupExpired(t *testing.T) {
	mockClient := &AWSClient{}
	cache := NewSSMCache(mockClient, 10*time.Millisecond)

	// Add some entries
	cache.setCached("param1", "value1", false)
	cache.setCached("param2", "value2", false)

	stats := cache.GetStats()
	if stats["total_entries"].(int) != 2 {
		t.Errorf("Expected 2 entries, got %d", stats["total_entries"])
	}

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Cleanup
	cache.CleanupExpired()

	stats = cache.GetStats()
	if stats["total_entries"].(int) != 0 {
		t.Errorf("Expected 0 entries after cleanup, got %d", stats["total_entries"])
	}
}

func TestSSMCache_GetStats(t *testing.T) {
	mockClient := &AWSClient{}
	cache := NewSSMCache(mockClient, 1*time.Hour)

	// Add entries with different expiration
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

	if stats["total_entries"].(int) != 3 {
		t.Errorf("Expected 3 total entries, got %d", stats["total_entries"])
	}

	if stats["expired_entries"].(int) != 1 {
		t.Errorf("Expected 1 expired entry, got %d", stats["expired_entries"])
	}

	if stats["valid_entries"].(int) != 2 {
		t.Errorf("Expected 2 valid entries, got %d", stats["valid_entries"])
	}
}

func TestSSMCache_GetParameters(t *testing.T) {
	// This test requires mocking the actual AWS client
	// For comprehensive testing, you would use AWS SDK mocking tools
	// Here we test the cache logic

	mockClient := &AWSClient{}
	cache := NewSSMCache(mockClient, 1*time.Hour)

	// Prepopulate cache
	cache.setCached("cached-param", "cached-value", false)

	t.Run("retrieves cached parameters", func(t *testing.T) {
		// This would need a proper mock to test fully
		// For now, just verify the cached parameter is accessible
		cached := cache.getCached("cached-param")
		if cached == nil || cached.Value != "cached-value" {
			t.Error("Failed to retrieve cached parameter")
		}
	})
}
