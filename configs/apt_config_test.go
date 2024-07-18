package configs

import (
	"fmt"
	"github.com/go-playground/assert/v2"
	"testing"
)

func TestAptConfig_ReadConfig(t *testing.T) {
	cfg := MavenConfig{}
	cfg.ReadConfig("../conf/proxy-apt.yaml")
	fmt.Println(cfg)
	assert.Equal(t, cfg.Path, "~/tmp/cachedir/proxy/apt")
}
