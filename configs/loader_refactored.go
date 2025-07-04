package configs

import (
	"fmt"
	"path"

	"proxynd/helpers"
)

// LoaderRefactored is a generic configuration loader that removes global state access
type LoaderRefactored struct {
	configDir string
}

// NewLoaderRefactored creates a new configuration loader
func NewLoaderRefactored(configDir string) *LoaderRefactored {
	if configDir == "" {
		configDir = helpers.GetConfigDir()
	}
	return &LoaderRefactored{
		configDir: configDir,
	}
}

// LoadGlobalConfig loads the global configuration
func (l *LoaderRefactored) LoadGlobalConfig() (*GlobalConfig, error) {
	cfg := &GlobalConfig{}
	if err := l.loadConfig("global.yaml", cfg); err != nil {
		return nil, fmt.Errorf("failed to load global config: %w", err)
	}
	return cfg, nil
}

// LoadMavenProxyConfig loads the Maven proxy configuration
func (l *LoaderRefactored) LoadMavenProxyConfig() (*MavenProxyConfig, error) {
	cfg := &MavenProxyConfig{}
	if err := l.loadConfig("maven-proxy.yaml", cfg); err != nil {
		return nil, fmt.Errorf("failed to load maven proxy config: %w", err)
	}
	return cfg, nil
}

// LoadAptProxyConfig loads the APT proxy configuration
func (l *LoaderRefactored) LoadAptProxyConfig() (*AptProxyConfig, error) {
	cfg := &AptProxyConfig{}
	if err := l.loadConfig("apt-proxy.yaml", cfg); err != nil {
		return nil, fmt.Errorf("failed to load apt proxy config: %w", err)
	}
	return cfg, nil
}

// LoadNpmProxyConfig loads the NPM proxy configuration
func (l *LoaderRefactored) LoadNpmProxyConfig() (*NpmProxyConfig, error) {
	cfg := &NpmProxyConfig{}
	if err := l.loadConfig("npm-proxy.yaml", cfg); err != nil {
		return nil, fmt.Errorf("failed to load npm proxy config: %w", err)
	}
	return cfg, nil
}

// LoadDockerProxyConfig loads the Docker proxy configuration
func (l *LoaderRefactored) LoadDockerProxyConfig() (*DockerProxyConfig, error) {
	cfg := &DockerProxyConfig{}
	if err := l.loadConfig("docker-proxy.yaml", cfg); err != nil {
		return nil, fmt.Errorf("failed to load docker proxy config: %w", err)
	}
	return cfg, nil
}

// LoadPipProxyConfig loads the PIP proxy configuration
func (l *LoaderRefactored) LoadPipProxyConfig() (*PipProxyConfig, error) {
	cfg := &PipProxyConfig{}
	if err := l.loadConfig("pip-proxy.yaml", cfg); err != nil {
		return nil, fmt.Errorf("failed to load pip proxy config: %w", err)
	}
	return cfg, nil
}

// LoadYumProxyConfig loads the YUM proxy configuration
func (l *LoaderRefactored) LoadYumProxyConfig() (*YumProxyConfig, error) {
	cfg := &YumProxyConfig{}
	if err := l.loadConfig("yum-proxy.yaml", cfg); err != nil {
		return nil, fmt.Errorf("failed to load yum proxy config: %w", err)
	}
	return cfg, nil
}

// LoadApkProxyConfig loads the APK proxy configuration
func (l *LoaderRefactored) LoadApkProxyConfig() (*ApkProxyConfig, error) {
	cfg := &ApkProxyConfig{}
	if err := l.loadConfig("apk-proxy.yaml", cfg); err != nil {
		return nil, fmt.Errorf("failed to load apk proxy config: %w", err)
	}
	return cfg, nil
}

// LoadGemProxyConfig loads the Gem proxy configuration
func (l *LoaderRefactored) LoadGemProxyConfig() (*GemProxyConfig, error) {
	cfg := &GemProxyConfig{}
	if err := l.loadConfig("gem-proxy.yaml", cfg); err != nil {
		return nil, fmt.Errorf("failed to load gem proxy config: %w", err)
	}
	return cfg, nil
}

// ConfigExists checks if a configuration file exists
func (l *LoaderRefactored) ConfigExists(filename string) bool {
	return helpers.FileExists(path.Join(l.configDir, filename))
}

// loadConfig is a generic method to load configuration from file
func (l *LoaderRefactored) loadConfig(filename string, cfg interface{}) error {
	configPath := path.Join(l.configDir, filename)
	
	if !helpers.FileExists(configPath) {
		// Return empty config if file doesn't exist
		return nil
	}
	
	if err := helpers.ReadYamlSafe(configPath, cfg); err != nil {
		return err
	}
	
	return nil
}