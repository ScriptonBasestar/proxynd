package adapters

import (
	"context"
	"fmt"
	"time"

	"proxynd/internal/repositories/config"
	"proxynd/internal/services/proxy"
)

// ConfigAdapter adapts the config repository to the service interface
type ConfigAdapter struct {
	repo config.Repository
}

// NewConfigAdapter creates a new config adapter
func NewConfigAdapter(repo config.Repository) *ConfigAdapter {
	return &ConfigAdapter{
		repo: repo,
	}
}

// GetProxyConfig returns configuration for a specific proxy type
func (a *ConfigAdapter) GetProxyConfig(ctx context.Context, proxyType string) (interface{}, error) {
	return a.repo.LoadProxyConfig(ctx, proxyType)
}

// GetGlobalConfig returns global configuration
func (a *ConfigAdapter) GetGlobalConfig(ctx context.Context) (interface{}, error) {
	return a.repo.LoadGlobalConfig(ctx)
}

// ReloadConfig reloads configuration from disk
func (a *ConfigAdapter) ReloadConfig(_ context.Context) error {
	// The file repository automatically reloads on access
	// This could trigger a manual reload if needed
	return nil
}

// WatchConfig sets up configuration watching with a callback
func (a *ConfigAdapter) WatchConfig(callback func(proxyType string, config interface{})) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return a.repo.WatchConfig(ctx, callback)
}

// SaveProxyConfig saves a proxy configuration
func (a *ConfigAdapter) SaveProxyConfig(proxyType string, config interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Validate before saving
	if err := a.repo.ValidateConfig(ctx, proxyType, config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	return a.repo.SaveProxyConfig(ctx, proxyType, config)
}

// SaveGlobalConfig saves the global configuration
func (a *ConfigAdapter) SaveGlobalConfig(config interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Validate before saving
	if err := a.repo.ValidateConfig(ctx, "global", config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	return a.repo.SaveGlobalConfig(ctx, config)
}

// Ensure ConfigAdapter implements the ConfigService interface
var _ proxy.ConfigService = (*ConfigAdapter)(nil)
