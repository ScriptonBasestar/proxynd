package configs

import (
	"deproxy/helpers"
	"fmt"
	"github.com/go-playground/assert/v2"
	"testing"
)

func TestMavenConfig_MavenMirror(t *testing.T) {
	cfg := MavenMirrorConfig{}
	cfg.ReadConfig("../conf/mirror-maven.yaml")
	//fmt.Println(cfg)
	assert.Equal(t, cfg.Path, "tmp/cachedir/mirror/maven")
	fmt.Println(helpers.ToStringYaml(cfg))
}
