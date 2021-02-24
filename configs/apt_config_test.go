package configs

import (
	"fmt"
	"testing"
)

func TestAptConfig_ReadConfig(t *testing.T) {
	cfg := MavenConfig{}
	cfg.ReadConfig("apt-proxy.yaml")
	fmt.Println(cfg)
}
