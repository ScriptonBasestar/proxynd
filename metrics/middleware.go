package metrics

import (
	"crypto/rand"
	"encoding/hex"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/logging"
)

// Constants for metrics middleware
const (
	metricsPath   = "/metrics"
	resultFailure = "failure"
	statusUnknown = "unknown"

	// Tracing constants
	traceIDHeader    = "X-Trace-ID"
	spanIDHeader     = "X-Span-ID"
	parentSpanHeader = "X-Parent-Span-ID"
	tracingEnabled   = "X-Tracing-Enabled"

	// Performance thresholds
	slowRequestThreshold     = 2 * time.Second
	criticalRequestThreshold = 5 * time.Second
)

// RequestTrace contains detailed tracing information for a request
type RequestTrace struct {
	TraceID          string              `json:"trace_id"`
	SpanID           string              `json:"span_id"`
	ParentSpanID     string              `json:"parent_span_id,omitempty"`
	StartTime        time.Time           `json:"start_time"`
	EndTime          time.Time           `json:"end_time"`
	Duration         time.Duration       `json:"duration"`
	Method           string              `json:"method"`
	Path             string              `json:"path"`
	StatusCode       int                 `json:"status_code"`
	UserAgent        string              `json:"user_agent"`
	RemoteIP         string              `json:"remote_ip"`
	RequestSize      int64               `json:"request_size"`
	ResponseSize     int64               `json:"response_size"`
	HandlerType      string              `json:"handler_type"`
	CacheHit         bool                `json:"cache_hit"`
	UpstreamHost     string              `json:"upstream_host,omitempty"`
	UpstreamDuration time.Duration       `json:"upstream_duration,omitempty"`
	Annotations      map[string]string   `json:"annotations,omitempty"`
	Tags             map[string]string   `json:"tags,omitempty"`
	Error            string              `json:"error,omitempty"`
	Performance      *PerformanceMetrics `json:"performance,omitempty"`
}

// PerformanceMetrics contains detailed performance information
type PerformanceMetrics struct {
	MemoryBefore     uint64        `json:"memory_before"`
	MemoryAfter      uint64        `json:"memory_after"`
	MemoryAllocated  uint64        `json:"memory_allocated"`
	GoroutinesBefore int           `json:"goroutines_before"`
	GoroutinesAfter  int           `json:"goroutines_after"`
	CPUTimeBefore    time.Duration `json:"cpu_time_before"`
	CPUTimeAfter     time.Duration `json:"cpu_time_after"`
	DBConnections    int           `json:"db_connections,omitempty"`
	CacheOperations  int           `json:"cache_operations"`
	NetworkCalls     int           `json:"network_calls"`
}

// TraceCollector manages request traces and performance data
type TraceCollector struct {
	logger            logging.Logger
	advancedCollector *AdvancedCollector
	traces            map[string]*RequestTrace
	tracesMu          sync.RWMutex
	maxTraces         int
	traceTTL          time.Duration
	enableProfiling   bool
	enableSampling    bool
	samplingRate      float64
	slowTraces        []*RequestTrace
	slowTracesMu      sync.RWMutex
	maxSlowTraces     int
}

// TracingConfig contains configuration for advanced tracing
type TracingConfig struct {
	Enabled         bool          `yaml:"enabled" json:"enabled"`
	MaxTraces       int           `yaml:"max_traces" json:"max_traces"`
	TraceTTL        time.Duration `yaml:"trace_ttl" json:"trace_ttl"`
	EnableProfiling bool          `yaml:"enable_profiling" json:"enable_profiling"`
	EnableSampling  bool          `yaml:"enable_sampling" json:"enable_sampling"`
	SamplingRate    float64       `yaml:"sampling_rate" json:"sampling_rate"`
	MaxSlowTraces   int           `yaml:"max_slow_traces" json:"max_slow_traces"`
}

// Global trace collector instance
var globalTraceCollector *TraceCollector

