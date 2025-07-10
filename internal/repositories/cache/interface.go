package cache

//go:generate mockery --name Repository --output ./mocks --outpkg mocks --filename repository.go

import (
	"context"
	"io"
	"time"
)

// Repository defines the interface for cache storage operations
type Repository interface {
	// Get retrieves an item from the cache
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// Put stores an item in the cache
	Put(ctx context.Context, key string, content io.Reader, ttl time.Duration) error

	// Exists checks if a key exists in the cache
	Exists(ctx context.Context, key string) (bool, error)

	// Delete removes an item from the cache
	Delete(ctx context.Context, key string) error

	// List returns all cache keys matching a pattern
	List(ctx context.Context, pattern string) ([]string, error)

	// Size returns the size of a cached item in bytes
	Size(ctx context.Context, key string) (int64, error)

	// Clear removes all items from the cache
	Clear(ctx context.Context) error

	// Stats returns cache statistics
	Stats(ctx context.Context) (*CacheStats, error)
}

// CacheStats represents cache statistics
type CacheStats struct {
	TotalItems   int64
	TotalSize    int64
	OldestItem   time.Time
	NewestItem   time.Time
	HitCount     int64
	MissCount    int64
	EvictedCount int64
}

// CacheItem represents metadata about a cached item
type CacheItem struct {
	Key        string
	Size       int64
	ModifiedAt time.Time
	AccessedAt time.Time
	TTL        time.Duration
}
