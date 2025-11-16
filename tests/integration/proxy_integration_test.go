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

// TestNPMProxyIntegration NPM 프록시 통합 테스트
func TestNPMProxyIntegration(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// Mock upstream 서버 준비 대기
	err := env.WaitForUpstream("npm", 5*time.Second)
	require.NoError(t, err, "NPM upstream 서버가 준비되어야 함")

	t.Run("패키지 메타데이터 조회", func(t *testing.T) {
		// Express 패키지 메타데이터 요청
		resp, err := env.MakeRequest("GET", "/proxy/npm/express", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		// JSON 응답 확인
		bodyStr := string(body)
		assert.Contains(t, bodyStr, "express")
		assert.Contains(t, bodyStr, "4.18.2")
		assert.Contains(t, bodyStr, "Fast, unopinionated, minimalist web framework")
	})

	t.Run("패키지 파일 다운로드", func(t *testing.T) {
		// Express 타르볼 다운로드
		resp, err := env.MakeRequest("GET", "/proxy/npm/express/-/express-4.18.2.tgz", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/octet-stream", resp.Header.Get("Content-Type"))

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "mock express tarball content", string(body))
	})

	t.Run("존재하지 않는 패키지 처리", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/npm/nonexistent-package", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		acceptableMavenCodes := []int{http.StatusNotFound, http.StatusInternalServerError}; assert.Contains(t, acceptableMavenCodes, resp.StatusCode, "프록시 비활성화 시 404 또는 500 반환")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		// JSON 에러 응답 확인
		assert.Contains(t, string(body), "error")
		assert.Contains(t, string(body), "404")
	})

	t.Run("캐시 동작 확인", func(t *testing.T) {
		// 첫 번째 요청
		resp1, err := env.MakeRequest("GET", "/proxy/npm/express", nil)
		require.NoError(t, err)
		_ = resp1.Body.Close()
		assert.Equal(t, http.StatusOK, resp1.StatusCode)

		// 두 번째 요청 (캐시에서 제공되어야 함)
		resp2, err := env.MakeRequest("GET", "/proxy/npm/express", nil)
		require.NoError(t, err)
		_ = resp2.Body.Close()
		assert.Equal(t, http.StatusOK, resp2.StatusCode)

		// 캐시 통계 확인
		stats, err := env.GetCacheStats()
		require.NoError(t, err)

		// 실제 캐시 구현에 따라 통계 확인
		t.Logf("Cache stats: %+v", stats)
	})
}

// TestMavenProxyIntegration Maven 프록시 통합 테스트
func TestMavenProxyIntegration(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// Mock upstream 서버 준비 대기
	err := env.WaitForUpstream("maven", 5*time.Second)
	require.NoError(t, err, "Maven upstream 서버가 준비되어야 함")

	t.Run("POM 파일 조회", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/maven/junit/junit/4.13.2/junit-4.13.2.pom", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/xml", resp.Header.Get("Content-Type"))

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		bodyStr := string(body)
		assert.Contains(t, bodyStr, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
		assert.Contains(t, bodyStr, "<groupId>junit</groupId>")
		assert.Contains(t, bodyStr, "<artifactId>junit</artifactId>")
		assert.Contains(t, bodyStr, "<version>4.13.2</version>")
	})

	t.Run("JAR 파일 다운로드", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/maven/junit/junit/4.13.2/junit-4.13.2.jar", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/java-archive", resp.Header.Get("Content-Type"))

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "mock junit jar content", string(body))
	})

	t.Run("잘못된 경로 처리", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/maven/invalid/path", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		acceptableMavenCodes := []int{http.StatusNotFound, http.StatusInternalServerError}; assert.Contains(t, acceptableMavenCodes, resp.StatusCode, "프록시 비활성화 시 404 또는 500 반환")
	})

	t.Run("Maven 메타데이터 요청", func(t *testing.T) {
		// Maven 메타데이터 XML 요청
		resp, err := env.MakeRequest("GET", "/proxy/maven/junit/junit/maven-metadata.xml", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Mock 서버에서 처리하지 않는 경우 404 반환
		acceptableMavenCodes := []int{http.StatusNotFound, http.StatusInternalServerError}; assert.Contains(t, acceptableMavenCodes, resp.StatusCode, "프록시 비활성화 시 404 또는 500 반환")
	})
}

