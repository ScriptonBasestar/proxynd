package integration

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"

	"proxynd/internal/app"
	"proxynd/internal/config"
	"proxynd/internal/metrics"
	"proxynd/internal/routers"
)

// IntegrationTestEnvironment 통합 테스트 환경
type IntegrationTestEnvironment struct {
	// ProxyND 서버
	ProxyServer *fiber.App
	ServerURL   string

	// Mock upstream 서버들
	MockUpstreams map[string]*httptest.Server

	// 테스트 설정
	Config    *config.UnifiedConfig
	ConfigDir string
	CacheDir  string

	// 정리 함수들
	cleanup []func()
}

// MockUpstreamResponse Mock upstream 응답 정의
type MockUpstreamResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
	Delay      time.Duration
}

// SetupIntegrationTest 통합 테스트 환경 설정
func SetupIntegrationTest(t *testing.T) *IntegrationTestEnvironment {
	// Set development mode for tests to bypass authentication
	os.Setenv("PROXYND_ENV", "development")

	// 임시 디렉토리 생성
	configDir, err := os.MkdirTemp("", "proxynd-integration-config-*")
	require.NoError(t, err)

	cacheDir, err := os.MkdirTemp("", "proxynd-integration-cache-*")
	require.NoError(t, err)

	env := &IntegrationTestEnvironment{
		MockUpstreams: make(map[string]*httptest.Server),
		ConfigDir:     configDir,
		CacheDir:      cacheDir,
		cleanup: []func(){
			func() { _ = os.RemoveAll(configDir) },
			func() { _ = os.RemoveAll(cacheDir) },
		},
	}

	// Mock upstream 서버들 설정
	env.setupMockUpstreams(t)

	// ProxyND 설정 생성
	env.setupConfiguration(t)

	// ProxyND 서버 시작
	env.setupProxyServer(t)

	return env
}

