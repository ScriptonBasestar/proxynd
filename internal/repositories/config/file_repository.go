// Package config provides configuration repository implementations
package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"

	internalconfig "proxynd/internal/config"
	"proxynd/internal/logging"
)

// Config type constants
const (
	// ConfigTypeGlobal is a const that config type global
	ConfigTypeGlobal = "global"
)

// FileRepository implements configuration storage using the file system
type FileRepository struct {
	configDir string
	logger    logging.Logger
	mu        sync.RWMutex
	watcher   *fsnotify.Watcher
	configs   map[string]interface{} // Cache loaded configs
}

// NewFileRepository creates a new file-based configuration repository
func NewFileRepository(configDir string) (*FileRepository, error) {
	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	// Add config directory to watcher
	if err := watcher.Add(configDir); err != nil {
		_ = watcher.Close()
		return nil, fmt.Errorf("failed to watch config directory: %w", err)
	}

	return &FileRepository{
		configDir: configDir,
		logger:    logging.GetLogger(),
		watcher:   watcher,
		configs:   make(map[string]interface{}),
	}, nil
}

// LoadGlobalConfig loads the global configuration
func (r *FileRepository) LoadGlobalConfig(_ context.Context) (interface{}, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check cache first
	if config, exists := r.configs[ConfigTypeGlobal]; exists {
		return config, nil
	}

	// Load from file
	config := &internalconfig.GlobalConfig{}
	configPath := filepath.Join(r.configDir, "global.yaml")

	if err := r.loadYAMLFile(configPath, config); err != nil {
		// Try alternative names
		altPaths := []string{
			filepath.Join(r.configDir, "global.yml"),
			filepath.Join(r.configDir, "config.yaml"),
			filepath.Join(r.configDir, "config.yml"),
		}

		loaded := false
		for _, altPath := range altPaths {
			if err := r.loadYAMLFile(altPath, config); err == nil {
				loaded = true
				break
			}
		}

		if !loaded {
			return nil, fmt.Errorf("failed to load global config: %w", err)
		}
	}

	// Cache the config
	r.configs[ConfigTypeGlobal] = config

	return config, nil
}

// LoadProxyConfig loads configuration for a specific proxy type
func (r *FileRepository) LoadProxyConfig(_ context.Context, proxyType string) (interface{}, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check cache first
	if config, exists := r.configs[proxyType]; exists {
		return config, nil
	}

	// Create appropriate config struct based on type
	var cfg interface{}
	switch proxyType {
	case "apt":
		cfg = &internalconfig.AptProxyConfig{}
	case "maven":
		cfg = &internalconfig.MavenProxySettings{}
	case "npm":
		cfg = &internalconfig.NpmProxySettings{}
	case "docker":
		cfg = &internalconfig.DockerProxySettings{}
	case "pip":
		cfg = &internalconfig.PipProxySettings{}
	case "yum":
		cfg = &internalconfig.YumProxySettings{}
	case "apk":
		cfg = &internalconfig.ApkProxySettings{}
	default:
		return nil, fmt.Errorf("unsupported proxy type: %s", proxyType)
	}

	// Try to load config file
	configPath := filepath.Join(r.configDir, fmt.Sprintf("%s-proxy.yaml", proxyType))

	if err := r.loadYAMLFile(configPath, cfg); err != nil {
		// Try alternative names
		altPath := filepath.Join(r.configDir, fmt.Sprintf("%s-proxy.yml", proxyType))
		if err := r.loadYAMLFile(altPath, cfg); err != nil {
			r.logger.Warn("Failed to load proxy config",
				logging.F("proxy_type", proxyType),
				logging.F("error", err))
			return cfg, nil // Return empty config
		}
	}

	// Cache the config
	r.configs[proxyType] = cfg

	return cfg, nil
}

// SaveGlobalConfig saves the global configuration
func (r *FileRepository) SaveGlobalConfig(_ context.Context, config interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	configPath := filepath.Join(r.configDir, "global.yaml")

	if err := r.saveYAMLFile(configPath, config); err != nil {
		return fmt.Errorf("failed to save global config: %w", err)
	}

	// Update cache
	r.configs[ConfigTypeGlobal] = config

	return nil
}

