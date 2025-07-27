package config

import (
	"path"

	"proxynd/helpers"
)

// AptMirror is exported
// AptMirror represents a data structure
type AptMirror struct {
	URL string `yaml:"url,omitempty"`
	// AptMirrors is exported
}

// AptMirrors represents the collection of apt mirrors for different distributions
type AptMirrors struct {
	Ubuntu map[string]AptMirror `yaml:"ubuntu"`
	Debian map[string]AptMirror `yaml:"debian"`
}

// AptMirrorConfig represents the configuration for APT mirror settings
type AptMirrorConfig struct {
	Path    string     `yaml:"path,omitempty"`
	Mirrors AptMirrors `yaml:"mirrors"`
}

// ReadConfig reads the APT mirror configuration from the YAML file
func (cfg *AptMirrorConfig) ReadConfig() {
	confDir := helpers.GetConfigDir()
	helpers.ReadYaml(path.Join(confDir, "apt-mirror.yaml"), cfg)
}
