package configs

import (
	"deproxy/helpers"
)

type AptMirror struct {
	URL string `json:"url"`
}

type AptMirrors struct {
	Ubuntu map[string]AptMirror `json:"ubuntu"`
	Debian map[string]AptMirror `json:"debian"`
}

type AptMirrorConfig struct {
	Path    string     `json:"path"`
	Mirrors AptMirrors `json:"mirrors"`
}

func (cfg *AptMirrorConfig) ReadConfig(path string) {
	helpers.ReadYaml(path, cfg)
}
