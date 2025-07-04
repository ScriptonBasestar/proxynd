package configs

import (
	"path"

	"proxynd/helpers"
)

type AptProxy struct {
	Name string `yaml:"name,omitempty"`
	URL  string `yaml:"url,omitempty"`
}

type AptProxyConfig struct {
	Path      string                `yaml:"path,omitempty"`
	UserCache bool                  `yaml:"user_cache,omitempty" default:false`
	Proxies   map[string][]AptProxy `yaml:"proxies"`
}

func (cfg *AptProxyConfig) ConfigExists() bool {
	confDir := helpers.GetConfigDir()
	return helpers.FileExists(path.Join(confDir, "apt-proxy.yaml"))
}

func (cfg *AptProxyConfig) ReadConfig() {
	confDir := helpers.GetConfigDir()
	helpers.ReadYaml(path.Join(confDir, "apt-proxy.yaml"), cfg)
}
