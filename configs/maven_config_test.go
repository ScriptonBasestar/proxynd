package configs

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func TestMavenConfig_MavenProxy(t *testing.T) {
	cfg := MavenConfig{}
	cfg.ReadConfig("maven-proxy.yaml")
	fmt.Println(cfg)
}

func TestYamlMake(t *testing.T) {
	mavenConfig := MavenConfig{}
	mavenConfig.Servers = []MavenServer{
		{
			Id:          "maven-center",
			Name:        "Center",
			Url:         "https://repo.maven.com",
			Description: "desc1",
		},
		{
			Id:          "jcenter-center",
			Name:        "JCenter",
			Url:         "https://repo.jmaven.com",
			Description: "desc2",
		},
		{
			Url:         "https://repo.jmaven.com",
			Description: "desc2",
		},
		{
			Name: "springrelease",
			Url:  "https://repo.jmaven.com",
		},
	}
	yamlFile, err := yaml.Marshal(mavenConfig)
	if err != nil {
		fmt.Println("errrrr write")
	}
	ioutil.WriteFile("../configs/maven-tmp.yaml", yamlFile, os.ModeAppend)
	fmt.Println(string(yamlFile))

	mavenConfig2 := MavenConfig{
		Servers: []MavenServer{},
	}
	err = yaml.Unmarshal(yamlFile, mavenConfig2)
	fmt.Println(mavenConfig2)
}

func TestYaml(t *testing.T) {
	yamlFile, err := ioutil.ReadFile("../configs/maven-proxy.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	fmt.Println(string(yamlFile))
	mavenConfig := MavenConfig{
		Servers: []MavenServer{},
	}
	err = yaml.Unmarshal(yamlFile, mavenConfig)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
}
