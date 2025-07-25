package benchmark

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// LoadTestConfig 로드 테스트 설정
type LoadTestConfig struct {
	Duration        time.Duration
	ConcurrentUsers int
	RequestsPerUser int
	RampUpTime      time.Duration
	ThinkTime       time.Duration
	MaxErrorRate    float64
	TargetRPS       int
	TestScenarios   []LoadTestScenario
}

// LoadTestScenario 로드 테스트 시나리오
type LoadTestScenario struct {
	Name        string
	Path        string
	Weight      int // 요청 비율 가중치
	Method      string
	Headers     map[string]string
	MinDuration time.Duration // 최소 응답 시간
	MaxDuration time.Duration // 최대 응답 시간
}

// LoadTestResults 로드 테스트 결과
type LoadTestResults struct {
	TotalRequests    int64
	SuccessfulReqs   int64
	FailedRequests   int64
	ErrorRate        float64
	AverageLatency   time.Duration
	MedianLatency    time.Duration
	P95Latency       time.Duration
	P99Latency       time.Duration
	MaxLatency       time.Duration
	MinLatency       time.Duration
	ThroughputRPS    float64
	BytesTransferred int64
	StartTime        time.Time
	EndTime          time.Time
	Duration         time.Duration
	Latencies        []time.Duration
	ErrorsByType     map[string]int64
}

// LoadTestExecutor 로드 테스트 실행기
type LoadTestExecutor struct {
	env     *HandlerBenchmarkEnvironment
	config  LoadTestConfig
	results *LoadTestResults
	mu      sync.RWMutex
}

// NewLoadTestExecutor 새 로드 테스트 실행기 생성
func NewLoadTestExecutor(env *HandlerBenchmarkEnvironment, config LoadTestConfig) *LoadTestExecutor {
	return &LoadTestExecutor{
		env:    env,
		config: config,
		results: &LoadTestResults{
			ErrorsByType: make(map[string]int64),
			Latencies:    make([]time.Duration, 0),
		},
	}
}

// Execute 로드 테스트 실행
func (e *LoadTestExecutor) Execute() *LoadTestResults {
	e.results.StartTime = time.Now()

	var wg sync.WaitGroup
	stopChan := make(chan struct{})

	// 램프업을 위한 사용자 추가 간격 계산
	userRampInterval := e.config.RampUpTime / time.Duration(e.config.ConcurrentUsers)

	// 동시 사용자 시뮬레이션
	for i := 0; i < e.config.ConcurrentUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			// 램프업 지연
			time.Sleep(time.Duration(userID) * userRampInterval)

			e.simulateUser(userID, stopChan)
		}(i)
	}

	// 테스트 지속 시간 타이머
	if e.config.Duration > 0 {
		go func() {
			time.Sleep(e.config.Duration)
			close(stopChan)
		}()
	}

	// 모든 사용자 완료 대기
	wg.Wait()

	e.results.EndTime = time.Now()
	e.results.Duration = e.results.EndTime.Sub(e.results.StartTime)
	e.calculateStatistics()

	return e.results
}

// simulateUser 개별 사용자 시뮬레이션
func (e *LoadTestExecutor) simulateUser(userID int, stopChan <-chan struct{}) {
	requestCount := 0
	maxRequests := e.config.RequestsPerUser

	for {
		select {
		case <-stopChan:
			return
		default:
			if maxRequests > 0 && requestCount >= maxRequests {
				return
			}

			scenario := e.selectScenario()
			e.executeRequest(scenario, userID)
			requestCount++

			// Think time 적용
			if e.config.ThinkTime > 0 {
				jitter := time.Duration(rand.Int63n(int64(e.config.ThinkTime / 2)))
				time.Sleep(e.config.ThinkTime + jitter)
			}
		}
	}
}

