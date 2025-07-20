package adapters

import (
	"context"
	"fmt"

	configService "proxynd/internal/services/config"
	"proxynd/internal/services/proxy"
)

// ConfigServiceAdapter adapts the centralized config service to the proxy service interface
type ConfigServiceAdapter struct {
	service configService.Service
}

// NewConfigServiceAdapter creates a new config service adapter
func NewConfigServiceAdapter(service configService.Service) *ConfigServiceAdapter {
	return &ConfigServiceAdapter{
		service: service,
	}
}

// GetProxyConfig returns configuration for a specific proxy type
func (a *ConfigServiceAdapter) GetProxyConfig(ctx context.Context, proxyType string) (interface{}, error) {
	switch proxyType {
	case "apt":
		return a.service.GetAptConfig(ctx)
	case "maven":
		return a.service.GetMavenConfig(ctx)
	case "npm":
		return a.service.GetNpmConfig(ctx)
	case "pip":
		return a.service.GetPipConfig(ctx)
	case "yum":
		return a.service.GetYumConfig(ctx)
	case "apk":
		return a.service.GetApkConfig(ctx)
	case "docker":
		return a.service.GetDockerConfig(ctx)
	default:
		return nil, fmt.Errorf("unknown proxy type: %s", proxyType)
	}
}

// GetGlobalConfig returns global configuration
func (a *ConfigServiceAdapter) GetGlobalConfig(ctx context.Context) (interface{}, error) {
	return a.service.GetGlobalConfig(ctx)
}

// ReloadConfig reloads configuration from disk
func (a *ConfigServiceAdapter) ReloadConfig(ctx context.Context) error {
	return a.service.Reload(ctx)
}

// WatchConfig sets up configuration watching with a callback
func (a *ConfigServiceAdapter) WatchConfig(_ func(proxyType string, config interface{})) error {
	// The centralized config service handles validation on load
	// This method is kept for interface compatibility
	return nil
}

// SaveProxyConfig saves a proxy configuration
func (a *ConfigServiceAdapter) SaveProxyConfig(_ string, _ interface{}) error {
	// Not implemented for centralized config service
	// Configuration should be managed through files
	return fmt.Errorf("saving configuration through service is not supported")
}

// SaveGlobalConfig saves the global configuration
func (a *ConfigServiceAdapter) SaveGlobalConfig(_ interface{}) error {
	// Not implemented for centralized config service
	// Configuration should be managed through files
	return fmt.Errorf("saving configuration through service is not supported")
}

// Ensure ConfigServiceAdapter implements the ConfigService interface
var _ proxy.ConfigService = (*ConfigServiceAdapter)(nil)
