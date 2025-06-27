package configs

import (
	"path"
	"proxynd/helpers"
)

type Cache struct {
	TTL int `yaml:"ttl,omitempty" default:36000`
}

type GlobalConfig struct {
	StorageDir string `yaml:"storage_dir,omitempty"`
	ConfigDir  string `yaml:"config_dir,omitempty"`
	Cache      Cache  `yaml:"cache,omitempty"`
}

func (cfg *GlobalConfig) ConfigExists() bool {
	confDir := helpers.GetConfigDir()
	return helpers.FileExists(path.Join(confDir, "global.yaml"))
}

func (cfg *GlobalConfig) ReadConfig() {
	confDir := helpers.GetConfigDir()
	helpers.ReadYaml(path.Join(confDir, "global.yaml"), cfg)
}
