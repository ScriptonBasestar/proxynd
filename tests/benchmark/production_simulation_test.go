package benchmark

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ProductionSimulationConfig 프로덕션 시뮬레이션 설정
type ProductionSimulationConfig struct {
	Duration             time.Duration
	PeakHours            []int   // 24시간 기준 (예: 9, 10, 11 = 9AM-11AM)
	BaseRPS              int     // 기본 RPS
	PeakRPSMultiplier    float64 // 피크 시간 RPS 배수
	UserBehaviorProfiles []UserBehaviorProfile
	EnableCircuitBreaker bool
	EnableRateLimit      bool
	EnableCaching        bool
	NetworkLatency       time.Duration // 네트워크 지연 시뮬레이션
	FailureRate          float64       // 인위적 실패율
}

// UserBehaviorProfile 사용자 행동 프로필
type UserBehaviorProfile struct {
	Name            string
	Weight          int // 비율 가중치
	SessionDuration time.Duration
	ThinkTime       time.Duration
	RequestPattern  []RequestPattern
	CacheHitRate    float64 // 캐시 히트율
	RetryBehavior   RetryBehavior
}

// RequestPattern 요청 패턴
type RequestPattern struct {
	ProxyType    string        // npm, maven, pip, docker, etc.
	Probability  float64       // 요청 확률
	PayloadSize  int           // 평균 페이로드 크기
	ResponseTime time.Duration // 예상 응답 시간
}

// RetryBehavior 재시도 행동
type RetryBehavior struct {
	MaxRetries    int
	BackoffFactor time.Duration
	GiveUpAfter   time.Duration
}

