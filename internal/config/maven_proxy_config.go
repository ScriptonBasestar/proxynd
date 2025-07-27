package config

import (
	"fmt"
	"path"
	"strings"

	"proxynd/helpers"
)

// MavenProxyServer is exported
// MavenProxyServer provides server functionality
type MavenProxyServer struct {
	ID          string    `yaml:"id,omitempty" validate:"omitempty,min=1,max=100"`
	Name        string    `yaml:"name" validate:"required,min=1,max=100"`
	URL         string    `yaml:"url,omitempty" validate:"required,url"`
	Description string    `yaml:"description" validate:"omitempty,max=500"`
	Enabled     bool      `yaml:"enabled,omitempty" default:"true"`
	BasicAuth   BasicAuth `yaml:"basic_auth,omitempty"`
	// BasicAuth is exported
}

// BasicAuth is exported

// BasicAuth represents a basic auth
type BasicAuth struct {
	// MavenProxyConfig is exported
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	// MavenProxyConfig is exported
}

// MavenProxyConfig represents the configuration for mavenproxy settings
type MavenProxyConfig struct {
	Path        string                 `yaml:"path,omitempty" validate:"required,min=1"`
	UseCache    bool                   `yaml:"use_cache,omitempty" default:"true"`
	Proxies     []MavenProxyServer     `yaml:"proxies" validate:"required,min=1,dive"`
	Cache       MavenProxyCacheConfig  `yaml:"cache,omitempty"`
	SearchIndex MavenSearchIndexConfig `yaml:"search_index,omitempty"`
}

// MavenProxyCacheConfig represents the configuration for mavenproxycache settings
type MavenProxyCacheConfig struct {
	Enabled bool `yaml:"enabled,omitempty" default:"true"`
}

// MavenSearchIndexConfig 검색 인덱스 설정
type MavenSearchIndexConfig struct {
	// 서버 시작 시 자동으로 인덱스 빌드 여부 (기본값: false)
	AutoBuildOnStartup bool `yaml:"auto_build_on_startup,omitempty" default:"false"`
	// 인덱스 자동 갱신 주기 (시간 단위, 0이면 자동 갱신 안함, 기본값: 0)
	AutoRebuildIntervalHours int `yaml:"auto_rebuild_interval_hours,omitempty" default:"0"`
	// 인덱스 저장 경로 (기본값: STORAGE_DIR/maven-index)
	StoragePath string `yaml:"storage_path,omitempty"`
}

// ConfigExists checks if the configuration file exists
func (cfg *MavenProxyConfig) ConfigExists() bool {
	confDir := helpers.GetConfigDir()
	return helpers.FileExists(path.Join(confDir, "maven-proxy.yaml"))
}

// ReadConfig reads the configuration from file
func (cfg *MavenProxyConfig) ReadConfig() error {
	confDir := helpers.GetConfigDir()
	if err := helpers.ReadYamlSafe(path.Join(confDir, "maven-proxy.yaml"), cfg); err != nil {
		return err
	}
	return cfg.Validate()
}

// Validate validates the Maven proxy configuration
func (cfg *MavenProxyConfig) Validate() error {
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
			Message: "Maven 프록시가 활성화되어 있으나 서버가 설정되지 않음",
			Value:   cfg.Proxies,
		}
	}

	// 각 프록시 서버 검증
	for i, proxy := range cfg.Proxies {
		if proxy.Name == "" {
			return &ValidationError{
				Field:   fmt.Sprintf("proxies[%d].name", i),
				Message: "프록시 서버 이름이 비어 있음",
				Value:   proxy.Name,
			}
		}

		if proxy.URL == "" {
			return &ValidationError{
				Field:   fmt.Sprintf("proxies[%d].url", i),
				Message: "프록시 서버 URL이 비어 있음",
				Value:   proxy.URL,
			}
		}

		// URL 형식 검증
		if !strings.HasPrefix(proxy.URL, "http://") && !strings.HasPrefix(proxy.URL, "https://") {
			return &ValidationError{
				Field:   fmt.Sprintf("proxies[%d].url", i),
				Message: "잘못된 프록시 서버 URL 형식",
				Value:   proxy.URL,
			}
		}

		// Basic Auth 검증
		if proxy.BasicAuth.Username != "" && proxy.BasicAuth.Password == "" {
			return &ValidationError{
				Field:   fmt.Sprintf("proxies[%d].basic_auth.password", i),
				Message: "사용자명이 설정되었으나 비밀번호가 비어 있음",
				Value:   proxy.BasicAuth.Password,
			}
		}
	}

	return nil
}
