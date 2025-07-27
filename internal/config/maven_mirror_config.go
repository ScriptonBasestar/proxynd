package config

import (
	"path"

	"proxynd/helpers"
)

// MavenMirrorServer is exported
// MavenMirrorServer provides server functionality
type MavenMirrorServer struct {
	Name        string `yaml:"name"`
	URL         string `yaml:"url,omitempty"`
	Description string `yaml:"description"`
	// MavenMirrorConfig is exported
}

// MavenMirrorConfig represents the configuration for Maven mirror settings
type MavenMirrorConfig struct {
	Path     string                       `yaml:"path,omitempty"`
	UseCache bool                         `yaml:"use_cache,omitempty"`
	Mirrors  map[string]MavenMirrorServer `yaml:"mirrors"`
}

// ReadConfig reads the Maven mirror configuration from the YAML file
func (cfg *MavenMirrorConfig) ReadConfig() {
	confDir := helpers.GetConfigDir()
	helpers.ReadYaml(path.Join(confDir, "maven-mirror.yaml"), cfg)
}
