package configs

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
)

type AptServer struct {
	Id          string `yaml:"id"`
	Name        string `yaml:"name"`
	Url         string `yaml:"url"`
	Description string `yaml:"description"`
}

type AptConfig struct {
	Server map[string]AptServer `yaml:"server"`
}

func (cfg *AptConfig) ReadConfig(path string) {
	// cache - no expire
	yamlFile, err := ioutil.ReadFile(path)
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
