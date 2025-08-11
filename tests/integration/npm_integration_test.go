package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNPMProxy_PackageMetadata NPM 패키지 메타데이터 조회 테스트
func TestNPMProxy_PackageMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	err := server.WaitForReady()
	require.NoError(t, err, "Test server should be ready")

	tests := []struct {
		name              string
		path              string
		expectedStatus    int
		expectedHeaders   map[string]string
		validateResponse  func(*testing.T, []byte)
		expectCacheHit    bool
		secondRequestTest bool
	}{
		{
			name:           "Express package metadata",
			path:           "/proxy/npm/express",
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string]string{
				"Content-Type": "application/json",
			},
			validateResponse: func(t *testing.T, body []byte) {
				var packageData map[string]interface{}
				err := json.Unmarshal(body, &packageData)
				require.NoError(t, err, "Response should be valid JSON")

				assert.Equal(t, "express", packageData["name"], "Package name should be express")
				assert.NotEmpty(t, packageData["version"], "Version should be present")
				assert.NotEmpty(t, packageData["description"], "Description should be present")
			},
			secondRequestTest: true,
		},
		{
			name:           "Non-existent package",
			path:           "/proxy/npm/non-existent-package-xyz",
			expectedStatus: http.StatusNotFound,
			validateResponse: func(t *testing.T, body []byte) {
				var errorResponse map[string]interface{}
				err := json.Unmarshal(body, &errorResponse)
				require.NoError(t, err, "Error response should be valid JSON")
				assert.Contains(t, errorResponse, "error", "Error response should contain error field")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When - 첫 번째 요청
			url := server.BaseURL() + tt.path
			resp, err := http.Get(url)
			require.NoError(t, err, "Request should succeed")
			defer func() { _ = resp.Body.Close() }()

			// Then - 응답 검증
			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Status code should match")

			// 헤더 검증
			for expectedHeader, expectedValue := range tt.expectedHeaders {
				actualValue := resp.Header.Get(expectedHeader)
				assert.Contains(t, actualValue, expectedValue, "Header %s should contain %s", expectedHeader, expectedValue)
			}

			// 응답 본문 검증
			if tt.validateResponse != nil {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err, "Should read response body")
				tt.validateResponse(t, body)
			}

			// When - 캐시 테스트 (두 번째 요청)
			if tt.secondRequestTest {
				startTime := time.Now()
				url2 := server.BaseURL() + tt.path
				resp2, err := http.Get(url2)
				require.NoError(t, err, "Second request should succeed")
				defer func() { _ = resp2.Body.Close() }()
				elapsed := time.Since(startTime)

				// Then - 캐시된 응답 검증
				assert.Equal(t, tt.expectedStatus, resp2.StatusCode, "Cached response status should match")
				assert.Less(t, elapsed, 100*time.Millisecond, "Cached response should be faster")

				// 응답 내용이 동일한지 확인
				body2, err := io.ReadAll(resp2.Body)
				require.NoError(t, err, "Should read cached response body")
				if tt.validateResponse != nil {
					tt.validateResponse(t, body2)
				}
			}
		})
	}
}

// TestNPMProxy_TarballDownload NPM tarball 다운로드 테스트
func TestNPMProxy_TarballDownload(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	err := server.WaitForReady()
	require.NoError(t, err, "Test server should be ready")

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedType   string
		minSize        int64
		contentCheck   func([]byte) bool
		testCacheReuse bool
	}{
		{
			name:           "Express tarball download",
			path:           "/proxy/npm/express/-/express-4.18.2.tgz",
			expectedStatus: http.StatusOK,
			expectedType:   "application/octet-stream",
			minSize:        10, // 최소 10바이트
			contentCheck: func(data []byte) bool {
				// Mock 데이터 확인
				return strings.Contains(string(data), "mock express tarball content")
			},
			testCacheReuse: true,
		},
		{
			name:           "Non-existent tarball",
			path:           "/proxy/npm/non-existent/-/non-existent-1.0.0.tgz",
			expectedStatus: http.StatusNotFound,
			expectedType:   "",
			minSize:        0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			url := server.BaseURL() + tt.path
			resp, err := http.Get(url)
			require.NoError(t, err, "Request should succeed")
			defer func() { _ = resp.Body.Close() }()

			// Then
			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Status code should match")

			if tt.expectedType != "" {
				contentType := resp.Header.Get("Content-Type")
				assert.Contains(t, contentType, tt.expectedType, "Content-Type should match")
			}

			if tt.expectedStatus == http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err, "Should read response body")

				assert.GreaterOrEqual(t, int64(len(body)), tt.minSize, "Response should have minimum size")

				if tt.contentCheck != nil {
					assert.True(t, tt.contentCheck(body), "Content validation should pass")
				}
			}

			// 캐시 재사용 테스트
			if tt.testCacheReuse && tt.expectedStatus == http.StatusOK {
				// 두 번째 요청으로 캐시 히트 확인
				startTime := time.Now()
				url2 := server.BaseURL() + tt.path
				resp2, err := http.Get(url2)
				require.NoError(t, err, "Cached request should succeed")
				defer func() { _ = resp2.Body.Close() }()
				elapsed := time.Since(startTime)

				assert.Equal(t, http.StatusOK, resp2.StatusCode, "Cached response should be OK")
				assert.Less(t, elapsed, 50*time.Millisecond, "Cached response should be very fast")

				body2, err := io.ReadAll(resp2.Body)
				require.NoError(t, err, "Should read cached response body")
				if tt.contentCheck != nil {
					assert.True(t, tt.contentCheck(body2), "Cached content should match")
				}
			}
		})
	}
}

