package configs

import (
	"deproxy/helpers"
	"path"
	"strings"
)

type GlobalConfig struct {
	BaseDir string `yaml:"base_dir,omitempty"`
	//ConfigDir string `yaml:"config_dir,omitempty"`
}

func (cfg *GlobalConfig) ReadConfig() {
	confDir := helpers.GetEnv("CONFIG_DIR", "conf/")
	helpers.ReadYaml(path.Join(confDir, "global.yaml"), cfg)

	if cfg.BaseDir == "" {
		cfg.BaseDir = "~/tmp/deproxy"
	}
	if strings.HasPrefix(cfg.BaseDir, "~") {
		cfg.BaseDir = helpers.ExpandHome(cfg.BaseDir)
	}
}
