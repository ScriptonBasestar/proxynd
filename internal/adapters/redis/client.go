package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// ClientConfig holds Redis client configuration
type ClientConfig struct {
	Address     string
	Password    string
	DB          int
	KeyPrefix   string
	MaxRetries  int
	DialTimeout time.Duration
	ReadTimeout time.Duration
}

// RedisClient wraps the Redis client with ProxyND-specific functionality
type RedisClient struct {
	client    *redis.Client
	config    ClientConfig
	keyPrefix string
}

// NewRedisClient creates a new Redis client with the provided configuration
func NewRedisClient(cfg ClientConfig) (*RedisClient, error) {
	// Set defaults
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.DialTimeout == 0 {
		cfg.DialTimeout = 10 * time.Second
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 10 * time.Second
	}
	if cfg.KeyPrefix == "" {
		cfg.KeyPrefix = "proxynd:"
	}

	// Create Redis client options
	opts := &redis.Options{
		Addr:         cfg.Address,
		Password:     cfg.Password,
		DB:           cfg.DB,
		MaxRetries:   cfg.MaxRetries,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.ReadTimeout,
	}

	client := redis.NewClient(opts)

	redisClient := &RedisClient{
		client:    client,
		config:    cfg,
		keyPrefix: cfg.KeyPrefix,
	}

	return redisClient, nil
}

// HealthCheck verifies that the Redis client can connect to Redis server
func (r *RedisClient) HealthCheck(ctx context.Context) error {
	pong, err := r.client.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}
	if pong != "PONG" {
		return fmt.Errorf("redis health check failed: expected PONG, got %s", pong)
	}
	return nil
}

// Get retrieves a value from Redis
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	fullKey := r.keyPrefix + key
	val, err := r.client.Get(ctx, fullKey).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("key not found: %s", key)
	}
	if err != nil {
		return "", fmt.Errorf("failed to get key %s: %w", key, err)
	}
	return val, nil
}

// Set stores a value in Redis with TTL
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	fullKey := r.keyPrefix + key
	err := r.client.Set(ctx, fullKey, value, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set key %s: %w", key, err)
	}
	return nil
}

// Delete removes a key from Redis
func (r *RedisClient) Delete(ctx context.Context, key string) error {
	fullKey := r.keyPrefix + key
	err := r.client.Del(ctx, fullKey).Err()
	if err != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}
	return nil
}

// Exists checks if a key exists in Redis
func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := r.keyPrefix + key
	count, err := r.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check key existence %s: %w", key, err)
	}
	return count > 0, nil
}

// SetNX sets a key only if it doesn't exist (atomic operation)
func (r *RedisClient) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	fullKey := r.keyPrefix + key
	set, err := r.client.SetNX(ctx, fullKey, value, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to setnx key %s: %w", key, err)
	}
	return set, nil
}

// Increment atomically increments a key's value
func (r *RedisClient) Increment(ctx context.Context, key string) (int64, error) {
	fullKey := r.keyPrefix + key
	val, err := r.client.Incr(ctx, fullKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment key %s: %w", key, err)
	}
	return val, nil
}

// Expire sets TTL on an existing key
func (r *RedisClient) Expire(ctx context.Context, key string, ttl time.Duration) error {
	fullKey := r.keyPrefix + key
	err := r.client.Expire(ctx, fullKey, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set expiry for key %s: %w", key, err)
	}
	return nil
}

// FlushWithPrefix removes all keys with the client's prefix
func (r *RedisClient) FlushWithPrefix(ctx context.Context) error {
	pattern := r.keyPrefix + "*"
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get keys with prefix %s: %w", r.keyPrefix, err)
	}

	if len(keys) > 0 {
		err = r.client.Del(ctx, keys...).Err()
		if err != nil {
			return fmt.Errorf("failed to delete keys with prefix %s: %w", r.keyPrefix, err)
		}
	}

	return nil
}

// GetClient returns the underlying Redis client for advanced operations
func (r *RedisClient) GetClient() *redis.Client {
	return r.client
}

// Close closes the Redis client connection
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// GetInfo returns Redis server information
func (r *RedisClient) GetInfo(ctx context.Context) (map[string]string, error) {
	info, err := r.client.Info(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get redis info: %w", err)
	}

	// Parse info string into map
	infoMap := make(map[string]string)
	// Basic parsing - for more complex parsing, consider using a library
	// This is simplified for the current use case
	infoMap["raw"] = info
	
	return infoMap, nil
}