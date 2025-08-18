package config

import (
	"path"

	"proxynd/internal/helpers"
)

// YumProxy yum 프록시 서버 정보
type YumProxy struct {
	Name string `yaml:"name" validate:"required,min=1,max=100"`
	URL  string `yaml:"url" validate:"required,url"`
}

// YumProxySettings yum 프록시 설정 구조체
type YumProxySettings struct {
	Path     string     `yaml:"path" validate:"required,min=1"`
	UseCache bool       `yaml:"use_cache" default:"true"`
	Proxies  []YumProxy `yaml:"proxies" validate:"required,min=1,dive"`
}

// ConfigExists yum 프록시 설정 파일 존재 여부 확인
func (y *YumProxySettings) ConfigExists() bool {
	confDir := helpers.GetConfigDir()
	return helpers.FileExists(path.Join(confDir, "yum-proxy.yaml"))
}

// ReadConfig yum 프록시 설정 파일 읽기
func (y *YumProxySettings) ReadConfig() error {
	confDir := helpers.GetConfigDir()
	if err := helpers.ReadYamlSafe(path.Join(confDir, "yum-proxy.yaml"), y); err != nil {
		return err
	}
	return y.Validate()
}

// Validate validates the YUM proxy configuration
func (y *YumProxySettings) Validate() error {
	return ValidateStruct(y)
}
