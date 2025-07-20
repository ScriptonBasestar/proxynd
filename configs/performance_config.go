package configs

import (
	"fmt"
	"time"

	"proxynd/internal/performance"
)

// PerformanceConfig holds performance-related configuration for ProxyND
type PerformanceConfig struct {
	// Enable/disable performance optimization
	Enabled bool `yaml:"enabled" json:"enabled" default:"true"`

	// Cache optimization configuration
	CacheOptimizer *CacheOptimizerConfig `yaml:"cache_optimizer,omitempty" json:"cache_optimizer,omitempty"`

	// Connection pooling configuration
	ConnectionPool *ConnectionPoolConfig `yaml:"connection_pool,omitempty" json:"connection_pool,omitempty"`

	// Resource monitoring configuration
	ResourceMonitor *ResourceMonitorConfig `yaml:"resource_monitor,omitempty" json:"resource_monitor,omitempty"`

	// Request optimization configuration
	RequestOptimizer *RequestOptimizerConfig `yaml:"request_optimizer,omitempty" json:"request_optimizer,omitempty"`

	// Performance middleware configuration
	Middleware *PerformanceMiddlewareConfig `yaml:"middleware,omitempty" json:"middleware,omitempty"`
}

// CacheOptimizerConfig configures cache optimization strategies
type CacheOptimizerConfig struct {
	// Analysis settings
	AnalysisWindow time.Duration `yaml:"analysis_window" json:"analysis_window" default:"1h"`
	MinDataPoints  int           `yaml:"min_data_points" json:"min_data_points" default:"100"`

	// Optimization thresholds
	HitRateThreshold  float64 `yaml:"hit_rate_threshold" json:"hit_rate_threshold" default:"0.8"`
	MissRateThreshold float64 `yaml:"miss_rate_threshold" json:"miss_rate_threshold" default:"0.3"`
	EvictionThreshold float64 `yaml:"eviction_threshold" json:"eviction_threshold" default:"0.1"`

	// TTL optimization
	EnableTTLOptimization bool          `yaml:"enable_ttl_optimization" json:"enable_ttl_optimization" default:"true"`
	MinTTL                time.Duration `yaml:"min_ttl" json:"min_ttl" default:"5m"`
	MaxTTL                time.Duration `yaml:"max_ttl" json:"max_ttl" default:"24h"`
	TTLAdjustmentFactor   float64       `yaml:"ttl_adjustment_factor" json:"ttl_adjustment_factor" default:"0.1"`

	// Size optimization
	EnableSizeOptimization   bool          `yaml:"enable_size_optimization" json:"enable_size_optimization" default:"true"`
	MaxCacheSize             int64         `yaml:"max_cache_size" json:"max_cache_size" default:"1073741824"` // 1GB
	SizeOptimizationInterval time.Duration `yaml:"size_opt_interval" json:"size_opt_interval" default:"10m"`

	// Prewarming
	EnablePrewarming   bool     `yaml:"enable_prewarming" json:"enable_prewarming" default:"true"`
	PrewarmingSchedule string   `yaml:"prewarming_schedule" json:"prewarming_schedule" default:"0 6 * * *"` // 6 AM daily
	PrewarmingPatterns []string `yaml:"prewarming_patterns,omitempty" json:"prewarming_patterns,omitempty"`
}