// ProductionSimulator 프로덕션 시뮬레이터
type ProductionSimulator struct {
	config ProductionSimulationConfig
	env    *HandlerBenchmarkEnvironment
	stats  *SimulationStats
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// SimulationStats 시뮬레이션 통계
type SimulationStats struct {
	TotalRequests       int64
	SuccessfulRequests  int64
	FailedRequests      int64
	CacheHits           int64
	CacheMisses         int64
	CircuitBreakerTrips int64
	RateLimitHits       int64
	AverageLatency      time.Duration
	P95Latency          time.Duration
	P99Latency          time.Duration
	ErrorsByType        map[string]int64
	RequestsByProxy     map[string]int64
	ThroughputByHour    map[int]float64
	mu                  sync.RWMutex
	latencies           []time.Duration
}

// NewProductionSimulator 새 프로덕션 시뮬레이터 생성
func NewProductionSimulator(config ProductionSimulationConfig, env *HandlerBenchmarkEnvironment) *ProductionSimulator {
	ctx, cancel := context.WithCancel(context.Background())

	stats := &SimulationStats{
		ErrorsByType:     make(map[string]int64),
		RequestsByProxy:  make(map[string]int64),
		ThroughputByHour: make(map[int]float64),
		latencies:        make([]time.Duration, 0),
	}

	return &ProductionSimulator{
		config: config,
		env:    env,
		stats:  stats,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Run 시뮬레이션 실행
func (ps *ProductionSimulator) Run() *SimulationStats {
	startTime := time.Now()

	// 시간대별 부하 시뮬레이션
	ps.wg.Add(1)
	go ps.timeBasedLoadGenerator()

	// 사용자 행동 시뮬레이션
	for _, profile := range ps.config.UserBehaviorProfiles {
		userCount := ps.calculateUserCount(profile)
		for i := 0; i < userCount; i++ {
			ps.wg.Add(1)
			go ps.simulateUser(profile, i)
		}
	}

	// 통계 수집
	ps.wg.Add(1)
	go ps.collectStatistics()

	// 시뮬레이션 지속 시간 제한
	go func() {
		time.Sleep(ps.config.Duration)
		ps.cancel()
	}()

	ps.wg.Wait()

	// 최종 통계 계산
	ps.calculateFinalStats(time.Since(startTime))

	return ps.stats
}

// timeBasedLoadGenerator 시간대별 부하 생성
func (ps *ProductionSimulator) timeBasedLoadGenerator() {
	defer ps.wg.Done()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ps.ctx.Done():
			return
		case now := <-ticker.C:
			currentHour := now.Hour()
			currentRPS := ps.calculateCurrentRPS(currentHour)

			ps.stats.mu.Lock()
			ps.stats.ThroughputByHour[currentHour] = float64(currentRPS)
			ps.stats.mu.Unlock()

			// RPS에 따른 요청 생성
			for i := 0; i < currentRPS; i++ {
				if ps.shouldStop() {
					return
				}

				ps.wg.Add(1)
				go ps.executeBackgroundRequest()
			}
		}
	}
}

// calculateCurrentRPS 현재 시간의 RPS 계산
func (ps *ProductionSimulator) calculateCurrentRPS(hour int) int {
	baseRPS := ps.config.BaseRPS

	// 피크 시간 확인
	for _, peakHour := range ps.config.PeakHours {
		if hour == peakHour {
			return int(float64(baseRPS) * ps.config.PeakRPSMultiplier)
		}
	}

	return baseRPS
}

// calculateUserCount 사용자 수 계산
func (ps *ProductionSimulator) calculateUserCount(profile UserBehaviorProfile) int {
	totalWeight := 0
	for _, p := range ps.config.UserBehaviorProfiles {
		totalWeight += p.Weight
	}

	// 기본 동시 사용자 수를 50명으로 가정
	baseConcurrentUsers := 50
	return (baseConcurrentUsers * profile.Weight) / totalWeight
}

// simulateUser 개별 사용자 시뮬레이션
func (ps *ProductionSimulator) simulateUser(profile UserBehaviorProfile, userID int) {
	defer ps.wg.Done()

	sessionStart := time.Now()
	sessionEnd := sessionStart.Add(profile.SessionDuration)

	for time.Now().Before(sessionEnd) {
		if ps.shouldStop() {
			return
		}

		// 요청 패턴에 따른 요청 실행
		pattern := ps.selectRequestPattern(profile.RequestPattern)
		ps.executeUserRequest(pattern, profile, userID)

		// Think time 적용
		jitter := time.Duration(rand.Int63n(int64(profile.ThinkTime / 2)))
		time.Sleep(profile.ThinkTime + jitter)
	}
}

// selectRequestPattern 요청 패턴 선택
func (ps *ProductionSimulator) selectRequestPattern(patterns []RequestPattern) RequestPattern {
	if len(patterns) == 0 {
		return RequestPattern{
			ProxyType:    "npm",
			Probability:  1.0,
			PayloadSize:  1024,
			ResponseTime: 100 * time.Millisecond,
		}
	}

	randVal := rand.Float64()
	var cumulative float64

	for _, pattern := range patterns {
		cumulative += pattern.Probability
		if randVal <= cumulative {
			return pattern
		}
	}

	return patterns[0]
}

// executeUserRequest 사용자 요청 실행
func (ps *ProductionSimulator) executeUserRequest(pattern RequestPattern, profile UserBehaviorProfile, userID int) {
	// 캐시 히트 시뮬레이션
	if rand.Float64() < profile.CacheHitRate {
		atomic.AddInt64(&ps.stats.CacheHits, 1)
		// 캐시 히트의 경우 빠른 응답
		time.Sleep(5 * time.Millisecond)
		atomic.AddInt64(&ps.stats.SuccessfulRequests, 1)
		return
	}

	atomic.AddInt64(&ps.stats.CacheMisses, 1)

	// 실제 요청 실행
	ps.executeRequest(pattern, profile.RetryBehavior, userID)
}

// executeBackgroundRequest 백그라운드 요청 실행
func (ps *ProductionSimulator) executeBackgroundRequest() {
	defer ps.wg.Done()

	// 랜덤 프록시 타입 선택
	proxyTypes := []string{"npm", "maven", "pip", "docker", "apt", "yum", "apk"}
	proxyType := proxyTypes[rand.Intn(len(proxyTypes))]

	pattern := RequestPattern{
		ProxyType:    proxyType,
		Probability:  1.0,
		PayloadSize:  1024,
		ResponseTime: 50 * time.Millisecond,
	}

	retryBehavior := RetryBehavior{
		MaxRetries:    2,
		BackoffFactor: 100 * time.Millisecond,
		GiveUpAfter:   5 * time.Second,
	}

	ps.executeRequest(pattern, retryBehavior, -1) // -1은 백그라운드 요청을 의미
}

// executeRequest 요청 실행
func (ps *ProductionSimulator) executeRequest(pattern RequestPattern, retryBehavior RetryBehavior, userID int) {
	atomic.AddInt64(&ps.stats.TotalRequests, 1)

	startTime := time.Now()
	path := ps.buildRequestPath(pattern.ProxyType)

	// Rate Limiting 시뮬레이션
	if ps.config.EnableRateLimit && ps.shouldRateLimit() {
		atomic.AddInt64(&ps.stats.RateLimitHits, 1)
		atomic.AddInt64(&ps.stats.FailedRequests, 1)
		ps.recordError("rate_limit")
		return
	}

	// Circuit Breaker 시뮬레이션
	if ps.config.EnableCircuitBreaker && ps.shouldCircuitBreak() {
		atomic.AddInt64(&ps.stats.CircuitBreakerTrips, 1)
		atomic.AddInt64(&ps.stats.FailedRequests, 1)
		ps.recordError("circuit_breaker")
		return
	}

	// 재시도 로직
	var lastErr error
	for attempt := 0; attempt <= retryBehavior.MaxRetries; attempt++ {
		if ps.shouldStop() {
			return
		}

		// 네트워크 지연 시뮬레이션
		if ps.config.NetworkLatency > 0 {
			jitter := time.Duration(rand.Int63n(int64(ps.config.NetworkLatency / 2)))
			time.Sleep(ps.config.NetworkLatency + jitter)
		}

		// 인위적 실패 시뮬레이션
		if rand.Float64() < ps.config.FailureRate {
			lastErr = fmt.Errorf("simulated network failure")
			ps.recordError("network_failure")

			if attempt < retryBehavior.MaxRetries {
				backoff := time.Duration(attempt+1) * retryBehavior.BackoffFactor
				time.Sleep(backoff)
				continue
			}
			break
		}

		// 실제 HTTP 요청
		req, err := http.NewRequest("GET", path, nil)
		if err != nil {
			lastErr = err
			continue
		}

		req.Header.Set("User-Agent", fmt.Sprintf("ProductionSim-User-%d", userID))

		resp, err := ps.env.ProxyServer.Test(req, 30000) // 30초 타임아웃
		if err != nil {
			lastErr = err
			ps.recordError("request_error")

			if attempt < retryBehavior.MaxRetries {
				backoff := time.Duration(attempt+1) * retryBehavior.BackoffFactor
				time.Sleep(backoff)
				continue
			}
			break
		}

		// 응답 처리
		if resp.StatusCode >= 400 {
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			ps.recordError(fmt.Sprintf("http_%d", resp.StatusCode))
			_ = resp.Body.Close()

			if attempt < retryBehavior.MaxRetries {
				backoff := time.Duration(attempt+1) * retryBehavior.BackoffFactor
				time.Sleep(backoff)
				continue
			}
			break
		}

		// 성공
		_ = resp.Body.Close()
		atomic.AddInt64(&ps.stats.SuccessfulRequests, 1)

		duration := time.Since(startTime)
		ps.recordLatency(duration)
		ps.recordProxyRequest(pattern.ProxyType)

		return
	}

	// 모든 재시도 실패
	atomic.AddInt64(&ps.stats.FailedRequests, 1)
	if lastErr != nil {
		ps.recordError("max_retries_exceeded")
	}
}

// buildRequestPath 요청 경로 구성
func (ps *ProductionSimulator) buildRequestPath(proxyType string) string {
	paths := map[string][]string{
		"npm":    {"/proxy/npm/express", "/proxy/npm/react", "/proxy/npm/lodash"},
		"maven":  {"/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.pom", "/proxy/maven/junit/junit/4.13.2/junit-4.13.2.jar"},
		"pip":    {"/proxy/pip/pypi/requests/json", "/proxy/pip/simple/numpy/"},
		"docker": {"/proxy/docker/v2/", "/proxy/docker/v2/library/nginx/manifests/latest"},
		"apt":    {"/proxy/apt/ubuntu/dists/jammy/Release", "/proxy/apt/ubuntu/dists/jammy/main/binary-amd64/Packages"},
		"yum":    {"/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/repomd.xml"},
		"apk":    {"/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz"},
	}

	if pathList, exists := paths[proxyType]; exists {
		return pathList[rand.Intn(len(pathList))]
	}

	return "/proxy/npm/express" // 기본값
}

// shouldRateLimit Rate Limiting 적용 여부
func (ps *ProductionSimulator) shouldRateLimit() bool {
	// 간단한 확률 기반 rate limiting 시뮬레이션
	return rand.Float64() < 0.02 // 2% 확률
}

// shouldCircuitBreak Circuit Breaker 작동 여부
func (ps *ProductionSimulator) shouldCircuitBreak() bool {
	// 에러율이 높을 때 circuit breaker 작동 시뮬레이션
	errorRate := float64(ps.stats.FailedRequests) / float64(ps.stats.TotalRequests+1)
	return errorRate > 0.1 && rand.Float64() < 0.1
}

// recordError 에러 기록
func (ps *ProductionSimulator) recordError(errorType string) {
	ps.stats.mu.Lock()
	ps.stats.ErrorsByType[errorType]++
	ps.stats.mu.Unlock()
}

// recordLatency 지연시간 기록
func (ps *ProductionSimulator) recordLatency(duration time.Duration) {
	ps.stats.mu.Lock()
	ps.stats.latencies = append(ps.stats.latencies, duration)
	ps.stats.mu.Unlock()
}

// recordProxyRequest 프록시 요청 기록
func (ps *ProductionSimulator) recordProxyRequest(proxyType string) {
	ps.stats.mu.Lock()
	ps.stats.RequestsByProxy[proxyType]++
	ps.stats.mu.Unlock()
}

// collectStatistics 통계 수집
func (ps *ProductionSimulator) collectStatistics() {
	defer ps.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ps.ctx.Done():
			return
		case <-ticker.C:
			// 주기적 통계 업데이트 (메모리 사용량 등)
			ps.calculateLatencyStats()
		}
	}
}

