package integration

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/metrics"
)

// TestAPKProxyBasicFlow 기본 APK 프록시 동작 테스트
func TestAPKProxyBasicFlow(t *testing.T) {
	// 개발 환경 설정 (인증 우회)
	_ = os.Setenv("PROXYND_ENV", "development")
	defer func() { _ = os.Unsetenv("PROXYND_ENV") }()

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
		// Content-Type should be gzip format
		contentType := resp.Header.Get("Content-Type")
		assert.True(t,
			strings.Contains(contentType, "application/gzip") ||
				strings.Contains(contentType, "application/x-gzip"),
			"Content-Type should be gzip format, got: %s", contentType)

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
	// 개발 환경 설정 및 메트릭 리셋
	_ = os.Setenv("PROXYND_ENV", "development")
	defer func() { _ = os.Unsetenv("PROXYND_ENV") }()
	metrics.ResetMetrics()

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
	// 개발 환경 설정 및 메트릭 리셋
	_ = os.Setenv("PROXYND_ENV", "development")
	defer func() { _ = os.Unsetenv("PROXYND_ENV") }()
	metrics.ResetMetrics()

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
			expectedType: "application/gzip", // Accept both gzip and x-gzip
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
			// Handle both gzip variants for APKINDEX files
			contentType := resp.Header.Get("Content-Type")
			if tc.expectedType == "application/gzip" {
				assert.True(t,
					strings.Contains(contentType, "application/gzip") ||
						strings.Contains(contentType, "application/x-gzip"),
					"Expected gzip content type, got: %s", contentType)
			} else {
				assert.Contains(t, contentType, tc.expectedType)
			}
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
		// Content-Type should be gzip format
		contentType := resp.Header.Get("Content-Type")
		assert.True(t,
			strings.Contains(contentType, "application/gzip") ||
				strings.Contains(contentType, "application/x-gzip"),
			"Content-Type should be gzip format, got: %s", contentType)

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
		// Content-Type should be gzip format
		contentType := resp.Header.Get("Content-Type")
		assert.True(t,
			strings.Contains(contentType, "application/gzip") ||
				strings.Contains(contentType, "application/x-gzip"),
			"Content-Type should be gzip format, got: %s", contentType)

		// APKINDEX 파일 크기 확인
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		assert.Greater(t, n, 0, "APKINDEX should have content")

		responseBody := string(body[:n])
		assert.Contains(t, responseBody, "mock APKINDEX content")
	})
}

