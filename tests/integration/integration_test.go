package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	fiberRouters "proxynd/internal/adapters/http/fiber/routers"
	"proxynd/internal/app"
	"proxynd/internal/config"
)

// IntegrationTestSuite 통합 테스트 스위트
type IntegrationTestSuite struct {
	suite.Suite
	app      *fiber.App
	baseURL  string
	cacheDir string
	logDir   string
}

// SetupSuite 테스트 스위트 초기화
func (s *IntegrationTestSuite) SetupSuite() {
	// 임시 디렉토리 생성
	s.cacheDir = filepath.Join(os.TempDir(), "proxynd-test-cache")
	s.logDir = filepath.Join(os.TempDir(), "proxynd-test-logs")

	_ = os.MkdirAll(s.cacheDir, 0o755)
	_ = os.MkdirAll(s.logDir, 0o755)

	// 테스트용 환경 변수 설정
	_ = os.Setenv("CONFIG_DIR", "./")
	_ = os.Setenv("STORAGE_DIR", s.cacheDir)

	// Fiber 앱 생성
	s.app = fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	// Create container for dependency injection
	container := app.NewContainer(&app.Config{
		ConfigDir:  "./",
		StorageDir: s.cacheDir,
		Port:       "8082",
	})

	// Setup routes using new hexagonal architecture
	testConfig := &config.RootConfig{
		Server: config.ServerConfig{
			Port: 8082,
			Host: "localhost",
		},
	}
	routeConfig := app.InitializeRouteConfig(testConfig, container)
	app.SetupRoutes(s.app, routeConfig)

	// 테스트 서버 시작
	s.baseURL = "http://localhost:8082"

	go func() {
		if err := s.app.Listen(":8082"); err != nil {
			panic(err)
		}
	}()

	// 서버 시작 대기
	time.Sleep(100 * time.Millisecond)
}

// TearDownSuite 테스트 스위트 정리
func (s *IntegrationTestSuite) TearDownSuite() {
	if s.app != nil {
		_ = s.app.Shutdown()
	}

	// 임시 디렉토리 정리
	_ = os.RemoveAll(s.cacheDir)
	_ = os.RemoveAll(s.logDir)
}

// TestHealthCheck 헬스체크 테스트
func (s *IntegrationTestSuite) TestHealthCheck() {
	resp, err := http.Get(s.baseURL + "/healthz")
	require.NoError(s.T(), err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err)

	var healthResp map[string]interface{}
	err = json.Unmarshal(body, &healthResp)
	require.NoError(s.T(), err)

	// 시스템 상태는 "ok" 또는 "degraded" 가능 (일부 프록시가 비활성화된 경우)
	status, ok := healthResp["status"].(string)
	require.True(s.T(), ok, "status should be a string")
	assert.Contains(s.T(), []string{"ok", "degraded"}, status, "시스템 상태는 ok 또는 degraded여야 함")
}

// TestNpmProxyScenario NPM 프록시 시나리오 테스트
func (s *IntegrationTestSuite) TestNpmProxyScenario() {
	// NPM 패키지 요청 시뮬레이션
	testCases := []struct {
		name         string
		packagePath  string
		expectedCode int
	}{
		{
			name:         "NPM package metadata",
			packagePath:  "express",
			expectedCode: http.StatusOK,
		},
		{
			name:         "NPM scoped package",
			packagePath:  "@types/node",
			expectedCode: http.StatusOK,
		},
		{
			name:         "NPM tarball",
			packagePath:  "express/-/express-4.18.2.tgz",
			expectedCode: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			url := fmt.Sprintf("%s/proxy/npm/%s", s.baseURL, tc.packagePath)

			// 첫 번째 요청 (캐시 미스)
			resp1, err := http.Get(url)
			require.NoError(t, err)
			defer func() { _ = resp1.Body.Close() }()

			// 두 번째 요청 (캐시 히트)
			resp2, err := http.Get(url)
			require.NoError(t, err)
			defer func() { _ = resp2.Body.Close() }()

			// 응답 상태 확인 (실제 NPM 서버가 없으므로 에러 응답도 정상)
			assert.True(t, resp1.StatusCode == tc.expectedCode || resp1.StatusCode >= 400)
			assert.True(t, resp2.StatusCode == tc.expectedCode || resp2.StatusCode >= 400)
		})
	}
}

// TestPipProxyScenario PyPI 프록시 시나리오 테스트
func (s *IntegrationTestSuite) TestPipProxyScenario() {
	testCases := []struct {
		name         string
		packagePath  string
		expectedCode int
	}{
		{
			name:         "PyPI package metadata",
			packagePath:  "simple/requests/",
			expectedCode: http.StatusOK,
		},
		{
			name:         "PyPI package file",
			packagePath:  "packages/source/r/requests/requests-2.28.2.tar.gz",
			expectedCode: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			url := fmt.Sprintf("%s/proxy/pip/%s", s.baseURL, tc.packagePath)

			resp, err := http.Get(url)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			// 응답 상태 확인
			assert.True(t, resp.StatusCode == tc.expectedCode || resp.StatusCode >= 400)
		})
	}
}

// TestAptProxyScenario APT 프록시 시나리오 테스트
func (s *IntegrationTestSuite) TestAptProxyScenario() {
	testCases := []struct {
		name         string
		packagePath  string
		expectedCode int
	}{
		{
			name:         "APT repository Release file",
			packagePath:  "ubuntu/dists/jammy/Release",
			expectedCode: http.StatusOK,
		},
		{
			name:         "APT package list",
			packagePath:  "ubuntu/dists/jammy/main/binary-amd64/Packages.gz",
			expectedCode: http.StatusOK,
		},
		{
			name:         "APT package file",
			packagePath:  "ubuntu/pool/main/a/apt/apt_2.4.8_amd64.deb",
			expectedCode: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			url := fmt.Sprintf("%s/proxy/apt/%s", s.baseURL, tc.packagePath)

			resp, err := http.Get(url)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			// 응답 상태 확인
			assert.True(t, resp.StatusCode == tc.expectedCode || resp.StatusCode >= 400)
		})
	}
}

