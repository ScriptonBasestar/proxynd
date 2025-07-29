package config

import (
	"fmt"
	"path"
	"strings"

	"proxynd/helpers"
)

// AptProxySettings represents the configuration for APT proxy settings
// This replaces the old AptProxyConfig for clearer naming
type AptProxySettings struct {
	Path      string                `yaml:"path,omitempty" validate:"required,min=1"`
	UseCache  bool                  `yaml:"use_cache,omitempty" default:"true"`
	UserCache bool                  `yaml:"user_cache,omitempty" default:"false"`
	Proxies   map[string][]AptProxy `yaml:"proxies" validate:"required,min=1,dive,keys,min=1,endkeys,min=1,dive"`
}

// ConfigExists checks if the APT proxy configuration file exists
func (cfg *AptProxySettings) ConfigExists() bool {
	confDir := helpers.GetConfigDir()
	return helpers.FileExists(path.Join(confDir, "apt-proxy.yaml"))
}

// ReadConfig reads and validates the APT proxy configuration
func (cfg *AptProxySettings) ReadConfig() error {
	confDir := helpers.GetConfigDir()
	if err := helpers.ReadYamlSafe(path.Join(confDir, "apt-proxy.yaml"), cfg); err != nil {
		return err
	}
	return cfg.Validate()
}

// Validate validates the APT proxy configuration
func (cfg *AptProxySettings) Validate() error {
	// 기본 필드 검증
	if cfg.Path == "" {
		return &ValidationError{
			Field:   "path",
			Message: "프록시 경로가 설정되지 않음",
			Value:   cfg.Path,
		}
	}

	// 프록시가 활성화되어 있는데 미러가 없는 경우 검증
	if len(cfg.Proxies) == 0 {
		return &ValidationError{
			Field:   "proxies",
			Message: "APT 프록시가 활성화되어 있으나 미러가 설정되지 않음",
			Value:   cfg.Proxies,
		}
	}

	// 각 프록시의 URL 검증
	for distro, proxies := range cfg.Proxies {
		if len(proxies) == 0 {
			return &ValidationError{
				Field:   "proxies." + distro,
				Message: "배포판에 대한 프록시가 설정되지 않음",
				Value:   proxies,
			}
		}

		for i, proxy := range proxies {
			if proxy.Name == "" {
				return &ValidationError{
					Field:   fmt.Sprintf("proxies.%s[%d].name", distro, i),
					Message: "프록시 이름이 비어 있음",
					Value:   proxy.Name,
				}
			}

			if proxy.URL == "" {
				return &ValidationError{
					Field:   fmt.Sprintf("proxies.%s[%d].url", distro, i),
					Message: "프록시 URL이 비어 있음",
					Value:   proxy.URL,
				}
			}

			// URL 형식 검증
			if !strings.HasPrefix(proxy.URL, "http://") && !strings.HasPrefix(proxy.URL, "https://") {
				return &ValidationError{
					Field:   fmt.Sprintf("proxies.%s[%d].url", distro, i),
					Message: "잘못된 프록시 URL 형식",
					Value:   proxy.URL,
				}
			}
		}
	}

	return nil
}
