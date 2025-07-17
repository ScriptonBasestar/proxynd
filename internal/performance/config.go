package performance

import (
	"fmt"
	"time"
)

// Config holds comprehensive performance configuration
type Config struct {
	// Cache optimization
	CacheOptimizer *CacheOptimizerConfig `yaml:"cache_optimizer,omitempty" json:"cache_optimizer,omitempty"`

	// Connection pooling
	ConnectionPool *PoolConfig `yaml:"connection_pool,omitempty" json:"connection_pool,omitempty"`

	// Resource monitoring
	ResourceMonitor *ResourceConfig `yaml:"resource_monitor,omitempty" json:"resource_monitor,omitempty"`

	// Request optimization
	RequestOptimizer *RequestOptimizerConfig `yaml:"request_optimizer,omitempty" json:"request_optimizer,omitempty"`

	// Performance middleware
	Middleware *PerformanceConfig `yaml:"middleware,omitempty" json:"middleware,omitempty"`
}

// DefaultConfig returns default performance configuration
func DefaultConfig() *Config {
	return &Config{
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
		ConnectionPool: &PoolConfig{
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
		ResourceMonitor: &ResourceConfig{
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
		Middleware: &PerformanceConfig{
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
}

// Validate validates the performance configuration
func (c *Config) Validate() error {
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
func (c *Config) validateCacheOptimizer() error {
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
func (c *Config) validateConnectionPool() error {
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
func (c *Config) validateResourceMonitor() error {
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
func (c *Config) validateRequestOptimizer() error {
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
func (c *Config) validateMiddleware() error {
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

// GetCacheOptimizerConfig returns cache optimizer configuration with defaults
func (c *Config) GetCacheOptimizerConfig() *CacheOptimizerConfig {
	if c.CacheOptimizer != nil {
		return c.CacheOptimizer
	}
	return DefaultConfig().CacheOptimizer
}

// GetConnectionPoolConfig returns connection pool configuration with defaults
func (c *Config) GetConnectionPoolConfig() *PoolConfig {
	if c.ConnectionPool != nil {
		return c.ConnectionPool
	}
	return DefaultConfig().ConnectionPool
}

// GetResourceMonitorConfig returns resource monitor configuration with defaults
func (c *Config) GetResourceMonitorConfig() *ResourceConfig {
	if c.ResourceMonitor != nil {
		return c.ResourceMonitor
	}
	return DefaultConfig().ResourceMonitor
}

// GetRequestOptimizerConfig returns request optimizer configuration with defaults
func (c *Config) GetRequestOptimizerConfig() *RequestOptimizerConfig {
	if c.RequestOptimizer != nil {
		return c.RequestOptimizer
	}
	return DefaultConfig().RequestOptimizer
}

// GetMiddlewareConfig returns performance middleware configuration with defaults
func (c *Config) GetMiddlewareConfig() *PerformanceConfig {
	if c.Middleware != nil {
		return c.Middleware
	}
	return DefaultConfig().Middleware
}
