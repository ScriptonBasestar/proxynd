package plugins

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// pluginEventsTotal counts total events dispatched by type
	pluginEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "proxynd",
			Subsystem: "plugin",
			Name:      "events_total",
			Help:      "Total number of plugin events dispatched by type",
		},
		[]string{"event_type"},
	)

	// pluginEventProcessingDuration tracks event processing duration
	pluginEventProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "proxynd",
			Subsystem: "plugin",
			Name:      "event_processing_duration_seconds",
			Help:      "Duration of plugin event processing in seconds",
			Buckets:   prometheus.DefBuckets, // Default buckets: 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10
		},
		[]string{"event_type", "plugin_name"},
	)

	// pluginEventErrorsTotal counts event processing errors
	pluginEventErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "proxynd",
			Subsystem: "plugin",
			Name:      "event_errors_total",
			Help:      "Total number of plugin event processing errors",
		},
		[]string{"event_type", "plugin_name", "error_type"},
	)

	// pluginEventListenersTotal tracks number of event listeners
	pluginEventListenersTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "proxynd",
			Subsystem: "plugin",
			Name:      "event_listeners_total",
			Help:      "Current number of plugins listening to events",
		},
		[]string{"event_type"},
	)
)

// RecordEventDispatched records when an event is dispatched
func RecordEventDispatched(eventType string) {
	pluginEventsTotal.WithLabelValues(eventType).Inc()
}

// RecordEventProcessing records event processing duration
func RecordEventProcessing(eventType, pluginName string, durationSeconds float64) {
	pluginEventProcessingDuration.WithLabelValues(eventType, pluginName).Observe(durationSeconds)
}

// RecordEventError records an event processing error
func RecordEventError(eventType, pluginName, errorType string) {
	pluginEventErrorsTotal.WithLabelValues(eventType, pluginName, errorType).Inc()
}

// UpdateEventListeners updates the gauge of event listeners
func UpdateEventListeners(eventType string, count float64) {
	pluginEventListenersTotal.WithLabelValues(eventType).Set(count)
}

// GetEventMetricsSummary returns a summary of event metrics for observability
func GetEventMetricsSummary() map[string]interface{} {
	// Note: This function returns a basic summary
	// For detailed metrics, use the Prometheus /metrics endpoint
	return map[string]interface{}{
		"metrics_enabled":  true,
		"metrics_endpoint": "/metrics",
		"available_metrics": []string{
			"proxynd_plugin_events_total",
			"proxynd_plugin_event_processing_duration_seconds",
			"proxynd_plugin_event_errors_total",
			"proxynd_plugin_event_listeners_total",
		},
	}
}
