package configs

import (
	"deproxy/helpers"
	"path"
	"strings"
)

type GlobalConfig struct {
	StorageDir string `yaml:"storage_dir,omitempty"`
	//ConfigDir string `yaml:"config_dir,omitempty"`
}

func (cfg *GlobalConfig) ReadConfig() {
	confDir := helpers.GetConfigDir()
	helpers.ReadYaml(path.Join(confDir, "global.yaml"), cfg)

	cfg.StorageDir = helpers.GetStorageDir()
	if cfg.StorageDir == "" {
		cfg.StorageDir = "~/tmp/deproxy"
	}
	if strings.HasPrefix(cfg.StorageDir, "~") {
		//homeDir, err := os.UserHomeDir()
		//if err != nil {
		//	fmt.Printf("Error retrieving home directory: %v\n", err)
		//	return
		//}
		cfg.StorageDir = helpers.ExpandHome(cfg.StorageDir)
	}
}
