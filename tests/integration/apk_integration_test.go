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

// TestAPKProxyBasicFlow 기본 APK 프록시 동작 테스트
func TestAPKProxyBasicFlow(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// APK upstream 준비 확인
	err := env.WaitForUpstream("apk", 5*time.Second)
	require.NoError(t, err, "APK mock upstream should be ready")

	// Package index (APKINDEX.tar.gz) 테스트
	t.Run("Package Index", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/x-gzip")

		// APKINDEX.tar.gz 응답 검증
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		responseBody := string(body[:n])
		assert.Contains(t, responseBody, "mock APKINDEX content")
	})

	// APK 패키지 다운로드 테스트
	t.Run("APK Package Download", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/nginx-1.22.0-r1.apk", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/vnd.alpine.apk")

		// APK 파일 내용 검증
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		responseBody := string(body[:n])
		assert.Contains(t, responseBody, "mock apk content")
	})
}

// TestAPKProxyErrorHandling APK 프록시 에러 처리 테스트
func TestAPKProxyErrorHandling(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// 존재하지 않는 저장소 테스트
	t.Run("Non-existent Repository", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/apk/nonexistent/v1.0/main/x86_64/APKINDEX.tar.gz", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// 존재하지 않는 패키지 테스트
	t.Run("Non-existent Package", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/nonexistent-package.apk", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// 잘못된 Alpine 버전 테스트
	t.Run("Invalid Alpine Version", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v99.99/main/x86_64/APKINDEX.tar.gz", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// 잘못된 아키텍처 테스트
	t.Run("Invalid Architecture", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/invalid-arch/APKINDEX.tar.gz", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// 지원하지 않는 HTTP 메서드 테스트
	t.Run("Unsupported HTTP Method", func(t *testing.T) {
		resp, err := env.MakeRequest("PUT", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// PUT 메서드는 일반적으로 405 또는 404를 반환
		assert.True(t, resp.StatusCode >= 400)
	})
}

// TestAPKProxyCaching APK 프록시 캐싱 동작 테스트
func TestAPKProxyCaching(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	indexURL := "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz"

	// 첫 번째 요청 (캐시 미스)
	t.Run("First Request - Cache Miss", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", indexURL, nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 캐시 통계 확인
		stats, err := env.GetCacheStats()
		require.NoError(t, err)
		assert.NotNil(t, stats)
	})

	// 두 번째 요청 (캐시 히트 예상)
	t.Run("Second Request - Cache Hit Expected", func(t *testing.T) {
		time.Sleep(100 * time.Millisecond)

		resp, err := env.MakeRequest("GET", indexURL, nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 응답 내용이 동일한지 확인
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		responseBody := string(body[:n])
		assert.Contains(t, responseBody, "mock APKINDEX content")
	})
}

// TestAPKProxyContentTypes APK 프록시 Content-Type 처리 테스트
func TestAPKProxyContentTypes(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	testCases := []struct {
		name         string
		path         string
		expectedType string
	}{
		{
			name:         "APKINDEX Gzip",
			path:         "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz",
			expectedType: "application/x-gzip",
		},
		{
			name:         "APK Package",
			path:         "/proxy/apk/alpine/v3.16/main/x86_64/nginx-1.22.0-r1.apk",
			expectedType: "application/vnd.alpine.apk",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := env.MakeRequest("GET", tc.path, nil)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Contains(t, resp.Header.Get("Content-Type"), tc.expectedType)
		})
	}
}

// TestAPKProxyHeaders APK 프록시 헤더 처리 테스트
func TestAPKProxyHeaders(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// APK User-Agent 헤더 테스트
	t.Run("APK User-Agent", func(t *testing.T) {
		headers := map[string]string{
			"User-Agent": "apk-tools/2.12.7",
		}

		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// If-Modified-Since 헤더 테스트
	t.Run("If-Modified-Since Header", func(t *testing.T) {
		headers := map[string]string{
			"If-Modified-Since": "Wed, 21 Oct 2015 07:28:00 GMT",
		}

		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Mock 서버는 조건부 요청을 처리하지 않으므로 200 응답 예상
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Range 헤더 테스트 (부분 다운로드)
	t.Run("Range Header", func(t *testing.T) {
		headers := map[string]string{
			"Range": "bytes=0-1023",
		}

		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/nginx-1.22.0-r1.apk", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Mock 서버는 Range를 지원하지 않으므로 200 응답 (전체 내용)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Accept-Encoding 헤더 테스트
	t.Run("Accept-Encoding Header", func(t *testing.T) {
		headers := map[string]string{
			"Accept-Encoding": "gzip, deflate",
		}

		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// TestAPKProxyPerformance APK 프록시 성능 테스트
func TestAPKProxyPerformance(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// 응답 시간 테스트
	t.Run("Response Time", func(t *testing.T) {
		start := time.Now()

		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		duration := time.Since(start)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Less(t, duration, 5*time.Second, "Response should be fast")
	})

	// 동시 요청 테스트
	t.Run("Concurrent Requests", func(t *testing.T) {
		const numRequests = 5
		done := make(chan bool, numRequests)
		errors := make(chan error, numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", nil)
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

// TestAPKProxyPathVariations APK 프록시 경로 변형 테스트
func TestAPKProxyPathVariations(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	pathVariations := []struct {
		name        string
		path        string
		expectValid bool
	}{
		{
			name:        "Standard APKINDEX path",
			path:        "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz",
			expectValid: true,
		},
		{
			name:        "APK package path",
			path:        "/proxy/apk/alpine/v3.16/main/x86_64/nginx-1.22.0-r1.apk",
			expectValid: true,
		},
		{
			name:        "Different Alpine version",
			path:        "/proxy/apk/alpine/v3.15/main/x86_64/APKINDEX.tar.gz",
			expectValid: false, // Mock에서 지원하지 않음
		},
		{
			name:        "Community repository",
			path:        "/proxy/apk/alpine/v3.16/community/x86_64/APKINDEX.tar.gz",
			expectValid: false, // Mock에서 지원하지 않음
		},
		{
			name:        "Different architecture",
			path:        "/proxy/apk/alpine/v3.16/main/aarch64/APKINDEX.tar.gz",
			expectValid: false, // Mock에서 지원하지 않음
		},
		{
			name:        "Invalid path structure",
			path:        "/proxy/apk/invalid/path",
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

// TestAPKProxyRepositoryStructure APK 저장소 구조 테스트
func TestAPKProxyRepositoryStructure(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// APKINDEX.tar.gz 파일 구조 테스트
	t.Run("APKINDEX Structure", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/x-gzip")

		// 파일 크기가 0보다 큰지 확인
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		assert.Greater(t, n, 0, "APKINDEX should have content")
	})

	// 패키지 파일 구조 테스트
	t.Run("Package File Structure", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/nginx-1.22.0-r1.apk", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/vnd.alpine.apk")

		// APK 파일이 실제 내용을 가지고 있는지 확인
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		assert.Greater(t, n, 0, "APK package should have content")
	})
}

// TestAPKProxyVersionHandling APK 버전 처리 테스트
func TestAPKProxyVersionHandling(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// 지원되는 Alpine 버전 테스트
	t.Run("Supported Alpine Version", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// 지원되지 않는 Alpine 버전 테스트
	t.Run("Unsupported Alpine Version", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v2.0/main/x86_64/APKINDEX.tar.gz", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// Edge 버전 테스트 (실제로는 mock에서 지원하지 않음)
	t.Run("Alpine Edge Version", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/edge/main/x86_64/APKINDEX.tar.gz", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// TestAPKProxyUpstreamConnectivity 업스트림 연결성 테스트
func TestAPKProxyUpstreamConnectivity(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// Mock upstream이 올바르게 응답하는지 확인
	t.Run("Upstream Response Validation", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 응답이 upstream에서 온 것인지 확인
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		responseBody := string(body[:n])

		// Mock upstream의 예상 응답과 일치하는지 확인
		assert.Contains(t, responseBody, "mock APKINDEX content")
	})

	// 패키지 파일 upstream 검증
	t.Run("Package Upstream Validation", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/nginx-1.22.0-r1.apk", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		responseBody := string(body[:n])

		// Mock upstream의 패키지 응답 검증
		assert.Contains(t, responseBody, "mock apk content")
	})
}

// BenchmarkAPKProxyThroughput APK 프록시 처리량 벤치마크
func BenchmarkAPKProxyThroughput(b *testing.B) {
	env := SetupIntegrationTest(&testing.T{})
	defer env.Cleanup()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", nil)
		if err != nil {
			b.Fatal(err)
		}
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
	}
}

// TestAPKProxyMetrics APK 프록시 메트릭 수집 테스트
func TestAPKProxyMetrics(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// 여러 APK 요청 수행
	t.Run("Request Metrics", func(t *testing.T) {
		paths := []string{
			"/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz",
			"/proxy/apk/alpine/v3.16/main/x86_64/nginx-1.22.0-r1.apk",
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

		// APK 관련 메트릭이 있는지 확인
		assert.True(t, strings.Contains(metricsBody, "http_requests_total") ||
			strings.Contains(metricsBody, "proxynd_"),
			"Should contain HTTP or ProxyND metrics")
	})
}

// TestAPKProxyHTTPMethods APK 프록시 HTTP 메서드 테스트
func TestAPKProxyHTTPMethods(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// GET 메서드 테스트 (이미 다른 테스트에서 커버됨)
	t.Run("GET Method", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// HEAD 메서드 테스트
	t.Run("HEAD Method", func(t *testing.T) {
		resp, err := env.MakeRequest("HEAD", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// POST 메서드 테스트 (일반적으로 지원하지 않음)
	t.Run("POST Method", func(t *testing.T) {
		resp, err := env.MakeRequest("POST", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// APK는 읽기 전용이므로 POST는 지원하지 않음
		assert.True(t, resp.StatusCode >= 400)
	})
}

// TestAPKProxyPackageValidation APK 패키지 검증 테스트
func TestAPKProxyPackageValidation(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// APK 패키지 파일 다운로드 및 기본 검증
	t.Run("APK Package Download", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/nginx-1.22.0-r1.apk", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/vnd.alpine.apk")

		// Content-Length 헤더가 있는지 확인 (파일 크기 정보)
		contentLength := resp.Header.Get("Content-Length")
		assert.NotEmpty(t, contentLength, "Content-Length header should be present")

		// 실제 데이터가 반환되는지 확인
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		assert.Greater(t, n, 0, "Should have APK content")

		responseBody := string(body[:n])
		assert.Contains(t, responseBody, "mock apk content")
	})

	// APKINDEX 파일 검증
	t.Run("APKINDEX Validation", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/x-gzip")

		// APKINDEX 파일 크기 확인
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		assert.Greater(t, n, 0, "APKINDEX should have content")

		responseBody := string(body[:n])
		assert.Contains(t, responseBody, "mock APKINDEX content")
	})
}
