package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/go-playground/assert/v2"
	"github.com/stretchr/testify/require"

	"proxynd/internal/helpers"
)

func TestRead_GlobalConfig(t *testing.T) {
	require.NoError(t, os.Setenv("CONFIG_DIR", "../../examples/"))
	require.NoError(t, os.Setenv("STORAGE_DIR", "../../examples/"))
	cfg := GlobalConfig{}
	require.NoError(t, cfg.ReadConfig())
	assert.Equal(t, cfg.Cache.TTL, 3600)
	// assert.Equal(t, cfg.ConfigDir, "~/tmp/config")
	fmt.Println(helpers.ToStringYaml(cfg))
}
