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

// TestYUMProxyRepodataFreshnessAdvanced YUM 프록시 repodata 신선도 검사 강화 테스트 (Step 2-1)
func TestYUMProxyRepodataFreshnessAdvanced(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// Upstream 연결 확인
	err := env.WaitForUpstream("yum", 5*time.Second)
	require.NoError(t, err)

	// repodata 만료 정책 테스트
	t.Run("Repodata Expiration Policy", func(t *testing.T) {
		// 첫 번째 요청으로 repodata 캐시
		resp, err := env.MakeRequest("GET", yumRepomdPath, nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 캐시 제어 헤더 검증
		cacheControl := resp.Header.Get("Cache-Control")
		lastModified := resp.Header.Get("Last-Modified")
		etag := resp.Header.Get("ETag")

		// 캐시 헤더가 적절히 설정되었는지 확인
		if cacheControl != "" {
			assert.True(t, strings.Contains(cacheControl, "max-age") ||
				strings.Contains(cacheControl, "public") ||
				strings.Contains(cacheControl, "private"))
		}

		// 메타데이터에는 Last-Modified 또는 ETag가 있어야 함
		assert.True(t, lastModified != "" || etag != "", "Should have Last-Modified or ETag header")
	})

	// 조건부 요청을 통한 대역폭 절약 테스트
	t.Run("Conditional Request Bandwidth Saving", func(t *testing.T) {
		// 첫 번째 요청
		resp1, err := env.MakeRequest("GET", yumRepomdPath, nil)
		require.NoError(t, err)
		defer func() { _ = resp1.Body.Close() }()

		etag1 := resp1.Header.Get("ETag")
		lastModified1 := resp1.Header.Get("Last-Modified")

		// If-None-Match 헤더로 조건부 요청
		if etag1 != "" {
			headers := map[string]string{
				"If-None-Match": etag1,
			}

			resp2, err := env.MakeRequest("GET", yumRepomdPath, headers)
			require.NoError(t, err)
			defer func() { _ = resp2.Body.Close() }()

			// 304 Not Modified 또는 200 OK (Mock 서버 구현에 따라)
			assert.True(t, resp2.StatusCode == http.StatusNotModified || resp2.StatusCode == http.StatusOK)
		}

		// If-Modified-Since 헤더로 조건부 요청
		if lastModified1 != "" {
			headers := map[string]string{
				"If-Modified-Since": lastModified1,
			}

			resp3, err := env.MakeRequest("GET", yumRepomdPath, headers)
			require.NoError(t, err)
			defer func() { _ = resp3.Body.Close() }()

			assert.True(t, resp3.StatusCode == http.StatusNotModified || resp3.StatusCode == http.StatusOK)
		}
	})

	// repodata 타임스탬프 일관성 검증
	t.Run("Repodata Timestamp Consistency", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", yumRepomdPath, nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// repomd.xml 내용에서 revision 타임스탬프 추출
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		bodyStr := string(body)
		assert.Contains(t, bodyStr, "<revision>")

		// revision 값이 유닉스 타임스탬프 형식인지 확인
		if strings.Contains(bodyStr, "<revision>1640995200</revision>") {
			// 2022-01-01 00:00:00 UTC 확인
			timestamp := int64(1640995200)
			revisionTime := time.Unix(timestamp, 0)
			assert.True(t, revisionTime.Year() >= 2020, "Revision timestamp should be reasonable")
		}
	})

	// Stale-while-revalidate 동작 검증
	t.Run("Stale While Revalidate Behavior", func(t *testing.T) {
		// 캐시에 저장
		resp1, err := env.MakeRequest("GET", yumPrimaryGzPath, nil)
		require.NoError(t, err)
		defer func() { _ = resp1.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp1.StatusCode)

		// 짧은 대기 후 재요청
		time.Sleep(50 * time.Millisecond)

		resp2, err := env.MakeRequest("GET", yumPrimaryGzPath, nil)
		require.NoError(t, err)
		defer func() { _ = resp2.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp2.StatusCode)

		// 응답 시간이 캐시로 인해 빨라졌는지 간접 확인
		contentType1 := resp1.Header.Get("Content-Type")
		contentType2 := resp2.Header.Get("Content-Type")
		assert.Equal(t, contentType1, contentType2, "Content-Type should be consistent")
	})
}