// selectScenario 가중치 기반 시나리오 선택
func (e *LoadTestExecutor) selectScenario() LoadTestScenario {
	if len(e.config.TestScenarios) == 0 {
		// 기본 시나리오
		return LoadTestScenario{
			Name:   "Default",
			Path:   "/proxy/npm/express",
			Weight: 1,
			Method: "GET",
		}
	}

	totalWeight := 0
	for _, scenario := range e.config.TestScenarios {
		totalWeight += scenario.Weight
	}

	randomValue := rand.Intn(totalWeight)
	currentWeight := 0

	for _, scenario := range e.config.TestScenarios {
		currentWeight += scenario.Weight
		if randomValue < currentWeight {
			return scenario
		}
	}

	return e.config.TestScenarios[0]
}

// executeRequest 요청 실행
func (e *LoadTestExecutor) executeRequest(scenario LoadTestScenario, userID int) {
	atomic.AddInt64(&e.results.TotalRequests, 1)

	start := time.Now()

	req, err := http.NewRequest(scenario.Method, scenario.Path, nil)
	if err != nil {
		e.recordError("request_creation", err)
		return
	}

	// 헤더 설정
	for key, value := range scenario.Headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("User-Agent", fmt.Sprintf("LoadTest-User-%d", userID))

	resp, err := e.env.ProxyServer.Test(req, 30000) // 30초 타임아웃
	if err != nil {
		e.recordError("request_execution", err)
		return
	}

	// 응답 본문 읽기
	bytesRead, err := io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	duration := time.Since(start)

	if err != nil {
		e.recordError("response_reading", err)
		return
	}

	// 응답 상태 코드 확인
	if resp.StatusCode >= 400 {
		e.recordError(fmt.Sprintf("http_%d", resp.StatusCode), fmt.Errorf("HTTP %d", resp.StatusCode))
		return
	}

	// 성공한 요청 기록
	atomic.AddInt64(&e.results.SuccessfulReqs, 1)
	atomic.AddInt64(&e.results.BytesTransferred, bytesRead)

	e.mu.Lock()
	e.results.Latencies = append(e.results.Latencies, duration)
	e.mu.Unlock()
}

// recordError 에러 기록
func (e *LoadTestExecutor) recordError(errorType string, err error) {
	atomic.AddInt64(&e.results.FailedRequests, 1)

	e.mu.Lock()
	e.results.ErrorsByType[errorType]++
	e.mu.Unlock()
}

// calculateStatistics 통계 계산
func (e *LoadTestExecutor) calculateStatistics() {
	if e.results.TotalRequests == 0 {
		return
	}

	e.results.ErrorRate = float64(e.results.FailedRequests) / float64(e.results.TotalRequests)
	e.results.ThroughputRPS = float64(e.results.SuccessfulReqs) / e.results.Duration.Seconds()

	if len(e.results.Latencies) == 0 {
		return
	}

	// 지연시간 정렬
	latencies := make([]time.Duration, len(e.results.Latencies))
	copy(latencies, e.results.Latencies)

	// 간단한 버블 정렬 (성능보다는 정확성 우선)
	for i := 0; i < len(latencies); i++ {
		for j := i + 1; j < len(latencies); j++ {
			if latencies[i] > latencies[j] {
				latencies[i], latencies[j] = latencies[j], latencies[i]
			}
		}
	}

	// 통계 계산
	var total time.Duration
	for _, lat := range latencies {
		total += lat
	}

	e.results.AverageLatency = total / time.Duration(len(latencies))
	e.results.MinLatency = latencies[0]
	e.results.MaxLatency = latencies[len(latencies)-1]
	e.results.MedianLatency = latencies[len(latencies)/2]
	e.results.P95Latency = latencies[int(float64(len(latencies))*0.95)]
	e.results.P99Latency = latencies[int(float64(len(latencies))*0.99)]
}

