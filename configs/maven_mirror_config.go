package configs

import (
	"path"
	"proxynd/helpers"
)

type MavenMirrorServer struct {
	Name        string `yaml:"name"`
	Url         string `yaml:"url,omitempty"`
	Description string `yaml:"description"`
}

type MavenMirrorConfig struct {
	Path     string                       `yaml:"path,omitempty"`
	UseCache bool                         `yaml:"use_cache,omitempty"`
	Mirrors  map[string]MavenMirrorServer `yaml:"mirrors"`
}

func (cfg *MavenMirrorConfig) ReadConfig() {
	confDir := helpers.GetConfigDir()
	helpers.ReadYaml(path.Join(confDir, "maven-mirror.yaml"), cfg)
}
