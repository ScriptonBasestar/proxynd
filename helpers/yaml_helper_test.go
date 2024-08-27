package helpers

import (
	"fmt"
	"testing"
)

func TestReadYaml(t *testing.T) {
	var cfg struct {
		Server map[string]interface{} `yaml:"server"`
	}
	cfg.Server = map[string]interface{}{}
	ReadYaml("../conf/maven-proxy.yaml", cfg)
	fmt.Println(cfg)
}