// BenchmarkLoadTestSmall 소규모 로드 테스트
func BenchmarkLoadTestSmall(b *testing.B) {
	env := setupHandlerBenchmarkEnvironment(b)
	defer env.Cleanup()

	config := LoadTestConfig{
		Duration:        30 * time.Second,
		ConcurrentUsers: 10,
		RampUpTime:      5 * time.Second,
		ThinkTime:       100 * time.Millisecond,
		MaxErrorRate:    0.05, // 5% 에러율 허용
		TestScenarios: []LoadTestScenario{
			{Name: "NPM", Path: "/proxy/npm/express", Weight: 3, Method: "GET"},
			{Name: "PIP", Path: "/proxy/pip/pypi/requests/json", Weight: 2, Method: "GET"},
			{Name: "Docker", Path: "/proxy/docker/v2/", Weight: 1, Method: "GET"},
		},
	}

	b.ResetTimer()

	executor := NewLoadTestExecutor(env, config)
	results := executor.Execute()

	// 결과 검증
	require.LessOrEqual(b, results.ErrorRate, config.MaxErrorRate, "Error rate should be within acceptable limits")
	require.Greater(b, results.ThroughputRPS, 0.0, "Throughput should be positive")

	// 결과 보고
	b.ReportMetric(float64(results.TotalRequests), "requests")
	b.ReportMetric(results.ThroughputRPS, "rps")
	b.ReportMetric(float64(results.AverageLatency.Nanoseconds()), "avg_latency_ns")
	b.ReportMetric(results.ErrorRate*100, "error_rate_percent")
	b.ReportMetric(float64(results.BytesTransferred)/1024/1024, "mb_transferred")

	b.Logf("Load Test Results:")
	b.Logf("  Total Requests: %d", results.TotalRequests)
	b.Logf("  Successful: %d", results.SuccessfulReqs)
	b.Logf("  Failed: %d", results.FailedRequests)
	b.Logf("  Error Rate: %.2f%%", results.ErrorRate*100)
	b.Logf("  Throughput: %.2f RPS", results.ThroughputRPS)
	b.Logf("  Average Latency: %v", results.AverageLatency)
	b.Logf("  P95 Latency: %v", results.P95Latency)
	b.Logf("  P99 Latency: %v", results.P99Latency)
}

// BenchmarkLoadTestMedium 중간 규모 로드 테스트
func BenchmarkLoadTestMedium(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping medium load test in short mode")
	}

	env := setupHandlerBenchmarkEnvironment(b)
	defer env.Cleanup()

	config := LoadTestConfig{
		Duration:        60 * time.Second,
		ConcurrentUsers: 50,
		RampUpTime:      10 * time.Second,
		ThinkTime:       200 * time.Millisecond,
		MaxErrorRate:    0.05,
		TestScenarios: []LoadTestScenario{
			{Name: "NPM-Metadata", Path: "/proxy/npm/express", Weight: 4, Method: "GET"},
			{Name: "NPM-Package", Path: "/proxy/npm/express/-/express-4.18.2.tgz", Weight: 1, Method: "GET"},
			{Name: "Maven-POM", Path: "/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.pom", Weight: 3, Method: "GET"},
			{Name: "Maven-JAR", Path: "/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar", Weight: 1, Method: "GET"},
			{Name: "PIP-Metadata", Path: "/proxy/pip/pypi/requests/json", Weight: 3, Method: "GET"},
			{Name: "Docker-API", Path: "/proxy/docker/v2/", Weight: 2, Method: "GET"},
			{Name: "APT-Release", Path: "/proxy/apt/ubuntu/dists/jammy/Release", Weight: 2, Method: "GET"},
		},
	}

	b.ResetTimer()

	executor := NewLoadTestExecutor(env, config)
	results := executor.Execute()

	// 결과 검증
	require.LessOrEqual(b, results.ErrorRate, config.MaxErrorRate)
	require.Greater(b, results.ThroughputRPS, 0.0)

	// 결과 보고
	b.ReportMetric(float64(results.TotalRequests), "requests")
	b.ReportMetric(results.ThroughputRPS, "rps")
	b.ReportMetric(float64(results.AverageLatency.Nanoseconds()), "avg_latency_ns")
	b.ReportMetric(results.ErrorRate*100, "error_rate_percent")

	b.Logf("Medium Load Test Results:")
	b.Logf("  Duration: %v", results.Duration)
	b.Logf("  Total Requests: %d", results.TotalRequests)
	b.Logf("  Throughput: %.2f RPS", results.ThroughputRPS)
	b.Logf("  Error Rate: %.2f%%", results.ErrorRate*100)
	b.Logf("  Average Latency: %v", results.AverageLatency)
	b.Logf("  P95 Latency: %v", results.P95Latency)
	b.Logf("  P99 Latency: %v", results.P99Latency)
	b.Logf("  Data Transferred: %.2f MB", float64(results.BytesTransferred)/1024/1024)
}

