package proxy

import (
	"context"
	"fmt"

	"proxynd/internal/config"
)

// NpmService handles NPM repository proxy requests
type NpmService struct {
	*BaseProxyService
	config *config.NpmProxyConfig
}

// NewNpmService creates a new NPM proxy service
func NewNpmService(
	cache CacheService,
	configService ConfigService,
	upstreamClient UpstreamClient,
) (*NpmService, error) {
	base := NewBaseProxyService("npm", cache, configService, upstreamClient)

	// Load NPM-specific configuration
	configInterface, err := configService.GetProxyConfig(context.Background(), "npm")
	if err != nil {
		return nil, fmt.Errorf("failed to load npm config: %w", err)
	}

	npmConfig, ok := configInterface.(*config.NpmProxyConfig)
	if !ok {
		return nil, fmt.Errorf("invalid npm config type")
	}

	return &NpmService{
		BaseProxyService: base,
		config:           npmConfig,
	}, nil
}

// HandleRequest processes an NPM proxy request
func (s *NpmService) HandleRequest(_ context.Context, _ ProxyRequest) (*ProxyResponse, error) {
	// TODO: Implement NPM-specific logic
	// This is a stub implementation
	return s.HandleError(fmt.Errorf("NPM service not yet implemented"), 501), nil
}
