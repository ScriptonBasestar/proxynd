package configs

import (
	"deproxy/helpers"
	"fmt"
	"github.com/go-playground/assert/v2"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"testing"
)

func TestMavenConfig_MavenProxy(t *testing.T) {
	os.Setenv("CONFIG_DIR", "../sample-conf/")
	cfg := MavenProxyConfig{}
	cfg.ReadConfig()
	//fmt.Println(cfg)
	assert.Equal(t, cfg.Path, "~/tmp/proxy/maven")
	fmt.Println(helpers.ToStringYaml(cfg))
}

func TestYamlMake(t *testing.T) {
	mavenConfig := MavenProxyConfig{}
	mavenConfig.Path = "tmp"
	mavenConfig.Proxies = []MavenProxyServer{
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
	yamlFile, err := os.ReadFile("../configs/maven-proxy.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	fmt.Println(string(yamlFile))
	mavenConfig := MavenProxyConfig{
		Proxies: []MavenProxyServer{},
	}
	err = yaml.Unmarshal(yamlFile, mavenConfig)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
}
