package prometheus

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	dto "github.com/prometheus/client_model/go"

	"proxynd/internal/ports"
)

// MetricsCollector implements ports.MetricsCollector using Prometheus
type MetricsCollector struct {
	registry   prometheus.Registerer
	namespace  string
	subsystem  string
	counters   map[string]*prometheus.CounterVec
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec
	summaries  map[string]*prometheus.SummaryVec
	mutex      sync.RWMutex
	factory    promauto.Factory
}

// Timer implements ports.Timer using Prometheus
type Timer struct {
	histogram *prometheus.HistogramVec
	labels    prometheus.Labels
	startTime time.Time
}

// NewMetricsCollector creates a new Prometheus metrics collector
func NewMetricsCollector(config *MetricsConfig) ports.MetricsCollector {
	registry := prometheus.DefaultRegisterer
	if config.Registry != nil {
		registry = config.Registry
	}

	namespace := config.Namespace
	if namespace == "" {
		namespace = "proxynd"
	}

	subsystem := config.Subsystem
	if subsystem == "" {
		subsystem = "http"
	}

	factory := promauto.With(registry)

	return &MetricsCollector{
		registry:   registry,
		namespace:  namespace,
		subsystem:  subsystem,
		counters:   make(map[string]*prometheus.CounterVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
		histograms: make(map[string]*prometheus.HistogramVec),
		summaries:  make(map[string]*prometheus.SummaryVec),
		factory:    factory,
	}
}

// IncCounter increments a counter
func (m *MetricsCollector) IncCounter(name string, labels map[string]string) {
	m.AddCounter(name, 1, labels)
}

// AddCounter adds to a counter
func (m *MetricsCollector) AddCounter(name string, value float64, labels map[string]string) {
	counter := m.getOrCreateCounter(name, labels)
	labelValues := m.getLabelValues(labels)
	counter.With(labelValues).Add(value)
}

// SetGauge sets a gauge value
func (m *MetricsCollector) SetGauge(name string, value float64, labels map[string]string) {
	gauge := m.getOrCreateGauge(name, labels)
	labelValues := m.getLabelValues(labels)
	gauge.With(labelValues).Set(value)
}

// AddGauge adds to a gauge
func (m *MetricsCollector) AddGauge(name string, value float64, labels map[string]string) {
	gauge := m.getOrCreateGauge(name, labels)
	labelValues := m.getLabelValues(labels)
	gauge.With(labelValues).Add(value)
}

// ObserveHistogram observes a value in a histogram
func (m *MetricsCollector) ObserveHistogram(name string, value float64, labels map[string]string) {
	histogram := m.getOrCreateHistogram(name, labels)
	labelValues := m.getLabelValues(labels)
	histogram.With(labelValues).Observe(value)
}

// ObserveSummary observes a value in a summary
func (m *MetricsCollector) ObserveSummary(name string, value float64, labels map[string]string) {
	summary := m.getOrCreateSummary(name, labels)
	labelValues := m.getLabelValues(labels)
	summary.With(labelValues).Observe(value)
}

// StartTimer starts a new timer
func (m *MetricsCollector) StartTimer(name string, labels map[string]string) ports.Timer {
	histogram := m.getOrCreateHistogram(name+"_duration_seconds", labels)
	labelValues := m.getLabelValues(labels)

	return &Timer{
		histogram: histogram,
		labels:    labelValues,
		startTime: time.Now(),
	}
}

// GetMetrics returns current metrics snapshot
func (m *MetricsCollector) GetMetrics() (*ports.MetricsSnapshot, error) {
	// Gather metrics from Prometheus registry
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		return nil, fmt.Errorf("failed to gather metrics: %w", err)
	}

	snapshot := &ports.MetricsSnapshot{
		Timestamp:  time.Now(),
		Counters:   make(map[string]float64),
		Gauges:     make(map[string]float64),
		Histograms: make(map[string]*ports.Histogram),
		Summaries:  make(map[string]*ports.Summary),
	}

	// Convert Prometheus metrics to snapshot format
	for _, mf := range metricFamilies {
		name := mf.GetName()

		switch mf.GetType() {
		case dto.MetricType_COUNTER:
			for _, metric := range mf.GetMetric() {
				key := m.buildMetricKey(name, metric.GetLabel())
				snapshot.Counters[key] = metric.GetCounter().GetValue()
			}
		case dto.MetricType_GAUGE:
			for _, metric := range mf.GetMetric() {
				key := m.buildMetricKey(name, metric.GetLabel())
				snapshot.Gauges[key] = metric.GetGauge().GetValue()
			}
		case dto.MetricType_HISTOGRAM:
			for _, metric := range mf.GetMetric() {
				key := m.buildMetricKey(name, metric.GetLabel())
				hist := metric.GetHistogram()

				buckets := make(map[float64]uint64)
				for _, bucket := range hist.GetBucket() {
					buckets[bucket.GetUpperBound()] = bucket.GetCumulativeCount()
				}

				snapshot.Histograms[key] = &ports.Histogram{
					Count:   hist.GetSampleCount(),
					Sum:     hist.GetSampleSum(),
					Buckets: buckets,
				}
			}
		case dto.MetricType_SUMMARY:
			for _, metric := range mf.GetMetric() {
				key := m.buildMetricKey(name, metric.GetLabel())
				sum := metric.GetSummary()

				quantiles := make(map[float64]float64)
				for _, quantile := range sum.GetQuantile() {
					quantiles[quantile.GetQuantile()] = quantile.GetValue()
				}

				snapshot.Summaries[key] = &ports.Summary{
					Count:     sum.GetSampleCount(),
					Sum:       sum.GetSampleSum(),
					Quantiles: quantiles,
				}
			}
		}
	}

	return snapshot, nil
}

