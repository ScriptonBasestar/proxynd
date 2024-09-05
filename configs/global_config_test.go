package configs

import (
	"fmt"
	"github.com/go-playground/assert/v2"
	"os"
	"proxynd/helpers"
	"testing"
)

func TestRead_GlobalConfig(t *testing.T) {
	os.Setenv("CONFIG_DIR", "../sample-conf/")
	os.Setenv("STORAGE_DIR", "../sample-conf/")
	cfg := GlobalConfig{}
	cfg.ReadConfig()
	assert.Equal(t, cfg.Cache.TTL, 3600)
	//assert.Equal(t, cfg.ConfigDir, "~/tmp/config")
	fmt.Println(helpers.ToStringYaml(cfg))
}
