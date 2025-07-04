package configs

import (
	"path"

	"proxynd/helpers"
)

type GemProxy struct {
	Name string `yaml:"name,omitempty"`
	URL  string `yaml:"url,omitempty"`
}

type GemProxyConfig struct {
	Path      string                `yaml:"path,omitempty"`
	UserCache bool                  `yaml:"user_cache,omitempty" default:false`
	Proxies   map[string][]AptProxy `yaml:"proxies"`
}

func (cfg *GemProxyConfig) ConfigExists() bool {
	confDir := helpers.GetConfigDir()
	return helpers.FileExists(path.Join(confDir, "gem-proxy.yaml"))
}

func (cfg *GemProxyConfig) ReadConfig() {
	confDir := helpers.GetConfigDir()
	helpers.ReadYaml(path.Join(confDir, "gem-proxy.yaml"), cfg)
}