// Stop stops the timer and records the duration
func (t *Timer) Stop() {
	duration := time.Since(t.startTime)
	t.histogram.With(t.labels).Observe(duration.Seconds())
}

// Duration returns elapsed duration
func (t *Timer) Duration() time.Duration {
	return time.Since(t.startTime)
}

// getOrCreateCounter gets or creates a counter
func (m *MetricsCollector) getOrCreateCounter(name string, labels map[string]string) *prometheus.CounterVec {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := m.buildKey(name)
	if counter, exists := m.counters[key]; exists {
		return counter
	}

	labelNames := m.getLabelNames(labels)
	counter := m.factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      name,
		Help:      fmt.Sprintf("Counter metric for %s", name),
	}, labelNames)

	m.counters[key] = counter
	return counter
}

// getOrCreateGauge gets or creates a gauge
func (m *MetricsCollector) getOrCreateGauge(name string, labels map[string]string) *prometheus.GaugeVec {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := m.buildKey(name)
	if gauge, exists := m.gauges[key]; exists {
		return gauge
	}

	labelNames := m.getLabelNames(labels)
	gauge := m.factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      name,
		Help:      fmt.Sprintf("Gauge metric for %s", name),
	}, labelNames)

	m.gauges[key] = gauge
	return gauge
}

// getOrCreateHistogram gets or creates a histogram
func (m *MetricsCollector) getOrCreateHistogram(name string, labels map[string]string) *prometheus.HistogramVec {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := m.buildKey(name)
	if histogram, exists := m.histograms[key]; exists {
		return histogram
	}

	labelNames := m.getLabelNames(labels)
	histogram := m.factory.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      name,
		Help:      fmt.Sprintf("Histogram metric for %s", name),
		Buckets:   prometheus.DefBuckets,
	}, labelNames)

	m.histograms[key] = histogram
	return histogram
}

// getOrCreateSummary gets or creates a summary
func (m *MetricsCollector) getOrCreateSummary(name string, labels map[string]string) *prometheus.SummaryVec {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := m.buildKey(name)
	if summary, exists := m.summaries[key]; exists {
		return summary
	}

	labelNames := m.getLabelNames(labels)
	summary := m.factory.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      name,
		Help:      fmt.Sprintf("Summary metric for %s", name),
		Objectives: map[float64]float64{
			0.5:  0.05,
			0.9:  0.01,
			0.95: 0.005,
			0.99: 0.001,
		},
	}, labelNames)

	m.summaries[key] = summary
	return summary
}

// buildKey builds a cache key for metrics
func (m *MetricsCollector) buildKey(name string) string {
	return fmt.Sprintf("%s_%s_%s", m.namespace, m.subsystem, name)
}

// getLabelNames extracts label names from labels map
func (m *MetricsCollector) getLabelNames(labels map[string]string) []string {
	if len(labels) == 0 {
		return []string{}
	}

	names := make([]string, 0, len(labels))
	for name := range labels {
		names = append(names, name)
	}
	return names
}

// getLabelValues extracts label values from labels map
func (m *MetricsCollector) getLabelValues(labels map[string]string) prometheus.Labels {
	if len(labels) == 0 {
		return prometheus.Labels{}
	}

	values := make(prometheus.Labels, len(labels))
	for name, value := range labels {
		values[name] = value
	}
	return values
}

// buildMetricKey builds a metric key with labels
func (m *MetricsCollector) buildMetricKey(name string, labels []*dto.LabelPair) string {
	if len(labels) == 0 {
		return name
	}

	var parts []string
	parts = append(parts, name)

	for _, label := range labels {
		parts = append(parts, fmt.Sprintf("%s=%s", label.GetName(), label.GetValue()))
	}

	return strings.Join(parts, ",")
}

// MetricsConfig defines Prometheus metrics configuration
type MetricsConfig struct {
	Namespace string                `json:"namespace" yaml:"namespace"`
	Subsystem string                `json:"subsystem" yaml:"subsystem"`
	Registry  prometheus.Registerer `json:"-" yaml:"-"`
}

// DefaultMetricsConfig returns default metrics configuration
func DefaultMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		Namespace: "proxynd",
		Subsystem: "http",
		Registry:  prometheus.DefaultRegisterer,
	}
}