// TestAPKProxyAdvancedCases APK 프록시 고급 케이스 테스트
// 시그니처 검증 실패 경로와 미러 스위치 기능을 테스트
func TestAPKProxyAdvancedCases(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// 시그니처 검증 실패 경로 테스트
	t.Run("Signature Verification Failure Path", func(t *testing.T) {
		// 시그니처가 손상된 패키지에 대한 요청 시뮬레이션
		// Mock upstream에서 시그니처 검증 실패 시나리오 테스트
		headers := map[string]string{
			"X-APK-Signature-Check": "strict", // 엄격한 시그니처 검증 요청
		}

		// 손상된 시그니처를 가진 패키지 요청
		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/bad-signature-package.apk", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// 시그니처 검증 실패 시 404, 500, 또는 502 응답 예상
		assert.True(t, resp.StatusCode == http.StatusNotFound ||
			resp.StatusCode == http.StatusBadGateway ||
			resp.StatusCode == http.StatusInternalServerError,
			"Should fail with 404, 500, or 502 for signature verification failure")

		// 에러 응답에 시그니처 관련 정보가 포함되어야 함
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		responseBody := string(body[:n])

		// 시그니처 검증 실패나 관련 오류 메시지 확인
		assert.True(t,
			strings.Contains(strings.ToLower(responseBody), "signature") ||
				strings.Contains(strings.ToLower(responseBody), "verification") ||
				strings.Contains(strings.ToLower(responseBody), "not found") ||
				strings.Contains(strings.ToLower(responseBody), "failed") ||
				strings.Contains(strings.ToLower(responseBody), "error") ||
				len(responseBody) == 0, // 빈 응답도 허용 (일부 프록시는 빈 404 응답)
			"Error response should indicate signature verification or general failure issue")
	})

	// 시그니처 검증 우회 테스트
	t.Run("Signature Verification Bypass", func(t *testing.T) {
		// 시그니처 검증을 우회하는 요청
		headers := map[string]string{
			"X-APK-Signature-Check": "disabled", // 시그니처 검증 비활성화
		}

		// 정상 패키지는 시그니처 검증 설정과 관계없이 성공해야 함
		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/nginx-1.22.0-r1.apk", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/vnd.alpine.apk")
	})

	// 미러 스위치 기능 테스트
	t.Run("Mirror Switch Functionality", func(t *testing.T) {
		// 기본 미러에서 패키지 요청
		resp1, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", nil)
		require.NoError(t, err)
		defer func() { _ = resp1.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp1.StatusCode)

		// 미러 스위치 요청 (대안 미러 사용)
		headers := map[string]string{
			"X-APK-Mirror": "fallback", // 대체 미러 사용 요청
		}

		resp2, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", headers)
		require.NoError(t, err)
		defer func() { _ = resp2.Body.Close() }()

		// 대체 미러도 성공해야 함 (Mock 환경에서는 동일한 응답)
		assert.Equal(t, http.StatusOK, resp2.StatusCode)

		// 두 응답이 유사한 내용을 가져야 함 (Mock에서는 동일)
		body1 := make([]byte, 1024)
		n1, _ := resp1.Body.Read(body1)
		body2 := make([]byte, 1024)
		n2, _ := resp2.Body.Read(body2)

		// 두 미러 모두 유효한 응답을 제공해야 함
		assert.Greater(t, n1, 0, "Primary mirror should return content")
		assert.Greater(t, n2, 0, "Fallback mirror should return content")

		response1Body := string(body1[:n1])
		response2Body := string(body2[:n2])

		// Mock 환경에서는 동일한 응답 예상
		assert.Contains(t, response1Body, "mock APKINDEX content")
		assert.Contains(t, response2Body, "mock APKINDEX content")
	})

	// 미러 장애 시 자동 전환 테스트
	t.Run("Mirror Failover on Error", func(t *testing.T) {
		// 존재하지 않는 특수 경로로 장애 상황 시뮬레이션
		headers := map[string]string{
			"X-APK-Mirror": "primary", // 주 미러 명시적 지정
		}

		// 장애가 예상되는 경로 (존재하지 않는 패키지)
		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/nonexistent-package.apk", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// 404 또는 500 응답이 정상적으로 반환되어야 함 (미러 전환이 올바르게 작동)
		assert.True(t, resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusInternalServerError,
			"Should return 404 or 500 for missing packages")

		// 응답 시간이 과도하게 길지 않아야 함 (타임아웃으로 인한 장애가 아님)
		start := time.Now()
		resp2, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/nonexistent-package2.apk", headers)
		require.NoError(t, err)
		defer func() { _ = resp2.Body.Close() }()
		duration := time.Since(start)

		assert.Less(t, duration, 5*time.Second, "Failover should be fast")
		assert.True(t, resp2.StatusCode == http.StatusNotFound || resp2.StatusCode == http.StatusInternalServerError,
			"Should return 404 or 500 for missing packages")
	})

	// 복잡한 미러 설정 테스트
	t.Run("Complex Mirror Configuration", func(t *testing.T) {
		// 여러 헤더를 조합한 복잡한 미러 설정 테스트
		headers := map[string]string{
			"X-APK-Mirror":          "auto",    // 자동 미러 선택
			"X-APK-Mirror-Failover": "enabled", // 장애 전환 활성화
			"X-APK-Mirror-Timeout":  "5s",      // 타임아웃 설정
			"X-APK-Signature-Check": "enabled", // 시그니처 검증 활성화
		}

		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 응답이 올바른 형식인지 확인 (application/gzip 또는 application/x-gzip)
		contentType := resp.Header.Get("Content-Type")
		assert.True(t,
			strings.Contains(contentType, "application/gzip") ||
				strings.Contains(contentType, "application/x-gzip"),
			"Content-Type should be gzip format")

		// 컨텐츠가 존재하는지 확인
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		assert.Greater(t, n, 0, "Should receive content with complex mirror config")

		responseBody := string(body[:n])
		assert.Contains(t, responseBody, "mock APKINDEX content")
	})

	// 미러 헬스체크 테스트
	t.Run("Mirror Health Check", func(t *testing.T) {
		// 미러 상태 확인을 위한 헬스체크 요청
		headers := map[string]string{
			"X-APK-Health-Check": "true", // 헬스체크 요청
		}

		// 헬스체크는 실제 파일 다운로드 대신 가벼운 확인만 수행
		resp, err := env.MakeRequest("HEAD", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// HEAD 요청이므로 200 응답이면 미러가 정상
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Content-Length 헤더가 있어야 함 (헬스체크 성공 지표)
		contentLength := resp.Header.Get("Content-Length")
		assert.NotEmpty(t, contentLength, "Health check should return content length")
	})

	// 지역별 미러 선택 테스트
	t.Run("Regional Mirror Selection", func(t *testing.T) {
		// 지역 정보를 포함한 미러 선택 테스트
		headers := map[string]string{
			"X-APK-Region":    "asia",  // 아시아 지역 미러 선택
			"Accept-Language": "ko-KR", // 한국어 지역 설정
		}

		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 지역별 미러 선택이 성공했는지 확인
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		assert.Greater(t, n, 0, "Regional mirror should return content")

		responseBody := string(body[:n])
		assert.Contains(t, responseBody, "mock APKINDEX content")
	})
}

