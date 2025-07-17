package performance

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"proxynd/logging"
)

// PerformanceMiddleware provides comprehensive performance monitoring middleware
type PerformanceMiddleware struct {
	logger           logging.Logger
	cacheOptimizer   *CacheOptimizer
	connectionPool   *ConnectionPool
	resourceMonitor  *ResourceMonitor
	requestOptimizer *RequestOptimizer
	config           *PerformanceConfig
}

// PerformanceConfig configures performance middleware
type PerformanceConfig struct {
	// Monitoring settings
	EnableMetrics   bool `yaml:"enable_metrics" json:"enable_metrics" default:"true"`
	EnableTracing   bool `yaml:"enable_tracing" json:"enable_tracing" default:"true"`
	EnableProfiling bool `yaml:"enable_profiling" json:"enable_profiling" default:"false"`

	// Performance thresholds
	SlowRequestThreshold     time.Duration `yaml:"slow_request_threshold" json:"slow_request_threshold" default:"2s"`
	CriticalRequestThreshold time.Duration `yaml:"critical_request_threshold" json:"critical_request_threshold" default:"5s"`

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

// PerformanceMetrics tracks comprehensive performance metrics
type PerformanceMetrics struct {
	// Request metrics
	TotalRequests       int64         `json:"total_requests"`
	SlowRequests        int64         `json:"slow_requests"`
	CriticalRequests    int64         `json:"critical_requests"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	P95ResponseTime     time.Duration `json:"p95_response_time"`
	P99ResponseTime     time.Duration `json:"p99_response_time"`

	// Resource metrics
	CPUUsage       float64 `json:"cpu_usage"`
	MemoryUsage    float64 `json:"memory_usage"`
	GoroutineCount int     `json:"goroutine_count"`

	// Cache metrics
	CacheHitRate   float64 `json:"cache_hit_rate"`
	CacheMissRate  float64 `json:"cache_miss_rate"`
	CacheEvictions int64   `json:"cache_evictions"`

	// Connection metrics
	ActiveConnections int64   `json:"active_connections"`
	ConnectionErrors  int64   `json:"connection_errors"`
	PoolUtilization   float64 `json:"pool_utilization"`

	// Error metrics
	ErrorRate        float64 `json:"error_rate"`
	TimeoutErrors    int64   `json:"timeout_errors"`
	OptimizationRuns int64   `json:"optimization_runs"`

	LastUpdate time.Time `json:"last_update"`
}

// NewPerformanceMiddleware creates a new performance middleware
func NewPerformanceMiddleware(
	logger logging.Logger,
	cacheOptimizer *CacheOptimizer,
	connectionPool *ConnectionPool,
	resourceMonitor *ResourceMonitor,
	requestOptimizer *RequestOptimizer,
	config *PerformanceConfig,
) *PerformanceMiddleware {
	return &PerformanceMiddleware{
		logger:           logger.WithComponent("performance.middleware"),
		cacheOptimizer:   cacheOptimizer,
		connectionPool:   connectionPool,
		resourceMonitor:  resourceMonitor,
		requestOptimizer: requestOptimizer,
		config:           config,
	}
}

// Handler returns the Fiber middleware handler
func (pm *PerformanceMiddleware) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Skip monitoring for certain paths
		if pm.shouldSkipPath(c.Path()) {
			return c.Next()
		}

		// Add performance context
		ctx := pm.addPerformanceContext(c.UserContext())
		c.SetUserContext(ctx)

		// Apply request optimizations
		if pm.config.EnableRequestOptimization && pm.requestOptimizer != nil {
			if err := pm.requestOptimizer.OptimizeRequest(c); err != nil {
				return err
			}
		}

		// Monitor resource usage before request
		var beforeMetrics *ResourceStats
		if pm.config.EnableResourceMonitoring && pm.resourceMonitor != nil {
			beforeMetrics = pm.resourceMonitor.GetStats()
		}

		// Process request
		err := c.Next()

		// Calculate request duration
		duration := time.Since(start)

		// Record performance metrics
		pm.recordMetrics(c, duration, err, beforeMetrics)

		// Trigger optimizations if needed
		pm.triggerOptimizations(c, duration)

		// Log performance information
		pm.logPerformance(c, duration, err)

		return err
	}
}

// shouldSkipPath checks if path should be skipped from monitoring
func (pm *PerformanceMiddleware) shouldSkipPath(path string) bool {
	for _, skipPath := range pm.config.SkipPaths {
		if path == skipPath {
			return true
		}
	}
	return false
}

// addPerformanceContext adds performance-related context
func (pm *PerformanceMiddleware) addPerformanceContext(ctx context.Context) context.Context {
	// Add performance tracking context
	ctx = context.WithValue(ctx, "performance_start", time.Now())
	ctx = context.WithValue(ctx, "performance_enabled", true)

	if pm.config.EnableTracing {
		// Add tracing context (in practice, use proper tracing library)
		traceID := generateTraceID()
		ctx = context.WithValue(ctx, "trace_id", traceID)
	}

	return ctx
}

// recordMetrics records comprehensive performance metrics
func (pm *PerformanceMiddleware) recordMetrics(c *fiber.Ctx, duration time.Duration, err error, beforeMetrics *ResourceStats) {
	// Record request optimizer metrics
	if pm.requestOptimizer != nil {
		pm.requestOptimizer.recordRequestMetrics(c, duration, err)
	}

	// Record cache access metrics
	if pm.cacheOptimizer != nil {
		// This would be called from cache operations
		cacheKey := generateCacheKey(c)
		hit := false // This would come from actual cache operation
		size := int64(len(c.Response().Body()))
		pm.cacheOptimizer.RecordCacheAccess(cacheKey, hit, size, duration)
	}

	// Record connection pool metrics
	if pm.connectionPool != nil {
		host := c.Hostname()
		success := err == nil && c.Response().StatusCode() < 400
		pm.connectionPool.RecordRequest(host, success, duration)
	}

	pm.logger.Debug("Performance metrics recorded",
		logging.Duration("duration", duration),
		logging.String("path", c.Path()),
		logging.String("method", c.Method()),
		logging.Int("status", c.Response().StatusCode()))
}

// triggerOptimizations triggers performance optimizations based on metrics
func (pm *PerformanceMiddleware) triggerOptimizations(c *fiber.Ctx, duration time.Duration) {
	// Trigger cache optimization for slow requests
	if pm.config.EnableCacheOptimization && duration > pm.config.SlowRequestThreshold {
		if pm.cacheOptimizer != nil {
			go func() {
				ctx := context.Background()
				reports, err := pm.cacheOptimizer.AnalyzePerformance(ctx)
				if err != nil {
					pm.logger.Error("Cache optimization analysis failed", logging.Error(err))
					return
				}

				if len(reports) > 0 {
					if err := pm.cacheOptimizer.ApplyOptimizations(ctx, reports); err != nil {
						pm.logger.Error("Cache optimization application failed", logging.Error(err))
					}
				}
			}()
		}
	}

	// Trigger resource monitoring alerts for critical requests
	if duration > pm.config.CriticalRequestThreshold {
		pm.logger.Warn("Critical request performance detected",
			logging.Duration("duration", duration),
			logging.String("path", c.Path()),
			logging.String("method", c.Method()))

		if pm.config.EnableAlerts {
			pm.triggerPerformanceAlert(c, duration)
		}
	}
}

// logPerformance logs performance information
func (pm *PerformanceMiddleware) logPerformance(c *fiber.Ctx, duration time.Duration, err error) {
	level := "info"
	if duration > pm.config.CriticalRequestThreshold {
		level = "error"
	} else if duration > pm.config.SlowRequestThreshold {
		level = "warn"
	}

	fields := []logging.Field{
		logging.Duration("duration", duration),
		logging.String("method", c.Method()),
		logging.String("path", c.Path()),
		logging.Int("status", c.Response().StatusCode()),
		logging.String("user_agent", c.Get("User-Agent")),
		logging.String("remote_ip", c.IP()),
	}

	if err != nil {
		fields = append(fields, logging.Error(err))
	}

	message := "Request processed"
	switch level {
	case "error":
		pm.logger.Error(message, fields...)
	case "warn":
		pm.logger.Warn(message, fields...)
	default:
		pm.logger.Info(message, fields...)
	}
}

// triggerPerformanceAlert triggers a performance alert
func (pm *PerformanceMiddleware) triggerPerformanceAlert(c *fiber.Ctx, duration time.Duration) {
	alert := map[string]interface{}{
		"type":      "performance",
		"severity":  "critical",
		"message":   "Critical request performance detected",
		"duration":  duration.String(),
		"threshold": pm.config.CriticalRequestThreshold.String(),
		"path":      c.Path(),
		"method":    c.Method(),
		"timestamp": time.Now(),
		"suggestions": []string{
			"Review endpoint implementation",
			"Check database query performance",
			"Analyze cache effectiveness",
			"Consider request optimization",
		},
	}

	pm.logger.Error("Performance alert triggered",
		logging.Any("alert", alert))
}

// GetMetrics returns comprehensive performance metrics
func (pm *PerformanceMiddleware) GetMetrics() *PerformanceMetrics {
	metrics := &PerformanceMetrics{
		LastUpdate: time.Now(),
	}

	// Get request optimizer metrics
	if pm.requestOptimizer != nil {
		reqStats := pm.requestOptimizer.GetStats()
		metrics.TotalRequests = reqStats.TotalRequests
		metrics.SlowRequests = reqStats.SlowRequests
		metrics.AverageResponseTime = reqStats.AverageResponseTime
		metrics.TimeoutErrors = reqStats.TimeoutErrors
		metrics.ErrorRate = float64(reqStats.OptimizationErrors) / float64(reqStats.TotalRequests)
	}

	// Get cache optimizer metrics
	if pm.cacheOptimizer != nil {
		cacheMetrics := pm.cacheOptimizer.GetCurrentMetrics()
		metrics.CacheHitRate = cacheMetrics.HitRate
		metrics.CacheMissRate = cacheMetrics.MissRate
	}

	// Get connection pool metrics
	if pm.connectionPool != nil {
		poolStats := pm.connectionPool.GetStats()
		metrics.ActiveConnections = poolStats.ActiveConnections
		metrics.ConnectionErrors = poolStats.ConnectionFailures
		if poolStats.TotalConnections > 0 {
			metrics.PoolUtilization = float64(poolStats.ActiveConnections) / float64(poolStats.TotalConnections)
		}
	}

	// Get resource monitor metrics
	if pm.resourceMonitor != nil {
		resourceStats := pm.resourceMonitor.GetStats()
		metrics.CPUUsage = resourceStats.CPUPercent
		metrics.MemoryUsage = resourceStats.MemoryPercent
		metrics.GoroutineCount = resourceStats.GoroutineCount
		metrics.OptimizationRuns = resourceStats.OptimizationRuns
	}

	return metrics
}

// GetHealthStatus returns health status based on performance metrics
func (pm *PerformanceMiddleware) GetHealthStatus() map[string]interface{} {
	metrics := pm.GetMetrics()

	status := map[string]interface{}{
		"overall_status": "healthy",
		"performance": map[string]interface{}{
			"status":                "healthy",
			"average_response_time": metrics.AverageResponseTime.String(),
			"slow_requests":         metrics.SlowRequests,
			"error_rate":            metrics.ErrorRate,
		},
		"resources": map[string]interface{}{
			"status":       "healthy",
			"cpu_usage":    metrics.CPUUsage,
			"memory_usage": metrics.MemoryUsage,
			"goroutines":   metrics.GoroutineCount,
		},
		"cache": map[string]interface{}{
			"status":    "healthy",
			"hit_rate":  metrics.CacheHitRate,
			"miss_rate": metrics.CacheMissRate,
		},
		"connections": map[string]interface{}{
			"status":      "healthy",
			"active":      metrics.ActiveConnections,
			"utilization": metrics.PoolUtilization,
			"errors":      metrics.ConnectionErrors,
		},
	}

	// Determine overall health
	overallHealthy := true

	// Check performance health
	if metrics.AverageResponseTime > pm.config.SlowRequestThreshold {
		status["performance"].(map[string]interface{})["status"] = "degraded"
		overallHealthy = false
	}

	if metrics.ErrorRate > 0.05 { // 5% error rate threshold
		status["performance"].(map[string]interface{})["status"] = "unhealthy"
		overallHealthy = false
	}

	// Check resource health
	if metrics.CPUUsage > 0.8 || metrics.MemoryUsage > 0.8 {
		status["resources"].(map[string]interface{})["status"] = "degraded"
		overallHealthy = false
	}

	// Check cache health
	if metrics.CacheHitRate < 0.5 { // Less than 50% hit rate
		status["cache"].(map[string]interface{})["status"] = "degraded"
	}

	// Check connection health
	if metrics.PoolUtilization > 0.9 { // 90% pool utilization
		status["connections"].(map[string]interface{})["status"] = "degraded"
	}

	if !overallHealthy {
		status["overall_status"] = "degraded"
	}

	return status
}

// Helper functions

func generateTraceID() string {
	// Simplified trace ID generation
	return "trace-" + string(rune(time.Now().UnixNano()))
}

func generateCacheKey(c *fiber.Ctx) string {
	return c.Method() + ":" + c.Path()
}
