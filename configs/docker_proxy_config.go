package configs

import (
	"path"

	"proxynd/helpers"
)

// DockerProxyServer is exported
// DockerProxyServer provides server functionality
type DockerProxyServer struct {
	Name string     `yaml:"name,omitempty" validate:"required,min=1,max=100"`
	URL  string     `yaml:"url,omitempty" validate:"required,url"`
	Auth DockerAuth `yaml:"auth,omitempty" validate:"dive"`
	// DockerAuth is exported
}

// DockerAuth represents authentication credentials for Docker registry
type DockerAuth struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

// DockerProxyConfig represents the configuration for Docker proxy settings
type DockerProxyConfig struct {
	Path       string                               `yaml:"path,omitempty" validate:"required,min=1"`
	UseCache   bool                                 `yaml:"use_cache,omitempty" default:"true"`
	Proxies    []DockerProxyServer                  `yaml:"proxies" validate:"required,min=1,dive"`
	Registries map[string]DockerProxyRegistryConfig `yaml:"registries,omitempty" validate:"dive,keys,min=1,endkeys,dive"`
}

// DockerProxyRegistryConfig represents configuration for a specific Docker registry
type DockerProxyRegistryConfig struct {
	URL      string     `yaml:"url" validate:"required,url"`
	Auth     DockerAuth `yaml:"auth,omitempty" validate:"dive"`
	Enabled  bool       `yaml:"enabled,omitempty" default:"true"`
	UseCache bool       `yaml:"use_cache,omitempty" default:"true"`
}

// ConfigExists checks if the Docker proxy configuration file exists
func (cfg *DockerProxyConfig) ConfigExists() bool {
	confDir := helpers.GetConfigDir()
	return helpers.FileExists(path.Join(confDir, "docker-proxy.yaml"))
}

// ReadConfig reads and validates the Docker proxy configuration
func (cfg *DockerProxyConfig) ReadConfig() error {
	confDir := helpers.GetConfigDir()
	if err := helpers.ReadYamlSafe(path.Join(confDir, "docker-proxy.yaml"), cfg); err != nil {
		return err
	}
	return cfg.Validate()
}

// Validate validates the Docker proxy configuration
func (cfg *DockerProxyConfig) Validate() error {
	return ValidateStruct(cfg)
}