// ConnectionPoolConfig configures HTTP connection pooling
type ConnectionPoolConfig struct {
	// Pool sizing
	MaxIdleConns        int `yaml:"max_idle_conns" json:"max_idle_conns" default:"100"`
	MaxIdleConnsPerHost int `yaml:"max_idle_conns_per_host" json:"max_idle_conns_per_host" default:"10"`
	MaxConnsPerHost     int `yaml:"max_conns_per_host" json:"max_conns_per_host" default:"50"`

	// Timeouts
	IdleConnTimeout       time.Duration `yaml:"idle_conn_timeout" json:"idle_conn_timeout" default:"90s"`
	ConnTimeout           time.Duration `yaml:"conn_timeout" json:"conn_timeout" default:"30s"`
	KeepAliveTimeout      time.Duration `yaml:"keep_alive_timeout" json:"keep_alive_timeout" default:"30s"`
	TLSHandshakeTimeout   time.Duration `yaml:"tls_handshake_timeout" json:"tls_handshake_timeout" default:"10s"`
	ResponseHeaderTimeout time.Duration `yaml:"response_header_timeout" json:"response_header_timeout" default:"30s"`

	// Health checking
	EnableHealthCheck   bool          `yaml:"enable_health_check" json:"enable_health_check" default:"true"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval" json:"health_check_interval" default:"30s"`
	HealthCheckTimeout  time.Duration `yaml:"health_check_timeout" json:"health_check_timeout" default:"5s"`
	FailureThreshold    int           `yaml:"failure_threshold" json:"failure_threshold" default:"3"`

	// Circuit breaker
	EnableCircuitBreaker    bool          `yaml:"enable_circuit_breaker" json:"enable_circuit_breaker" default:"true"`
	CircuitBreakerThreshold int           `yaml:"circuit_breaker_threshold" json:"circuit_breaker_threshold" default:"5"`
	CircuitBreakerWindow    time.Duration `yaml:"circuit_breaker_window" json:"circuit_breaker_window" default:"60s"`
	CircuitBreakerTimeout   time.Duration `yaml:"circuit_breaker_timeout" json:"circuit_breaker_timeout" default:"30s"`

	// Optimization
	EnableConnectionReuse bool          `yaml:"enable_connection_reuse" json:"enable_connection_reuse" default:"true"`
	EnableTCPKeepAlive    bool          `yaml:"enable_tcp_keep_alive" json:"enable_tcp_keep_alive" default:"true"`
	EnableCompression     bool          `yaml:"enable_compression" json:"enable_compression" default:"true"`
	OptimizationInterval  time.Duration `yaml:"optimization_interval" json:"optimization_interval" default:"5m"`
}

// ResourceMonitorConfig configures system resource monitoring
type ResourceMonitorConfig struct {
	// Monitoring intervals
	MemoryInterval  time.Duration `yaml:"memory_interval" json:"memory_interval" default:"30s"`
	CPUInterval     time.Duration `yaml:"cpu_interval" json:"cpu_interval" default:"30s"`
	DiskInterval    time.Duration `yaml:"disk_interval" json:"disk_interval" default:"60s"`
	NetworkInterval time.Duration `yaml:"network_interval" json:"network_interval" default:"30s"`

	// Thresholds for alerts
	MemoryThreshold    float64 `yaml:"memory_threshold" json:"memory_threshold" default:"0.8"` // 80%
	CPUThreshold       float64 `yaml:"cpu_threshold" json:"cpu_threshold" default:"0.8"`       // 80%
	DiskThreshold      float64 `yaml:"disk_threshold" json:"disk_threshold" default:"0.9"`     // 90%
	GoroutineThreshold int     `yaml:"goroutine_threshold" json:"goroutine_threshold" default:"1000"`

	// Optimization settings
	EnableAutoGC       bool    `yaml:"enable_auto_gc" json:"enable_auto_gc" default:"true"`
	GCThreshold        float64 `yaml:"gc_threshold" json:"gc_threshold" default:"0.7"` // 70% memory usage
	EnableOptimization bool    `yaml:"enable_optimization" json:"enable_optimization" default:"true"`
}

