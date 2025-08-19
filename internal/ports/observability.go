package ports

import (
	"context"
	"time"
)

// Logger defines logging interface
// TODO: Migrate from logging/ package and replace direct zap/logrus usage
type Logger interface {
	// Debug logs debug level message
	Debug(ctx context.Context, msg string, fields ...Field)

	// Info logs info level message
	Info(ctx context.Context, msg string, fields ...Field)

	// Warn logs warning level message
	Warn(ctx context.Context, msg string, fields ...Field)

	// Error logs error level message
	Error(ctx context.Context, msg string, fields ...Field)

	// Fatal logs fatal level message and exits
	Fatal(ctx context.Context, msg string, fields ...Field)

	// With creates logger with additional fields
	With(fields ...Field) Logger

	// WithContext creates logger with context
	WithContext(ctx context.Context) Logger
}

// Field represents a log field
type Field interface {
	Key() string
	Value() interface{}
}

// MetricsCollector defines metrics collection interface
// TODO: Migrate from metrics/ package
type MetricsCollector interface {
	// Counter operations
	IncCounter(name string, labels map[string]string)
	AddCounter(name string, value float64, labels map[string]string)

	// Gauge operations
	SetGauge(name string, value float64, labels map[string]string)
	AddGauge(name string, value float64, labels map[string]string)

	// Histogram operations
	ObserveHistogram(name string, value float64, labels map[string]string)

	// Summary operations
	ObserveSummary(name string, value float64, labels map[string]string)

	// Timer operations
	StartTimer(name string, labels map[string]string) Timer

	// Get metrics
	GetMetrics() (*MetricsSnapshot, error)
}

// Timer represents a metrics timer
type Timer interface {
	// Stop stops the timer and records the duration
	Stop()

	// Duration returns elapsed duration
	Duration() time.Duration
}

// MetricsSnapshot represents metrics at a point in time
type MetricsSnapshot struct {
	Timestamp  time.Time             `json:"timestamp"`
	Counters   map[string]float64    `json:"counters"`
	Gauges     map[string]float64    `json:"gauges"`
	Histograms map[string]*Histogram `json:"histograms"`
	Summaries  map[string]*Summary   `json:"summaries"`
}

// Histogram represents histogram metrics
type Histogram struct {
	Count   uint64             `json:"count"`
	Sum     float64            `json:"sum"`
	Buckets map[float64]uint64 `json:"buckets"`
}

// Summary represents summary metrics
type Summary struct {
	Count     uint64              `json:"count"`
	Sum       float64             `json:"sum"`
	Quantiles map[float64]float64 `json:"quantiles"`
}

// Tracer defines distributed tracing interface
// TODO: Add tracing support for request flows
type Tracer interface {
	// StartSpan starts a new span
	StartSpan(ctx context.Context, operationName string, opts ...SpanOption) (Span, context.Context)

	// Extract extracts span context from carrier
	Extract(carrier interface{}) (SpanContext, error)

	// Inject injects span context into carrier
	Inject(spanCtx SpanContext, carrier interface{}) error

	// Close closes the tracer
	Close() error
}

// Span represents a trace span
type Span interface {
	// SetTag sets a tag on the span
	SetTag(key string, value interface{}) Span

	// LogEvent logs an event
	LogEvent(event string, fields ...Field) Span

	// SetError marks span as error
	SetError(err error) Span

	// Finish finishes the span
	Finish()

	// Context returns span context
	Context() SpanContext
}

// SpanContext represents span context
type SpanContext interface {
	// TraceID returns trace ID
	TraceID() string

	// SpanID returns span ID
	SpanID() string

	// IsSampled returns if span is sampled
	IsSampled() bool
}

// SpanOption configures span creation
type SpanOption interface {
	Apply(*SpanConfig)
}

// SpanConfig configures span
type SpanConfig struct {
	Tags       map[string]interface{}
	References []SpanReference
	StartTime  time.Time
}

// SpanReference represents span reference
type SpanReference struct {
	Type              string
	ReferencedContext SpanContext
}

// HealthChecker defines health checking interface
// TODO: Migrate from health/ package
type HealthChecker interface {
	// Check performs health check
	Check(ctx context.Context) *HealthResult

	// GetName returns checker name
	GetName() string

	// GetDescription returns checker description
	GetDescription() string
}

// HealthService aggregates health checks
// TODO: Migrate from health/enhanced_health_service.go
type HealthService interface {
	// RegisterChecker registers a health checker
	RegisterChecker(checker HealthChecker)

	// UnregisterChecker unregisters a health checker
	UnregisterChecker(name string)

	// CheckHealth runs all health checks
	CheckHealth(ctx context.Context) *OverallHealthResult

	// CheckComponent checks specific component
	CheckComponent(ctx context.Context, name string) *HealthResult

	// GetRegisteredCheckers returns registered checkers
	GetRegisteredCheckers() []string
}

