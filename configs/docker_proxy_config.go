package configs

import (
	"path"
	"proxynd/helpers"
)

type DockerProxyServer struct {
	Name string `yaml:"name,omitempty"`
	URL  string `yaml:"url,omitempty"`
	Auth DockerAuth `yaml:"auth,omitempty"`
}

type DockerAuth struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

type DockerProxyConfig struct {
	Path      string              `yaml:"path,omitempty"`
	UseCache  bool                `yaml:"use_cache,omitempty" default:"true"`
	Proxies   []DockerProxyServer `yaml:"proxies"`
}

func (cfg *DockerProxyConfig) ConfigExists() bool {
	confDir := helpers.GetConfigDir()
	return helpers.FileExists(path.Join(confDir, "docker-proxy.yaml"))
}

func (cfg *DockerProxyConfig) ReadConfig() {
	confDir := helpers.GetConfigDir()
	helpers.ReadYaml(path.Join(confDir, "docker-proxy.yaml"), cfg)
}