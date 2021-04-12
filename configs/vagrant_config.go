package configs

import (
	"dohoarding/helpers"
)

type VagrantServer struct {
	Id          string `yaml:"id"`
	Name        string `yaml:"name"`
	Url         string `yaml:"url"`
	Description string `yaml:"description"`
}

type VagrantConfig struct {
	Path   string                 `yaml:"path"`
	Server map[string]MavenServer `yaml:"server"`
}

func (cfg *VagrantConfig) ReadConfig(path string) {
	helpers.ReadYaml(path, cfg)
}
