package configs

import (
	"context"
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
func (l *LoaderRefactored) LoadGlobalConfig(ctx context.Context) (*GlobalConfig, error) {
	cfg := &GlobalConfig{}
	if err := l.loadConfig(ctx, "global.yaml", cfg); err != nil {
		return nil, fmt.Errorf("failed to load global config: %w", err)
	}
	return cfg, nil
}

// LoadMavenProxyConfig loads the Maven proxy configuration
func (l *LoaderRefactored) LoadMavenProxyConfig(ctx context.Context) (*MavenProxyConfig, error) {
	cfg := &MavenProxyConfig{}
	if err := l.loadConfig(ctx, "maven-proxy.yaml", cfg); err != nil {
		return nil, fmt.Errorf("failed to load maven proxy config: %w", err)
	}
	return cfg, nil
}

// LoadAptProxyConfig loads the APT proxy configuration
func (l *LoaderRefactored) LoadAptProxyConfig(ctx context.Context) (*AptProxyConfig, error) {
	cfg := &AptProxyConfig{}
	if err := l.loadConfig(ctx, "apt-proxy.yaml", cfg); err != nil {
		return nil, fmt.Errorf("failed to load apt proxy config: %w", err)
	}
	return cfg, nil
}

// LoadNpmProxyConfig loads the NPM proxy configuration
func (l *LoaderRefactored) LoadNpmProxyConfig(ctx context.Context) (*NpmProxyConfig, error) {
	cfg := &NpmProxyConfig{}
	if err := l.loadConfig(ctx, "npm-proxy.yaml", cfg); err != nil {
		return nil, fmt.Errorf("failed to load npm proxy config: %w", err)
	}
	return cfg, nil
}

// LoadDockerProxyConfig loads the Docker proxy configuration
func (l *LoaderRefactored) LoadDockerProxyConfig(ctx context.Context) (*DockerProxyConfig, error) {
	cfg := &DockerProxyConfig{}
	if err := l.loadConfig(ctx, "docker-proxy.yaml", cfg); err != nil {
		return nil, fmt.Errorf("failed to load docker proxy config: %w", err)
	}
	return cfg, nil
}

// LoadPipProxyConfig loads the PIP proxy configuration
func (l *LoaderRefactored) LoadPipProxyConfig(ctx context.Context) (*PipProxyConfig, error) {
	cfg := &PipProxyConfig{}
	if err := l.loadConfig(ctx, "pip-proxy.yaml", cfg); err != nil {
		return nil, fmt.Errorf("failed to load pip proxy config: %w", err)
	}
	return cfg, nil
}

// LoadYumProxyConfig loads the YUM proxy configuration
func (l *LoaderRefactored) LoadYumProxyConfig(ctx context.Context) (*YumProxyConfig, error) {
	cfg := &YumProxyConfig{}
	if err := l.loadConfig(ctx, "yum-proxy.yaml", cfg); err != nil {
		return nil, fmt.Errorf("failed to load yum proxy config: %w", err)
	}
	return cfg, nil
}

// LoadApkProxyConfig loads the APK proxy configuration
func (l *LoaderRefactored) LoadApkProxyConfig(ctx context.Context) (*ApkProxyConfig, error) {
	cfg := &ApkProxyConfig{}
	if err := l.loadConfig(ctx, "apk-proxy.yaml", cfg); err != nil {
		return nil, fmt.Errorf("failed to load apk proxy config: %w", err)
	}
	return cfg, nil
}

// LoadGemProxyConfig loads the Gem proxy configuration
func (l *LoaderRefactored) LoadGemProxyConfig(ctx context.Context) (*GemProxyConfig, error) {
	cfg := &GemProxyConfig{}
	if err := l.loadConfig(ctx, "gem-proxy.yaml", cfg); err != nil {
		return nil, fmt.Errorf("failed to load gem proxy config: %w", err)
	}
	return cfg, nil
}

// ConfigExists checks if a configuration file exists
func (l *LoaderRefactored) ConfigExists(filename string) bool {
	return helpers.FileExists(path.Join(l.configDir, filename))
}

// loadConfig is a generic method to load configuration from file
func (l *LoaderRefactored) loadConfig(ctx context.Context, filename string, cfg interface{}) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	configPath := path.Join(l.configDir, filename)
	
	if !helpers.FileExists(configPath) {
		// Return empty config if file doesn't exist
		return nil
	}
	
	// Read file with context awareness
	readDone := make(chan error, 1)
	go func() {
		readDone <- helpers.ReadYamlSafe(configPath, cfg)
	}()
	
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-readDone:
		return err
	}
}