// setupMockUpstreams Mock upstream 서버들 설정
//
//nolint:gocyclo // Test setup complexity is acceptable
func (env *IntegrationTestEnvironment) setupMockUpstreams(_ *testing.T) {
	// NPM Mock Server
	npmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[NPM Mock] Request received: %s %s", r.Method, r.URL.Path)
		switch r.URL.Path {
		case "/express":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{
				"name": "express",
				"version": "4.18.2",
				"description": "Fast, unopinionated, minimalist web framework"
			}`)); err != nil {
				log.Printf("Failed to write NPM response: %v", err)
			}
		case "/express/-/express-4.18.2.tgz":
			log.Printf("[NPM Mock] Tarball request matched!")
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("mock express tarball content")); err != nil {
				log.Printf("Failed to write tarball content: %v", err)
			}
		default:
			log.Printf("[NPM Mock] Path not found: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			if _, err := w.Write([]byte(`{"error": "Not Found"}`)); err != nil {
				log.Printf("Failed to write error response: %v", err)
			}
		}
	}))
	env.MockUpstreams["npm"] = npmServer
	env.cleanup = append(env.cleanup, npmServer.Close)

	// Maven Mock Server
	mavenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/junit/junit/4.13.2/junit-4.13.2.pom":
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
				<project xmlns="http://maven.apache.org/POM/4.0.0">
					<groupId>junit</groupId>
					<artifactId>junit</artifactId>
					<version>4.13.2</version>
				</project>`)); err != nil {
				log.Printf("Failed to write Maven POM: %v", err)
			}
		case "/junit/junit/4.13.2/junit-4.13.2.jar":
			w.Header().Set("Content-Type", "application/java-archive")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("mock junit jar content")); err != nil {
				log.Printf("Failed to write JAR content: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
			if _, err := w.Write([]byte("Not Found")); err != nil {
				log.Printf("Failed to write error response: %v", err)
			}
		}
	}))
	env.MockUpstreams["maven"] = mavenServer
	env.cleanup = append(env.cleanup, mavenServer.Close)

	// APT Mock Server
	aptServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ubuntu/dists/jammy/Release":
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`Origin: Ubuntu
Label: Ubuntu
Suite: jammy
Version: 22.04
Codename: jammy`)); err != nil {
				log.Printf("Failed to write APT Release: %v", err)
			}
		case "/ubuntu/dists/jammy/main/binary-amd64/Packages":
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`Package: nginx
Version: 1.18.0-6ubuntu14.4
Architecture: amd64
Maintainer: Ubuntu Developers
Description: small, powerful, scalable web/proxy server`)); err != nil {
				log.Printf("Failed to write APT Packages: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
			if _, err := w.Write([]byte("Not Found")); err != nil {
				log.Printf("Failed to write error response: %v", err)
			}
		}
	}))
	env.MockUpstreams["apt"] = aptServer
	env.cleanup = append(env.cleanup, aptServer.Close)

	// PIP Mock Server
	pipServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only allow GET and HEAD methods
		if r.Method != "GET" && r.Method != "HEAD" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			if _, err := w.Write([]byte("Method Not Allowed")); err != nil {
				log.Printf("Failed to write error response: %v", err)
			}
			return
		}

		// Normalize path by removing trailing slash for comparison
		path := strings.TrimSuffix(r.URL.Path, "/")

		switch path {
		case "/simple/requests":
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`<!DOCTYPE html>
				<html>
				<head><title>Links for requests</title></head>
				<body>
					<h1>Links for requests</h1>
					<a href="requests-2.28.1-py3-none-any.whl">requests-2.28.1-py3-none-any.whl</a>
				</body>
				</html>`)); err != nil {
				log.Printf("Failed to write PIP simple response: %v", err)
			}
		case "/simple/requests/requests-2.28.1-py3-none-any.whl":
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("mock requests wheel content")); err != nil {
				log.Printf("Failed to write wheel content: %v", err)
			}
		case "/pypi/requests/json":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{
				"info": {
					"name": "requests",
					"version": "2.28.1",
					"summary": "Python HTTP for Humans."
				}
			}`)); err != nil {
				log.Printf("Failed to write PIP JSON response: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
			if _, err := w.Write([]byte("Not Found")); err != nil {
				log.Printf("Failed to write error response: %v", err)
			}
		}
	}))
	env.MockUpstreams["pip"] = pipServer
	env.cleanup = append(env.cleanup, pipServer.Close)

	// Docker Mock Server
	dockerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{}`)); err != nil {
				log.Printf("Failed to write Docker API response: %v", err)
			}
		case "/v2/library/nginx/manifests/latest":
			w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{
				"schemaVersion": 2,
				"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
				"config": {
					"mediaType": "application/vnd.docker.container.image.v1+json",
					"size": 7023,
					"digest": "sha256:mock-config-digest"
				},
				"layers": [
					{
						"mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
						"size": 25,
						"digest": "sha256:4d8c5374677d80499161a0df308f361ecc2cb794ae6326e23931b6e4f66c4a10"
					}
				]
			}`)); err != nil {
				log.Printf("Failed to write Docker manifest: %v", err)
			}
		case "/v2/library/nginx/blobs/sha256:4d8c5374677d80499161a0df308f361ecc2cb794ae6326e23931b6e4f66c4a10":
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Docker-Content-Digest", "sha256:4d8c5374677d80499161a0df308f361ecc2cb794ae6326e23931b6e4f66c4a10")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("mock docker layer content")); err != nil {
				log.Printf("Failed to write Docker blob: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
			if _, err := w.Write([]byte(`{"errors":[{"code":"NAME_UNKNOWN","message":"repository name not known"}]}`)); err != nil { //nolint:lll
				log.Printf("Failed to write error response: %v", err)
			}
		}
	}))
	env.MockUpstreams["docker"] = dockerServer
	env.cleanup = append(env.cleanup, dockerServer.Close)

	// YUM Mock Server
	yumServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/centos/8/BaseOS/x86_64/os/repodata/repomd.xml":
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
				<repomd xmlns="http://linux.duke.edu/metadata/repo">
					<revision>1640995200</revision>
					<data type="primary">
						<location href="repodata/primary.xml.gz"/>
					</data>
				</repomd>`)); err != nil {
				log.Printf("Failed to write YUM repomd: %v", err)
			}
		case "/centos/8/BaseOS/x86_64/os/repodata/primary.xml.gz":
			w.Header().Set("Content-Type", "application/x-gzip")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("mock compressed primary.xml content")); err != nil {
				log.Printf("Failed to write YUM primary: %v", err)
			}
		case "/centos/8/BaseOS/x86_64/os/Packages/nginx-1.20.1-1.el8.x86_64.rpm":
			w.Header().Set("Content-Type", "application/x-rpm")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("mock rpm content")); err != nil {
				log.Printf("Failed to write RPM content: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
			if _, err := w.Write([]byte("Not Found")); err != nil {
				log.Printf("Failed to write error response: %v", err)
			}
		}
	}))
	env.MockUpstreams["yum"] = yumServer
	env.cleanup = append(env.cleanup, yumServer.Close)

	// APK Mock Server
	apkServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/alpine/v3.16/main/x86_64/APKINDEX.tar.gz":
			w.Header().Set("Content-Type", "application/x-gzip")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("mock APKINDEX content")); err != nil {
				log.Printf("Failed to write APK index: %v", err)
			}
		case "/alpine/v3.16/main/x86_64/nginx-1.22.0-r1.apk":
			w.Header().Set("Content-Type", "application/vnd.alpine.apk")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("mock apk content")); err != nil {
				log.Printf("Failed to write APK content: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
			if _, err := w.Write([]byte("Not Found")); err != nil {
				log.Printf("Failed to write error response: %v", err)
			}
		}
	}))
	env.MockUpstreams["apk"] = apkServer
	env.cleanup = append(env.cleanup, apkServer.Close)
}