// InitTraceCollector initializes the global trace collector
func InitTraceCollector(logger logging.Logger, config *TracingConfig) {
	if config == nil {
		config = &TracingConfig{
			Enabled:         true,
			MaxTraces:       10000,
			TraceTTL:        time.Hour,
			EnableProfiling: false,
			EnableSampling:  true,
			SamplingRate:    0.1,
			MaxSlowTraces:   1000,
		}
	}

	globalTraceCollector = &TraceCollector{
		logger:            logger.WithField("component", "metrics.trace_collector"),
		advancedCollector: NewAdvancedCollector(logger),
		traces:            make(map[string]*RequestTrace),
		maxTraces:         config.MaxTraces,
		traceTTL:          config.TraceTTL,
		enableProfiling:   config.EnableProfiling,
		enableSampling:    config.EnableSampling,
		samplingRate:      config.SamplingRate,
		slowTraces:        make([]*RequestTrace, 0, config.MaxSlowTraces),
		maxSlowTraces:     config.MaxSlowTraces,
	}

	// Start cleanup goroutine
	go globalTraceCollector.cleanupTraces()

	// Start advanced collector
	if err := globalTraceCollector.advancedCollector.Start(); err != nil {
		logger.Error("Failed to start advanced collector", logging.ErrorField(err))
	}
}

// GetTraceCollector returns the global trace collector
func GetTraceCollector() *TraceCollector {
	return globalTraceCollector
}

// PrometheusMiddleware Prometheus 메트릭 수집 미들웨어 (enhanced with tracing)
func PrometheusMiddleware() fiber.Handler {
	metrics := GetMetrics()

	return func(c *fiber.Ctx) error {
		// /metrics 엔드포인트는 제외
		if c.Path() == metricsPath {
			return c.Next()
		}

		// Initialize tracing for this request
		var trace *RequestTrace
		if globalTraceCollector != nil && globalTraceCollector.shouldTrace() {
			trace = globalTraceCollector.startTrace(c)
			c.Locals("trace", trace)
		}

		// 활성 요청 수 증가
		metrics.HTTPActiveRequests.Inc()
		defer metrics.HTTPActiveRequests.Dec()

		// 시작 시간 기록 및 성능 메트릭 수집
		start := time.Now()
		var perfMetrics *PerformanceMetrics
		if trace != nil && globalTraceCollector.enableProfiling {
			perfMetrics = capturePerformanceMetricsBefore()
			trace.Performance = perfMetrics
		}

		// 요청 크기 기록
		if c.Request().Header.ContentLength() > 0 {
			size := float64(c.Request().Header.ContentLength())
			registryType := extractRegistryType(c)
			metrics.HTTPRequestSize.WithLabelValues(
				c.Method(),
				normalizePath(c.Path()),
				registryType,
			).Observe(size)
		}

		// 다음 핸들러 실행
		err := c.Next()

		// 응답 처리 및 트레이싱 완료
		duration := time.Since(start)
		status := strconv.Itoa(c.Response().StatusCode())
		path := normalizePath(c.Path())
		registryType := extractRegistryType(c)

		// Complete performance metrics collection
		if trace != nil && perfMetrics != nil {
			capturePerformanceMetricsAfter(perfMetrics)
			trace.EndTime = time.Now()
			trace.Duration = duration
			trace.StatusCode = c.Response().StatusCode()
			trace.ResponseSize = int64(len(c.Response().Body()))

			if err != nil {
				trace.Error = err.Error()
			}

			// Record latency for percentile calculations
			if globalTraceCollector.advancedCollector != nil {
				globalTraceCollector.advancedCollector.RecordLatency(registryType, duration.Seconds())
			}

			// Store trace and check if it's slow
			globalTraceCollector.finishTrace(trace)
		}

		// HTTP 메트릭 기록
		metrics.HTTPRequestsTotal.WithLabelValues(
			c.Method(),
			path,
			status,
			registryType,
		).Inc()

		metrics.HTTPRequestDuration.WithLabelValues(
			c.Method(),
			path,
			status,
			registryType,
		).Observe(duration.Seconds())

		// 응답 크기 기록
		if size := len(c.Response().Body()); size > 0 {
			metrics.HTTPResponseSize.WithLabelValues(
				c.Method(),
				path,
				status,
				registryType,
			).Observe(float64(size))
		}

		// 캐시 메트릭 업데이트
		updateCacheMetrics(c, metrics, registryType)

		// 프록시 메트릭 업데이트
		updateProxyMetrics(c, metrics, registryType)

		// 인증 메트릭 업데이트
		updateAuthMetrics(c, metrics)

		// 검증 메트릭 업데이트
		updateVerificationMetrics(c, metrics, registryType)

		return err
	}
}

