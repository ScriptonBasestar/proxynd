package configs

import (
	"deproxy/helpers"
)

type MavenMirrorServer struct {
	Name        string `yaml:"name"`
	Url         string `yaml:"url"`
	Description string `yaml:"description"`
}

type MavenMirrorConfig struct {
	Path     string                       `yaml:"path"`
	UseCache bool                         `yaml:"use_cache"`
	Mirrors  map[string]MavenMirrorServer `yaml:"mirrors"`
}

func (cfg *MavenMirrorConfig) ReadConfig(path string) {
	helpers.ReadYaml(path, cfg)
}
