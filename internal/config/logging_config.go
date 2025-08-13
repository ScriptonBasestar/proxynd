package config

import (
	"fmt"
	"time"

	"proxynd/internal/logging"
)

// LoggingConfig holds comprehensive logging configuration
type LoggingConfig struct {
	// Core logging settings
	Level  string                 `yaml:"level" json:"level" default:"info"`
	Format string                 `yaml:"format" json:"format" default:"json"`
	Output []logging.OutputConfig `yaml:"output" json:"output"`
	Caller bool                   `yaml:"caller" json:"caller" default:"false"`
	Fields map[string]interface{} `yaml:"fields,omitempty" json:"fields,omitempty"`

	// Advanced features
	Correlation *CorrelationConfig      `yaml:"correlation,omitempty" json:"correlation,omitempty"`
	Sampling    *logging.SamplingConfig `yaml:"sampling,omitempty" json:"sampling,omitempty"`

	// Component-specific logging
	Components map[string]ComponentConfig `yaml:"components,omitempty" json:"components,omitempty"`

	// Middleware configuration
	Middleware *MiddlewareConfig `yaml:"middleware,omitempty" json:"middleware,omitempty"`

	// Audit logging
	Audit *AuditConfig `yaml:"audit,omitempty" json:"audit,omitempty"`

	// Security logging
	Security *SecurityLoggingConfig `yaml:"security,omitempty" json:"security,omitempty"`

	// Performance logging
	Performance *PerformanceLoggingConfig `yaml:"performance,omitempty" json:"performance,omitempty"`

	// Log aggregation and analysis
	Aggregation *AggregationConfig `yaml:"aggregation,omitempty" json:"aggregation,omitempty"`
}

// CorrelationConfig configures request correlation
type CorrelationConfig struct {
	Enabled    bool   `yaml:"enabled" json:"enabled" default:"true"`
	HeaderName string `yaml:"header_name" json:"header_name" default:"X-Correlation-ID"`
	Generate   bool   `yaml:"generate" json:"generate" default:"true"`
}

// ComponentConfig allows per-component logging configuration
type ComponentConfig struct {
	Level   string                 `yaml:"level,omitempty" json:"level,omitempty"`
	Output  []logging.OutputConfig `yaml:"output,omitempty" json:"output,omitempty"`
	Enabled bool                   `yaml:"enabled" json:"enabled" default:"true"`
	Fields  map[string]interface{} `yaml:"fields,omitempty" json:"fields,omitempty"`
}

// MiddlewareConfig configures HTTP request logging middleware
type MiddlewareConfig struct {
	Enabled         bool     `yaml:"enabled" json:"enabled" default:"true"`
	SkipPaths       []string `yaml:"skip_paths,omitempty" json:"skip_paths,omitempty"`
	SkipSuccessLogs bool     `yaml:"skip_success_logs" json:"skip_success_logs" default:"false"`
	LogRequestBody  bool     `yaml:"log_request_body" json:"log_request_body" default:"false"`
	LogResponseBody bool     `yaml:"log_response_body" json:"log_response_body" default:"false"`
	MaxBodySize     int      `yaml:"max_body_size" json:"max_body_size" default:"1024"`
	TimeFormat      string   `yaml:"time_format" json:"time_format" default:"RFC3339Nano"`
}

// AuditConfig configures audit logging
type AuditConfig struct {
	Enabled       bool     `yaml:"enabled" json:"enabled" default:"true"`
	File          string   `yaml:"file" json:"file" default:"/var/log/proxynd/audit.log"`
	Level         string   `yaml:"level" json:"level" default:"info"`
	Format        string   `yaml:"format" json:"format" default:"json"`
	Events        []string `yaml:"events,omitempty" json:"events,omitempty"`
	SkipEndpoints []string `yaml:"skip_endpoints,omitempty" json:"skip_endpoints,omitempty"`
	RetentionDays int      `yaml:"retention_days" json:"retention_days" default:"90"`
	MaxFileSize   string   `yaml:"max_file_size" json:"max_file_size" default:"100MB"`
	MaxFiles      int      `yaml:"max_files" json:"max_files" default:"10"`
}

// SecurityLoggingConfig configures security event logging
type SecurityLoggingConfig struct {
	Enabled         bool   `yaml:"enabled" json:"enabled" default:"true"`
	File            string `yaml:"file" json:"file" default:"/var/log/proxynd/security.log"`
	Level           string `yaml:"level" json:"level" default:"warn"`
	Format          string `yaml:"format" json:"format" default:"json"`
	ThreatDetection bool   `yaml:"threat_detection" json:"threat_detection" default:"true"`
	AlertThreshold  int    `yaml:"alert_threshold" json:"alert_threshold" default:"10"`
	BlockAfter      int    `yaml:"block_after" json:"block_after" default:"50"`
	RetentionDays   int    `yaml:"retention_days" json:"retention_days" default:"365"`
}

