package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
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
	assert.Equal(t, "test", service.proxyType)
	assert.Equal(t, cache, service.cache)
	assert.Equal(t, config, service.configService)
	assert.Equal(t, upstream, service.upstreamClient)
}

func TestBaseProxyService_GetProxyType(t *testing.T) {
	service := &BaseProxyService{proxyType: "maven"}
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
			errMsg:  "path is required",
		},
		{
			name: "empty proxy type",
			req: ProxyRequest{
				Path:      "/maven/test",
				ProxyType: "",
				Method:    "GET",
			},
			wantErr: true,
			errMsg:  "proxy type is required",
		},
		{
			name: "empty method",
			req: ProxyRequest{
				Path:      "/maven/test",
				ProxyType: "maven",
				Method:    "",
			},
			wantErr: true,
			errMsg:  "method is required",
		},
		{
			name: "proxy type mismatch",
			req: ProxyRequest{
				Path:      "/maven/test",
				ProxyType: "npm",
				Method:    "GET",
			},
			wantErr: true,
			errMsg:  "proxy type mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &BaseProxyService{proxyType: "maven"}
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

func TestBaseProxyService_CacheKey(t *testing.T) {
	tests := []struct {
		name     string
		req      ProxyRequest
		expected string
	}{
		{
			name: "simple path",
			req: ProxyRequest{
				Path: "/maven/central/test.jar",
			},
			expected: "maven:/maven/central/test.jar",
		},
		{
			name: "path with query",
			req: ProxyRequest{
				Path: "/npm/package.json?version=1.0.0",
			},
			expected: "npm:/npm/package.json?version=1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &BaseProxyService{proxyType: tt.req.ProxyType}
			if service.proxyType == "" {
				service.proxyType = strings.Split(tt.expected, ":")[0]
			}
			key := service.CacheKey(tt.req)
			assert.Equal(t, tt.expected, key)
		})
	}
}

func TestBaseProxyService_FetchFromCacheOrUpstream(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		setupMocks  func(*MockCacheService, *MockUpstreamClient)
		req         ProxyRequest
		upstreamURL string
		wantCached  bool
		wantErr     bool
	}{
		{
			name: "cache hit",
			setupMocks: func(cache *MockCacheService, upstream *MockUpstreamClient) {
				content := ioutil.NopCloser(bytes.NewBufferString("cached content"))
				cache.On("Get", ctx, "test:/test/path").Return(content, true, nil)
			},
			req: ProxyRequest{
				Path: "/test/path",
			},
			upstreamURL: "http://example.com/test/path",
			wantCached:  true,
			wantErr:     false,
		},
		{
			name: "cache miss - fetch from upstream",
			setupMocks: func(cache *MockCacheService, upstream *MockUpstreamClient) {
				cache.On("Get", ctx, "test:/test/path").Return(nil, false, nil)

				resp := &ProxyResponse{
					Body:       ioutil.NopCloser(bytes.NewBufferString("upstream content")),
					StatusCode: 200,
				}
				upstream.On("Fetch", ctx, "http://example.com/test/path", map[string]string(nil)).
					Return(resp, nil)
			},
			req: ProxyRequest{
				Path: "/test/path",
			},
			upstreamURL: "http://example.com/test/path",
			wantCached:  false,
			wantErr:     false,
		},
		{
			name: "cache error - fallback to upstream",
			setupMocks: func(cache *MockCacheService, upstream *MockUpstreamClient) {
				cache.On("Get", ctx, "test:/test/path").Return(nil, false, fmt.Errorf("cache error"))

				resp := &ProxyResponse{
					Body:       ioutil.NopCloser(bytes.NewBufferString("upstream content")),
					StatusCode: 200,
				}
				upstream.On("Fetch", ctx, "http://example.com/test/path", map[string]string(nil)).
					Return(resp, nil)
			},
			req: ProxyRequest{
				Path: "/test/path",
			},
			upstreamURL: "http://example.com/test/path",
			wantCached:  false,
			wantErr:     false,
		},
		{
			name: "upstream error",
			setupMocks: func(cache *MockCacheService, upstream *MockUpstreamClient) {
				cache.On("Get", ctx, "test:/test/path").Return(nil, false, nil)
				upstream.On("Fetch", ctx, "http://example.com/test/path", map[string]string(nil)).
					Return(nil, fmt.Errorf("upstream error"))
			},
			req: ProxyRequest{
				Path: "/test/path",
			},
			upstreamURL: "http://example.com/test/path",
			wantCached:  false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := &MockCacheService{}
			config := &MockConfigService{}
			upstream := &MockUpstreamClient{}

			tt.setupMocks(cache, upstream)

			service := NewBaseProxyService("test", cache, config, upstream)
			resp, err := service.FetchFromCacheOrUpstream(ctx, tt.req, tt.upstreamURL)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, tt.wantCached, resp.Cached)
			}

			cache.AssertExpectations(t)
			upstream.AssertExpectations(t)
		})
	}
}

func TestBaseProxyService_CacheResponse(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		setupMocks func(*MockCacheService)
		req        ProxyRequest
		resp       *ProxyResponse
		wantErr    bool
	}{
		{
			name: "successful cache",
			setupMocks: func(cache *MockCacheService) {
				cache.On("Put", ctx, "test:/test/path", mock.Anything).Return(nil)
			},
			req: ProxyRequest{
				Path: "/test/path",
			},
			resp: &ProxyResponse{
				Body:       ioutil.NopCloser(bytes.NewBufferString("content to cache")),
				StatusCode: 200,
			},
			wantErr: false,
		},
		{
			name: "cache error",
			setupMocks: func(cache *MockCacheService) {
				cache.On("Put", ctx, "test:/test/path", mock.Anything).
					Return(fmt.Errorf("cache write error"))
			},
			req: ProxyRequest{
				Path: "/test/path",
			},
			resp: &ProxyResponse{
				Body:       ioutil.NopCloser(bytes.NewBufferString("content")),
				StatusCode: 200,
			},
			wantErr: true,
		},
		{
			name:       "nil response body",
			setupMocks: func(cache *MockCacheService) {},
			req: ProxyRequest{
				Path: "/test/path",
			},
			resp: &ProxyResponse{
				Body:       nil,
				StatusCode: 200,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := &MockCacheService{}
			config := &MockConfigService{}
			upstream := &MockUpstreamClient{}

			tt.setupMocks(cache)

			service := NewBaseProxyService("test", cache, config, upstream)
			err := service.CacheResponse(ctx, tt.req, tt.resp)

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
			body, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Contains(t, string(body), tt.expectedMsg)
		})
	}
}

func TestBaseProxyService_HandleRequest(t *testing.T) {
	service := &BaseProxyService{proxyType: "base"}
	ctx := context.Background()
	req := ProxyRequest{Path: "/test"}

	resp, err := service.HandleRequest(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 501, resp.StatusCode)

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "not implemented")
}
