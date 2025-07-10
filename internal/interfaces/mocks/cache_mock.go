package mocks

import (
	"context"
	"sync"
	"time"

	"proxynd/internal/interfaces"
)

// MockCacheManager is a mock implementation of CacheManager for testing
type MockCacheManager struct {
	mu         sync.RWMutex
	data       map[string][]byte
	expiration map[string]time.Time
	stats      interfaces.CacheStats

	// Behavior control
	GetFunc    func(ctx context.Context, key string) ([]byte, bool, error)
	PutFunc    func(ctx context.Context, key string, data []byte, ttl time.Duration) error
	DeleteFunc func(ctx context.Context, key string) error
}

// NewMockCacheManager creates a new mock cache manager
func NewMockCacheManager() *MockCacheManager {
	return &MockCacheManager{
		data:       make(map[string][]byte),
		expiration: make(map[string]time.Time),
		stats:      interfaces.CacheStats{},
	}
}

// Get retrieves data from mock cache
func (m *MockCacheManager) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, key)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check expiration
	if exp, exists := m.expiration[key]; exists && time.Now().After(exp) {
		m.stats.Misses++
		return nil, false, nil
	}

	data, exists := m.data[key]
	if exists {
		m.stats.Hits++
	} else {
		m.stats.Misses++
	}

	return data, exists, nil
}

// Put stores data in mock cache
func (m *MockCacheManager) Put(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	if m.PutFunc != nil {
		return m.PutFunc(ctx, key, data, ttl)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[key] = data
	m.expiration[key] = time.Now().Add(ttl)
	m.stats.ItemCount = int64(len(m.data))

	return nil
}

// Exists checks if key exists in mock cache
func (m *MockCacheManager) Exists(ctx context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check expiration
	if exp, exists := m.expiration[key]; exists && time.Now().After(exp) {
		return false, nil
	}

	_, exists := m.data[key]
	return exists, nil
}

// Delete removes key from mock cache
func (m *MockCacheManager) Delete(ctx context.Context, key string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, key)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, key)
	delete(m.expiration, key)
	m.stats.ItemCount = int64(len(m.data))

	return nil
}

// Clear removes all entries from mock cache
func (m *MockCacheManager) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data = make(map[string][]byte)
	m.expiration = make(map[string]time.Time)
	m.stats.ItemCount = 0

	return nil
}

// GetStats returns mock cache statistics
func (m *MockCacheManager) GetStats() interfaces.CacheStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.stats
}

// StartEviction is a no-op for mock
func (m *MockCacheManager) StartEviction(ctx context.Context, interval time.Duration) {
	// No-op for mock
}

// StopEviction is a no-op for mock
func (m *MockCacheManager) StopEviction() {
	// No-op for mock
}

// Ensure MockCacheManager implements CacheManager interface
var _ interfaces.CacheManager = (*MockCacheManager)(nil)
