package integration

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CacheScenarioPattern 캐시 시나리오 검증 공통 패턴
type CacheScenarioPattern struct {
	ProxyType     string
	Path          string
	ExpectedMiss  string // 캐시 MISS 시 기대하는 응답 내용
	ExpectedHit   string // 캐시 HIT 시 기대하는 응답 내용  
	CacheTTL      time.Duration
	ValidateFunc  func(t *testing.T, body []byte, headers http.Header) // 커스텀 검증 함수
}

// TestCacheScenario 캐시 시나리오 테스트 패턴
func TestCacheScenario(t *testing.T, env *IntegrationTestEnvironment, pattern CacheScenarioPattern) {
	t.Helper()

	t.Run(fmt.Sprintf("Cache_%s_%s", pattern.ProxyType, pattern.Path), func(t *testing.T) {
		// Step 1: 캐시 MISS 테스트 (첫 번째 요청)
		t.Run("CacheMiss", func(t *testing.T) {
			resp, err := env.MakeRequest("GET", fmt.Sprintf("/proxy/%s/%s", pattern.ProxyType, pattern.Path), nil)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			// 캐시 상태 헤더 확인 (MISS이어야 함)
			cacheStatus := resp.Header.Get("X-Cache-Status")
			t.Logf("Cache status: %s", cacheStatus)
			
			// 응답 내용 검증
			if pattern.ExpectedMiss != "" {
				assert.Contains(t, string(body), pattern.ExpectedMiss)
			}
			
			// 커스텀 검증 함수 실행
			if pattern.ValidateFunc != nil {
				pattern.ValidateFunc(t, body, resp.Header)
			}
		})

		// Step 2: 캐시 HIT 테스트 (두 번째 요청)
		t.Run("CacheHit", func(t *testing.T) {
			// 즉시 두 번째 요청 실행
			resp, err := env.MakeRequest("GET", fmt.Sprintf("/proxy/%s/%s", pattern.ProxyType, pattern.Path), nil)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			// 응답 내용 검증
			if pattern.ExpectedHit != "" {
				assert.Contains(t, string(body), pattern.ExpectedHit)
			}
			
			// 커스텀 검증 함수 실행
			if pattern.ValidateFunc != nil {
				pattern.ValidateFunc(t, body, resp.Header)
			}
		})

		// Step 3: 캐시 만료 테스트 (TTL 이후)
		t.Run("CacheExpiration", func(t *testing.T) {
			if pattern.CacheTTL <= 0 {
				t.Skip("CacheTTL not specified, skipping expiration test")
			}

			// TTL보다 조금 더 오래 대기
			time.Sleep(pattern.CacheTTL + 100*time.Millisecond)
			
			resp, err := env.MakeRequest("GET", fmt.Sprintf("/proxy/%s/%s", pattern.ProxyType, pattern.Path), nil)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			
			// 캐시가 만료되어 다시 MISS가 되어야 함
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			if pattern.ExpectedMiss != "" {
				assert.Contains(t, string(body), pattern.ExpectedMiss)
			}
		})
	})
}

// ErrorRecoveryPattern 에러 처리 및 복구 패턴
type ErrorRecoveryPattern struct {
	ProxyType        string
	Path             string
	ErrorSimulation  func(env *IntegrationTestEnvironment) // 에러 상황 시뮬레이션
	RecoveryAction   func(env *IntegrationTestEnvironment) // 복구 액션
	ExpectedError    int                                   // 기대하는 에러 코드
	ExpectedRecovery string                               // 복구 후 기대하는 응답
}

