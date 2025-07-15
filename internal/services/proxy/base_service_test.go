package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock implementations

type MockCacheService struct {
	mock.Mock
}

func (m *MockCacheService) Get(ctx context.Context, key string) (io.ReadCloser, bool, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).(io.ReadCloser), args.Bool(1), args.Error(2)
}

func (m *MockCacheService) Put(ctx context.Context, key string, content io.Reader) error {
	args := m.Called(ctx, key, content)
	return args.Error(0)
}

func (m *MockCacheService) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockCacheService) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

type MockConfigService struct {
	mock.Mock
}

func (m *MockConfigService) GetProxyConfig(ctx context.Context, proxyType string) (interface{}, error) {
	args := m.Called(ctx, proxyType)
	return args.Get(0), args.Error(1)
}

func (m *MockConfigService) GetGlobalConfig(ctx context.Context) (interface{}, error) {
	args := m.Called(ctx)
	return args.Get(0), args.Error(1)
}

func (m *MockConfigService) ReloadConfig(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type MockUpstreamClient struct {
	mock.Mock
}

func (m *MockUpstreamClient) Fetch(ctx context.Context, url string, headers map[string]string) (*ProxyResponse, error) {
	args := m.Called(ctx, url, headers)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ProxyResponse), args.Error(1)
}

// Tests

func TestNewBaseProxyService(t *testing.T) {
	cache := &MockCacheService{}
	config := &MockConfigService{}
	upstream := &MockUpstreamClient{}

	service := NewBaseProxyService("test", cache, config, upstream)

	assert.NotNil(t, service)
	assert.Equal(t, "test", service.ProxyType)
	assert.Equal(t, cache, service.Cache)
	assert.Equal(t, config, service.Config)
	assert.Equal(t, upstream, service.UpstreamClient)
}

func TestBaseProxyService_GetProxyType(t *testing.T) {
	service := &BaseProxyService{ProxyType: "maven"}
	assert.Equal(t, "maven", service.GetProxyType())
}

func TestBaseProxyService_ValidateRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     ProxyRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			req: ProxyRequest{
				Path:      "/maven/central/com/example/lib/1.0/lib-1.0.jar",
				ProxyType: "maven",
				Method:    "GET",
			},
			wantErr: false,
		},
		{
			name: "empty path",
			req: ProxyRequest{
				Path:      "",
				ProxyType: "maven",
				Method:    "GET",
			},
			wantErr: true,
			errMsg:  "request path cannot be empty",
		},
		{
			name: "path traversal attack",
			req: ProxyRequest{
				Path:      "/maven/../../../etc/passwd",
				ProxyType: "maven",
				Method:    "GET",
			},
			wantErr: true,
			errMsg:  "directory traversal detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &BaseProxyService{ProxyType: "maven"}
			err := service.ValidateRequest(tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBaseProxyService_BuildCacheKey(t *testing.T) {
	tests := []struct {
		name        string
		proxyType   string
		requestPath string
		expected    string
	}{
		{
			name:        "simple path",
			proxyType:   "maven",
			requestPath: "/maven/central/test.jar",
			expected:    "maven/maven/central/test.jar",
		},
		{
			name:        "path with query",
			proxyType:   "npm",
			requestPath: "/npm/package.json",
			expected:    "npm/npm/package.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &BaseProxyService{ProxyType: tt.proxyType}
			key := service.BuildCacheKey(tt.requestPath)
			assert.Equal(t, tt.expected, key)
		})
	}
}

func TestBaseProxyService_TryCache(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		setupMocks func(*MockCacheService)
		cacheKey   string
		wantFound  bool
		wantErr    bool
	}{
		{
			name: "cache hit",
			setupMocks: func(cache *MockCacheService) {
				cache.On("Exists", ctx, "test:/test/path").Return(true, nil)
				content := io.NopCloser(bytes.NewBufferString("cached content"))
				cache.On("Get", ctx, "test:/test/path").Return(content, true, nil)
			},
			cacheKey:  "test:/test/path",
			wantFound: true,
			wantErr:   false,
		},
		{
			name: "cache miss",
			setupMocks: func(cache *MockCacheService) {
				cache.On("Exists", ctx, "test:/test/path").Return(false, nil)
			},
			cacheKey:  "test:/test/path",
			wantFound: false,
			wantErr:   false,
		},
		{
			name: "cache exists but get fails",
			setupMocks: func(cache *MockCacheService) {
				cache.On("Exists", ctx, "test:/test/path").Return(true, nil)
				cache.On("Get", ctx, "test:/test/path").Return(nil, false, fmt.Errorf("cache error"))
			},
			cacheKey:  "test:/test/path",
			wantFound: false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := &MockCacheService{}
			config := &MockConfigService{}
			upstream := &MockUpstreamClient{}

			tt.setupMocks(cache)

			service := NewBaseProxyService("test", cache, config, upstream)
			content, found, err := service.TryCache(ctx, tt.cacheKey)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.wantFound, found)
			if found {
				assert.NotNil(t, content)
			} else if !tt.wantErr {
				assert.Nil(t, content)
			}

			cache.AssertExpectations(t)
		})
	}
}

func TestBaseProxyService_CacheResponse(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		setupMocks func(*MockCacheService)
		cacheKey   string
		content    io.Reader
		wantErr    bool
	}{
		{
			name: "successful cache",
			setupMocks: func(cache *MockCacheService) {
				cache.On("Put", ctx, "test:/test/path", mock.Anything).Return(nil)
			},
			cacheKey: "test:/test/path",
			content:  bytes.NewBufferString("content to cache"),
			wantErr:  false,
		},
		{
			name: "cache error",
			setupMocks: func(cache *MockCacheService) {
				cache.On("Put", ctx, "test:/test/path", mock.Anything).
					Return(fmt.Errorf("cache write error"))
			},
			cacheKey: "test:/test/path",
			content:  bytes.NewBufferString("content"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := &MockCacheService{}
			config := &MockConfigService{}
			upstream := &MockUpstreamClient{}

			tt.setupMocks(cache)

			service := NewBaseProxyService("test", cache, config, upstream)
			err := service.CacheResponse(ctx, tt.cacheKey, tt.content)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			cache.AssertExpectations(t)
		})
	}
}

func TestBaseProxyService_HandleError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode int
		expectedMsg  string
	}{
		{
			name:         "generic error",
			err:          fmt.Errorf("something went wrong"),
			expectedCode: 500,
			expectedMsg:  "something went wrong",
		},
		{
			name:         "nil error",
			err:          nil,
			expectedCode: 500,
			expectedMsg:  "unknown error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &BaseProxyService{}
			resp := service.HandleError(tt.err, tt.expectedCode)

			assert.NotNil(t, resp)
			assert.Equal(t, tt.expectedCode, resp.StatusCode)

			// Read the body to check error message
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			if tt.err != nil {
				assert.Contains(t, string(body), tt.expectedMsg)
			} else {
				assert.Contains(t, string(body), "unknown error")
			}
		})
	}
}

func TestBaseProxyService_HandleRequest(t *testing.T) {
	service := &BaseProxyService{ProxyType: "base"}
	ctx := context.Background()
	req := ProxyRequest{Path: "/test"}

	resp, err := service.HandleRequest(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 501, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "service not implemented")
}
