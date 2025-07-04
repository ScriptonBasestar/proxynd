package configs

import (
	"fmt"
	"os"
	"testing"

	"github.com/go-playground/assert/v2"

	"proxynd/helpers"
)

func TestAptMirrorConfig_ReadConfig(t *testing.T) {
	os.Setenv("CONFIG_DIR", "../sample-conf/")
	cfg := AptMirrorConfig{}
	cfg.ReadConfig()
	fmt.Println(cfg)
	assert.Equal(t, cfg.Path, "mirror/apt")
	fmt.Println(helpers.ToStringYaml(cfg))
}
