package benchmark

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ProfilingConfig 프로파일링 설정
type ProfilingConfig struct {
	EnableCPU    bool
	EnableMemory bool
	EnableTrace  bool
	OutputDir    string
	Duration     time.Duration
	Prefix       string
}

// ProfilingSession 프로파일링 세션
type ProfilingSession struct {
	config    ProfilingConfig
	cpuFile   *os.File
	traceFile *os.File
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	started   bool
	mu        sync.Mutex
}

// NewProfilingSession 새 프로파일링 세션 생성
func NewProfilingSession(config ProfilingConfig) *ProfilingSession {
	ctx, cancel := context.WithCancel(context.Background())

	// 출력 디렉토리 확인
	if config.OutputDir == "" {
		config.OutputDir = "profiles"
	}

	_ = os.MkdirAll(config.OutputDir, 0o755)

	return &ProfilingSession{
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start 프로파일링 시작
func (ps *ProfilingSession) Start() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.started {
		return fmt.Errorf("profiling session already started")
	}

	timestamp := time.Now().Format("20060102_150405")
	prefix := ps.config.Prefix
	if prefix == "" {
		prefix = "benchmark"
	}

	// CPU 프로파일링 시작
	if ps.config.EnableCPU {
		cpuPath := filepath.Join(ps.config.OutputDir, fmt.Sprintf("%s_cpu_%s.prof", prefix, timestamp))
		cpuFile, err := os.Create(cpuPath)
		if err != nil {
			return fmt.Errorf("failed to create CPU profile file: %w", err)
		}
		ps.cpuFile = cpuFile

		if err := pprof.StartCPUProfile(cpuFile); err != nil {
			_ = cpuFile.Close()
			return fmt.Errorf("failed to start CPU profiling: %w", err)
		}
	}

	// Trace 프로파일링 시작
	if ps.config.EnableTrace {
		tracePath := filepath.Join(ps.config.OutputDir, fmt.Sprintf("%s_trace_%s.trace", prefix, timestamp))
		traceFile, err := os.Create(tracePath)
		if err != nil {
			return fmt.Errorf("failed to create trace file: %w", err)
		}
		ps.traceFile = traceFile

		if err := trace.Start(traceFile); err != nil {
			_ = traceFile.Close()
			return fmt.Errorf("failed to start trace: %w", err)
		}
	}

	// 메모리 프로파일링 (주기적 캡처)
	if ps.config.EnableMemory {
		ps.wg.Add(1)
		go ps.memoryProfiler(timestamp, prefix)
	}

	// 지속 시간 제한
	if ps.config.Duration > 0 {
		ps.wg.Add(1)
		go func() {
			defer ps.wg.Done()
			select {
			case <-time.After(ps.config.Duration):
				ps.Stop()
			case <-ps.ctx.Done():
				return
			}
		}()
	}

	ps.started = true
	return nil
}

// Stop 프로파일링 중지
func (ps *ProfilingSession) Stop() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if !ps.started {
		return
	}

	// 컨텍스트 취소
	ps.cancel()

	// CPU 프로파일링 중지
	if ps.config.EnableCPU && ps.cpuFile != nil {
		pprof.StopCPUProfile()
		_ = ps.cpuFile.Close()
		ps.cpuFile = nil
	}

	// Trace 중지
	if ps.config.EnableTrace && ps.traceFile != nil {
		trace.Stop()
		_ = ps.traceFile.Close()
		ps.traceFile = nil
	}

	ps.started = false

	// 고루틴 종료 대기
	ps.wg.Wait()
}

// memoryProfiler 주기적 메모리 프로파일링
func (ps *ProfilingSession) memoryProfiler(timestamp, prefix string) {
	defer ps.wg.Done()

	ticker := time.NewTicker(10 * time.Second) // 10초마다 메모리 프로파일 수집
	defer ticker.Stop()

	counter := 0
	for {
		select {
		case <-ticker.C:
			counter++
			ps.captureMemoryProfile(timestamp, prefix, counter)
		case <-ps.ctx.Done():
			// 마지막 메모리 프로파일 캡처
			ps.captureMemoryProfile(timestamp, prefix, counter+1)
			return
		}
	}
}

