package config

import (
	"fmt"

	"proxynd/internal/config"
)

// globalConfigValidator validates global configuration
type globalConfigValidator struct{}

// Validate validates the global configuration
func (v *globalConfigValidator) Validate(config interface{}) error {
	cfg, ok := config.(*config.GlobalConfig)
	if !ok {
		return fmt.Errorf("invalid config type, expected *config.GlobalConfig")
	}

	// 캐시 디렉토리 검증
	if cfg.CacheDir == "" {
		return fmt.Errorf("cache directory cannot be empty")
	}

	// 캐시 TTL 검증
	if cfg.CacheTTL < 0 {
		return fmt.Errorf("cache TTL cannot be negative")
	}

	// 최대 캐시 크기 검증
	if cfg.MaxCacheSize < 0 {
		return fmt.Errorf("max cache size cannot be negative")
	}

	return nil
}

// proxyConfigValidator validates proxy configurations
type proxyConfigValidator struct {
	proxyType string
}

// Validate validates a proxy configuration
func (v *proxyConfigValidator) Validate(config interface{}) error {
	switch v.proxyType {
	case "apt":
		return v.validateAptConfig(config)
	case "maven":
		return v.validateMavenConfig(config)
	case "npm":
		return v.validateNpmConfig(config)
	case "pip":
		return v.validatePipConfig(config)
	case "yum":
		return v.validateYumConfig(config)
	case "apk":
		return v.validateApkConfig(config)
	case "helm":
		return v.validateHelmConfig(config)
	case "docker":
		return v.validateDockerConfig(config)
	default:
		return fmt.Errorf("unknown proxy type: %s", v.proxyType)
	}
}

// validateAptConfig validates APT proxy configuration
func (v *proxyConfigValidator) validateAptConfig(config interface{}) error {
	cfg, ok := config.(*config.AptProxyConfig)
	if !ok {
		return fmt.Errorf("invalid config type, expected *config.AptProxyConfig")
	}

	// Use the config's own validation method which includes struct tag validation
	return cfg.Validate()
}

// validateMavenConfig validates Maven proxy configuration
func (v *proxyConfigValidator) validateMavenConfig(config interface{}) error {
	cfg, ok := config.(*config.MavenProxyConfig)
	if !ok {
		return fmt.Errorf("invalid config type, expected *config.MavenProxyConfig")
	}

	// Use the config's own validation method which includes struct tag validation
	return cfg.Validate()
}

// validateNpmConfig validates NPM proxy configuration
func (v *proxyConfigValidator) validateNpmConfig(config interface{}) error {
	cfg, ok := config.(*config.NpmProxyConfig)
	if !ok {
		return fmt.Errorf("invalid config type, expected *config.NpmProxyConfig")
	}

	// Use the config's own validation method which includes struct tag validation
	return cfg.Validate()
}

// validatePipConfig validates PIP proxy configuration
func (v *proxyConfigValidator) validatePipConfig(config interface{}) error {
	cfg, ok := config.(*config.PipProxyConfig)
	if !ok {
		return fmt.Errorf("invalid config type, expected *config.PipProxyConfig")
	}

	// Use the config's own validation method which includes struct tag validation
	return cfg.Validate()
}

// validateYumConfig validates YUM proxy configuration
func (v *proxyConfigValidator) validateYumConfig(config interface{}) error {
	cfg, ok := config.(*config.YumProxyConfig)
	if !ok {
		return fmt.Errorf("invalid config type, expected *config.YumProxyConfig")
	}

	// Use the config's own validation method which includes struct tag validation
	return cfg.Validate()
}

// validateApkConfig validates APK proxy configuration
func (v *proxyConfigValidator) validateApkConfig(config interface{}) error {
	cfg, ok := config.(*config.ApkProxyConfig)
	if !ok {
		return fmt.Errorf("invalid config type, expected *config.ApkProxyConfig")
	}

	// Use the config's own validation method which includes struct tag validation
	return cfg.Validate()
}

// validateHelmConfig validates Helm proxy configuration
func (v *proxyConfigValidator) validateHelmConfig(_ interface{}) error {
	// Helm config not implemented yet
	return nil
}

// validateDockerConfig validates Docker proxy configuration
func (v *proxyConfigValidator) validateDockerConfig(config interface{}) error {
	cfg, ok := config.(*config.DockerProxyConfig)
	if !ok {
		return fmt.Errorf("invalid config type, expected *config.DockerProxyConfig")
	}

	// Use the config's own validation method which includes struct tag validation
	return cfg.Validate()
}
