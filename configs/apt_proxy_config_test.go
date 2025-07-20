package configs

import (
	"fmt"
	"os"
	"testing"

	"github.com/go-playground/assert/v2"
	"github.com/stretchr/testify/require"

	"proxynd/helpers"
)

func TestAptProxyConfig_ReadConfig(t *testing.T) {
	require.NoError(t, os.Setenv("CONFIG_DIR", "../sample-conf/"))
	cfg := AptProxyConfig{}
	require.NoError(t, cfg.ReadConfig())
	fmt.Println(cfg)
	assert.Equal(t, cfg.Path, "proxy/apt")
	fmt.Println(helpers.ToStringYaml(cfg))
}
