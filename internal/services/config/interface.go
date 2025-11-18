// Package config provides configuration service interfaces
package config

//go:generate mockery --name Service --output ./mocks --outpkg mocks --filename service.go
//go:generate mockery --name Validator --output ./mocks --outpkg mocks --filename validator.go
//go:generate mockery --name ConfigLoader --output ./mocks --outpkg mocks --filename config_loader.go

import (
	"context"

	"proxynd/internal/config"
)

// Service provides access to configuration without global state
type Service interface {
	// GetGlobalConfig returns the global configuration
	GetGlobalConfig(ctx context.Context) (*config.GlobalConfig, error)

	// GetMavenConfig returns the Maven proxy configuration
	GetMavenConfig(ctx context.Context) (*config.MavenProxySettings, error)

	// GetAptConfig returns the APT proxy configuration
	GetAptConfig(ctx context.Context) (*config.AptProxyConfig, error)

	// GetNpmConfig returns the NPM proxy configuration
	GetNpmConfig(ctx context.Context) (*config.NpmProxySettings, error)

	// GetDockerConfig returns the Docker proxy configuration
	GetDockerConfig(ctx context.Context) (*config.DockerProxySettings, error)

	// GetPipConfig returns the PIP proxy configuration
	GetPipConfig(ctx context.Context) (*config.PipProxySettings, error)

	// GetYumConfig returns the YUM proxy configuration
	GetYumConfig(ctx context.Context) (*config.YumProxySettings, error)

	// GetApkConfig returns the APK proxy configuration
	GetApkConfig(ctx context.Context) (*config.ApkProxySettings, error)

	// ValidateAll validates all loaded configurations
	ValidateAll(ctx context.Context) error

	// Reload reloads all configurations
	Reload(ctx context.Context) error

	// SaveConfig saves the current configuration to disk
	SaveConfig(ctx context.Context) error

	// GetRootConfig returns the full RootConfig (for direct access)
	GetRootConfig(ctx context.Context) (*config.RootConfig, error)
}

// Validator interface for configuration validation
type Validator interface {
	Validate(config interface{}) error
}

// ConfigLoader is a function that loads a specific configuration
type ConfigLoader interface {
	Load(configDir string) error
}