// RequestOptimizerConfig configures HTTP request optimization
type RequestOptimizerConfig struct {
	// Compression settings
	EnableCompression    bool     `yaml:"enable_compression" json:"enable_compression" default:"true"`
	CompressionLevel     int      `yaml:"compression_level" json:"compression_level" default:"6"`
	CompressionThreshold int      `yaml:"compression_threshold" json:"compression_threshold" default:"1024"`
	CompressibleTypes    []string `yaml:"compressible_types,omitempty" json:"compressible_types,omitempty"`

	// Caching optimization
	EnableCacheOptimization bool          `yaml:"enable_cache_opt" json:"enable_cache_opt" default:"true"`
	CacheControlMaxAge      time.Duration `yaml:"cache_control_max_age" json:"cache_control_max_age" default:"1h"`
	ETags                   bool          `yaml:"etags" json:"etags" default:"true"`

	// Rate limiting
	EnableAdaptiveRateLimit bool          `yaml:"enable_adaptive_rate" json:"enable_adaptive_rate" default:"true"`
	BaseRateLimit           int           `yaml:"base_rate_limit" json:"base_rate_limit" default:"1000"`
	BurstLimit              int           `yaml:"burst_limit" json:"burst_limit" default:"100"`
	RateLimitWindow         time.Duration `yaml:"rate_limit_window" json:"rate_limit_window" default:"1m"`

	// Request optimization
	MaxRequestSize   int64         `yaml:"max_request_size" json:"max_request_size" default:"10485760"` // 10MB
	RequestTimeout   time.Duration `yaml:"request_timeout" json:"request_timeout" default:"30s"`
	KeepAliveTimeout time.Duration `yaml:"keep_alive_timeout" json:"keep_alive_timeout" default:"30s"`

	// Response optimization
	EnableResponseStreaming bool  `yaml:"enable_response_streaming" json:"enable_response_streaming" default:"true"`
	StreamingThreshold      int64 `yaml:"streaming_threshold" json:"streaming_threshold" default:"1048576"` // 1MB
	MaxResponseSize         int64 `yaml:"max_response_size" json:"max_response_size" default:"104857600"`   // 100MB

	// Performance monitoring
	SlowRequestThreshold time.Duration `yaml:"slow_request_threshold" json:"slow_request_threshold" default:"2s"`
	EnableMetrics        bool          `yaml:"enable_metrics" json:"enable_metrics" default:"true"`
}

// PerformanceMiddlewareConfig configures performance monitoring middleware
type PerformanceMiddlewareConfig struct {
	// Monitoring settings
	EnableMetrics   bool `yaml:"enable_metrics" json:"enable_metrics" default:"true"`
	EnableTracing   bool `yaml:"enable_tracing" json:"enable_tracing" default:"true"`
	EnableProfiling bool `yaml:"enable_profiling" json:"enable_profiling" default:"false"`

	// Performance thresholds
	SlowRequestThreshold     time.Duration `yaml:"slow_request_threshold" json:"slow_request_threshold" default:"2s"`
	CriticalRequestThreshold time.Duration `yaml:"critical_req_threshold" json:"critical_req_threshold" default:"5s"`

	// Optimization settings
	EnableRequestOptimization bool `yaml:"enable_request_optimization" json:"enable_request_optimization" default:"true"`
	EnableCacheOptimization   bool `yaml:"enable_cache_optimization" json:"enable_cache_optimization" default:"true"`
	EnableResourceMonitoring  bool `yaml:"enable_resource_monitoring" json:"enable_resource_monitoring" default:"true"`

	// Alert settings
	EnableAlerts   bool `yaml:"enable_alerts" json:"enable_alerts" default:"true"`
	AlertThreshold int  `yaml:"alert_threshold" json:"alert_threshold" default:"10"`

	// Skip paths for performance monitoring
	SkipPaths []string `yaml:"skip_paths,omitempty" json:"skip_paths,omitempty"`
}