// TestErrorRecovery 에러 처리 및 복구 테스트 패턴
func TestErrorRecovery(t *testing.T, env *IntegrationTestEnvironment, pattern ErrorRecoveryPattern) {
	t.Helper()

	t.Run(fmt.Sprintf("ErrorRecovery_%s_%s", pattern.ProxyType, pattern.Path), func(t *testing.T) {
		// Step 1: 정상 상태 확인
		t.Run("NormalOperation", func(t *testing.T) {
			resp, err := env.MakeRequest("GET", fmt.Sprintf("/proxy/%s/%s", pattern.ProxyType, pattern.Path), nil)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})

		// Step 2: 에러 시뮬레이션 및 테스트
		t.Run("ErrorSimulation", func(t *testing.T) {
			if pattern.ErrorSimulation != nil {
				pattern.ErrorSimulation(env)
			}

			resp, err := env.MakeRequest("GET", fmt.Sprintf("/proxy/%s/%s", pattern.ProxyType, pattern.Path), nil)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			if pattern.ExpectedError > 0 {
				assert.Equal(t, pattern.ExpectedError, resp.StatusCode)
			} else {
				assert.NotEqual(t, http.StatusOK, resp.StatusCode)
			}
		})

		// Step 3: 복구 및 테스트
		t.Run("Recovery", func(t *testing.T) {
			if pattern.RecoveryAction != nil {
				pattern.RecoveryAction(env)
			}

			// 복구 후 재요청
			resp, err := env.MakeRequest("GET", fmt.Sprintf("/proxy/%s/%s", pattern.ProxyType, pattern.Path), nil)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			if pattern.ExpectedRecovery != "" {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Contains(t, string(body), pattern.ExpectedRecovery)
			}
		})

		// Step 4: 복구 후 안정성 테스트 (여러 번 요청)
		t.Run("PostRecoveryStability", func(t *testing.T) {
			for i := 0; i < 5; i++ {
				resp, err := env.MakeRequest("GET", fmt.Sprintf("/proxy/%s/%s", pattern.ProxyType, pattern.Path), nil)
				require.NoError(t, err)
				defer func() { _ = resp.Body.Close() }()

				assert.Equal(t, http.StatusOK, resp.StatusCode, "Request %d should succeed after recovery", i+1)
			}
		})
	})
}

// PerformancePattern 성능 검증 패턴
type PerformancePattern struct {
	ProxyType           string
	Path                string
	MaxResponseTime     time.Duration // 최대 응답 시간
	MinThroughput       int           // 최소 처리량 (requests/second)
	ConcurrentUsers     int           // 동시 사용자 수
	TestDuration        time.Duration // 테스트 지속 시간
	WarmupRequests      int           // 워밍업 요청 수
	AcceptableErrorRate float64       // 허용 가능한 에러율 (0.0-1.0)
}