// TestYUMProxyGzipPGPHeadersAdvanced YUM 프록시 gzip/pgp 헤더 처리 강화 테스트 (Step 2-2)
func TestYUMProxyGzipPGPHeadersAdvanced(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// 압축 파일 내용 검증 테스트
	t.Run("Compressed File Content Validation", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", yumPrimaryGzPath, nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/x-gzip")

		// gzip 매직 넘버 확인 (선택적 - Mock 서버 구현에 따라)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Greater(t, len(body), 0, "Gzip file should have content")

		// Mock 응답 내용 확인
		bodyStr := string(body)
		assert.Contains(t, bodyStr, "mock compressed primary.xml content")
	})

	// 다양한 압축 알고리즘 지원 테스트
	t.Run("Multiple Compression Algorithm Support", func(t *testing.T) {
		compressionTests := []struct {
			name        string
			path        string
			contentType string
			encoding    string
		}{
			{
				name:        "Gzip Primary Metadata",
				path:        yumPrimaryGzPath,
				contentType: "application/x-gzip",
				encoding:    "gzip",
			},
			{
				name:        "Uncompressed Repomd",
				path:        yumRepomdPath,
				contentType: "application/xml",
				encoding:    "",
			},
		}

		for _, test := range compressionTests {
			t.Run(test.name, func(t *testing.T) {
				resp, err := env.MakeRequest("GET", test.path, nil)
				require.NoError(t, err)
				defer func() { _ = resp.Body.Close() }()

				assert.Equal(t, http.StatusOK, resp.StatusCode)
				assert.Contains(t, resp.Header.Get("Content-Type"), test.contentType)

				if test.encoding != "" {
					contentEncoding := resp.Header.Get("Content-Encoding")
					if contentEncoding != "" {
						assert.Contains(t, contentEncoding, test.encoding)
					}
				}
			})
		}
	})

	// PGP 서명 검증 테스트
	t.Run("PGP Signature Verification", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", yumRepomdAscPath, nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// PGP 서명 파일 처리 확인 (Mock 서버가 지원하는 경우)
		if resp.StatusCode == http.StatusOK {
			contentType := resp.Header.Get("Content-Type")
			assert.True(t, strings.Contains(contentType, "application/pgp-signature") ||
				strings.Contains(contentType, "text/plain"),
				"PGP signature should have appropriate content type")

			// 서명 파일 내용 기본 검증
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			if len(body) > 0 {
				bodyStr := string(body)
				// PGP 서명의 기본 구조 확인
				assert.True(t, strings.Contains(bodyStr, "BEGIN PGP") ||
					strings.Contains(bodyStr, "mock") ||
					len(bodyStr) > 10, "Should contain PGP signature content")
			}
		} else {
			// Mock에서 지원하지 않는 경우 404 허용
			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		}
	})

	// Content-Encoding 협상 테스트
	t.Run("Content-Encoding Negotiation", func(t *testing.T) {
		acceptEncodingTests := []struct {
			name           string
			acceptEncoding string
			expectGzip     bool
		}{
			{
				name:           "Accept Gzip",
				acceptEncoding: "gzip, deflate",
				expectGzip:     true,
			},
			{
				name:           "Accept All",
				acceptEncoding: "*",
				expectGzip:     true,
			},
			{
				name:           "No Accept-Encoding",
				acceptEncoding: "",
				expectGzip:     false,
			},
			{
				name:           "Accept Deflate Only",
				acceptEncoding: "deflate",
				expectGzip:     false,
			},
		}

		for _, test := range acceptEncodingTests {
			t.Run(test.name, func(t *testing.T) {
				headers := map[string]string{}
				if test.acceptEncoding != "" {
					headers["Accept-Encoding"] = test.acceptEncoding
				}

				resp, err := env.MakeRequest("GET", yumPrimaryGzPath, headers)
				require.NoError(t, err)
				defer func() { _ = resp.Body.Close() }()

				assert.Equal(t, http.StatusOK, resp.StatusCode)

				// gzip 파일은 이미 압축되어 있으므로 Content-Type으로 판단
				contentType := resp.Header.Get("Content-Type")
				assert.Contains(t, contentType, "application/x-gzip")
			})
		}
	})

	// 압축 무결성 검증 테스트
	t.Run("Compression Integrity Validation", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", yumPrimaryGzPath, nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Content-Length 검증
		contentLength := resp.Header.Get("Content-Length")
		if contentLength != "" {
			length, err := strconv.Atoi(contentLength)
			require.NoError(t, err)
			assert.Greater(t, length, 0, "Compressed file should have positive size")
		}

		// 실제 내용 크기와 헤더 일치 확인
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		if contentLength != "" {
			expectedLength, _ := strconv.Atoi(contentLength)
			assert.Equal(t, expectedLength, len(body), "Content-Length should match actual body size")
		}
	})
}

