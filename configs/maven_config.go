package configs

import (
	"gopkg.in/yaml.v3"
	"log"
	"os"
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
	//helpers.ReadYaml(path, cfg)
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		log.Printf("read configs err #%v ", err)
	}
	log.Printf("read yaml file path: %s", path)
	err = yaml.Unmarshal(yamlFile, cfg)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
		panic(err)
	}
	log.Printf("read config success %s \n", cfg)
}
