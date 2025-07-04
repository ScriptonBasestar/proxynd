package example_test

import (
	"context"
	"errors"
	"testing"
	"time"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	
	"proxynd/internal/interfaces/mocks"
	"proxynd/internal/services/example"
)

// MockConfigService implements ConfigServiceWithContext for testing
type MockConfigService struct {
	GetConfigFunc    func(ctx context.Context, key string) (interface{}, error)
	ReloadConfigFunc func(ctx context.Context) error
}

func (m *MockConfigService) GetConfig(ctx context.Context, key string) (interface{}, error) {
	if m.GetConfigFunc != nil {
		return m.GetConfigFunc(ctx, key)
	}
	return nil, nil
}

func (m *MockConfigService) ReloadConfig(ctx context.Context) error {
	if m.ReloadConfigFunc != nil {
		return m.ReloadConfigFunc(ctx)
	}
	return nil
}

func TestServiceWithContext_ProcessRequest(t *testing.T) {
	t.Run("successful request with cache miss", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockCache := mocks.NewMockCacheManager()
		mockConfig := &MockConfigService{}
		
		// Configure cache to return miss
		mockCache.GetFunc = func(ctx context.Context, key string) ([]byte, bool, error) {
			assert.Equal(t, "request:test-123", key)
			return nil, false, nil
		}
		
		// Configure cache put to succeed
		putCalled := false
		mockCache.PutFunc = func(ctx context.Context, key string, data []byte, ttl time.Duration) error {
			putCalled = true
			assert.Equal(t, "request:test-123", key)
			assert.Equal(t, []byte("data for test-123"), data)
			assert.Equal(t, 5*time.Minute, ttl)
			return nil
		}
		
		// Create service
		service, err := example.NewServiceWithContext(ctx, mockCache, mockConfig)
		require.NoError(t, err)
		
		// Test
		response, err := service.ProcessRequest(ctx, "test-123")
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.False(t, response.Cached)
		assert.Equal(t, []byte("data for test-123"), response.Data)
		
		// Wait briefly for async cache put
		time.Sleep(100 * time.Millisecond)
		assert.True(t, putCalled, "cache put should have been called")
	})
	
	t.Run("successful request with cache hit", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockCache := mocks.NewMockCacheManager()
		mockConfig := &MockConfigService{}
		
		cachedData := []byte("cached data")
		
		// Configure cache to return hit
		mockCache.GetFunc = func(ctx context.Context, key string) ([]byte, bool, error) {
			return cachedData, true, nil
		}
		
		// Create service
		service, err := example.NewServiceWithContext(ctx, mockCache, mockConfig)
		require.NoError(t, err)
		
		// Test
		response, err := service.ProcessRequest(ctx, "test-456")
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.True(t, response.Cached)
		assert.Equal(t, cachedData, response.Data)
	})
	
	t.Run("context cancellation during request", func(t *testing.T) {
		// Setup
		ctx, cancel := context.WithCancel(context.Background())
		mockCache := mocks.NewMockCacheManager()
		mockConfig := &MockConfigService{}
		
		// Configure cache to simulate slow operation
		mockCache.GetFunc = func(ctx context.Context, key string) ([]byte, bool, error) {
			// Cancel context during cache operation
			cancel()
			
			select {
			case <-ctx.Done():
				return nil, false, ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return nil, false, nil
			}
		}
		
		// Create service
		service, err := example.NewServiceWithContext(context.Background(), mockCache, mockConfig)
		require.NoError(t, err)
		
		// Test
		_, err = service.ProcessRequest(ctx, "test-789")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})
}

func TestServiceWithContext_PerformLongRunningOperation(t *testing.T) {
	t.Run("successful completion", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockCache := mocks.NewMockCacheManager()
		mockConfig := &MockConfigService{}
		
		service, err := example.NewServiceWithContext(ctx, mockCache, mockConfig)
		require.NoError(t, err)
		
		// Test with adequate timeout
		opCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		
		err = service.PerformLongRunningOperation(opCtx, "test-input")
		assert.NoError(t, err)
	})
	
	t.Run("timeout during operation", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockCache := mocks.NewMockCacheManager()
		mockConfig := &MockConfigService{}
		
		service, err := example.NewServiceWithContext(ctx, mockCache, mockConfig)
		require.NoError(t, err)
		
		// Test with very short timeout
		opCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()
		
		err = service.PerformLongRunningOperation(opCtx, "test-input")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, context.DeadlineExceeded))
	})
}

func TestServiceWithContext_ReloadConfiguration(t *testing.T) {
	t.Run("successful reload", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockCache := mocks.NewMockCacheManager()
		mockConfig := &MockConfigService{
			ReloadConfigFunc: func(ctx context.Context) error {
				// Verify context has timeout
				deadline, ok := ctx.Deadline()
				assert.True(t, ok, "context should have deadline")
				assert.WithinDuration(t, time.Now().Add(30*time.Second), deadline, 1*time.Second)
				return nil
			},
		}
		
		service, err := example.NewServiceWithContext(ctx, mockCache, mockConfig)
		require.NoError(t, err)
		
		// Test
		err = service.ReloadConfiguration(ctx)
		assert.NoError(t, err)
	})
	
	t.Run("reload failure", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockCache := mocks.NewMockCacheManager()
		mockConfig := &MockConfigService{
			ReloadConfigFunc: func(ctx context.Context) error {
				return errors.New("config file not found")
			},
		}
		
		service, err := example.NewServiceWithContext(ctx, mockCache, mockConfig)
		require.NoError(t, err)
		
		// Test
		err = service.ReloadConfiguration(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config reload failed")
	})
}

func TestNewServiceWithContext_Validation(t *testing.T) {
	t.Run("successful initialization", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockCache := mocks.NewMockCacheManager()
		mockConfig := &MockConfigService{}
		
		// Test
		service, err := example.NewServiceWithContext(ctx, mockCache, mockConfig)
		assert.NoError(t, err)
		assert.NotNil(t, service)
	})
	
	t.Run("initialization with context timeout", func(t *testing.T) {
		// Setup - context that times out immediately
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		
		// Wait for context to expire
		time.Sleep(10 * time.Millisecond)
		
		mockCache := mocks.NewMockCacheManager()
		mockConfig := &MockConfigService{}
		
		// Test - should still succeed as validation creates its own timeout context
		service, err := example.NewServiceWithContext(ctx, mockCache, mockConfig)
		assert.NoError(t, err)
		assert.NotNil(t, service)
	})
}