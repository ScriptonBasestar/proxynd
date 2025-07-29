package proxy

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewAptService(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*MockCacheService, *MockConfigService, *MockUpstreamClient)
		wantErr    bool
		errMsg     string
	}{
		{
			name: "successful creation",
			setupMocks: func(_ *MockCacheService, config *MockConfigService, _ *MockUpstreamClient) {
				aptConfig := &config.AptProxyConfig{
					Path:     "/apt",
					UseCache: true,
					Proxies: map[string][]config.AptProxy{
						"default": {
							{Name: "ubuntu", URL: "http://archive.ubuntu.com/ubuntu"},
						},
					},
				}
				config.On("GetProxyConfig", mock.Anything, "apt").Return(aptConfig, nil)
			},
			wantErr: false,
		},
		{
			name: "config load error",
			setupMocks: func(_ *MockCacheService, config *MockConfigService, _ *MockUpstreamClient) {
				config.On("GetProxyConfig", mock.Anything, "apt").Return(nil, fmt.Errorf("config not found"))
			},
			wantErr: true,
			errMsg:  "failed to load apt config",
		},
		{
			name: "invalid config type",
			setupMocks: func(_ *MockCacheService, config *MockConfigService, _ *MockUpstreamClient) {
				config.On("GetProxyConfig", mock.Anything, "apt").Return("invalid type", nil)
			},
			wantErr: true,
			errMsg:  "invalid apt config type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := &MockCacheService{}
			config := &MockConfigService{}
			upstream := &MockUpstreamClient{}

			tt.setupMocks(cache, config, upstream)

			service, err := NewAptService(cache, config, upstream)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, service)
				assert.NotNil(t, service.config)
				assert.Equal(t, "apt", service.GetProxyType())
			}

			cache.AssertExpectations(t)
			config.AssertExpectations(t)
			upstream.AssertExpectations(t)
		})
	}
}

func TestAptService_HandleRequest(t *testing.T) {
	cache := &MockCacheService{}
	config := &MockConfigService{}
	upstream := &MockUpstreamClient{}

	aptConfig := &config.AptProxyConfig{
		Path:     "/apt",
		UseCache: true,
	}
	config.On("GetProxyConfig", mock.Anything, "apt").Return(aptConfig, nil)

	service, err := NewAptService(cache, config, upstream)
	require.NoError(t, err)

	ctx := context.Background()
	req := ProxyRequest{
		Path:      "/apt/dists/focal/Release",
		ProxyType: "apt",
		Method:    "GET",
	}

	// Currently returns not implemented
	resp, err := service.HandleRequest(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 501, resp.StatusCode)
}
