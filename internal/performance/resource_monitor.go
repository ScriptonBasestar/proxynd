package performance

import (
	"context"
	"runtime"
	"sync"
	"time"

	"proxynd/logging"
)

// ResourceMonitor monitors system resource usage
type ResourceMonitor struct {
	logger    logging.Logger
	config    *ResourceConfig
	stats     *ResourceStats
	mu        sync.RWMutex
	stopCh    chan struct{}
	isRunning bool
}

// ResourceConfig configures resource monitoring
type ResourceConfig struct {
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

// ResourceStats tracks resource usage statistics
type ResourceStats struct {
	mu sync.RWMutex

	// Memory statistics
	MemoryUsed    uint64    `json:"memory_used"`
	MemoryTotal   uint64    `json:"memory_total"`
	MemoryPercent float64   `json:"memory_percent"`
	HeapUsed      uint64    `json:"heap_used"`
	HeapIdle      uint64    `json:"heap_idle"`
	HeapInuse     uint64    `json:"heap_inuse"`
	GCCount       uint32    `json:"gc_count"`
	LastGC        time.Time `json:"last_gc"`

	// CPU statistics
	CPUPercent     float64 `json:"cpu_percent"`
	GoroutineCount int     `json:"goroutine_count"`

	// Disk statistics
	DiskUsed    uint64  `json:"disk_used"`
	DiskTotal   uint64  `json:"disk_total"`
	DiskPercent float64 `json:"disk_percent"`

	// Network statistics
	NetworkIn  uint64 `json:"network_in"`
	NetworkOut uint64 `json:"network_out"`

	// Optimization statistics
	OptimizationRuns int64     `json:"optimization_runs"`
	LastOptimization time.Time `json:"last_optimization"`
	GCTriggered      int64     `json:"gc_triggered"`

	// Alerts
	MemoryAlerts    int64 `json:"memory_alerts"`
	CPUAlerts       int64 `json:"cpu_alerts"`
	DiskAlerts      int64 `json:"disk_alerts"`
	GoroutineAlerts int64 `json:"goroutine_alerts"`
}

// ResourceAlert represents a resource usage alert
type ResourceAlert struct {
	Type      string      `json:"type"`
	Severity  string      `json:"severity"`
	Message   string      `json:"message"`
	Value     interface{} `json:"value"`
	Threshold interface{} `json:"threshold"`
	Timestamp time.Time   `json:"timestamp"`
	Suggested []string    `json:"suggested_actions"`
}

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor(logger logging.Logger, config *ResourceConfig) *ResourceMonitor {
	return &ResourceMonitor{
		logger: logger.WithComponent("resource.monitor"),
		config: config,
		stats:  NewResourceStats(),
		stopCh: make(chan struct{}),
	}
}

// NewResourceStats creates new resource statistics
func NewResourceStats() *ResourceStats {
	return &ResourceStats{}
}

// Start starts resource monitoring
func (rm *ResourceMonitor) Start(ctx context.Context) error {
	rm.mu.Lock()
	if rm.isRunning {
		rm.mu.Unlock()
		return nil
	}
	rm.isRunning = true
	rm.mu.Unlock()

	rm.logger.Info("Starting resource monitoring")

	// Start monitoring goroutines
	go rm.monitorMemory(ctx)
	go rm.monitorCPU(ctx)
	go rm.monitorDisk(ctx)
	go rm.monitorNetwork(ctx)

	if rm.config.EnableOptimization {
		go rm.optimizationLoop(ctx)
	}

	return nil
}

// Stop stops resource monitoring
func (rm *ResourceMonitor) Stop() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if !rm.isRunning {
		return
	}

	rm.isRunning = false
	close(rm.stopCh)
	rm.logger.Info("Resource monitoring stopped")
}

