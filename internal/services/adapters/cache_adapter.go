package adapters

import (
	"context"
	"io"
	"time"

	"proxynd/internal/repositories/cache"
	"proxynd/internal/services/proxy"
)

// CacheAdapter adapts the cache repository to the service interface
type CacheAdapter struct {
	repo cache.Repository
	ttl  time.Duration
}

// NewCacheAdapter creates a new cache adapter
func NewCacheAdapter(repo cache.Repository, defaultTTL time.Duration) *CacheAdapter {
	return &CacheAdapter{
		repo: repo,
		ttl:  defaultTTL,
	}
}

// Get retrieves content from cache
func (a *CacheAdapter) Get(ctx context.Context, key string) (io.ReadCloser, bool, error) {
	content, err := a.repo.Get(ctx, key)
	if err != nil {
		// Cache miss is not an error in this context
		return nil, false, nil
	}
	
	return content, true, nil
}

// Put stores content in cache
func (a *CacheAdapter) Put(ctx context.Context, key string, content io.Reader) error {
	return a.repo.Put(ctx, key, content, a.ttl)
}

// Exists checks if a key exists in cache
func (a *CacheAdapter) Exists(ctx context.Context, key string) (bool, error) {
	return a.repo.Exists(ctx, key)
}

// Delete removes content from cache
func (a *CacheAdapter) Delete(ctx context.Context, key string) error {
	return a.repo.Delete(ctx, key)
}

// Ensure CacheAdapter implements the CacheService interface
var _ proxy.CacheService = (*CacheAdapter)(nil)