// setupConfiguration ProxyND 설정 생성
func (env *IntegrationTestEnvironment) setupConfiguration(t *testing.T) {
	// 통합 설정 생성
	env.Config = &config.UnifiedConfig{
		Server: config.ServerConfig{
			Port: 0, // 동적 포트 할당
			Host: "127.0.0.1",
		},
		Cache: config.CacheConfig{
			Backend: "file",
			File: config.FileCacheConfig{
				Directory: env.CacheDir,
			},
			TTL:      time.Hour,
			MaxSize:  "100MB",
			MaxItems: 10000,
		},
		Registries: config.RegistryConfig{
			NPM: config.NPMRegistryConfig{
				Enabled:  true,
				Upstream: env.MockUpstreams["npm"].URL,
				Timeout:  30 * time.Second,
			},
			Maven: config.MavenRegistryConfig{
				Enabled: true,
				Repositories: []config.MavenRepositoryConfig{
					{
						ID:   "central",
						Name: "Maven Central",
						URL:  env.MockUpstreams["maven"].URL,
					},
				},
			},
			APT: config.APTRegistryConfig{
				Enabled: true,
				Mirrors: map[string][]config.APTMirror{
					"ubuntu": {
						{
							Name: "main",
							URL:  env.MockUpstreams["apt"].URL + "/ubuntu",
						},
					},
				},
			},
		},
		Logging: config.UnifiedLoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
	}

	// 설정 파일 저장
	env.saveConfigFiles(t)
}