// BenchmarkLoadTestLarge 대규모 로드 테스트
func BenchmarkLoadTestLarge(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping large load test in short mode")
	}

	env := setupHandlerBenchmarkEnvironment(b)
	defer env.Cleanup()

	config := LoadTestConfig{
		Duration:        120 * time.Second,
		ConcurrentUsers: 100,
		RampUpTime:      20 * time.Second,
		ThinkTime:       300 * time.Millisecond,
		MaxErrorRate:    0.10, // 대규모 테스트에서는 에러율을 약간 높게 설정
		TestScenarios: []LoadTestScenario{
			{Name: "NPM-Metadata", Path: "/proxy/npm/express", Weight: 5, Method: "GET"},
			{Name: "Maven-POM", Path: "/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.pom", Weight: 4, Method: "GET"},
			{Name: "PIP-Metadata", Path: "/proxy/pip/pypi/requests/json", Weight: 4, Method: "GET"},
			{Name: "Docker-API", Path: "/proxy/docker/v2/", Weight: 3, Method: "GET"},
			{Name: "YUM-Metadata", Path: "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/repomd.xml", Weight: 2, Method: "GET"},
			{Name: "APK-Index", Path: "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", Weight: 2, Method: "GET"},
			{Name: "APT-Release", Path: "/proxy/apt/ubuntu/dists/jammy/Release", Weight: 3, Method: "GET"},
		},
	}

	b.ResetTimer()

	executor := NewLoadTestExecutor(env, config)
	results := executor.Execute()

	// 결과 검증
	require.LessOrEqual(b, results.ErrorRate, config.MaxErrorRate)
	require.Greater(b, results.ThroughputRPS, 0.0)

	// 결과 보고
	b.ReportMetric(float64(results.TotalRequests), "requests")
	b.ReportMetric(results.ThroughputRPS, "rps")
	b.ReportMetric(float64(results.AverageLatency.Nanoseconds()), "avg_latency_ns")
	b.ReportMetric(results.ErrorRate*100, "error_rate_percent")

	b.Logf("Large Load Test Results:")
	b.Logf("  Duration: %v", results.Duration)
	b.Logf("  Concurrent Users: %d", config.ConcurrentUsers)
	b.Logf("  Total Requests: %d", results.TotalRequests)
	b.Logf("  Throughput: %.2f RPS", results.ThroughputRPS)
	b.Logf("  Error Rate: %.2f%%", results.ErrorRate*100)
	b.Logf("  Average Latency: %v", results.AverageLatency)
	b.Logf("  P95 Latency: %v", results.P95Latency)
	b.Logf("  P99 Latency: %v", results.P99Latency)
	b.Logf("  Data Transferred: %.2f MB", float64(results.BytesTransferred)/1024/1024)

	// 에러 타입별 분석
	if len(results.ErrorsByType) > 0 {
		b.Logf("  Error Breakdown:")
		for errorType, count := range results.ErrorsByType {
			b.Logf("    %s: %d", errorType, count)
		}
	}
}