// TestPerformance 성능 검증 테스트 패턴
func TestPerformance(t *testing.T, env *IntegrationTestEnvironment, pattern PerformancePattern) {
	t.Helper()

	t.Run(fmt.Sprintf("Performance_%s_%s", pattern.ProxyType, pattern.Path), func(t *testing.T) {
		url := fmt.Sprintf("/proxy/%s/%s", pattern.ProxyType, pattern.Path)

		// Step 1: 워밍업 요청
		t.Run("Warmup", func(t *testing.T) {
			if pattern.WarmupRequests <= 0 {
				pattern.WarmupRequests = 5 // 기본값
			}

			for i := 0; i < pattern.WarmupRequests; i++ {
				resp, err := env.MakeRequest("GET", url, nil)
				if err == nil && resp != nil {
					_ = resp.Body.Close()
				}
			}
			t.Logf("Completed %d warmup requests", pattern.WarmupRequests)
		})

		// Step 2: 응답 시간 테스트
		t.Run("ResponseTime", func(t *testing.T) {
			if pattern.MaxResponseTime <= 0 {
				t.Skip("MaxResponseTime not specified, skipping response time test")
			}

			start := time.Now()
			resp, err := env.MakeRequest("GET", url, nil)
			duration := time.Since(start)
			
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.True(t, duration <= pattern.MaxResponseTime,
				"Response time %v should be <= %v", duration, pattern.MaxResponseTime)
			
			t.Logf("Response time: %v", duration)
		})

		// Step 3: 동시성 테스트
		t.Run("Concurrency", func(t *testing.T) {
			if pattern.ConcurrentUsers <= 0 {
				pattern.ConcurrentUsers = 10 // 기본값
			}

			var wg sync.WaitGroup
			results := make([]bool, pattern.ConcurrentUsers)
			responseTimes := make([]time.Duration, pattern.ConcurrentUsers)

			for i := 0; i < pattern.ConcurrentUsers; i++ {
				wg.Add(1)
				go func(index int) {
					defer wg.Done()
					
					start := time.Now()
					resp, err := env.MakeRequest("GET", url, nil)
					responseTimes[index] = time.Since(start)
					
					if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
						results[index] = true
						_ = resp.Body.Close()
					} else {
						results[index] = false
					}
				}(i)
			}

			wg.Wait()

			// 성공률 계산
			successCount := 0
			totalTime := time.Duration(0)
			for i, success := range results {
				if success {
					successCount++
				}
				totalTime += responseTimes[i]
			}

			successRate := float64(successCount) / float64(pattern.ConcurrentUsers)
			avgResponseTime := totalTime / time.Duration(pattern.ConcurrentUsers)
			
			// 에러율 검증
			errorRate := 1.0 - successRate
			if pattern.AcceptableErrorRate > 0 {
				assert.True(t, errorRate <= pattern.AcceptableErrorRate,
					"Error rate %.2f%% should be <= %.2f%%", errorRate*100, pattern.AcceptableErrorRate*100)
			} else {
				assert.True(t, successRate >= 0.95,
					"Success rate %.2f%% should be >= 95%%", successRate*100)
			}

			t.Logf("Concurrent users: %d, Success rate: %.2f%%, Avg response time: %v",
				pattern.ConcurrentUsers, successRate*100, avgResponseTime)
		})

		// Step 4: 지속성 테스트
		t.Run("Sustainability", func(t *testing.T) {
			if pattern.TestDuration <= 0 {
				t.Skip("TestDuration not specified, skipping sustainability test")
			}

			ctx, cancel := context.WithTimeout(context.Background(), pattern.TestDuration)
			defer cancel()

			var totalRequests int
			var successRequests int
			var totalResponseTime time.Duration

			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					goto finished
				case <-ticker.C:
					start := time.Now()
					resp, err := env.MakeRequest("GET", url, nil)
					duration := time.Since(start)
					
					totalRequests++
					totalResponseTime += duration
					
					if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
						successRequests++
						_ = resp.Body.Close()
					}
				}
			}

		finished:
			throughput := float64(successRequests) / pattern.TestDuration.Seconds()
			avgResponseTime := totalResponseTime / time.Duration(totalRequests)
			successRate := float64(successRequests) / float64(totalRequests)

			// 처리량 검증
			if pattern.MinThroughput > 0 {
				assert.True(t, throughput >= float64(pattern.MinThroughput),
					"Throughput %.2f req/s should be >= %d req/s", throughput, pattern.MinThroughput)
			}

			t.Logf("Duration: %v, Total requests: %d, Success rate: %.2f%%, Throughput: %.2f req/s, Avg response time: %v",
				pattern.TestDuration, totalRequests, successRate*100, throughput, avgResponseTime)
		})
	})
}

// ProxyTestSuite 모든 프록시 타입에 대한 공통 테스트 스위트
type ProxyTestSuite struct {
	ProxyTypes []string
	TestPaths  map[string]string // ProxyType -> TestPath 매핑
}

