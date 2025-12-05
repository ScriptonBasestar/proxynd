package config

import (
	"proxynd/internal/config"
	"proxynd/internal/ports"
)

// hotReloadAdapter adapts the config.HotReloadManager to implement ports.ConfigObserver.
// This allows the existing hot reload system to work with the new hexagonal architecture.
type hotReloadAdapter struct {
	manager *config.HotReloadManager
}

// NewHotReloadAdapter creates a new adapter that wraps the existing hot reload manager.
func NewHotReloadAdapter(manager *config.HotReloadManager) ports.ConfigObserver {
	return &hotReloadAdapter{
		manager: manager,
	}
}

// OnConfigChange implements ports.ConfigObserver by delegating to the hot reload manager.
func (a *hotReloadAdapter) OnConfigChange(old, new *config.RootConfig) error {
	// UnifiedConfig is an alias for RootConfig, so direct pass-through works
	return a.manager.OnConfigChange(old, new)
}

// Name implements ports.ConfigObserver.
func (a *hotReloadAdapter) Name() string {
	return "HotReloadAdapter"
}

// Verify that hotReloadAdapter implements the required interface at compile time
var _ ports.ConfigObserver = (*hotReloadAdapter)(nil)
