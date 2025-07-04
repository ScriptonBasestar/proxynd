package config

import (
	"fmt"
	"proxynd/configs"
)

// globalConfigValidator validates global configuration
type globalConfigValidator struct{}

// Validate validates the global configuration
func (v *globalConfigValidator) Validate(config interface{}) error {
	cfg, ok := config.(*configs.GlobalConfig)
	if !ok {
		return fmt.Errorf("invalid config type, expected *configs.GlobalConfig")
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
	cfg, ok := config.(*configs.AptProxyConfig)
	if !ok {
		return fmt.Errorf("invalid config type, expected *configs.AptProxyConfig")
	}
	
	// 경로 검증
	if cfg.Path == "" {
		return fmt.Errorf("apt proxy path cannot be empty")
	}
	
	// 프록시 서버 검증
	if cfg.Proxies == nil || len(cfg.Proxies) == 0 {
		return fmt.Errorf("at least one apt proxy server must be configured")
	}
	
	// 각 OS 타입별 프록시 검증
	for osType, servers := range cfg.Proxies {
		if len(servers) == 0 {
			return fmt.Errorf("no servers configured for OS type: %s", osType)
		}
		
		for i, server := range servers {
			if server.URL == "" {
				return fmt.Errorf("server URL cannot be empty for OS type %s, server index %d", osType, i)
			}
			if server.Name == "" {
				return fmt.Errorf("server name cannot be empty for OS type %s, server index %d", osType, i)
			}
		}
	}
	
	return nil
}

// validateMavenConfig validates Maven proxy configuration
func (v *proxyConfigValidator) validateMavenConfig(config interface{}) error {
	cfg, ok := config.(*configs.MavenProxyConfig)
	if !ok {
		return fmt.Errorf("invalid config type, expected *configs.MavenProxyConfig")
	}
	
	// 경로 검증
	if cfg.Path == "" {
		return fmt.Errorf("maven proxy path cannot be empty")
	}
	
	// 프록시 서버 검증
	if len(cfg.Proxies) == 0 {
		return fmt.Errorf("at least one maven proxy server must be configured")
	}
	
	for i, proxy := range cfg.Proxies {
		if proxy.Url == "" {
			return fmt.Errorf("proxy URL cannot be empty at index %d", i)
		}
		if proxy.Name == "" {
			return fmt.Errorf("proxy name cannot be empty at index %d", i)
		}
	}
	
	return nil
}

// validateNpmConfig validates NPM proxy configuration
func (v *proxyConfigValidator) validateNpmConfig(config interface{}) error {
	cfg, ok := config.(*configs.NpmProxyConfig)
	if !ok {
		return fmt.Errorf("invalid config type, expected *configs.NpmProxyConfig")
	}
	
	// 경로 검증
	if cfg.Path == "" {
		return fmt.Errorf("npm proxy path cannot be empty")
	}
	
	// 프록시 서버 검증
	if cfg.Proxies == nil || len(cfg.Proxies) == 0 {
		return fmt.Errorf("at least one npm proxy configuration must be defined")
	}
	
	// default 프록시 검증
	defaultProxies, exists := cfg.Proxies["default"]
	if !exists || len(defaultProxies) == 0 {
		return fmt.Errorf("default npm proxy configuration must be defined")
	}
	
	for registryName, proxies := range cfg.Proxies {
		for i, proxy := range proxies {
			if proxy.URL == "" {
				return fmt.Errorf("proxy URL cannot be empty for registry %s at index %d", registryName, i)
			}
			if proxy.Name == "" {
				return fmt.Errorf("proxy name cannot be empty for registry %s at index %d", registryName, i)
			}
		}
	}
	
	return nil
}

// validatePipConfig validates PIP proxy configuration
func (v *proxyConfigValidator) validatePipConfig(config interface{}) error {
	cfg, ok := config.(*configs.PipProxyConfig)
	if !ok {
		return fmt.Errorf("invalid config type, expected *configs.PipProxyConfig")
	}
	
	// 경로 검증
	if cfg.Path == "" {
		return fmt.Errorf("pip proxy path cannot be empty")
	}
	
	// 프록시 서버 검증
	if len(cfg.Proxies) == 0 {
		return fmt.Errorf("at least one pip proxy server must be configured")
	}
	
	for i, proxy := range cfg.Proxies {
		if proxy.URL == "" {
			return fmt.Errorf("proxy URL cannot be empty at index %d", i)
		}
		if proxy.Name == "" {
			return fmt.Errorf("proxy name cannot be empty at index %d", i)
		}
	}
	
	return nil
}

// validateYumConfig validates YUM proxy configuration
func (v *proxyConfigValidator) validateYumConfig(config interface{}) error {
	cfg, ok := config.(*configs.YumProxyConfig)
	if !ok {
		return fmt.Errorf("invalid config type, expected *configs.YumProxyConfig")
	}
	
	// 경로 검증
	if cfg.Path == "" {
		return fmt.Errorf("yum proxy path cannot be empty")
	}
	
	// 프록시 서버 검증
	if len(cfg.Proxies) == 0 {
		return fmt.Errorf("at least one yum proxy server must be configured")
	}
	
	for i, proxy := range cfg.Proxies {
		if proxy.Url == "" {
			return fmt.Errorf("proxy URL cannot be empty at index %d", i)
		}
		if proxy.Name == "" {
			return fmt.Errorf("proxy name cannot be empty at index %d", i)
		}
	}
	
	return nil
}

// validateApkConfig validates APK proxy configuration
func (v *proxyConfigValidator) validateApkConfig(config interface{}) error {
	cfg, ok := config.(*configs.ApkProxyConfig)
	if !ok {
		return fmt.Errorf("invalid config type, expected *configs.ApkProxyConfig")
	}
	
	// 경로 검증
	if cfg.Path == "" {
		return fmt.Errorf("apk proxy path cannot be empty")
	}
	
	// 프록시 서버 검증
	if len(cfg.Proxies) == 0 {
		return fmt.Errorf("at least one apk proxy server must be configured")
	}
	
	for i, proxy := range cfg.Proxies {
		if proxy.Url == "" {
			return fmt.Errorf("proxy URL cannot be empty at index %d", i)
		}
		if proxy.Name == "" {
			return fmt.Errorf("proxy name cannot be empty at index %d", i)
		}
	}
	
	// 검증 설정 검증
	if cfg.Verification.Enabled && cfg.Verification.KeyDirectory == "" {
		return fmt.Errorf("key directory must be specified when verification is enabled")
	}
	
	return nil
}

// validateHelmConfig validates Helm proxy configuration
func (v *proxyConfigValidator) validateHelmConfig(config interface{}) error {
	cfg, ok := config.(*configs.HelmProxyConfig)
	if !ok {
		return fmt.Errorf("invalid config type, expected *configs.HelmProxyConfig")
	}
	
	// 경로 검증
	if cfg.Path == "" {
		return fmt.Errorf("helm proxy path cannot be empty")
	}
	
	// 레포지토리 검증
	if len(cfg.Repositories) == 0 {
		return fmt.Errorf("at least one helm repository must be configured")
	}
	
	for name, repo := range cfg.Repositories {
		if repo.URL == "" {
			return fmt.Errorf("repository URL cannot be empty for repository: %s", name)
		}
	}
	
	return nil
}

// validateDockerConfig validates Docker proxy configuration
func (v *proxyConfigValidator) validateDockerConfig(config interface{}) error {
	cfg, ok := config.(*configs.DockerProxyConfig)
	if !ok {
		return fmt.Errorf("invalid config type, expected *configs.DockerProxyConfig")
	}
	
	// 경로 검증
	if cfg.Path == "" {
		return fmt.Errorf("docker proxy path cannot be empty")
	}
	
	// 레지스트리 검증
	if len(cfg.Registries) == 0 {
		return fmt.Errorf("at least one docker registry must be configured")
	}
	
	for name, registry := range cfg.Registries {
		if registry.URL == "" {
			return fmt.Errorf("registry URL cannot be empty for registry: %s", name)
		}
	}
	
	return nil
}