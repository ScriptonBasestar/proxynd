package config

import (
	"context"
)

// Repository defines the interface for configuration storage operations
type Repository interface {
	// LoadGlobalConfig loads the global configuration
	LoadGlobalConfig(ctx context.Context) (interface{}, error)

	// LoadProxyConfig loads configuration for a specific proxy type
	LoadProxyConfig(ctx context.Context, proxyType string) (interface{}, error)

	// SaveGlobalConfig saves the global configuration
	SaveGlobalConfig(ctx context.Context, config interface{}) error

	// SaveProxyConfig saves configuration for a specific proxy type
	SaveProxyConfig(ctx context.Context, proxyType string, config interface{}) error

	// ListProxyTypes returns all configured proxy types
	ListProxyTypes(ctx context.Context) ([]string, error)

	// ValidateConfig validates a configuration object
	ValidateConfig(ctx context.Context, proxyType string, config interface{}) error

	// WatchConfig watches for configuration changes
	WatchConfig(ctx context.Context, callback func(proxyType string, config interface{})) error

	// GetConfigPath returns the path to configuration files
	GetConfigPath() string
}
