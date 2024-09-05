package configs

import (
	"path"
	"proxynd/helpers"
)

type AptMirror struct {
	URL string `json:"url,omitempty"`
}

type AptMirrors struct {
	Ubuntu map[string]AptMirror `json:"ubuntu"`
	Debian map[string]AptMirror `json:"debian"`
}

type AptMirrorConfig struct {
	Path    string     `json:"path,omitempty"`
	Mirrors AptMirrors `json:"mirrors"`
}

func (cfg *AptMirrorConfig) ReadConfig() {
	confDir := helpers.GetConfigDir()
	helpers.ReadYaml(path.Join(confDir, "apt-mirror.yaml"), cfg)
}
