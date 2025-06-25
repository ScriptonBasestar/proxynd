package configs

import (
	"fmt"
	"github.com/go-playground/assert/v2"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"proxynd/helpers"
	"testing"
)

func TestNpmConfig_NpmProxy(t *testing.T) {
	os.Setenv("CONFIG_DIR", "../sample-conf/")
	cfg := NpmProxyConfig{}
	cfg.ReadConfig()
	//fmt.Println(cfg)
	assert.Equal(t, cfg.Path, "proxy/npm")
	fmt.Println(helpers.ToStringYaml(cfg))
}

func TestNpmYamlMake(t *testing.T) {
	npmConfig := NpmProxyConfig{}
	npmConfig.Path = "tmp"
	npmConfig.Proxies = []NpmProxyServer{
		{
			Name: "Center",
			URL:  "https://repo.npm.com",
		},
		{
			Name: "npm-github-packages",
			URL:  "https://repo.jmaven.com",
		},
	}
	yamlFile, err := yaml.Marshal(npmConfig)
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

func TestNpmYaml(t *testing.T) {
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