// calculateLatencyStats 지연시간 통계 계산
func (ps *ProductionSimulator) calculateLatencyStats() {
	ps.stats.mu.Lock()
	defer ps.stats.mu.Unlock()

	if len(ps.stats.latencies) == 0 {
		return
	}

	// 평균 지연시간 계산
	var total time.Duration
	for _, lat := range ps.stats.latencies {
		total += lat
	}
	ps.stats.AverageLatency = total / time.Duration(len(ps.stats.latencies))

	// 지연시간 정렬하여 백분위수 계산
	latencies := make([]time.Duration, len(ps.stats.latencies))
	copy(latencies, ps.stats.latencies)

	// 간단한 정렬 (성능보다는 정확성 우선)
	for i := 0; i < len(latencies); i++ {
		for j := i + 1; j < len(latencies); j++ {
			if latencies[i] > latencies[j] {
				latencies[i], latencies[j] = latencies[j], latencies[i]
			}
		}
	}

	// P95, P99 계산
	p95Index := int(float64(len(latencies)) * 0.95)
	p99Index := int(float64(len(latencies)) * 0.99)

	if p95Index < len(latencies) {
		ps.stats.P95Latency = latencies[p95Index]
	}
	if p99Index < len(latencies) {
		ps.stats.P99Latency = latencies[p99Index]
	}
}