// Default performance configurations
var (
	// DefaultPerformanceConfig provides the default performanceconfig
	DefaultPerformanceConfig = PerformanceConfig{
		Enabled: true,
		CacheOptimizer: &CacheOptimizerConfig{
			AnalysisWindow:           time.Hour,
			MinDataPoints:            100,
			HitRateThreshold:         0.8,
			MissRateThreshold:        0.3,
			EvictionThreshold:        0.1,
			EnableTTLOptimization:    true,
			MinTTL:                   5 * time.Minute,
			MaxTTL:                   24 * time.Hour,
			TTLAdjustmentFactor:      0.1,
			EnableSizeOptimization:   true,
			MaxCacheSize:             1024 * 1024 * 1024, // 1GB
			SizeOptimizationInterval: 10 * time.Minute,
			EnablePrewarming:         true,
			PrewarmingSchedule:       "0 6 * * *", // 6 AM daily
			PrewarmingPatterns:       []string{"*.deb", "*.rpm", "maven-metadata.xml"},
		},
		ConnectionPool: &ConnectionPoolConfig{
			MaxIdleConns:            100,
			MaxIdleConnsPerHost:     10,
			MaxConnsPerHost:         50,
			IdleConnTimeout:         90 * time.Second,
			ConnTimeout:             30 * time.Second,
			KeepAliveTimeout:        30 * time.Second,
			TLSHandshakeTimeout:     10 * time.Second,
			ResponseHeaderTimeout:   30 * time.Second,
			EnableHealthCheck:       true,
			HealthCheckInterval:     30 * time.Second,
			HealthCheckTimeout:      5 * time.Second,
			FailureThreshold:        3,
			EnableCircuitBreaker:    true,
			CircuitBreakerThreshold: 5,
			CircuitBreakerWindow:    60 * time.Second,
			CircuitBreakerTimeout:   30 * time.Second,
			EnableConnectionReuse:   true,
			EnableTCPKeepAlive:      true,
			EnableCompression:       true,
			OptimizationInterval:    5 * time.Minute,
		},
		ResourceMonitor: &ResourceMonitorConfig{
			MemoryInterval:     30 * time.Second,
			CPUInterval:        30 * time.Second,
			DiskInterval:       60 * time.Second,
			NetworkInterval:    30 * time.Second,
			MemoryThreshold:    0.8, // 80%
			CPUThreshold:       0.8, // 80%
			DiskThreshold:      0.9, // 90%
			GoroutineThreshold: 1000,
			EnableAutoGC:       true,
			GCThreshold:        0.7, // 70%
			EnableOptimization: true,
		},
		RequestOptimizer: &RequestOptimizerConfig{
			EnableCompression:    true,
			CompressionLevel:     6,
			CompressionThreshold: 1024,
			CompressibleTypes: []string{
				"text/html",
				"text/css",
				"text/javascript",
				"application/javascript",
				"application/json",
				"text/plain",
				"application/xml",
				"text/xml",
			},
			EnableCacheOptimization: true,
			CacheControlMaxAge:      time.Hour,
			ETags:                   true,
			EnableAdaptiveRateLimit: true,
			BaseRateLimit:           1000,
			BurstLimit:              100,
			RateLimitWindow:         time.Minute,
			MaxRequestSize:          10 * 1024 * 1024, // 10MB
			RequestTimeout:          30 * time.Second,
			KeepAliveTimeout:        30 * time.Second,
			EnableResponseStreaming: true,
			StreamingThreshold:      1024 * 1024,       // 1MB
			MaxResponseSize:         100 * 1024 * 1024, // 100MB
			SlowRequestThreshold:    2 * time.Second,
			EnableMetrics:           true,
		},
		Middleware: &PerformanceMiddlewareConfig{
			EnableMetrics:             true,
			EnableTracing:             true,
			EnableProfiling:           false,
			SlowRequestThreshold:      2 * time.Second,
			CriticalRequestThreshold:  5 * time.Second,
			EnableRequestOptimization: true,
			EnableCacheOptimization:   true,
			EnableResourceMonitoring:  true,
			EnableAlerts:              true,
			AlertThreshold:            10,
			SkipPaths: []string{
				"/health",
				"/metrics",
				"/favicon.ico",
				"/_internal",
			},
		},
	}
)

