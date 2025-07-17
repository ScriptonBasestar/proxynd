package integration

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("부하 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	err := server.WaitForReady()
	require.NoError(t, err)

	const (
		concurrency            = 20  // 동시 연결 수
		totalRequests          = 200 // 총 요청 수
		requestsPerGoroutine   = totalRequests / concurrency
		maxAcceptableErrorRate = 5.0             // 최대 허용 에러율 (%)
		maxAverageLatency      = 1 * time.Second // 최대 평균 응답 시간
	)

	var (
		successCount int64
		errorCount   int64
		totalLatency int64 // 나노초 단위
	)

	results := make(chan LoadTestResult, totalRequests)
	var wg sync.WaitGroup

	// When: 동시 요청 실행
	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			client := &http.Client{
				Timeout: 30 * time.Second,
			}

			for j := 0; j < requestsPerGoroutine; j++ {
				result := performLoadTestRequest(client, server.ProxyURL("apt", "dists/focal/Release"))
				results <- result

				if result.Error != nil {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
					atomic.AddInt64(&totalLatency, result.Duration.Nanoseconds())
				}

				// 약간의 지연으로 서버 부하 조절
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	close(results)

	totalTime := time.Since(start)

	// Then: 결과 분석
	var minDuration, maxDuration time.Duration
	var durations []time.Duration

	for result := range results {
		durations = append(durations, result.Duration)

		if minDuration == 0 || result.Duration < minDuration {
			minDuration = result.Duration
		}
		if result.Duration > maxDuration {
			maxDuration = result.Duration
		}
	}

	// 통계 계산
	successCountVal := atomic.LoadInt64(&successCount)
	errorCountVal := atomic.LoadInt64(&errorCount)
	totalLatencyVal := atomic.LoadInt64(&totalLatency)

	errorRate := float64(errorCountVal) / float64(totalRequests) * 100
	avgDuration := time.Duration(totalLatencyVal / successCountVal)
	rps := float64(totalRequests) / totalTime.Seconds()

	// 결과 로깅
	t.Logf("=== 부하 테스트 결과 ===")
	t.Logf("총 요청: %d, 성공: %d, 실패: %d (%.2f%%)",
		totalRequests, successCountVal, errorCountVal, errorRate)
	t.Logf("성능: RPS=%.2f, 평균=%.2fms, 최소=%.2fms, 최대=%.2fms",
		rps, float64(avgDuration.Nanoseconds())/1e6,
		float64(minDuration.Nanoseconds())/1e6,
		float64(maxDuration.Nanoseconds())/1e6)
	t.Logf("전체 실행 시간: %v", totalTime)

	// 성능 기준 검증
	assert.Less(t, errorRate, maxAcceptableErrorRate,
		"에러율이 %.1f%%를 초과했습니다", maxAcceptableErrorRate)
	assert.Less(t, avgDuration, maxAverageLatency,
		"평균 응답 시간이 %v를 초과했습니다", maxAverageLatency)
	assert.Greater(t, rps, 50.0,
		"RPS가 최소 기준(50)을 하회했습니다")
}

func TestStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("스트레스 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	err := server.WaitForReady()
	require.NoError(t, err)

	const (
		duration    = 30 * time.Second // 테스트 지속 시간
		concurrency = 10               // 동시 연결 수
	)

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	var (
		totalRequests int64
		successCount  int64
		errorCount    int64
	)

	var wg sync.WaitGroup

	// When: 지속적인 부하 생성
	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			client := &http.Client{
				Timeout: 10 * time.Second,
			}

			for {
				select {
				case <-ctx.Done():
					return
				default:
					atomic.AddInt64(&totalRequests, 1)

					result := performLoadTestRequest(client, server.ProxyURL("apt", "dists/focal/Release"))

					if result.Error != nil {
						atomic.AddInt64(&errorCount, 1)
					} else {
						atomic.AddInt64(&successCount, 1)
					}

					// 짧은 지연
					time.Sleep(50 * time.Millisecond)
				}
			}
		}(i)
	}

	wg.Wait()
	totalTime := time.Since(start)

	// Then: 결과 분석
	totalReq := atomic.LoadInt64(&totalRequests)
	successReq := atomic.LoadInt64(&successCount)
	errorReq := atomic.LoadInt64(&errorCount)

	errorRate := float64(errorReq) / float64(totalReq) * 100
	rps := float64(totalReq) / totalTime.Seconds()

	t.Logf("=== 스트레스 테스트 결과 ===")
	t.Logf("지속 시간: %v", totalTime)
	t.Logf("총 요청: %d, 성공: %d, 실패: %d (%.2f%%)",
		totalReq, successReq, errorReq, errorRate)
	t.Logf("평균 RPS: %.2f", rps)

	// 기본 안정성 검증
	assert.Less(t, errorRate, 10.0, "장기간 부하에서 에러율이 10%를 초과했습니다")
	assert.Greater(t, totalReq, int64(100), "최소 100개의 요청이 처리되어야 합니다")
}

