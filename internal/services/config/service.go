package config

import (
	"context"
	"fmt"
	"path"
	"sync"
	
	"proxynd/configs"
	"proxynd/helpers"
)

// service implements the Service interface
type service struct {
	configDir   string
	mu          sync.RWMutex
	
	// Cached configurations
	globalConfig  *configs.GlobalConfig
	mavenConfig   *configs.MavenProxyConfig
	aptConfig     *configs.AptProxyConfig
	npmConfig     *configs.NpmProxyConfig
	dockerConfig  *configs.DockerProxyConfig
	pipConfig     *configs.PipProxyConfig
	yumConfig     *configs.YumProxyConfig
	apkConfig     *configs.ApkProxyConfig
}

// NewService creates a new configuration service
func NewService(configDir string) (Service, error) {
	if configDir == "" {
		configDir = helpers.GetConfigDir()
	}
	
	s := &service{
		configDir: configDir,
	}
	
	// Load initial configurations
	if err := s.loadAll(); err != nil {
		return nil, fmt.Errorf("failed to load configurations: %w", err)
	}
	
	return s, nil
}

// GetGlobalConfig returns the global configuration
func (s *service) GetGlobalConfig(ctx context.Context) (*configs.GlobalConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.globalConfig == nil {
		return nil, fmt.Errorf("global configuration not loaded")
	}
	
	return s.globalConfig, nil
}

// GetMavenConfig returns the Maven proxy configuration
func (s *service) GetMavenConfig(ctx context.Context) (*configs.MavenProxyConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.mavenConfig == nil {
		return nil, fmt.Errorf("maven configuration not loaded")
	}
	
	return s.mavenConfig, nil
}

// GetAptConfig returns the APT proxy configuration
func (s *service) GetAptConfig(ctx context.Context) (*configs.AptProxyConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.aptConfig == nil {
		return nil, fmt.Errorf("apt configuration not loaded")
	}
	
	return s.aptConfig, nil
}

// GetNpmConfig returns the NPM proxy configuration
func (s *service) GetNpmConfig(ctx context.Context) (*configs.NpmProxyConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.npmConfig == nil {
		return nil, fmt.Errorf("npm configuration not loaded")
	}
	
	return s.npmConfig, nil
}

// GetDockerConfig returns the Docker proxy configuration
func (s *service) GetDockerConfig(ctx context.Context) (*configs.DockerProxyConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.dockerConfig == nil {
		return nil, fmt.Errorf("docker configuration not loaded")
	}
	
	return s.dockerConfig, nil
}

// GetPipConfig returns the PIP proxy configuration
func (s *service) GetPipConfig(ctx context.Context) (*configs.PipProxyConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.pipConfig == nil {
		return nil, fmt.Errorf("pip configuration not loaded")
	}
	
	return s.pipConfig, nil
}

// GetYumConfig returns the YUM proxy configuration
func (s *service) GetYumConfig(ctx context.Context) (*configs.YumProxyConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.yumConfig == nil {
		return nil, fmt.Errorf("yum configuration not loaded")
	}
	
	return s.yumConfig, nil
}

// GetApkConfig returns the APK proxy configuration
func (s *service) GetApkConfig(ctx context.Context) (*configs.ApkProxyConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.apkConfig == nil {
		return nil, fmt.Errorf("apk configuration not loaded")
	}
	
	return s.apkConfig, nil
}

// Reload reloads all configurations
func (s *service) Reload(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	return s.loadAll()
}

// loadAll loads all configurations
func (s *service) loadAll() error {
	loader := configs.NewLoaderRefactored(s.configDir)
	
	// Load global config
	globalConfig, err := loader.LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}
	s.globalConfig = globalConfig
	
	// Load Maven config
	mavenConfig, err := loader.LoadMavenProxyConfig()
	if err != nil {
		return fmt.Errorf("failed to load maven config: %w", err)
	}
	s.mavenConfig = mavenConfig
	
	// Load APT config
	aptConfig, err := loader.LoadAptProxyConfig()
	if err != nil {
		return fmt.Errorf("failed to load apt config: %w", err)
	}
	s.aptConfig = aptConfig
	
	// Load NPM config
	npmConfig, err := loader.LoadNpmProxyConfig()
	if err != nil {
		return fmt.Errorf("failed to load npm config: %w", err)
	}
	s.npmConfig = npmConfig
	
	// Load Docker config
	dockerConfig, err := loader.LoadDockerProxyConfig()
	if err != nil {
		return fmt.Errorf("failed to load docker config: %w", err)
	}
	s.dockerConfig = dockerConfig
	
	// Load PIP config
	pipConfig, err := loader.LoadPipProxyConfig()
	if err != nil {
		return fmt.Errorf("failed to load pip config: %w", err)
	}
	s.pipConfig = pipConfig
	
	// Load YUM config
	yumConfig, err := loader.LoadYumProxyConfig()
	if err != nil {
		return fmt.Errorf("failed to load yum config: %w", err)
	}
	s.yumConfig = yumConfig
	
	// Load APK config
	apkConfig, err := loader.LoadApkProxyConfig()
	if err != nil {
		return fmt.Errorf("failed to load apk config: %w", err)
	}
	s.apkConfig = apkConfig
	
	return nil
}