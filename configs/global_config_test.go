package configs

import (
	"deproxy/helpers"
	"fmt"
	"github.com/go-playground/assert/v2"
	"os"
	"testing"
)

func TestRead_GlobalConfig(t *testing.T) {
	os.Setenv("CONFIG_DIR", "../sample-conf/")
	cfg := GlobalConfig{}
	cfg.ReadConfig()
	assert.Equal(t, cfg.StorageDir, "~/tmp/deproxy")
	//assert.Equal(t, cfg.ConfigDir, "~/tmp/config")
	fmt.Println(helpers.ToStringYaml(cfg))
}
