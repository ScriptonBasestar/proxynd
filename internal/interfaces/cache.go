package interfaces

import (
	"context"
	"time"
)

// CacheManager defines the interface for cache management operations
type CacheManager interface {
	// Get retrieves data from cache by key
	Get(ctx context.Context, key string) ([]byte, bool, error)
	
	// Put stores data in cache with TTL
	Put(ctx context.Context, key string, data []byte, ttl time.Duration) error
	
	// Exists checks if a key exists in cache
	Exists(ctx context.Context, key string) (bool, error)
	
	// Delete removes a key from cache
	Delete(ctx context.Context, key string) error
	
	// Clear removes all entries from cache
	Clear(ctx context.Context) error
	
	// GetStats returns cache statistics
	GetStats() CacheStats
	
	// StartEviction starts the cache eviction process
	StartEviction(ctx context.Context, interval time.Duration)
	
	// StopEviction stops the cache eviction process
	StopEviction()
}

// CacheStats represents cache statistics
type CacheStats struct {
	Hits        int64
	Misses      int64
	Evictions   int64
	Size        int64
	MaxSize     int64
	ItemCount   int64
	LastEvicted time.Time
}

// CacheBackend defines the interface for cache backend implementations
type CacheBackend interface {
	// Get retrieves value by key
	Get(key string) ([]byte, bool)
	
	// Set stores value with expiration
	Set(key string, value []byte, expiration time.Duration) error
	
	// Delete removes key
	Delete(key string) error
	
	// Clear removes all keys
	Clear() error
	
	// Size returns current cache size in bytes
	Size() int64
	
	// Keys returns all cache keys
	Keys() []string
}