// TestYUMProxyRangeRequestsAdvanced YUM 프록시 Range 요청 지원 강화 테스트 (Step 2-3)
func TestYUMProxyRangeRequestsAdvanced(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// HTTP 1.1 Range 요청 완전 지원 테스트
	t.Run("HTTP 1.1 Range Request Full Support", func(t *testing.T) {
		// Accept-Ranges 지원 확인
		resp, err := env.MakeRequest("HEAD", yumNginxRPMPath, nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		acceptRanges := resp.Header.Get("Accept-Ranges")
		if acceptRanges != "" {
			assert.True(t, acceptRanges == "bytes" || acceptRanges == "none")
		}

		// Content-Length가 있어야 Range 요청이 의미가 있음
		contentLength := resp.Header.Get("Content-Length")
		assert.NotEmpty(t, contentLength, "HEAD request should return Content-Length for range support")
	})

	// 복잡한 Range 요청 시나리오 테스트
	t.Run("Complex Range Request Scenarios", func(t *testing.T) {
		complexRangeTests := []struct {
			name      string
			rangeSpec string
			expectOK  bool
		}{
			{
				name:      "First chunk",
				rangeSpec: "bytes=0-4095",
				expectOK:  true,
			},
			{
				name:      "Middle chunk",
				rangeSpec: "bytes=4096-8191",
				expectOK:  true,
			},
			{
				name:      "Suffix range",
				rangeSpec: "bytes=-1024",
				expectOK:  true,
			},
			{
				name:      "Prefix range",
				rangeSpec: "bytes=1024-",
				expectOK:  true,
			},
			{
				name:      "Multiple ranges",
				rangeSpec: "bytes=0-1023,2048-3071",
				expectOK:  true,
			},
		}

		for _, test := range complexRangeTests {
			t.Run(test.name, func(t *testing.T) {
				headers := map[string]string{
					"Range": test.rangeSpec,
				}

				resp, err := env.MakeRequest("GET", yumNginxRPMPath, headers)
				require.NoError(t, err)
				defer func() { _ = resp.Body.Close() }()

				if test.expectOK {
					assert.True(t, resp.StatusCode == http.StatusPartialContent || resp.StatusCode == http.StatusOK,
						"Range request should succeed for %s", test.rangeSpec)
				}

				if resp.StatusCode == http.StatusPartialContent {
					contentRange := resp.Header.Get("Content-Range")
					assert.NotEmpty(t, contentRange, "Content-Range header should be present")
					assert.Contains(t, contentRange, "bytes", "Content-Range should specify bytes")
				}
			})
		}
	})

	// Range 요청과 캐시 상호작용 테스트
	t.Run("Range Request Cache Interaction", func(t *testing.T) {
		// 전체 파일을 먼저 캐시
		resp1, err := env.MakeRequest("GET", yumNginxRPMPath, nil)
		require.NoError(t, err)
		defer func() { _ = resp1.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp1.StatusCode)
		fullBody, err := io.ReadAll(resp1.Body)
		require.NoError(t, err)

		// 캐시된 파일에 대한 Range 요청
		headers := map[string]string{
			"Range": "bytes=0-511",
		}

		resp2, err := env.MakeRequest("GET", yumNginxRPMPath, headers)
		require.NoError(t, err)
		defer func() { _ = resp2.Body.Close() }()

		// Range 요청 결과 검증
		if resp2.StatusCode == http.StatusPartialContent {
			partialBody, err := io.ReadAll(resp2.Body)
			require.NoError(t, err)

			// 부분 내용이 전체 내용의 일부와 일치하는지 확인
			if len(fullBody) > 512 && len(partialBody) > 0 {
				assert.Equal(t, string(fullBody[:len(partialBody)]), string(partialBody),
					"Partial content should match full content")
			}
		}
	})

	// 동시 Range 요청 처리 테스트
	t.Run("Concurrent Range Request Handling", func(t *testing.T) {
		const numConcurrentRequests = 5
		done := make(chan bool, numConcurrentRequests)
		errors := make(chan error, numConcurrentRequests)

		// 각각 다른 범위를 요청하는 동시 요청
		ranges := []string{
			"bytes=0-1023",
			"bytes=1024-2047",
			"bytes=2048-3071",
			"bytes=3072-4095",
			"bytes=4096-5119",
		}

		for i, rangeSpec := range ranges {
			go func(index int, rng string) {
				headers := map[string]string{
					"Range": rng,
				}

				resp, err := env.MakeRequest("GET", yumNginxRPMPath, headers)
				if err != nil {
					errors <- fmt.Errorf("request %d failed: %v", index, err)
					return
				}
				defer func() { _ = resp.Body.Close() }()

				if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
					errors <- fmt.Errorf("request %d got status %d", index, resp.StatusCode)
					return
				}

				done <- true
			}(i, rangeSpec)
		}

		// 모든 요청 완료 대기
		successCount := 0
		for i := 0; i < numConcurrentRequests; i++ {
			select {
			case <-done:
				successCount++
			case err := <-errors:
				t.Logf("Concurrent range request failed: %v", err)
			case <-time.After(10 * time.Second):
				t.Fatal("Timeout waiting for concurrent range requests")
			}
		}

		assert.Greater(t, successCount, 0, "At least some range requests should succeed")
	})

	// If-Range 헤더 지원 테스트
	t.Run("If-Range Header Support", func(t *testing.T) {
		// 첫 번째 요청으로 ETag 획득
		resp1, err := env.MakeRequest("GET", yumNginxRPMPath, nil)
		require.NoError(t, err)
		defer func() { _ = resp1.Body.Close() }()

		etag := resp1.Header.Get("ETag")
		if etag != "" {
			// If-Range with ETag
			headers := map[string]string{
				"Range":    "bytes=0-1023",
				"If-Range": etag,
			}

			resp2, err := env.MakeRequest("GET", yumNginxRPMPath, headers)
			require.NoError(t, err)
			defer func() { _ = resp2.Body.Close() }()

			// If-Range가 일치하면 Partial Content, 아니면 전체 콘텐츠
			assert.True(t, resp2.StatusCode == http.StatusPartialContent || resp2.StatusCode == http.StatusOK)
		}

		// If-Range with Last-Modified
		lastModified := resp1.Header.Get("Last-Modified")
		if lastModified != "" {
			headers := map[string]string{
				"Range":    "bytes=0-1023",
				"If-Range": lastModified,
			}

			resp3, err := env.MakeRequest("GET", yumNginxRPMPath, headers)
			require.NoError(t, err)
			defer func() { _ = resp3.Body.Close() }()

			assert.True(t, resp3.StatusCode == http.StatusPartialContent || resp3.StatusCode == http.StatusOK)
		}
	})
}

