package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/adaptor/v2"

	"proxynd/configs"
	"proxynd/internal/app"
)

// TestServer 통합 테스트용 서버 구조체
type TestServer struct {
	App       *fiber.App
	Server    *httptest.Server
	Container *app.Container
	Config    *configs.UnifiedConfig
}

// SetupTestServer 테스트 서버 설정
func SetupTestServer(t *testing.T) *TestServer {
	// 테스트용 통합 설정
	testConfig := &configs.UnifiedConfig{
		Server: configs.ServerConfig{
			Port: 8080,
			Host: "localhost",
		},
		Registries: configs.RegistryConfig{
			APT: configs.APTRegistryConfig{
				Enabled: true,
				Mirrors: map[string][]configs.APTMirror{
					"ubuntu": {
						{
							Name:       "test-mirror",
							URL:        "http://archive.ubuntu.com/ubuntu",
							Suites:     []string{"focal"},
							Components: []string{"main", "universe"},
						},
					},
				},
			},
			Maven: configs.MavenRegistryConfig{
				Enabled: true,
				Repositories: []configs.MavenRepositoryConfig{
					{
						ID:        "central",
						Name:      "Maven Central",
						URL:       "https://repo1.maven.org/maven2",
						Releases:  true,
						Snapshots: false,
					},
				},
			},
			NPM: configs.NPMRegistryConfig{
				Enabled:  true,
				Upstream: "https://registry.npmjs.org",
				Timeout:  30 * time.Second,
			},
		},
		Cache: configs.CacheConfig{
			Backend:  "file",
			TTL:      time.Hour,
			MaxSize:  "100MB",
			MaxItems: 10000,
			File: configs.FileCacheConfig{
				Directory: "/tmp/proxynd-test-cache",
			},
		},
		Security: configs.SecurityConfig{
			Authentication: configs.AuthenticationConfig{
				BasicAuth: nil,
				OAuth2:    nil,
			},
		},
	}

	// 앱 설정 생성
	appConfig := &app.Config{
		ConfigDir:  "/tmp/proxynd-test-config",
		StorageDir: "/tmp/proxynd-test-storage",
	}

	// 컨테이너 생성
	container := app.NewContainer(appConfig)

	// Fiber 앱 설정
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ReadTimeout:          30 * time.Second,
		WriteTimeout:         30 * time.Second,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error":   true,
				"message": err.Error(),
				"code":    code,
			})
		},
	})

	// 미들웨어 설정
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))
	
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders: "*",
	}))

	// 라우터 설정
	setupRoutes(app, container)

	// 테스트 서버 시작 (Fiber를 http.Handler로 변환)
	server := httptest.NewServer(adaptor.FiberApp(app))

	return &TestServer{
		App:       app,
		Server:    server,
		Container: container,
		Config:    testConfig,
	}
}

// setupRoutes 라우터 설정
func setupRoutes(app *fiber.App, container *app.Container) {
	// 헬스체크 엔드포인트
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "healthy",
			"timestamp": time.Now(),
			"service":   "proxynd-test",
		})
	})

	// 프록시 라우트 그룹
	proxy := app.Group("/proxy")

	// APT 프록시 라우트
	proxy.All("/apt/*", func(c *fiber.Ctx) error {
		path := c.Params("*")
		c.Set("X-Proxy-Type", "apt")
		c.Set("X-Cache-Status", "MISS") // 실제 구현에서는 캐시 로직에 따라 결정
		
		// 간단한 모의 응답 (실제 구현에서는 핸들러 사용)
		if path == "dists/focal/Release" {
			return c.SendString("Origin: Ubuntu\nSuite: focal\nArchitectures: amd64 arm64\n")
		}
		if path == "dists/focal/main/binary-amd64/Packages" {
			return c.SendString("Package: test-package\nVersion: 1.0.0\nArchitecture: amd64\n")
		}
		
		return c.Status(404).SendString("Not Found")
	})

	// Maven 프록시 라우트
	proxy.All("/maven/*", func(c *fiber.Ctx) error {
		path := c.Params("*")
		c.Set("X-Proxy-Type", "maven")
		c.Set("X-Cache-Status", "MISS")
		
		// 간단한 모의 응답
		if path == "org/springframework/spring-core/5.3.21/spring-core-5.3.21.pom" {
			c.Set("Content-Type", "application/xml")
			return c.SendString(`<?xml version="1.0" encoding="UTF-8"?>
<project>
    <groupId>org.springframework</groupId>
    <artifactId>spring-core</artifactId>
    <version>5.3.21</version>
</project>`)
		}
		
		if path == "org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar" {
			c.Set("Content-Type", "application/java-archive")
			// 모의 JAR 데이터
			return c.Send(make([]byte, 1024*1024)) // 1MB 모의 데이터
		}
		
		return c.Status(404).SendString("Not Found")
	})

	// NPM 프록시 라우트
	proxy.All("/npm/*", func(c *fiber.Ctx) error {
		path := c.Params("*")
		c.Set("X-Proxy-Type", "npm")
		c.Set("X-Cache-Status", "MISS")
		
		// 간단한 모의 응답
		if path == "express" {
			c.Set("Content-Type", "application/json")
			return c.JSON(fiber.Map{
				"name":        "express",
				"version":     "4.18.1",
				"description": "Fast, unopinionated, minimalist web framework",
			})
		}
		
		return c.Status(404).SendString("Not Found")
	})

	// 메트릭 엔드포인트
	app.Get("/metrics", func(c *fiber.Ctx) error {
		return c.SendString("# HELP proxynd_requests_total Total number of requests\n# TYPE proxynd_requests_total counter\nproxynd_requests_total 100\n")
	})
}

// Close 테스트 서버 종료
func (ts *TestServer) Close() {
	ts.Server.Close()
}

// URL 테스트 서버 URL 반환
func (ts *TestServer) URL() string {
	return ts.Server.URL
}

// BaseURL 기본 URL 반환 (프로토콜 포함)
func (ts *TestServer) BaseURL() string {
	return ts.Server.URL
}

// ProxyURL 프록시 URL 반환
func (ts *TestServer) ProxyURL(proxyType, path string) string {
	return fmt.Sprintf("%s/proxy/%s/%s", ts.Server.URL, proxyType, path)
}

// GetClient HTTP 클라이언트 반환
func (ts *TestServer) GetClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
	}
}

// WaitForReady 서버가 준비될 때까지 대기
func (ts *TestServer) WaitForReady() error {
	client := ts.GetClient()
	
	for i := 0; i < 10; i++ {
		resp, err := client.Get(ts.URL() + "/health")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}
	
	return fmt.Errorf("서버가 준비되지 않음")
}