// RunCommonTestSuite 공통 테스트 스위트 실행
func (pts *ProxyTestSuite) RunCommonTestSuite(t *testing.T, env *IntegrationTestEnvironment) {
	t.Helper()

	for _, proxyType := range pts.ProxyTypes {
		testPath, exists := pts.TestPaths[proxyType]
		if !exists {
			continue
		}

		t.Run(fmt.Sprintf("CommonSuite_%s", proxyType), func(t *testing.T) {
			// 1. 캐시 시나리오 테스트
			cachePattern := CacheScenarioPattern{
				ProxyType:    proxyType,
				Path:         testPath,
				ExpectedMiss: "mock", // 모든 mock 서버가 "mock" 문자열을 포함하는 응답 반환
				ExpectedHit:  "mock",
				CacheTTL:     time.Second,
			}
			TestCacheScenario(t, env, cachePattern)

			// 2. 에러 복구 테스트
			errorPattern := ErrorRecoveryPattern{
				ProxyType:        proxyType,
				Path:             "nonexistent/path",
				ExpectedError:    http.StatusNotFound,
				ExpectedRecovery: "mock",
			}
			TestErrorRecovery(t, env, errorPattern)

			// 3. 성능 테스트 (기본 설정)
			perfPattern := PerformancePattern{
				ProxyType:           proxyType,
				Path:                testPath,
				MaxResponseTime:     500 * time.Millisecond,
				ConcurrentUsers:     5,
				TestDuration:        2 * time.Second,
				AcceptableErrorRate: 0.05, // 5% 에러율 허용
			}
			TestPerformance(t, env, perfPattern)
		})
	}
}

// CreateDefaultTestSuite 기본 테스트 스위트 생성
func CreateDefaultTestSuite() *ProxyTestSuite {
	return &ProxyTestSuite{
		ProxyTypes: []string{"npm", "maven", "apt", "apk", "yum", "docker"},
		TestPaths: map[string]string{
			"npm":    "express",
			"maven":  "junit/junit/4.13.2/junit-4.13.2.pom",
			"apt":    "dists/jammy/Release",
			"apk":    "alpine/v3.16/main/x86_64/APKINDEX.tar.gz",
			"yum":    "centos/8/BaseOS/x86_64/os/repodata/repomd.xml",
			"docker": "v2/library/nginx/manifests/latest",
		},
	}
}

// TestEnvironment 호환성 함수 - NewTestEnvironment와 SetupIntegrationTest 통합
func NewTestEnvironment(t *testing.T) *IntegrationTestEnvironment {
	return SetupIntegrationTest(t)
}

// HealthCheckPattern 헬스체크 공통 패턴
type HealthCheckPattern struct {
	Endpoint         string
	ExpectedStatus   int
	ExpectedContent  string
	MaxResponseTime  time.Duration
	RequiredHeaders  map[string]string
}

// TestHealthCheck 헬스체크 테스트 패턴
func TestHealthCheck(t *testing.T, env *IntegrationTestEnvironment, pattern HealthCheckPattern) {
	t.Helper()

	t.Run("HealthCheck", func(t *testing.T) {
		start := time.Now()
		resp, err := env.MakeRequest("GET", pattern.Endpoint, nil)
		duration := time.Since(start)
		
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// 상태 코드 검증
		expectedStatus := pattern.ExpectedStatus
		if expectedStatus == 0 {
			expectedStatus = http.StatusOK
		}
		assert.Equal(t, expectedStatus, resp.StatusCode)

		// 응답 시간 검증
		if pattern.MaxResponseTime > 0 {
			assert.True(t, duration <= pattern.MaxResponseTime,
				"Health check response time %v should be <= %v", duration, pattern.MaxResponseTime)
		}

		// 필수 헤더 검증
		for header, expectedValue := range pattern.RequiredHeaders {
			actualValue := resp.Header.Get(header)
			assert.Equal(t, expectedValue, actualValue, "Header %s should be %s", header, expectedValue)
		}

		// 응답 내용 검증
		if pattern.ExpectedContent != "" {
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Contains(t, string(body), pattern.ExpectedContent)
		}

		t.Logf("Health check response time: %v", duration)
	})
}