// updateCacheMetrics 캐시 관련 메트릭 업데이트
func updateCacheMetrics(c *fiber.Ctx, metrics *Metrics, registryType string) {
	// 캐시 히트/미스 확인
	cacheHit := c.Locals("cache_hit")
	cacheBackend := getCacheBackend(c)

	if cacheHit != nil {
		if hit, ok := cacheHit.(bool); ok {
			if hit {
				metrics.CacheHitsTotal.WithLabelValues(registryType, cacheBackend).Inc()

				// 대역폭 절약량 계산
				if size := len(c.Response().Body()); size > 0 {
					metrics.CacheBandwidthSaved.WithLabelValues(registryType).Add(float64(size))
				}
			} else {
				metrics.CacheMissesTotal.WithLabelValues(registryType, cacheBackend).Inc()
			}
		}
	}

	// 캐시 제거 이벤트
	if evicted := c.Locals("cache_evicted"); evicted != nil {
		if reason, ok := evicted.(string); ok {
			metrics.CacheEvictionsTotal.WithLabelValues(registryType, cacheBackend, reason).Inc()
		}
	}
}

// updateProxyMetrics 프록시 관련 메트릭 업데이트
func updateProxyMetrics(c *fiber.Ctx, metrics *Metrics, registryType string) {
	// 프록시 요청인지 확인
	if !strings.HasPrefix(c.Path(), "/proxy/") {
		return
	}

	upstream := getUpstream(c, registryType)

	// 프록시 요청 카운트
	metrics.ProxyRequestsTotal.WithLabelValues(
		registryType,
		upstream,
		c.Method(),
	).Inc()

	// 프록시 오류
	if c.Response().StatusCode() >= 400 {
		errorType := getErrorType(c.Response().StatusCode())
		metrics.ProxyErrorsTotal.WithLabelValues(
			registryType,
			upstream,
			errorType,
		).Inc()
	}

	// 업스트림 요청 시간
	if upstreamDuration := c.Locals("upstream_duration"); upstreamDuration != nil {
		if duration, ok := upstreamDuration.(time.Duration); ok {
			metrics.ProxyUpstreamDuration.WithLabelValues(
				registryType,
				upstream,
			).Observe(duration.Seconds())
		}
	}

	// 전송 바이트 수
	if c.Method() == "GET" || c.Method() == "HEAD" {
		// 다운로드
		if size := len(c.Response().Body()); size > 0 {
			metrics.ProxyBytesTransferred.WithLabelValues(
				registryType,
				"download",
			).Add(float64(size))
		}
	} else if c.Method() == "POST" || c.Method() == "PUT" {
		// 업로드
		if c.Request().Header.ContentLength() > 0 {
			metrics.ProxyBytesTransferred.WithLabelValues(
				registryType,
				"upload",
			).Add(float64(c.Request().Header.ContentLength()))
		}
	}
}