// SaveProxyConfig saves configuration for a specific proxy type
func (r *FileRepository) SaveProxyConfig(_ context.Context, proxyType string, config interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	configPath := filepath.Join(r.configDir, fmt.Sprintf("%s-proxy.yaml", proxyType))

	if err := r.saveYAMLFile(configPath, config); err != nil {
		return fmt.Errorf("failed to save %s config: %w", proxyType, err)
	}

	// Update cache
	r.configs[proxyType] = config

	return nil
}

// ListProxyTypes returns all configured proxy types
func (r *FileRepository) ListProxyTypes(_ context.Context) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var proxyTypes []string

	// Scan config directory for proxy config files
	entries, err := os.ReadDir(r.configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Match files like "apt-proxy.yaml" or "maven-proxy.yml"
		if strings.HasSuffix(name, "-proxy.yaml") || strings.HasSuffix(name, "-proxy.yml") {
			proxyType := strings.TrimSuffix(name, "-proxy.yaml")
			proxyType = strings.TrimSuffix(proxyType, "-proxy.yml")
			proxyTypes = append(proxyTypes, proxyType)
		}
	}

	return proxyTypes, nil
}

// ValidateConfig validates a configuration object
func (r *FileRepository) ValidateConfig(_ context.Context, proxyType string, config interface{}) error {
	// Basic validation - ensure config is not nil
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Type-specific validation
	switch proxyType {
	case ConfigTypeGlobal:
		globalConfig, ok := config.(*internalconfig.GlobalConfig)
		if !ok {
			return fmt.Errorf("invalid config type for global config")
		}
		// Add specific validation for global config
		if globalConfig.CacheDir == "" && globalConfig.StorageDir == "" {
			return fmt.Errorf("either cache_dir or storage_dir must be specified")
		}

	case "maven", "apt", "npm", "docker", "pip", "yum", "apk":
		// Add proxy-specific validation as needed
		// For now, just ensure it's the right type

	default:
		return fmt.Errorf("unknown proxy type: %s", proxyType)
	}

	return nil
}

// WatchConfig watches for configuration changes
func (r *FileRepository) WatchConfig(ctx context.Context, callback func(proxyType string, config interface{})) error {
	go func() {
		for {
			select {
			case event, ok := <-r.watcher.Events:
				if !ok {
					return
				}

				if event.Op&fsnotify.Write == fsnotify.Write {
					r.handleConfigChange(event.Name, callback)
				}

			case err, ok := <-r.watcher.Errors:
				if !ok {
					return
				}
				r.logger.Error("Config watcher error", logging.F("error", err))

			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// GetConfigPath returns the path to configuration files
func (r *FileRepository) GetConfigPath() string {
	return r.configDir
}

// Close closes the file watcher
func (r *FileRepository) Close() error {
	return r.watcher.Close()
}

// Helper methods

func (r *FileRepository) loadYAMLFile(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, v)
}

func (r *FileRepository) saveYAMLFile(path string, v interface{}) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

func (r *FileRepository) handleConfigChange(filename string, callback func(proxyType string, config interface{})) {
	basename := filepath.Base(filename)

	// Determine proxy type from filename
	var proxyType string
	if basename == "global.yaml" || basename == "global.yml" {
		proxyType = ConfigTypeGlobal
	} else if strings.HasSuffix(basename, "-proxy.yaml") || strings.HasSuffix(basename, "-proxy.yml") {
		proxyType = strings.TrimSuffix(basename, "-proxy.yaml")
		proxyType = strings.TrimSuffix(proxyType, "-proxy.yml")
	} else {
		return // Not a config file we care about
	}

	// Reload the config
	r.mu.Lock()
	delete(r.configs, proxyType) // Clear cache
	r.mu.Unlock()

	// Load new config
	var config interface{}
	var err error

	if proxyType == ConfigTypeGlobal {
		config, err = r.LoadGlobalConfig(context.Background())
	} else {
		config, err = r.LoadProxyConfig(context.Background(), proxyType)
	}

	if err != nil {
		r.logger.Error("Failed to reload config",
			logging.F("proxy_type", proxyType),
			logging.F("error", err))
		return
	}

	// Notify callback
	callback(proxyType, config)

	r.logger.Info("Configuration reloaded",
		logging.F("proxy_type", proxyType),
		logging.F("file", filename))
}