func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("메모리 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	err := server.WaitForReady()
	require.NoError(t, err)

	// 초기 메모리 사용량 측정
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	var initialStats runtime.MemStats
	runtime.ReadMemStats(&initialStats)

	// When: 많은 요청 수행
	const requestCount = 500
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	for i := 0; i < requestCount; i++ {
		resp, err := client.Get(server.ProxyURL("apt", "dists/focal/Release"))
		if err != nil {
			t.Logf("Request %d failed: %v", i, err)
			continue
		}
		resp.Body.Close()

		// 주기적으로 GC 실행
		if i%100 == 0 {
			runtime.GC()
		}
	}

	// 메모리 정리
	runtime.GC()
	runtime.GC() // 두 번 실행으로 확실한 정리
	time.Sleep(200 * time.Millisecond)

	// 최종 메모리 사용량 측정
	var finalStats runtime.MemStats
	runtime.ReadMemStats(&finalStats)

	// Then: 메모리 사용량 검증
	memoryIncrease := int64(finalStats.HeapAlloc) - int64(initialStats.HeapAlloc)

	t.Logf("=== 메모리 사용량 분석 ===")
	t.Logf("초기 메모리: %s", formatBytes(initialStats.HeapAlloc))
	t.Logf("최종 메모리: %s", formatBytes(finalStats.HeapAlloc))
	t.Logf("메모리 증가: %s", formatBytes(uint64(memoryIncrease)))
	t.Logf("시스템 메모리: %s", formatBytes(finalStats.Sys))
	t.Logf("GC 실행 횟수: %d", finalStats.NumGC-initialStats.NumGC)

	// 메모리 누수 검사 (20MB 이하 증가 허용)
	maxAllowedIncrease := int64(20 * 1024 * 1024) // 20MB
	if memoryIncrease > maxAllowedIncrease {
		t.Errorf("메모리 증가량이 허용 기준을 초과했습니다: %s > %s",
			formatBytes(uint64(memoryIncrease)), formatBytes(uint64(maxAllowedIncrease)))
	}

	// 힙 사용률 검사
	heapUsage := float64(finalStats.HeapAlloc) / float64(finalStats.HeapSys) * 100
	t.Logf("힙 사용률: %.2f%%", heapUsage)
}

func TestResourceCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("리소스 정리 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	err := server.WaitForReady()
	require.NoError(t, err)

	// When: 다양한 요청으로 리소스 사용
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	urls := []string{
		server.ProxyURL("apt", "dists/focal/Release"),
		server.ProxyURL("maven", "org/springframework/spring-core/5.3.21/spring-core-5.3.21.pom"),
		server.ProxyURL("apt", "dists/focal/main/binary-amd64/Packages"),
	}

	// 고루틴 수 추적
	initialGoroutines := runtime.NumGoroutine()

	for i := 0; i < 100; i++ {
		url := urls[i%len(urls)]
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		resp.Body.Close()
	}

	// 잠시 대기 후 고루틴 수 확인
	time.Sleep(200 * time.Millisecond)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()

	// Then: 리소스 정리 확인
	goroutineIncrease := finalGoroutines - initialGoroutines

	t.Logf("=== 리소스 정리 분석 ===")
	t.Logf("초기 고루틴: %d", initialGoroutines)
	t.Logf("최종 고루틴: %d", finalGoroutines)
	t.Logf("고루틴 증가: %d", goroutineIncrease)

	// 고루틴 누수 검사 (최대 10개 증가 허용)
	assert.Less(t, goroutineIncrease, 10,
		"고루틴이 과도하게 증가했습니다. 리소스 누수 가능성이 있습니다.")
}

// LoadTestResult 부하 테스트 결과
type LoadTestResult struct {
	StatusCode int
	Duration   time.Duration
	Error      error
	Size       int64
}

// performLoadTestRequest 부하 테스트용 요청 수행
func performLoadTestRequest(client *http.Client, url string) LoadTestResult {
	start := time.Now()

	resp, err := client.Get(url)
	if err != nil {
		return LoadTestResult{
			Duration: time.Since(start),
			Error:    err,
		}
	}
	defer resp.Body.Close()

	// 응답 크기 측정을 위해 본문 읽기
	size := int64(0)
	buffer := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buffer)
		size += int64(n)
		if err != nil {
			break
		}
	}

	return LoadTestResult{
		StatusCode: resp.StatusCode,
		Duration:   time.Since(start),
		Size:       size,
	}
}

// formatBytes 바이트 크기를 읽기 쉬운 형태로 포맷
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
