package configs

import (
	"dohoarding/helpers"
)

type AptServer struct {
	//Id          string `yaml:"id"`
	//Name        string `yaml:"name"`
	Url string `yaml:"url"`
	//Description string `yaml:"description"`
}

type AptConfig struct {
	Server map[string]AptServer `yaml:"server"`
}

func (cfg *AptConfig) ReadConfig(path string) {
	helpers.ReadYaml(path, cfg)
}
