package adapters

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"proxynd/internal/repositories/cache"
)

// Mock repository
type MockCacheRepository struct {
	mock.Mock
}

func (m *MockCacheRepository) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockCacheRepository) Put(ctx context.Context, key string, content io.Reader, ttl time.Duration) error {
	args := m.Called(ctx, key, content, ttl)
	return args.Error(0)
}

func (m *MockCacheRepository) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCacheRepository) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockCacheRepository) List(ctx context.Context, pattern string) ([]string, error) {
	args := m.Called(ctx, pattern)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockCacheRepository) Size(ctx context.Context, key string) (int64, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCacheRepository) Clear(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCacheRepository) Stats(ctx context.Context) (*cache.CacheStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*cache.CacheStats), args.Error(1)
}

// Tests

func TestNewCacheAdapter(t *testing.T) {
	repo := &MockCacheRepository{}
	ttl := 5 * time.Minute
	adapter := NewCacheAdapter(repo, ttl)

	assert.NotNil(t, adapter)
	assert.Equal(t, repo, adapter.repo)
	assert.Equal(t, ttl, adapter.ttl)
}

func TestCacheAdapter_Get(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		key         string
		setupMock   func(*MockCacheRepository)
		wantExists  bool
		wantErr     bool
		wantContent string
	}{
		{
			name: "cache hit",
			key:  "test-key",
			setupMock: func(m *MockCacheRepository) {
				content := io.NopCloser(bytes.NewBufferString("cached content"))
				m.On("Get", ctx, "test-key").Return(content, nil)
			},
			wantExists:  true,
			wantErr:     false,
			wantContent: "cached content",
		},
		{
			name: "cache miss",
			key:  "missing-key",
			setupMock: func(m *MockCacheRepository) {
				m.On("Get", ctx, "missing-key").Return(nil, fmt.Errorf("not found"))
			},
			wantExists: false,
			wantErr:    false,
		},
		{
			name: "repository error",
			key:  "error-key",
			setupMock: func(m *MockCacheRepository) {
				m.On("Get", ctx, "error-key").Return(nil, fmt.Errorf("repo error"))
			},
			wantExists: false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockCacheRepository{}
			tt.setupMock(repo)

			adapter := NewCacheAdapter(repo, 5*time.Minute)
			content, exists, err := adapter.Get(ctx, tt.key)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantExists, exists)

				if tt.wantExists {
					require.NotNil(t, content)
					data, err := io.ReadAll(content)
					require.NoError(t, err)
					assert.Equal(t, tt.wantContent, string(data))
				} else {
					assert.Nil(t, content)
				}
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestCacheAdapter_Put(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		key       string
		content   string
		setupMock func(*MockCacheRepository)
		wantErr   bool
	}{
		{
			name:    "successful put",
			key:     "test-key",
			content: "test content",
			setupMock: func(m *MockCacheRepository) {
				m.On("Put", ctx, "test-key", mock.AnythingOfType("*bytes.Buffer"), mock.AnythingOfType("time.Duration")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:    "repository error",
			key:     "error-key",
			content: "test content",
			setupMock: func(m *MockCacheRepository) {
				m.On("Put", ctx, "error-key", mock.AnythingOfType("*bytes.Buffer"), mock.AnythingOfType("time.Duration")).
					Return(fmt.Errorf("write error"))
			},
			wantErr: true,
		},
		{
			name:    "empty content",
			key:     "empty-key",
			content: "",
			setupMock: func(m *MockCacheRepository) {
				m.On("Put", ctx, "empty-key", mock.AnythingOfType("*bytes.Buffer"),
					mock.AnythingOfType("time.Duration")).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockCacheRepository{}
			tt.setupMock(repo)

			adapter := NewCacheAdapter(repo, 5*time.Minute)
			reader := bytes.NewBufferString(tt.content)
			err := adapter.Put(ctx, tt.key, reader)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestCacheAdapter_Exists(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		key        string
		setupMock  func(*MockCacheRepository)
		wantExists bool
		wantErr    bool
	}{
		{
			name: "key exists",
			key:  "existing-key",
			setupMock: func(m *MockCacheRepository) {
				m.On("Exists", ctx, "existing-key").Return(true, nil)
			},
			wantExists: true,
			wantErr:    false,
		},
		{
			name: "key does not exist",
			key:  "missing-key",
			setupMock: func(m *MockCacheRepository) {
				m.On("Exists", ctx, "missing-key").Return(false, nil)
			},
			wantExists: false,
			wantErr:    false,
		},
		{
			name: "repository error",
			key:  "error-key",
			setupMock: func(m *MockCacheRepository) {
				m.On("Exists", ctx, "error-key").Return(false, fmt.Errorf("check error"))
			},
			wantExists: false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockCacheRepository{}
			tt.setupMock(repo)

			adapter := NewCacheAdapter(repo, 5*time.Minute)
			exists, err := adapter.Exists(ctx, tt.key)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantExists, exists)
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestCacheAdapter_Delete(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		key       string
		setupMock func(*MockCacheRepository)
		wantErr   bool
	}{
		{
			name: "successful delete",
			key:  "test-key",
			setupMock: func(m *MockCacheRepository) {
				m.On("Delete", ctx, "test-key").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "repository error",
			key:  "error-key",
			setupMock: func(m *MockCacheRepository) {
				m.On("Delete", ctx, "error-key").Return(fmt.Errorf("delete error"))
			},
			wantErr: true,
		},
		{
			name: "delete non-existent key",
			key:  "missing-key",
			setupMock: func(m *MockCacheRepository) {
				m.On("Delete", ctx, "missing-key").Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockCacheRepository{}
			tt.setupMock(repo)

			adapter := NewCacheAdapter(repo, 5*time.Minute)
			err := adapter.Delete(ctx, tt.key)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestCacheAdapter_LargeContent(t *testing.T) {
	ctx := context.Background()
	repo := &MockCacheRepository{}

	// Create large content (1MB)
	largeContent := make([]byte, 1024*1024)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	repo.On("Put", ctx, "large-key", mock.AnythingOfType("*bytes.Reader"),
		mock.AnythingOfType("time.Duration")).Return(nil)

	adapter := NewCacheAdapter(repo, 5*time.Minute)
	reader := bytes.NewReader(largeContent)
	err := adapter.Put(ctx, "large-key", reader)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}