// captureMemoryProfile 메모리 프로파일 캡처
func (ps *ProfilingSession) captureMemoryProfile(timestamp, prefix string, counter int) {
	runtime.GC() // 정확한 메모리 측정을 위해 GC 실행

	memPath := filepath.Join(ps.config.OutputDir, fmt.Sprintf("%s_mem_%s_%03d.prof", prefix, timestamp, counter))
	memFile, err := os.Create(memPath)
	if err != nil {
		return
	}
	defer func() { _ = memFile.Close() }()

	_ = pprof.WriteHeapProfile(memFile)
}

// ProfiledLoadTest 프로파일링을 포함한 로드 테스트
func ProfiledLoadTest(b *testing.B, config LoadTestConfig, profilingConfig ProfilingConfig) *LoadTestResults {
	// 프로파일링 세션 시작
	session := NewProfilingSession(profilingConfig)
	err := session.Start()
	require.NoError(b, err)
	defer session.Stop()

	// 핸들러 벤치마크 환경 설정
	env := setupHandlerBenchmarkEnvironment(b)
	defer env.Cleanup()

	// 로드 테스트 실행
	executor := NewLoadTestExecutor(env, config)
	results := executor.Execute()

	return results
}

// BenchmarkProfiledLoadTestSmall 프로파일링을 포함한 소규모 로드 테스트
func BenchmarkProfiledLoadTestSmall(b *testing.B) {
	profilingConfig := ProfilingConfig{
		EnableCPU:    true,
		EnableMemory: true,
		EnableTrace:  true,
		OutputDir:    "profiles/load_test_small",
		Prefix:       "load_small",
		Duration:     35 * time.Second, // 테스트보다 약간 길게
	}

	loadConfig := LoadTestConfig{
		Duration:        30 * time.Second,
		ConcurrentUsers: 10,
		RampUpTime:      5 * time.Second,
		ThinkTime:       100 * time.Millisecond,
		MaxErrorRate:    0.05,
		TestScenarios: []LoadTestScenario{
			{Name: "NPM", Path: "/proxy/npm/express", Weight: 3, Method: "GET"},
			{Name: "PIP", Path: "/proxy/pip/pypi/requests/json", Weight: 2, Method: "GET"},
			{Name: "Docker", Path: "/proxy/docker/v2/", Weight: 1, Method: "GET"},
		},
	}

	b.ResetTimer()

	results := ProfiledLoadTest(b, loadConfig, profilingConfig)

	// 결과 검증
	require.LessOrEqual(b, results.ErrorRate, loadConfig.MaxErrorRate)
	require.Greater(b, results.ThroughputRPS, 0.0)

	// 프로파일링 결과 보고
	b.Logf("Profiled Load Test Results:")
	b.Logf("  Total Requests: %d", results.TotalRequests)
	b.Logf("  Throughput: %.2f RPS", results.ThroughputRPS)
	b.Logf("  Error Rate: %.2f%%", results.ErrorRate*100)
	b.Logf("  Average Latency: %v", results.AverageLatency)
	b.Logf("  P95 Latency: %v", results.P95Latency)
	b.Logf("  P99 Latency: %v", results.P99Latency)
	b.Logf("  Profiles saved to: %s", profilingConfig.OutputDir)
}

