package configs

import (
	"path"

	"proxynd/helpers"
)

type AptProxy struct {
	Name string `yaml:"name,omitempty" validate:"required,min=1,max=100"`
	URL  string `yaml:"url,omitempty" validate:"required,url"`
}

type AptProxyConfig struct {
	Path      string                `yaml:"path,omitempty" validate:"required,min=1"`
	UseCache  bool                  `yaml:"use_cache,omitempty" default:"true"`
	UserCache bool                  `yaml:"user_cache,omitempty" default:false`
	Proxies   map[string][]AptProxy `yaml:"proxies" validate:"required,min=1,dive,keys,min=1,endkeys,min=1,dive"`
}

func (cfg *AptProxyConfig) ConfigExists() bool {
	confDir := helpers.GetConfigDir()
	return helpers.FileExists(path.Join(confDir, "apt-proxy.yaml"))
}

func (cfg *AptProxyConfig) ReadConfig() error {
	confDir := helpers.GetConfigDir()
	if err := helpers.ReadYamlSafe(path.Join(confDir, "apt-proxy.yaml"), cfg); err != nil {
		return err
	}
	return cfg.Validate()
}

// Validate validates the APT proxy configuration
func (cfg *AptProxyConfig) Validate() error {
	return ValidateStruct(cfg)
}
