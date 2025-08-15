package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

// UnifiedConfigLoader provides struct-based configuration loading with environment variable validation
type UnifiedConfigLoader struct {
	configDir string
	config    *RootConfig
}

// NewUnifiedConfigLoader creates a new unified configuration loader
func NewUnifiedConfigLoader(configDir string) (*UnifiedConfigLoader, error) {
	if configDir == "" {
		configDir = os.Getenv("CONFIG_DIR")
		if configDir == "" {
			return nil, fmt.Errorf("CONFIG_DIR environment variable is required")
		}
	}

	// Validate that config directory exists
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("config directory does not exist: %s", configDir)
	}

	return &UnifiedConfigLoader{
		configDir: configDir,
	}, nil
}

// LoadConfig loads the complete configuration with environment variable validation
func (loader *UnifiedConfigLoader) LoadConfig() (*RootConfig, error) {
	if err := loader.validateRequiredEnvVars(); err != nil {
		return nil, fmt.Errorf("environment variable validation failed: %w", err)
	}

	if err := loader.loadConfigFile(); err != nil {
		return nil, fmt.Errorf("failed to load configuration file: %w", err)
	}

	if err := loader.setDefaults(); err != nil {
		return nil, fmt.Errorf("failed to set configuration defaults: %w", err)
	}

	if err := loader.validateConfig(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return loader.config, nil
}

// validateRequiredEnvVars validates that all required environment variables are set
func (loader *UnifiedConfigLoader) validateRequiredEnvVars() error {
	requiredVars := []struct {
		name        string
		description string
		required    bool
		defaultVal  string
	}{
		{"CONFIG_DIR", "Configuration directory path", true, ""},
		{"STORAGE_DIR", "Storage directory path for cache and data", true, ""},
		{"SERVER_PORT", "Server port number", false, "8080"},
		{"LOG_LEVEL", "Logging level (debug, info, warn, error)", false, "info"},
		{"LOG_FORMAT", "Log format (json, text)", false, "json"},
	}

	for _, envVar := range requiredVars {
		value := os.Getenv(envVar.name)
		if value == "" {
			if envVar.required {
				return fmt.Errorf("required environment variable %s is not set (%s)", envVar.name, envVar.description)
			}
			// Set default value for optional variables
			if envVar.defaultVal != "" {
				os.Setenv(envVar.name, envVar.defaultVal)
				value = envVar.defaultVal
			}
		}

		// Special validation for SERVER_PORT
		if envVar.name == "SERVER_PORT" && value != "" {
			if _, err := strconv.Atoi(value); err != nil {
				return fmt.Errorf("SERVER_PORT must be a valid integer, got: %s", value)
			}
		}

		// Special validation for directory paths
		if (envVar.name == "CONFIG_DIR" || envVar.name == "STORAGE_DIR") && value != "" {
			if !filepath.IsAbs(value) {
				// Convert relative paths to absolute
				absPath, err := filepath.Abs(value)
				if err != nil {
					return fmt.Errorf("failed to resolve absolute path for %s: %w", envVar.name, err)
				}
				os.Setenv(envVar.name, absPath)
			}
		}

		// Special validation for LOG_LEVEL
		if envVar.name == "LOG_LEVEL" && value != "" {
			validLevels := []string{"debug", "info", "warn", "error"}
			valid := false
			for _, level := range validLevels {
				if value == level {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("LOG_LEVEL must be one of: debug, info, warn, error; got: %s", value)
			}
		}

		// Special validation for LOG_FORMAT
		if envVar.name == "LOG_FORMAT" && value != "" {
			validFormats := []string{"json", "text"}
			valid := false
			for _, format := range validFormats {
				if value == format {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("LOG_FORMAT must be one of: json, text; got: %s", value)
			}
		}
	}

	return nil
}

// loadConfigFile loads configuration from YAML file using Viper
func (loader *UnifiedConfigLoader) loadConfigFile() error {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(loader.configDir)

	// Set up environment variable bindings
	v.AutomaticEnv()
	
	// Bind specific environment variables
	envBindings := map[string]string{
		"server.port":           "SERVER_PORT",
		"cache.file.directory":  "STORAGE_DIR",
		"cache.redis.address":   "REDIS_ADDRESS",
		"cache.redis.password":  "REDIS_PASSWORD",
		"cache.s3.access_key_id": "AWS_ACCESS_KEY_ID",
		"cache.s3.secret_access_key": "AWS_SECRET_ACCESS_KEY",
		"cache.s3.region":       "AWS_REGION",
		"cache.s3.bucket":       "S3_BUCKET",
		"logging.level":         "LOG_LEVEL",
		"logging.format":        "LOG_FORMAT",
	}

	for configKey, envKey := range envBindings {
		v.BindEnv(configKey, envKey)
	}

	// Try to read the config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, use defaults and environment variables
			fmt.Printf("Warning: config file not found in %s, using environment variables and defaults\n", loader.configDir)
		} else {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal into struct
	loader.config = &RootConfig{}
	if err := v.Unmarshal(loader.config); err != nil {
		return fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	return nil
}

// setDefaults sets default values for configuration
func (loader *UnifiedConfigLoader) setDefaults() error {
	config := loader.config

	// Initialize struct to avoid nil pointer dereferences
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	// Server defaults
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}
	if config.Server.Port == 0 {
		// Use default or environment variable
		portStr := os.Getenv("SERVER_PORT")
		if portStr == "" {
			portStr = "8080" // Default fallback
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("failed to parse SERVER_PORT: %w", err)
		}
		config.Server.Port = port
	}
	if config.Server.ReadTimeout == 0 {
		config.Server.ReadTimeout = 30 * time.Second
	}
	if config.Server.WriteTimeout == 0 {
		config.Server.WriteTimeout = 30 * time.Second
	}
	if config.Server.IdleTimeout == 0 {
		config.Server.IdleTimeout = 120 * time.Second
	}

	// Cache defaults
	if config.Cache.Backend == "" {
		config.Cache.Backend = "file"
	}
	if config.Cache.TTL == 0 {
		config.Cache.TTL = 3600 * time.Second // Use time.Duration properly
	}
	if config.Cache.File.Directory == "" {
		storageDir := os.Getenv("STORAGE_DIR")
		if storageDir == "" {
			storageDir = "./tmp/storage" // Default fallback
		}
		config.Cache.File.Directory = storageDir
	}
	if config.Cache.MaxSize == "" {
		config.Cache.MaxSize = "10GB"
	}
	if config.Cache.CleanupInterval == 0 {
		config.Cache.CleanupInterval = 1 * time.Hour
	}
	if config.Cache.EvictionPolicy == "" {
		config.Cache.EvictionPolicy = "lru"
	}

	// Logging defaults
	if config.Logging.Level == "" {
		logLevel := os.Getenv("LOG_LEVEL")
		if logLevel == "" {
			logLevel = "info" // Default fallback
		}
		config.Logging.Level = logLevel
	}
	if config.Logging.Format == "" {
		logFormat := os.Getenv("LOG_FORMAT")
		if logFormat == "" {
			logFormat = "json" // Default fallback
		}
		config.Logging.Format = logFormat
	}
	if config.Logging.Output == "" {
		config.Logging.Output = "stdout"
	}

	// Metrics defaults
	if config.Metrics.Path == "" {
		config.Metrics.Path = "/metrics"
	}

	return nil
}

// validateConfig validates the loaded configuration
func (loader *UnifiedConfigLoader) validateConfig() error {
	config := loader.config

	// Validate server configuration
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	// Validate cache directory exists or can be created
	cacheDir := config.Cache.File.Directory
	if cacheDir != "" {
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			return fmt.Errorf("failed to create cache directory %s: %w", cacheDir, err)
		}
	}

	// Validate cache backend
	validBackends := []string{"file", "redis", "s3"}
	backendValid := false
	for _, backend := range validBackends {
		if config.Cache.Backend == backend {
			backendValid = true
			break
		}
	}
	if !backendValid {
		return fmt.Errorf("invalid cache backend: %s (valid options: %v)", config.Cache.Backend, validBackends)
	}

	// Validate logging level
	validLevels := []string{"debug", "info", "warn", "error"}
	levelValid := false
	for _, level := range validLevels {
		if config.Logging.Level == level {
			levelValid = true
			break
		}
	}
	if !levelValid {
		return fmt.Errorf("invalid logging level: %s (valid options: %v)", config.Logging.Level, validLevels)
	}

	return nil
}

// GetConfig returns the loaded configuration
func (loader *UnifiedConfigLoader) GetConfig() *RootConfig {
	return loader.config
}

// GetConfigDir returns the configuration directory
func (loader *UnifiedConfigLoader) GetConfigDir() string {
	return loader.configDir
}