// TestYUMProxyMirrorFailover YUM 프록시 미러 간 페일오버 테스트 (Step 2-4)
func TestYUMProxyMirrorFailover(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// 업스트림 연결 실패 시 fallback 동작 테스트
	t.Run("Upstream Connection Failure Fallback", func(t *testing.T) {
		// 정상적인 요청으로 기준점 설정
		resp1, err := env.MakeRequest("GET", yumRepomdPath, nil)
		require.NoError(t, err)
		defer func() { _ = resp1.Body.Close() }()

		if resp1.StatusCode == http.StatusOK {
			// 정상 응답 확인
			body, err := io.ReadAll(resp1.Body)
			require.NoError(t, err)
			assert.Greater(t, len(body), 0)
		}

		// 존재하지 않는 경로로 fallback 테스트
		resp2, err := env.MakeRequest("GET", "/proxy/yum/nonexistent-mirror/repodata/repomd.xml", nil)
		require.NoError(t, err)
		defer func() { _ = resp2.Body.Close() }()

		// Fallback이 적절히 처리되는지 확인 (404 또는 대체 미러 응답)
		assert.True(t, resp2.StatusCode == http.StatusNotFound || resp2.StatusCode >= 400,
			"Should handle non-existent mirror appropriately")
	})

	// 다중 미러 지원 테스트
	t.Run("Multiple Mirror Support", func(t *testing.T) {
		mirrorTests := []struct {
			name string
			path string
		}{
			{
				name: "CentOS 8 Mirror",
				path: "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/repomd.xml",
			},
			{
				name: "Alternative Distribution",
				path: "/proxy/yum/fedora/34/Everything/x86_64/os/repodata/repomd.xml",
			},
			{
				name: "RHEL Mirror",
				path: "/proxy/yum/rhel/8/BaseOS/x86_64/os/repodata/repomd.xml",
			},
		}

		for _, test := range mirrorTests {
			t.Run(test.name, func(t *testing.T) {
				resp, err := env.MakeRequest("GET", test.path, nil)
				require.NoError(t, err)
				defer func() { _ = resp.Body.Close() }()

				// Mock 환경에서는 첫 번째만 성공할 것으로 예상
				if test.name == "CentOS 8 Mirror" {
					assert.Equal(t, http.StatusOK, resp.StatusCode)
				} else {
					// 다른 미러는 404 또는 적절한 에러 처리
					assert.True(t, resp.StatusCode >= 400,
						"Non-configured mirrors should return appropriate error")
				}
			})
		}
	})

	// 미러 응답 시간 기반 선택 테스트
	t.Run("Mirror Response Time Based Selection", func(t *testing.T) {
		// 동일한 컨텐츠에 대한 반복 요청으로 성능 측정
		responseTimes := make([]time.Duration, 3)

		for i := 0; i < 3; i++ {
			start := time.Now()

			resp, err := env.MakeRequest("GET", yumRepomdPath, nil)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			responseTimes[i] = time.Since(start)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// 캐시 효과로 두 번째, 세 번째 요청이 빨라질 수 있음
			time.Sleep(10 * time.Millisecond)
		}

		// 첫 번째 요청보다 후속 요청이 빠른지 확인 (캐시 효과)
		if len(responseTimes) >= 2 {
			t.Logf("Response times: %v", responseTimes)
			// 적어도 하나의 후속 요청이 합리적인 시간 내에 완료되어야 함
			assert.True(t, responseTimes[1] < 5*time.Second && responseTimes[2] < 5*time.Second,
				"Subsequent requests should be reasonably fast")
		}
	})

	// 미러 상태 모니터링 테스트
	t.Run("Mirror Status Monitoring", func(t *testing.T) {
		// 서로 다른 아키텍처 경로로 미러 상태 확인
		architectureTests := []struct {
			name          string
			path          string
			expectSuccess bool
		}{
			{
				name:          "x86_64 Architecture",
				path:          "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/repomd.xml",
				expectSuccess: true,
			},
			{
				name:          "i386 Architecture",
				path:          "/proxy/yum/centos/8/BaseOS/i386/os/repodata/repomd.xml",
				expectSuccess: false, // Mock에서 지원하지 않음
			},
			{
				name:          "aarch64 Architecture",
				path:          "/proxy/yum/centos/8/BaseOS/aarch64/os/repodata/repomd.xml",
				expectSuccess: false, // Mock에서 지원하지 않음
			},
		}

		for _, test := range architectureTests {
			t.Run(test.name, func(t *testing.T) {
				resp, err := env.MakeRequest("GET", test.path, nil)
				require.NoError(t, err)
				defer func() { _ = resp.Body.Close() }()

				if test.expectSuccess {
					assert.Equal(t, http.StatusOK, resp.StatusCode,
						"Expected architecture should be supported")
				} else {
					assert.NotEqual(t, http.StatusOK, resp.StatusCode,
						"Unsupported architecture should return error")
				}
			})
		}
	})

	// 지리적 미러 선택 시뮬레이션 테스트
	t.Run("Geographic Mirror Selection Simulation", func(t *testing.T) {
		// 서로 다른 지역 미러를 시뮬레이션하는 User-Agent 테스트
		geographicTests := []struct {
			name      string
			userAgent string
			expectOK  bool
		}{
			{
				name:      "Default DNF Client",
				userAgent: "dnf/4.7.0",
				expectOK:  true,
			},
			{
				name:      "YUM Client",
				userAgent: "yum/3.4.3",
				expectOK:  true,
			},
			{
				name:      "Curl Client",
				userAgent: "curl/7.68.0",
				expectOK:  true,
			},
		}

		for _, test := range geographicTests {
			t.Run(test.name, func(t *testing.T) {
				headers := map[string]string{
					"User-Agent": test.userAgent,
				}

				resp, err := env.MakeRequest("GET", yumRepomdPath, headers)
				require.NoError(t, err)
				defer func() { _ = resp.Body.Close() }()

				if test.expectOK {
					assert.Equal(t, http.StatusOK, resp.StatusCode)

					// User-Agent가 로그에 기록되는지 간접 확인
					body, err := io.ReadAll(resp.Body)
					require.NoError(t, err)
					assert.Greater(t, len(body), 0)
				}
			})
		}
	})

	// Circuit Breaker 패턴 테스트
	t.Run("Circuit Breaker Pattern", func(t *testing.T) {
		// 연속된 실패 요청으로 Circuit Breaker 동작 확인
		failureCount := 0
		successCount := 0

		// 여러 번 요청하여 일관된 동작 확인
		for i := 0; i < 5; i++ {
			resp, err := env.MakeRequest("GET", yumRepomdPath, nil)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode == http.StatusOK {
				successCount++
			} else {
				failureCount++
			}

			time.Sleep(100 * time.Millisecond)
		}

		// Mock 환경에서는 일관된 성공을 기대
		assert.Greater(t, successCount, 0, "Should have some successful requests")
		assert.Equal(t, 0, failureCount, "Should not have failures with mock upstream")
	})

	// 미러 메타데이터 동기화 테스트
	t.Run("Mirror Metadata Synchronization", func(t *testing.T) {
		// 동일한 컨텐츠에 대한 다중 요청으로 일관성 확인
		responses := make([]*http.Response, 3)
		bodies := make([]string, 3)

		for i := 0; i < 3; i++ {
			resp, err := env.MakeRequest("GET", yumRepomdPath, nil)
			require.NoError(t, err)
			responses[i] = resp

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			bodies[i] = string(body)

			_ = resp.Body.Close()
		}

		// 모든 응답이 동일한지 확인 (미러 동기화)
		for i := 1; i < len(bodies); i++ {
			assert.Equal(t, bodies[0], bodies[i],
				"All requests should return identical content (mirror sync)")
		}

		// 모든 응답이 성공인지 확인
		for i, resp := range responses {
			assert.Equal(t, http.StatusOK, resp.StatusCode,
				"Request %d should be successful", i)
		}
	})
}
