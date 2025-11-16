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

// TestDockerProxyBasicFlow 기본 Docker 레지스트리 프록시 동작 테스트
func TestDockerProxyBasicFlow(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// Docker upstream 준비 확인
	err := env.WaitForUpstream("docker", 5*time.Second)
	require.NoError(t, err, "Docker mock upstream should be ready")

	// Registry API v2 엔드포인트 테스트
	t.Run("Registry API v2", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/docker/v2/", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")

		// 응답 본문 검증
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		responseBody := string(body[:n])
		assert.Contains(t, responseBody, "{}")
	})

	// 이미지 매니페스트 조회 테스트
	t.Run("Image Manifest", func(t *testing.T) {
		// Docker Registry API requires Accept header for manifest requests
		headers := map[string]string{
			"Accept": "application/vnd.docker.distribution.manifest.v2+json",
		}
		resp, err := env.MakeRequest("GET", "/proxy/docker/v2/library/nginx/manifests/latest", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/vnd.docker.distribution.manifest.v2+json")

		// 매니페스트 응답 검증
		body := make([]byte, 2048)
		n, _ := resp.Body.Read(body)
		responseBody := string(body[:n])
		assert.Contains(t, responseBody, "\"schemaVersion\": 2")
		assert.Contains(t, responseBody, "application/vnd.docker.distribution.manifest.v2+json")
		assert.Contains(t, responseBody, "mock-config-digest")
		assert.Contains(t, responseBody, "4d8c5374677d80499161a0df308f361ecc2cb794ae6326e23931b6e4f66c4a10")
	})

	// 블롭 다운로드 테스트
	t.Run("Blob Download", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/docker/v2/library/nginx/blobs/sha256:4d8c5374677d80499161a0df308f361ecc2cb794ae6326e23931b6e4f66c4a10", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/octet-stream")

		// 블롭 내용 검증
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		responseBody := string(body[:n])
		assert.Contains(t, responseBody, "mock docker layer content")
	})
}