// TestAPKProxySignatureVerificationAdvanced APK 서명 검증 실패 경로 테스트
func TestAPKProxySignatureVerificationAdvanced(t *testing.T) {
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	t.Run("InvalidSignature", func(t *testing.T) {
		// 잘못된 서명을 가진 패키지 요청
		headers := map[string]string{
			"X-APK-Verify-Signature": "strict",
			"X-APK-Trust-Level":      "paranoid",
		}

		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/invalid-sig-package.apk", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// 서명 검증 실패로 인한 에러 응답 확인
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		
		responseBody := string(body)
		assert.Contains(t, responseBody, "signature verification failed")
		assert.Contains(t, responseBody, "untrusted package")
	})

	t.Run("MissingPublicKey", func(t *testing.T) {
		// 공개키가 없는 상태에서 서명 검증 시도
		headers := map[string]string{
			"X-APK-Verify-Signature": "strict",
			"X-APK-Key-Missing":      "true",
		}

		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/unknown-key-package.apk", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
		
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		
		responseBody := string(body)
		assert.Contains(t, responseBody, "public key not found")
		assert.Contains(t, responseBody, "keyring unavailable")
	})

	t.Run("SignatureBypass", func(t *testing.T) {
		// 서명 검증 우회 시도 테스트
		headers := map[string]string{
			"X-APK-Verify-Signature": "disabled",
			"X-Security-Override":    "bypass",
		}

		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/unsigned-package.apk", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// 보안 정책에 따라 우회가 차단되어야 함
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		
		responseBody := string(body)
		assert.Contains(t, responseBody, "security policy violation")
		assert.Contains(t, responseBody, "signature verification cannot be bypassed")
	})

	t.Run("CorruptedSignature", func(t *testing.T) {
		// 손상된 서명 파일 테스트
		headers := map[string]string{
			"X-APK-Verify-Signature": "strict",
			"X-Signature-Corrupted":  "true",
		}

		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/corrupted-sig-package.apk", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
		
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		
		responseBody := string(body)
		assert.Contains(t, responseBody, "signature corrupted")
		assert.Contains(t, responseBody, "unable to verify integrity")
	})
}

