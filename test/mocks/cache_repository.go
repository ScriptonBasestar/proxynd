package mocks

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"

	"proxynd/internal/repositories/cache"
)

// MockCacheRepository는 cache.Repository 인터페이스의 모의 구현
type MockCacheRepository struct {
	mock.Mock
}

// NewMockCacheRepository는 새로운 MockCacheRepository를 생성
func NewMockCacheRepository() *MockCacheRepository {
	return &MockCacheRepository{}
}

// Get은 캐시에서 값을 가져옴
func (m *MockCacheRepository) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

// Set은 캐시에 값을 저장
func (m *MockCacheRepository) Set(ctx context.Context, key string, value []byte) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

// SetWithTTL은 TTL과 함께 캐시에 값을 저장
func (m *MockCacheRepository) SetWithTTL(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

// Delete는 캐시에서 값을 삭제
func (m *MockCacheRepository) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

// Exists는 키가 존재하는지 확인
func (m *MockCacheRepository) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

// GetStats는 캐시 통계를 반환
func (m *MockCacheRepository) GetStats(ctx context.Context) (*cache.Stats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*cache.Stats), args.Error(1)
}

// Clear는 모든 캐시를 삭제
func (m *MockCacheRepository) Clear(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Keys는 패턴과 일치하는 모든 키를 반환
func (m *MockCacheRepository) Keys(ctx context.Context, pattern string) ([]string, error) {
	args := m.Called(ctx, pattern)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

// GetMultiple은 여러 키의 값을 한 번에 가져옴
func (m *MockCacheRepository) GetMultiple(ctx context.Context, keys []string) (map[string][]byte, error) {
	args := m.Called(ctx, keys)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string][]byte), args.Error(1)
}

// SetMultiple은 여러 키-값 쌍을 한 번에 저장
func (m *MockCacheRepository) SetMultiple(ctx context.Context, items map[string][]byte) error {
	args := m.Called(ctx, items)
	return args.Error(0)
}

// Ping은 캐시 연결을 확인
func (m *MockCacheRepository) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}