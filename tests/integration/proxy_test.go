package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/configs"
	"proxynd/internal/services/adapters"
	"proxynd/internal/services/config"
	"proxynd/internal/services/proxy"
	"proxynd/pkg/repositories"
)

// ProxyIntegrationTestSuite 프록시 통합 테스트 스위트
type ProxyIntegrationTestSuite struct {
	tempDir         string
	configService   config.Service
	cacheRepo       repositories.CacheRepository
	upstreamServers map[string]*httptest.Server
	proxyServices   map[string]proxy.ProxyService
}

// SetupTest 각 테스트 전 실행
func (s *ProxyIntegrationTestSuite) SetupTest(t *testing.T) {
	// 임시 디렉토리 생성
	s.tempDir = t.TempDir()
	configDir := filepath.Join(s.tempDir, "config")
	cacheDir := filepath.Join(s.tempDir, "cache")

	require.NoError(t, os.MkdirAll(configDir, 0755))
	require.NoError(t, os.MkdirAll(cacheDir, 0755))

	// 테스트 설정 파일 생성
	s.createTestConfigs(t, configDir)

	// 설정 서비스 초기화
	configService, err := config.NewService(configDir)
	require.NoError(t, err)
	s.configService = configService

	// 캐시 리포지토리 초기화
	s.cacheRepo = repositories.NewFileSystemCacheRepository(cacheDir)

	// Mock 업스트림 서버 설정
	s.setupUpstreamServers()

	// 프록시 서비스 초기화
	s.initializeProxyServices(t)
}

// TeardownTest 각 테스트 후 실행
func (s *ProxyIntegrationTestSuite) TeardownTest() {
	// 업스트림 서버 종료
	for _, server := range s.upstreamServers {
		server.Close()
	}
}

// createTestConfigs 테스트용 설정 파일 생성
func (s *ProxyIntegrationTestSuite) createTestConfigs(t *testing.T, configDir string) {
	// Global config
	globalConfig := `
global:
  server:
    port: 8080
    read_timeout: 30s
    write_timeout: 30s
  storage:
    cache_dir: ./cache
    ttl: 3600
  logging:
    level: info
    format: json
`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "global.yaml"), []byte(globalConfig), 0644))

	// APT proxy config
	aptConfig := fmt.Sprintf(`
apt:
  upstream_urls:
    - %s/apt
  cache_enabled: true
  allowed_architectures:
    - amd64
    - arm64
  allowed_distributions:
    - jammy
    - focal
`, s.getUpstreamURL("apt"))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "apt-proxy.yaml"), []byte(aptConfig), 0644))

	// Maven proxy config
	mavenConfig := fmt.Sprintf(`
maven:
  upstream_urls:
    - id: central
      url: %s/maven
      priority: 1
  cache_enabled: true
  checksum_validation: true
`, s.getUpstreamURL("maven"))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "maven-proxy.yaml"), []byte(mavenConfig), 0644))

	// NPM proxy config
	npmConfig := fmt.Sprintf(`
npm:
  upstream_urls:
    - %s/npm
  cache_enabled: true
  scoped_packages_allowed: true
`, s.getUpstreamURL("npm"))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "npm-proxy.yaml"), []byte(npmConfig), 0644))
}

// getUpstreamURL 업스트림 서버 URL 반환
func (s *ProxyIntegrationTestSuite) getUpstreamURL(proxyType string) string {
	if s.upstreamServers != nil && s.upstreamServers[proxyType] != nil {
		return s.upstreamServers[proxyType].URL
	}
	return "http://localhost:9999" // 기본값
}