// updateAuthMetrics 인증 관련 메트릭 업데이트
func updateAuthMetrics(c *fiber.Ctx, metrics *Metrics) {
	// 인증 시도
	if authAttempt := c.Locals("auth_attempt"); authAttempt != nil {
		if method, ok := authAttempt.(string); ok {
			authResult := "success"
			if c.Response().StatusCode() == 401 || c.Response().StatusCode() == 403 {
				authResult = resultFailure
			}

			metrics.AuthAttemptsTotal.WithLabelValues(method, authResult).Inc()

			// 인증 실패
			if authResult == resultFailure {
				reason := "invalid_credentials"
				if c.Response().StatusCode() == 403 {
					reason = "forbidden"
				}
				metrics.AuthFailuresTotal.WithLabelValues(method, reason).Inc()
			}
		}
	}
}

// updateVerificationMetrics 검증 관련 메트릭 업데이트
func updateVerificationMetrics(c *fiber.Ctx, metrics *Metrics, registryType string) {
	// 패키지 검증 수행 여부
	if verified := c.Locals("packageVerified"); verified != nil {
		result := "success"
		if verifiedBool, ok := verified.(bool); ok && !verifiedBool {
			result = resultFailure
		}

		metrics.PackageVerifications.WithLabelValues(registryType, result).Inc()

		// 검증 실패 상세
		if result == resultFailure {
			if failureType := c.Locals("verificationFailureType"); failureType != nil {
				if ft, ok := failureType.(string); ok {
					metrics.VerificationFailures.WithLabelValues(registryType, ft).Inc()
				}
			}
		}

		// 검증 소요 시간
		if verificationDuration := c.Locals("verificationDuration"); verificationDuration != nil {
			if duration, ok := verificationDuration.(time.Duration); ok {
				metrics.VerificationDuration.WithLabelValues(registryType).Observe(duration.Seconds())
			}
		}
	}
}

// extractRegistryType 요청에서 레지스트리 타입 추출
func extractRegistryType(c *fiber.Ctx) string {
	// /proxy/:type/* 패턴에서 추출
	if strings.HasPrefix(c.Path(), "/proxy/") {
		parts := strings.Split(c.Path(), "/")
		if len(parts) >= 3 {
			return parts[2]
		}
	}

	// Locals에서 확인
	if registryType := c.Locals("registry_type"); registryType != nil {
		if rt, ok := registryType.(string); ok {
			return rt
		}
	}

	return statusUnknown
}

// normalizePath 경로 정규화 (카디널리티 감소)
func normalizePath(path string) string {
	// /proxy/:type/* 패턴 정규화
	if strings.HasPrefix(path, "/proxy/") {
		parts := strings.Split(path, "/")
		if len(parts) >= 3 {
			registryType := parts[2]
			return "/proxy/" + registryType + "/*"
		}
	}

	// 기타 공통 패턴
	switch {
	case path == "/":
		return "/"
	case path == "/healthz":
		return "/healthz"
	case path == "/metrics":
		return "/metrics"
	case strings.HasPrefix(path, "/api/"):
		return "/api/*"
	default:
		return "/other"
	}
}

// getCacheBackend 캐시 백엔드 타입 반환
func getCacheBackend(c *fiber.Ctx) string {
	if backend := c.Locals("cache_backend"); backend != nil {
		if b, ok := backend.(string); ok {
			return b
		}
	}
	return "file" // 기본값
}

// getUpstream 업스트림 서버 정보 반환
func getUpstream(c *fiber.Ctx, registryType string) string {
	if upstream := c.Locals("upstream"); upstream != nil {
		if u, ok := upstream.(string); ok {
			return u
		}
	}

	// 레지스트리 타입별 기본 업스트림
	switch registryType {
	case "npm":
		return "registry.npmjs.org"
	case "pypi":
		return "pypi.org"
	case "docker":
		return "registry-1.docker.io"
	case "maven":
		return "repo1.maven.org"
	case "apt":
		return "archive.ubuntu.com"
	default:
		return statusUnknown
	}
}

