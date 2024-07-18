package configs

import (
	"fmt"
	"github.com/go-playground/assert/v2"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"testing"
)

func TestMavenConfig_MavenProxy(t *testing.T) {
	cfg := MavenConfig{}
	cfg.ReadConfig("../conf/proxy-maven.yaml")
	fmt.Println(cfg)
	assert.Equal(t, cfg.Path, "d:/tmp/cachedir/proxy/maven")
}

func TestYamlMake(t *testing.T) {
	mavenConfig := MavenConfig{}
	mavenConfig.Path = "tmp"
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
	err = os.MkdirAll("tmp/", 0766)
	if err != nil {
		t.Error(err)
		return
	}
	err = os.WriteFile("tmp/maven-tmp.yaml", yamlFile, 0766)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(string(yamlFile))
}

func TestYaml(t *testing.T) {
	yamlFile, err := os.ReadFile("../configs/proxy-maven.yaml")
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