// Validate validates the performance configuration
func (c *PerformanceConfig) Validate() error {
	if !c.Enabled {
		return nil // Skip validation if performance is disabled
	}

	if c.CacheOptimizer != nil {
		if err := c.validateCacheOptimizer(); err != nil {
			return fmt.Errorf("cache optimizer config invalid: %w", err)
		}
	}

	if c.ConnectionPool != nil {
		if err := c.validateConnectionPool(); err != nil {
			return fmt.Errorf("connection pool config invalid: %w", err)
		}
	}

	if c.ResourceMonitor != nil {
		if err := c.validateResourceMonitor(); err != nil {
			return fmt.Errorf("resource monitor config invalid: %w", err)
		}
	}

	if c.RequestOptimizer != nil {
		if err := c.validateRequestOptimizer(); err != nil {
			return fmt.Errorf("request optimizer config invalid: %w", err)
		}
	}

	if c.Middleware != nil {
		if err := c.validateMiddleware(); err != nil {
			return fmt.Errorf("performance middleware config invalid: %w", err)
		}
	}

	return nil
}

// validateCacheOptimizer validates cache optimizer configuration
func (c *PerformanceConfig) validateCacheOptimizer() error {
	co := c.CacheOptimizer

	if co.AnalysisWindow <= 0 {
		return fmt.Errorf("analysis_window must be positive")
	}

	if co.MinDataPoints < 0 {
		return fmt.Errorf("min_data_points must be non-negative")
	}

	if co.HitRateThreshold < 0 || co.HitRateThreshold > 1 {
		return fmt.Errorf("hit_rate_threshold must be between 0 and 1")
	}

	if co.MissRateThreshold < 0 || co.MissRateThreshold > 1 {
		return fmt.Errorf("miss_rate_threshold must be between 0 and 1")
	}

	if co.EvictionThreshold < 0 || co.EvictionThreshold > 1 {
		return fmt.Errorf("eviction_threshold must be between 0 and 1")
	}

	if co.EnableTTLOptimization {
		if co.MinTTL <= 0 {
			return fmt.Errorf("min_ttl must be positive when TTL optimization is enabled")
		}

		if co.MaxTTL <= co.MinTTL {
			return fmt.Errorf("max_ttl must be greater than min_ttl")
		}

		if co.TTLAdjustmentFactor <= 0 || co.TTLAdjustmentFactor >= 1 {
			return fmt.Errorf("ttl_adjustment_factor must be between 0 and 1")
		}
	}

	if co.EnableSizeOptimization {
		if co.MaxCacheSize <= 0 {
			return fmt.Errorf("max_cache_size must be positive when size optimization is enabled")
		}

		if co.SizeOptimizationInterval <= 0 {
			return fmt.Errorf("size_optimization_interval must be positive")
		}
	}

	return nil
}

// validateConnectionPool validates connection pool configuration
func (c *PerformanceConfig) validateConnectionPool() error {
	cp := c.ConnectionPool

	if cp.MaxIdleConns < 0 {
		return fmt.Errorf("max_idle_conns must be non-negative")
	}

	if cp.MaxIdleConnsPerHost < 0 {
		return fmt.Errorf("max_idle_conns_per_host must be non-negative")
	}

	if cp.MaxConnsPerHost < 0 {
		return fmt.Errorf("max_conns_per_host must be non-negative")
	}

	if cp.MaxIdleConnsPerHost > cp.MaxConnsPerHost {
		return fmt.Errorf("max_idle_conns_per_host cannot exceed max_conns_per_host")
	}

	if cp.IdleConnTimeout <= 0 {
		return fmt.Errorf("idle_conn_timeout must be positive")
	}

	if cp.ConnTimeout <= 0 {
		return fmt.Errorf("conn_timeout must be positive")
	}

	if cp.EnableHealthCheck {
		if cp.HealthCheckInterval <= 0 {
			return fmt.Errorf("health_check_interval must be positive when health check is enabled")
		}

		if cp.HealthCheckTimeout <= 0 {
			return fmt.Errorf("health_check_timeout must be positive when health check is enabled")
		}

		if cp.FailureThreshold <= 0 {
			return fmt.Errorf("failure_threshold must be positive")
		}
	}

	if cp.EnableCircuitBreaker {
		if cp.CircuitBreakerThreshold <= 0 {
			return fmt.Errorf("circuit_breaker_threshold must be positive when circuit breaker is enabled")
		}

		if cp.CircuitBreakerWindow <= 0 {
			return fmt.Errorf("circuit_breaker_window must be positive")
		}

		if cp.CircuitBreakerTimeout <= 0 {
			return fmt.Errorf("circuit_breaker_timeout must be positive")
		}
	}

	return nil
}

