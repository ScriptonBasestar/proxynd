package configs

import (
	"path"
	"proxynd/helpers"
)

type AptProxy struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

type AptProxyConfig struct {
	Path      string                `json:"path,omitempty"`
	UserCache bool                  `json:"user_cache,omitempty" default:false`
	Proxies   map[string][]AptProxy `json:"proxies"`
}

func (cfg *AptProxyConfig) ConfigExists() bool {
	confDir := helpers.GetConfigDir()
	return helpers.FileExists(path.Join(confDir, "apt-proxy.yaml"))
}

func (cfg *AptProxyConfig) ReadConfig() {
	confDir := helpers.GetConfigDir()
	helpers.ReadYaml(path.Join(confDir, "apt-proxy.yaml"), cfg)
}