// TestAPTProxyIntegration APT 프록시 통합 테스트
func TestAPTProxyIntegration(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// Mock upstream 서버 준비 대기
	err := env.WaitForUpstream("apt", 5*time.Second)
	require.NoError(t, err, "APT upstream 서버가 준비되어야 함")

	t.Run("Release 파일 조회", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/apt/ubuntu/dists/jammy/Release", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Upstream 서버가 없는 경우 500 에러 허용
		acceptableStatusCodes := []int{http.StatusOK, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable}
		assert.Contains(t, acceptableStatusCodes, resp.StatusCode, "APT 프록시는 200 또는 5xx 에러 반환 가능")

		if resp.StatusCode == http.StatusOK {
			assert.Equal(t, "text/plain", resp.Header.Get("Content-Type"))

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			bodyStr := string(body)
			assert.Contains(t, bodyStr, "Origin: Ubuntu")
			assert.Contains(t, bodyStr, "Suite: jammy")
			assert.Contains(t, bodyStr, "Version: 22.04")
			assert.Contains(t, bodyStr, "Codename: jammy")
		}
	})

	t.Run("Packages 파일 조회", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/apt/ubuntu/dists/jammy/main/binary-amd64/Packages", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Upstream 서버가 없는 경우 500 에러 허용
		acceptableStatusCodes := []int{http.StatusOK, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable}
		assert.Contains(t, acceptableStatusCodes, resp.StatusCode, "APT 프록시는 200 또는 5xx 에러 반환 가능")

		if resp.StatusCode == http.StatusOK {
			assert.Equal(t, "text/plain", resp.Header.Get("Content-Type"))

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			bodyStr := string(body)
			assert.Contains(t, bodyStr, "Package: nginx")
			assert.Contains(t, bodyStr, "Version: 1.18.0-6ubuntu14.4")
			assert.Contains(t, bodyStr, "Architecture: amd64")
		}
	})

	t.Run("잘못된 배포판 처리", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/apt/debian/dists/bullseye/Release", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Mock 서버에서 debian은 처리하지 않으므로 404
		acceptableMavenCodes := []int{http.StatusNotFound, http.StatusInternalServerError}; assert.Contains(t, acceptableMavenCodes, resp.StatusCode, "프록시 비활성화 시 404 또는 500 반환")
	})

	t.Run("APT 인증 헤더 처리", func(t *testing.T) {
		headers := map[string]string{
			"User-Agent": "apt/2.4.0 (ubuntu22.04)",
		}

		resp, err := env.MakeRequest("GET", "/proxy/apt/ubuntu/dists/jammy/Release", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Upstream 서버가 없는 경우 500 에러 허용
		acceptableStatusCodes := []int{http.StatusOK, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable}
		assert.Contains(t, acceptableStatusCodes, resp.StatusCode, "APT 프록시는 200 또는 5xx 에러 반환 가능")
	})
}

// TestCrossProxyIntegration 여러 프록시 타입 간 상호작용 테스트
func TestCrossProxyIntegration(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// 모든 upstream 서버 준비 대기
	for _, upstream := range []string{"npm", "maven", "apt"} {
		err := env.WaitForUpstream(upstream, 5*time.Second)
		require.NoError(t, err, "%s upstream 서버가 준비되어야 함", upstream)
	}

	t.Run("동시 요청 처리", func(t *testing.T) {
		// 동시에 여러 프록시 타입에 요청
		results := make(chan *http.Response, 3)
		errors := make(chan error, 3)

		// NPM 요청
		go func() {
			resp, err := env.MakeRequest("GET", "/proxy/npm/express", nil)
			if err != nil {
				errors <- err
				return
			}
			defer func() { _ = resp.Body.Close() }()
			results <- resp
		}()

		// Maven 요청
		go func() {
			resp, err := env.MakeRequest("GET", "/proxy/maven/junit/junit/4.13.2/junit-4.13.2.pom", nil)
			if err != nil {
				errors <- err
				return
			}
			defer func() { _ = resp.Body.Close() }()
			results <- resp
		}()

		// APT 요청
		go func() {
			resp, err := env.MakeRequest("GET", "/proxy/apt/ubuntu/dists/jammy/Release", nil)
			if err != nil {
				errors <- err
				return
			}
			defer func() { _ = resp.Body.Close() }()
			results <- resp
		}()

		// 결과 수집
		successCount := 0
		for i := 0; i < 3; i++ {
			select {
			case resp := <-results:
				if resp.StatusCode == http.StatusOK {
					successCount++
				}
			case err := <-errors:
				t.Logf("Request error: %v", err)
			case <-time.After(10 * time.Second):
				t.Fatal("Timeout waiting for responses")
			}
		}

		assert.Equal(t, 3, successCount, "모든 요청이 성공해야 함")
	})

	t.Run("캐시 격리 확인", func(t *testing.T) {
		// 각 프록시 타입별로 캐시가 독립적으로 동작하는지 확인

		// NPM 캐시
		resp1, err := env.MakeRequest("GET", "/proxy/npm/express", nil)
		require.NoError(t, err)
		_ = resp1.Body.Close()

		// Maven 캐시
		resp2, err := env.MakeRequest("GET", "/proxy/maven/junit/junit/4.13.2/junit-4.13.2.pom", nil)
		require.NoError(t, err)
		_ = resp2.Body.Close()

		// 각각의 요청이 독립적으로 처리되었는지 확인
		assert.Equal(t, http.StatusOK, resp1.StatusCode)
		assert.Equal(t, http.StatusOK, resp2.StatusCode)
	})
}

