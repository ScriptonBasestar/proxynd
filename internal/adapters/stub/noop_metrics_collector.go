package stub

import (
	"time"

	"proxynd/internal/logging"
	"proxynd/internal/ports"
)

// NoOpMetricsCollector is a no-operation implementation of ports.MetricsCollector
type NoOpMetricsCollector struct {
	logger logging.Logger
}

// NewNoOpMetricsCollector creates a new NoOp metrics collector
func NewNoOpMetricsCollector(logger logging.Logger) ports.MetricsCollector {
	return &NoOpMetricsCollector{logger: logger}
}

// IncCounter increments a counter (NoOp implementation)
func (mc *NoOpMetricsCollector) IncCounter(name string, labels map[string]string) {
	mc.logger.Debug("NoOpMetricsCollector.IncCounter called (stub)",
		logging.F("name", name))
}

// AddCounter adds to a counter (NoOp implementation)
func (mc *NoOpMetricsCollector) AddCounter(name string, value float64, labels map[string]string) {
	mc.logger.Debug("NoOpMetricsCollector.AddCounter called (stub)",
		logging.F("name", name),
		logging.F("value", value))
}

// SetGauge sets a gauge value (NoOp implementation)
func (mc *NoOpMetricsCollector) SetGauge(name string, value float64, labels map[string]string) {
	mc.logger.Debug("NoOpMetricsCollector.SetGauge called (stub)",
		logging.F("name", name),
		logging.F("value", value))
}

// AddGauge adds to a gauge (NoOp implementation)
func (mc *NoOpMetricsCollector) AddGauge(name string, value float64, labels map[string]string) {
	mc.logger.Debug("NoOpMetricsCollector.AddGauge called (stub)",
		logging.F("name", name),
		logging.F("value", value))
}

// ObserveHistogram observes a histogram value (NoOp implementation)
func (mc *NoOpMetricsCollector) ObserveHistogram(name string, value float64, labels map[string]string) {
	mc.logger.Debug("NoOpMetricsCollector.ObserveHistogram called (stub)",
		logging.F("name", name),
		logging.F("value", value))
}

// ObserveSummary observes a summary value (NoOp implementation)
func (mc *NoOpMetricsCollector) ObserveSummary(name string, value float64, labels map[string]string) {
	mc.logger.Debug("NoOpMetricsCollector.ObserveSummary called (stub)",
		logging.F("name", name),
		logging.F("value", value))
}

// StartTimer starts a timer (NoOp implementation)
func (mc *NoOpMetricsCollector) StartTimer(name string, labels map[string]string) ports.Timer {
	mc.logger.Debug("NoOpMetricsCollector.StartTimer called (stub)",
		logging.F("name", name))

	return &noOpTimer{startTime: time.Now()}
}

// GetMetrics returns metrics snapshot (NoOp implementation)
func (mc *NoOpMetricsCollector) GetMetrics() (*ports.MetricsSnapshot, error) {
	mc.logger.Debug("NoOpMetricsCollector.GetMetrics called (stub)")

	return &ports.MetricsSnapshot{
		Timestamp:  time.Now(),
		Counters:   make(map[string]float64),
		Gauges:     make(map[string]float64),
		Histograms: make(map[string]*ports.Histogram),
		Summaries:  make(map[string]*ports.Summary),
	}, nil
}

// noOpTimer is a no-operation timer implementation
type noOpTimer struct {
	startTime time.Time
}

// Stop stops the timer (NoOp implementation)
func (t *noOpTimer) Stop() {
	// No operation
}

// Duration returns elapsed duration
func (t *noOpTimer) Duration() time.Duration {
	return time.Since(t.startTime)
}
