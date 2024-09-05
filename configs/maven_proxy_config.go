package configs

import (
	"path"
	"proxynd/helpers"
)

type MavenProxyServer struct {
	Id          string `yaml:"id,omitempty"`
	Name        string `yaml:"name"`
	Url         string `yaml:"url,omitempty"`
	Description string `yaml:"description"`
}

type MavenProxyConfig struct {
	Path    string             `yaml:"path,omitempty"`
	Proxies []MavenProxyServer `yaml:"proxies"`
}

func (cfg *MavenProxyConfig) ConfigExists() bool {
	confDir := helpers.GetConfigDir()
	return helpers.FileExists(path.Join(confDir, "maven-proxy.yaml"))
}

func (cfg *MavenProxyConfig) ReadConfig() {
	confDir := helpers.GetConfigDir()
	helpers.ReadYaml(path.Join(confDir, "maven-proxy.yaml"), cfg)
}
