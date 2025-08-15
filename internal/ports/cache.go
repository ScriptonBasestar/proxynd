package ports

import (
	"context"
	"io"
	"time"
)

// CacheBackend defines the cache storage interface
// TODO: Migrate from cache/ and internal/repositories/cache/
type CacheBackend interface {
	// Get retrieves data from cache
	Get(ctx context.Context, key string) ([]byte, error)
	
	// Set stores data in cache
	Set(ctx context.Context, key string, data []byte, ttl time.Duration) error
	
	// Delete removes data from cache
	Delete(ctx context.Context, key string) error
	
	// Exists checks if key exists in cache
	Exists(ctx context.Context, key string) (bool, error)
	
	// Clear clears all cache data
	Clear(ctx context.Context) error
	
	// GetStats returns cache statistics
	GetStats(ctx context.Context) (*CacheStats, error)
}

// StreamCacheBackend defines streaming cache interface for large files
// TODO: Migrate from cache/filesystem.go and cache/s3.go
type StreamCacheBackend interface {
	CacheBackend
	
	// GetStream retrieves data as stream
	GetStream(ctx context.Context, key string) (io.ReadCloser, error)
	
	// SetStream stores data from stream
	SetStream(ctx context.Context, key string, data io.Reader, ttl time.Duration) error
	
	// GetSize returns cached item size
	GetSize(ctx context.Context, key string) (int64, error)
}

// CacheManager manages multiple cache backends and strategies
// TODO: Migrate from cache/manager.go
type CacheManager interface {
	// Get retrieves from cache with fallback strategy
	Get(ctx context.Context, req *CacheRequest) (*CacheResponse, error)
	
	// Set stores in cache with strategy
	Set(ctx context.Context, req *CacheSetRequest) error
	
	// Invalidate removes from cache
	Invalidate(ctx context.Context, pattern string) error
	
	// GetBackend returns specific backend
	GetBackend(name string) (CacheBackend, error)
	
	// ListBackends returns available backends
	ListBackends() []string
	
	// GetStrategy returns caching strategy for type
	GetStrategy(packageType string) CacheStrategy
}

// CacheRequest represents cache retrieval request
type CacheRequest struct {
	Key         string        `json:"key"`
	PackageType string        `json:"package_type"`
	Fallback    bool          `json:"fallback"`
	TTL         time.Duration `json:"ttl"`
	Metadata    CacheMetadata `json:"metadata"`
}

// CacheResponse represents cache response
type CacheResponse struct {
	Data      []byte        `json:"-"`
	Stream    io.ReadCloser `json:"-"`
	Hit       bool          `json:"hit"`
	Backend   string        `json:"backend"`
	Size      int64         `json:"size"`
	Metadata  CacheMetadata `json:"metadata"`
	ExpiresAt time.Time     `json:"expires_at"`
}

// CacheSetRequest represents cache storage request
type CacheSetRequest struct {
	Key         string        `json:"key"`
	Data        []byte        `json:"-"`
	Stream      io.Reader     `json:"-"`
	PackageType string        `json:"package_type"`
	TTL         time.Duration `json:"ttl"`
	Metadata    CacheMetadata `json:"metadata"`
	Backend     string        `json:"backend,omitempty"`
}

// CacheMetadata represents cache metadata
type CacheMetadata struct {
	ContentType   string            `json:"content_type"`
	Size          int64             `json:"size"`
	Checksum      string            `json:"checksum"`
	Headers       map[string]string `json:"headers"`
	Source        string            `json:"source"`
	CachedAt      time.Time         `json:"cached_at"`
	LastAccessed  time.Time         `json:"last_accessed"`
	AccessCount   int64             `json:"access_count"`
	Tags          []string          `json:"tags"`
}

