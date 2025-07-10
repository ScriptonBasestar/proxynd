package adapters

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock repository
type MockCacheRepository struct {
	mock.Mock
}

func (m *MockCacheRepository) Get(ctx context.Context, key string) (io.ReadCloser, bool, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).(io.ReadCloser), args.Bool(1), args.Error(2)
}

func (m *MockCacheRepository) Set(ctx context.Context, key string, content []byte, metadata map[string]string) error {
	args := m.Called(ctx, key, content, metadata)
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

func (m *MockCacheRepository) List(ctx context.Context, prefix string) ([]string, error) {
	args := m.Called(ctx, prefix)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockCacheRepository) GetMetadata(ctx context.Context, key string) (map[string]string, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]string), args.Error(1)
}

// Tests

func TestNewCacheAdapter(t *testing.T) {
	repo := &MockCacheRepository{}
	adapter := NewCacheAdapter(repo)

	assert.NotNil(t, adapter)
	assert.Equal(t, repo, adapter.repository)
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
				content := ioutil.NopCloser(bytes.NewBufferString("cached content"))
				m.On("Get", ctx, "test-key").Return(content, true, nil)
			},
			wantExists:  true,
			wantErr:     false,
			wantContent: "cached content",
		},
		{
			name: "cache miss",
			key:  "missing-key",
			setupMock: func(m *MockCacheRepository) {
				m.On("Get", ctx, "missing-key").Return(nil, false, nil)
			},
			wantExists: false,
			wantErr:    false,
		},
		{
			name: "repository error",
			key:  "error-key",
			setupMock: func(m *MockCacheRepository) {
				m.On("Get", ctx, "error-key").Return(nil, false, fmt.Errorf("repo error"))
			},
			wantExists: false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockCacheRepository{}
			tt.setupMock(repo)

			adapter := NewCacheAdapter(repo)
			content, exists, err := adapter.Get(ctx, tt.key)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantExists, exists)

				if tt.wantExists {
					require.NotNil(t, content)
					data, err := ioutil.ReadAll(content)
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
				m.On("Set", ctx, "test-key", []byte("test content"), mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name:    "repository error",
			key:     "error-key",
			content: "test content",
			setupMock: func(m *MockCacheRepository) {
				m.On("Set", ctx, "error-key", []byte("test content"), mock.Anything).
					Return(fmt.Errorf("write error"))
			},
			wantErr: true,
		},
		{
			name:    "empty content",
			key:     "empty-key",
			content: "",
			setupMock: func(m *MockCacheRepository) {
				m.On("Set", ctx, "empty-key", []byte(""), mock.Anything).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockCacheRepository{}
			tt.setupMock(repo)

			adapter := NewCacheAdapter(repo)
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

			adapter := NewCacheAdapter(repo)
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

			adapter := NewCacheAdapter(repo)
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

	repo.On("Set", ctx, "large-key", largeContent, mock.Anything).Return(nil)

	adapter := NewCacheAdapter(repo)
	reader := bytes.NewReader(largeContent)
	err := adapter.Put(ctx, "large-key", reader)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}
