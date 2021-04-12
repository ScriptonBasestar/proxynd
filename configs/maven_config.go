package configs

import (
	"dohoarding/helpers"
)

type MavenServer struct {
	Id          string `yaml:"id"`
	Name        string `yaml:"name"`
	Url         string `yaml:"url"`
	Description string `yaml:"description"`
}

type MavenConfig struct {
	Path    string        `yaml:"path"`
	Servers []MavenServer `yaml:"servers"`
}

func (cfg *MavenConfig) ReadConfig(path string) {
	helpers.ReadYaml(path, cfg)
}
