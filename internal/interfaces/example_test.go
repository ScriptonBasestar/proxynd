package interfaces_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"proxynd/internal/interfaces"
	"proxynd/internal/interfaces/mocks"
)

// ExampleService demonstrates a service that uses interfaces for testability
type ExampleService struct {
	cache  interfaces.CacheManager
	auth   interfaces.AuthService
	health interfaces.HealthService
}

// NewExampleService creates a new example service with injected dependencies
func NewExampleService(
	cache interfaces.CacheManager,
	auth interfaces.AuthService,
	health interfaces.HealthService,
) *ExampleService {
	return &ExampleService{
		cache:  cache,
		auth:   auth,
		health: health,
	}
}

// GetUserData demonstrates using the cache interface
func (s *ExampleService) GetUserData(ctx context.Context, userID string) ([]byte, error) {
	// Try to get from cache first
	cacheKey := "user:" + userID
	data, found, err := s.cache.Get(ctx, cacheKey)
	if err != nil {
		return nil, err
	}

	if found {
		return data, nil
	}

	// If not in cache, fetch from auth service
	user, err := s.auth.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Convert user to JSON (simplified for example)
	userData := []byte(user.Name)

	// Store in cache for 5 minutes
	if err := s.cache.Put(ctx, cacheKey, userData, 5*time.Minute); err != nil {
		// Log error but don't fail the request
		_ = err
	}

	return userData, nil
}

// TestExampleService_GetUserData demonstrates testing with mock interfaces
func TestExampleService_GetUserData(t *testing.T) {
	ctx := context.Background()

	t.Run("cache hit", func(t *testing.T) {
		// Setup mocks
		mockCache := mocks.NewMockCacheManager(t)
		expectedData := []byte("John Doe")

		// Configure mock behavior
		mockCache.EXPECT().Get(
			mock.Anything,
			"user:123",
		).Return(expectedData, true, nil)

		// Create service with mocks
		service := NewExampleService(mockCache, nil, nil)

		// Test
		data, err := service.GetUserData(ctx, "123")
		require.NoError(t, err)
		assert.Equal(t, expectedData, data)

		// Verify cache stats
		stats := mockCache.GetStats()
		assert.Equal(t, int64(1), stats.Hits)
	})

	t.Run("cache miss", func(t *testing.T) {
		// Setup mocks
		mockCache := mocks.NewMockCacheManager(t)
		mockAuth := &MockAuthService{
			GetUserFunc: func(_ context.Context, userID string) (*interfaces.User, error) {
				return &interfaces.User{
					ID:   userID,
					Name: "Jane Doe",
				}, nil
			},
		}

		// Configure cache to return miss
		mockCache.EXPECT().Get(
			mock.Anything,
			"user:456",
		).Return(nil, false, nil)

		mockCache.EXPECT().Put(
			mock.Anything,
			"user:456",
			mock.Anything,
			mock.Anything,
		).Return(nil)

		// Create service with mocks
		service := NewExampleService(mockCache, mockAuth, nil)

		// Test
		data, err := service.GetUserData(ctx, "456")
		require.NoError(t, err)
		assert.Equal(t, []byte("Jane Doe"), data)
		// Mock assertions are handled automatically by testify/mock
	})
}

// MockAuthService is a simple mock for the AuthService interface
type MockAuthService struct {
	GetUserFunc func(ctx context.Context, userID string) (*interfaces.User, error)
}

func (m *MockAuthService) Authenticate(_ context.Context, _ interfaces.Credentials) (*interfaces.User, error) {
	panic("not implemented")
}

func (m *MockAuthService) AuthorizeRequest(_ context.Context, _ string, _ string, _ string) (bool, error) {
	panic("not implemented")
}

func (m *MockAuthService) GetUser(ctx context.Context, userID string) (*interfaces.User, error) {
	if m.GetUserFunc != nil {
		return m.GetUserFunc(ctx, userID)
	}
	panic("GetUserFunc not set")
}

func (m *MockAuthService) CreateUser(_ context.Context, _ *interfaces.User) error {
	panic("not implemented")
}

func (m *MockAuthService) UpdateUser(_ context.Context, _ string, _ map[string]interface{}) error {
	panic("not implemented")
}

// Ensure MockAuthService implements AuthService interface
var _ interfaces.AuthService = (*MockAuthService)(nil)