// HealthResult represents health check result
type HealthResult struct {
	Name      string                    `json:"name"`
	Status    ObservabilityHealthStatus `json:"status"`
	Message   string                    `json:"message"`
	Details   map[string]interface{}    `json:"details"`
	Duration  time.Duration             `json:"duration"`
	CheckedAt time.Time                 `json:"checked_at"`
	Error     error                     `json:"error,omitempty"`
}

// OverallHealthResult represents overall health
type OverallHealthResult struct {
	Status    ObservabilityHealthStatus `json:"status"`
	Checks    []*HealthResult           `json:"checks"`
	Duration  time.Duration             `json:"duration"`
	CheckedAt time.Time                 `json:"checked_at"`
	Summary   *HealthSummary            `json:"summary"`
}

// HealthSummary provides health summary
type HealthSummary struct {
	Total     int `json:"total"`
	Healthy   int `json:"healthy"`
	Degraded  int `json:"degraded"`
	Unhealthy int `json:"unhealthy"`
}

// ObservabilityHealthStatus represents health status for observability
type ObservabilityHealthStatus string

const (
	ObservabilityHealthStatusHealthy   ObservabilityHealthStatus = "healthy"
	ObservabilityHealthStatusDegraded  ObservabilityHealthStatus = "degraded"
	ObservabilityHealthStatusUnhealthy ObservabilityHealthStatus = "unhealthy"
)

// AuditLogger defines audit logging interface
// TODO: Create audit logging for security events
type AuditLogger interface {
	// LogEvent logs audit event
	LogEvent(ctx context.Context, event *AuditEvent)

	// GetEvents retrieves audit events
	GetEvents(ctx context.Context, req *AuditQueryRequest) (*AuditQueryResponse, error)
}

// AuditEvent represents audit event
type AuditEvent struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	UserID    string                 `json:"user_id"`
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource"`
	Result    string                 `json:"result"` // success, failure
	IPAddress string                 `json:"ip_address"`
	UserAgent string                 `json:"user_agent"`
	Details   map[string]interface{} `json:"details"`
}

// AuditQueryRequest represents audit query request
type AuditQueryRequest struct {
	UserID    string    `json:"user_id,omitempty"`
	Action    string    `json:"action,omitempty"`
	Resource  string    `json:"resource,omitempty"`
	StartTime time.Time `json:"start_time,omitempty"`
	EndTime   time.Time `json:"end_time,omitempty"`
	Page      int       `json:"page"`
	PageSize  int       `json:"page_size"`
}

// AuditQueryResponse represents audit query response
type AuditQueryResponse struct {
	Events   []*AuditEvent `json:"events"`
	Total    int           `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

// PerformanceMonitor monitors performance metrics
// TODO: Migrate from performance/ package
type PerformanceMonitor interface {
	// RecordLatency records operation latency
	RecordLatency(operation string, duration time.Duration, labels map[string]string)

	// RecordThroughput records operation throughput
	RecordThroughput(operation string, count int64, labels map[string]string)

	// RecordError records error occurrence
	RecordError(operation, error string, labels map[string]string)

	// GetPerformanceReport generates performance report
	GetPerformanceReport(ctx context.Context, req *PerformanceReportRequest) (*PerformanceReport, error)
}

// PerformanceReportRequest represents performance report request
type PerformanceReportRequest struct {
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Operations  []string  `json:"operations"`
	Granularity string    `json:"granularity"` // minute, hour, day
}

// PerformanceReport represents performance report
type PerformanceReport struct {
	StartTime      time.Time                     `json:"start_time"`
	EndTime        time.Time                     `json:"end_time"`
	Operations     map[string]*OperationMetrics  `json:"operations"`
	Summary        *PerformanceSummary           `json:"summary"`
	TimeSeriesData map[string][]*TimeSeriesPoint `json:"time_series_data"`
}

// OperationMetrics represents metrics for an operation
type OperationMetrics struct {
	Operation       string        `json:"operation"`
	TotalRequests   int64         `json:"total_requests"`
	SuccessRequests int64         `json:"success_requests"`
	ErrorRequests   int64         `json:"error_requests"`
	AvgLatency      time.Duration `json:"avg_latency"`
	P50Latency      time.Duration `json:"p50_latency"`
	P95Latency      time.Duration `json:"p95_latency"`
	P99Latency      time.Duration `json:"p99_latency"`
	Throughput      float64       `json:"throughput"` // requests per second
}

// PerformanceSummary provides overall performance summary
type PerformanceSummary struct {
	TotalRequests   int64         `json:"total_requests"`
	SuccessRate     float64       `json:"success_rate"`
	AvgLatency      time.Duration `json:"avg_latency"`
	TotalThroughput float64       `json:"total_throughput"`
}

// TimeSeriesPoint represents a point in time series data
type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}