// validateResourceMonitor validates resource monitor configuration
func (c *PerformanceConfig) validateResourceMonitor() error {
	rm := c.ResourceMonitor

	if rm.MemoryInterval <= 0 {
		return fmt.Errorf("memory_interval must be positive")
	}

	if rm.CPUInterval <= 0 {
		return fmt.Errorf("cpu_interval must be positive")
	}

	if rm.DiskInterval <= 0 {
		return fmt.Errorf("disk_interval must be positive")
	}

	if rm.NetworkInterval <= 0 {
		return fmt.Errorf("network_interval must be positive")
	}

	if rm.MemoryThreshold < 0 || rm.MemoryThreshold > 1 {
		return fmt.Errorf("memory_threshold must be between 0 and 1")
	}

	if rm.CPUThreshold < 0 || rm.CPUThreshold > 1 {
		return fmt.Errorf("cpu_threshold must be between 0 and 1")
	}

	if rm.DiskThreshold < 0 || rm.DiskThreshold > 1 {
		return fmt.Errorf("disk_threshold must be between 0 and 1")
	}

	if rm.GoroutineThreshold <= 0 {
		return fmt.Errorf("goroutine_threshold must be positive")
	}

	if rm.EnableAutoGC && (rm.GCThreshold <= 0 || rm.GCThreshold > 1) {
		return fmt.Errorf("gc_threshold must be between 0 and 1 when auto GC is enabled")
	}

	return nil
}

// validateRequestOptimizer validates request optimizer configuration
func (c *PerformanceConfig) validateRequestOptimizer() error {
	ro := c.RequestOptimizer

	if ro.EnableCompression {
		if ro.CompressionLevel < 1 || ro.CompressionLevel > 9 {
			return fmt.Errorf("compression_level must be between 1 and 9")
		}

		if ro.CompressionThreshold < 0 {
			return fmt.Errorf("compression_threshold must be non-negative")
		}
	}

	if ro.EnableCacheOptimization {
		if ro.CacheControlMaxAge <= 0 {
			return fmt.Errorf("cache_control_max_age must be positive when cache optimization is enabled")
		}
	}

	if ro.EnableAdaptiveRateLimit {
		if ro.BaseRateLimit <= 0 {
			return fmt.Errorf("base_rate_limit must be positive when adaptive rate limiting is enabled")
		}

		if ro.BurstLimit <= 0 {
			return fmt.Errorf("burst_limit must be positive")
		}

		if ro.RateLimitWindow <= 0 {
			return fmt.Errorf("rate_limit_window must be positive")
		}
	}

	if ro.MaxRequestSize <= 0 {
		return fmt.Errorf("max_request_size must be positive")
	}

	if ro.RequestTimeout <= 0 {
		return fmt.Errorf("request_timeout must be positive")
	}

	if ro.EnableResponseStreaming {
		if ro.StreamingThreshold <= 0 {
			return fmt.Errorf("streaming_threshold must be positive when response streaming is enabled")
		}

		if ro.MaxResponseSize <= 0 {
			return fmt.Errorf("max_response_size must be positive")
		}

		if ro.StreamingThreshold > ro.MaxResponseSize {
			return fmt.Errorf("streaming_threshold cannot exceed max_response_size")
		}
	}

	if ro.SlowRequestThreshold <= 0 {
		return fmt.Errorf("slow_request_threshold must be positive")
	}

	return nil
}

