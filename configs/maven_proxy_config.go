package configs

import (
	"path"

	"proxynd/helpers"
)

type MavenProxyServer struct {
	Id          string `yaml:"id,omitempty" validate:"omitempty,min=1,max=100"`
	Name        string `yaml:"name" validate:"required,min=1,max=100"`
	Url         string `yaml:"url,omitempty" validate:"required,url"`
	Description string `yaml:"description" validate:"omitempty,max=500"`
}

type MavenProxyConfig struct {
	Path     string             `yaml:"path,omitempty" validate:"required,min=1"`
	UseCache bool               `yaml:"use_cache,omitempty" default:"true"`
	Proxies  []MavenProxyServer `yaml:"proxies" validate:"required,min=1,dive"`
}

func (cfg *MavenProxyConfig) ConfigExists() bool {
	confDir := helpers.GetConfigDir()
	return helpers.FileExists(path.Join(confDir, "maven-proxy.yaml"))
}

func (cfg *MavenProxyConfig) ReadConfig() error {
	confDir := helpers.GetConfigDir()
	if err := helpers.ReadYamlSafe(path.Join(confDir, "maven-proxy.yaml"), cfg); err != nil {
		return err
	}
	return cfg.Validate()
}

// Validate validates the Maven proxy configuration
func (cfg *MavenProxyConfig) Validate() error {
	return ValidateStruct(cfg)
}