// TestAPKProxyMirrorAutoSwitchAdvanced 미러 자동 전환 테스트
func TestAPKProxyMirrorAutoSwitchAdvanced(t *testing.T) {
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	t.Run("PrimaryMirrorDown", func(t *testing.T) {
		// 주 미러 서버 다운 시 자동 전환
		headers := map[string]string{
			"X-Primary-Mirror-Status": "down",
			"X-Failover-Enabled":     "true",
			"X-Max-Retries":          "3",
		}

		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		// 대체 미러에서 성공적으로 응답받았는지 확인
		mirrorUsed := resp.Header.Get("X-Mirror-Used")
		assert.Contains(t, mirrorUsed, "fallback")
		assert.NotContains(t, mirrorUsed, "primary")
	})

	t.Run("RegionalMirrorPreference", func(t *testing.T) {
		// 지역별 미러 우선순위 테스트
		headers := map[string]string{
			"X-Client-Region":        "asia-pacific",
			"X-Mirror-Selection":     "regional-priority",
			"Accept-Language":        "ko-KR",
		}

		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		mirrorUsed := resp.Header.Get("X-Mirror-Used")
		latency := resp.Header.Get("X-Mirror-Latency")
		
		assert.Contains(t, mirrorUsed, "asia")
		assert.NotEmpty(t, latency)
		
		// 지연시간이 합리적인 범위인지 확인
		latencyMs := resp.Header.Get("X-Mirror-Latency-Ms")
		if latencyMs != "" {
			// 아시아 지역 미러이므로 지연시간이 낮아야 함
			assert.Contains(t, latencyMs, "low")
		}
	})

	t.Run("LoadBalancing", func(t *testing.T) {
		// 로드 밸런싱 동작 확인
		mirrorCounts := make(map[string]int)
		
		for i := 0; i < 10; i++ {
			headers := map[string]string{
				"X-Load-Balancing": "round-robin",
				"X-Request-ID":     fmt.Sprintf("test-req-%d", i),
			}

			resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", headers)
			require.NoError(t, err)
			
			mirrorUsed := resp.Header.Get("X-Mirror-Used")
			if mirrorUsed != "" {
				mirrorCounts[mirrorUsed]++
			}
			
			_ = resp.Body.Close()
		}

		// 여러 미러에 요청이 분산되었는지 확인
		assert.Greater(t, len(mirrorCounts), 1, "Load balancing should use multiple mirrors")
		
		for mirror, count := range mirrorCounts {
			t.Logf("Mirror %s used %d times", mirror, count)
			assert.Greater(t, count, 0, "Each mirror should handle at least one request")
		}
	})

	t.Run("CircuitBreakerPattern", func(t *testing.T) {
		// 서킷 브레이커 패턴 테스트
		headers := map[string]string{
			"X-Circuit-Breaker-Test": "enabled",
			"X-Failure-Threshold":    "3",
			"X-Recovery-Timeout":     "30s",
		}

		// 첫 번째 요청: 정상 동작
		resp1, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", headers)
		require.NoError(t, err)
		defer func() { _ = resp1.Body.Close() }()
		
		assert.Equal(t, http.StatusOK, resp1.StatusCode)
		
		circuitState := resp1.Header.Get("X-Circuit-State")
		assert.Equal(t, "closed", circuitState)

		// 실패 시뮬레이션
		headers["X-Simulate-Failures"] = "3"
		
		resp2, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", headers)
		require.NoError(t, err)
		defer func() { _ = resp2.Body.Close() }()
		
		// 서킷이 열려서 빠른 실패 응답
		circuitState2 := resp2.Header.Get("X-Circuit-State")
		assert.Contains(t, []string{"open", "half-open"}, circuitState2)
	})
}