// PerformanceLoggingConfig configures performance logging
type PerformanceLoggingConfig struct {
	Enabled           bool          `yaml:"enabled" json:"enabled" default:"true"`
	File              string        `yaml:"file" json:"file" default:"/var/log/proxynd/performance.log"`
	Level             string        `yaml:"level" json:"level" default:"info"`
	SlowThreshold     time.Duration `yaml:"slow_threshold" json:"slow_threshold" default:"2s"`
	VerySlowThreshold time.Duration `yaml:"very_slow_threshold" json:"very_slow_threshold" default:"5s"`
	MemoryLogging     bool          `yaml:"memory_logging" json:"memory_logging" default:"true"`
	MemoryInterval    time.Duration `yaml:"memory_interval" json:"memory_interval" default:"5m"`
	MetricsInterval   time.Duration `yaml:"metrics_interval" json:"metrics_interval" default:"1m"`
}

// AggregationConfig configures log aggregation and analysis
type AggregationConfig struct {
	Enabled         bool              `yaml:"enabled" json:"enabled" default:"false"`
	LogPaths        []string          `yaml:"log_paths,omitempty" json:"log_paths,omitempty"`
	OutputPath      string            `yaml:"output_path" json:"output_path" default:"/var/log/proxynd/analytics"`
	AnalysisWindow  time.Duration     `yaml:"analysis_window" json:"analysis_window" default:"1h"`
	Patterns        map[string]string `yaml:"patterns,omitempty" json:"patterns,omitempty"`
	MaxFileSize     int64             `yaml:"max_file_size" json:"max_file_size" default:"104857600"`  // 100MB
	RetentionPeriod time.Duration     `yaml:"retention_period" json:"retention_period" default:"720h"` // 30 days
}

// Default configurations
var (
	// DefaultLoggingConfig provides the default logging configuration
	DefaultLoggingConfig = LoggingConfig{
		Level:  "info",
		Format: "json",
		Output: []logging.OutputConfig{
			{
				Type: "stdout",
			},
			{
				Type:       "file",
				Path:       "/var/log/proxynd/app.log",
				MaxSize:    100, // 100MB
				MaxAge:     7,   // 7 days
				MaxBackups: 10,
				Compress:   true,
			},
		},
		Caller: false,
		Correlation: &CorrelationConfig{
			Enabled:    true,
			HeaderName: "X-Correlation-ID",
			Generate:   true,
		},
		Middleware: &MiddlewareConfig{
			Enabled: true,
			SkipPaths: []string{
				"/health",
				"/metrics",
				"/favicon.ico",
			},
			SkipSuccessLogs: false,
			LogRequestBody:  false,
			LogResponseBody: false,
			MaxBodySize:     1024,
			TimeFormat:      "RFC3339Nano",
		},
		Audit: &AuditConfig{
			Enabled: true,
			File:    "/var/log/proxynd/audit.log",
			Level:   "info",
			Format:  "json",
			Events: []string{
				"authentication",
				"authorization",
				"file_access",
				"configuration_change",
				"admin_action",
			},
			RetentionDays: 90,
			MaxFileSize:   "100MB",
			MaxFiles:      10,
		},
		Security: &SecurityLoggingConfig{
			Enabled:         true,
			File:            "/var/log/proxynd/security.log",
			Level:           "warn",
			Format:          "json",
			ThreatDetection: true,
			AlertThreshold:  10,
			BlockAfter:      50,
			RetentionDays:   365,
		},
		Performance: &PerformanceLoggingConfig{
			Enabled:           true,
			File:              "/var/log/proxynd/performance.log",
			Level:             "info",
			SlowThreshold:     2 * time.Second,
			VerySlowThreshold: 5 * time.Second,
			MemoryLogging:     true,
			MemoryInterval:    5 * time.Minute,
			MetricsInterval:   1 * time.Minute,
		},
		Components: map[string]ComponentConfig{
			"http": {
				Level:   "info",
				Enabled: true,
			},
			"cache": {
				Level:   "info",
				Enabled: true,
			},
			"auth": {
				Level:   "info",
				Enabled: true,
			},
			"proxy": {
				Level:   "info",
				Enabled: true,
			},
			"security": {
				Level:   "warn",
				Enabled: true,
			},
		},
	}

	// DefaultAggregationConfig provides the default log aggregation configuration
	DefaultAggregationConfig = AggregationConfig{
		Enabled: false,
		LogPaths: []string{
			"/var/log/proxynd/app.log",
			"/var/log/proxynd/audit.log",
			"/var/log/proxynd/security.log",
			"/var/log/proxynd/performance.log",
		},
		OutputPath:     "/var/log/proxynd/analytics",
		AnalysisWindow: 1 * time.Hour,
		Patterns: map[string]string{
			"http_request": `^(?P<timestamp>\S+)\s+(?P<level>\w+)\s+.*HTTP request.*` +
				`method=(?P<method>\w+).*path=(?P<path>\S+).*status=(?P<status>\d+).*duration=(?P<duration>[\d.]+\w+)`,
			"error":    `^(?P<timestamp>\S+)\s+(?P<level>ERROR|FATAL)\s+(?P<component>\w+)\s+(?P<message>.*)`,
			"security": `^(?P<timestamp>\S+)\s+(?P<level>\w+)\s+security\s+(?P<event_type>\w+)\s+(?P<message>.*)`,
		},
		MaxFileSize:     100 * 1024 * 1024,   // 100MB
		RetentionPeriod: 30 * 24 * time.Hour, // 30 days
	}
)