// calculateFinalStats 최종 통계 계산
func (ps *ProductionSimulator) calculateFinalStats(totalDuration time.Duration) {
	ps.calculateLatencyStats()

	// 처리량 계산은 이미 시간별로 기록됨
}

// shouldStop 중지 여부 확인
func (ps *ProductionSimulator) shouldStop() bool {
	select {
	case <-ps.ctx.Done():
		return true
	default:
		return false
	}
}

// BenchmarkProductionSimulation 프로덕션 시뮬레이션 벤치마크
func BenchmarkProductionSimulation(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping production simulation in short mode")
	}

	env := setupHandlerBenchmarkEnvironment(b)
	defer env.Cleanup()

	// 프로덕션 시뮬레이션 설정
	config := ProductionSimulationConfig{
		Duration:             120 * time.Second,        // 2분 시뮬레이션
		PeakHours:            []int{9, 10, 11, 14, 15}, // 오전 9-11시, 오후 2-3시
		BaseRPS:              20,
		PeakRPSMultiplier:    3.0,
		EnableCircuitBreaker: true,
		EnableRateLimit:      true,
		EnableCaching:        true,
		NetworkLatency:       50 * time.Millisecond,
		FailureRate:          0.02, // 2% 실패율
		UserBehaviorProfiles: []UserBehaviorProfile{
			{
				Name:            "DeveloperUser",
				Weight:          40, // 40% 비율
				SessionDuration: 30 * time.Minute,
				ThinkTime:       5 * time.Second,
				CacheHitRate:    0.7, // 70% 캐시 히트율
				RequestPattern: []RequestPattern{
					{ProxyType: "npm", Probability: 0.4, PayloadSize: 2048, ResponseTime: 100 * time.Millisecond},
					{ProxyType: "maven", Probability: 0.3, PayloadSize: 5120, ResponseTime: 200 * time.Millisecond},
					{ProxyType: "pip", Probability: 0.3, PayloadSize: 1024, ResponseTime: 150 * time.Millisecond},
				},
				RetryBehavior: RetryBehavior{
					MaxRetries:    3,
					BackoffFactor: 500 * time.Millisecond,
					GiveUpAfter:   10 * time.Second,
				},
			},
			{
				Name:            "CIUser",
				Weight:          30, // 30% 비율
				SessionDuration: 10 * time.Minute,
				ThinkTime:       1 * time.Second,
				CacheHitRate:    0.5, // 50% 캐시 히트율
				RequestPattern: []RequestPattern{
					{ProxyType: "docker", Probability: 0.5, PayloadSize: 10240, ResponseTime: 500 * time.Millisecond},
					{ProxyType: "npm", Probability: 0.3, PayloadSize: 2048, ResponseTime: 100 * time.Millisecond},
					{ProxyType: "maven", Probability: 0.2, PayloadSize: 5120, ResponseTime: 200 * time.Millisecond},
				},
				RetryBehavior: RetryBehavior{
					MaxRetries:    5,
					BackoffFactor: 1 * time.Second,
					GiveUpAfter:   30 * time.Second,
				},
			},
			{
				Name:            "AdminUser",
				Weight:          30, // 30% 비율
				SessionDuration: 60 * time.Minute,
				ThinkTime:       10 * time.Second,
				CacheHitRate:    0.3, // 30% 캐시 히트율 (다양한 패키지 요청)
				RequestPattern: []RequestPattern{
					{ProxyType: "apt", Probability: 0.25, PayloadSize: 8192, ResponseTime: 300 * time.Millisecond},
					{ProxyType: "yum", Probability: 0.25, PayloadSize: 8192, ResponseTime: 300 * time.Millisecond},
					{ProxyType: "apk", Probability: 0.25, PayloadSize: 4096, ResponseTime: 200 * time.Millisecond},
					{ProxyType: "docker", Probability: 0.25, PayloadSize: 10240, ResponseTime: 500 * time.Millisecond},
				},
				RetryBehavior: RetryBehavior{
					MaxRetries:    2,
					BackoffFactor: 2 * time.Second,
					GiveUpAfter:   15 * time.Second,
				},
			},
		},
	}

	b.ResetTimer()

	// 시뮬레이션 실행
	simulator := NewProductionSimulator(config, env)
	stats := simulator.Run()

	// 결과 검증
	require.Greater(b, stats.TotalRequests, int64(0), "Should have processed requests")
	require.Greater(b, stats.SuccessfulRequests, int64(0), "Should have successful requests")

	successRate := float64(stats.SuccessfulRequests) / float64(stats.TotalRequests)
	require.Greater(b, successRate, 0.8, "Success rate should be above 80%")

	// 성능 메트릭 보고
	b.ReportMetric(float64(stats.TotalRequests), "total_requests")
	b.ReportMetric(float64(stats.SuccessfulRequests), "successful_requests")
	b.ReportMetric(successRate*100, "success_rate_percent")
	b.ReportMetric(float64(stats.AverageLatency.Nanoseconds()), "avg_latency_ns")
	b.ReportMetric(float64(stats.P95Latency.Nanoseconds()), "p95_latency_ns")
	b.ReportMetric(float64(stats.P99Latency.Nanoseconds()), "p99_latency_ns")

	// 상세 결과 로그
	b.Logf("Production Simulation Results:")
	b.Logf("  Duration: %v", config.Duration)
	b.Logf("  Total Requests: %d", stats.TotalRequests)
	b.Logf("  Successful: %d", stats.SuccessfulRequests)
	b.Logf("  Failed: %d", stats.FailedRequests)
	b.Logf("  Success Rate: %.2f%%", successRate*100)
	b.Logf("  Cache Hits: %d", stats.CacheHits)
	b.Logf("  Cache Misses: %d", stats.CacheMisses)
	b.Logf("  Cache Hit Rate: %.2f%%", float64(stats.CacheHits)/float64(stats.CacheHits+stats.CacheMisses)*100)
	b.Logf("  Circuit Breaker Trips: %d", stats.CircuitBreakerTrips)
	b.Logf("  Rate Limit Hits: %d", stats.RateLimitHits)
	b.Logf("  Average Latency: %v", stats.AverageLatency)
	b.Logf("  P95 Latency: %v", stats.P95Latency)
	b.Logf("  P99 Latency: %v", stats.P99Latency)

	// 프록시별 요청 분포
	b.Logf("  Requests by Proxy Type:")
	for proxyType, count := range stats.RequestsByProxy {
		percentage := float64(count) / float64(stats.SuccessfulRequests) * 100
		b.Logf("    %s: %d (%.1f%%)", proxyType, count, percentage)
	}

	// 에러 분석
	if len(stats.ErrorsByType) > 0 {
		b.Logf("  Error Breakdown:")
		for errorType, count := range stats.ErrorsByType {
			percentage := float64(count) / float64(stats.TotalRequests) * 100
			b.Logf("    %s: %d (%.2f%%)", errorType, count, percentage)
		}
	}

	// 시간별 처리량
	if len(stats.ThroughputByHour) > 0 {
		b.Logf("  Throughput by Hour:")
		for hour, rps := range stats.ThroughputByHour {
			b.Logf("    %02d:00: %.1f RPS", hour, rps)
		}
	}
}