// CacheStats represents cache statistics
type CacheStats struct {
	Backend      string    `json:"backend"`
	Size         int64     `json:"size"`
	ItemCount    int64     `json:"item_count"`
	HitRate      float64   `json:"hit_rate"`
	HitCount     int64     `json:"hit_count"`
	MissCount    int64     `json:"miss_count"`
	LastEviction time.Time `json:"last_eviction"`
	FreeSpace    int64     `json:"free_space"`
}

// CacheStrategy defines caching behavior per package type
// TODO: Migrate from internal/services cache strategies
type CacheStrategy interface {
	// ShouldCache determines if item should be cached
	ShouldCache(req *CacheRequest) bool
	
	// GetTTL returns TTL for item
	GetTTL(req *CacheRequest) time.Duration
	
	// GetBackend returns preferred backend
	GetBackend(req *CacheRequest) string
	
	// GenerateKey generates cache key
	GenerateKey(req *CacheRequest) string
	
	// GetEvictionPolicy returns eviction policy
	GetEvictionPolicy() EvictionPolicy
}

// EvictionPolicy defines cache eviction behavior
type EvictionPolicy interface {
	// ShouldEvict determines if item should be evicted
	ShouldEvict(metadata CacheMetadata, stats CacheStats) bool
	
	// GetPriority returns eviction priority (higher = evict first)
	GetPriority(metadata CacheMetadata) int
}

// CacheKeyBuilder builds cache keys
// TODO: Migrate from domain/common/cache_key_builder.go
type CacheKeyBuilder interface {
	// BuildKey builds cache key for request
	BuildKey(packageType, repository, name, version, path string) string
	
	// ParseKey parses cache key components
	ParseKey(key string) (*KeyComponents, error)
	
	// BuildPattern builds key pattern for invalidation
	BuildPattern(packageType, repository string) string
}

// KeyComponents represents parsed cache key
type KeyComponents struct {
	PackageType string `json:"package_type"`
	Repository  string `json:"repository"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Path        string `json:"path"`
}

// CacheEviction handles cache eviction logic
// TODO: Migrate from cache/eviction.go
type CacheEviction interface {
	// StartEviction starts eviction process
	StartEviction(ctx context.Context) error
	
	// StopEviction stops eviction process
	StopEviction(ctx context.Context) error
	
	// RunEviction runs single eviction cycle
	RunEviction(ctx context.Context) (*EvictionResult, error)
	
	// GetEvictionStats returns eviction statistics
	GetEvictionStats() *EvictionStats
}

// EvictionResult represents eviction operation result
type EvictionResult struct {
	ItemsEvicted int64         `json:"items_evicted"`
	SpaceFreed   int64         `json:"space_freed"`
	Duration     time.Duration `json:"duration"`
	Errors       []string      `json:"errors"`
}

// EvictionStats represents eviction statistics
type EvictionStats struct {
	TotalEvictions  int64     `json:"total_evictions"`
	TotalSpaceFreed int64     `json:"total_space_freed"`
	LastEviction    time.Time `json:"last_eviction"`
	EvictionRate    float64   `json:"eviction_rate"`
	ErrorCount      int64     `json:"error_count"`
}

// CacheHealth monitors cache health
type CacheHealth interface {
	// CheckHealth checks cache backend health
	CheckHealth(ctx context.Context, backend string) (*HealthStatus, error)
	
	// GetOverallHealth returns overall cache health
	GetOverallHealth(ctx context.Context) (*OverallHealth, error)
}

// HealthStatus represents cache backend health
type HealthStatus struct {
	Backend   string    `json:"backend"`
	Status    string    `json:"status"` // healthy, degraded, unhealthy
	Latency   int64     `json:"latency_ms"`
	Available bool      `json:"available"`
	Message   string    `json:"message"`
	CheckedAt time.Time `json:"checked_at"`
}

// OverallHealth represents overall cache health
type OverallHealth struct {
	Status    string          `json:"status"`
	Backends  []*HealthStatus `json:"backends"`
	CheckedAt time.Time       `json:"checked_at"`
}