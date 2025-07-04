package configs

import (
	"fmt"
	"path"

	"proxynd/helpers"
)

// MavenProxyConfigLoader provides methods to load Maven proxy configuration without global state
type MavenProxyConfigLoader struct {
	configDir string
}

// NewMavenProxyConfigLoader creates a new Maven proxy config loader
func NewMavenProxyConfigLoader(configDir string) *MavenProxyConfigLoader {
	if configDir == "" {
		configDir = helpers.GetConfigDir()
	}
	return &MavenProxyConfigLoader{
		configDir: configDir,
	}
}

// Load loads the Maven proxy configuration from file
func (l *MavenProxyConfigLoader) Load() (*MavenProxyConfig, error) {
	cfg := &MavenProxyConfig{}
	configPath := path.Join(l.configDir, "maven-proxy.yaml")
	
	if !helpers.FileExists(configPath) {
		// Return empty config if file doesn't exist
		return cfg, nil
	}
	
	if err := helpers.ReadYamlSafe(configPath, cfg); err != nil {
		return nil, fmt.Errorf("failed to read maven proxy config: %w", err)
	}
	
	return cfg, nil
}

// ConfigExists checks if the Maven proxy config file exists
func (l *MavenProxyConfigLoader) ConfigExists() bool {
	return helpers.FileExists(path.Join(l.configDir, "maven-proxy.yaml"))
}