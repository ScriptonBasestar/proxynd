package performance

import (
	"context"
	"sync"
	"time"

	"proxynd/logging"
)

// Health status constants
const (
	// HealthStatusHealthy is a const that health status healthy
	// HealthStatusDegraded is a const that health status degraded
	HealthStatusHealthy  = "healthy"
	HealthStatusDegraded = "degraded"
)

// Manager coordinates all performance optimization components
type Manager struct {
	logger           logging.Logger
	config           *ManagerConfig
	cacheOptimizer   *CacheOptimizer
	connectionPool   *ConnectionPool
	resourceMonitor  *ResourceMonitor
	requestOptimizer *RequestOptimizer
	middleware       *PerformanceMiddleware
	mu               sync.RWMutex
	running          bool
	stopCh           chan struct{}
}

// ManagerConfig contains all performance configuration for manager
type ManagerConfig struct {
	CacheOptimizer   *CacheOptimizerConfig   `yaml:"cache_optimizer,omitempty" json:"cache_optimizer,omitempty"`
	ConnectionPool   *PoolConfig             `yaml:"connection_pool,omitempty" json:"connection_pool,omitempty"`
	ResourceMonitor  *ResourceConfig         `yaml:"resource_monitor,omitempty" json:"resource_monitor,omitempty"`
	RequestOptimizer *RequestOptimizerConfig `yaml:"request_optimizer,omitempty" json:"request_optimizer,omitempty"`
	Middleware       *PerformanceConfig      `yaml:"middleware,omitempty" json:"middleware,omitempty"`
}