// getErrorType HTTP 상태 코드에서 오류 타입 추출
func getErrorType(statusCode int) string {
	switch {
	case statusCode >= 400 && statusCode < 500:
		switch statusCode {
		case 400:
			return "bad_request"
		case 401:
			return "unauthorized"
		case 403:
			return "forbidden"
		case 404:
			return "not_found"
		case 429:
			return "rate_limited"
		default:
			return "client_error"
		}
	case statusCode >= 500:
		switch statusCode {
		case 500:
			return "internal_error"
		case 502:
			return "bad_gateway"
		case 503:
			return "service_unavailable"
		case 504:
			return "gateway_timeout"
		default:
			return "server_error"
		}
	default:
		return statusUnknown
	}
}

// TraceCollector methods

// shouldTrace determines if this request should be traced based on sampling
func (tc *TraceCollector) shouldTrace() bool {
	if !tc.enableSampling {
		return true
	}

	// Simple random sampling
	buf := make([]byte, 1)
	rand.Read(buf)
	return float64(buf[0])/255.0 < tc.samplingRate
}

// startTrace initiates a new trace for the request
func (tc *TraceCollector) startTrace(c *fiber.Ctx) *RequestTrace {
	traceID := generateTraceID()
	spanID := generateSpanID()

	trace := &RequestTrace{
		TraceID:     traceID,
		SpanID:      spanID,
		StartTime:   time.Now(),
		Method:      c.Method(),
		Path:        c.Path(),
		UserAgent:   c.Get("User-Agent"),
		RemoteIP:    c.IP(),
		RequestSize: int64(c.Request().Header.ContentLength()),
		HandlerType: extractRegistryType(c),
		Annotations: make(map[string]string),
		Tags:        make(map[string]string),
	}

	// Check for parent span
	if parentSpan := c.Get(parentSpanHeader); parentSpan != "" {
		trace.ParentSpanID = parentSpan
	}

	// Set tracing headers in response
	c.Set(traceIDHeader, traceID)
	c.Set(spanIDHeader, spanID)

	return trace
}

// finishTrace completes the trace and stores it
func (tc *TraceCollector) finishTrace(trace *RequestTrace) {
	tc.tracesMu.Lock()
	defer tc.tracesMu.Unlock()

	// Store the trace
	if len(tc.traces) >= tc.maxTraces {
		// Remove oldest trace
		oldestTime := time.Now()
		var oldestID string
		for id, tr := range tc.traces {
			if tr.StartTime.Before(oldestTime) {
				oldestTime = tr.StartTime
				oldestID = id
			}
		}
		if oldestID != "" {
			delete(tc.traces, oldestID)
		}
	}

	tc.traces[trace.TraceID] = trace

	// Check if this is a slow trace
	if trace.Duration >= slowRequestThreshold {
		tc.addSlowTrace(trace)
	}

	// Log slow or critical requests
	if trace.Duration >= criticalRequestThreshold {
		tc.logger.Warn("Critical slow request detected",
			logging.F("trace_id", trace.TraceID),
			logging.Duration("duration", trace.Duration),
			logging.F("path", trace.Path),
			logging.F("status", trace.StatusCode))
	} else if trace.Duration >= slowRequestThreshold {
		tc.logger.Info("Slow request detected",
			logging.F("trace_id", trace.TraceID),
			logging.Duration("duration", trace.Duration),
			logging.F("path", trace.Path))
	}
}

// addSlowTrace adds a trace to the slow traces collection
func (tc *TraceCollector) addSlowTrace(trace *RequestTrace) {
	tc.slowTracesMu.Lock()
	defer tc.slowTracesMu.Unlock()

	if len(tc.slowTraces) >= tc.maxSlowTraces {
		// Remove oldest slow trace
		tc.slowTraces = tc.slowTraces[1:]
	}

	// Make a copy to avoid mutations
	slowTrace := &RequestTrace{}
	*slowTrace = *trace
	tc.slowTraces = append(tc.slowTraces, slowTrace)
}

// cleanupTraces periodically removes old traces
func (tc *TraceCollector) cleanupTraces() {
	ticker := time.NewTicker(time.Minute * 10) // Cleanup every 10 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tc.performCleanup()
		}
	}
}