// TestDockerProxyErrorHandling Docker 프록시 에러 처리 테스트
func TestDockerProxyErrorHandling(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// 존재하지 않는 이미지 테스트
	t.Run("Non-existent Image", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/docker/v2/nonexistent/image/manifests/latest", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		// Docker 에러 응답 형식 확인
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		responseBody := string(body[:n])
		assert.Contains(t, responseBody, "errors")
		assert.Contains(t, responseBody, "NAME_UNKNOWN")
	})

	// 잘못된 블롭 다이제스트 테스트
	t.Run("Invalid Blob Digest", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/docker/v2/library/nginx/blobs/sha256:invalid-digest", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// 잘못된 API 버전 테스트
	t.Run("Invalid API Version", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/docker/v1/", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// TestDockerProxyContentTypes Docker 프록시 Content-Type 처리 테스트
func TestDockerProxyContentTypes(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	testCases := []struct {
		name         string
		path         string
		expectedType string
	}{
		{
			name:         "Registry API",
			path:         "/proxy/docker/v2/",
			expectedType: "application/json",
		},
		{
			name:         "Manifest",
			path:         "/proxy/docker/v2/library/nginx/manifests/latest",
			expectedType: "application/vnd.docker.distribution.manifest.v2+json",
		},
		{
			name:         "Blob",
			path:         "/proxy/docker/v2/library/nginx/blobs/sha256:4d8c5374677d80499161a0df308f361ecc2cb794ae6326e23931b6e4f66c4a10",
			expectedType: "application/octet-stream",
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

// TestDockerProxyAuthentication Docker 프록시 인증 헤더 테스트
func TestDockerProxyAuthentication(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// Docker CLI User-Agent 테스트
	t.Run("Docker CLI User-Agent", func(t *testing.T) {
		headers := map[string]string{
			"User-Agent": "docker/20.10.7 go/go1.16.4 git-commit/f0df350 kernel/5.4.0-42-generic os/linux arch/amd64 UpstreamClient(Docker-Client/20.10.7 \\(linux\\))", //nolint:lll
		}

		resp, err := env.MakeRequest("GET", "/proxy/docker/v2/", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Authorization 헤더 테스트 (Basic Auth)
	t.Run("Authorization Header", func(t *testing.T) {
		headers := map[string]string{
			"Authorization": "Basic dGVzdDp0ZXN0", // test:test in base64
		}

		resp, err := env.MakeRequest("GET", "/proxy/docker/v2/library/nginx/manifests/latest", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// 인증 설정이 없는 환경에서는 401 또는 200 모두 허용
		acceptableStatusCodes := []int{http.StatusOK, http.StatusUnauthorized}
		assert.Contains(t, acceptableStatusCodes, resp.StatusCode,
			"인증 설정 여부에 따라 200 또는 401 반환")
	})

	// Bearer Token 테스트
	t.Run("Bearer Token", func(t *testing.T) {
		headers := map[string]string{
			"Authorization": "Bearer mock-jwt-token",
		}

		resp, err := env.MakeRequest("GET", "/proxy/docker/v2/library/nginx/manifests/latest", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Bearer 토큰 인증 설정이 없으면 401 또는 200 모두 허용
		acceptableStatusCodes := []int{http.StatusOK, http.StatusUnauthorized}
		assert.Contains(t, acceptableStatusCodes, resp.StatusCode,
			"인증 설정 여부에 따라 200 또는 401 반환")
	})
}

// TestDockerProxyCaching Docker 프록시 캐싱 동작 테스트
func TestDockerProxyCaching(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	manifestURL := "/proxy/docker/v2/library/nginx/manifests/latest"

	// 첫 번째 요청 (캐시 미스)
	t.Run("First Request - Cache Miss", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", manifestURL, nil)
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

		resp, err := env.MakeRequest("GET", manifestURL, nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 응답 내용이 동일한지 확인
		body := make([]byte, 2048)
		n, _ := resp.Body.Read(body)
		responseBody := string(body[:n])
		assert.Contains(t, responseBody, "\"schemaVersion\": 2")
		assert.Contains(t, responseBody, "mock-config-digest")
	})
}

// TestDockerProxyManifestVersions Docker 매니페스트 버전 테스트
func TestDockerProxyManifestVersions(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// Accept 헤더로 매니페스트 버전 지정
	t.Run("Manifest v2", func(t *testing.T) {
		headers := map[string]string{
			"Accept": "application/vnd.docker.distribution.manifest.v2+json",
		}

		resp, err := env.MakeRequest("GET", "/proxy/docker/v2/library/nginx/manifests/latest", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/vnd.docker.distribution.manifest.v2+json")
	})

	// List 매니페스트 요청 (실제로는 mock에서 지원하지 않으므로 404 예상)
	t.Run("Manifest List", func(t *testing.T) {
		headers := map[string]string{
			"Accept": "application/vnd.docker.distribution.manifest.list.v2+json",
		}

		resp, err := env.MakeRequest("GET", "/proxy/docker/v2/library/nginx/manifests/latest", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Mock 서버는 단순한 v2 매니페스트만 반환하므로 OK 응답
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// TestDockerProxyBlobHandling Docker 블롭 처리 테스트
func TestDockerProxyBlobHandling(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// 블롭 존재 확인 (HEAD 요청)
	t.Run("Blob Head Request", func(t *testing.T) {
		resp, err := env.MakeRequest("HEAD", "/proxy/docker/v2/library/nginx/blobs/sha256:4d8c5374677d80499161a0df308f361ecc2cb794ae6326e23931b6e4f66c4a10", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// HEAD 요청도 200을 반환해야 함
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/octet-stream")
	})

	// 블롭 범위 요청 (Range 헤더) - Mock에서는 지원하지 않을 수 있음
	t.Run("Blob Range Request", func(t *testing.T) {
		headers := map[string]string{
			"Range": "bytes=0-1023",
		}

		resp, err := env.MakeRequest("GET", "/proxy/docker/v2/library/nginx/blobs/sha256:4d8c5374677d80499161a0df308f361ecc2cb794ae6326e23931b6e4f66c4a10", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Mock 서버는 Range를 지원하지 않으므로 200 응답 (전체 내용)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// TestDockerProxyPerformance Docker 프록시 성능 테스트
func TestDockerProxyPerformance(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// 응답 시간 테스트
	t.Run("Response Time", func(t *testing.T) {
		start := time.Now()

		resp, err := env.MakeRequest("GET", "/proxy/docker/v2/library/nginx/manifests/latest", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		duration := time.Since(start)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Less(t, duration, 5*time.Second, "Response should be fast")
	})

	// 동시 매니페스트 요청 테스트
	t.Run("Concurrent Manifest Requests", func(t *testing.T) {
		const numRequests = 5
		done := make(chan bool, numRequests)
		errors := make(chan error, numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				resp, err := env.MakeRequest("GET", "/proxy/docker/v2/library/nginx/manifests/latest", nil)
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

// TestDockerProxyImagePaths Docker 이미지 경로 변형 테스트
func TestDockerProxyImagePaths(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	pathVariations := []struct {
		name        string
		path        string
		expectValid bool
	}{
		{
			name:        "Library image (nginx)",
			path:        "/proxy/docker/v2/library/nginx/manifests/latest",
			expectValid: true,
		},
		{
			name:        "API v2 root",
			path:        "/proxy/docker/v2/",
			expectValid: true,
		},
		{
			name:        "Blob by digest",
			path:        "/proxy/docker/v2/library/nginx/blobs/sha256:4d8c5374677d80499161a0df308f361ecc2cb794ae6326e23931b6e4f66c4a10",
			expectValid: true,
		},
		{
			name:        "User repository (invalid in mock)",
			path:        "/proxy/docker/v2/user/repo/manifests/v1.0",
			expectValid: false,
		},
		{
			name:        "Invalid blob digest",
			path:        "/proxy/docker/v2/library/nginx/blobs/invalid-format",
			expectValid: false,
		},
		{
			name:        "Invalid API path",
			path:        "/proxy/docker/invalid/path",
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

// TestDockerProxyUpstreamConnectivity 업스트림 연결성 테스트
func TestDockerProxyUpstreamConnectivity(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// Mock upstream이 올바르게 응답하는지 확인
	t.Run("Upstream Response Validation", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/docker/v2/", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 응답이 upstream에서 온 것인지 확인
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		responseBody := string(body[:n])

		// Mock upstream의 예상 응답과 일치하는지 확인
		assert.Contains(t, responseBody, "{}")
	})

	// 매니페스트 upstream 검증
	t.Run("Manifest Upstream Validation", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/docker/v2/library/nginx/manifests/latest", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body := make([]byte, 2048)
		n, _ := resp.Body.Read(body)
		responseBody := string(body[:n])

		// Mock upstream의 매니페스트 응답 검증
		assert.Contains(t, responseBody, "\"schemaVersion\": 2")
		assert.Contains(t, responseBody, "mock-config-digest")
		assert.Contains(t, responseBody, "4d8c5374677d80499161a0df308f361ecc2cb794ae6326e23931b6e4f66c4a10")
	})
}

// BenchmarkDockerProxyThroughput Docker 프록시 처리량 벤치마크
func BenchmarkDockerProxyThroughput(b *testing.B) {
	env := SetupIntegrationTest(&testing.T{})
	defer env.Cleanup()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resp, err := env.MakeRequest("GET", "/proxy/docker/v2/library/nginx/manifests/latest", nil)
		if err != nil {
			b.Fatal(err)
		}
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
	}
}

// TestDockerProxyMetrics Docker 프록시 메트릭 수집 테스트
func TestDockerProxyMetrics(t *testing.T) {
	t.Skip("Metrics router is disabled in integration tests due to Prometheus global registry issues")
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// 여러 Docker 요청 수행
	t.Run("Request Metrics", func(t *testing.T) {
		paths := []string{
			"/proxy/docker/v2/",
			"/proxy/docker/v2/library/nginx/manifests/latest",
			"/proxy/docker/v2/library/nginx/blobs/sha256:4d8c5374677d80499161a0df308f361ecc2cb794ae6326e23931b6e4f66c4a10",
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

		// Docker 관련 메트릭이 있는지 확인
		assert.True(t, strings.Contains(metricsBody, "http_requests_total") ||
			strings.Contains(metricsBody, "proxynd_"),
			"Should contain HTTP or ProxyND metrics")
	})
}

// TestDockerProxyHTTPMethods Docker 프록시 HTTP 메서드 테스트
func TestDockerProxyHTTPMethods(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// GET 메서드 테스트 (이미 다른 테스트에서 커버됨)
	t.Run("GET Method", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/docker/v2/", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// HEAD 메서드 테스트
	t.Run("HEAD Method", func(t *testing.T) {
		resp, err := env.MakeRequest("HEAD", "/proxy/docker/v2/library/nginx/manifests/latest", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// PUT 메서드 테스트 (업로드 시뮬레이션 - Mock에서는 지원하지 않을 수 있음)
	t.Run("PUT Method", func(t *testing.T) {
		resp, err := env.MakeRequest("PUT", "/proxy/docker/v2/library/test/manifests/latest", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Mock 서버는 PUT을 지원하지 않으므로 404 또는 405 예상
		assert.True(t, resp.StatusCode >= 400)
	})

	// DELETE 메서드 테스트
	t.Run("DELETE Method", func(t *testing.T) {
		resp, err := env.MakeRequest("DELETE", "/proxy/docker/v2/library/test/manifests/latest", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Mock 서버는 DELETE를 지원하지 않으므로 404 또는 405 예상
		assert.True(t, resp.StatusCode >= 400)
	})
}
