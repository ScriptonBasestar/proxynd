package configs

import (
	"deproxy/helpers"
)

type VagrantServer struct {
	Id          string `yaml:"id,omitempty"`
	Name        string `yaml:"name"`
	Url         string `yaml:"url,omitempty"`
	Description string `yaml:"description"`
}

type VagrantConfig struct {
	Path   string                   `yaml:"path,omitempty"`
	Server map[string]VagrantServer `yaml:"server"`
}

func (cfg *VagrantConfig) ReadConfig(path string) {
	helpers.ReadYaml(path, cfg)
}
