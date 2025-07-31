package config

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/go-playground/assert/v2"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"proxynd/helpers"
)

func TestNpmConfig_NpmProxy(t *testing.T) {
	require.NoError(t, os.Setenv("CONFIG_DIR", "../../examples/"))
	cfg := NpmProxySettings{}
	require.NoError(t, cfg.ReadConfig())
	// fmt.Println(cfg)
	assert.Equal(t, cfg.Path, "proxy/npm")
	fmt.Println(helpers.ToStringYaml(cfg))
}

func TestNpmYamlMake(t *testing.T) {
	npmConfig := NpmProxySettings{}
	npmConfig.Path = "tmp"
	npmConfig.Proxies = map[string][]NpmProxyServer{
		"default": {
			{
				Name: "Center",
				URL:  "https://repo.npm.com",
			},
			{
				Name: "npm-github-packages",
				URL:  "https://repo.jmaven.com",
			},
		},
	}
	yamlFile, err := yaml.Marshal(npmConfig)
	if err != nil {
		fmt.Println("errrrr write")
	}
	err = os.MkdirAll("tmp/", 0o766)
	if err != nil {
		t.Error(err)
		return
	}
	err = os.WriteFile("tmp/maven-tmp.yaml", yamlFile, 0o766)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(string(yamlFile))
}

func TestNpmYaml(_ *testing.T) {
	yamlFile, err := os.ReadFile("../configs/maven-proxy.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	fmt.Println(string(yamlFile))
	mavenConfig := MavenProxySettings{
		Proxies: []MavenProxyServer{},
	}
	err = yaml.Unmarshal(yamlFile, mavenConfig)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
}
