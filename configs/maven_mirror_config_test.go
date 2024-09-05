package configs

import (
	"deproxy/helpers"
	"fmt"
	"github.com/go-playground/assert/v2"
	"os"
	"testing"
)

func TestMavenConfig_MavenMirror(t *testing.T) {
	os.Setenv("CONFIG_DIR", "../sample-conf/")
	cfg := MavenMirrorConfig{}
	cfg.ReadConfig()
	//fmt.Println(cfg)
	assert.Equal(t, cfg.Path, "mirror/maven")
	fmt.Println(helpers.ToStringYaml(cfg))
}
