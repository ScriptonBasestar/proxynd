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

// TestYUMProxyBasicFlow 기본 YUM 프록시 동작 테스트
func TestYUMProxyBasicFlow(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// YUM upstream 준비 확인
	err := env.WaitForUpstream("yum", 5*time.Second)
	require.NoError(t, err, "YUM mock upstream should be ready")

	// Repository metadata (repomd.xml) 테스트
	t.Run("Repository Metadata", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/repomd.xml", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/xml")

		// repomd.xml 응답 검증
		body := make([]byte, 2048)
		n, _ := resp.Body.Read(body)
		responseBody := string(body[:n])
		assert.Contains(t, responseBody, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
		assert.Contains(t, responseBody, "<repomd xmlns=\"http://linux.duke.edu/metadata/repo\">")
		assert.Contains(t, responseBody, "<revision>1640995200</revision>")
		assert.Contains(t, responseBody, "<data type=\"primary\">")
		assert.Contains(t, responseBody, "repodata/primary.xml.gz")
	})

	// Primary metadata 테스트
	t.Run("Primary Metadata", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/primary.xml.gz", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/x-gzip")

		// 압축된 primary.xml 내용 검증
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		responseBody := string(body[:n])
		assert.Contains(t, responseBody, "mock compressed primary.xml content")
	})

	// RPM 패키지 다운로드 테스트
	t.Run("RPM Package Download", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/yum/centos/8/BaseOS/x86_64/os/Packages/nginx-1.20.1-1.el8.x86_64.rpm", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/x-rpm")

		// RPM 파일 내용 검증
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		responseBody := string(body[:n])
		assert.Contains(t, responseBody, "mock rpm content")
	})
}

