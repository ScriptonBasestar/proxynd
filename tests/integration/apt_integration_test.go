// Package integration provides integration test suites
package integration

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPTProxy_FullFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	// 서버 준비 대기
	err := server.WaitForReady()
	require.NoError(t, err, "서버 준비 실패")

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectCached   bool
		contentCheck   func(string) bool
	}{
		{
			name:           "Release 파일 요청",
			path:           "dists/focal/Release",
			expectedStatus: 200,
			expectCached:   false, // 테스트 환경에서는 캐시 미구현
			contentCheck: func(content string) bool {
				return strings.Contains(content, "Origin: Ubuntu") &&
					strings.Contains(content, "Suite: focal")
			},
		},
		{
			name:           "Packages 파일 요청",
			path:           "dists/focal/main/binary-amd64/Packages",
			expectedStatus: 200,
			expectCached:   false,
			contentCheck: func(content string) bool {
				return strings.Contains(content, "Package:") &&
					strings.Contains(content, "Version:")
			},
		},
		{
			name:           "존재하지 않는 패키지",
			path:           "pool/main/nonexistent/package.deb",
			expectedStatus: 404,
			expectCached:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: 첫 번째 요청
			url := server.ProxyURL("apt", tt.path)
			resp, err := http.Get(url)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			// Then: 응답 검증
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == 200 {
				assert.Equal(t, "apt", resp.Header.Get("X-Proxy-Type"))

				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)

				if tt.contentCheck != nil {
					assert.True(t, tt.contentCheck(string(body)),
						"Content check failed for %s", tt.name)
				}

				// 캐시 확인을 위한 두 번째 요청 (캐시 구현 시)
				if tt.expectCached {
					resp2, err := http.Get(url)
					require.NoError(t, err)
					defer func() { _ = resp2.Body.Close() }()

					assert.Equal(t, 200, resp2.StatusCode)
					assert.Equal(t, "HIT", resp2.Header.Get("X-Cache-Status"))
				}
			}
		})
	}
}

func TestAPTProxy_Headers(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	err := server.WaitForReady()
	require.NoError(t, err)

	// When
	url := server.ProxyURL("apt", "dists/focal/Release")
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Then: 헤더 검증
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "apt", resp.Header.Get("X-Proxy-Type"))
	assert.Contains(t, []string{"HIT", "MISS"}, resp.Header.Get("X-Cache-Status"))
}

func TestAPTProxy_ConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	err := server.WaitForReady()
	require.NoError(t, err)

	const concurrency = 10
	const requestsPerWorker = 5

	results := make(chan TestResult, concurrency*requestsPerWorker)
	done := make(chan bool)

	// When: 동시 요청 실행
	for i := 0; i < concurrency; i++ {
		go func() {
			defer func() { done <- true }()

			for j := 0; j < requestsPerWorker; j++ {
				url := server.ProxyURL("apt", "dists/focal/Release")
				result := performRequest(url)
				results <- result
			}
		}()
	}

	// 모든 워커 완료 대기
	for i := 0; i < concurrency; i++ {
		<-done
	}
	close(results)

	// Then: 결과 분석
	var successCount, errorCount int
	var totalDuration time.Duration

	for result := range results {
		if result.Error != nil {
			errorCount++
			t.Logf("Request failed: %v", result.Error)
		} else {
			successCount++
			totalDuration += result.Duration
			assert.Equal(t, 200, result.StatusCode)
		}
	}

	totalRequests := concurrency * requestsPerWorker
	errorRate := float64(errorCount) / float64(totalRequests) * 100

	t.Logf("Results: Total=%d, Success=%d, Error=%d (%.2f%%)",
		totalRequests, successCount, errorCount, errorRate)

	if successCount > 0 {
		avgDuration := totalDuration / time.Duration(successCount)
		t.Logf("Average request duration: %v", avgDuration)

		// 성능 기준
		assert.Less(t, avgDuration, 2*time.Second, "평균 응답 시간이 너무 깁니다")
	}

	// 에러율 검증
	assert.Less(t, errorRate, 10.0, "에러율이 10%를 초과했습니다")
}

func TestAPTProxy_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	err := server.WaitForReady()
	require.NoError(t, err)

	// When: 존재하지 않는 경로 요청
	url := server.ProxyURL("apt", "invalid/path/that/does/not/exist")
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Then: 적절한 에러 응답
	assert.Equal(t, 404, resp.StatusCode)
	assert.Equal(t, "apt", resp.Header.Get("X-Proxy-Type"))
}

func TestAPTProxy_LargeResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	err := server.WaitForReady()
	require.NoError(t, err)

	// When: 큰 응답이 예상되는 요청
	url := server.ProxyURL("apt", "dists/focal/main/binary-amd64/Packages")
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Then: 응답 처리 확인
	assert.Equal(t, 200, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// 최소한의 콘텐츠 확인
	assert.True(t, len(body) > 0, "응답 본문이 비어있습니다")
	assert.Contains(t, string(body), "Package:", "패키지 정보가 포함되어야 합니다")
}

// TestResult 테스트 결과 구조체
type TestResult struct {
	StatusCode int
	Duration   time.Duration
	Error      error
}

// performRequest HTTP 요청 수행 및 결과 반환
func performRequest(url string) TestResult {
	start := time.Now()

	resp, err := http.Get(url)
	if err != nil {
		return TestResult{
			Duration: time.Since(start),
			Error:    err,
		}
	}
	defer func() { _ = resp.Body.Close() }()

	// 응답 본문 읽기 (실제 처리 시뮬레이션)
	_, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return TestResult{
			StatusCode: resp.StatusCode,
			Duration:   time.Since(start),
			Error:      readErr,
		}
	}

	return TestResult{
		StatusCode: resp.StatusCode,
		Duration:   time.Since(start),
	}
}