// GlobalMetrics aggregates metrics from all performance components
type GlobalMetrics struct {
	// Request metrics
	TotalRequests       int64         `json:"total_requests"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	SlowRequests        int64         `json:"slow_requests"`
	CriticalRequests    int64         `json:"critical_requests"`

	// Cache metrics
	CacheHitRate       float64 `json:"cache_hit_rate"`
	CacheMissRate      float64 `json:"cache_miss_rate"`
	CacheOptimizations int64   `json:"cache_optimizations"`

	// Connection metrics
	ActiveConnections  int64 `json:"active_connections"`
	ConnectionsCreated int64 `json:"connections_created"`
	ConnectionsReused  int64 `json:"connections_reused"`
	ConnectionFailures int64 `json:"connection_failures"`

	// Resource metrics
	CPUUsage       float64 `json:"cpu_usage"`
	MemoryUsage    float64 `json:"memory_usage"`
	GoroutineCount int     `json:"goroutine_count"`
	GCTriggered    int64   `json:"gc_triggered"`

	// Optimization metrics
	OptimizationsApplied int64     `json:"optimizations_applied"`
	LastOptimization     time.Time `json:"last_optimization"`
	AlertsTriggered      int64     `json:"alerts_triggered"`

	// Performance improvements
	BytesSaved         int64         `json:"bytes_saved"`
	LatencyReduction   time.Duration `json:"latency_reduction"`
	ThroughputIncrease float64       `json:"throughput_increase"`

	LastUpdate time.Time `json:"last_update"`
}

// NewManager creates a new performance manager
func NewManager(logger logging.Logger, config *ManagerConfig) (*Manager, error) {
	manager := &Manager{
		logger: logger.WithField("component", "performance.manager"),
		config: config,
		stopCh: make(chan struct{}),
	}

	// Initialize components
	if err := manager.initializeComponents(); err != nil {
		return nil, err
	}

	return manager, nil
}

// initializeComponents initializes all performance components
func (m *Manager) initializeComponents() error {
	// Initialize cache optimizer
	if m.config.CacheOptimizer != nil {
		m.cacheOptimizer = NewCacheOptimizer(m.logger, m.config.CacheOptimizer)
		m.logger.Info("Cache optimizer initialized")
	}

	// Initialize connection pool
	if m.config.ConnectionPool != nil {
		m.connectionPool = NewConnectionPool(m.logger, m.config.ConnectionPool)
		m.logger.Info("Connection pool initialized")
	}

	// Initialize resource monitor
	if m.config.ResourceMonitor != nil {
		m.resourceMonitor = NewResourceMonitor(m.logger, m.config.ResourceMonitor)
		m.logger.Info("Resource monitor initialized")
	}

	// Initialize request optimizer
	if m.config.RequestOptimizer != nil {
		m.requestOptimizer = NewRequestOptimizer(m.logger, m.config.RequestOptimizer)
		m.logger.Info("Request optimizer initialized")
	}

	// Initialize performance middleware
	if m.config.Middleware != nil {
		m.middleware = NewPerformanceMiddleware(
			m.logger,
			m.cacheOptimizer,
			m.connectionPool,
			m.resourceMonitor,
			m.requestOptimizer,
			m.config.Middleware,
		)
		m.logger.Info("Performance middleware initialized")
	}

	return nil
}

// Start starts all performance components
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	m.logger.Info("Starting performance manager")

	// Start resource monitor
	if m.resourceMonitor != nil {
		if err := m.resourceMonitor.Start(ctx); err != nil {
			return err
		}
		m.logger.Info("Resource monitor started")
	}

	// Start cache optimizer periodic optimization
	if m.cacheOptimizer != nil {
		go m.cacheOptimizer.StartPeriodicOptimization(ctx)
		m.logger.Info("Cache optimizer periodic optimization started")
	}

	// Start global monitoring and optimization
	go m.startGlobalOptimization(ctx)

	m.running = true
	m.logger.Info("Performance manager started successfully")

	return nil
}

// Stop stops all performance components
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	m.logger.Info("Stopping performance manager")

	// Stop global optimization
	close(m.stopCh)

	// Stop resource monitor
	if m.resourceMonitor != nil {
		m.resourceMonitor.Stop()
		m.logger.Info("Resource monitor stopped")
	}

	// Close connection pool
	if m.connectionPool != nil {
		if err := m.connectionPool.Close(); err != nil {
			m.logger.Error("Failed to close connection pool", logging.ErrorField(err))
		} else {
			m.logger.Info("Connection pool closed")
		}
	}

	m.running = false
	m.logger.Info("Performance manager stopped")

	return nil
}

// startGlobalOptimization starts global performance optimization
func (m *Manager) startGlobalOptimization(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute) // Global optimization every 10 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.performGlobalOptimization(ctx)
		}
	}
}

// performGlobalOptimization performs comprehensive performance optimization
func (m *Manager) performGlobalOptimization(ctx context.Context) {
	m.logger.Info("Starting global performance optimization")

	optimizationsApplied := 0

	// Analyze and optimize cache performance
	if m.cacheOptimizer != nil {
		reports, err := m.cacheOptimizer.AnalyzePerformance(ctx)
		if err != nil {
			m.logger.Error("Cache performance analysis failed", logging.ErrorField(err))
		} else if len(reports) > 0 {
			if err := m.cacheOptimizer.ApplyOptimizations(ctx, reports); err != nil {
				m.logger.Error("Cache optimization application failed", logging.ErrorField(err))
			} else {
				optimizationsApplied += len(reports)
				m.logger.Info("Cache optimizations applied", logging.F("count", len(reports)))
			}
		}
	}

	// Analyze system resource usage and optimize if needed
	if m.resourceMonitor != nil {
		stats := m.resourceMonitor.GetStats()

		// Trigger optimizations based on resource usage
		if stats.MemoryPercent > 0.8 {
			m.logger.Warn("High memory usage detected, triggering memory optimization")
			optimizationsApplied++
		}

		if stats.GoroutineCount > 1000 {
			m.logger.Warn("High goroutine count detected, consider reviewing goroutine management")
		}
	}

	// Analyze connection pool performance
	if m.connectionPool != nil {
		poolStats := m.connectionPool.GetStats()

		// Log connection pool health
		utilizationRate := float64(poolStats.ActiveConnections) / float64(poolStats.TotalConnections)
		if utilizationRate > 0.9 {
			m.logger.Warn("High connection pool utilization",
				logging.Float64("utilization", utilizationRate),
				logging.F("active", poolStats.ActiveConnections),
				logging.F("total", poolStats.TotalConnections))
		}
	}

	// Generate performance report
	m.generatePerformanceReport()

	m.logger.Info("Global performance optimization completed",
		logging.F("optimizations_applied", optimizationsApplied))
}

// generatePerformanceReport generates a comprehensive performance report
func (m *Manager) generatePerformanceReport() {
	metrics := m.GetGlobalMetrics()

	m.logger.Info("Performance Report",
		logging.F("total_requests", metrics.TotalRequests),
		logging.Duration("avg_response_time", metrics.AverageResponseTime),
		logging.Float64("cache_hit_rate", metrics.CacheHitRate),
		logging.Float64("cpu_usage", metrics.CPUUsage),
		logging.Float64("memory_usage", metrics.MemoryUsage),
		logging.F("active_connections", metrics.ActiveConnections),
		logging.F("optimizations_applied", metrics.OptimizationsApplied))

	// Performance recommendations
	recommendations := m.generateRecommendations(metrics)
	if len(recommendations) > 0 {
		m.logger.Info("Performance Recommendations",
			logging.Strings("recommendations", recommendations))
	}
}

// generateRecommendations generates performance recommendations based on metrics
func (m *Manager) generateRecommendations(metrics *GlobalMetrics) []string {
	var recommendations []string

	// Cache recommendations
	if metrics.CacheHitRate < 0.7 {
		recommendations = append(recommendations, "Consider optimizing cache TTL settings or prewarming strategies")
	}

	// Resource recommendations
	if metrics.MemoryUsage > 0.8 {
		recommendations = append(recommendations,
			"High memory usage detected - consider increasing available memory or optimizing memory-intensive operations")
	}

	if metrics.GoroutineCount > 1000 {
		recommendations = append(recommendations,
			"High goroutine count - review goroutine lifecycle management and consider pooling")
	}

	// Performance recommendations
	if metrics.AverageResponseTime > 2*time.Second {
		recommendations = append(recommendations,
			"High average response time - analyze slow endpoints and consider optimization")
	}

	// Connection recommendations
	utilizationRate := float64(metrics.ActiveConnections) / float64(metrics.ConnectionsCreated)
	if utilizationRate > 0.9 {
		recommendations = append(recommendations,
			"High connection pool utilization - consider increasing pool size or optimizing connection reuse")
	}

	return recommendations
}

// GetGlobalMetrics returns aggregated metrics from all components
func (m *Manager) GetGlobalMetrics() *GlobalMetrics {
	metrics := &GlobalMetrics{
		LastUpdate: time.Now(),
	}

	// Aggregate request optimizer metrics
	if m.requestOptimizer != nil {
		reqStats := m.requestOptimizer.GetStats()
		metrics.TotalRequests = reqStats.TotalRequests
		metrics.AverageResponseTime = reqStats.AverageResponseTime
		metrics.SlowRequests = reqStats.SlowRequests
		metrics.BytesSaved = reqStats.BytesSaved
	}

	// Aggregate cache optimizer metrics
	if m.cacheOptimizer != nil {
		cacheMetrics := m.cacheOptimizer.GetCurrentMetrics()
		metrics.CacheHitRate = cacheMetrics.HitRate
		metrics.CacheMissRate = cacheMetrics.MissRate
	}

	// Aggregate connection pool metrics
	if m.connectionPool != nil {
		poolStats := m.connectionPool.GetStats()
		metrics.ActiveConnections = poolStats.ActiveConnections
		metrics.ConnectionsCreated = poolStats.ConnectionsCreated
		metrics.ConnectionsReused = poolStats.ConnectionsReused
		metrics.ConnectionFailures = poolStats.ConnectionFailures
	}

	// Aggregate resource monitor metrics
	if m.resourceMonitor != nil {
		resourceStats := m.resourceMonitor.GetStats()
		metrics.CPUUsage = resourceStats.CPUPercent
		metrics.MemoryUsage = resourceStats.MemoryPercent
		metrics.GoroutineCount = resourceStats.GoroutineCount
		metrics.GCTriggered = resourceStats.GCTriggered
		metrics.OptimizationsApplied = resourceStats.OptimizationRuns
		metrics.LastOptimization = resourceStats.LastOptimization
	}

	// Aggregate middleware metrics
	if m.middleware != nil {
		middlewareMetrics := m.middleware.GetMetrics()
		// Update with any additional middleware-specific metrics
		if middlewareMetrics.TotalRequests > metrics.TotalRequests {
			metrics.TotalRequests = middlewareMetrics.TotalRequests
		}
	}

	return metrics
}

// GetHealthStatus returns comprehensive health status
func (m *Manager) GetHealthStatus() map[string]interface{} {
	status := map[string]interface{}{
		"overall_status": HealthStatusHealthy,
		"components":     map[string]interface{}{},
		"metrics":        m.GetGlobalMetrics(),
		"last_check":     time.Now(),
	}

	overallHealthy := true

	// Check cache optimizer health
	if m.cacheOptimizer != nil {
		cacheMetrics := m.cacheOptimizer.GetCurrentMetrics()
		cacheHealthy := cacheMetrics.HitRate > 0.5
		if components, ok := status["components"].(map[string]interface{}); ok {
			components["cache_optimizer"] = map[string]interface{}{
				"status":   getHealthStatusString(cacheHealthy),
				"hit_rate": cacheMetrics.HitRate,
			}
		}
		if !cacheHealthy {
			overallHealthy = false
		}
	}

	// Check connection pool health
	if m.connectionPool != nil {
		poolStats := m.connectionPool.GetStats()
		poolHealthy := poolStats.ConnectionFailures < 10
		if components, ok := status["components"].(map[string]interface{}); ok {
			components["connection_pool"] = map[string]interface{}{
				"status":             getHealthStatusString(poolHealthy),
				"active_connections": poolStats.ActiveConnections,
				"failures":           poolStats.ConnectionFailures,
			}
		}
		if !poolHealthy {
			overallHealthy = false
		}
	}

	// Check resource monitor health
	if m.resourceMonitor != nil {
		resourceHealthStatus := m.resourceMonitor.GetHealthStatus()
		resourceHealthy := resourceHealthStatus["overall_status"] == HealthStatusHealthy
		if components, ok := status["components"].(map[string]interface{}); ok {
			components["resource_monitor"] = resourceHealthStatus
		}
		if !resourceHealthy {
			overallHealthy = false
		}
	}

	// Check request optimizer health
	if m.requestOptimizer != nil {
		reqStats := m.requestOptimizer.GetStats()
		reqHealthy := reqStats.OptimizationErrors < 10
		if components, ok := status["components"].(map[string]interface{}); ok {
			components["request_optimizer"] = map[string]interface{}{
				"status":              getHealthStatusString(reqHealthy),
				"total_requests":      reqStats.TotalRequests,
				"optimization_errors": reqStats.OptimizationErrors,
			}
		}
		if !reqHealthy {
			overallHealthy = false
		}
	}

	// Check middleware health
	if m.middleware != nil {
		middlewareHealthStatus := m.middleware.GetHealthStatus()
		middlewareHealthy := middlewareHealthStatus["overall_status"] == HealthStatusHealthy
		if components, ok := status["components"].(map[string]interface{}); ok {
			components["middleware"] = middlewareHealthStatus
		}
		if !middlewareHealthy {
			overallHealthy = false
		}
	}

	if !overallHealthy {
		status["overall_status"] = HealthStatusDegraded
	}

	return status
}

// GetMiddleware returns the performance middleware for Fiber integration
func (m *Manager) GetMiddleware() *PerformanceMiddleware {
	return m.middleware
}

// GetConnectionPool returns the connection pool for HTTP client usage
func (m *Manager) GetConnectionPool() *ConnectionPool {
	return m.connectionPool
}

// GetCacheOptimizer returns the cache optimizer for cache operations
func (m *Manager) GetCacheOptimizer() *CacheOptimizer {
	return m.cacheOptimizer
}

// GetResourceMonitor returns the resource monitor for system monitoring
func (m *Manager) GetResourceMonitor() *ResourceMonitor {
	return m.resourceMonitor
}

// GetRequestOptimizer returns the request optimizer for request processing
func (m *Manager) GetRequestOptimizer() *RequestOptimizer {
	return m.requestOptimizer
}

// IsRunning returns whether the manager is currently running
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// Helper functions

func getHealthStatusString(healthy bool) string {
	if healthy {
		return HealthStatusHealthy
	}
	return "unhealthy"
}

// DefaultManagerConfig returns a default manager configuration
func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
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
				"text/html", "text/css", "text/javascript", "application/javascript",
				"application/json", "text/plain", "application/xml", "text/xml",
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
				"/health", "/metrics", "/favicon.ico", "/_internal",
			},
		},
	}
}
