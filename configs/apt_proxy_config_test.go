package configs

import (
	"deproxy/helpers"
	"fmt"
	"github.com/go-playground/assert/v2"
	"testing"
)

func TestAptProxyConfig_ReadConfig(t *testing.T) {
	cfg := AptProxyConfig{}
	cfg.ReadConfig("../conf/proxy-apt.yaml")
	fmt.Println(cfg)
	assert.Equal(t, cfg.Path, "~/tmp/proxy/apt")
	fmt.Println(helpers.ToStringYaml(cfg))
}
