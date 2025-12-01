package config

import (
	"context"
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"

	"proxynd/internal/config"
	"proxynd/internal/logging"
)

// unifiedService implements the Service interface using UnifiedConfigLoader
type unifiedService struct {
	mu            sync.RWMutex
	configLoader  *config.UnifiedConfigLoader
	unifiedConfig *config.RootConfig
	logger        logging.Logger
	configDir     string
}

// NewUnifiedService creates a new unified configuration service
func NewUnifiedService(ctx context.Context, configDir string) (Service, error) {
	logger := logging.GetLogger()

	if configDir == "" {
		return nil, fmt.Errorf("configDir is required")
	}

	// Create unified config loader
	loader, err := config.NewUnifiedConfigLoader(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create unified config loader: %w", err)
	}

	s := &unifiedService{
		configLoader: loader,
		logger:       logger,
		configDir:    configDir,
	}

	// Load initial configuration
	if err := s.loadConfiguration(ctx); err != nil {
		return nil, fmt.Errorf("failed to load initial configuration: %w", err)
	}

	// Validate configuration
	if err := s.ValidateAll(ctx); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return s, nil
}

// loadConfiguration loads the unified configuration
func (s *unifiedService) loadConfiguration(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	unifiedConfig, err := s.configLoader.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load unified configuration: %w", err)
	}

	s.unifiedConfig = unifiedConfig
	s.logger.Info("Unified configuration loaded successfully")

	return nil
}

// GetGlobalConfig returns the global configuration
func (s *unifiedService) GetGlobalConfig(ctx context.Context) (*config.GlobalConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.unifiedConfig == nil {
		return nil, fmt.Errorf("unified configuration not loaded")
	}

	// Convert RootConfig to GlobalConfig for compatibility
	globalConfig := &config.GlobalConfig{
		ConfigDir:  s.configDir,
		StorageDir: s.unifiedConfig.Cache.File.Directory,
		CacheDir:   s.unifiedConfig.Cache.File.Directory,
		Cache: config.Cache{
			TTL: int(s.unifiedConfig.Cache.TTL.Seconds()),
		},
	}

	return globalConfig, nil
}

// GetMavenConfig returns the Maven proxy configuration
func (s *unifiedService) GetMavenConfig(ctx context.Context) (*config.MavenProxySettings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.unifiedConfig == nil {
		return nil, fmt.Errorf("unified configuration not loaded")
	}

	// Convert from unified config to Maven specific config
	mavenRegistry := s.unifiedConfig.Registries.Maven
	if !mavenRegistry.Enabled {
		return &config.MavenProxySettings{}, nil // Return empty config if disabled
	}

	// Convert repositories to proxies format
	proxies := make([]config.MavenProxyServer, len(mavenRegistry.Repositories))
	for i, repo := range mavenRegistry.Repositories {
		proxies[i] = config.MavenProxyServer{
			ID:      repo.ID,
			Name:    repo.Name,
			URL:     repo.URL,
			Enabled: repo.Releases || repo.Snapshots, // Enabled if either releases or snapshots are enabled
		}
	}

	mavenConfig := &config.MavenProxySettings{
		Path:     "/maven2", // Default Maven path
		UseCache: true,      // Default to use cache
		Proxies:  proxies,
		Cache: config.MavenProxyCacheConfig{
			Enabled: true,
		},
	}

	return mavenConfig, nil
}

// GetAptConfig returns the APT proxy configuration
func (s *unifiedService) GetAptConfig(ctx context.Context) (*config.AptProxyConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.unifiedConfig == nil {
		return nil, fmt.Errorf("unified configuration not loaded")
	}

	// Convert from unified config to APT specific config
	aptRegistry := s.unifiedConfig.Registries.APT // Note: it's APT not Apt
	if !aptRegistry.Enabled {
		return &config.AptProxyConfig{}, nil // Return empty config if disabled
	}

	// For now, return a simple APT config
	// TODO: Implement proper conversion when APT config structure is clarified
	aptConfig := &config.AptProxyConfig{
		// The actual structure will depend on the existing AptProxyConfig definition
	}

	return aptConfig, nil
}

// GetNpmConfig returns the NPM proxy configuration
func (s *unifiedService) GetNpmConfig(ctx context.Context) (*config.NpmProxySettings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.unifiedConfig == nil {
		return nil, fmt.Errorf("unified configuration not loaded")
	}

	// Convert from unified config to NPM specific config
	npmRegistry := s.unifiedConfig.Registries.NPM // Note: it's NPM not Npm
	if !npmRegistry.Enabled {
		return &config.NpmProxySettings{}, nil // Return empty config if disabled
	}

	// Create a default proxy server from upstream
	proxies := make(map[string][]config.NpmProxyServer)
	if npmRegistry.Upstream != "" {
		proxies["default"] = []config.NpmProxyServer{
			{
				Name: "default",
				URL:  npmRegistry.Upstream,
			},
		}
	}

	npmConfig := &config.NpmProxySettings{
		Path:      "/npm", // Default NPM path
		UseCache:  true,   // Default to use cache
		UserCache: npmRegistry.UserCache,
		Proxies:   proxies,
	}

	return npmConfig, nil
}

