package configs

import (
	"deproxy/helpers"
	"fmt"
	"github.com/go-playground/assert/v2"
	"os"
	"testing"
)

func TestRead_GlobalConfig(t *testing.T) {
	os.Setenv("CONFIG_DIR", "../conf/")
	cfg := GlobalConfig{}
	cfg.ReadConfig()
	assert.Equal(t, cfg.BaseDir, "~/tmp/deproxy")
	//assert.Equal(t, cfg.ConfigDir, "~/tmp/config")
	fmt.Println(helpers.ToStringYaml(cfg))
}
