package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"

	"proxynd/internal/ports"
)

// RedisRateLimiter implements ports.RateLimiter using Redis for distributed rate limiting
type RedisRateLimiter struct {
	client    *redis.Client
	config    *ports.RateLimitConfig
	keyPrefix string
	logger    ports.Logger
}

// RedisRateLimiterConfig holds Redis rate limiter configuration
type RedisRateLimiterConfig struct {
	Address     string
	Password    string
	DB          int
	KeyPrefix   string
	MaxRetries  int
	DialTimeout time.Duration
}

// NewRedisRateLimiter creates a new Redis-backed rate limiter
func NewRedisRateLimiter(
	redisConfig RedisRateLimiterConfig,
	config *ports.RateLimitConfig,
	logger ports.Logger,
) (ports.RateLimiter, error) {
	if config == nil {
		config = &ports.RateLimitConfig{
			RequestsPerSecond: 100,
			BurstSize:         20,
			SkipSuccessful:    false,
			TrustedProxies:    []string{"127.0.0.1", "::1"},
		}
	}

	// Set defaults
	if redisConfig.MaxRetries == 0 {
		redisConfig.MaxRetries = 3
	}
	if redisConfig.DialTimeout == 0 {
		redisConfig.DialTimeout = 5 * time.Second
	}
	if redisConfig.KeyPrefix == "" {
		redisConfig.KeyPrefix = "proxynd:ratelimit:"
	}

	client := redis.NewClient(&redis.Options{
		Addr:        redisConfig.Address,
		Password:    redisConfig.Password,
		DB:          redisConfig.DB,
		MaxRetries:  redisConfig.MaxRetries,
		DialTimeout: redisConfig.DialTimeout,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), redisConfig.DialTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisRateLimiter{
		client:    client,
		config:    config,
		keyPrefix: redisConfig.KeyPrefix,
		logger:    logger,
	}, nil
}

// NewRedisRateLimiterWithClient creates a rate limiter with an existing Redis client
func NewRedisRateLimiterWithClient(
	client *redis.Client,
	config *ports.RateLimitConfig,
	keyPrefix string,
	logger ports.Logger,
) ports.RateLimiter {
	if config == nil {
		config = &ports.RateLimitConfig{
			RequestsPerSecond: 100,
			BurstSize:         20,
			SkipSuccessful:    false,
			TrustedProxies:    []string{"127.0.0.1", "::1"},
		}
	}

	if keyPrefix == "" {
		keyPrefix = "proxynd:ratelimit:"
	}

	return &RedisRateLimiter{
		client:    client,
		config:    config,
		keyPrefix: keyPrefix,
		logger:    logger,
	}
}

// Allow checks if a single request should be allowed
func (rl *RedisRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	return rl.AllowN(ctx, key, 1)
}

// AllowN checks if n requests should be allowed using sliding window counter
func (rl *RedisRateLimiter) AllowN(ctx context.Context, key string, n int) (bool, error) {
	fullKey := rl.keyPrefix + key
	now := time.Now()
	windowSize := time.Second // 1-second windows for per-second rate limiting

	// Use Lua script for atomic operation
	script := redis.NewScript(`
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])
		local requested = tonumber(ARGV[4])

		-- Remove old entries
		redis.call('ZREMRANGEBYSCORE', key, '-inf', now - window)

		-- Count current entries
		local current = redis.call('ZCARD', key)

		if current + requested <= limit then
			-- Add new entries
			for i = 1, requested do
				redis.call('ZADD', key, now, now .. '-' .. i .. '-' .. math.random())
			end
			-- Set expiry
			redis.call('EXPIRE', key, window / 1000)
			return 1
		else
			return 0
		end
	`)

	// Calculate limit based on requests per second and window
	limit := int(rl.config.RequestsPerSecond)

	result, err := script.Run(ctx, rl.client, []string{fullKey},
		now.UnixMilli(),
		windowSize.Milliseconds(),
		limit,
		n,
	).Int()
	if err != nil {
		return false, fmt.Errorf("rate limit check failed: %w", err)
	}

	return result == 1, nil
}

// Reserve reserves n tokens for a key
func (rl *RedisRateLimiter) Reserve(ctx context.Context, key string, n int) (*ports.Reservation, error) {
	allowed, err := rl.AllowN(ctx, key, n)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	if allowed {
		return &ports.Reservation{
			OK:        true,
			Delay:     0,
			TimeToAct: now,
			Cancel: func() {
				// For Redis, we can't easily return tokens, so this is a no-op
			},
		}, nil
	}

	// Calculate delay until next available slot
	delay := time.Duration(float64(n)/rl.config.RequestsPerSecond*1000) * time.Millisecond

	return &ports.Reservation{
		OK:        false,
		Delay:     delay,
		TimeToAct: now.Add(delay),
		Cancel:    nil,
	}, nil
}

// Reset resets the rate limiter for a key
func (rl *RedisRateLimiter) Reset(ctx context.Context, key string) error {
	fullKey := rl.keyPrefix + key
	return rl.client.Del(ctx, fullKey).Err()
}

// CheckLimit checks if a request should be rate limited
func (rl *RedisRateLimiter) CheckLimit(ctx context.Context, req *ports.RateLimitRequest) error {
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
func (rl *RedisRateLimiter) GetConfig() *ports.RateLimitConfig {
	return rl.config
}

// GetCurrentCount returns the current request count for a key
func (rl *RedisRateLimiter) GetCurrentCount(ctx context.Context, key string) (int64, error) {
	fullKey := rl.keyPrefix + key
	now := time.Now()
	windowSize := time.Second

	// Remove old entries first
	rl.client.ZRemRangeByScore(ctx, fullKey, "-inf", strconv.FormatInt(now.Add(-windowSize).UnixMilli(), 10))

	// Count current entries
	return rl.client.ZCard(ctx, fullKey).Result()
}

// Close closes the Redis connection
func (rl *RedisRateLimiter) Close() error {
	return rl.client.Close()
}
