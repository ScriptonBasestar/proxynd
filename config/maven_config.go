package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
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

type DownloadService struct {
	MavenConfig
}

func (ds *DownloadService) DownloadProxy(path string) (MavenConfig, error) {
	// cache - no expire
	yamlFile, err := ioutil.ReadFile("config/maven-proxy-public.yaml")
	if err != nil {
		log.Printf("read config err #%v ", err)
	}
	fmt.Println(yamlFile)
	mavenConfig := MavenConfig{
		Server: map[string]MavenServer{},
	}
	err = yaml.Unmarshal(yamlFile, mavenConfig)
	fmt.Println(mavenConfig)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	fmt.Printf("config %s \n", mavenConfig)
	return mavenConfig, nil
}
