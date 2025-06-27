package configs

import (
	"path"
	"proxynd/helpers"
)

// ApkProxy APK 프록시 서버 정보
type ApkProxy struct {
	Name string `yaml:"name"`
	Url  string `yaml:"url"`
}

// ApkProxyConfig APK 프록시 설정 구조체
type ApkProxyConfig struct {
	Path     string     `yaml:"path"`
	UseCache bool       `yaml:"use_cache"`
	Proxies  []ApkProxy `yaml:"proxies"`
}

// ConfigExists APK 프록시 설정 파일 존재 여부 확인
func (a *ApkProxyConfig) ConfigExists() bool {
	confDir := helpers.GetConfigDir()
	return helpers.FileExists(path.Join(confDir, "apk-proxy.yaml"))
}

// ReadConfig APK 프록시 설정 파일 읽기
func (a *ApkProxyConfig) ReadConfig() {
	confDir := helpers.GetConfigDir()
	helpers.ReadYaml(path.Join(confDir, "apk-proxy.yaml"), a)
}