// GetStats returns current resource statistics
func (rm *ResourceMonitor) GetStats() *ResourceStats {
	rm.stats.mu.RLock()
	defer rm.stats.mu.RUnlock()

	// Create a copy to avoid race conditions
	stats := &ResourceStats{
		MemoryUsed:       rm.stats.MemoryUsed,
		MemoryTotal:      rm.stats.MemoryTotal,
		MemoryPercent:    rm.stats.MemoryPercent,
		HeapUsed:         rm.stats.HeapUsed,
		HeapIdle:         rm.stats.HeapIdle,
		HeapInuse:        rm.stats.HeapInuse,
		GCCount:          rm.stats.GCCount,
		LastGC:           rm.stats.LastGC,
		CPUPercent:       rm.stats.CPUPercent,
		GoroutineCount:   rm.stats.GoroutineCount,
		DiskUsed:         rm.stats.DiskUsed,
		DiskTotal:        rm.stats.DiskTotal,
		DiskPercent:      rm.stats.DiskPercent,
		NetworkIn:        rm.stats.NetworkIn,
		NetworkOut:       rm.stats.NetworkOut,
		OptimizationRuns: rm.stats.OptimizationRuns,
		LastOptimization: rm.stats.LastOptimization,
		GCTriggered:      rm.stats.GCTriggered,
		MemoryAlerts:     rm.stats.MemoryAlerts,
		CPUAlerts:        rm.stats.CPUAlerts,
		DiskAlerts:       rm.stats.DiskAlerts,
		GoroutineAlerts:  rm.stats.GoroutineAlerts,
	}

	return stats
}

// monitorMemory monitors memory usage
func (rm *ResourceMonitor) monitorMemory(ctx context.Context) {
	ticker := time.NewTicker(rm.config.MemoryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-rm.stopCh:
			return
		case <-ticker.C:
			rm.updateMemoryStats()
		}
	}
}

// monitorCPU monitors CPU usage and goroutines
func (rm *ResourceMonitor) monitorCPU(ctx context.Context) {
	ticker := time.NewTicker(rm.config.CPUInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-rm.stopCh:
			return
		case <-ticker.C:
			rm.updateCPUStats()
		}
	}
}

// monitorDisk monitors disk usage
func (rm *ResourceMonitor) monitorDisk(ctx context.Context) {
	ticker := time.NewTicker(rm.config.DiskInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-rm.stopCh:
			return
		case <-ticker.C:
			rm.updateDiskStats()
		}
	}
}

// monitorNetwork monitors network usage
func (rm *ResourceMonitor) monitorNetwork(ctx context.Context) {
	ticker := time.NewTicker(rm.config.NetworkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-rm.stopCh:
			return
		case <-ticker.C:
			rm.updateNetworkStats()
		}
	}
}

// updateMemoryStats updates memory statistics
func (rm *ResourceMonitor) updateMemoryStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	rm.stats.mu.Lock()
	defer rm.stats.mu.Unlock()

	rm.stats.HeapUsed = m.HeapAlloc
	rm.stats.HeapIdle = m.HeapIdle
	rm.stats.HeapInuse = m.HeapInuse
	rm.stats.GCCount = m.NumGC
	rm.stats.LastGC = time.Unix(0, int64(m.LastGC))

	// Calculate memory percentage (simplified)
	totalMemory := uint64(8 * 1024 * 1024 * 1024) // Assume 8GB, in practice get from system
	rm.stats.MemoryUsed = m.Sys
	rm.stats.MemoryTotal = totalMemory
	rm.stats.MemoryPercent = float64(rm.stats.MemoryUsed) / float64(totalMemory)

	// Check memory threshold
	if rm.stats.MemoryPercent > rm.config.MemoryThreshold {
		rm.stats.MemoryAlerts++
		rm.triggerMemoryAlert()

		// Trigger GC if configured
		if rm.config.EnableAutoGC && rm.stats.MemoryPercent > rm.config.GCThreshold {
			rm.triggerGC()
		}
	}

	rm.logger.Debug("Memory stats updated",
		logging.Uint64("heap_used", rm.stats.HeapUsed),
		logging.Float64("memory_percent", rm.stats.MemoryPercent),
		logging.Uint32("gc_count", rm.stats.GCCount))
}

