package config

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/go-playground/assert/v2"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"proxynd/internal/helpers"
)

func TestMavenConfig_MavenProxy(t *testing.T) {
	require.NoError(t, os.Setenv("CONFIG_DIR", "../../examples/"))
	cfg := MavenProxySettings{}
	require.NoError(t, cfg.ReadConfig())
	// fmt.Println(cfg)
	assert.Equal(t, cfg.Path, "proxy/maven")
	fmt.Println(helpers.ToStringYaml(cfg))
}

func TestYamlMake(t *testing.T) {
	mavenConfig := MavenProxySettings{}
	mavenConfig.Path = "tmp"
	mavenConfig.Proxies = []MavenProxyServer{
		{
			ID:          "maven-center",
			Name:        "Center",
			URL:         "https://repo.maven.com",
			Description: "desc1",
		},
		{
			ID:          "jcenter-center",
			Name:        "JCenter",
			URL:         "https://repo.jmaven.com",
			Description: "desc2",
		},
		{
			URL:         "https://repo.jmaven.com",
			Description: "desc2",
		},
		{
			Name: "springrelease",
			URL:  "https://repo.jmaven.com",
		},
	}
	yamlFile, err := yaml.Marshal(mavenConfig)
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

func TestYaml(_ *testing.T) {
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
