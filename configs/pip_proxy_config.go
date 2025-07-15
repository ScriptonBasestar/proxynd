package configs

import (
	"path"

	"proxynd/helpers"
)

type PipProxyServer struct {
	Name string `yaml:"name,omitempty" validate:"required,min=1,max=100"`
	URL  string `yaml:"url,omitempty" validate:"required,url"`
}

type PipProxyConfig struct {
	Path     string           `yaml:"path,omitempty" validate:"required,min=1"`
	UseCache bool             `yaml:"use_cache,omitempty" default:"true"`
	Proxies  []PipProxyServer `yaml:"proxies" validate:"required,min=1,dive"`
}

func (cfg *PipProxyConfig) ConfigExists() bool {
	confDir := helpers.GetConfigDir()
	return helpers.FileExists(path.Join(confDir, "pip-proxy.yaml"))
}

func (cfg *PipProxyConfig) ReadConfig() error {
	confDir := helpers.GetConfigDir()
	if err := helpers.ReadYamlSafe(path.Join(confDir, "pip-proxy.yaml"), cfg); err != nil {
		return err
	}
	return cfg.Validate()
}

// Validate validates the PIP proxy configuration
func (cfg *PipProxyConfig) Validate() error {
	return ValidateStruct(cfg)
}

// PipProxy is an alias for PipProxyServer for backwards compatibility
type PipProxy = PipProxyServer