// GetDockerConfig returns the Docker proxy configuration
func (s *unifiedService) GetDockerConfig(ctx context.Context) (*config.DockerProxySettings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.unifiedConfig == nil {
		return nil, fmt.Errorf("unified configuration not loaded")
	}

	// Convert from unified config to Docker specific config
	dockerRegistry := s.unifiedConfig.Registries.Docker
	if !dockerRegistry.Enabled {
		return &config.DockerProxySettings{}, nil // Return empty config if disabled
	}

	// Convert registry endpoints to proxies
	proxies := make([]config.DockerProxyServer, len(dockerRegistry.Registries))
	for i, registry := range dockerRegistry.Registries {
		proxies[i] = config.DockerProxyServer{
			Name: registry.Name,
			URL:  registry.URL,
			Auth: config.DockerAuth{
				Username: registry.Username,
				Password: registry.Password,
			},
		}
	}

	dockerConfig := &config.DockerProxySettings{
		Path:     "/docker", // Default Docker path
		UseCache: dockerRegistry.UseCache,
		Proxies:  proxies,
	}

	return dockerConfig, nil
}

// GetPipConfig returns the PIP proxy configuration
func (s *unifiedService) GetPipConfig(ctx context.Context) (*config.PipProxySettings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.unifiedConfig == nil {
		return nil, fmt.Errorf("unified configuration not loaded")
	}

	// Convert from unified config to PIP specific config
	pipRegistry := s.unifiedConfig.Registries.PyPI // Note: it's PyPI not Pypi
	if !pipRegistry.Enabled {
		return &config.PipProxySettings{}, nil // Return empty config if disabled
	}

	// Create proxies from upstream configuration
	proxies := []config.PipProxyServer{}
	if pipRegistry.Upstream != "" {
		proxies = append(proxies, config.PipProxyServer{
			Name: "default",
			URL:  pipRegistry.Upstream,
		})
	}

	pipConfig := &config.PipProxySettings{
		Path:     "/pip", // Default PIP path
		UseCache: true,   // Default to use cache
		Proxies:  proxies,
	}

	return pipConfig, nil
}

// GetYumConfig returns the YUM proxy configuration
func (s *unifiedService) GetYumConfig(ctx context.Context) (*config.YumProxySettings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.unifiedConfig == nil {
		return nil, fmt.Errorf("unified configuration not loaded")
	}

	// For now, return empty YUM config since it's not defined in RootConfig
	// TODO: Add YUM configuration to RootConfig structure
	return &config.YumProxySettings{}, nil
}

// GetApkConfig returns the APK proxy configuration
func (s *unifiedService) GetApkConfig(ctx context.Context) (*config.ApkProxySettings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.unifiedConfig == nil {
		return nil, fmt.Errorf("unified configuration not loaded")
	}

	// For now, return empty APK config since it's not defined in RootConfig
	// TODO: Add APK configuration to RootConfig structure
	return &config.ApkProxySettings{}, nil
}

// ValidateAll validates all loaded configurations
func (s *unifiedService) ValidateAll(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.unifiedConfig == nil {
		return fmt.Errorf("no configuration loaded to validate")
	}

	// Basic validation
	if s.unifiedConfig.Server.Port <= 0 || s.unifiedConfig.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", s.unifiedConfig.Server.Port)
	}

	if s.unifiedConfig.Cache.Backend == "" {
		return fmt.Errorf("cache backend is required")
	}

	if s.unifiedConfig.Cache.File.Directory == "" {
		return fmt.Errorf("cache directory is required")
	}

	s.logger.Debug("Configuration validation completed successfully")
	return nil
}

// Reload reloads all configurations
func (s *unifiedService) Reload(ctx context.Context) error {
	s.logger.Info("Reloading unified configuration")

	if err := s.loadConfiguration(ctx); err != nil {
		return fmt.Errorf("failed to reload configuration: %w", err)
	}

	if err := s.ValidateAll(ctx); err != nil {
		return fmt.Errorf("configuration validation failed after reload: %w", err)
	}

	s.logger.Info("Unified configuration reloaded successfully")
	return nil
}

// GetUnifiedConfig returns the full unified configuration (additional method)
func (s *unifiedService) GetUnifiedConfig() *config.RootConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.unifiedConfig
}

// GetRootConfig returns the full RootConfig (interface implementation)
func (s *unifiedService) GetRootConfig(ctx context.Context) (*config.RootConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.unifiedConfig == nil {
		return nil, fmt.Errorf("unified configuration not loaded")
	}

	return s.unifiedConfig, nil
}

// SaveConfig saves the current configuration to disk
func (s *unifiedService) SaveConfig(ctx context.Context) error {
	s.mu.RLock()
	cfg := s.unifiedConfig
	configDir := s.configDir
	s.mu.RUnlock()

	if cfg == nil {
		return fmt.Errorf("no configuration loaded to save")
	}

	// Save to config.yaml in the config directory
	configPath := fmt.Sprintf("%s/config.yaml", configDir)

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	// Write to file with appropriate permissions (0644)
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	s.logger.Info("Configuration saved successfully",
		logging.F("path", configPath))

	return nil
}