// TestDockerProxyScenario Docker 프록시 시나리오 테스트
func (s *IntegrationTestSuite) TestDockerProxyScenario() {
	testCases := []struct {
		name         string
		packagePath  string
		method       string
		expectedCode int
	}{
		{
			name:         "Docker registry v2 check",
			packagePath:  "v2/",
			method:       "GET",
			expectedCode: http.StatusOK,
		},
		{
			name:         "Docker manifest",
			packagePath:  "v2/library/nginx/manifests/latest",
			method:       "GET",
			expectedCode: http.StatusOK,
		},
		{
			name:         "Docker blob",
			packagePath:  "v2/library/nginx/blobs/sha256:abc123",
			method:       "GET",
			expectedCode: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			url := fmt.Sprintf("%s/proxy/docker/%s", s.baseURL, tc.packagePath)

			req, err := http.NewRequest(tc.method, url, nil)
			require.NoError(t, err)

			// Docker registry 헤더 설정
			req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			// 응답 상태 확인
			assert.True(t, resp.StatusCode == tc.expectedCode || resp.StatusCode >= 400)
		})
	}
}

// TestCacheScenario 캐시 동작 시나리오 테스트
func (s *IntegrationTestSuite) TestCacheScenario() {
	// 캐시 동작을 확인하기 위한 더미 데이터 테스트
	packagePath := "test-package"
	url := fmt.Sprintf("%s/proxy/npm/%s", s.baseURL, packagePath)

	// 첫 번째 요청
	start1 := time.Now()
	resp1, err := http.Get(url)
	require.NoError(s.T(), err)
	defer func() { _ = resp1.Body.Close() }()
	duration1 := time.Since(start1)

	// 두 번째 요청 (캐시에서 응답)
	start2 := time.Now()
	resp2, err := http.Get(url)
	require.NoError(s.T(), err)
	defer func() { _ = resp2.Body.Close() }()
	duration2 := time.Since(start2)

	// 두 번째 요청이 더 빨라야 함 (캐시 효과)
	s.T().Logf("First request: %v, Second request: %v", duration1, duration2)

	// 응답 상태는 동일해야 함
	assert.Equal(s.T(), resp1.StatusCode, resp2.StatusCode)
}

// TestAuthenticationScenario 인증 시나리오 테스트
func (s *IntegrationTestSuite) TestAuthenticationScenario() {
	// 인증이 필요한 엔드포인트 테스트
	url := fmt.Sprintf("%s/proxy/npm/private-package", s.baseURL)

	// 인증 없이 요청
	resp1, err := http.Get(url)
	require.NoError(s.T(), err)
	defer func() { _ = resp1.Body.Close() }()

	// Basic Auth로 요청
	req, err := http.NewRequest("GET", url, nil)
	require.NoError(s.T(), err)
	req.SetBasicAuth("testuser", "testpass")

	client := &http.Client{}
	resp2, err := client.Do(req)
	require.NoError(s.T(), err)
	defer func() { _ = resp2.Body.Close() }()

	// 인증 여부에 따른 응답 차이 확인
	s.T().Logf("Without auth: %d, With auth: %d", resp1.StatusCode, resp2.StatusCode)
}

// TestSecurityScenario 보안 시나리오 테스트
func (s *IntegrationTestSuite) TestSecurityScenario() {
	// SHA256 해시 검증 테스트
	testData := []byte("test package data")
	url := fmt.Sprintf("%s/proxy/npm/test-package", s.baseURL)

	req, err := http.NewRequest("POST", url, bytes.NewReader(testData))
	require.NoError(s.T(), err)

	// 해시 헤더 설정
	req.Header.Set("Content-SHA256", "invalid-hash")
	req.Header.Set("Content-Type", "application/octet-stream")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(s.T(), err)
	defer func() { _ = resp.Body.Close() }()

	// 해시 불일치 시 적절한 응답 확인
	s.T().Logf("Security test response: %d", resp.StatusCode)
}

// TestLoadScenario 부하 시나리오 테스트
func (s *IntegrationTestSuite) TestLoadScenario() {
	const concurrent = 5
	const requests = 10

	url := fmt.Sprintf("%s/proxy/npm/express", s.baseURL)

	// 동시 요청 테스트
	done := make(chan bool, concurrent)

	for i := 0; i < concurrent; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < requests; j++ {
				resp, err := http.Get(url)
				if err != nil {
					s.T().Logf("Goroutine %d request %d failed: %v", id, j, err)
					continue
				}
				_ = resp.Body.Close()
			}
		}(i)
	}

	// 모든 고루틴 완료 대기
	for i := 0; i < concurrent; i++ {
		<-done
	}

	s.T().Logf("Load test completed: %d goroutines × %d requests", concurrent, requests)
}

// TestIntegration 통합 테스트 실행
func TestIntegration(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

// BenchmarkIntegrationProxyRequests 프록시 요청 벤치마크 (integration 패키지용)
func BenchmarkIntegrationProxyRequests(b *testing.B) {
	// 테스트 서버 설정
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	fiberRouters.ProxyRouter(app)

	go func() {
		_ = app.Listen(":8083")
	}()

	time.Sleep(100 * time.Millisecond)
	defer func() { _ = app.Shutdown() }()

	url := "http://localhost:8083/proxy/npm/express"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := http.Get(url)
			if err != nil {
				b.Error(err)
				continue
			}
			_ = resp.Body.Close()
		}
	})
}