// BenchmarkProfiledLoadTestMedium 프로파일링을 포함한 중간 규모 로드 테스트
func BenchmarkProfiledLoadTestMedium(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping profiled medium load test in short mode")
	}

	profilingConfig := ProfilingConfig{
		EnableCPU:    true,
		EnableMemory: true,
		EnableTrace:  false, // 중간 규모에서는 trace 비활성화 (파일 크기 고려)
		OutputDir:    "profiles/load_test_medium",
		Prefix:       "load_medium",
		Duration:     65 * time.Second,
	}

	loadConfig := LoadTestConfig{
		Duration:        60 * time.Second,
		ConcurrentUsers: 50,
		RampUpTime:      10 * time.Second,
		ThinkTime:       200 * time.Millisecond,
		MaxErrorRate:    0.05,
		TestScenarios: []LoadTestScenario{
			{Name: "NPM-Metadata", Path: "/proxy/npm/express", Weight: 4, Method: "GET"},
			{Name: "Maven-POM", Path: "/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.pom", Weight: 3, Method: "GET"},
			{Name: "PIP-Metadata", Path: "/proxy/pip/pypi/requests/json", Weight: 3, Method: "GET"},
			{Name: "Docker-API", Path: "/proxy/docker/v2/", Weight: 2, Method: "GET"},
		},
	}

	b.ResetTimer()

	results := ProfiledLoadTest(b, loadConfig, profilingConfig)

	// 결과 검증 및 보고
	require.LessOrEqual(b, results.ErrorRate, loadConfig.MaxErrorRate)
	require.Greater(b, results.ThroughputRPS, 0.0)

	// 자세한 성능 메트릭 보고
	b.ReportMetric(float64(results.TotalRequests), "requests")
	b.ReportMetric(results.ThroughputRPS, "rps")
	b.ReportMetric(float64(results.AverageLatency.Nanoseconds()), "avg_latency_ns")
	b.ReportMetric(results.ErrorRate*100, "error_rate_percent")

	b.Logf("Profiled Medium Load Test Results:")
	b.Logf("  Duration: %v", results.Duration)
	b.Logf("  Total Requests: %d", results.TotalRequests)
	b.Logf("  Throughput: %.2f RPS", results.ThroughputRPS)
	b.Logf("  Error Rate: %.2f%%", results.ErrorRate*100)
	b.Logf("  Average Latency: %v", results.AverageLatency)
	b.Logf("  P95 Latency: %v", results.P95Latency)
	b.Logf("  P99 Latency: %v", results.P99Latency)
	b.Logf("  Data Transferred: %.2f MB", float64(results.BytesTransferred)/1024/1024)
	b.Logf("  CPU and Memory profiles saved to: %s", profilingConfig.OutputDir)
}

// BenchmarkMemoryProfiling 메모리 집중 프로파일링 벤치마크
func BenchmarkMemoryProfiling(b *testing.B) {
	profilingConfig := ProfilingConfig{
		EnableCPU:    false,
		EnableMemory: true,
		EnableTrace:  false,
		OutputDir:    "profiles/memory_intensive",
		Prefix:       "memory",
		Duration:     45 * time.Second,
	}

	// 메모리 집중적인 시나리오
	loadConfig := LoadTestConfig{
		Duration:        40 * time.Second,
		ConcurrentUsers: 30,
		RampUpTime:      5 * time.Second,
		ThinkTime:       50 * time.Millisecond,
		MaxErrorRate:    0.10,
		TestScenarios: []LoadTestScenario{
			// 큰 파일 다운로드 시나리오
			{Name: "Maven-JAR", Path: "/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar", Weight: 2, Method: "GET"},
			{Name: "Docker-Layer", Path: "/proxy/docker/v2/library/nginx/blobs/sha256:layer-digest", Weight: 1, Method: "GET"},
			{Name: "YUM-Package", Path: "/proxy/yum/centos/8/BaseOS/x86_64/os/Packages/nginx-1.20.1-1.el8.x86_64.rpm", Weight: 1, Method: "GET"},
		},
	}

	b.ResetTimer()

	results := ProfiledLoadTest(b, loadConfig, profilingConfig)

	// 메모리 관련 메트릭 수집
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	b.Logf("Memory Profiling Results:")
	b.Logf("  Total Requests: %d", results.TotalRequests)
	b.Logf("  Data Transferred: %.2f MB", float64(results.BytesTransferred)/1024/1024)
	b.Logf("  Error Rate: %.2f%%", results.ErrorRate*100)
	b.Logf("  Current Memory Usage:")
	b.Logf("    Alloc: %.2f MB", float64(memStats.Alloc)/1024/1024)
	b.Logf("    TotalAlloc: %.2f MB", float64(memStats.TotalAlloc)/1024/1024)
	b.Logf("    Sys: %.2f MB", float64(memStats.Sys)/1024/1024)
	b.Logf("    NumGC: %d", memStats.NumGC)
	b.Logf("  Memory profiles saved to: %s", profilingConfig.OutputDir)
}

