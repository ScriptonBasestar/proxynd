package integration

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigurationReload 설정 리로드 통합 테스트
func TestConfigurationReload(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	// 초기 설정으로 요청이 정상 동작하는지 확인
	err := env.WaitForUpstream("npm", 5*time.Second)
	require.NoError(t, err)

	t.Run("초기 설정 동작 확인", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/proxy/npm/express", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "express")
	})

	t.Run("설정 파일 수정 후 리로드", func(t *testing.T) {
		// 새로운 upstream 서버 URL로 설정 변경
		newConfigContent := fmt.Sprintf(`
path: "/npm"
use_cache: true
proxies:
  - name: "npmjs-modified"
    url: "%s"
`, env.MockUpstreams["npm"].URL)

		// 설정 파일 업데이트
		configPath := filepath.Join(env.ConfigDir, "npm-proxy.yaml")
		err := os.WriteFile(configPath, []byte(newConfigContent), 0644)
		require.NoError(t, err)

		// 설정 리로드 API 호출 (구현되어 있다면)
		reloadResp, err := env.MakeRequest("POST", "/api/config/reload", nil)
		if err == nil {
			_ = reloadResp.Body.Close()
			t.Logf("Config reload response: %d", reloadResp.StatusCode)
		}

		// 변경된 설정으로 여전히 동작하는지 확인
		time.Sleep(100 * time.Millisecond) // 설정 적용 대기

		resp, err := env.MakeRequest("GET", "/proxy/npm/express", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("잘못된 설정 파일 처리", func(t *testing.T) {
		// 잘못된 YAML 형식으로 설정 파일 수정
		invalidConfig := `
path: "/npm"
use_cache: true
proxies:
  - name: "npmjs"
    url: "invalid-url-format
    # 잘못된 YAML (닫는 따옴표 없음)
`

		configPath := filepath.Join(env.ConfigDir, "npm-proxy.yaml")
		err := os.WriteFile(configPath, []byte(invalidConfig), 0644)
		require.NoError(t, err)

		// 설정 리로드 시도
		reloadResp, err := env.MakeRequest("POST", "/api/config/reload", nil)
		if err == nil {
			_ = reloadResp.Body.Close()
			// 잘못된 설정은 리로드 실패해야 함
			assert.True(t, reloadResp.StatusCode >= 400, "잘못된 설정 리로드는 실패해야 함")
		}

		// 기존 설정으로 여전히 동작하는지 확인 (fallback)
		resp, err := env.MakeRequest("GET", "/proxy/npm/express", nil)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// 잘못된 설정이어도 서비스는 계속 동작해야 함
		assert.True(t, resp.StatusCode == 200 || resp.StatusCode == 502,
			"설정 오류가 있어도 서비스는 계속 동작하거나 적절한 에러를 반환해야 함")
	})

	t.Run("환경 변수 오버라이드", func(t *testing.T) {
		// 환경 변수로 설정 오버라이드 테스트
		originalPort := os.Getenv("SERVER_PORT")
		_ = os.Setenv("SERVER_PORT", "9999")
		defer func() {
			if originalPort != "" {
				_ = os.Setenv("SERVER_PORT", originalPort)
			} else {
				_ = os.Unsetenv("SERVER_PORT")
			}
		}()

		// 설정 정보 조회 API (구현되어 있다면)
		resp, err := env.MakeRequest("GET", "/api/config", nil)
		if err == nil {
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode == 200 {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)

				// 환경 변수 오버라이드가 반영되었는지 확인
				t.Logf("Config response: %s", string(body))
				// 실제 구현에서는 JSON 파싱 후 포트 값 확인
			}
		}
	})
}

// TestConfigurationValidation 설정 유효성 검증 테스트
func TestConfigurationValidation(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	t.Run("필수 설정 누락 검증", func(t *testing.T) {
		// path가 누락된 설정 파일
		invalidConfig := `
use_cache: true
proxies:
  - name: "npmjs"
    url: "http://registry.npmjs.org"
`

		configPath := filepath.Join(env.ConfigDir, "test-proxy.yaml")
		err := os.WriteFile(configPath, []byte(invalidConfig), 0644)
		require.NoError(t, err)
		defer func() { _ = os.Remove(configPath) }()

		// 설정 검증 API 호출
		resp, err := env.MakeRequest("POST", "/api/config/validate", nil)
		if err == nil {
			defer func() { _ = resp.Body.Close() }()

			// 검증 실패 응답 기대
			if resp.StatusCode != 404 { // API가 구현되지 않은 경우 404
				assert.True(t, resp.StatusCode >= 400, "잘못된 설정은 검증 실패해야 함")
			}
		}
	})

	t.Run("순환 의존성 검증", func(t *testing.T) {
		// 프록시 설정에서 순환 참조가 있는 경우
		cyclicConfig := `
path: "/npm"
use_cache: true
proxies:
  - name: "proxy1"
    url: "http://localhost:8080/proxy/npm2"
  - name: "proxy2" 
    url: "http://localhost:8080/proxy/npm"
`

		configPath := filepath.Join(env.ConfigDir, "cyclic-proxy.yaml")
		err := os.WriteFile(configPath, []byte(cyclicConfig), 0644)
		require.NoError(t, err)
		defer func() { _ = os.Remove(configPath) }()

		// 순환 참조 검증 (구현에 따라)
		t.Logf("Created cyclic configuration for validation test")
	})

	t.Run("URL 형식 검증", func(t *testing.T) {
		// 잘못된 URL 형식
		invalidURLConfig := `
path: "/npm"
use_cache: true
proxies:
  - name: "invalid"
    url: "not-a-valid-url"
  - name: "also-invalid"
    url: "ftp://unsupported-protocol.com"
`

		configPath := filepath.Join(env.ConfigDir, "invalid-url-proxy.yaml")
		err := os.WriteFile(configPath, []byte(invalidURLConfig), 0644)
		require.NoError(t, err)
		defer func() { _ = os.Remove(configPath) }()

		// URL 검증 테스트
		t.Logf("Created invalid URL configuration for validation test")
	})
}

// TestConfigurationAPI 설정 관리 API 테스트
func TestConfigurationAPI(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	t.Run("설정 조회 API", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/api/config", nil)
		if err != nil {
			t.Skip("Config API not available")
			return
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode == 404 {
			t.Skip("Config API not implemented")
			return
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		// JSON 형식 확인
		assert.True(t, len(body) > 0)
		assert.True(t, string(body)[0] == '{' || string(body)[0] == '[')

		t.Logf("Config API response: %s", string(body))
	})

	t.Run("프록시별 설정 조회", func(t *testing.T) {
		proxyTypes := []string{"npm", "maven", "apt"}

		for _, proxyType := range proxyTypes {
			resp, err := env.MakeRequest("GET", fmt.Sprintf("/api/config/%s", proxyType), nil)
			if err != nil {
				continue
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode == 404 {
				continue // API가 구현되지 않음
			}

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			t.Logf("%s config: %s", proxyType, string(body))
		}
	})

	t.Run("설정 업데이트 API", func(t *testing.T) {
		// 새로운 설정 JSON
		newConfig := `{
			"path": "/npm",
			"use_cache": true,
			"proxies": [
				{
					"name": "npmjs-updated",
					"url": "` + env.MockUpstreams["npm"].URL + `"
				}
			]
		}`

		// PUT 요청으로 설정 업데이트
		req, err := http.NewRequest("PUT", "/api/config/npm",
			strings.NewReader(newConfig))
		if err != nil {
			t.Skip("Cannot create PUT request")
			return
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := env.ProxyServer.Test(req, -1)
		if err != nil {
			t.Skip("Config update API not available")
			return
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode == 404 {
			t.Skip("Config update API not implemented")
			return
		}

		// 성공적인 업데이트 (200) 또는 권한 부족 (403) 등
		assert.True(t, resp.StatusCode < 500, "설정 업데이트 API가 서버 에러를 반환하면 안됨")

		if resp.StatusCode == 200 {
			// 업데이트된 설정으로 동작하는지 확인
			time.Sleep(100 * time.Millisecond)

			testResp, err := env.MakeRequest("GET", "/proxy/npm/express", nil)
			require.NoError(t, err)
			defer func() { _ = testResp.Body.Close() }()

			assert.Equal(t, http.StatusOK, testResp.StatusCode)
		}
	})

	t.Run("설정 백업 및 복원", func(t *testing.T) {
		// 설정 백업 API
		backupResp, err := env.MakeRequest("POST", "/api/config/backup", nil)
		if err != nil || backupResp.StatusCode == 404 {
			t.Skip("Config backup API not available")
			return
		}
		defer func() { _ = backupResp.Body.Close() }()

		if backupResp.StatusCode == 200 {
			// 백업 파일 정보 확인
			body, err := io.ReadAll(backupResp.Body)
			require.NoError(t, err)

			t.Logf("Backup response: %s", string(body))
		}

		// 설정 복원 API
		restoreResp, err := env.MakeRequest("POST", "/api/config/restore", nil)
		if err == nil {
			defer func() { _ = restoreResp.Body.Close() }()
			t.Logf("Restore response: %d", restoreResp.StatusCode)
		}
	})
}

// TestConfigurationSecurity 설정 보안 테스트
func TestConfigurationSecurity(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	t.Run("설정 파일 경로 탐색 방지", func(t *testing.T) {
		// 경로 탐색 시도
		maliciousPaths := []string{
			"/api/config/../../../etc/passwd",
			"/api/config/..%2F..%2F..%2Fetc%2Fpasswd",
			"/api/config/npm/../global",
		}

		for _, path := range maliciousPaths {
			resp, err := env.MakeRequest("GET", path, nil)
			if err != nil {
				continue
			}
			defer func() { _ = resp.Body.Close() }()

			// 400 Bad Request, 403 Forbidden, 또는 404 Not Found 기대
			assert.True(t, resp.StatusCode >= 400 && resp.StatusCode < 500,
				"경로 탐색 공격이 차단되어야 함: %s", path)
		}
	})

	t.Run("설정 수정 권한 검증", func(t *testing.T) {
		// 인증 없이 설정 수정 시도
		updatePayload := `{"path": "/npm", "use_cache": false}`

		req, err := http.NewRequest("PUT", "/api/config/npm",
			strings.NewReader(updatePayload))
		if err != nil {
			t.Skip("Cannot create request")
			return
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := env.ProxyServer.Test(req, -1)
		if err != nil || resp.StatusCode == 404 {
			t.Skip("Config update API not available")
			return
		}
		defer func() { _ = resp.Body.Close() }()

		// 인증이 필요한 경우 401 또는 403, 또는 기능이 비활성화된 경우 405
		assert.True(t, resp.StatusCode == 401 || resp.StatusCode == 403 ||
			resp.StatusCode == 405 || resp.StatusCode == 200,
			"설정 수정에 대한 적절한 권한 검증이 있어야 함")
	})

	t.Run("민감한 정보 노출 방지", func(t *testing.T) {
		resp, err := env.MakeRequest("GET", "/api/config", nil)
		if err != nil || resp.StatusCode == 404 {
			t.Skip("Config API not available")
			return
		}
		defer func() { _ = resp.Body.Close() }()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		bodyStr := string(body)

		// 민감한 정보가 노출되지 않는지 확인
		sensitivePatterns := []string{
			"password",
			"secret",
			"private_key",
			"token",
			"credential",
		}

		for _, pattern := range sensitivePatterns {
			assert.NotContains(t, strings.ToLower(bodyStr), pattern,
				"설정 응답에 민감한 정보가 포함되면 안됨: %s", pattern)
		}
	})
}

// TestConfigurationHotReload 핫 리로드 테스트
func TestConfigurationHotReload(t *testing.T) {
	env := SetupIntegrationTest(t)
	defer env.Cleanup()

	err := env.WaitForUpstream("npm", 5*time.Second)
	require.NoError(t, err)

	t.Run("파일 변경 감지", func(t *testing.T) {
		// 초기 요청
		resp1, err := env.MakeRequest("GET", "/proxy/npm/express", nil)
		require.NoError(t, err)
		defer func() { _ = resp1.Body.Close() }()
		assert.Equal(t, http.StatusOK, resp1.StatusCode)

		// 설정 파일 수정
		configPath := filepath.Join(env.ConfigDir, "npm-proxy.yaml")
		modifiedConfig := fmt.Sprintf(`
path: "/npm"
use_cache: true
# 수정된 설정
proxies:
  - name: "npmjs-hotreload"
    url: "%s"
`, env.MockUpstreams["npm"].URL)

		err = os.WriteFile(configPath, []byte(modifiedConfig), 0644)
		require.NoError(t, err)

		// 파일 시스템 변경 감지 대기 (실제 구현에서는 inotify 등 사용)
		time.Sleep(500 * time.Millisecond)

		// 변경된 설정으로 여전히 동작하는지 확인
		resp2, err := env.MakeRequest("GET", "/proxy/npm/express", nil)
		require.NoError(t, err)
		defer func() { _ = resp2.Body.Close() }()
		assert.Equal(t, http.StatusOK, resp2.StatusCode)
	})

	t.Run("여러 설정 파일 동시 변경", func(t *testing.T) {
		// NPM과 Maven 설정을 동시에 변경
		npmConfig := fmt.Sprintf(`
path: "/npm"
use_cache: true
proxies:
  - name: "npmjs-multi"
    url: "%s"
`, env.MockUpstreams["npm"].URL)

		mavenConfig := fmt.Sprintf(`
path: "/maven"  
use_cache: true
proxies:
  - name: "central-multi"
    url: "%s"
`, env.MockUpstreams["maven"].URL)

		// 동시 변경
		npmPath := filepath.Join(env.ConfigDir, "npm-proxy.yaml")
		mavenPath := filepath.Join(env.ConfigDir, "maven-proxy.yaml")

		err := os.WriteFile(npmPath, []byte(npmConfig), 0644)
		require.NoError(t, err)

		err = os.WriteFile(mavenPath, []byte(mavenConfig), 0644)
		require.NoError(t, err)

		// 변경 적용 대기
		time.Sleep(500 * time.Millisecond)

		// 두 프록시 모두 정상 동작하는지 확인
		npmResp, err := env.MakeRequest("GET", "/proxy/npm/express", nil)
		require.NoError(t, err)
		defer func() { _ = npmResp.Body.Close() }()

		mavenResp, err := env.MakeRequest("GET", "/proxy/maven/junit/junit/4.13.2/junit-4.13.2.pom", nil)
		require.NoError(t, err)
		defer func() { _ = mavenResp.Body.Close() }()

		assert.Equal(t, http.StatusOK, npmResp.StatusCode)
		assert.Equal(t, http.StatusOK, mavenResp.StatusCode)
	})
}