// saveConfigFiles 설정 파일들을 파일시스템에 저장
func (env *IntegrationTestEnvironment) saveConfigFiles(t *testing.T) {
	// 환경 변수 설정
	_ = os.Setenv("CONFIG_DIR", env.ConfigDir)
	_ = os.Setenv("STORAGE_DIR", env.CacheDir)

	// global.yaml 생성
	globalYAML := fmt.Sprintf(`
server:
  port: 0
  host: "127.0.0.1"
cache:
  backend: "file"
  file:
    directory: "%s"
  ttl: "1h"
  max_size: 104857600
  max_items: 10000
`, env.CacheDir)

	err := os.WriteFile(filepath.Join(env.ConfigDir, "global.yaml"), []byte(globalYAML), 0o644)
	require.NoError(t, err)

	// npm-proxy.yaml 생성
	npmYAML := fmt.Sprintf(`
path: "/npm"
use_cache: true
proxies:
  default:
    - name: "npmjs"
      url: "%s"
`, env.MockUpstreams["npm"].URL)

	err = os.WriteFile(filepath.Join(env.ConfigDir, "npm-proxy.yaml"), []byte(npmYAML), 0o644)
	require.NoError(t, err)

	// maven-proxy.yaml 생성
	mavenYAML := fmt.Sprintf(`
path: "/maven"
use_cache: true
proxies:
  - name: "central"
    url: "%s"
`, env.MockUpstreams["maven"].URL)

	err = os.WriteFile(filepath.Join(env.ConfigDir, "maven-proxy.yaml"), []byte(mavenYAML), 0o644)
	require.NoError(t, err)

	// apt-proxy.yaml 생성
	aptYAML := fmt.Sprintf(`
path: "/apt"
use_cache: true
proxies:
  ubuntu:
    - name: "main"
      url: "%s/ubuntu"
`, env.MockUpstreams["apt"].URL)

	err = os.WriteFile(filepath.Join(env.ConfigDir, "apt-proxy.yaml"), []byte(aptYAML), 0o644)
	require.NoError(t, err)

	// apk-proxy.yaml 생성
	apkYAML := fmt.Sprintf(`
path: "/apk"
use_cache: true
proxies:
  - name: "alpine"
    url: "%s"
`, env.MockUpstreams["apk"].URL)

	err = os.WriteFile(filepath.Join(env.ConfigDir, "apk-proxy.yaml"), []byte(apkYAML), 0o644)
	require.NoError(t, err)

	// yum-proxy.yaml 생성
	yumYAML := fmt.Sprintf(`
path: "/yum"
use_cache: true
proxies:
  - name: "centos"
    url: "%s"
`, env.MockUpstreams["yum"].URL)

	err = os.WriteFile(filepath.Join(env.ConfigDir, "yum-proxy.yaml"), []byte(yumYAML), 0o644)
	require.NoError(t, err)

	// docker-proxy.yaml 생성
	dockerYAML := fmt.Sprintf(`
path: "/docker"
use_cache: true
proxies:
  - name: "dockerhub"
    url: "%s"
`, env.MockUpstreams["docker"].URL)

	err = os.WriteFile(filepath.Join(env.ConfigDir, "docker-proxy.yaml"), []byte(dockerYAML), 0o644)
	require.NoError(t, err)

	// pip-proxy.yaml 생성
	pipYAML := fmt.Sprintf(`
path: "/pip"
use_cache: true
proxies:
  - name: "pypi"
    url: "%s"
`, env.MockUpstreams["pip"].URL)

	err = os.WriteFile(filepath.Join(env.ConfigDir, "pip-proxy.yaml"), []byte(pipYAML), 0o644)
	require.NoError(t, err)
}

