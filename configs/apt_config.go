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
	Path    string        `yaml:"path"`
	Servers []MavenServer `yaml:"servers"`
}

func (cfg *AptConfig) ReadConfig(path string) {
	helpers.ReadYaml(path, cfg)
}