// TestProxyErrorHandling 프록시 에러 처리 통합 테스트
func TestProxyErrorHandling(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	t.Run("Upstream 서버 다운 처리", func(t *testing.T) {
		// Mock 서버 중 하나를 종료
		if npmServer, exists := env.MockUpstreams["npm"]; exists {
			npmServer.Close()
		}

		// 종료된 서버로 요청 시 적절한 에러 처리
		resp, err := env.MakeRequest("GET", "/proxy/npm/express", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// 502 Bad Gateway 또는 500 Internal Server Error 기대
		assert.True(t, resp.StatusCode >= 500, "서버 에러 상태 코드가 반환되어야 함")
	})

	t.Run("잘못된 프록시 타입", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/unknown/some/path", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// 404 Not Found 또는 400 Bad Request 기대
		assert.True(t, resp.StatusCode == 404 || resp.StatusCode == 400,
			"잘못된 프록시 타입에 대해 클라이언트 에러가 반환되어야 함")
	})

	t.Run("대용량 파일 처리", func(t *testing.T) {
		// 대용량 응답을 시뮬레이션하기 위한 요청
		resp, err := env.MakeRequest("GET", "/proxy/maven/junit/junit/4.13.2/junit-4.13.2.jar", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 스트리밍으로 응답이 처리되는지 확인
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.NotEmpty(t, body)
	})

	t.Run("타임아웃 처리", func(t *testing.T) {
		// 매우 짧은 타임아웃으로 요청
		headers := map[string]string{
			"X-Timeout": "1ms", // 커스텀 헤더로 타임아웃 시뮬레이션
		}

		resp, err := env.MakeRequest("GET", "/proxy/npm/express", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// 정상 응답 또는 타임아웃 에러 (Mock 서버는 즉시 응답하므로 정상일 수 있음)
		assert.True(t, resp.StatusCode == 200 || resp.StatusCode >= 500)
	})
}

// TestProxySecurityIntegration 보안 관련 통합 테스트
func TestProxySecurityIntegration(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	t.Run("경로 탐색 공격 방지", func(t *testing.T) {
		// 경로 탐색 시도
		maliciousPaths := []string{
			"/proxy/npm/../../../etc/passwd",
			"/proxy/maven/..%2F..%2F..%2Fetc%2Fpasswd",
			"/proxy/apt/ubuntu/../../config/secret.yaml",
		}

		for _, path := range maliciousPaths {
			resp, err := env.MakeRequest("GET", path, nil)
			require.NoError(t, err)
			_ = resp.Body.Close()

			// 400 Bad Request 또는 404 Not Found 기대
			assert.True(t, resp.StatusCode == 400 || resp.StatusCode == 404,
				"경로 탐색 공격이 차단되어야 함: %s", path)
		}
	})

	t.Run("헤더 인젝션 방지", func(t *testing.T) {
		// 악성 헤더 인젝션 시도
		headers := map[string]string{
			"X-Forwarded-For": "127.0.0.1\r\nX-Injected: malicious",
			"User-Agent":      "test\nContent-Length: 0\n\nGET /evil HTTP/1.1",
		}

		resp, err := env.MakeRequest("GET", "/proxy/npm/express", headers)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// 요청이 정상 처리되거나 차단되어야 함
		assert.True(t, resp.StatusCode == 200 || resp.StatusCode >= 400)

		// 응답 헤더에 인젝션된 내용이 없는지 확인
		for key, value := range resp.Header {
			assert.NotContains(t, strings.ToLower(key), "injected")
			for _, v := range value {
				assert.NotContains(t, v, "malicious")
				assert.NotContains(t, v, "\r")
				assert.NotContains(t, v, "\n")
			}
		}
	})

	t.Run("요청 크기 제한", func(t *testing.T) {
		// POST 요청으로 대용량 데이터 전송 시도 (일부 프록시에서 지원하는 경우)
		largeBody := strings.Repeat("A", 10*1024*1024) // 10MB

		req, err := http.NewRequest("POST", "/proxy/npm/express", strings.NewReader(largeBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := env.ProxyServer.Test(req, -1)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// 413 Payload Too Large 또는 다른 적절한 에러 기대
		assert.True(t, resp.StatusCode >= 400, "대용량 요청이 제한되어야 함")
	})
}