// setupProxyServer ProxyND 서버 설정 및 시작
func (env *IntegrationTestEnvironment) setupProxyServer(_ *testing.T) {
	// 애플리케이션 설정 생성
	appConfig := &app.Config{
		Port:        "0", // 동적 포트 할당
		Version:     "test",
		BuildTime:   "test",
		CommitSHA:   "test",
		StorageDir:  env.CacheDir,
		ConfigDir:   env.ConfigDir,
		CacheMaxAge: time.Hour,
	}

	// 컨테이너 생성
	container := app.NewContainer(appConfig)

	// Fiber 앱 생성 (BaseRouter 사용)
	fiberApp := routers.BaseRouter()

	// 프록시 라우터 추가
	routers.ProxyRouter(fiberApp)
	routers.HealthRouter(fiberApp)
	routers.CacheRouter(fiberApp)
	routers.ConfigRouter(fiberApp)
	routers.StatusRouter(fiberApp)
	routers.UserRouter(fiberApp)
	routers.TestRouter(fiberApp)
	routers.WebhookRouter(fiberApp)
	// MetricsRouter는 Prometheus 글로벌 레지스트리 중복 등록 문제로 인해
	// 통합 테스트에서 비활성화 (메트릭 테스트는 별도 수행 필요)
	// routers.MetricsRouter(fiberApp, env.Config)

	// 컨테이너를 앱 로컬에 저장 (handlers가 사용할 수 있도록)
	fiberApp.Use(func(c *fiber.Ctx) error {
		c.Locals("container", container)
		return c.Next()
	})

	// Test 메서드용으로 설정 (포트 바인딩 없이)
	env.ProxyServer = fiberApp
	env.ServerURL = "http://test-server" // Test 메서드용 더미 URL

	env.cleanup = append(env.cleanup, func() {
		if container != nil {
			_ = container.Close()
		}
		// Metrics 리셋하여 다음 테스트에서 재초기화 가능하게 함
		metrics.ResetMetrics()
		routers.ResetMetricsRouter()
	})
}

// MakeRequest 통합 테스트용 HTTP 요청 실행
// body parameter can be:
//   - nil (no body)
//   - io.Reader (request body)
//   - map[string]string (headers) - for backward compatibility
func (env *IntegrationTestEnvironment) MakeRequest(method, path string,
	bodyOrHeaders interface{},
) (*http.Response, error) {
	var body io.Reader
	var headers map[string]string

	// Determine parameter type
	switch v := bodyOrHeaders.(type) {
	case nil:
		// No body, no headers
		body = nil
		headers = nil
	case io.Reader:
		// Request body provided
		body = v
		headers = nil
	case map[string]string:
		// Headers provided (backward compatibility)
		body = nil
		headers = v
	default:
		return nil, fmt.Errorf("unsupported parameter type: %T", bodyOrHeaders)
	}

	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return nil, err
	}

	// Set default Content-Type for POST/PUT/PATCH with body
	if body != nil && (method == "POST" || method == "PUT" || method == "PATCH") {
		req.Header.Set("Content-Type", "application/json")
	}

	// 헤더 설정
	if headers != nil {
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	}

	// Fiber의 Test 메서드 사용
	resp, err := env.ProxyServer.Test(req, -1)
	return resp, err
}

// WaitForUpstream Mock upstream 서버가 준비될 때까지 대기
func (env *IntegrationTestEnvironment) WaitForUpstream(upstreamName string, timeout time.Duration) error {
	upstream, exists := env.MockUpstreams[upstreamName]
	if !exists {
		return fmt.Errorf("upstream %s not found", upstreamName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for upstream %s", upstreamName)
		default:
			// Health check
			resp, err := http.Get(upstream.URL)
			if err == nil {
				_ = resp.Body.Close()
				if resp.StatusCode < 500 {
					return nil
				}
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// Cleanup 테스트 환경 정리
func (env *IntegrationTestEnvironment) Cleanup() {
	for i := len(env.cleanup) - 1; i >= 0; i-- {
		if env.cleanup[i] != nil {
			env.cleanup[i]()
		}
	}
}

// SetMockResponse Mock upstream의 특정 경로에 대한 응답 설정
func (env *IntegrationTestEnvironment) SetMockResponse(_, _ string, _ MockUpstreamResponse) {
	// 실제 구현에서는 Mock 서버의 응답을 동적으로 변경하는 로직 필요
	// 현재는 기본 핸들러가 설정되어 있음
}

// GetCacheStats 캐시 통계 조회
func (env *IntegrationTestEnvironment) GetCacheStats() (map[string]interface{}, error) {
	resp, err := env.MakeRequest("GET", "/api/cache/stats", nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	// JSON 응답 파싱 로직 (실제 구현 필요)
	return map[string]interface{}{
		"hits":   0,
		"misses": 0,
		"size":   0,
	}, nil
}
