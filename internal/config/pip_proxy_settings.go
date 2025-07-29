package config

import (
	"path"

	"proxynd/helpers"
)

// PipProxyServer is exported
// PipProxyServer provides server functionality
type PipProxyServer struct {
	Name string `yaml:"name,omitempty" validate:"required,min=1,max=100"`
	URL  string `yaml:"url,omitempty" validate:"required,url"`
	// PipProxySettings is exported
}

// PipProxySettings is exported

// PipProxySettings represents the configuration for pipproxy settings
type PipProxySettings struct {
	Path     string           `yaml:"path,omitempty" validate:"required,min=1"`
	UseCache bool             `yaml:"use_cache,omitempty" default:"true"`
	Proxies  []PipProxyServer `yaml:"proxies" validate:"required,min=1,dive"`
}

// ConfigExists checks if the configuration file exists
func (cfg *PipProxySettings) ConfigExists() bool {
	confDir := helpers.GetConfigDir()
	return helpers.FileExists(path.Join(confDir, "pip-proxy.yaml"))
}

// ReadConfig reads the configuration from file
func (cfg *PipProxySettings) ReadConfig() error {
	confDir := helpers.GetConfigDir()
	if err := helpers.ReadYamlSafe(path.Join(confDir, "pip-proxy.yaml"), cfg); err != nil {
		return err
	}
	return cfg.Validate()
}

// Validate validates the PIP proxy configuration
func (cfg *PipProxySettings) Validate() error {
	return ValidateStruct(cfg)
}

// PipProxy is an alias for PipProxyServer for backwards compatibility
type PipProxy = PipProxyServer
