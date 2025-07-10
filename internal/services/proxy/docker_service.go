package proxy

import (
	"context"
	"fmt"

	"proxynd/configs"
)

// DockerService handles Docker registry proxy requests
type DockerService struct {
	*BaseProxyService
	config *configs.DockerProxyConfig
}

// NewDockerService creates a new Docker proxy service
func NewDockerService(
	cache CacheService,
	configService ConfigService,
	upstreamClient UpstreamClient,
) (*DockerService, error) {
	base := NewBaseProxyService("docker", cache, configService, upstreamClient)

	// Load Docker-specific configuration
	configInterface, err := configService.GetProxyConfig("docker")
	if err != nil {
		return nil, fmt.Errorf("failed to load docker config: %w", err)
	}

	dockerConfig, ok := configInterface.(*configs.DockerProxyConfig)
	if !ok {
		return nil, fmt.Errorf("invalid docker config type")
	}

	return &DockerService{
		BaseProxyService: base,
		config:           dockerConfig,
	}, nil
}

// HandleRequest processes a Docker proxy request
func (s *DockerService) HandleRequest(ctx context.Context, req ProxyRequest) (*ProxyResponse, error) {
	// TODO: Implement Docker-specific logic
	// This is a stub implementation
	return s.HandleError(fmt.Errorf("Docker service not yet implemented"), 501), nil
}
