package configs

import (
	"path"
	"proxynd/helpers"
)

// YumProxy yum 프록시 서버 정보
type YumProxy struct {
	Name string `yaml:"name"`
	Url  string `yaml:"url"`
}

// YumProxyConfig yum 프록시 설정 구조체
type YumProxyConfig struct {
	Path     string     `yaml:"path"`
	UseCache bool       `yaml:"use_cache"`
	Proxies  []YumProxy `yaml:"proxies"`
}

// ConfigExists yum 프록시 설정 파일 존재 여부 확인
func (y *YumProxyConfig) ConfigExists() bool {
	confDir := helpers.GetConfigDir()
	return helpers.FileExists(path.Join(confDir, "yum-proxy.yaml"))
}

// ReadConfig yum 프록시 설정 파일 읽기
func (y *YumProxyConfig) ReadConfig() {
	confDir := helpers.GetConfigDir()
	helpers.ReadYaml(path.Join(confDir, "yum-proxy.yaml"), y)
}
