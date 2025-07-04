package configs

import (
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"proxynd/helpers"
)

// GlobalConfigLoader provides methods to load global configuration without global state
type GlobalConfigLoader struct {
	configDir string
}

// NewGlobalConfigLoader creates a new global config loader
func NewGlobalConfigLoader(configDir string) *GlobalConfigLoader {
	if configDir == "" {
		configDir = helpers.GetConfigDir()
	}
	return &GlobalConfigLoader{
		configDir: configDir,
	}
}

// Load loads the global configuration from file
func (l *GlobalConfigLoader) Load() (*GlobalConfig, error) {
	cfg := &GlobalConfig{}
	configPath := path.Join(l.configDir, "global.yaml")
	
	if !helpers.FileExists(configPath) {
		// Return default config if file doesn't exist
		return cfg, nil
	}
	
	if err := helpers.ReadYamlSafe(configPath, cfg); err != nil {
		return nil, fmt.Errorf("failed to read global config: %w", err)
	}
	
	return cfg, nil
}

// ConfigExists checks if the global config file exists
func (l *GlobalConfigLoader) ConfigExists() bool {
	return helpers.FileExists(path.Join(l.configDir, "global.yaml"))
}

// The following methods remain the same as they are on the struct itself
// They don't access global state, just operate on the struct's data