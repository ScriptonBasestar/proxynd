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
	Server map[string]MavenServer `yaml:"server"`
}

func (cfg *MavenConfig) ReadConfig(path string) {
	helpers.ReadYaml(path, cfg)
}
