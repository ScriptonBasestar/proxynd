package proxy

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"proxynd/configs"
)

func TestNewProxyServiceFactory(t *testing.T) {
	cache := &MockCacheService{}
	config := &MockConfigService{}
	upstream := &MockUpstreamClient{}

	factory := NewProxyServiceFactory(cache, config, upstream)

	assert.NotNil(t, factory)
	assert.Equal(t, cache, factory.cache)
	assert.Equal(t, config, factory.configService)
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
				aptConfig := &configs.AptProxyConfig{
					Path:     "/apt",
					UseCache: true,
				}
				config.On("GetProxyConfig", "apt").Return(aptConfig, nil)
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
				mavenConfig := &configs.MavenProxyConfig{
					Path:     "/maven",
					UseCache: true,
				}
				config.On("GetProxyConfig", "maven").Return(mavenConfig, nil)
			},
			wantErr: false,
			checkType: func(s ProxyService) bool {
				_, ok := s.(*MavenService)
				return ok
			},
		},
		{
			name:      "create npm service",
			proxyType: "npm",
			setupMocks: func(config *MockConfigService) {
				npmConfig := &configs.NpmProxyConfig{
					Path:     "/npm",
					UseCache: true,
				}
				config.On("GetProxyConfig", "npm").Return(npmConfig, nil)
			},
			wantErr: false,
			checkType: func(s ProxyService) bool {
				_, ok := s.(*NpmService)
				return ok
			},
		},
		{
			name:      "create docker service",
			proxyType: "docker",
			setupMocks: func(config *MockConfigService) {
				dockerConfig := &configs.DockerProxyConfig{
					Path:     "/docker",
					UseCache: true,
				}
				config.On("GetProxyConfig", "docker").Return(dockerConfig, nil)
			},
			wantErr: false,
			checkType: func(s ProxyService) bool {
				_, ok := s.(*DockerService)
				return ok
			},
		},
		{
			name:      "create pip service",
			proxyType: "pip",
			setupMocks: func(config *MockConfigService) {
				pipConfig := &configs.PipProxyConfig{
					Path:     "/pip",
					UseCache: true,
				}
				config.On("GetProxyConfig", "pip").Return(pipConfig, nil)
			},
			wantErr: false,
			checkType: func(s ProxyService) bool {
				_, ok := s.(*OtherProxyService)
				return ok && s.GetProxyType() == "pip"
			},
		},
		{
			name:      "unknown proxy type",
			proxyType: "unknown",
			setupMocks: func(config *MockConfigService) {
				// No setup needed
			},
			wantErr: true,
			errMsg:  "unsupported proxy type: unknown",
		},
		{
			name:      "empty proxy type",
			proxyType: "",
			setupMocks: func(config *MockConfigService) {
				// No setup needed
			},
			wantErr: true,
			errMsg:  "unsupported proxy type: ",
		},
		{
			name:      "config load error",
			proxyType: "apt",
			setupMocks: func(config *MockConfigService) {
				config.On("GetProxyConfig", "apt").Return(nil, fmt.Errorf("config error"))
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

	proxyTypes := []string{"apt", "maven", "npm", "docker", "pip", "yum", "apk", "helm"}

	for _, proxyType := range proxyTypes {
		t.Run(proxyType, func(t *testing.T) {
			cache := &MockCacheService{}
			config := &MockConfigService{}
			upstream := &MockUpstreamClient{}

			// Setup config mock based on proxy type
			switch proxyType {
			case "apt":
				config.On("GetProxyConfig", proxyType).Return(&configs.AptProxyConfig{}, nil)
			case "maven":
				config.On("GetProxyConfig", proxyType).Return(&configs.MavenProxyConfig{}, nil)
			case "npm":
				config.On("GetProxyConfig", proxyType).Return(&configs.NpmProxyConfig{}, nil)
			case "docker":
				config.On("GetProxyConfig", proxyType).Return(&configs.DockerProxyConfig{}, nil)
			case "pip":
				config.On("GetProxyConfig", proxyType).Return(&configs.PipProxyConfig{}, nil)
			case "yum":
				config.On("GetProxyConfig", proxyType).Return(&configs.YumProxyConfig{}, nil)
			case "apk":
				config.On("GetProxyConfig", proxyType).Return(&configs.ApkProxyConfig{}, nil)
			default:
				// For helm and others that use generic config
				config.On("GetProxyConfig", proxyType).Return(struct{}{}, nil)
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
