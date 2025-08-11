package integration

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// YUM 테스트용 공통 URL 경로들
	yumRepomdPath        = "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/repomd.xml"
	yumPrimaryGzPath     = "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/primary.xml.gz"
	yumNginxRPMPath      = "/proxy/yum/centos/8/BaseOS/x86_64/os/Packages/nginx-1.20.1-1.el8.x86_64.rpm"
	yumRepomdAscPath     = "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/repomd.xml.asc"
	yumNonExistentRPM    = "/proxy/yum/centos/8/BaseOS/x86_64/os/Packages/nonexistent.rpm"
	yumNonExistentPkgRPM = "/proxy/yum/centos/8/BaseOS/x86_64/os/Packages/nonexistent-package.rpm"
	yumInvalidArchPath   = "/proxy/yum/centos/8/BaseOS/invalid-arch/os/repodata/repomd.xml"
	yumArm32Path         = "/proxy/yum/centos/8/BaseOS/arm32/os/repodata/repomd.xml"
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
		resp, err := env.MakeRequest("GET", yumRepomdPath, nil)
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
		resp, err := env.MakeRequest("GET", yumPrimaryGzPath, nil)
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
		resp, err := env.MakeRequest("GET", yumNginxRPMPath, nil)
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
		resp, err := env.MakeRequest("GET", yumNonExistentPkgRPM, nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// 잘못된 아키텍처 경로 테스트
	t.Run("Invalid Architecture", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", yumInvalidArchPath, nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// 지원하지 않는 HTTP 메서드 테스트
	t.Run("Unsupported HTTP Method", func(t *testing.T) {
		resp, err := env.MakeRequest("PUT", yumRepomdPath, nil)
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

	repomdURL := yumRepomdPath

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
			path:         yumRepomdPath,
			expectedType: "application/xml",
		},
		{
			name:         "Primary Metadata Gzip",
			path:         yumPrimaryGzPath,
			expectedType: "application/x-gzip",
		},
		{
			name:         "RPM Package",
			path:         yumNginxRPMPath,
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

		resp, err := env.MakeRequest("GET", yumRepomdPath, headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// If-Modified-Since 헤더 테스트
	t.Run("If-Modified-Since Header", func(t *testing.T) {
		headers := map[string]string{
			"If-Modified-Since": "Wed, 21 Oct 2015 07:28:00 GMT",
		}

		resp, err := env.MakeRequest("GET", yumRepomdPath, headers)
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

		resp, err := env.MakeRequest("GET", yumNginxRPMPath, headers)
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

		resp, err := env.MakeRequest("GET", yumRepomdPath, nil)
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
				resp, err := env.MakeRequest("GET", yumRepomdPath, nil)
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
			path:        yumRepomdPath,
			expectValid: true,
		},
		{
			name:        "Primary metadata path",
			path:        yumPrimaryGzPath,
			expectValid: true,
		},
		{
			name:        "RPM package path",
			path:        yumNginxRPMPath,
			expectValid: true,
		},
		{
			name:        "Different distribution",
			path:        "/proxy/yum/fedora/34/Everything/x86_64/os/repodata/repomd.xml",
			expectValid: false, // Mock에서 지원하지 않음
		},
		{
			name:        "Invalid architecture",
			path:        yumArm32Path,
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
		resp, err := env.MakeRequest("GET", yumRepomdPath, nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// repomd.xml에서 참조하는 primary.xml.gz도 존재해야 함
		resp2, err := env.MakeRequest("GET", yumPrimaryGzPath, nil)
		require.NoError(t, err)
		defer func() { _ = resp2.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp2.StatusCode)
	})

	// Packages 디렉토리 구조 테스트
	t.Run("Packages Structure", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", yumNginxRPMPath, nil)
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
		resp, err := env.MakeRequest("GET", yumRepomdPath, nil)
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
		resp, err := env.MakeRequest("GET", yumRepomdPath, nil)
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
			yumRepomdPath,
			yumPrimaryGzPath,
			yumNginxRPMPath,
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
		resp, err := env.MakeRequest("GET", yumRepomdPath, nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// HEAD 메서드 테스트
	t.Run("HEAD Method", func(t *testing.T) {
		resp, err := env.MakeRequest("HEAD", yumRepomdPath, nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// POST 메서드 테스트 (일반적으로 지원하지 않음)
	t.Run("POST Method", func(t *testing.T) {
		resp, err := env.MakeRequest("POST", yumRepomdPath, nil)
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
		resp, err := env.MakeRequest("GET", yumNginxRPMPath, nil)
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

// TestYUMProxyRepodataFreshness YUM 프록시 repodata 신선도 테스트
func TestYUMProxyRepodataFreshness(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// Repository metadata 메타데이터 신선도 테스트
	t.Run("Metadata Staleness Detection", func(t *testing.T) {
		// 첫 번째 요청으로 메타데이터 캐시
		resp, err := env.MakeRequest("GET", yumRepomdPath, nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Last-Modified 헤더 확인
		lastModified := resp.Header.Get("Last-Modified")
		assert.NotEmpty(t, lastModified, "Last-Modified header should be present")

		// ETag 헤더 확인
		etag := resp.Header.Get("ETag")
		assert.NotEmpty(t, etag, "ETag header should be present")
	})

	// 조건부 요청 테스트
	t.Run("Conditional Requests with If-Modified-Since", func(t *testing.T) {
		// 과거 날짜로 If-Modified-Since 헤더 설정
		oldDate := "Wed, 21 Oct 2015 07:28:00 GMT"
		headers := map[string]string{
			"If-Modified-Since": oldDate,
		}

		resp, err := env.MakeRequest("GET", yumRepomdPath, headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// 수정되지 않은 경우 304 또는 200 응답 (Mock 서버 동작에 따라)
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotModified)
	})

	t.Run("Conditional Requests with If-None-Match", func(t *testing.T) {
		// 약한 ETag로 조건부 요청
		headers := map[string]string{
			"If-None-Match": "W/\"test-etag\"",
		}

		resp, err := env.MakeRequest("GET", yumRepomdPath, headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotModified)
	})

	// TTL 및 만료 처리 테스트
	t.Run("TTL and Expiration Handling", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", yumRepomdPath, nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Cache-Control 헤더 확인
		cacheControl := resp.Header.Get("Cache-Control")
		if cacheControl != "" {
			assert.True(t, strings.Contains(cacheControl, "max-age") ||
				strings.Contains(cacheControl, "no-cache") ||
				strings.Contains(cacheControl, "must-revalidate"))
		}

		// Expires 헤더 확인
		expires := resp.Header.Get("Expires")
		if expires != "" {
			_, err := time.Parse(time.RFC1123, expires)
			assert.NoError(t, err, "Expires header should be valid RFC1123 format")
		}
	})

	// Stale-while-revalidate 테스트
	t.Run("Stale While Revalidate", func(t *testing.T) {
		// 캐시된 응답 요청
		resp1, err := env.MakeRequest("GET", yumPrimaryGzPath, nil)
		require.NoError(t, err)
		defer func() { _ = resp1.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp1.StatusCode)

		// 잠시 대기 후 재요청
		time.Sleep(100 * time.Millisecond)

		resp2, err := env.MakeRequest("GET", yumPrimaryGzPath, nil)
		require.NoError(t, err)
		defer func() { _ = resp2.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp2.StatusCode)

		// 두 응답이 일관성 있게 처리되었는지 확인
		contentType1 := resp1.Header.Get("Content-Type")
		contentType2 := resp2.Header.Get("Content-Type")
		assert.Equal(t, contentType1, contentType2)
	})
}

// TestYUMProxyGzipAndPGPHeaders YUM 프록시 Gzip 및 PGP 헤더 테스트
func TestYUMProxyGzipAndPGPHeaders(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// Gzip 압축 처리 테스트
	t.Run("Gzip Compression Handling", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", yumPrimaryGzPath, nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/x-gzip")

		// Content-Encoding 헤더 확인
		contentEncoding := resp.Header.Get("Content-Encoding")
		if contentEncoding != "" {
			assert.Equal(t, "gzip", contentEncoding)
		}
	})

	// Accept-Encoding 처리 테스트
	t.Run("Accept-Encoding Processing", func(t *testing.T) {
		headers := map[string]string{
			"Accept-Encoding": "gzip, deflate, br",
		}

		resp, err := env.MakeRequest("GET", yumPrimaryGzPath, headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 요청한 압축 형식에 따른 응답 확인
		contentType := resp.Header.Get("Content-Type")
		assert.Contains(t, contentType, "application/x-gzip")
	})

	// GPG 서명 헤더 테스트
	t.Run("GPG Signature Headers", func(t *testing.T) {
		// repomd.xml.asc 서명 파일 요청
		resp, err := env.MakeRequest("GET", yumRepomdAscPath, nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// 서명 파일이 있는 경우 (Mock에서 지원하는 경우)
		if resp.StatusCode == http.StatusOK {
			assert.Contains(t, resp.Header.Get("Content-Type"), "application/pgp-signature")
		} else {
			// Mock에서 지원하지 않는 경우 404 예상
			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		}
	})

	// 체크섬 검증 헤더 테스트
	t.Run("Checksum Verification Headers", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", yumPrimaryGzPath, nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// MD5 체크섬 헤더 확인
		md5Header := resp.Header.Get("Content-MD5")
		if md5Header != "" {
			// Base64로 인코딩된 MD5 해시인지 확인
			assert.Regexp(t, `^[A-Za-z0-9+/]+=*$`, md5Header)
		}

		// SHA256 체크섬 헤더 확인 (확장)
		sha256Header := resp.Header.Get("X-Content-SHA256")
		if sha256Header != "" {
			// 64자리 16진수 해시인지 확인
			assert.Regexp(t, `^[a-fA-F0-9]{64}$`, sha256Header)
		}
	})

	// 압축 무결성 테스트
	t.Run("Compression Integrity", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", yumPrimaryGzPath, nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 응답 내용 읽기
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Greater(t, len(body), 0, "Compressed file should have content")

		// Mock 응답인지 확인
		bodyStr := string(body)
		assert.Contains(t, bodyStr, "mock compressed primary.xml content")
	})

	// 다중 압축 형식 지원 테스트
	t.Run("Multiple Compression Format Support", func(t *testing.T) {
		compressionTests := []struct {
			name     string
			path     string
			mimeType string
		}{
			{
				name:     "Gzip Compressed XML",
				path:     yumPrimaryGzPath,
				mimeType: "application/x-gzip",
			},
			{
				name:     "Uncompressed XML",
				path:     yumRepomdPath,
				mimeType: "application/xml",
			},
		}

		for _, test := range compressionTests {
			t.Run(test.name, func(t *testing.T) {
				resp, err := env.MakeRequest("GET", test.path, nil)
				require.NoError(t, err)
				defer func() { _ = resp.Body.Close() }()

				if resp.StatusCode == http.StatusOK {
					assert.Contains(t, resp.Header.Get("Content-Type"), test.mimeType)
				}
			})
		}
	})
}

// TestYUMProxyRangeRequests YUM 프록시 Range 요청 테스트
func TestYUMProxyRangeRequests(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// 부분 다운로드 지원 테스트
	t.Run("Partial Download Support", func(t *testing.T) {
		// Range 헤더로 부분 요청
		headers := map[string]string{
			"Range": "bytes=0-1023",
		}

		resp, err := env.MakeRequest("GET", yumNginxRPMPath, headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Mock 서버가 Range를 지원하는지에 따라 206 또는 200
		assert.True(t, resp.StatusCode == http.StatusPartialContent || resp.StatusCode == http.StatusOK)

		if resp.StatusCode == http.StatusPartialContent {
			// Content-Range 헤더 확인
			contentRange := resp.Header.Get("Content-Range")
			assert.NotEmpty(t, contentRange)
			assert.Contains(t, contentRange, "bytes")
		}
	})

	// Accept-Ranges 헤더 테스트
	t.Run("Accept-Ranges Header", func(t *testing.T) {
		resp, err := env.MakeRequest("HEAD", yumNginxRPMPath, nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Accept-Ranges 헤더 확인
		acceptRanges := resp.Header.Get("Accept-Ranges")
		if acceptRanges != "" {
			assert.True(t, acceptRanges == "bytes" || acceptRanges == "none")
		}
	})

	// 단일 바이트 범위 요청 테스트
	t.Run("Single Byte Range Request", func(t *testing.T) {
		testCases := []struct {
			name      string
			rangeSpec string
		}{
			{"First 512 bytes", "bytes=0-511"},
			{"Middle range", "bytes=512-1023"},
			{"Last 100 bytes", "bytes=-100"},
			{"From byte 1000", "bytes=1000-"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				headers := map[string]string{
					"Range": tc.rangeSpec,
				}

				resp, err := env.MakeRequest("GET", yumNginxRPMPath, headers)
				require.NoError(t, err)
				defer func() { _ = resp.Body.Close() }()

				// Range 요청이 처리되는지 확인
				assert.True(t, resp.StatusCode == http.StatusPartialContent || resp.StatusCode == http.StatusOK)

				if resp.StatusCode == http.StatusPartialContent {
					contentRange := resp.Header.Get("Content-Range")
					assert.NotEmpty(t, contentRange)
				}
			})
		}
	})

	// 다중 범위 요청 테스트
	t.Run("Multiple Range Request", func(t *testing.T) {
		headers := map[string]string{
			"Range": "bytes=0-99,200-299,400-499",
		}

		resp, err := env.MakeRequest("GET", yumNginxRPMPath, headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// 다중 범위는 더 복잡하므로 서버가 지원하지 않을 수 있음
		assert.True(t, resp.StatusCode == http.StatusPartialContent ||
			resp.StatusCode == http.StatusOK ||
			resp.StatusCode == http.StatusRequestedRangeNotSatisfiable)

		if resp.StatusCode == http.StatusPartialContent {
			contentType := resp.Header.Get("Content-Type")
			// 다중 범위인 경우 multipart/byteranges
			if strings.Contains(contentType, "multipart/byteranges") {
				assert.Contains(t, contentType, "boundary=")
			}
		}
	})

	// 잘못된 범위 요청 테스트
	t.Run("Invalid Range Requests", func(t *testing.T) {
		invalidRanges := []string{
			"bytes=abc-def",   // 잘못된 숫자
			"bytes=1000-500",  // 끝이 시작보다 작음
			"bytes=99999999-", // 파일 크기보다 큰 시작점
			"invalid-range",   // 잘못된 형식
		}

		for _, rangeSpec := range invalidRanges {
			t.Run("Invalid: "+rangeSpec, func(t *testing.T) {
				headers := map[string]string{
					"Range": rangeSpec,
				}

				resp, err := env.MakeRequest("GET", yumNginxRPMPath, headers)
				require.NoError(t, err)
				defer func() { _ = resp.Body.Close() }()

				// 잘못된 범위에 대해 적절한 응답
				assert.True(t, resp.StatusCode == http.StatusOK ||
					resp.StatusCode == http.StatusBadRequest ||
					resp.StatusCode == http.StatusRequestedRangeNotSatisfiable)
			})
		}
	})

	// 경계 케이스 테스트
	t.Run("Boundary Cases", func(t *testing.T) {
		// 0바이트 범위
		t.Run("Zero byte range", func(t *testing.T) {
			headers := map[string]string{
				"Range": "bytes=0-0",
			}

			resp, err := env.MakeRequest("GET", yumNginxRPMPath, headers)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.True(t, resp.StatusCode == http.StatusPartialContent || resp.StatusCode == http.StatusOK)
		})

		// 빈 파일 범위 요청
		t.Run("Empty file range", func(t *testing.T) {
			headers := map[string]string{
				"Range": "bytes=0-999",
			}

			// 존재하지 않는 파일에 대한 범위 요청
			resp, err := env.MakeRequest("GET", yumNonExistentRPM, headers)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})
	})

	// 캐시와 Range 요청 상호작용 테스트
	t.Run("Cache and Range Interaction", func(t *testing.T) {
		// 전체 파일 요청으로 캐시에 저장
		resp1, err := env.MakeRequest("GET", yumNginxRPMPath, nil)
		require.NoError(t, err)
		defer func() { _ = resp1.Body.Close() }()
		assert.Equal(t, http.StatusOK, resp1.StatusCode)

		// 캐시된 파일에 대한 Range 요청
		headers := map[string]string{
			"Range": "bytes=0-511",
		}

		resp2, err := env.MakeRequest("GET", yumNginxRPMPath, headers)
		require.NoError(t, err)
		defer func() { _ = resp2.Body.Close() }()

		// Range 요청이 캐시와 올바르게 상호작용하는지 확인
		assert.True(t, resp2.StatusCode == http.StatusPartialContent || resp2.StatusCode == http.StatusOK)
	})

	// Range 요청과 압축 파일 테스트
	t.Run("Range Requests with Compressed Files", func(t *testing.T) {
		headers := map[string]string{
			"Range": "bytes=0-255",
		}

		resp, err := env.MakeRequest("GET", yumPrimaryGzPath, headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// 압축 파일에 대한 Range 요청 처리
		assert.True(t, resp.StatusCode == http.StatusPartialContent || resp.StatusCode == http.StatusOK)

		if resp.StatusCode == http.StatusPartialContent {
			// 압축 파일의 부분 내용이 여전히 유효한지 확인
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Greater(t, len(body), 0)
		}
	})

	// Content-Length와 Range 일관성 테스트
	t.Run("Content-Length and Range Consistency", func(t *testing.T) {
		headers := map[string]string{
			"Range": "bytes=0-1023",
		}

		resp, err := env.MakeRequest("GET", yumNginxRPMPath, headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode == http.StatusPartialContent {
			contentLength := resp.Header.Get("Content-Length")
			if contentLength != "" {
				length, err := strconv.Atoi(contentLength)
				require.NoError(t, err)

				// Content-Length가 요청한 범위 크기와 일치하는지 확인
				assert.LessOrEqual(t, length, 1024) // 0-1023 = 1024 바이트
			}

			// Content-Range 헤더와의 일관성 확인
			contentRange := resp.Header.Get("Content-Range")
			if contentRange != "" {
				assert.Contains(t, contentRange, "bytes")
			}
		}
	})
}
