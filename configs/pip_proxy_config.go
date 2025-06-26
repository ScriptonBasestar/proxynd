package configs

import (
	"path"
	"proxynd/helpers"
)

type PipProxyServer struct {
	Name string `yaml:"name,omitempty"`
	URL  string `yaml:"url,omitempty"`
}

type PipProxyConfig struct {
	Path     string           `yaml:"path,omitempty"`
	UseCache bool             `yaml:"use_cache,omitempty" default:"true"`
	Proxies  []PipProxyServer `yaml:"proxies"`
}

func (cfg *PipProxyConfig) ConfigExists() bool {
	confDir := helpers.GetConfigDir()
	return helpers.FileExists(path.Join(confDir, "pip-proxy.yaml"))
}

func (cfg *PipProxyConfig) ReadConfig() {
	confDir := helpers.GetConfigDir()
	helpers.ReadYaml(path.Join(confDir, "pip-proxy.yaml"), cfg)
}
