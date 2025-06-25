package configs

import (
	"path"
	"proxynd/helpers"
)

type NpmProxyServer struct {
	Name string `yaml:"name,omitempty"`
	URL  string `yaml:"url,omitempty"`
}

type NpmProxyConfig struct {
	Path      string                      `yaml:"path,omitempty"`
	UserCache bool                        `yaml:"user_cache,omitempty" default:false`
	Proxies   map[string][]NpmProxyServer `yaml:"proxies"`
}

func (cfg *NpmProxyConfig) ConfigExists() bool {
	confDir := helpers.GetConfigDir()
	return helpers.FileExists(path.Join(confDir, "npm-proxy.yaml"))
}

func (cfg *NpmProxyConfig) ReadConfig() {
	confDir := helpers.GetConfigDir()
	helpers.ReadYaml(path.Join(confDir, "npm-proxy.yaml"), cfg)
}
