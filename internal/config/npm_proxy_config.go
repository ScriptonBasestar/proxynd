package config

import (
	"fmt"
	"path"
	"strings"

	"proxynd/helpers"
)

// NpmProxyServer is exported
// NpmProxyServer provides server functionality
type NpmProxyServer struct {
	Name string `yaml:"name,omitempty" validate:"required,min=1,max=100"`
	URL  string `yaml:"url,omitempty" validate:"required,url"`
	// NpmProxyConfig is exported
}

// NpmProxyConfig is exported

// NpmProxyConfig represents the configuration for npmproxy settings
type NpmProxyConfig struct {
	Path      string                      `yaml:"path,omitempty" validate:"required,min=1"`
	UseCache  bool                        `yaml:"use_cache,omitempty" default:"true"`
	UserCache bool                        `yaml:"user_cache,omitempty" default:"false"`
	Proxies   map[string][]NpmProxyServer `yaml:"proxies" validate:"required,min=1,dive,keys,min=1,endkeys,min=1,dive"`
}

// NpmProxy is an alias for NpmProxyConfig for backwards compatibility
type NpmProxy = NpmProxyConfig

// ConfigExists checks if the configuration file exists
func (cfg *NpmProxyConfig) ConfigExists() bool {
	confDir := helpers.GetConfigDir()
	return helpers.FileExists(path.Join(confDir, "npm-proxy.yaml"))
}

// ReadConfig reads the configuration from file
func (cfg *NpmProxyConfig) ReadConfig() error {
	confDir := helpers.GetConfigDir()
	if err := helpers.ReadYamlSafe(path.Join(confDir, "npm-proxy.yaml"), cfg); err != nil {
		return err
	}
	return cfg.Validate()
}

// Validate validates the NPM proxy configuration
func (cfg *NpmProxyConfig) Validate() error {
	// 기본 필드 검증
	if cfg.Path == "" {
		return &ValidationError{
			Field:   "path",
			Message: "프록시 경로가 설정되지 않음",
			Value:   cfg.Path,
		}
	}

	// 프록시가 없는 경우 검증
	if len(cfg.Proxies) == 0 {
		return &ValidationError{
			Field:   "proxies",
			Message: "NPM 프록시가 활성화되어 있으나 서버가 설정되지 않음",
			Value:   cfg.Proxies,
		}
	}

	// 기본 레지스트리 존재 확인
	if _, exists := cfg.Proxies["default"]; !exists {
		return &ValidationError{
			Field:   "proxies.default",
			Message: "기본 레지스트리 설정이 필요합니다",
			Value:   cfg.Proxies,
		}
	}

	// 각 레지스트리의 프록시 서버 검증
	for registry, servers := range cfg.Proxies {
		if len(servers) == 0 {
			return &ValidationError{
				Field:   "proxies." + registry,
				Message: "레지스트리에 대한 프록시 서버가 설정되지 않음",
				Value:   servers,
			}
		}

		for i, server := range servers {
			if server.Name == "" {
				return &ValidationError{
					Field:   fmt.Sprintf("proxies.%s[%d].name", registry, i),
					Message: "프록시 서버 이름이 비어 있음",
					Value:   server.Name,
				}
			}

			if server.URL == "" {
				return &ValidationError{
					Field:   fmt.Sprintf("proxies.%s[%d].url", registry, i),
					Message: "프록시 서버 URL이 비어 있음",
					Value:   server.URL,
				}
			}

			// URL 형식 검증
			if !strings.HasPrefix(server.URL, "http://") && !strings.HasPrefix(server.URL, "https://") {
				return &ValidationError{
					Field:   fmt.Sprintf("proxies.%s[%d].url", registry, i),
					Message: "잘못된 프록시 서버 URL 형식",
					Value:   server.URL,
				}
			}
		}
	}

	return nil
}