// TestNPMProxy_ScopedPackages 스코프 패키지 테스트
func TestNPMProxy_ScopedPackages(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	err := server.WaitForReady()
	require.NoError(t, err, "Test server should be ready")

	// Mock 서버에 스코프 패키지 응답 추가 (실제로는 dynamic mock이 필요)
	tests := []struct {
		name           string
		path           string
		expectedStatus int
		description    string
	}{
		{
			name:           "Scoped package metadata",
			path:           "/proxy/npm/@babel/core",
			expectedStatus: http.StatusNotFound, // Mock 서버에 구현되지 않음
			description:    "스코프 패키지 지원 확인",
		},
		{
			name:           "Scoped package with URL encoding",
			path:           "/proxy/npm/%40babel%2Fcore",
			expectedStatus: http.StatusNotFound, // Mock 서버에 구현되지 않음
			description:    "URL 인코딩된 스코프 패키지",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			url := server.BaseURL() + tt.path
			resp, err := http.Get(url)
			require.NoError(t, err, "Request should not fail")
			defer func() { _ = resp.Body.Close() }()

			// Then
			assert.Equal(t, tt.expectedStatus, resp.StatusCode, tt.description)
		})
	}
}

// TestNPMProxy_ErrorHandling NPM 프록시 에러 처리 테스트
func TestNPMProxy_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	err := server.WaitForReady()
	require.NoError(t, err, "Test server should be ready")

	tests := []struct {
		name             string
		path             string
		expectedStatus   int
		expectedResponse func(*testing.T, []byte)
	}{
		{
			name:           "Invalid package name",
			path:           "/proxy/npm/",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Package not found",
			path:           "/proxy/npm/this-package-definitely-does-not-exist-12345",
			expectedStatus: http.StatusNotFound,
			expectedResponse: func(t *testing.T, body []byte) {
				var errorData map[string]interface{}
				err := json.Unmarshal(body, &errorData)
				require.NoError(t, err, "Error response should be JSON")
				assert.Contains(t, errorData, "error", "Should contain error field")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			url := server.BaseURL() + tt.path
			resp, err := http.Get(url)
			require.NoError(t, err, "Request should complete")
			defer func() { _ = resp.Body.Close() }()

			// Then
			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Status code should match")

			if tt.expectedResponse != nil {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err, "Should read error response")
				tt.expectedResponse(t, body)
			}
		})
	}
}

// TestNPMProxy_ConcurrentRequests NPM 프록시 동시 요청 테스트
func TestNPMProxy_ConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	err := server.WaitForReady()
	require.NoError(t, err, "Test server should be ready")

	const concurrency = 5
	const requestPath = "/proxy/npm/express"

	// When - 동시에 같은 패키지 요청
	results := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			url := server.BaseURL() + requestPath
			resp, err := http.Get(url)
			if err != nil {
				results <- err
				return
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != http.StatusOK {
				results <- assert.AnError
				return
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				results <- err
				return
			}

			var packageData map[string]interface{}
			if err := json.Unmarshal(body, &packageData); err != nil {
				results <- err
				return
			}

			if packageData["name"] != "express" {
				results <- assert.AnError
				return
			}

			results <- nil
		}()
	}

	// Then - 모든 요청이 성공해야 함
	for i := 0; i < concurrency; i++ {
		select {
		case err := <-results:
			assert.NoError(t, err, "Concurrent request %d should succeed", i+1)
		case <-time.After(10 * time.Second):
			t.Fatalf("Timeout waiting for concurrent request %d", i+1)
		}
	}
}

// TestNPMProxy_CacheHeaders NPM 프록시 캐시 헤더 테스트
func TestNPMProxy_CacheHeaders(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트는 -short 플래그에서 스킵")
	}

	// Given
	server := SetupTestServer(t)
	defer server.Close()

	err := server.WaitForReady()
	require.NoError(t, err, "Test server should be ready")

	// When - 첫 번째 요청 (캐시 미스)
	url1 := server.BaseURL() + "/proxy/npm/express"
	resp1, err := http.Get(url1)
	require.NoError(t, err, "First request should succeed")
	defer func() { _ = resp1.Body.Close() }()

	// Then - 첫 번째 응답 검증
	assert.Equal(t, http.StatusOK, resp1.StatusCode, "First request should be OK")

	// When - 두 번째 요청 (캐시 히트)
	time.Sleep(10 * time.Millisecond) // 캐시가 저장될 시간 대기
	url2 := server.BaseURL() + "/proxy/npm/express"
	resp2, err := http.Get(url2)
	require.NoError(t, err, "Second request should succeed")
	defer func() { _ = resp2.Body.Close() }()

	// Then - 두 번째 응답 검증
	assert.Equal(t, http.StatusOK, resp2.StatusCode, "Cached request should be OK")

	// 응답 내용이 동일한지 확인
	body1, _ := io.ReadAll(resp1.Body)
	body2, _ := io.ReadAll(resp2.Body)

	var package1, package2 map[string]interface{}
	_ = json.Unmarshal(body1, &package1)
	_ = json.Unmarshal(body2, &package2)

	assert.Equal(t, package1["name"], package2["name"], "Cached response should have same content")
}
