package mocks

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"

	"proxynd/internal/repositories/cache"
)

// MockCacheRepository is a mock implementation of the cache.Repository interface
type MockCacheRepository struct {
	mock.Mock
}

// NewMockCacheRepository creates a new MockCacheRepository instance
func NewMockCacheRepository() *MockCacheRepository {
	return &MockCacheRepository{}
}

// Get retrieves a value from the cache
func (m *MockCacheRepository) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

// Set stores a value in the cache
func (m *MockCacheRepository) Set(ctx context.Context, key string, value []byte) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

// SetWithTTL stores a value in the cache with TTL
func (m *MockCacheRepository) SetWithTTL(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

// Delete removes a value from the cache
func (m *MockCacheRepository) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

// Exists checks if a key exists in the cache
func (m *MockCacheRepository) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

// GetStats returns cache statistics
func (m *MockCacheRepository) GetStats(ctx context.Context) (*cache.CacheStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*cache.CacheStats), args.Error(1)
}

// Clear removes all values from the cache
func (m *MockCacheRepository) Clear(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Keys returns all keys matching the pattern
func (m *MockCacheRepository) Keys(ctx context.Context, pattern string) ([]string, error) {
	args := m.Called(ctx, pattern)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

// GetMultiple retrieves multiple values at once
func (m *MockCacheRepository) GetMultiple(ctx context.Context, keys []string) (map[string][]byte, error) {
	args := m.Called(ctx, keys)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string][]byte), args.Error(1)
}

// SetMultiple stores multiple key-value pairs at once
func (m *MockCacheRepository) SetMultiple(ctx context.Context, items map[string][]byte) error {
	args := m.Called(ctx, items)
	return args.Error(0)
}

// Ping checks the cache connection
func (m *MockCacheRepository) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
