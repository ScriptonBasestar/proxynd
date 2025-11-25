package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"proxynd/internal/ports"
)

// TokenBucketRateLimiter implements ports.RateLimiter using token bucket algorithm
type TokenBucketRateLimiter struct {
	mu      sync.RWMutex
	buckets map[string]*bucket
	config  *ports.RateLimitConfig
	logger  ports.Logger

	// Cleanup settings
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

// bucket represents a token bucket for a single key
type bucket struct {
	tokens     float64
	lastRefill time.Time
	mu         sync.Mutex
}

// NewTokenBucketRateLimiter creates a new token bucket rate limiter
func NewTokenBucketRateLimiter(config *ports.RateLimitConfig, logger ports.Logger) ports.RateLimiter {
	if config == nil {
		config = &ports.RateLimitConfig{
			RequestsPerSecond: 100,
			BurstSize:         20,
			SkipSuccessful:    false,
			TrustedProxies:    []string{"127.0.0.1", "::1"},
		}
	}

	rl := &TokenBucketRateLimiter{
		buckets:         make(map[string]*bucket),
		config:          config,
		logger:          logger,
		cleanupInterval: time.Minute,
		stopCleanup:     make(chan struct{}),
	}

	// Start background cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// Allow checks if a single request should be allowed
func (rl *TokenBucketRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	return rl.AllowN(ctx, key, 1)
}

// AllowN checks if n requests should be allowed
func (rl *TokenBucketRateLimiter) AllowN(ctx context.Context, key string, n int) (bool, error) {
	b := rl.getBucket(key)

	b.mu.Lock()
	defer b.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * rl.config.RequestsPerSecond
	if b.tokens > float64(rl.config.BurstSize) {
		b.tokens = float64(rl.config.BurstSize)
	}
	b.lastRefill = now

	// Check if we have enough tokens
	if b.tokens >= float64(n) {
		b.tokens -= float64(n)
		return true, nil
	}

	return false, nil
}

// Reserve reserves n tokens for a key and returns reservation info
func (rl *TokenBucketRateLimiter) Reserve(ctx context.Context, key string, n int) (*ports.Reservation, error) {
	b := rl.getBucket(key)

	b.mu.Lock()
	defer b.mu.Unlock()

	// Refill tokens
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * rl.config.RequestsPerSecond
	if b.tokens > float64(rl.config.BurstSize) {
		b.tokens = float64(rl.config.BurstSize)
	}
	b.lastRefill = now

	// Calculate delay if not enough tokens
	if b.tokens >= float64(n) {
		b.tokens -= float64(n)
		return &ports.Reservation{
			OK:        true,
			Delay:     0,
			TimeToAct: now,
			Cancel:    func() { rl.returnTokens(key, n) },
		}, nil
	}

	// Calculate how long until we have enough tokens
	tokensNeeded := float64(n) - b.tokens
	delay := time.Duration(tokensNeeded/rl.config.RequestsPerSecond*1000) * time.Millisecond
	timeToAct := now.Add(delay)

	return &ports.Reservation{
		OK:        false,
		Delay:     delay,
		TimeToAct: timeToAct,
		Cancel:    nil,
	}, nil
}

// Reset resets the rate limiter for a key
func (rl *TokenBucketRateLimiter) Reset(ctx context.Context, key string) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.buckets, key)
	return nil
}

// CheckLimit checks if a request should be rate limited
func (rl *TokenBucketRateLimiter) CheckLimit(ctx context.Context, req *ports.RateLimitRequest) error {
	key := req.Key
	if key == "" {
		key = req.ClientIP
	}

	allowed, err := rl.AllowN(ctx, key, req.Count)
	if err != nil {
		return err
	}

	if !allowed {
		return fmt.Errorf("rate limit exceeded for key: %s", key)
	}

	return nil
}

// GetConfig returns the current rate limit configuration
func (rl *TokenBucketRateLimiter) GetConfig() *ports.RateLimitConfig {
	return rl.config
}

// getBucket returns or creates a bucket for the given key
func (rl *TokenBucketRateLimiter) getBucket(key string) *bucket {
	rl.mu.RLock()
	b, exists := rl.buckets[key]
	rl.mu.RUnlock()

	if exists {
		return b
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock
	if b, exists = rl.buckets[key]; exists {
		return b
	}

	b = &bucket{
		tokens:     float64(rl.config.BurstSize),
		lastRefill: time.Now(),
	}
	rl.buckets[key] = b
	return b
}

// returnTokens returns tokens to the bucket (for cancelled reservations)
func (rl *TokenBucketRateLimiter) returnTokens(key string, n int) {
	b := rl.getBucket(key)
	b.mu.Lock()
	defer b.mu.Unlock()

	b.tokens += float64(n)
	if b.tokens > float64(rl.config.BurstSize) {
		b.tokens = float64(rl.config.BurstSize)
	}
}

// cleanupLoop periodically removes stale buckets
func (rl *TokenBucketRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-rl.stopCleanup:
			return
		}
	}
}

// cleanup removes buckets that have been idle for too long
func (rl *TokenBucketRateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	staleThreshold := 5 * time.Minute

	for key, b := range rl.buckets {
		b.mu.Lock()
		idle := now.Sub(b.lastRefill) > staleThreshold
		b.mu.Unlock()

		if idle {
			delete(rl.buckets, key)
		}
	}

	// Log if too many buckets (potential memory issue)
	if len(rl.buckets) > 10000 && rl.logger != nil {
		ctx := context.Background()
		rl.logger.Warn(ctx, "Rate limiter has many tracked keys",
			&logField{"tracked_keys", len(rl.buckets)})
	}
}

// Stop stops the background cleanup goroutine
func (rl *TokenBucketRateLimiter) Stop() {
	close(rl.stopCleanup)
}

// GetStats returns statistics about the rate limiter
func (rl *TokenBucketRateLimiter) GetStats() RateLimiterStats {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return RateLimiterStats{
		TrackedKeys: len(rl.buckets),
		Config:      rl.config,
	}
}

// RateLimiterStats holds rate limiter statistics
type RateLimiterStats struct {
	TrackedKeys int
	Config      *ports.RateLimitConfig
}

// logField implements ports.Field for logging
type logField struct {
	key   string
	value interface{}
}

func (f *logField) Key() string {
	return f.key
}

func (f *logField) Value() interface{} {
	return f.value
}