// validateMiddleware validates performance middleware configuration
func (c *PerformanceConfig) validateMiddleware() error {
	pm := c.Middleware

	if pm.SlowRequestThreshold <= 0 {
		return fmt.Errorf("slow_request_threshold must be positive")
	}

	if pm.CriticalRequestThreshold <= 0 {
		return fmt.Errorf("critical_request_threshold must be positive")
	}

	if pm.CriticalRequestThreshold <= pm.SlowRequestThreshold {
		return fmt.Errorf("critical_request_threshold must be greater than slow_request_threshold")
	}

	if pm.EnableAlerts && pm.AlertThreshold <= 0 {
		return fmt.Errorf("alert_threshold must be positive when alerts are enabled")
	}

	return nil
}

// ToPerformanceConfig converts to internal performance config
func (c *PerformanceConfig) ToPerformanceConfig() *performance.Config {
	config := &performance.Config{}

	if c.CacheOptimizer != nil {
		config.CacheOptimizer = &performance.CacheOptimizerConfig{
			AnalysisWindow:           c.CacheOptimizer.AnalysisWindow,
			MinDataPoints:            c.CacheOptimizer.MinDataPoints,
			HitRateThreshold:         c.CacheOptimizer.HitRateThreshold,
			MissRateThreshold:        c.CacheOptimizer.MissRateThreshold,
			EvictionThreshold:        c.CacheOptimizer.EvictionThreshold,
			EnableTTLOptimization:    c.CacheOptimizer.EnableTTLOptimization,
			MinTTL:                   c.CacheOptimizer.MinTTL,
			MaxTTL:                   c.CacheOptimizer.MaxTTL,
			TTLAdjustmentFactor:      c.CacheOptimizer.TTLAdjustmentFactor,
			EnableSizeOptimization:   c.CacheOptimizer.EnableSizeOptimization,
			MaxCacheSize:             c.CacheOptimizer.MaxCacheSize,
			SizeOptimizationInterval: c.CacheOptimizer.SizeOptimizationInterval,
			EnablePrewarming:         c.CacheOptimizer.EnablePrewarming,
			PrewarmingSchedule:       c.CacheOptimizer.PrewarmingSchedule,
			PrewarmingPatterns:       c.CacheOptimizer.PrewarmingPatterns,
		}
	}

	if c.ConnectionPool != nil {
		config.ConnectionPool = &performance.PoolConfig{
			MaxIdleConns:            c.ConnectionPool.MaxIdleConns,
			MaxIdleConnsPerHost:     c.ConnectionPool.MaxIdleConnsPerHost,
			MaxConnsPerHost:         c.ConnectionPool.MaxConnsPerHost,
			IdleConnTimeout:         c.ConnectionPool.IdleConnTimeout,
			ConnTimeout:             c.ConnectionPool.ConnTimeout,
			KeepAliveTimeout:        c.ConnectionPool.KeepAliveTimeout,
			TLSHandshakeTimeout:     c.ConnectionPool.TLSHandshakeTimeout,
			ResponseHeaderTimeout:   c.ConnectionPool.ResponseHeaderTimeout,
			EnableHealthCheck:       c.ConnectionPool.EnableHealthCheck,
			HealthCheckInterval:     c.ConnectionPool.HealthCheckInterval,
			HealthCheckTimeout:      c.ConnectionPool.HealthCheckTimeout,
			FailureThreshold:        c.ConnectionPool.FailureThreshold,
			EnableCircuitBreaker:    c.ConnectionPool.EnableCircuitBreaker,
			CircuitBreakerThreshold: c.ConnectionPool.CircuitBreakerThreshold,
			CircuitBreakerWindow:    c.ConnectionPool.CircuitBreakerWindow,
			CircuitBreakerTimeout:   c.ConnectionPool.CircuitBreakerTimeout,
			EnableConnectionReuse:   c.ConnectionPool.EnableConnectionReuse,
			EnableTCPKeepAlive:      c.ConnectionPool.EnableTCPKeepAlive,
			EnableCompression:       c.ConnectionPool.EnableCompression,
			OptimizationInterval:    c.ConnectionPool.OptimizationInterval,
		}
	}

	if c.ResourceMonitor != nil {
		config.ResourceMonitor = &performance.ResourceConfig{
			MemoryInterval:     c.ResourceMonitor.MemoryInterval,
			CPUInterval:        c.ResourceMonitor.CPUInterval,
			DiskInterval:       c.ResourceMonitor.DiskInterval,
			NetworkInterval:    c.ResourceMonitor.NetworkInterval,
			MemoryThreshold:    c.ResourceMonitor.MemoryThreshold,
			CPUThreshold:       c.ResourceMonitor.CPUThreshold,
			DiskThreshold:      c.ResourceMonitor.DiskThreshold,
			GoroutineThreshold: c.ResourceMonitor.GoroutineThreshold,
			EnableAutoGC:       c.ResourceMonitor.EnableAutoGC,
			GCThreshold:        c.ResourceMonitor.GCThreshold,
			EnableOptimization: c.ResourceMonitor.EnableOptimization,
		}
	}

	if c.RequestOptimizer != nil {
		config.RequestOptimizer = &performance.RequestOptimizerConfig{
			EnableCompression:       c.RequestOptimizer.EnableCompression,
			CompressionLevel:        c.RequestOptimizer.CompressionLevel,
			CompressionThreshold:    c.RequestOptimizer.CompressionThreshold,
			CompressibleTypes:       c.RequestOptimizer.CompressibleTypes,
			EnableCacheOptimization: c.RequestOptimizer.EnableCacheOptimization,
			CacheControlMaxAge:      c.RequestOptimizer.CacheControlMaxAge,
			ETags:                   c.RequestOptimizer.ETags,
			EnableAdaptiveRateLimit: c.RequestOptimizer.EnableAdaptiveRateLimit,
			BaseRateLimit:           c.RequestOptimizer.BaseRateLimit,
			BurstLimit:              c.RequestOptimizer.BurstLimit,
			RateLimitWindow:         c.RequestOptimizer.RateLimitWindow,
			MaxRequestSize:          c.RequestOptimizer.MaxRequestSize,
			RequestTimeout:          c.RequestOptimizer.RequestTimeout,
			KeepAliveTimeout:        c.RequestOptimizer.KeepAliveTimeout,
			EnableResponseStreaming: c.RequestOptimizer.EnableResponseStreaming,
			StreamingThreshold:      c.RequestOptimizer.StreamingThreshold,
			MaxResponseSize:         c.RequestOptimizer.MaxResponseSize,
			SlowRequestThreshold:    c.RequestOptimizer.SlowRequestThreshold,
			EnableMetrics:           c.RequestOptimizer.EnableMetrics,
		}
	}

	if c.Middleware != nil {
		config.Middleware = &performance.PerformanceConfig{
			EnableMetrics:             c.Middleware.EnableMetrics,
			EnableTracing:             c.Middleware.EnableTracing,
			EnableProfiling:           c.Middleware.EnableProfiling,
			SlowRequestThreshold:      c.Middleware.SlowRequestThreshold,
			CriticalRequestThreshold:  c.Middleware.CriticalRequestThreshold,
			EnableRequestOptimization: c.Middleware.EnableRequestOptimization,
			EnableCacheOptimization:   c.Middleware.EnableCacheOptimization,
			EnableResourceMonitoring:  c.Middleware.EnableResourceMonitoring,
			EnableAlerts:              c.Middleware.EnableAlerts,
			AlertThreshold:            c.Middleware.AlertThreshold,
			SkipPaths:                 c.Middleware.SkipPaths,
		}
	}

	return config
}
