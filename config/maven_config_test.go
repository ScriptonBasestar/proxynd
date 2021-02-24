package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func TestYamlMake(t *testing.T) {
	mavenConfig := MavenConfig{}
	mavenConfig.Server = map[string]MavenServer{
		"centeral": {
			Id:          "maven-center",
			Name:        "Center",
			Url:         "https://repo.maven.com",
			Description: "desc1",
		},
		"jcenter": {
			Id:          "jcenter-center",
			Name:        "JCenter",
			Url:         "https://repo.jmaven.com",
			Description: "desc2",
		},
		"google": {
			Url:         "https://repo.jmaven.com",
			Description: "desc2",
		},
		"spring-release": {
			Name: "springrelease",
			Url:  "https://repo.jmaven.com",
		},
	}
	yamlFile, err := yaml.Marshal(mavenConfig)
	if err != nil {
		fmt.Println("errrrr write")
	}
	ioutil.WriteFile("../config/maven-tmp.yaml", yamlFile, os.ModeAppend)
	fmt.Println(string(yamlFile))

	mavenConfig2 := MavenConfig{
		Server: map[string]MavenServer{},
	}
	err = yaml.Unmarshal(yamlFile, mavenConfig2)
	fmt.Println(mavenConfig2)
}

func TestYaml(t *testing.T) {

	yamlFile, err := ioutil.ReadFile("../config/maven-proxy-public.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	fmt.Println(string(yamlFile))
	mavenConfig := MavenConfig{
		Server: map[string]MavenServer{},
	}
	err = yaml.Unmarshal(yamlFile, mavenConfig)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

}