// updateCPUStats updates CPU and goroutine statistics
func (rm *ResourceMonitor) updateCPUStats() {
	rm.stats.mu.Lock()
	defer rm.stats.mu.Unlock()

	// Get goroutine count
	rm.stats.GoroutineCount = runtime.NumGoroutine()

	// Simplified CPU usage (in practice, use system calls or third-party libs)
	rm.stats.CPUPercent = 0.0 // Placeholder

	// Check goroutine threshold
	if rm.stats.GoroutineCount > rm.config.GoroutineThreshold {
		rm.stats.GoroutineAlerts++
		rm.triggerGoroutineAlert()
	}

	rm.logger.Debug("CPU stats updated",
		logging.Int("goroutine_count", rm.stats.GoroutineCount),
		logging.Float64("cpu_percent", rm.stats.CPUPercent))
}

// updateDiskStats updates disk usage statistics
func (rm *ResourceMonitor) updateDiskStats() {
	rm.stats.mu.Lock()
	defer rm.stats.mu.Unlock()

	// Simplified disk usage (in practice, use syscalls)
	rm.stats.DiskUsed = 0      // Placeholder
	rm.stats.DiskTotal = 0     // Placeholder
	rm.stats.DiskPercent = 0.0 // Placeholder

	// Check disk threshold
	if rm.stats.DiskPercent > rm.config.DiskThreshold {
		rm.stats.DiskAlerts++
		rm.triggerDiskAlert()
	}

	rm.logger.Debug("Disk stats updated",
		logging.Uint64("disk_used", rm.stats.DiskUsed),
		logging.Float64("disk_percent", rm.stats.DiskPercent))
}

// updateNetworkStats updates network statistics
func (rm *ResourceMonitor) updateNetworkStats() {
	rm.stats.mu.Lock()
	defer rm.stats.mu.Unlock()

	// Simplified network stats (in practice, use system monitoring)
	rm.stats.NetworkIn = 0  // Placeholder
	rm.stats.NetworkOut = 0 // Placeholder

	rm.logger.Debug("Network stats updated",
		logging.Uint64("network_in", rm.stats.NetworkIn),
		logging.Uint64("network_out", rm.stats.NetworkOut))
}

// optimizationLoop performs periodic resource optimization
func (rm *ResourceMonitor) optimizationLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute) // Optimization every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-rm.stopCh:
			return
		case <-ticker.C:
			rm.performOptimization()
		}
	}
}

// performOptimization performs resource optimization
func (rm *ResourceMonitor) performOptimization() {
	rm.stats.mu.Lock()
	rm.stats.OptimizationRuns++
	rm.stats.LastOptimization = time.Now()
	rm.stats.mu.Unlock()

	// Memory optimization
	if rm.stats.MemoryPercent > 0.6 { // 60% threshold for optimization
		rm.optimizeMemory()
	}

	// Goroutine optimization
	if rm.stats.GoroutineCount > rm.config.GoroutineThreshold/2 {
		rm.optimizeGoroutines()
	}

	rm.logger.Info("Resource optimization completed",
		logging.Float64("memory_percent", rm.stats.MemoryPercent),
		logging.Int("goroutine_count", rm.stats.GoroutineCount))
}

// optimizeMemory performs memory optimization
func (rm *ResourceMonitor) optimizeMemory() {
	// Force GC if memory usage is high
	if rm.config.EnableAutoGC {
		rm.triggerGC()
	}

	rm.logger.Info("Memory optimization performed")
}

// optimizeGoroutines performs goroutine optimization
func (rm *ResourceMonitor) optimizeGoroutines() {
	// Log goroutine stack for analysis
	rm.logger.Warn("High goroutine count detected, consider reviewing goroutine management",
		logging.Int("count", rm.stats.GoroutineCount),
		logging.Int("threshold", rm.config.GoroutineThreshold))
}

// triggerGC triggers garbage collection
func (rm *ResourceMonitor) triggerGC() {
	runtime.GC()

	rm.stats.mu.Lock()
	rm.stats.GCTriggered++
	rm.stats.mu.Unlock()

	rm.logger.Info("Garbage collection triggered",
		logging.F("gc_triggered_count", rm.stats.GCTriggered))
}

