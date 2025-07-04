package configs

import (
	"fmt"
	"os"
	"testing"

	"github.com/go-playground/assert/v2"

	"proxynd/helpers"
)

func TestMavenConfig_MavenMirror(t *testing.T) {
	os.Setenv("CONFIG_DIR", "../sample-conf/")
	cfg := MavenMirrorConfig{}
	cfg.ReadConfig()
	//fmt.Println(cfg)
	assert.Equal(t, cfg.Path, "mirror/maven")
	fmt.Println(helpers.ToStringYaml(cfg))
}
