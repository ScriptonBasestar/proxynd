package configs

import (
	"fmt"
	"os"
	"testing"

	"github.com/go-playground/assert/v2"

	"proxynd/helpers"
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
