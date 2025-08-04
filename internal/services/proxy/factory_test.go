package proxy

import (
	"context"
	"fmt"
	"testing"

	configpkg "proxynd/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Test constants for proxy types
const (
	proxyTypeNpm    = "npm"
	proxyTypeDocker = "docker"
)

func TestNewProxyServiceFactory(t *testing.T) {
	cache := &MockCacheService{}
	config := &MockConfigService{}
	upstream := &MockUpstreamClient{}

	factory := NewProxyServiceFactory(cache, config, upstream)

	assert.NotNil(t, factory)
	assert.Equal(t, cache, factory.cache)
	assert.Equal(t, config, factory.config)
	assert.Equal(t, upstream, factory.upstreamClient)
}

func TestProxyServiceFactory_CreateProxyService(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		proxyType  string
		setupMocks func(*MockConfigService)
		wantErr    bool
		errMsg     string
		checkType  func(ProxyService) bool
	}{
		{
			name:      "create apt service",
			proxyType: "apt",
			setupMocks: func(config *MockConfigService) {
				aptConfig := &configpkg.AptProxyConfig{
					Path:     "/apt",
					UseCache: true,
				}
				config.On("GetProxyConfig", mock.Anything, "apt").Return(aptConfig, nil)
			},
			wantErr: false,
			checkType: func(s ProxyService) bool {
				_, ok := s.(*AptService)
				return ok
			},
		},
		{
			name:      "create maven service",
			proxyType: "maven",
			setupMocks: func(config *MockConfigService) {
				mavenConfig := &configpkg.MavenProxySettings{
					Path:     "/maven",
					UseCache: true,
				}
				config.On("GetProxyConfig", mock.Anything, "maven").Return(mavenConfig, nil)
			},
			wantErr: false,
			checkType: func(s ProxyService) bool {
				_, ok := s.(*MavenService)
				return ok
			},
		},
		{
			name:      "create npm service",
			proxyType: proxyTypeNpm,
			setupMocks: func(config *MockConfigService) {
				npmConfig := &configpkg.NpmProxySettings{
					Path:     "/npm",
					UseCache: true,
				}
				config.On("GetProxyConfig", mock.Anything, proxyTypeNpm).Return(npmConfig, nil)
			},
			wantErr: false,
			checkType: func(s ProxyService) bool {
				return s.GetProxyType() == proxyTypeNpm
			},
		},
		{
			name:      "create docker service",
			proxyType: proxyTypeDocker,
			setupMocks: func(config *MockConfigService) {
				dockerConfig := &configpkg.DockerProxySettings{
					Path:     "/docker",
					UseCache: true,
				}
				config.On("GetProxyConfig", mock.Anything, proxyTypeDocker).Return(dockerConfig, nil)
			},
			wantErr: false,
			checkType: func(s ProxyService) bool {
				return s.GetProxyType() == proxyTypeDocker
			},
		},
		{
			name:      "unknown proxy type",
			proxyType: "unknown",
			setupMocks: func(_ *MockConfigService) {
				// No setup needed
			},
			wantErr: true,
			errMsg:  "unsupported proxy type: unknown",
		},
		{
			name:      "empty proxy type",
			proxyType: "",
			setupMocks: func(_ *MockConfigService) {
				// No setup needed
			},
			wantErr: true,
			errMsg:  "unsupported proxy type: ",
		},
		{
			name:      "config load error",
			proxyType: "apt",
			setupMocks: func(config *MockConfigService) {
				config.On("GetProxyConfig", mock.Anything, "apt").Return(nil, fmt.Errorf("config error"))
			},
			wantErr: true,
			errMsg:  "failed to load apt config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := &MockCacheService{}
			config := &MockConfigService{}
			upstream := &MockUpstreamClient{}

			tt.setupMocks(config)

			factory := NewProxyServiceFactory(cache, config, upstream)
			service, err := factory.CreateProxyService(ctx, tt.proxyType)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, service)
				assert.Equal(t, tt.proxyType, service.GetProxyType())
				if tt.checkType != nil {
					assert.True(t, tt.checkType(service), "service type mismatch")
				}
			}

			config.AssertExpectations(t)
		})
	}
}

// Table-driven test for all supported proxy types
func TestProxyServiceFactory_AllProxyTypes(t *testing.T) {
	ctx := context.Background()

	proxyTypes := []string{"apt", "maven", proxyTypeNpm, proxyTypeDocker}

	for _, proxyType := range proxyTypes {
		t.Run(proxyType, func(t *testing.T) {
			cache := &MockCacheService{}
			config := &MockConfigService{}
			upstream := &MockUpstreamClient{}

			// Setup config mock based on proxy type
			switch proxyType {
			case "apt":
				config.On("GetProxyConfig", mock.Anything, proxyType).Return(&configpkg.AptProxyConfig{}, nil)
			case "maven":
				config.On("GetProxyConfig", mock.Anything, proxyType).Return(&configpkg.MavenProxySettings{}, nil)
			case proxyTypeNpm:
				config.On("GetProxyConfig", mock.Anything, proxyType).Return(&configpkg.NpmProxySettings{}, nil)
			case proxyTypeDocker:
				config.On("GetProxyConfig", mock.Anything, proxyType).Return(&configpkg.DockerProxySettings{}, nil)
			case "pip":
				config.On("GetProxyConfig", mock.Anything, proxyType).Return(struct{}{}, nil)
			case "yum":
				config.On("GetProxyConfig", mock.Anything, proxyType).Return(struct{}{}, nil)
			case "apk":
				config.On("GetProxyConfig", mock.Anything, proxyType).Return(struct{}{}, nil)
			default:
				// For helm and others that use generic config
				config.On("GetProxyConfig", mock.Anything, proxyType).Return(struct{}{}, nil)
			}

			factory := NewProxyServiceFactory(cache, config, upstream)
			service, err := factory.CreateProxyService(ctx, proxyType)

			assert.NoError(t, err)
			assert.NotNil(t, service)
			assert.Equal(t, proxyType, service.GetProxyType())

			config.AssertExpectations(t)
		})
	}
}