// BenchmarkStressTest 스트레스 테스트
func BenchmarkStressTest(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping stress test in short mode")
	}

	env := setupHandlerBenchmarkEnvironment(b)
	defer env.Cleanup()

	// 스트레스 테스트: 매우 높은 동시성으로 시스템 한계 테스트
	config := LoadTestConfig{
		Duration:        60 * time.Second,
		ConcurrentUsers: 200,
		RampUpTime:      5 * time.Second,       // 빠른 램프업
		ThinkTime:       50 * time.Millisecond, // 매우 짧은 think time
		MaxErrorRate:    0.20,                  // 스트레스 테스트에서는 높은 에러율 허용
		TestScenarios: []LoadTestScenario{
			{Name: "High-Load-NPM", Path: "/proxy/npm/express", Weight: 1, Method: "GET"},
			{Name: "High-Load-PIP", Path: "/proxy/pip/pypi/requests/json", Weight: 1, Method: "GET"},
			{Name: "High-Load-Docker", Path: "/proxy/docker/v2/", Weight: 1, Method: "GET"},
		},
	}

	b.ResetTimer()

	executor := NewLoadTestExecutor(env, config)
	results := executor.Execute()

	// 스트레스 테스트는 에러율이 높을 수 있으므로 완료만 확인
	require.Greater(b, results.TotalRequests, int64(0))

	// 결과 보고
	b.ReportMetric(float64(results.TotalRequests), "requests")
	b.ReportMetric(results.ThroughputRPS, "rps")
	b.ReportMetric(float64(results.AverageLatency.Nanoseconds()), "avg_latency_ns")
	b.ReportMetric(results.ErrorRate*100, "error_rate_percent")

	b.Logf("Stress Test Results:")
	b.Logf("  Concurrent Users: %d", config.ConcurrentUsers)
	b.Logf("  Total Requests: %d", results.TotalRequests)
	b.Logf("  Successful: %d", results.SuccessfulReqs)
	b.Logf("  Failed: %d", results.FailedRequests)
	b.Logf("  Error Rate: %.2f%%", results.ErrorRate*100)
	b.Logf("  Throughput: %.2f RPS", results.ThroughputRPS)
	b.Logf("  Average Latency: %v", results.AverageLatency)
	b.Logf("  Max Latency: %v", results.MaxLatency)

	// 시스템이 완전히 실패하지 않았는지 확인
	require.Less(b, results.ErrorRate, 0.90, "System should not completely fail under stress")
}

// BenchmarkSpikeTest 스파이크 테스트
func BenchmarkSpikeTest(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping spike test in short mode")
	}

	env := setupHandlerBenchmarkEnvironment(b)
	defer env.Cleanup()

	// 스파이크 테스트: 갑작스러운 부하 증가 시뮬레이션
	config := LoadTestConfig{
		Duration:        30 * time.Second,
		ConcurrentUsers: 150,
		RampUpTime:      1 * time.Second,       // 매우 빠른 램프업으로 스파이크 시뮬레이션
		ThinkTime:       10 * time.Millisecond, // 거의 연속적인 요청
		MaxErrorRate:    0.30,                  // 스파이크 테스트에서는 높은 에러율 허용
		TestScenarios: []LoadTestScenario{
			{Name: "Spike-NPM", Path: "/proxy/npm/express", Weight: 1, Method: "GET"},
		},
	}

	b.ResetTimer()

	executor := NewLoadTestExecutor(env, config)
	results := executor.Execute()

	// 결과 보고
	b.ReportMetric(float64(results.TotalRequests), "requests")
	b.ReportMetric(results.ThroughputRPS, "rps")
	b.ReportMetric(results.ErrorRate*100, "error_rate_percent")

	b.Logf("Spike Test Results:")
	b.Logf("  Total Requests: %d", results.TotalRequests)
	b.Logf("  Error Rate: %.2f%%", results.ErrorRate*100)
	b.Logf("  Peak Throughput: %.2f RPS", results.ThroughputRPS)
	b.Logf("  Average Latency: %v", results.AverageLatency)
	b.Logf("  P99 Latency: %v", results.P99Latency)

	// 스파이크 테스트에서도 시스템이 완전히 실패하지 않아야 함
	require.Less(b, results.ErrorRate, 0.80, "System should survive spike loads")
}
