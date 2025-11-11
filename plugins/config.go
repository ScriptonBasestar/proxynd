package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the root configuration for the plugin system
type Config struct {
	Enabled        bool                            `yaml:"enabled" json:"enabled"`
	DiscoveryPaths []string                        `yaml:"discoveryPaths" json:"discoveryPaths"`
	Lifecycle      LifecycleConfig                 `yaml:"lifecycle" json:"lifecycle"`
	Registry       RegistryConfig                  `yaml:"registry" json:"registry"`
	Environments   map[string]EnvironmentOverrides `yaml:"environments" json:"environments"`
}

// LifecycleConfig controls lifecycle hook behavior
type LifecycleConfig struct {
	InitTimeout     time.Duration `yaml:"initTimeout" json:"initTimeout"`
	ReadyTimeout    time.Duration `yaml:"readyTimeout" json:"readyTimeout"`
	ShutdownTimeout time.Duration `yaml:"shutdownTimeout" json:"shutdownTimeout"`
	FailurePolicy   FailurePolicy `yaml:"failurePolicy" json:"failurePolicy"`
}

// FailurePolicy defines how to handle lifecycle failures
type FailurePolicy string

const (
	// FailurePolicyContinue logs error and continues
	FailurePolicyContinue FailurePolicy = "continue"
	// FailurePolicyHalt stops application startup
	FailurePolicyHalt FailurePolicy = "halt"
	// FailurePolicyWarn logs warning and continues
	FailurePolicyWarn FailurePolicy = "warn"
)

// RegistryConfig organizes plugins by edition
type RegistryConfig struct {
	Core       []PluginConfig `yaml:"core" json:"core"`
	Enterprise []PluginConfig `yaml:"enterprise" json:"enterprise"`
	Cloud      []PluginConfig `yaml:"cloud" json:"cloud"`
}

// PluginConfig defines a single plugin's configuration
type PluginConfig struct {
	Name     string                 `yaml:"name" json:"name"`
	Enabled  bool                   `yaml:"enabled" json:"enabled"`
	Priority int                    `yaml:"priority" json:"priority"` // Lower = earlier
	Config   map[string]interface{} `yaml:"config" json:"config"`

	// Runtime fields (not from config)
	instance Plugin
}

// SetInstance sets the runtime plugin instance
func (pc *PluginConfig) SetInstance(p Plugin) {
	pc.instance = p
}

// GetInstance returns the runtime plugin instance
func (pc *PluginConfig) GetInstance() Plugin {
	return pc.instance
}

// EnvironmentOverrides defines environment-specific configuration overrides
type EnvironmentOverrides struct {
	Overrides []PluginOverride `yaml:"overrides" json:"overrides"`
}

// PluginOverride specifies what to override for a plugin
type PluginOverride struct {
	Plugin  string                 `yaml:"plugin" json:"plugin"`
	Enabled *bool                  `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	Config  map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`
}

// DefaultConfig returns the default plugin configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:        true,
		DiscoveryPaths: []string{"./plugins", "/etc/proxynd/plugins"},
		Lifecycle: LifecycleConfig{
			InitTimeout:     30 * time.Second,
			ReadyTimeout:    10 * time.Second,
			ShutdownTimeout: 30 * time.Second,
			FailurePolicy:   FailurePolicyContinue,
		},
		Registry: RegistryConfig{
			Core:       []PluginConfig{},
			Enterprise: []PluginConfig{},
			Cloud:      []PluginConfig{},
		},
		Environments: map[string]EnvironmentOverrides{},
	}
}

// MergeConfig merges two configurations (base + override)
func MergeConfig(base, override *Config) *Config {
	if override == nil {
		return base
	}

	result := *base

	if override.Enabled != base.Enabled {
		result.Enabled = override.Enabled
	}

	if len(override.DiscoveryPaths) > 0 {
		result.DiscoveryPaths = override.DiscoveryPaths
	}

	// Merge lifecycle
	if override.Lifecycle.InitTimeout != 0 {
		result.Lifecycle.InitTimeout = override.Lifecycle.InitTimeout
	}
	if override.Lifecycle.ReadyTimeout != 0 {
		result.Lifecycle.ReadyTimeout = override.Lifecycle.ReadyTimeout
	}
	if override.Lifecycle.ShutdownTimeout != 0 {
		result.Lifecycle.ShutdownTimeout = override.Lifecycle.ShutdownTimeout
	}
	if override.Lifecycle.FailurePolicy != "" {
		result.Lifecycle.FailurePolicy = override.Lifecycle.FailurePolicy
	}

	// Merge registry
	if len(override.Registry.Core) > 0 {
		result.Registry.Core = override.Registry.Core
	}
	if len(override.Registry.Enterprise) > 0 {
		result.Registry.Enterprise = override.Registry.Enterprise
	}
	if len(override.Registry.Cloud) > 0 {
		result.Registry.Cloud = override.Registry.Cloud
	}

	// Merge environments
	if len(override.Environments) > 0 {
		result.Environments = override.Environments
	}

	return &result
}

// LoadConfigFromFile loads plugin configuration from a YAML file
func LoadConfigFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return &config, nil
}

// LoadConfigFromDir looks for plugins.yaml in the given directory
func LoadConfigFromDir(dir string) (*Config, error) {
	path := filepath.Join(dir, "plugins.yaml")

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Not found - return default config
		return DefaultConfig(), nil
	}

	return LoadConfigFromFile(path)
}

// LoadConfig tries to load plugin config from multiple locations
// Priority: explicit path > config dir > default config
func LoadConfig(configDir string) (*Config, error) {
	// Try loading from config directory
	if configDir != "" {
		config, err := LoadConfigFromDir(configDir)
		if err != nil {
			return nil, fmt.Errorf("failed to load config from %s: %w", configDir, err)
		}
		return config, nil
	}

	// Fall back to defaults
	return DefaultConfig(), nil
}