// BenchmarkCPUProfiling CPU 집중 프로파일링 벤치마크
func BenchmarkCPUProfiling(b *testing.B) {
	profilingConfig := ProfilingConfig{
		EnableCPU:    true,
		EnableMemory: false,
		EnableTrace:  false,
		OutputDir:    "profiles/cpu_intensive",
		Prefix:       "cpu",
		Duration:     35 * time.Second,
	}

	// CPU 집중적인 시나리오 (많은 동시 요청)
	loadConfig := LoadTestConfig{
		Duration:        30 * time.Second,
		ConcurrentUsers: 100,
		RampUpTime:      2 * time.Second,
		ThinkTime:       10 * time.Millisecond,
		MaxErrorRate:    0.15,
		TestScenarios: []LoadTestScenario{
			{Name: "NPM-Fast", Path: "/proxy/npm/express", Weight: 1, Method: "GET"},
			{Name: "PIP-Fast", Path: "/proxy/pip/pypi/requests/json", Weight: 1, Method: "GET"},
			{Name: "Docker-Fast", Path: "/proxy/docker/v2/", Weight: 1, Method: "GET"},
		},
	}

	b.ResetTimer()

	results := ProfiledLoadTest(b, loadConfig, profilingConfig)

	b.Logf("CPU Profiling Results:")
	b.Logf("  Total Requests: %d", results.TotalRequests)
	b.Logf("  Throughput: %.2f RPS", results.ThroughputRPS)
	b.Logf("  Error Rate: %.2f%%", results.ErrorRate*100)
	b.Logf("  Average Latency: %v", results.AverageLatency)
	b.Logf("  P99 Latency: %v", results.P99Latency)
	b.Logf("  CPU profile saved to: %s", profilingConfig.OutputDir)
}

// BenchmarkFullProfiling 전체 프로파일링 벤치마크 (CPU + Memory + Trace)
func BenchmarkFullProfiling(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping full profiling test in short mode")
	}

	profilingConfig := ProfilingConfig{
		EnableCPU:    true,
		EnableMemory: true,
		EnableTrace:  true,
		OutputDir:    "profiles/full_profiling",
		Prefix:       "full",
		Duration:     35 * time.Second,
	}

	loadConfig := LoadTestConfig{
		Duration:        30 * time.Second,
		ConcurrentUsers: 25,
		RampUpTime:      5 * time.Second,
		ThinkTime:       100 * time.Millisecond,
		MaxErrorRate:    0.10,
		TestScenarios: []LoadTestScenario{
			{Name: "NPM", Path: "/proxy/npm/express", Weight: 2, Method: "GET"},
			{Name: "Maven", Path: "/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.pom", Weight: 2, Method: "GET"},
			{Name: "PIP", Path: "/proxy/pip/pypi/requests/json", Weight: 2, Method: "GET"},
			{Name: "Docker", Path: "/proxy/docker/v2/", Weight: 1, Method: "GET"},
		},
	}

	b.ResetTimer()

	results := ProfiledLoadTest(b, loadConfig, profilingConfig)

	// 최종 메모리 상태 수집
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	b.Logf("Full Profiling Results:")
	b.Logf("  Total Requests: %d", results.TotalRequests)
	b.Logf("  Successful: %d", results.SuccessfulReqs)
	b.Logf("  Failed: %d", results.FailedRequests)
	b.Logf("  Throughput: %.2f RPS", results.ThroughputRPS)
	b.Logf("  Error Rate: %.2f%%", results.ErrorRate*100)
	b.Logf("  Latency Stats:")
	b.Logf("    Average: %v", results.AverageLatency)
	b.Logf("    Median: %v", results.MedianLatency)
	b.Logf("    P95: %v", results.P95Latency)
	b.Logf("    P99: %v", results.P99Latency)
	b.Logf("    Max: %v", results.MaxLatency)
	b.Logf("  Data Transferred: %.2f MB", float64(results.BytesTransferred)/1024/1024)
	b.Logf("  Memory Usage:")
	b.Logf("    Current Alloc: %.2f MB", float64(memStats.Alloc)/1024/1024)
	b.Logf("    Total Alloc: %.2f MB", float64(memStats.TotalAlloc)/1024/1024)
	b.Logf("    Sys: %.2f MB", float64(memStats.Sys)/1024/1024)
	b.Logf("    GC Cycles: %d", memStats.NumGC)
	b.Logf("  All profiles (CPU, Memory, Trace) saved to: %s", profilingConfig.OutputDir)

	// 에러 분석
	if len(results.ErrorsByType) > 0 {
		b.Logf("  Error Breakdown:")
		for errorType, count := range results.ErrorsByType {
			b.Logf("    %s: %d", errorType, count)
		}
	}
}
