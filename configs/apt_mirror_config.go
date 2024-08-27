package configs

import (
	"deproxy/helpers"
	"path"
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
	confDir := helpers.GetEnv("CONFIG_DIR", "conf/")
	helpers.ReadYaml(path.Join(confDir, "apt-mirror.yaml"), cfg)
}