// performCleanup removes traces older than TTL
func (tc *TraceCollector) performCleanup() {
	tc.tracesMu.Lock()
	defer tc.tracesMu.Unlock()

	cutoff := time.Now().Add(-tc.traceTTL)
	toDelete := make([]string, 0)

	for id, trace := range tc.traces {
		if trace.StartTime.Before(cutoff) {
			toDelete = append(toDelete, id)
		}
	}

	for _, id := range toDelete {
		delete(tc.traces, id)
	}

	if len(toDelete) > 0 {
		tc.logger.Debug("Cleaned up old traces", logging.F("count", len(toDelete)))
	}
}

// GetTrace retrieves a trace by ID
func (tc *TraceCollector) GetTrace(traceID string) (*RequestTrace, bool) {
	tc.tracesMu.RLock()
	defer tc.tracesMu.RUnlock()

	trace, exists := tc.traces[traceID]
	return trace, exists
}

// GetSlowTraces returns recent slow traces
func (tc *TraceCollector) GetSlowTraces() []*RequestTrace {
	tc.slowTracesMu.RLock()
	defer tc.slowTracesMu.RUnlock()

	result := make([]*RequestTrace, len(tc.slowTraces))
	copy(result, tc.slowTraces)
	return result
}

// GetTraceStats returns trace collection statistics
func (tc *TraceCollector) GetTraceStats() map[string]interface{} {
	tc.tracesMu.RLock()
	tc.slowTracesMu.RLock()
	defer tc.tracesMu.RUnlock()
	defer tc.slowTracesMu.RUnlock()

	return map[string]interface{}{
		"total_traces":      len(tc.traces),
		"slow_traces":       len(tc.slowTraces),
		"max_traces":        tc.maxTraces,
		"max_slow_traces":   tc.maxSlowTraces,
		"trace_ttl_seconds": tc.traceTTL.Seconds(),
		"enable_profiling":  tc.enableProfiling,
		"enable_sampling":   tc.enableSampling,
		"sampling_rate":     tc.samplingRate,
	}
}

// Performance metrics helpers

// capturePerformanceMetricsBefore captures performance metrics before request processing
func capturePerformanceMetricsBefore() *PerformanceMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &PerformanceMetrics{
		MemoryBefore:     m.HeapAlloc,
		GoroutinesBefore: runtime.NumGoroutine(),
		CPUTimeBefore:    time.Duration(m.TotalAlloc), // Approximation
	}
}

// capturePerformanceMetricsAfter captures performance metrics after request processing
func capturePerformanceMetricsAfter(perfMetrics *PerformanceMetrics) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	perfMetrics.MemoryAfter = m.HeapAlloc
	perfMetrics.GoroutinesAfter = runtime.NumGoroutine()
	perfMetrics.CPUTimeAfter = time.Duration(m.TotalAlloc)

	if perfMetrics.MemoryAfter > perfMetrics.MemoryBefore {
		perfMetrics.MemoryAllocated = perfMetrics.MemoryAfter - perfMetrics.MemoryBefore
	}
}

// ID generation helpers

// generateTraceID generates a unique trace ID
func generateTraceID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// generateSpanID generates a unique span ID
func generateSpanID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// AdvancedTracingMiddleware provides detailed request tracing
func AdvancedTracingMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if globalTraceCollector == nil {
			return c.Next()
		}

		// Get or create trace
		var trace *RequestTrace
		if existingTrace := c.Locals("trace"); existingTrace != nil {
			if tr, ok := existingTrace.(*RequestTrace); ok {
				trace = tr
			}
		}

		if trace == nil {
			trace = globalTraceCollector.startTrace(c)
			c.Locals("trace", trace)
		}

		// Add custom annotations
		trace.Annotations["middleware"] = "advanced_tracing"
		trace.Tags["version"] = "1.0"

		return c.Next()
	}
}