// TestYUMProxyErrorHandling YUM 프록시 에러 처리 테스트
func TestYUMProxyErrorHandling(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// 존재하지 않는 저장소 테스트
	t.Run("Non-existent Repository", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/yum/nonexistent/repo/repodata/repomd.xml", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// 존재하지 않는 패키지 테스트
	t.Run("Non-existent Package", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/yum/centos/8/BaseOS/x86_64/os/Packages/nonexistent-package.rpm", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// 잘못된 아키텍처 경로 테스트
	t.Run("Invalid Architecture", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/yum/centos/8/BaseOS/invalid-arch/os/repodata/repomd.xml", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// 지원하지 않는 HTTP 메서드 테스트
	t.Run("Unsupported HTTP Method", func(t *testing.T) {
		resp, err := env.MakeRequest("PUT", "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/repomd.xml", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// PUT 메서드는 일반적으로 405 또는 404를 반환
		assert.True(t, resp.StatusCode >= 400)
	})
}

// TestYUMProxyCaching YUM 프록시 캐싱 동작 테스트
func TestYUMProxyCaching(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	repomdURL := "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/repomd.xml"

	// 첫 번째 요청 (캐시 미스)
	t.Run("First Request - Cache Miss", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", repomdURL, nil)
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

		resp, err := env.MakeRequest("GET", repomdURL, nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 응답 내용이 동일한지 확인
		body := make([]byte, 2048)
		n, _ := resp.Body.Read(body)
		responseBody := string(body[:n])
		assert.Contains(t, responseBody, "<repomd xmlns=\"http://linux.duke.edu/metadata/repo\">")
		assert.Contains(t, responseBody, "<revision>1640995200</revision>")
	})
}

// TestYUMProxyContentTypes YUM 프록시 Content-Type 처리 테스트
func TestYUMProxyContentTypes(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	testCases := []struct {
		name         string
		path         string
		expectedType string
	}{
		{
			name:         "Repository Metadata XML",
			path:         "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/repomd.xml",
			expectedType: "application/xml",
		},
		{
			name:         "Primary Metadata Gzip",
			path:         "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/primary.xml.gz",
			expectedType: "application/x-gzip",
		},
		{
			name:         "RPM Package",
			path:         "/proxy/yum/centos/8/BaseOS/x86_64/os/Packages/nginx-1.20.1-1.el8.x86_64.rpm",
			expectedType: "application/x-rpm",
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

// TestYUMProxyHeaders YUM 프록시 헤더 처리 테스트
func TestYUMProxyHeaders(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// YUM/DNF User-Agent 헤더 테스트
	t.Run("YUM User-Agent", func(t *testing.T) {
		headers := map[string]string{
			"User-Agent": "dnf/4.7.0",
		}

		resp, err := env.MakeRequest("GET", "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/repomd.xml", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// If-Modified-Since 헤더 테스트
	t.Run("If-Modified-Since Header", func(t *testing.T) {
		headers := map[string]string{
			"If-Modified-Since": "Wed, 21 Oct 2015 07:28:00 GMT",
		}

		resp, err := env.MakeRequest("GET", "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/repomd.xml", headers)
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

		resp, err := env.MakeRequest("GET", "/proxy/yum/centos/8/BaseOS/x86_64/os/Packages/nginx-1.20.1-1.el8.x86_64.rpm", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Mock 서버는 Range를 지원하지 않으므로 200 응답 (전체 내용)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// TestYUMProxyPerformance YUM 프록시 성능 테스트
func TestYUMProxyPerformance(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// 응답 시간 테스트
	t.Run("Response Time", func(t *testing.T) {
		start := time.Now()

		resp, err := env.MakeRequest("GET", "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/repomd.xml", nil)
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
				resp, err := env.MakeRequest("GET", "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/repomd.xml", nil)
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

// TestYUMProxyPathVariations YUM 프록시 경로 변형 테스트
func TestYUMProxyPathVariations(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	pathVariations := []struct {
		name        string
		path        string
		expectValid bool
	}{
		{
			name:        "Standard repomd.xml path",
			path:        "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/repomd.xml",
			expectValid: true,
		},
		{
			name:        "Primary metadata path",
			path:        "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/primary.xml.gz",
			expectValid: true,
		},
		{
			name:        "RPM package path",
			path:        "/proxy/yum/centos/8/BaseOS/x86_64/os/Packages/nginx-1.20.1-1.el8.x86_64.rpm",
			expectValid: true,
		},
		{
			name:        "Different distribution",
			path:        "/proxy/yum/fedora/34/Everything/x86_64/os/repodata/repomd.xml",
			expectValid: false, // Mock에서 지원하지 않음
		},
		{
			name:        "Invalid architecture",
			path:        "/proxy/yum/centos/8/BaseOS/arm32/os/repodata/repomd.xml",
			expectValid: false,
		},
		{
			name:        "Invalid path structure",
			path:        "/proxy/yum/invalid/path",
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

// TestYUMProxyRepositoryStructure YUM 저장소 구조 테스트
func TestYUMProxyRepositoryStructure(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// repodata 디렉토리 구조 테스트
	t.Run("Repodata Structure", func(t *testing.T) {
		// repomd.xml은 모든 YUM 저장소의 진입점
		resp, err := env.MakeRequest("GET", "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/repomd.xml", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// repomd.xml에서 참조하는 primary.xml.gz도 존재해야 함
		resp2, err := env.MakeRequest("GET", "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/primary.xml.gz", nil)
		require.NoError(t, err)
		defer func() { _ = resp2.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp2.StatusCode)
	})

	// Packages 디렉토리 구조 테스트
	t.Run("Packages Structure", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/yum/centos/8/BaseOS/x86_64/os/Packages/nginx-1.20.1-1.el8.x86_64.rpm", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/x-rpm")
	})
}

// TestYUMProxyUpstreamConnectivity 업스트림 연결성 테스트
func TestYUMProxyUpstreamConnectivity(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// Mock upstream이 올바르게 응답하는지 확인
	t.Run("Upstream Response Validation", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/repomd.xml", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 응답이 upstream에서 온 것인지 확인
		body := make([]byte, 2048)
		n, _ := resp.Body.Read(body)
		responseBody := string(body[:n])

		// Mock upstream의 예상 응답과 일치하는지 확인
		assert.Contains(t, responseBody, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
		assert.Contains(t, responseBody, "<repomd xmlns=\"http://linux.duke.edu/metadata/repo\">")
		assert.Contains(t, responseBody, "<revision>1640995200</revision>")
	})
}

// BenchmarkYUMProxyThroughput YUM 프록시 처리량 벤치마크
func BenchmarkYUMProxyThroughput(b *testing.B) {
	env := SetupIntegrationTest(&testing.T{})
	defer env.Cleanup()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resp, err := env.MakeRequest("GET", "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/repomd.xml", nil)
		if err != nil {
			b.Fatal(err)
		}
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
	}
}

// TestYUMProxyMetrics YUM 프록시 메트릭 수집 테스트
func TestYUMProxyMetrics(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// 여러 YUM 요청 수행
	t.Run("Request Metrics", func(t *testing.T) {
		paths := []string{
			"/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/repomd.xml",
			"/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/primary.xml.gz",
			"/proxy/yum/centos/8/BaseOS/x86_64/os/Packages/nginx-1.20.1-1.el8.x86_64.rpm",
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

		// YUM 관련 메트릭이 있는지 확인
		assert.True(t, strings.Contains(metricsBody, "http_requests_total") ||
			strings.Contains(metricsBody, "proxynd_"),
			"Should contain HTTP or ProxyND metrics")
	})
}

// TestYUMProxyHTTPMethods YUM 프록시 HTTP 메서드 테스트
func TestYUMProxyHTTPMethods(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// GET 메서드 테스트 (이미 다른 테스트에서 커버됨)
	t.Run("GET Method", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/repomd.xml", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// HEAD 메서드 테스트
	t.Run("HEAD Method", func(t *testing.T) {
		resp, err := env.MakeRequest("HEAD", "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/repomd.xml", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// POST 메서드 테스트 (일반적으로 지원하지 않음)
	t.Run("POST Method", func(t *testing.T) {
		resp, err := env.MakeRequest("POST", "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/repomd.xml", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// YUM은 읽기 전용이므로 POST는 지원하지 않음
		assert.True(t, resp.StatusCode >= 400)
	})
}

// TestYUMProxyRPMValidation RPM 패키지 검증 테스트
func TestYUMProxyRPMValidation(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// RPM 파일 다운로드 및 기본 검증
	t.Run("RPM File Download", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/yum/centos/8/BaseOS/x86_64/os/Packages/nginx-1.20.1-1.el8.x86_64.rpm", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/x-rpm")

		// Content-Length 헤더가 있는지 확인 (파일 크기 정보)
		contentLength := resp.Header.Get("Content-Length")
		assert.NotEmpty(t, contentLength, "Content-Length header should be present")

		// 실제 데이터가 반환되는지 확인
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		assert.Greater(t, n, 0, "Should have RPM content")

		responseBody := string(body[:n])
		assert.Contains(t, responseBody, "mock rpm content")
	})
}
