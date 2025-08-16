package opentelemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"proxynd/internal/ports"
)

// TraceService implements ports.TraceService using OpenTelemetry
type TraceService struct {
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator
	enabled    bool
}

// TraceSpan implements ports.TraceSpan using OpenTelemetry
type TraceSpan struct {
	span trace.Span
}

// TraceContext implements ports.TraceContext using OpenTelemetry
type TraceContext struct {
	spanContext trace.SpanContext
	baggage     map[string]string
}

// NewTraceService creates a new OpenTelemetry trace service
func NewTraceService(config *TracingConfig) ports.TraceService {
	if config == nil {
		config = DefaultTracingConfig()
	}

	tracer := otel.Tracer(config.ServiceName)
	propagator := otel.GetTextMapPropagator()

	return &TraceService{
		tracer:     tracer,
		propagator: propagator,
		enabled:    config.Enabled,
	}
}

// StartSpan starts a new trace span
func (t *TraceService) StartSpan(ctx context.Context, operationName string) (ports.TraceSpan, context.Context) {
	if !t.enabled {
		return &TraceSpan{span: trace.SpanFromContext(ctx)}, ctx
	}

	ctx, span := t.tracer.Start(ctx, operationName)
	return &TraceSpan{span: span}, ctx
}

// StartChildSpan starts a child span
func (t *TraceService) StartChildSpan(ctx context.Context, parent ports.TraceSpan, operationName string) (ports.TraceSpan, context.Context) {
	if !t.enabled {
		return &TraceSpan{span: trace.SpanFromContext(ctx)}, ctx
	}

	// The context should already contain the parent span
	ctx, span := t.tracer.Start(ctx, operationName)
	return &TraceSpan{span: span}, ctx
}

// InjectHeaders injects trace headers into HTTP request
func (t *TraceService) InjectHeaders(span ports.TraceSpan, headers map[string]string) {
	if !t.enabled {
		return
	}

	ctx := trace.ContextWithSpan(context.Background(), span.(*TraceSpan).span)
	carrier := propagation.MapCarrier(headers)
	t.propagator.Inject(ctx, carrier)
}

// ExtractHeaders extracts trace context from HTTP headers
func (t *TraceService) ExtractHeaders(headers map[string]string) (ports.TraceContext, error) {
	if !t.enabled {
		return &TraceContext{}, nil
	}

	carrier := propagation.MapCarrier(headers)
	ctx := t.propagator.Extract(context.Background(), carrier)
	spanContext := trace.SpanContextFromContext(ctx)

	return &TraceContext{
		spanContext: spanContext,
		baggage:     make(map[string]string),
	}, nil
}

// IsEnabled returns whether tracing is enabled
func (t *TraceService) IsEnabled() bool {
	return t.enabled
}

// SetTag sets a tag on the span
func (s *TraceSpan) SetTag(key string, value interface{}) {
	if s.span == nil {
		return
	}

	attr := convertToAttribute(key, value)
	s.span.SetAttributes(attr)
}

// SetError marks the span as having an error
func (s *TraceSpan) SetError(err error) {
	if s.span == nil {
		return
	}

	s.span.SetStatus(codes.Error, err.Error())
	s.span.RecordError(err)
}

// LogEvent logs an event on the span
func (s *TraceSpan) LogEvent(event string, fields map[string]interface{}) {
	if s.span == nil {
		return
	}

	attrs := make([]attribute.KeyValue, 0, len(fields))
	for key, value := range fields {
		attrs = append(attrs, convertToAttribute(key, value))
	}

	s.span.AddEvent(event, trace.WithAttributes(attrs...))
}

// Finish finishes the span
func (s *TraceSpan) Finish() {
	if s.span != nil {
		s.span.End()
	}
}

// Context returns the span context
func (s *TraceSpan) Context() ports.TraceContext {
	if s.span == nil {
		return &TraceContext{}
	}

	return &TraceContext{
		spanContext: s.span.SpanContext(),
		baggage:     make(map[string]string),
	}
}

// TraceID returns the trace ID
func (tc *TraceContext) TraceID() string {
	if !tc.spanContext.IsValid() {
		return ""
	}
	return tc.spanContext.TraceID().String()
}

// SpanID returns the span ID
func (tc *TraceContext) SpanID() string {
	if !tc.spanContext.IsValid() {
		return ""
	}
	return tc.spanContext.SpanID().String()
}

// IsSampled returns whether the trace is sampled
func (tc *TraceContext) IsSampled() bool {
	if !tc.spanContext.IsValid() {
		return false
	}
	return tc.spanContext.IsSampled()
}

// BaggageItem returns a baggage item
func (tc *TraceContext) BaggageItem(key string) string {
	if tc.baggage == nil {
		return ""
	}
	return tc.baggage[key]
}

// SetBaggageItem sets a baggage item
func (tc *TraceContext) SetBaggageItem(key, value string) {
	if tc.baggage == nil {
		tc.baggage = make(map[string]string)
	}
	tc.baggage[key] = value
}

// convertToAttribute converts a value to OpenTelemetry attribute
func convertToAttribute(key string, value interface{}) attribute.KeyValue {
	switch v := value.(type) {
	case string:
		return attribute.String(key, v)
	case int:
		return attribute.Int(key, v)
	case int64:
		return attribute.Int64(key, v)
	case float64:
		return attribute.Float64(key, v)
	case bool:
		return attribute.Bool(key, v)
	default:
		return attribute.String(key, fmt.Sprintf("%v", v))
	}
}

// TracingConfig defines OpenTelemetry tracing configuration
type TracingConfig struct {
	Enabled     bool              `json:"enabled" yaml:"enabled"`
	ServiceName string            `json:"service_name" yaml:"service_name"`
	SampleRate  float64           `json:"sample_rate" yaml:"sample_rate"`
	Endpoint    string            `json:"endpoint" yaml:"endpoint"`
	Headers     map[string]string `json:"headers" yaml:"headers"`
}

// DefaultTracingConfig returns default tracing configuration
func DefaultTracingConfig() *TracingConfig {
	return &TracingConfig{
		Enabled:     false,
		ServiceName: "proxynd",
		SampleRate:  0.1, // 10% sampling
		Endpoint:    "",
		Headers:     make(map[string]string),
	}
}

// DevelopmentTracingConfig returns development tracing configuration
func DevelopmentTracingConfig() *TracingConfig {
	return &TracingConfig{
		Enabled:     true,
		ServiceName: "proxynd-dev",
		SampleRate:  1.0, // 100% sampling for development
		Endpoint:    "http://localhost:14268/api/traces",
		Headers:     make(map[string]string),
	}
}
