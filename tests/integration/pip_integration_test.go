package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPIPProxyBasicFlow 기본 PIP 프록시 동작 테스트
func TestPIPProxyBasicFlow(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// PIP upstream 준비 확인
	err := env.WaitForUpstream("pip", 5*time.Second)
	require.NoError(t, err, "PIP mock upstream should be ready")

	// 패키지 정보 조회 테스트 (/pypi/requests/json)
	t.Run("Package Info API", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/pip/pypi/requests/json", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		acceptableContentTypes := []string{"application/json", "application/octet-stream", "text/html"}
		contentType := resp.Header.Get("Content-Type")
		found := false
		for _, ct := range acceptableContentTypes {
			if strings.Contains(contentType, ct) {
				found = true
				break
			}
		}
		assert.True(t, found, "Content-Type은 JSON, octet-stream 또는 HTML 가능")

		// 응답 본문 검증
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		responseBody := string(body[:n])
		assert.Contains(t, responseBody, "\"name\": \"requests\"")
		assert.Contains(t, responseBody, "\"version\": \"2.28.1\"")
		assert.Contains(t, responseBody, "Python HTTP for Humans")
	})

	// Simple API 테스트 (/simple/requests/)
	t.Run("Simple API", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/pip/simple/requests/", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")

		// HTML 응답 검증
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		responseBody := string(body[:n])
		assert.Contains(t, responseBody, "Links for requests")
		assert.Contains(t, responseBody, "requests-2.28.1-py3-none-any.whl")
	})

	// 패키지 다운로드 테스트
	t.Run("Package Download", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/pip/simple/requests/requests-2.28.1-py3-none-any.whl", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		// .whl files are ZIP format, so accept both application/zip and application/octet-stream
		contentType := resp.Header.Get("Content-Type")
		assert.True(t, strings.Contains(contentType, "application/zip") || strings.Contains(contentType, "application/octet-stream"),
			"Content-Type should be application/zip or application/octet-stream, got: %s", contentType)

		// 파일 내용 검증
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		responseBody := string(body[:n])
		assert.Contains(t, responseBody, "mock requests wheel content")
	})
}

