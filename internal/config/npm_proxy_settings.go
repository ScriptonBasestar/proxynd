package config

import (
	"path"

	"proxynd/helpers"
)

// NpmProxyServer is exported
// NpmProxyServer provides server functionality
type NpmProxyServer struct {
	Name      string    `yaml:"name,omitempty" validate:"required,min=1,max=100"`
	URL       string    `yaml:"url,omitempty" validate:"required,url"`
	BasicAuth BasicAuth `yaml:"basic_auth,omitempty"`
}

// NpmProxySettings is exported

// NpmProxySettings represents the configuration for npmproxy settings
type NpmProxySettings struct {
	Path      string                      `yaml:"path,omitempty" validate:"required,min=1"`
	UseCache  bool                        `yaml:"use_cache,omitempty" default:"true"`
	UserCache bool                        `yaml:"user_cache,omitempty" default:"false"`
	Proxies   map[string][]NpmProxyServer `yaml:"proxies" validate:"required,min=1,dive,keys,min=1,endkeys,min=1,dive"`
}

// NpmProxy is an alias for NpmProxySettings for backwards compatibility
type NpmProxy = NpmProxySettings

// ConfigExists checks if the configuration file exists
func (cfg *NpmProxySettings) ConfigExists() bool {
	confDir := helpers.GetConfigDir()
	return helpers.FileExists(path.Join(confDir, "npm-proxy.yaml"))
}

// ReadConfig reads the configuration from file
func (cfg *NpmProxySettings) ReadConfig() error {
	confDir := helpers.GetConfigDir()
	if err := helpers.ReadYamlSafe(path.Join(confDir, "npm-proxy.yaml"), cfg); err != nil {
		return err
	}
	return cfg.Validate()
}

// Validate validates the NPM proxy configuration
func (cfg *NpmProxySettings) Validate() error {
	return ValidateStruct(cfg)
}
