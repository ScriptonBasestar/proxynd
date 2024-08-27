package configs

import (
	"deproxy/helpers"
)

type AptProxy struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type AptProxyConfig struct {
	Path      string                `json:"path"`
	UserCache string                `json:"user_cache"`
	Proxies   map[string][]AptProxy `json:"proxies"`
}

func (cfg *AptProxyConfig) ReadConfig(path string) {
	helpers.ReadYaml(path, cfg)
}
