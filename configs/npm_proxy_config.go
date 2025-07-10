package configs

import (
	"path"

	"proxynd/helpers"
)

type NpmProxyServer struct {
	Name string `yaml:"name,omitempty" validate:"required,min=1,max=100"`
	URL  string `yaml:"url,omitempty" validate:"required,url"`
}

type NpmProxyConfig struct {
	Path      string                      `yaml:"path,omitempty" validate:"required,min=1"`
	UseCache  bool                        `yaml:"use_cache,omitempty" default:"true"`
	UserCache bool                        `yaml:"user_cache,omitempty" default:false`
	Proxies   map[string][]NpmProxyServer `yaml:"proxies" validate:"required,min=1,dive,keys,min=1,endkeys,min=1,dive"`
}

func (cfg *NpmProxyConfig) ConfigExists() bool {
	confDir := helpers.GetConfigDir()
	return helpers.FileExists(path.Join(confDir, "npm-proxy.yaml"))
}

func (cfg *NpmProxyConfig) ReadConfig() error {
	confDir := helpers.GetConfigDir()
	if err := helpers.ReadYamlSafe(path.Join(confDir, "npm-proxy.yaml"), cfg); err != nil {
		return err
	}
	return cfg.Validate()
}

// Validate validates the NPM proxy configuration
func (cfg *NpmProxyConfig) Validate() error {
	// Validate struct tags
	if err := ValidateStruct(cfg); err != nil {
		return err
	}

	// Ensure default registry exists
	if _, exists := cfg.Proxies["default"]; !exists {
		return helpers.NewConfigFieldError("npm-proxy", "default registry configuration is required")
	}

	return nil
}
