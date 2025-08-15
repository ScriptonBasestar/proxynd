package usecase

import (
	"time"
	"proxynd/internal/ports"
)

// ReadThroughStrategy implements a read-through caching strategy
// Cache is populated on cache miss during read operations
type ReadThroughStrategy struct {
	ttl            time.Duration
	backend        string
	evictionPolicy ports.EvictionPolicy
}

// NewReadThroughStrategy creates a new read-through strategy
func NewReadThroughStrategy() *ReadThroughStrategy {
	return &ReadThroughStrategy{
		ttl:            time.Hour * 24,
		backend:        "filesystem",
		evictionPolicy: NewLRUEvictionPolicy(),
	}
}

// ShouldCache determines if item should be cached (always true for read-through)
func (rts *ReadThroughStrategy) ShouldCache(req *ports.CacheRequest) bool {
	// Always cache in read-through strategy
	return true
}

// GetTTL returns TTL for item
func (rts *ReadThroughStrategy) GetTTL(req *ports.CacheRequest) time.Duration {
	return rts.ttl
}

// GetBackend returns preferred backend
func (rts *ReadThroughStrategy) GetBackend(req *ports.CacheRequest) string {
	return rts.backend
}

// GenerateKey generates cache key
func (rts *ReadThroughStrategy) GenerateKey(req *ports.CacheRequest) string {
	return req.Key
}

// GetEvictionPolicy returns eviction policy
func (rts *ReadThroughStrategy) GetEvictionPolicy() ports.EvictionPolicy {
	return rts.evictionPolicy
}

// WriteThroughStrategy implements a write-through caching strategy
// Cache is populated immediately when data is written/updated
type WriteThroughStrategy struct {
	ttl            time.Duration
	backend        string
	evictionPolicy ports.EvictionPolicy
}

// NewWriteThroughStrategy creates a new write-through strategy
func NewWriteThroughStrategy(ttl time.Duration, backend string) *WriteThroughStrategy {
	return &WriteThroughStrategy{
		ttl:            ttl,
		backend:        backend,
		evictionPolicy: NewLRUEvictionPolicy(),
	}
}

// ShouldCache determines if item should be cached (always true for write-through)
func (wts *WriteThroughStrategy) ShouldCache(req *ports.CacheRequest) bool {
	return true
}

// GetTTL returns TTL for item
func (wts *WriteThroughStrategy) GetTTL(req *ports.CacheRequest) time.Duration {
	return wts.ttl
}

// GetBackend returns preferred backend
func (wts *WriteThroughStrategy) GetBackend(req *ports.CacheRequest) string {
	return wts.backend
}

// GenerateKey generates cache key
func (wts *WriteThroughStrategy) GenerateKey(req *ports.CacheRequest) string {
	return req.Key
}

// GetEvictionPolicy returns eviction policy
func (wts *WriteThroughStrategy) GetEvictionPolicy() ports.EvictionPolicy {
	return wts.evictionPolicy
}

// StaleWhileRevalidateStrategy implements stale-while-revalidate caching
// Serves stale content while asynchronously updating cache
type StaleWhileRevalidateStrategy struct {
	ttl                  time.Duration
	staleWindow          time.Duration
	backend              string
	evictionPolicy       ports.EvictionPolicy
}

// NewStaleWhileRevalidateStrategy creates a new stale-while-revalidate strategy
func NewStaleWhileRevalidateStrategy(ttl, staleWindow time.Duration) *StaleWhileRevalidateStrategy {
	return &StaleWhileRevalidateStrategy{
		ttl:            ttl,
		staleWindow:    staleWindow,
		backend:        "filesystem",
		evictionPolicy: NewLRUEvictionPolicy(),
	}
}

// ShouldCache determines if item should be cached
func (swrs *StaleWhileRevalidateStrategy) ShouldCache(req *ports.CacheRequest) bool {
	return true
}

// GetTTL returns TTL for item
func (swrs *StaleWhileRevalidateStrategy) GetTTL(req *ports.CacheRequest) time.Duration {
	return swrs.ttl
}

// GetBackend returns preferred backend
func (swrs *StaleWhileRevalidateStrategy) GetBackend(req *ports.CacheRequest) string {
	return swrs.backend
}

// GenerateKey generates cache key
func (swrs *StaleWhileRevalidateStrategy) GenerateKey(req *ports.CacheRequest) string {
	return req.Key
}

// GetEvictionPolicy returns eviction policy
func (swrs *StaleWhileRevalidateStrategy) GetEvictionPolicy() ports.EvictionPolicy {
	return swrs.evictionPolicy
}

// GetStaleWindow returns the stale window duration
func (swrs *StaleWhileRevalidateStrategy) GetStaleWindow() time.Duration {
	return swrs.staleWindow
}

// LRUEvictionPolicy implements Least Recently Used eviction policy
type LRUEvictionPolicy struct {
	maxSize int64
}

// NewLRUEvictionPolicy creates a new LRU eviction policy
func NewLRUEvictionPolicy() *LRUEvictionPolicy {
	return &LRUEvictionPolicy{
		maxSize: 1024 * 1024 * 1024, // 1GB default
	}
}

// ShouldEvict determines if item should be evicted
func (lru *LRUEvictionPolicy) ShouldEvict(metadata ports.CacheMetadata, stats ports.CacheStats) bool {
	// Evict if cache is over capacity
	if stats.Size > lru.maxSize {
		return true
	}
	
	// Evict old items that haven't been accessed recently
	if time.Since(metadata.LastAccessed) > time.Hour*24*7 { // 7 days
		return true
	}
	
	return false
}

// GetPriority returns eviction priority (higher = evict first)
func (lru *LRUEvictionPolicy) GetPriority(metadata ports.CacheMetadata) int {
	// Priority based on last access time and access count
	daysSinceAccess := int(time.Since(metadata.LastAccessed).Hours() / 24)
	
	// Higher priority (more likely to evict) for:
	// - Items accessed long ago
	// - Items with low access count
	priority := daysSinceAccess * 10
	
	if metadata.AccessCount < 5 {
		priority += 50 // Boost priority for rarely accessed items
	}
	
	return priority
}

// TTLEvictionPolicy implements Time-To-Live eviction policy
type TTLEvictionPolicy struct {
	defaultTTL time.Duration
}

// NewTTLEvictionPolicy creates a new TTL eviction policy
func NewTTLEvictionPolicy(defaultTTL time.Duration) *TTLEvictionPolicy {
	return &TTLEvictionPolicy{
		defaultTTL: defaultTTL,
	}
}

// ShouldEvict determines if item should be evicted based on TTL
func (ttl *TTLEvictionPolicy) ShouldEvict(metadata ports.CacheMetadata, stats ports.CacheStats) bool {
	// Calculate expiry time
	expiryTime := metadata.CachedAt.Add(ttl.defaultTTL)
	return time.Now().After(expiryTime)
}

// GetPriority returns eviction priority (higher = evict first)
func (ttl *TTLEvictionPolicy) GetPriority(metadata ports.CacheMetadata) int {
	expiryTime := metadata.CachedAt.Add(ttl.defaultTTL)
	
	if time.Now().After(expiryTime) {
		// Expired items get highest priority
		return 1000
	}
	
	// Priority based on how close to expiry
	timeToExpiry := time.Until(expiryTime)
	if timeToExpiry < time.Hour {
		return 100
	} else if timeToExpiry < time.Hour*6 {
		return 50
	}
	
	return 10
}