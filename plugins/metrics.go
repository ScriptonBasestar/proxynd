package plugins

import (
	"sync"
	"time"
)

// PluginMetrics tracks metrics for a single plugin
type PluginMetrics struct {
	PluginName      string        `json:"plugin_name"`
	Priority        int           `json:"priority"`
	Status          PluginStatus  `json:"status"`
	InitTime        time.Duration `json:"init_time_ms"`
	ReadyTime       time.Duration `json:"ready_time_ms"`
	ShutdownTime    time.Duration `json:"shutdown_time_ms"`
	ErrorCount      int           `json:"error_count"`
	LastError       string        `json:"last_error,omitempty"`
	LastErrorTime   time.Time     `json:"last_error_time,omitempty"`
	StateChanges    int           `json:"state_changes"`
	LastStateChange time.Time     `json:"last_state_change"`
	Enabled         bool          `json:"enabled"`
}

// PluginStatus represents the current state of a plugin
type PluginStatus string

const (
	PluginStatusUnknown      PluginStatus = "unknown"
	PluginStatusDiscovered   PluginStatus = "discovered"
	PluginStatusInitializing PluginStatus = "initializing"
	PluginStatusInitialized  PluginStatus = "initialized"
	PluginStatusReady        PluginStatus = "ready"
	PluginStatusFailed       PluginStatus = "failed"
	PluginStatusShuttingDown PluginStatus = "shutting_down"
	PluginStatusShutdown     PluginStatus = "shutdown"
)

// MetricsCollector collects and aggregates plugin metrics
type MetricsCollector struct {
	metrics map[string]*PluginMetrics
	mu      sync.RWMutex
	enabled bool
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(enabled bool) *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[string]*PluginMetrics),
		enabled: enabled,
	}
}

// RecordPluginDiscovered records when a plugin is discovered
func (mc *MetricsCollector) RecordPluginDiscovered(name string, priority int, enabled bool) {
	if !mc.enabled {
		return
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.metrics[name] = &PluginMetrics{
		PluginName:      name,
		Priority:        priority,
		Status:          PluginStatusDiscovered,
		Enabled:         enabled,
		StateChanges:    1,
		LastStateChange: time.Now(),
	}
}

// RecordPluginInit records plugin initialization metrics
func (mc *MetricsCollector) RecordPluginInit(name string, duration time.Duration, err error) {
	if !mc.enabled {
		return
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	m, ok := mc.metrics[name]
	if !ok {
		m = &PluginMetrics{PluginName: name}
		mc.metrics[name] = m
	}

	m.InitTime = duration
	m.StateChanges++
	m.LastStateChange = time.Now()

	if err != nil {
		m.Status = PluginStatusFailed
		m.ErrorCount++
		m.LastError = err.Error()
		m.LastErrorTime = time.Now()
	} else {
		m.Status = PluginStatusInitialized
	}
}

// RecordPluginReady records when a plugin becomes ready
func (mc *MetricsCollector) RecordPluginReady(name string, duration time.Duration, err error) {
	if !mc.enabled {
		return
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	m, ok := mc.metrics[name]
	if !ok {
		m = &PluginMetrics{PluginName: name}
		mc.metrics[name] = m
	}

	m.ReadyTime = duration
	m.StateChanges++
	m.LastStateChange = time.Now()

	if err != nil {
		m.Status = PluginStatusFailed
		m.ErrorCount++
		m.LastError = err.Error()
		m.LastErrorTime = time.Now()
	} else {
		m.Status = PluginStatusReady
	}
}

// RecordPluginShutdown records plugin shutdown metrics
func (mc *MetricsCollector) RecordPluginShutdown(name string, duration time.Duration, err error) {
	if !mc.enabled {
		return
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	m, ok := mc.metrics[name]
	if !ok {
		m = &PluginMetrics{PluginName: name}
		mc.metrics[name] = m
	}

	m.ShutdownTime = duration
	m.StateChanges++
	m.LastStateChange = time.Now()

	if err != nil {
		m.Status = PluginStatusFailed
		m.ErrorCount++
		m.LastError = err.Error()
		m.LastErrorTime = time.Now()
	} else {
		m.Status = PluginStatusShutdown
	}
}

// RecordPluginStateChange records a general state change
func (mc *MetricsCollector) RecordPluginStateChange(name string, newStatus PluginStatus) {
	if !mc.enabled {
		return
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	m, ok := mc.metrics[name]
	if !ok {
		m = &PluginMetrics{PluginName: name}
		mc.metrics[name] = m
	}

	m.Status = newStatus
	m.StateChanges++
	m.LastStateChange = time.Now()
}

// GetPluginMetrics returns metrics for a specific plugin
func (mc *MetricsCollector) GetPluginMetrics(name string) *PluginMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	m, ok := mc.metrics[name]
	if !ok {
		return nil
	}

	// Return a copy to prevent concurrent modification
	metricsCopy := *m
	return &metricsCopy
}

// GetAllMetrics returns metrics for all plugins
func (mc *MetricsCollector) GetAllMetrics() map[string]*PluginMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	result := make(map[string]*PluginMetrics, len(mc.metrics))
	for name, m := range mc.metrics {
		metricsCopy := *m
		result[name] = &metricsCopy
	}

	return result
}

// GetSummary returns aggregated metrics summary
func (mc *MetricsCollector) GetSummary() MetricsSummary {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	summary := MetricsSummary{
		TotalPlugins:      len(mc.metrics),
		EnabledPlugins:    0,
		DisabledPlugins:   0,
		ReadyPlugins:      0,
		FailedPlugins:     0,
		TotalErrors:       0,
		TotalStateChanges: 0,
	}

	for _, m := range mc.metrics {
		if m.Enabled {
			summary.EnabledPlugins++
		} else {
			summary.DisabledPlugins++
		}

		switch m.Status {
		case PluginStatusReady:
			summary.ReadyPlugins++
		case PluginStatusFailed:
			summary.FailedPlugins++
		}

		summary.TotalErrors += m.ErrorCount
		summary.TotalStateChanges += m.StateChanges
		summary.TotalInitTime += m.InitTime
		summary.TotalReadyTime += m.ReadyTime
		summary.TotalShutdownTime += m.ShutdownTime
	}

	if summary.TotalPlugins > 0 {
		summary.AverageInitTime = summary.TotalInitTime / time.Duration(summary.TotalPlugins)
		summary.AverageReadyTime = summary.TotalReadyTime / time.Duration(summary.TotalPlugins)
	}

	return summary
}

// MetricsSummary provides aggregated metrics across all plugins
type MetricsSummary struct {
	TotalPlugins      int           `json:"total_plugins"`
	EnabledPlugins    int           `json:"enabled_plugins"`
	DisabledPlugins   int           `json:"disabled_plugins"`
	ReadyPlugins      int           `json:"ready_plugins"`
	FailedPlugins     int           `json:"failed_plugins"`
	TotalErrors       int           `json:"total_errors"`
	TotalStateChanges int           `json:"total_state_changes"`
	TotalInitTime     time.Duration `json:"total_init_time_ms"`
	TotalReadyTime    time.Duration `json:"total_ready_time_ms"`
	TotalShutdownTime time.Duration `json:"total_shutdown_time_ms"`
	AverageInitTime   time.Duration `json:"average_init_time_ms"`
	AverageReadyTime  time.Duration `json:"average_ready_time_ms"`
}

// Reset clears all collected metrics
func (mc *MetricsCollector) Reset() {
	if !mc.enabled {
		return
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.metrics = make(map[string]*PluginMetrics)
}
