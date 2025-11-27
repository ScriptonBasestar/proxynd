package aws

import (
	"context"
	"sync"
	"time"
)

// CachedParameter represents a cached SSM parameter with expiration
type CachedParameter struct {
	Value      string
	ExpiresAt  time.Time
	Encrypted  bool
	LastUpdate time.Time
}

// SSMCache provides caching for SSM parameters to reduce API calls
type SSMCache struct {
	client     *AWSClient
	cache      map[string]*CachedParameter
	mutex      sync.RWMutex
	defaultTTL time.Duration
}

// NewSSMCache creates a new SSM parameter cache
func NewSSMCache(client *AWSClient, ttl time.Duration) *SSMCache {
	if ttl == 0 {
		ttl = 5 * time.Minute // Default 5 minutes
	}

	return &SSMCache{
		client:     client,
		cache:      make(map[string]*CachedParameter),
		defaultTTL: ttl,
	}
}

// GetParameter retrieves a parameter from cache or SSM
func (sc *SSMCache) GetParameter(ctx context.Context, name string, withDecryption bool) (string, error) {
	// Check cache first
	if cached := sc.getCached(name); cached != nil {
		return cached.Value, nil
	}

	// Fetch from SSM
	value, err := sc.client.GetParameter(ctx, name, withDecryption)
	if err != nil {
		return "", err
	}

	// Cache the value
	sc.setCached(name, value, withDecryption)

	return value, nil
}

// GetParameters retrieves multiple parameters from cache or SSM
func (sc *SSMCache) GetParameters(ctx context.Context, names []string, withDecryption bool) (map[string]string, error) {
	result := make(map[string]string)
	missing := make([]string, 0)

	// Check cache for each parameter
	sc.mutex.RLock()
	for _, name := range names {
		if cached, exists := sc.cache[name]; exists && time.Now().Before(cached.ExpiresAt) {
			result[name] = cached.Value
		} else {
			missing = append(missing, name)
		}
	}
	sc.mutex.RUnlock()

	// Fetch missing parameters from SSM
	if len(missing) > 0 {
		params, err := sc.client.GetParameters(ctx, missing, withDecryption)
		if err != nil {
			return result, err
		}

		// Add fetched parameters to result and cache
		for name, value := range params {
			result[name] = value
			sc.setCached(name, value, withDecryption)
		}
	}

	return result, nil
}

// GetParametersByPath retrieves parameters by path with caching
func (sc *SSMCache) GetParametersByPath(ctx context.Context, path string, withDecryption bool, recursive bool) (map[string]string, error) {
	// For path-based retrieval, we always fetch fresh data
	// as the parameters under a path can change frequently
	params, err := sc.client.GetParametersByPath(ctx, path, withDecryption, recursive)
	if err != nil {
		return nil, err
	}

	// Cache individual parameters
	for name, value := range params {
		sc.setCached(name, value, withDecryption)
	}

	return params, nil
}

// Invalidate removes a parameter from cache
func (sc *SSMCache) Invalidate(name string) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	delete(sc.cache, name)
}

// InvalidateAll clears the entire cache
func (sc *SSMCache) InvalidateAll() {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	sc.cache = make(map[string]*CachedParameter)
}

// getCached retrieves a parameter from cache if valid
func (sc *SSMCache) getCached(name string) *CachedParameter {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	cached, exists := sc.cache[name]
	if !exists {
		return nil
	}

	// Check if expired
	if time.Now().After(cached.ExpiresAt) {
		return nil
	}

	return cached
}

// setCached stores a parameter in cache
func (sc *SSMCache) setCached(name string, value string, encrypted bool) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	sc.cache[name] = &CachedParameter{
		Value:      value,
		ExpiresAt:  time.Now().Add(sc.defaultTTL),
		Encrypted:  encrypted,
		LastUpdate: time.Now(),
	}
}

// CleanupExpired removes expired entries from cache
func (sc *SSMCache) CleanupExpired() {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	now := time.Now()
	for name, cached := range sc.cache {
		if now.After(cached.ExpiresAt) {
			delete(sc.cache, name)
		}
	}
}

// StartAutoCleanup starts a background goroutine to cleanup expired entries
func (sc *SSMCache) StartAutoCleanup(interval time.Duration) {
	if interval == 0 {
		interval = 1 * time.Minute // Default 1 minute
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			sc.CleanupExpired()
		}
	}()
}

// GetStats returns cache statistics
func (sc *SSMCache) GetStats() map[string]interface{} {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	expired := 0
	now := time.Now()
	for _, cached := range sc.cache {
		if now.After(cached.ExpiresAt) {
			expired++
		}
	}

	return map[string]interface{}{
		"total_entries":   len(sc.cache),
		"expired_entries": expired,
		"valid_entries":   len(sc.cache) - expired,
		"default_ttl":     sc.defaultTTL.String(),
	}
}