// triggerMemoryAlert triggers a memory usage alert
func (rm *ResourceMonitor) triggerMemoryAlert() {
	alert := ResourceAlert{
		Type:      "memory",
		Severity:  "warning",
		Message:   "High memory usage detected",
		Value:     rm.stats.MemoryPercent,
		Threshold: rm.config.MemoryThreshold,
		Timestamp: time.Now(),
		Suggested: []string{
			"Consider increasing available memory",
			"Review memory-intensive operations",
			"Check for memory leaks",
			"Optimize cache sizes",
		},
	}

	rm.logger.Warn("Memory usage alert",
		logging.Float64("usage_percent", rm.stats.MemoryPercent),
		logging.Float64("threshold", rm.config.MemoryThreshold),
		logging.Uint64("heap_used", rm.stats.HeapUsed))

	// In a real implementation, this would send alerts to monitoring systems
	rm.logAlert(alert)
}

// triggerGoroutineAlert triggers a goroutine count alert
func (rm *ResourceMonitor) triggerGoroutineAlert() {
	alert := ResourceAlert{
		Type:      "goroutine",
		Severity:  "warning",
		Message:   "High goroutine count detected",
		Value:     rm.stats.GoroutineCount,
		Threshold: rm.config.GoroutineThreshold,
		Timestamp: time.Now(),
		Suggested: []string{
			"Review goroutine lifecycle management",
			"Check for goroutine leaks",
			"Implement proper goroutine pooling",
			"Monitor concurrent operations",
		},
	}

	rm.logger.Warn("Goroutine count alert",
		logging.Int("count", rm.stats.GoroutineCount),
		logging.Int("threshold", rm.config.GoroutineThreshold))

	rm.logAlert(alert)
}

// triggerDiskAlert triggers a disk usage alert
func (rm *ResourceMonitor) triggerDiskAlert() {
	alert := ResourceAlert{
		Type:      "disk",
		Severity:  "critical",
		Message:   "High disk usage detected",
		Value:     rm.stats.DiskPercent,
		Threshold: rm.config.DiskThreshold,
		Timestamp: time.Now(),
		Suggested: []string{
			"Clean up old log files",
			"Review cache storage usage",
			"Implement log rotation",
			"Monitor disk space regularly",
		},
	}

	rm.logger.Error("Disk usage alert",
		logging.Float64("usage_percent", rm.stats.DiskPercent),
		logging.Float64("threshold", rm.config.DiskThreshold),
		logging.Uint64("disk_used", rm.stats.DiskUsed))

	rm.logAlert(alert)
}

// logAlert logs a resource alert
func (rm *ResourceMonitor) logAlert(alert ResourceAlert) {
	rm.logger.Warn("Resource alert triggered",
		logging.String("alert_type", alert.Type),
		logging.String("severity", alert.Severity),
		logging.String("message", alert.Message),
		logging.Time("timestamp", alert.Timestamp))
}

// GetHealthStatus returns overall system health status
func (rm *ResourceMonitor) GetHealthStatus() map[string]interface{} {
	stats := rm.GetStats()

	status := map[string]interface{}{
		"overall_status": "healthy",
		"memory": map[string]interface{}{
			"status":    "healthy",
			"usage":     stats.MemoryPercent,
			"threshold": rm.config.MemoryThreshold,
		},
		"cpu": map[string]interface{}{
			"status":    "healthy",
			"usage":     stats.CPUPercent,
			"threshold": rm.config.CPUThreshold,
		},
		"disk": map[string]interface{}{
			"status":    "healthy",
			"usage":     stats.DiskPercent,
			"threshold": rm.config.DiskThreshold,
		},
		"goroutines": map[string]interface{}{
			"status":    "healthy",
			"count":     stats.GoroutineCount,
			"threshold": rm.config.GoroutineThreshold,
		},
	}

	// Determine overall status
	overallHealthy := true

	if stats.MemoryPercent > rm.config.MemoryThreshold {
		status["memory"].(map[string]interface{})["status"] = "warning"
		overallHealthy = false
	}

	if stats.CPUPercent > rm.config.CPUThreshold {
		status["cpu"].(map[string]interface{})["status"] = "warning"
		overallHealthy = false
	}

	if stats.DiskPercent > rm.config.DiskThreshold {
		status["disk"].(map[string]interface{})["status"] = "critical"
		overallHealthy = false
	}

	if stats.GoroutineCount > rm.config.GoroutineThreshold {
		status["goroutines"].(map[string]interface{})["status"] = "warning"
		overallHealthy = false
	}

	if !overallHealthy {
		status["overall_status"] = "degraded"
	}

	return status
}
