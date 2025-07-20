package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"

	"proxynd/configs"
	"proxynd/internal/app"
	"proxynd/routers"
)

// IntegrationTestEnvironment 통합 테스트 환경
type IntegrationTestEnvironment struct {
	// ProxyND 서버
	ProxyServer *fiber.App
	ServerURL   string

	// Mock upstream 서버들
	MockUpstreams map[string]*httptest.Server

	// 테스트 설정
	Config    *configs.UnifiedConfig
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
func (env *IntegrationTestEnvironment) setupMockUpstreams(_ *testing.T) {
	// NPM Mock Server
	npmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/express":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"name": "express",
				"version": "4.18.2",
				"description": "Fast, unopinionated, minimalist web framework"
			}`))
		case "/express/-/express-4.18.2.tgz":
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("mock express tarball content"))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": "Not Found"}`))
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
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
				<project xmlns="http://maven.apache.org/POM/4.0.0">
					<groupId>junit</groupId>
					<artifactId>junit</artifactId>
					<version>4.13.2</version>
				</project>`))
		case "/junit/junit/4.13.2/junit-4.13.2.jar":
			w.Header().Set("Content-Type", "application/java-archive")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("mock junit jar content"))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Not Found"))
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
			_, _ = w.Write([]byte(`Origin: Ubuntu
Label: Ubuntu
Suite: jammy
Version: 22.04
Codename: jammy`))
		case "/ubuntu/dists/jammy/main/binary-amd64/Packages":
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`Package: nginx
Version: 1.18.0-6ubuntu14.4
Architecture: amd64
Maintainer: Ubuntu Developers
Description: small, powerful, scalable web/proxy server`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Not Found"))
		}
	}))
	env.MockUpstreams["apt"] = aptServer
	env.cleanup = append(env.cleanup, aptServer.Close)
}

// setupConfiguration ProxyND 설정 생성
func (env *IntegrationTestEnvironment) setupConfiguration(t *testing.T) {
	// 통합 설정 생성
	env.Config = &configs.UnifiedConfig{
		Server: configs.ServerConfig{
			Port: 0, // 동적 포트 할당
			Host: "127.0.0.1",
		},
		Cache: configs.CacheConfig{
			Backend: "file",
			File: configs.FileCacheConfig{
				Directory: env.CacheDir,
			},
			TTL:      time.Hour,
			MaxSize:  "100MB",
			MaxItems: 10000,
		},
		Registries: configs.RegistryConfig{
			NPM: configs.NPMRegistryConfig{
				Enabled:  true,
				Upstream: env.MockUpstreams["npm"].URL,
				Timeout:  30 * time.Second,
			},
			Maven: configs.MavenRegistryConfig{
				Enabled: true,
				Repositories: []configs.MavenRepositoryConfig{
					{
						ID:   "central",
						Name: "Maven Central",
						URL:  env.MockUpstreams["maven"].URL,
					},
				},
			},
			APT: configs.APTRegistryConfig{
				Enabled: true,
				Mirrors: map[string][]configs.APTMirror{
					"ubuntu": {
						{
							Name: "main",
							URL:  env.MockUpstreams["apt"].URL + "/ubuntu",
						},
					},
				},
			},
		},
		Logging: configs.UnifiedLoggingConfig{
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

	err := os.WriteFile(filepath.Join(env.ConfigDir, "global.yaml"), []byte(globalYAML), 0644)
	require.NoError(t, err)

	// npm-proxy.yaml 생성
	npmYAML := fmt.Sprintf(`
path: "/npm"
use_cache: true
proxies:
  - name: "npmjs"
    url: "%s"
`, env.MockUpstreams["npm"].URL)

	err = os.WriteFile(filepath.Join(env.ConfigDir, "npm-proxy.yaml"), []byte(npmYAML), 0644)
	require.NoError(t, err)

	// maven-proxy.yaml 생성
	mavenYAML := fmt.Sprintf(`
path: "/maven"
use_cache: true
proxies:
  - name: "central"
    url: "%s"
`, env.MockUpstreams["maven"].URL)

	err = os.WriteFile(filepath.Join(env.ConfigDir, "maven-proxy.yaml"), []byte(mavenYAML), 0644)
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

	err = os.WriteFile(filepath.Join(env.ConfigDir, "apt-proxy.yaml"), []byte(aptYAML), 0644)
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
	})
}

// MakeRequest 통합 테스트용 HTTP 요청 실행
func (env *IntegrationTestEnvironment) MakeRequest(method, path string,
	headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		return nil, err
	}

	// 헤더 설정
	for key, value := range headers {
		req.Header.Set(key, value)
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
