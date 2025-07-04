package config

import (
	"context"
	
	"proxynd/configs"
)

// Service provides access to configuration without global state
type Service interface {
	// GetGlobalConfig returns the global configuration
	GetGlobalConfig(ctx context.Context) (*configs.GlobalConfig, error)
	
	// GetMavenConfig returns the Maven proxy configuration
	GetMavenConfig(ctx context.Context) (*configs.MavenProxyConfig, error)
	
	// GetAptConfig returns the APT proxy configuration
	GetAptConfig(ctx context.Context) (*configs.AptProxyConfig, error)
	
	// GetNpmConfig returns the NPM proxy configuration
	GetNpmConfig(ctx context.Context) (*configs.NpmProxyConfig, error)
	
	// GetDockerConfig returns the Docker proxy configuration
	GetDockerConfig(ctx context.Context) (*configs.DockerProxyConfig, error)
	
	// GetPipConfig returns the PIP proxy configuration
	GetPipConfig(ctx context.Context) (*configs.PipProxyConfig, error)
	
	// GetYumConfig returns the YUM proxy configuration
	GetYumConfig(ctx context.Context) (*configs.YumProxyConfig, error)
	
	// GetApkConfig returns the APK proxy configuration
	GetApkConfig(ctx context.Context) (*configs.ApkProxyConfig, error)
	
	// Reload reloads all configurations
	Reload(ctx context.Context) error
}

// ConfigLoader is a function that loads a specific configuration
type ConfigLoader interface {
	Load(configDir string) error
}