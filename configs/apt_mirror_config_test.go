package configs

import (
	"deproxy/helpers"
	"fmt"
	"github.com/go-playground/assert/v2"
	"testing"
)

func TestAptMirrorConfig_ReadConfig(t *testing.T) {
	cfg := AptMirrorConfig{}
	cfg.ReadConfig("../conf/mirror-apt.yaml")
	fmt.Println(cfg)
	assert.Equal(t, cfg.Path, "~/tmp/mirror/apt")
	fmt.Println(helpers.ToStringYaml(cfg))
}
