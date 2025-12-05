// Package config provides configuration loading adapters implementing the ports.ConfigLoader interface.
package config

import (
	"fmt"
	"sync"

	"proxynd/internal/config"
	"proxynd/internal/logging"
	"proxynd/internal/ports"
)

// unifiedConfigLoader implements ports.ConfigLoader using the existing config.UnifiedConfigLoader.
// This adapter bridges the hexagonal architecture port with the existing config loading implementation.
type unifiedConfigLoader struct {
	configPath string
	current    *config.RootConfig
	loader     *config.UnifiedConfigLoader
	observers  []ports.ConfigObserver
	mu         sync.RWMutex
	logger     logging.Logger
}

// NewUnifiedConfigLoader creates a new configuration loader adapter.
// configPath: path to the configuration file (e.g., "/etc/proxynd/config.yaml")
// Returns error if the config directory doesn't exist or CONFIG_DIR is not set.
func NewUnifiedConfigLoader(configPath string) (ports.ConfigLoader, error) {
	logger := logging.GetLogger()

	// Extract config directory from path
	// The existing UnifiedConfigLoader expects a directory, not a file path
	// We'll use the config.LoadRootConfig function instead which takes a file path

	loader := &unifiedConfigLoader{
		configPath: configPath,
		observers:  make([]ports.ConfigObserver, 0),
		logger:     logger,
	}

	// Load initial configuration
	cfg, err := config.LoadRootConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load initial config from %s: %w", configPath, err)
	}

	loader.current = cfg
	logger.Info("Configuration loaded successfully", logging.F("path", configPath))

	return loader, nil
}

// Load loads configuration from the specified path.
func (l *unifiedConfigLoader) Load(path string) (*config.RootConfig, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logger.Info("Loading configuration", logging.F("path", path))

	cfg, err := config.LoadRootConfig(path)
	if err != nil {
		l.logger.Error("Failed to load configuration",
			logging.F("path", path),
			logging.F("error", err))
		return nil, fmt.Errorf("failed to load config from %s: %w", path, err)
	}

	// Validate the loaded configuration
	if err := l.validateConfig(cfg); err != nil {
		l.logger.Error("Configuration validation failed",
			logging.F("path", path),
			logging.F("error", err))
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Update current config and path
	oldConfig := l.current
	l.current = cfg
	l.configPath = path

	// Notify observers of the config change
	l.notifyObservers(oldConfig, cfg)

	l.logger.Info("Configuration loaded and validated successfully",
		logging.F("path", path))

	return cfg, nil
}

// Reload reloads the configuration from the previously loaded path.
func (l *unifiedConfigLoader) Reload() (*config.RootConfig, error) {
	l.mu.RLock()
	path := l.configPath
	l.mu.RUnlock()

	l.logger.Info("Reloading configuration", logging.F("path", path))

	return l.Load(path)
}

// Validate validates the provided configuration structure.
func (l *unifiedConfigLoader) Validate(cfg *config.RootConfig) error {
	return l.validateConfig(cfg)
}

// GetCurrent returns the currently loaded configuration.
// Thread-safe for concurrent access.
func (l *unifiedConfigLoader) GetCurrent() *config.RootConfig {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.current
}

// GetPath returns the path to the currently loaded configuration file.
func (l *unifiedConfigLoader) GetPath() string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.configPath
}

// validateConfig performs validation on the configuration.
func (l *unifiedConfigLoader) validateConfig(cfg *config.RootConfig) error {
	if cfg == nil {
		return fmt.Errorf("configuration is nil")
	}

	// Use the existing validation function
	errors := config.ValidateRootConfig(cfg)
	if len(errors) > 0 {
		// Convert validation errors to a single error message
		return fmt.Errorf("configuration validation failed with %d errors: %v", len(errors), errors[0])
	}

	return nil
}

// notifyObservers notifies all registered observers of a configuration change.
// This is called with the write lock already held.
func (l *unifiedConfigLoader) notifyObservers(oldConfig, newConfig *config.RootConfig) {
	for _, observer := range l.observers {
		if err := observer.OnConfigChange(oldConfig, newConfig); err != nil {
			l.logger.Error("Observer failed to handle config change",
				logging.F("observer", observer.Name()),
				logging.F("error", err))
			// Continue notifying other observers even if one fails
		} else {
			l.logger.Debug("Observer handled config change successfully",
				logging.F("observer", observer.Name()))
		}
	}
}

// Subscribe registers an observer to receive config change notifications.
func (l *unifiedConfigLoader) Subscribe(observer ports.ConfigObserver) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.observers = append(l.observers, observer)
	l.logger.Info("Config observer registered", logging.F("observer", observer.Name()))
}

// Unsubscribe removes an observer from receiving notifications.
func (l *unifiedConfigLoader) Unsubscribe(observer ports.ConfigObserver) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for i, obs := range l.observers {
		if obs == observer {
			// Remove observer by replacing with last element and truncating
			l.observers[i] = l.observers[len(l.observers)-1]
			l.observers = l.observers[:len(l.observers)-1]
			l.logger.Info("Config observer unregistered", logging.F("observer", observer.Name()))
			return
		}
	}
}

// GetConfig returns the current configuration (implements ports.ConfigProvider).
func (l *unifiedConfigLoader) GetConfig() *config.RootConfig {
	return l.GetCurrent()
}

// Verify that unifiedConfigLoader implements the required interfaces at compile time
var _ ports.ConfigLoader = (*unifiedConfigLoader)(nil)
var _ ports.ConfigProvider = (*unifiedConfigLoader)(nil)
