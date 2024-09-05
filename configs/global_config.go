package configs

import (
	"deproxy/helpers"
	"path"
)

type Cache struct {
	TTL int `yaml:"ttl,omitempty" default:36000`
}

type GlobalConfig struct {
	//StorageDir string `yaml:"storage_dir,omitempty"`
	//ConfigDir string `yaml:"config_dir,omitempty"`
	//ProxyName
	Cache Cache `yaml:"cache,omitempty"`
}

func (cfg *GlobalConfig) ReadConfig() {
	confDir := helpers.GetConfigDir()
	helpers.ReadYaml(path.Join(confDir, "global.yaml"), cfg)
}
