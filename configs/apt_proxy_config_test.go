package configs

import (
	"fmt"
	"github.com/go-playground/assert/v2"
	"os"
	"proxynd/helpers"
	"testing"
)

func TestAptProxyConfig_ReadConfig(t *testing.T) {
	os.Setenv("CONFIG_DIR", "../sample-conf/")
	cfg := AptProxyConfig{}
	cfg.ReadConfig()
	fmt.Println(cfg)
	assert.Equal(t, cfg.Path, "proxy/apt")
	fmt.Println(helpers.ToStringYaml(cfg))
}
