package configs

import (
	"path"

	"proxynd/helpers"
)

type AptMirror struct {
	URL string `yaml:"url,omitempty"`
}

type AptMirrors struct {
	Ubuntu map[string]AptMirror `yaml:"ubuntu"`
	Debian map[string]AptMirror `yaml:"debian"`
}

type AptMirrorConfig struct {
	Path    string     `yaml:"path,omitempty"`
	Mirrors AptMirrors `yaml:"mirrors"`
}

func (cfg *AptMirrorConfig) ReadConfig() {
	confDir := helpers.GetConfigDir()
	helpers.ReadYaml(path.Join(confDir, "apt-mirror.yaml"), cfg)
}