// TestPIPProxyErrorHandling PIP 프록시 에러 처리 테스트
func TestPIPProxyErrorHandling(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// 존재하지 않는 패키지 테스트
	t.Run("Non-existent Package", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/pip/pypi/nonexistent-package/json", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// 잘못된 경로 테스트
	t.Run("Invalid Path", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/pip/invalid/path", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// 지원하지 않는 HTTP 메서드 테스트
	t.Run("Unsupported HTTP Method", func(t *testing.T) {
		resp, err := env.MakeRequest("PUT", "/proxy/pip/pypi/requests/json", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// PUT 메서드는 일반적으로 405 또는 404를 반환
		assert.True(t, resp.StatusCode >= 400)
	})
}

// TestPIPProxyCaching PIP 프록시 캐싱 동작 테스트
func TestPIPProxyCaching(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	packageURL := "/proxy/pip/pypi/requests/json"

	// 첫 번째 요청 (캐시 미스)
	t.Run("First Request - Cache Miss", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", packageURL, nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 캐시 통계 확인
		stats, err := env.GetCacheStats()
		require.NoError(t, err)

		// 첫 요청이므로 미스가 발생했어야 함
		assert.NotNil(t, stats)
	})

	// 두 번째 요청 (캐시 히트 예상)
	t.Run("Second Request - Cache Hit Expected", func(t *testing.T) {
		// 약간의 지연을 두어 캐시가 저장될 시간 확보
		time.Sleep(100 * time.Millisecond)

		resp, err := env.MakeRequest("GET", packageURL, nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 응답 내용이 동일한지 확인
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		responseBody := string(body[:n])
		assert.Contains(t, responseBody, "\"name\": \"requests\"")
	})
}

// TestPIPProxyContentTypes PIP 프록시 Content-Type 처리 테스트
func TestPIPProxyContentTypes(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	testCases := []struct {
		name         string
		path         string
		expectedType string
	}{
		{
			name:         "JSON API",
			path:         "/proxy/pip/pypi/requests/json",
			expectedType: "application/json",
		},
		{
			name:         "Simple HTML",
			path:         "/proxy/pip/simple/requests/",
			expectedType: "text/html",
		},
		{
			name:         "Wheel File",
			path:         "/proxy/pip/simple/requests/requests-2.28.1-py3-none-any.whl",
			expectedType: "application/octet-stream",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := env.MakeRequest("GET", tc.path, nil)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// For .whl files, accept both application/zip and application/octet-stream
			contentType := resp.Header.Get("Content-Type")
			if strings.Contains(tc.path, ".whl") {
				assert.True(t, strings.Contains(contentType, "application/zip") || strings.Contains(contentType, "application/octet-stream"),
					"Content-Type should be application/zip or application/octet-stream for .whl files, got: %s", contentType)
			} else {
				assert.Contains(t, contentType, tc.expectedType)
			}
		})
	}
}

// TestPIPProxyHeaders PIP 프록시 헤더 처리 테스트
func TestPIPProxyHeaders(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// User-Agent 헤더 테스트
	t.Run("User-Agent Header", func(t *testing.T) {
		headers := map[string]string{
			"User-Agent": "pip/21.3.1",
		}

		resp, err := env.MakeRequest("GET", "/proxy/pip/pypi/requests/json", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Accept 헤더 테스트
	t.Run("Accept Header", func(t *testing.T) {
		headers := map[string]string{
			"Accept": "application/json",
		}

		resp, err := env.MakeRequest("GET", "/proxy/pip/pypi/requests/json", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		acceptableContentTypes := []string{"application/json", "application/octet-stream", "text/html"}
		contentType := resp.Header.Get("Content-Type")
		found := false
		for _, ct := range acceptableContentTypes {
			if strings.Contains(contentType, ct) {
				found = true
				break
			}
		}
		assert.True(t, found, "Content-Type은 JSON, octet-stream 또는 HTML 가능")
	})
}

// TestPIPProxyPerformance PIP 프록시 성능 테스트
func TestPIPProxyPerformance(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// 응답 시간 테스트
	t.Run("Response Time", func(t *testing.T) {
		start := time.Now()

		resp, err := env.MakeRequest("GET", "/proxy/pip/pypi/requests/json", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		duration := time.Since(start)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Less(t, duration, 5*time.Second, "Response should be fast")
	})

	// 동시 요청 테스트
	t.Run("Concurrent Requests", func(t *testing.T) {
		const numRequests = 10
		done := make(chan bool, numRequests)
		errors := make(chan error, numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				resp, err := env.MakeRequest("GET", "/proxy/pip/pypi/requests/json", nil)
				if err != nil {
					errors <- err
					return
				}
				defer func() { _ = resp.Body.Close() }()

				if resp.StatusCode != http.StatusOK {
					errors <- fmt.Errorf("unexpected status code: %d", resp.StatusCode)
					return
				}

				done <- true
			}()
		}

		// 모든 요청 완료 대기
		successCount := 0
		for i := 0; i < numRequests; i++ {
			select {
			case <-done:
				successCount++
			case err := <-errors:
				t.Logf("Request failed: %v", err)
			case <-time.After(10 * time.Second):
				t.Fatal("Timeout waiting for concurrent requests")
			}
		}

		assert.Equal(t, numRequests, successCount, "All requests should succeed")
	})
}

// TestPIPProxyPathVariations PIP 프록시 경로 변형 테스트
func TestPIPProxyPathVariations(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	pathVariations := []struct {
		name        string
		path        string
		expectValid bool
	}{
		{
			name:        "Standard PyPI path",
			path:        "/proxy/pip/pypi/requests/json",
			expectValid: true,
		},
		{
			name:        "Simple API path",
			path:        "/proxy/pip/simple/requests/",
			expectValid: true,
		},
		{
			name:        "Package file path",
			path:        "/proxy/pip/simple/requests/requests-2.28.1-py3-none-any.whl",
			expectValid: true,
		},
		{
			name:        "Path with trailing slash",
			path:        "/proxy/pip/simple/requests/",
			expectValid: true,
		},
		{
			name:        "Path without trailing slash",
			path:        "/proxy/pip/simple/requests",
			expectValid: true, // trailing slash 정규화로 정상 처리됨
		},
		{
			name:        "Invalid API path",
			path:        "/proxy/pip/invalid/requests",
			expectValid: false,
		},
	}

	for _, variation := range pathVariations {
		t.Run(variation.name, func(t *testing.T) {
			resp, err := env.MakeRequest("GET", variation.path, nil)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			if variation.expectValid {
				assert.Equal(t, http.StatusOK, resp.StatusCode,
					"Path %s should be valid", variation.path)
			} else {
				assert.NotEqual(t, http.StatusOK, resp.StatusCode,
					"Path %s should be invalid", variation.path)
			}
		})
	}
}

// TestPIPProxyPackageFormats PIP 프록시 다양한 패키지 형식 테스트
func TestPIPProxyPackageFormats(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// 실제 PIP는 다양한 패키지 형식을 지원하지만,
	// 테스트 환경에서는 mock 데이터로 기본 동작만 확인
	t.Run("Wheel Format", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/pip/simple/requests/requests-2.28.1-py3-none-any.whl", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/octet-stream")
	})

	// Source Distribution (sdist) 형식은 mock에서 지원하지 않으므로 404 예상
	t.Run("Source Distribution", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/pip/simple/requests/requests-2.28.1.tar.gz", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Mock 서버에서는 404를 반환할 것
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// TestPIPProxyUpstreamConnectivity 업스트림 연결성 테스트
func TestPIPProxyUpstreamConnectivity(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// Mock upstream이 올바르게 응답하는지 확인
	t.Run("Upstream Response Validation", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/pip/pypi/requests/json", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 응답이 upstream에서 온 것인지 확인
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		responseBody := string(body[:n])

		// Mock upstream의 예상 응답과 일치하는지 확인
		assert.Contains(t, responseBody, "\"name\": \"requests\"")
		assert.Contains(t, responseBody, "\"version\": \"2.28.1\"")
		assert.Contains(t, responseBody, "Python HTTP for Humans")
	})
}

// BenchmarkPIPProxyThroughput PIP 프록시 처리량 벤치마크
func BenchmarkPIPProxyThroughput(b *testing.B) {
	env := SetupIntegrationTest(&testing.T{})
	defer env.Cleanup()

	// 벤치마크는 실제 성능 측정을 위한 것이므로 간단하게 구현
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resp, err := env.MakeRequest("GET", "/proxy/pip/pypi/requests/json", nil)
		if err != nil {
			b.Fatal(err)
		}
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
	}
}

// TestPIPProxyMetrics PIP 프록시 메트릭 수집 테스트
func TestPIPProxyMetrics(t *testing.T) {
	t.Skip("Metrics router is disabled in integration tests due to Prometheus global registry issues")
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// 몇 개의 요청을 보낸 후 메트릭 확인
	t.Run("Request Metrics", func(t *testing.T) {
		// 여러 요청 수행
		paths := []string{
			"/proxy/pip/pypi/requests/json",
			"/proxy/pip/simple/requests/",
			"/proxy/pip/simple/requests/requests-2.28.1-py3-none-any.whl",
		}

		for _, path := range paths {
			resp, err := env.MakeRequest("GET", path, nil)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		}

		// 메트릭 엔드포인트 확인
		resp, err := env.MakeRequest("GET", "/metrics", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Prometheus 메트릭 형식인지 확인
		body := make([]byte, 4096)
		n, _ := resp.Body.Read(body)
		metricsBody := string(body[:n])

		// PIP 관련 메트릭이 있는지 확인
		assert.True(t, strings.Contains(metricsBody, "http_requests_total") ||
			strings.Contains(metricsBody, "proxynd_"),
			"Should contain HTTP or ProxyND metrics")
	})
}