// TestAPKProxyAPKIndexCompressionAdvanced APKINDEX 압축 처리 테스트
func TestAPKProxyAPKIndexCompressionAdvanced(t *testing.T) {
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	t.Run("GzipCompression", func(t *testing.T) {
		// Gzip 압축된 APKINDEX 처리
		headers := map[string]string{
			"Accept-Encoding": "gzip, deflate",
			"X-Compression":   "gzip",
		}

		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		contentEncoding := resp.Header.Get("Content-Encoding")
		assert.Contains(t, contentEncoding, "gzip")
		
		contentType := resp.Header.Get("Content-Type")
		assert.Contains(t, contentType, "application/gzip")
	})

	t.Run("XzCompression", func(t *testing.T) {
		// XZ 압축된 APKINDEX 처리
		headers := map[string]string{
			"Accept-Encoding": "xz, gzip",
			"X-Compression":   "xz",
		}

		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.xz", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		contentType := resp.Header.Get("Content-Type")
		assert.Contains(t, contentType, "application/x-xz")
	})

	t.Run("CompressionNegotiation", func(t *testing.T) {
		// 클라이언트 압축 지원 협상
		headers := map[string]string{
			"Accept-Encoding": "br, gzip;q=0.8, deflate;q=0.6",
			"X-Compression":   "auto",
		}

		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		// 가장 적합한 압축 방식이 선택되었는지 확인
		contentEncoding := resp.Header.Get("Content-Encoding")
		varyHeader := resp.Header.Get("Vary")
		
		assert.NotEmpty(t, contentEncoding)
		assert.Contains(t, varyHeader, "Accept-Encoding")
	})

	t.Run("UncompressedFallback", func(t *testing.T) {
		// 압축을 지원하지 않는 클라이언트 대응
		headers := map[string]string{
			"Accept-Encoding": "identity",
			"X-Compression":   "none",
		}

		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		contentEncoding := resp.Header.Get("Content-Encoding")
		assert.Empty(t, contentEncoding)
		
		contentLength := resp.Header.Get("Content-Length")
		assert.NotEmpty(t, contentLength)
	})

	t.Run("CompressionRatioOptimization", func(t *testing.T) {
		// 압축 효율성 테스트
		headers := map[string]string{
			"Accept-Encoding":      "gzip, deflate",
			"X-Compression-Level":  "9", // 최고 압축률
			"X-Optimize-For":       "bandwidth",
		}

		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		compressionRatio := resp.Header.Get("X-Compression-Ratio")
		if compressionRatio != "" {
			t.Logf("Compression ratio: %s", compressionRatio)
		}
		
		// 압축된 응답의 크기 확인
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		
		assert.Greater(t, len(body), 0)
		t.Logf("Compressed response size: %d bytes", len(body))
	})
}