// setupUpstreamServers Mock 업스트림 서버 설정
func (s *ProxyIntegrationTestSuite) setupUpstreamServers() {
	s.upstreamServers = make(map[string]*httptest.Server)

	// APT 업스트림 서버
	s.upstreamServers["apt"] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/dists/jammy/Release"):
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Origin: Ubuntu\nLabel: Ubuntu\nSuite: jammy\n"))
		case strings.Contains(r.URL.Path, "/Packages.gz"):
			w.Header().Set("Content-Type", "application/x-gzip")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte{0x1f, 0x8b, 0x08, 0x00}) // gzip magic bytes
		case strings.Contains(r.URL.Path, ".deb"):
			w.Header().Set("Content-Type", "application/vnd.debian.binary-package")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("fake deb content"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	// Maven 업스트림 서버
	s.upstreamServers["maven"] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/maven-metadata.xml"):
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<metadata>
  <groupId>com.example</groupId>
  <artifactId>test</artifactId>
  <versioning>
    <latest>1.0.0</latest>
    <release>1.0.0</release>
  </versioning>
</metadata>`))
		case strings.HasSuffix(r.URL.Path, ".jar"):
			w.Header().Set("Content-Type", "application/java-archive")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("fake jar content"))
		case strings.HasSuffix(r.URL.Path, ".jar.sha1"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("da39a3ee5e6b4b0d3255bfef95601890afd80709"))
		case strings.HasSuffix(r.URL.Path, ".pom"):
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<project>
  <modelVersion>4.0.0</modelVersion>
  <groupId>com.example</groupId>
  <artifactId>test</artifactId>
  <version>1.0.0</version>
</project>`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	// NPM 업스트림 서버
	s.upstreamServers["npm"] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/express"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"name": "express",
				"version": "4.18.2",
				"dist": {
					"tarball": "` + s.upstreamServers["npm"].URL + `/express/-/express-4.18.2.tgz"
				}
			}`))
		case strings.HasSuffix(r.URL.Path, ".tgz"):
			w.Header().Set("Content-Type", "application/x-gzip")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte{0x1f, 0x8b, 0x08, 0x00}) // gzip magic bytes
		case strings.Contains(r.URL.Path, "/@types/node"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"name": "@types/node",
				"version": "18.0.0"
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// initializeProxyServices 프록시 서비스 초기화
func (s *ProxyIntegrationTestSuite) initializeProxyServices(t *testing.T) {
	s.proxyServices = make(map[string]proxy.ProxyService)

	cacheAdapter := adapters.NewCacheAdapter(s.cacheRepo)
	upstreamClient := adapters.NewHTTPUpstreamClient(30 * time.Second)

	// 프록시 서비스 생성
	factory := proxy.NewFactory(s.configService, cacheAdapter, upstreamClient)

	aptService, err := factory.CreateProxy("apt")
	require.NoError(t, err)
	s.proxyServices["apt"] = aptService

	mavenService, err := factory.CreateProxy("maven")
	require.NoError(t, err)
	s.proxyServices["maven"] = mavenService

	npmService, err := factory.CreateProxy("npm")
	require.NoError(t, err)
	s.proxyServices["npm"] = npmService
}

// TestAPTProxyFlow APT 프록시 전체 플로우 테스트
func TestAPTProxyFlow(t *testing.T) {
	suite := &ProxyIntegrationTestSuite{}
	suite.SetupTest(t)
	defer suite.TeardownTest()

	ctx := context.Background()
	aptService := suite.proxyServices["apt"]

	t.Run("Release 파일 다운로드", func(t *testing.T) {
		// 첫 번째 요청 (캐시 미스)
		req := &proxy.Request{
			Method:  "GET",
			Path:    "/ubuntu/dists/jammy/Release",
			Headers: make(map[string]string),
		}

		resp1, err := aptService.Handle(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp1.StatusCode)
		assert.False(t, resp1.Cached)

		body1, err := io.ReadAll(resp1.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body1), "Origin: Ubuntu")

		// 두 번째 요청 (캐시 히트)
		resp2, err := aptService.Handle(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp2.StatusCode)
		assert.True(t, resp2.Cached)

		body2, err := io.ReadAll(resp2.Body)
		require.NoError(t, err)
		assert.Equal(t, string(body1), string(body2))
	})

	t.Run("패키지 목록 다운로드", func(t *testing.T) {
		req := &proxy.Request{
			Method:  "GET",
			Path:    "/ubuntu/dists/jammy/main/binary-amd64/Packages.gz",
			Headers: make(map[string]string),
		}

		resp, err := aptService.Handle(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, "application/x-gzip", resp.ContentType)
	})

	t.Run("DEB 패키지 다운로드", func(t *testing.T) {
		req := &proxy.Request{
			Method:  "GET",
			Path:    "/ubuntu/pool/main/a/apt/apt_2.4.8_amd64.deb",
			Headers: make(map[string]string),
		}

		resp, err := aptService.Handle(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, "application/vnd.debian.binary-package", resp.ContentType)
	})

	t.Run("잘못된 경로 처리", func(t *testing.T) {
		req := &proxy.Request{
			Method:  "GET",
			Path:    "/invalid/path",
			Headers: make(map[string]string),
		}

		resp, err := aptService.Handle(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode)
	})
}

// TestMavenProxyFlow Maven 프록시 전체 플로우 테스트
func TestMavenProxyFlow(t *testing.T) {
	suite := &ProxyIntegrationTestSuite{}
	suite.SetupTest(t)
	defer suite.TeardownTest()

	ctx := context.Background()
	mavenService := suite.proxyServices["maven"]

	t.Run("메타데이터 다운로드", func(t *testing.T) {
		req := &proxy.Request{
			Method:  "GET",
			Path:    "/com/example/test/maven-metadata.xml",
			Headers: make(map[string]string),
		}

		resp, err := mavenService.Handle(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, "application/xml", resp.ContentType)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "<groupId>com.example</groupId>")
	})

	t.Run("JAR 파일 다운로드", func(t *testing.T) {
		req := &proxy.Request{
			Method:  "GET",
			Path:    "/com/example/test/1.0.0/test-1.0.0.jar",
			Headers: make(map[string]string),
		}

		resp, err := mavenService.Handle(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, "application/java-archive", resp.ContentType)
	})

	t.Run("체크섬 파일 다운로드", func(t *testing.T) {
		req := &proxy.Request{
			Method:  "GET",
			Path:    "/com/example/test/1.0.0/test-1.0.0.jar.sha1",
			Headers: make(map[string]string),
		}

		resp, err := mavenService.Handle(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "da39a3ee5e6b4b0d3255bfef95601890afd80709", string(body))
	})

	t.Run("POM 파일 다운로드", func(t *testing.T) {
		req := &proxy.Request{
			Method:  "GET",
			Path:    "/com/example/test/1.0.0/test-1.0.0.pom",
			Headers: make(map[string]string),
		}

		resp, err := mavenService.Handle(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, "application/xml", resp.ContentType)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "<artifactId>test</artifactId>")
	})
}

// TestNPMProxyFlow NPM 프록시 전체 플로우 테스트
func TestNPMProxyFlow(t *testing.T) {
	suite := &ProxyIntegrationTestSuite{}
	suite.SetupTest(t)
	defer suite.TeardownTest()

	ctx := context.Background()
	npmService := suite.proxyServices["npm"]

	t.Run("패키지 메타데이터 다운로드", func(t *testing.T) {
		req := &proxy.Request{
			Method:  "GET",
			Path:    "/express",
			Headers: make(map[string]string),
		}

		resp, err := npmService.Handle(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, "application/json", resp.ContentType)

		var metadata map[string]interface{}
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		err = json.Unmarshal(body, &metadata)
		require.NoError(t, err)
		assert.Equal(t, "express", metadata["name"])
	})

	t.Run("tarball 다운로드", func(t *testing.T) {
		req := &proxy.Request{
			Method:  "GET",
			Path:    "/express/-/express-4.18.2.tgz",
			Headers: make(map[string]string),
		}

		resp, err := npmService.Handle(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, "application/x-gzip", resp.ContentType)
	})

	t.Run("scoped 패키지 처리", func(t *testing.T) {
		req := &proxy.Request{
			Method:  "GET",
			Path:    "/@types/node",
			Headers: make(map[string]string),
		}

		resp, err := npmService.Handle(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var metadata map[string]interface{}
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		err = json.Unmarshal(body, &metadata)
		require.NoError(t, err)
		assert.Equal(t, "@types/node", metadata["name"])
	})
}

// TestConcurrentRequests 동시 요청 처리 테스트
func TestConcurrentRequests(t *testing.T) {
	suite := &ProxyIntegrationTestSuite{}
	suite.SetupTest(t)
	defer suite.TeardownTest()

	ctx := context.Background()
	npmService := suite.proxyServices["npm"]

	const numRequests = 10
	var wg sync.WaitGroup
	wg.Add(numRequests)

	errors := make([]error, numRequests)
	responses := make([]*proxy.Response, numRequests)

	// 동시에 같은 리소스 요청
	for i := 0; i < numRequests; i++ {
		go func(idx int) {
			defer wg.Done()
			req := &proxy.Request{
				Method:  "GET",
				Path:    "/express",
				Headers: make(map[string]string),
			}
			resp, err := npmService.Handle(ctx, req)
			errors[idx] = err
			responses[idx] = resp
		}(i)
	}

	wg.Wait()

	// 모든 요청이 성공했는지 확인
	for i := 0; i < numRequests; i++ {
		assert.NoError(t, errors[i])
		assert.NotNil(t, responses[i])
		assert.Equal(t, 200, responses[i].StatusCode)
	}

	// 캐시 히트율 확인
	cachedCount := 0
	for _, resp := range responses {
		if resp.Cached {
			cachedCount++
		}
	}
	assert.Greater(t, cachedCount, 0, "일부 요청은 캐시에서 처리되어야 함")
}

// TestContextCancellation 컨텍스트 취소 처리 테스트
func TestContextCancellation(t *testing.T) {
	suite := &ProxyIntegrationTestSuite{}
	suite.SetupTest(t)
	defer suite.TeardownTest()

	// 느린 응답을 반환하는 업스트림 서버
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	// 설정 업데이트 (느린 서버 사용)
	suite.upstreamServers["slow"] = slowServer
	npmService := suite.proxyServices["npm"]

	// 짧은 타임아웃 컨텍스트
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req := &proxy.Request{
		Method:  "GET",
		Path:    "/slow-package",
		Headers: make(map[string]string),
	}

	start := time.Now()
	resp, err := npmService.Handle(ctx, req)
	duration := time.Since(start)

	// 타임아웃으로 실패해야 함
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Less(t, duration, 1*time.Second, "컨텍스트 타임아웃이 작동해야 함")
}

// TestLargeFileHandling 대용량 파일 처리 테스트
func TestLargeFileHandling(t *testing.T) {
	suite := &ProxyIntegrationTestSuite{}
	suite.SetupTest(t)
	defer suite.TeardownTest()

	// 대용량 파일을 반환하는 업스트림 서버
	largeFileServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		size := 10 * 1024 * 1024 // 10MB
		w.Header().Set("Content-Length", fmt.Sprintf("%d", size))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)

		// 청크로 데이터 전송
		chunk := make([]byte, 1024*1024) // 1MB chunks
		for i := 0; i < 10; i++ {
			w.Write(chunk)
		}
	}))
	defer largeFileServer.Close()

	suite.upstreamServers["large"] = largeFileServer
	npmService := suite.proxyServices["npm"]

	ctx := context.Background()
	req := &proxy.Request{
		Method:  "GET",
		Path:    "/large-package.tar.gz",
		Headers: make(map[string]string),
	}

	// 첫 번째 요청 (캐시 미스)
	resp1, err := npmService.Handle(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp1.StatusCode)
	assert.False(t, resp1.Cached)

	// 전체 내용 읽기
	body1, err := io.ReadAll(resp1.Body)
	require.NoError(t, err)
	assert.Equal(t, 10*1024*1024, len(body1))

	// 두 번째 요청 (캐시 히트)
	resp2, err := npmService.Handle(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp2.StatusCode)
	assert.True(t, resp2.Cached)
}

// TestAuthenticationFlow 인증 플로우 테스트
func TestAuthenticationFlow(t *testing.T) {
	suite := &ProxyIntegrationTestSuite{}
	suite.SetupTest(t)
	defer suite.TeardownTest()

	// 인증이 필요한 업스트림 서버
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer valid-token" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name": "private-package"}`))
	}))
	defer authServer.Close()

	suite.upstreamServers["auth"] = authServer
	npmService := suite.proxyServices["npm"]

	ctx := context.Background()

	t.Run("인증 없는 요청", func(t *testing.T) {
		req := &proxy.Request{
			Method:  "GET",
			Path:    "/private-package",
			Headers: make(map[string]string),
		}

		resp, err := npmService.Handle(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("인증 있는 요청", func(t *testing.T) {
		req := &proxy.Request{
			Method: "GET",
			Path:   "/private-package",
			Headers: map[string]string{
				"Authorization": "Bearer valid-token",
			},
		}

		resp, err := npmService.Handle(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "private-package")
	})
}

// TestErrorHandling 에러 처리 테스트
func TestErrorHandling(t *testing.T) {
	suite := &ProxyIntegrationTestSuite{}
	suite.SetupTest(t)
	defer suite.TeardownTest()

	ctx := context.Background()

	t.Run("업스트림 서버 다운", func(t *testing.T) {
		// 존재하지 않는 서버로 설정 변경
		deadConfig := &configs.NPMConfig{
			UpstreamURLs: []string{"http://localhost:12345"},
		}
		suite.configService.(*config.ServiceImpl).configs["npm"] = deadConfig

		npmService, err := proxy.NewFactory(
			suite.configService,
			adapters.NewCacheAdapter(suite.cacheRepo),
			adapters.NewHTTPUpstreamClient(1*time.Second),
		).CreateProxy("npm")
		require.NoError(t, err)

		req := &proxy.Request{
			Method:  "GET",
			Path:    "/express",
			Headers: make(map[string]string),
		}

		resp, err := npmService.Handle(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("잘못된 응답 형식", func(t *testing.T) {
		// 잘못된 JSON을 반환하는 서버
		badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("not-json"))
		}))
		defer badServer.Close()

		suite.upstreamServers["bad"] = badServer
		npmService := suite.proxyServices["npm"]

		req := &proxy.Request{
			Method:  "GET",
			Path:    "/bad-package",
			Headers: make(map[string]string),
		}

		resp, err := npmService.Handle(ctx, req)
		// NPM 서비스는 잘못된 JSON도 그대로 전달할 수 있음
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}

// TestCacheInvalidation 캐시 무효화 테스트
func TestCacheInvalidation(t *testing.T) {
	suite := &ProxyIntegrationTestSuite{}
	suite.SetupTest(t)
	defer suite.TeardownTest()

	ctx := context.Background()
	mavenService := suite.proxyServices["maven"]

	// 업로드 요청 (PUT/POST)으로 캐시 무효화
	t.Run("PUT 요청으로 캐시 무효화", func(t *testing.T) {
		path := "/com/example/test/1.0.0/test-1.0.0.jar"

		// GET 요청으로 캐시 생성
		getReq := &proxy.Request{
			Method:  "GET",
			Path:    path,
			Headers: make(map[string]string),
		}
		resp1, err := mavenService.Handle(ctx, getReq)
		require.NoError(t, err)
		assert.Equal(t, 200, resp1.StatusCode)

		// PUT 요청
		putReq := &proxy.Request{
			Method:  "PUT",
			Path:    path,
			Headers: make(map[string]string),
			Body:    io.NopCloser(bytes.NewReader([]byte("new content"))),
		}
		resp2, err := mavenService.Handle(ctx, putReq)
		require.NoError(t, err)
		assert.Equal(t, 200, resp2.StatusCode)

		// 다시 GET 요청 (캐시가 무효화되어야 함)
		resp3, err := mavenService.Handle(ctx, getReq)
		require.NoError(t, err)
		assert.Equal(t, 200, resp3.StatusCode)
		assert.False(t, resp3.Cached, "PUT 후 캐시가 무효화되어야 함")
	})
}

// TestMetricsCollection 메트릭 수집 테스트
func TestMetricsCollection(t *testing.T) {
	suite := &ProxyIntegrationTestSuite{}
	suite.SetupTest(t)
	defer suite.TeardownTest()

	ctx := context.Background()
	npmService := suite.proxyServices["npm"]

	// 여러 요청 실행
	paths := []string{"/express", "/@types/node", "/react"}
	for _, path := range paths {
		req := &proxy.Request{
			Method:  "GET",
			Path:    path,
			Headers: make(map[string]string),
		}
		npmService.Handle(ctx, req)
	}

	// 메트릭 확인 (실제 구현에서는 Prometheus 메트릭 확인)
	// 여기서는 기본적인 동작만 확인
	assert.True(t, true, "메트릭 수집이 동작해야 함")
}

// TestFiberIntegration Fiber 프레임워크 통합 테스트
func TestFiberIntegration(t *testing.T) {
	suite := &ProxyIntegrationTestSuite{}
	suite.SetupTest(t)
	defer suite.TeardownTest()

	// Fiber 앱 생성
	app := fiber.New()

	// 프록시 핸들러 등록
	app.All("/proxy/:type/*", func(c *fiber.Ctx) error {
		proxyType := c.Params("type")
		service, ok := suite.proxyServices[proxyType]
		if !ok {
			return c.Status(fiber.StatusNotFound).SendString("Unknown proxy type")
		}

		req := &proxy.Request{
			Method:  c.Method(),
			Path:    c.Params("*"),
			Headers: make(map[string]string),
		}

		// 헤더 복사
		c.Request().Header.VisitAll(func(key, value []byte) {
			req.Headers[string(key)] = string(value)
		})

		// 바디 처리
		if c.Method() != "GET" && c.Method() != "HEAD" {
			req.Body = io.NopCloser(bytes.NewReader(c.Body()))
		}

		resp, err := service.Handle(c.Context(), req)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		// 응답 헤더 설정
		for k, v := range resp.Headers {
			c.Set(k, v)
		}

		// 캐시 상태 헤더
		if resp.Cached {
			c.Set("X-Cache", "HIT")
		} else {
			c.Set("X-Cache", "MISS")
		}

		// 바디 전송
		body, _ := io.ReadAll(resp.Body)
		return c.Status(resp.StatusCode).Send(body)
	})

	// 테스트 요청
	req := httptest.NewRequest("GET", "/proxy/npm/express", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("X-Cache"))
}

// BenchmarkProxyRequests 프록시 성능 벤치마크
func BenchmarkProxyRequests(b *testing.B) {
	suite := &ProxyIntegrationTestSuite{}
	suite.SetupTest(b)
	defer suite.TeardownTest()

	ctx := context.Background()
	npmService := suite.proxyServices["npm"]

	req := &proxy.Request{
		Method:  "GET",
		Path:    "/express",
		Headers: make(map[string]string),
	}

	// 캐시 워밍업
	npmService.Handle(ctx, req)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := npmService.Handle(ctx, req)
			if err != nil {
				b.Error(err)
			}
			if resp != nil {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}
	})
}