// Validate validates the logging configuration
func (c *LoggingConfig) Validate() error {
	// Validate log level
	validLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true, "fatal": true, "panic": true,
	}
	if !validLevels[c.Level] {
		return fmt.Errorf("invalid log level: %s", c.Level)
	}

	// Validate format
	validFormats := map[string]bool{
		"json": true, "text": true,
	}
	if !validFormats[c.Format] {
		return fmt.Errorf("invalid log format: %s", c.Format)
	}

	// Validate outputs
	if len(c.Output) == 0 {
		return fmt.Errorf("at least one output must be specified")
	}

	for i, output := range c.Output {
		if err := validateOutputConfig(output); err != nil {
			return fmt.Errorf("invalid output config at index %d: %w", i, err)
		}
	}

	// Validate component configs
	for component, config := range c.Components {
		if config.Level != "" && !validLevels[config.Level] {
			return fmt.Errorf("invalid log level for component %s: %s", component, config.Level)
		}
	}

	return nil
}

// validateOutputConfig validates an output configuration
func validateOutputConfig(config logging.OutputConfig) error {
	validTypes := map[string]bool{
		"stdout": true, "stderr": true, "file": true, "syslog": true,
	}
	if !validTypes[config.Type] {
		return fmt.Errorf("invalid output type: %s", config.Type)
	}

	if config.Type == backendFile && config.Path == "" {
		return fmt.Errorf("file path is required for file output")
	}

	if config.MaxSize < 0 {
		return fmt.Errorf("max_size must be non-negative")
	}

	if config.MaxAge < 0 {
		return fmt.Errorf("max_age must be non-negative")
	}

	if config.MaxBackups < 0 {
		return fmt.Errorf("max_backups must be non-negative")
	}

	return nil
}

// ToLoggingConfig converts to internal logging config
func (c *LoggingConfig) ToLoggingConfig() *logging.Config {
	return &logging.Config{
		Level:       c.Level,
		Format:      c.Format,
		Output:      c.Output,
		Sampling:    c.Sampling,
		Correlation: c.Correlation != nil && c.Correlation.Enabled,
		Caller:      c.Caller,
		Fields:      c.Fields,
	}
}

// ToAggregatorConfig converts to aggregator config
func (c *AggregationConfig) ToAggregatorConfig() *logging.AggregatorConfig {
	return &logging.AggregatorConfig{
		LogPaths:        c.LogPaths,
		OutputPath:      c.OutputPath,
		AnalysisWindow:  c.AnalysisWindow,
		Patterns:        c.Patterns,
		MaxFileSize:     c.MaxFileSize,
		RetentionPeriod: c.RetentionPeriod,
	}
}

// GetComponentLevel returns the log level for a specific component
func (c *LoggingConfig) GetComponentLevel(component string) string {
	if config, exists := c.Components[component]; exists && config.Level != "" {
		return config.Level
	}
	return c.Level
}

// IsComponentEnabled checks if logging is enabled for a component
func (c *LoggingConfig) IsComponentEnabled(component string) bool {
	if config, exists := c.Components[component]; exists {
		return config.Enabled
	}
	return true // Default to enabled
}

// GetComponentFields returns additional fields for a component
func (c *LoggingConfig) GetComponentFields(component string) map[string]interface{} {
	fields := make(map[string]interface{})

	// Add global fields
	for k, v := range c.Fields {
		fields[k] = v
	}

	// Add component-specific fields
	if config, exists := c.Components[component]; exists {
		for k, v := range config.Fields {
			fields[k] = v
		}
	}

	return fields
}
