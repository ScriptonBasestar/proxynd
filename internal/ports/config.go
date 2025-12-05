// Package ports defines the interface contracts for hexagonal architecture.
// Ports represent the boundaries between the domain/application core and external adapters.
package ports

import "proxynd/internal/config"

// ConfigLoader defines the port for configuration loading and management.
// Implementations must be thread-safe and support both initial load and runtime reload.
type ConfigLoader interface {
	// Load loads configuration from the specified path.
	// Returns error if the file doesn't exist, is invalid, or fails validation.
	Load(path string) (*config.RootConfig, error)

	// Reload reloads the configuration from the previously loaded path.
	// Returns error if reload fails or new config is invalid.
	Reload() (*config.RootConfig, error)

	// Validate validates the provided configuration structure.
	// Returns error if validation fails with specific validation errors.
	Validate(cfg *config.RootConfig) error

	// GetCurrent returns the currently loaded configuration.
	// Must be thread-safe for concurrent access.
	// Returns nil if no configuration has been loaded yet.
	GetCurrent() *config.RootConfig

	// GetPath returns the path to the currently loaded configuration file.
	GetPath() string
}

// ConfigProvider provides read-only access to configuration with observer pattern support.
// This interface separates config access from config management.
type ConfigProvider interface {
	// GetConfig returns the current configuration.
	// Must be thread-safe for concurrent access.
	GetConfig() *config.RootConfig

	// Subscribe registers an observer to receive config change notifications.
	// Observers are notified when configuration is reloaded.
	Subscribe(observer ConfigObserver)

	// Unsubscribe removes an observer from receiving notifications.
	Unsubscribe(observer ConfigObserver)
}

// ConfigObserver receives notifications when configuration changes.
// Implementations should handle errors gracefully and not block the notification process.
type ConfigObserver interface {
	// OnConfigChange is called when configuration is reloaded.
	// old: previous configuration (nil on first load)
	// new: newly loaded configuration
	// Returns error if the observer cannot handle the config change.
	// The error is logged but doesn't prevent other observers from being notified.
	OnConfigChange(old, new *config.RootConfig) error

	// Name returns the observer's name for logging purposes.
	Name() string
}

// ConfigWatcher watches configuration file for changes and triggers reload.
type ConfigWatcher interface {
	// Start begins watching the configuration file for changes.
	// Returns error if the file watcher cannot be initialized.
	Start() error

	// Stop stops watching for configuration changes.
	// Should cleanup any resources (file watchers, goroutines, etc.)
	Stop() error

	// IsRunning returns true if the watcher is currently active.
	IsRunning() bool
}
