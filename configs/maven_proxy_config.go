package configs

import (
	"deproxy/helpers"
)

type MavenProxyServer struct {
	Id          string `yaml:"id"`
	Name        string `yaml:"name"`
	Url         string `yaml:"url"`
	Description string `yaml:"description"`
}

type MavenProxyConfig struct {
	Path    string             `yaml:"path"`
	Proxies []MavenProxyServer `yaml:"proxies"`
}

func (cfg *MavenProxyConfig) ReadConfig(path string) {
	helpers.ReadYaml(path, cfg)
}