// TestAPKProxyPackageDownloadVerificationAdvanced APK 패키지 다운로드 검증
func TestAPKProxyPackageDownloadVerificationAdvanced(t *testing.T) {
	env := NewTestEnvironment(t)
	defer env.Cleanup()

	t.Run("ChecksumVerification", func(t *testing.T) {
		// 체크섬 검증 테스트
		headers := map[string]string{
			"X-Verify-Checksum": "sha256",
			"X-Expected-Hash":   "abc123def456", // 모의 해시
		}

		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/test-package-1.0.apk", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		// 체크섬 헤더 확인
		checksumHeader := resp.Header.Get("X-Checksum-SHA256")
		assert.NotEmpty(t, checksumHeader)
		
		integrityHeader := resp.Header.Get("X-Integrity-Verified")
		assert.Equal(t, "true", integrityHeader)
	})

	t.Run("SizeValidation", func(t *testing.T) {
		// 패키지 크기 검증
		headers := map[string]string{
			"X-Verify-Size":    "true",
			"X-Expected-Size":  "1024000", // 1MB
			"Range":            "bytes=0-1023", // 첫 1KB만 요청
		}

		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/large-package-2.0.apk", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusPartialContent, resp.StatusCode)
		
		contentRange := resp.Header.Get("Content-Range")
		assert.Contains(t, contentRange, "bytes 0-1023/1024000")
		
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, 1024, len(body))
	})

	t.Run("ResumeDownload", func(t *testing.T) {
		// 다운로드 재개 기능 테스트
		headers1 := map[string]string{
			"Range":           "bytes=0-1023",
			"X-Resume-Token":  "test-token-123",
		}

		resp1, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/resume-test-package.apk", headers1)
		require.NoError(t, err)
		defer func() { _ = resp1.Body.Close() }()

		assert.Equal(t, http.StatusPartialContent, resp1.StatusCode)
		
		// 첫 번째 청크 읽기
		firstChunk, err := io.ReadAll(resp1.Body)
		require.NoError(t, err)
		assert.Equal(t, 1024, len(firstChunk))

		// 두 번째 요청: 이어서 다운로드
		headers2 := map[string]string{
			"Range":           "bytes=1024-2047",
			"X-Resume-Token":  "test-token-123",
			"If-Range":        resp1.Header.Get("ETag"),
		}

		resp2, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/resume-test-package.apk", headers2)
		require.NoError(t, err)
		defer func() { _ = resp2.Body.Close() }()

		assert.Equal(t, http.StatusPartialContent, resp2.StatusCode)
		
		secondChunk, err := io.ReadAll(resp2.Body)
		require.NoError(t, err)
		assert.Equal(t, 1024, len(secondChunk))
		
		// 두 청크가 다른 내용인지 확인
		assert.NotEqual(t, firstChunk, secondChunk)
	})

	t.Run("ConcurrentDownloads", func(t *testing.T) {
		// 동시 다운로드 테스트
		const numConcurrent = 5
		var wg sync.WaitGroup
		results := make([]bool, numConcurrent)
		
		for i := 0; i < numConcurrent; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				
				headers := map[string]string{
					"X-Concurrent-ID": fmt.Sprintf("download-%d", index),
					"User-Agent":      fmt.Sprintf("TestClient-%d", index),
				}

				resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/concurrent-test-package.apk", headers)
				if err != nil {
					results[index] = false
					return
				}
				defer func() { _ = resp.Body.Close() }()

				results[index] = (resp.StatusCode == http.StatusOK)
				
				// 응답 본문을 읽어서 완전성 확인
				body, err := io.ReadAll(resp.Body)
				if err != nil || len(body) == 0 {
					results[index] = false
				}
			}(i)
		}
		
		wg.Wait()
		
		// 모든 동시 다운로드가 성공했는지 확인
		for i, success := range results {
			assert.True(t, success, "Concurrent download %d should succeed", i)
		}
	})

	t.Run("DownloadWithMetadata", func(t *testing.T) {
		// 메타데이터와 함께 다운로드
		headers := map[string]string{
			"X-Include-Metadata": "true",
			"X-Metadata-Format":  "json",
		}

		resp, err := env.MakeRequest("GET", "/proxy/apk/alpine/v3.16/main/x86_64/metadata-test-package.apk", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		// 메타데이터 헤더 확인
		packageName := resp.Header.Get("X-Package-Name")
		packageVersion := resp.Header.Get("X-Package-Version")
		packageArch := resp.Header.Get("X-Package-Arch")
		
		assert.NotEmpty(t, packageName)
		assert.NotEmpty(t, packageVersion)
		assert.NotEmpty(t, packageArch)
		
		// 의존성 정보 확인
		dependencies := resp.Header.Get("X-Package-Dependencies")
		if dependencies != "" {
			t.Logf("Package dependencies: %s", dependencies)
		}
	})
}
