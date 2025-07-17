package logging

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// PerformanceLogger handles performance-related logging
type PerformanceLogger struct {
	logger    Logger
	component string
	metrics   *PerformanceMetrics
}

// PerformanceMetrics tracks performance statistics
type PerformanceMetrics struct {
	mu                 sync.RWMutex
	RequestCount       int64
	TotalDuration      time.Duration
	MinDuration        time.Duration
	MaxDuration        time.Duration
	SlowRequestCount   int64
	ErrorCount         int64
	MemoryAllocations  int64
	GCCount            uint32
	LastResetTime      time.Time
}

// PerformanceEvent represents a performance-related event
type PerformanceEvent struct {
	Type          string                 `json:"type"`
	Component     string                 `json:"component"`
	Operation     string                 `json:"operation"`
	Duration      time.Duration          `json:"duration"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       time.Time              `json:"end_time"`
	MemoryBefore  uint64                 `json:"memory_before"`
	MemoryAfter   uint64                 `json:"memory_after"`
	Success       bool                   `json:"success"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	RequestID     string                 `json:"request_id,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
}

// Performance event types
const (
	PerfEventRequest      = "http_request"
	PerfEventDatabase     = "database_query"
	PerfEventCache        = "cache_operation"
	PerfEventFileIO       = "file_io"
	PerfEventNetworkCall  = "network_call"
	PerfEventComputation  = "computation"
	PerfEventMemoryGC     = "garbage_collection"
)

// NewPerformanceLogger creates a new performance logger
func NewPerformanceLogger(logger Logger) *PerformanceLogger {
	return &PerformanceLogger{
		logger:    logger.WithComponent("performance"),
		component: "performance",
		metrics: &PerformanceMetrics{
			LastResetTime: time.Now(),
		},
	}
}

// LogPerformanceEvent logs a performance event
func (p *PerformanceLogger) LogPerformanceEvent(ctx context.Context, event PerformanceEvent) {
	// Extract context information
	if event.RequestID == "" {
		event.RequestID = GetRequestID(ctx)
	}
	if event.CorrelationID == "" {
		event.CorrelationID = GetCorrelationID(ctx)
	}

	fields := []Field{
		String("perf_event_type", event.Type),
		String("component", event.Component),
		String("operation", event.Operation),
		Duration("duration", event.Duration),
		Bool("success", event.Success),
	}

	// Add timing information
	if !event.StartTime.IsZero() {
		fields = append(fields, NewField("start_time", event.StartTime))
	}
	if !event.EndTime.IsZero() {
		fields = append(fields, NewField("end_time", event.EndTime))
	}

	// Add memory information
	if event.MemoryBefore > 0 {
		fields = append(fields, NewField("memory_before", event.MemoryBefore))
	}
	if event.MemoryAfter > 0 {
		fields = append(fields, NewField("memory_after", event.MemoryAfter))
		if event.MemoryBefore > 0 {
			memoryDiff := int64(event.MemoryAfter) - int64(event.MemoryBefore)
			fields = append(fields, NewField("memory_diff", memoryDiff))
		}
	}

	// Add error information
	if event.ErrorMessage != "" {
		fields = append(fields, String("error_message", event.ErrorMessage))
	}

	// Add metadata
	if event.Metadata != nil {
		fields = append(fields, NewField("metadata", event.Metadata))
	}

	// Update metrics
	p.updateMetrics(event)

	// Log based on performance characteristics
	logger := p.logger.WithContext(ctx)
	message := "Performance event: " + event.Operation

	// Determine log level based on performance
	slowThreshold := 2 * time.Second
	verySlowThreshold := 5 * time.Second

	if !event.Success {
		logger.Error(message, fields...)
	} else if event.Duration > verySlowThreshold {
		logger.Error(message+" (very slow)", fields...)
	} else if event.Duration > slowThreshold {
		logger.Warn(message+" (slow)", fields...)
	} else {
		logger.Info(message, fields...)
	}
}

// updateMetrics updates internal performance metrics
func (p *PerformanceLogger) updateMetrics(event PerformanceEvent) {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()

	p.metrics.RequestCount++
	p.metrics.TotalDuration += event.Duration

	// Update min/max duration
	if p.metrics.MinDuration == 0 || event.Duration < p.metrics.MinDuration {
		p.metrics.MinDuration = event.Duration
	}
	if event.Duration > p.metrics.MaxDuration {
		p.metrics.MaxDuration = event.Duration
	}

	// Count slow requests (>2s)
	if event.Duration > 2*time.Second {
		p.metrics.SlowRequestCount++
	}

	// Count errors
	if !event.Success {
		p.metrics.ErrorCount++
	}

	// Update memory stats
	if event.MemoryAfter > event.MemoryBefore {
		p.metrics.MemoryAllocations += int64(event.MemoryAfter - event.MemoryBefore)
	}
}

// GetMetrics returns current performance metrics
func (p *PerformanceLogger) GetMetrics() PerformanceMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()

	// Get current GC stats
	var gcStats runtime.MemStats
	runtime.ReadMemStats(&gcStats)
	p.metrics.GCCount = gcStats.NumGC

	return *p.metrics
}

// ResetMetrics resets performance metrics
func (p *PerformanceLogger) ResetMetrics() {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()

	p.metrics.RequestCount = 0
	p.metrics.TotalDuration = 0
	p.metrics.MinDuration = 0
	p.metrics.MaxDuration = 0
	p.metrics.SlowRequestCount = 0
	p.metrics.ErrorCount = 0
	p.metrics.MemoryAllocations = 0
	p.metrics.LastResetTime = time.Now()
}

// PerformanceTracker helps track performance of operations
type PerformanceTracker struct {
	logger        *PerformanceLogger
	eventType     string
	component     string
	operation     string
	startTime     time.Time
	memoryBefore  uint64
	metadata      map[string]interface{}
	ctx           context.Context
}

// StartOperation starts tracking an operation
func (p *PerformanceLogger) StartOperation(ctx context.Context, eventType, component, operation string) *PerformanceTracker {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return &PerformanceTracker{
		logger:       p,
		eventType:    eventType,
		component:    component,
		operation:    operation,
		startTime:    time.Now(),
		memoryBefore: memStats.Alloc,
		metadata:     make(map[string]interface{}),
		ctx:          ctx,
	}
}

// AddMetadata adds metadata to the performance tracker
func (pt *PerformanceTracker) AddMetadata(key string, value interface{}) {
	if pt.metadata == nil {
		pt.metadata = make(map[string]interface{})
	}
	pt.metadata[key] = value
}

// Finish completes the operation tracking
func (pt *PerformanceTracker) Finish(success bool, errorMessage string) {
	endTime := time.Now()
	duration := endTime.Sub(pt.startTime)

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	event := PerformanceEvent{
		Type:          pt.eventType,
		Component:     pt.component,
		Operation:     pt.operation,
		Duration:      duration,
		StartTime:     pt.startTime,
		EndTime:       endTime,
		MemoryBefore:  pt.memoryBefore,
		MemoryAfter:   memStats.Alloc,
		Success:       success,
		ErrorMessage:  errorMessage,
		Metadata:      pt.metadata,
	}

	pt.logger.LogPerformanceEvent(pt.ctx, event)
}

// PerformanceMiddleware creates middleware for automatic performance tracking
func PerformanceMiddleware(perfLogger *PerformanceLogger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Start tracking
		tracker := perfLogger.StartOperation(
			c.UserContext(),
			PerfEventRequest,
			"http.middleware",
			c.Method()+" "+c.Path(),
		)

		// Add request metadata
		tracker.AddMetadata("method", c.Method())
		tracker.AddMetadata("path", c.Path())
		tracker.AddMetadata("route", c.Route().Path)
		tracker.AddMetadata("request_size", len(c.Body()))
		tracker.AddMetadata("ip", c.IP())

		// Process request
		err := c.Next()

		// Complete tracking
		success := err == nil && c.Response().StatusCode() < 400
		errorMessage := ""
		if err != nil {
			errorMessage = err.Error()
		}

		// Add response metadata
		tracker.AddMetadata("status_code", c.Response().StatusCode())
		tracker.AddMetadata("response_size", len(c.Response().Body()))

		tracker.Finish(success, errorMessage)

		return err
	}
}

// Convenience methods for common operations

// TrackDatabaseQuery tracks database query performance
func (p *PerformanceLogger) TrackDatabaseQuery(ctx context.Context, query string, args []interface{}) *PerformanceTracker {
	tracker := p.StartOperation(ctx, PerfEventDatabase, "database", "query")
	tracker.AddMetadata("query", query)
	tracker.AddMetadata("args_count", len(args))
	return tracker
}

// TrackCacheOperation tracks cache operation performance
func (p *PerformanceLogger) TrackCacheOperation(ctx context.Context, operation, key string) *PerformanceTracker {
	tracker := p.StartOperation(ctx, PerfEventCache, "cache", operation)
	tracker.AddMetadata("key", key)
	return tracker
}

// TrackFileOperation tracks file I/O performance
func (p *PerformanceLogger) TrackFileOperation(ctx context.Context, operation, filePath string) *PerformanceTracker {
	tracker := p.StartOperation(ctx, PerfEventFileIO, "filesystem", operation)
	tracker.AddMetadata("file_path", filePath)
	return tracker
}

// TrackNetworkCall tracks network call performance
func (p *PerformanceLogger) TrackNetworkCall(ctx context.Context, method, url string) *PerformanceTracker {
	tracker := p.StartOperation(ctx, PerfEventNetworkCall, "network", method+" "+url)
	tracker.AddMetadata("method", method)
	tracker.AddMetadata("url", url)
	return tracker
}

// LogSlowQuery logs slow database queries
func (p *PerformanceLogger) LogSlowQuery(ctx context.Context, query string, duration time.Duration, threshold time.Duration) {
	if duration > threshold {
		event := PerformanceEvent{
			Type:      PerfEventDatabase,
			Component: "database",
			Operation: "slow_query",
			Duration:  duration,
			Success:   true,
			Metadata: map[string]interface{}{
				"query":     query,
				"threshold": threshold,
			},
		}
		p.LogPerformanceEvent(ctx, event)
	}
}

// LogMemoryUsage logs current memory usage
func (p *PerformanceLogger) LogMemoryUsage(ctx context.Context) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	fields := []Field{
		NewField("heap_alloc", memStats.HeapAlloc),
		NewField("heap_sys", memStats.HeapSys),
		NewField("heap_idle", memStats.HeapIdle),
		NewField("heap_inuse", memStats.HeapInuse),
		NewField("stack_inuse", memStats.StackInuse),
		NewField("stack_sys", memStats.StackSys),
		NewField("num_gc", memStats.NumGC),
		NewField("gc_cpu_fraction", memStats.GCCPUFraction),
	}

	p.logger.WithContext(ctx).Info("Memory usage snapshot", fields...)
}

// StartPeriodicMemoryLogging starts periodic memory usage logging
func (p *PerformanceLogger) StartPeriodicMemoryLogging(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				p.LogMemoryUsage(ctx)
			}
		}
	}()
}

// LogPerformanceMetrics logs current performance metrics summary
func (p *PerformanceLogger) LogPerformanceMetrics(ctx context.Context) {
	metrics := p.GetMetrics()
	
	fields := []Field{
		NewField("request_count", metrics.RequestCount),
		NewField("total_duration", metrics.TotalDuration),
		NewField("min_duration", metrics.MinDuration),
		NewField("max_duration", metrics.MaxDuration),
		NewField("slow_request_count", metrics.SlowRequestCount),
		NewField("error_count", metrics.ErrorCount),
		NewField("memory_allocations", metrics.MemoryAllocations),
		NewField("gc_count", metrics.GCCount),
		NewField("last_reset_time", metrics.LastResetTime),
	}

	// Calculate averages
	if metrics.RequestCount > 0 {
		avgDuration := metrics.TotalDuration / time.Duration(metrics.RequestCount)
		fields = append(fields, Duration("avg_duration", avgDuration))
		
		errorRate := float64(metrics.ErrorCount) / float64(metrics.RequestCount)
		fields = append(fields, Float64("error_rate", errorRate))
		
		slowRequestRate := float64(metrics.SlowRequestCount) / float64(metrics.RequestCount)
		fields = append(fields, Float64("slow_request_rate", slowRequestRate))
	}

	p.logger.WithContext(ctx).Info("Performance metrics summary", fields...)
}