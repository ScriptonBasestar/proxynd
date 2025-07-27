package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/go-playground/assert/v2"
	"github.com/stretchr/testify/require"

	"proxynd/helpers"
)

func TestMavenConfig_MavenMirror(t *testing.T) {
	require.NoError(t, os.Setenv("CONFIG_DIR", "../../examples/"))
	cfg := MavenMirrorConfig{}
	cfg.ReadConfig()
	// fmt.Println(cfg)
	assert.Equal(t, cfg.Path, "mirror/maven")
	fmt.Println(helpers.ToStringYaml(cfg))
}
