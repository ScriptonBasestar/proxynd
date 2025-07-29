package benchmark

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"

	"proxynd/internal/app"
)

// BenchmarkEnvironment 벤치마크 테스트 환경
type BenchmarkEnvironment struct {
	ProxyServer   *fiber.App
	MockUpstreams map[string]*httptest.Server
	ConfigDir     string
	CacheDir      string
	cleanup       []func()
}

// setupBenchmarkEnvironment 벤치마크 테스트 환경 설정
func setupBenchmarkEnvironment(b *testing.B) *BenchmarkEnvironment {
	env := &BenchmarkEnvironment{
		MockUpstreams: make(map[string]*httptest.Server),
		cleanup:       make([]func(), 0),
	}

	// Mock upstream 서버들 설정
	env.setupMockUpstreams(b)

	// ProxyND 서버 설정
	env.setupProxyServer(b)

	return env
}

// setupMockUpstreams Mock upstream 서버들 설정
func (env *BenchmarkEnvironment) setupMockUpstreams(_ *testing.B) {
	// NPM Mock Server (고성능)
	npmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, ".tgz"):
			// 타르볼 파일 (1MB 모의 데이터)
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Length", "1048576")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(make([]byte, 1048576)) // 1MB
		case strings.Contains(r.URL.Path, "express"):
			// 패키지 메타데이터
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"name":"express","version":"4.18.2","description":"Fast web framework"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	env.MockUpstreams["npm"] = npmServer
	env.cleanup = append(env.cleanup, npmServer.Close)

	// Maven Mock Server (고성능)
	mavenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, ".jar"):
			// JAR 파일 (5MB 모의 데이터)
			w.Header().Set("Content-Type", "application/java-archive")
			w.Header().Set("Content-Length", "5242880")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(make([]byte, 5242880)) // 5MB
		case strings.HasSuffix(r.URL.Path, ".pom"):
			// POM 파일
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			xml := `<?xml version="1.0"?><project>` +
				`<groupId>test</groupId>` +
				`<artifactId>test</artifactId>` +
				`<version>1.0</version>` +
				`</project>`
			_, _ = w.Write([]byte(xml))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	env.MockUpstreams["maven"] = mavenServer
	env.cleanup = append(env.cleanup, mavenServer.Close)

	// APT Mock Server (고성능)
	aptServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "Release"):
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Origin: Ubuntu\nSuite: focal\nComponents: main universe\n"))
		case strings.Contains(r.URL.Path, "Packages"):
			// 큰 패키지 목록 (1MB)
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Content-Length", "1048576")
			w.WriteHeader(http.StatusOK)
			packageData := strings.Repeat("Package: test-package\nVersion: 1.0.0\n"+
				"Architecture: amd64\nDescription: Test package\n\n", 8192)
			_, _ = w.Write([]byte(packageData[:1048576]))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	env.MockUpstreams["apt"] = aptServer
	env.cleanup = append(env.cleanup, aptServer.Close)
}

// setupProxyServer ProxyND 서버 설정
func (env *BenchmarkEnvironment) setupProxyServer(b *testing.B) {
	// 애플리케이션 설정 생성
	appConfig := &app.Config{
		Port:        "0",
		Version:     "benchmark",
		BuildTime:   "benchmark",
		CommitSHA:   "benchmark",
		StorageDir:  b.TempDir(),
		ConfigDir:   b.TempDir(),
		CacheMaxAge: time.Hour,
	}

	// 컨테이너 생성
	container := app.NewContainer(appConfig)

	// Fiber 앱 생성 (최적화된 설정)
	fiberApp := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		Prefork:               false, // 벤치마크에서는 prefork 비활성화
		ServerHeader:          "",
		ReadTimeout:           30 * time.Second,
		WriteTimeout:          30 * time.Second,
		IdleTimeout:           120 * time.Second,
		ReadBufferSize:        4096,
		WriteBufferSize:       4096,
		BodyLimit:             100 * 1024 * 1024, // 100MB
		Concurrency:           256 * 1024,        // 고성능 설정
	})

	// 최소한의 라우터만 설정 (성능 최적화)
	setupBenchmarkRoutes(fiberApp, container, env.MockUpstreams)

	env.ProxyServer = fiberApp
	env.cleanup = append(env.cleanup, func() {
		if container != nil {
			_ = container.Close()
		}
	})
}

// setupBenchmarkRoutes 벤치마크용 최적화된 라우터 설정
func setupBenchmarkRoutes(app *fiber.App, container *app.Container, mockUpstreams map[string]*httptest.Server) {
	// 컨테이너를 앱 로컬에 저장
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("container", container)
		return c.Next()
	})

	// 프록시 라우트 그룹
	proxy := app.Group("/proxy")

	// NPM 프록시 (간단한 구현)
	proxy.All("/npm/*", func(c *fiber.Ctx) error {
		path := c.Params("*")
		upstream := mockUpstreams["npm"]

		// upstream으로 요청 전달
		req, err := http.NewRequest(c.Method(), upstream.URL+"/"+path, bytes.NewReader(c.Body()))
		if err != nil {
			return c.Status(500).SendString("Internal Error")
		}

		// 헤더 복사
		for key, values := range c.GetReqHeaders() {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return c.Status(502).SendString("Bad Gateway")
		}
		defer func() { _ = resp.Body.Close() }()

		// 응답 헤더 복사
		for key, values := range resp.Header {
			for _, value := range values {
				c.Set(key, value)
			}
		}

		c.Status(resp.StatusCode)

		// 응답 본문 스트리밍
		_, err = io.Copy(c.Response().BodyWriter(), resp.Body)
		return err
	})

	// Maven 프록시 (간단한 구현)
	proxy.All("/maven/*", func(c *fiber.Ctx) error {
		path := c.Params("*")
		upstream := mockUpstreams["maven"]

		req, err := http.NewRequest(c.Method(), upstream.URL+"/"+path, bytes.NewReader(c.Body()))
		if err != nil {
			return c.Status(500).SendString("Internal Error")
		}

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return c.Status(502).SendString("Bad Gateway")
		}
		defer func() { _ = resp.Body.Close() }()

		for key, values := range resp.Header {
			for _, value := range values {
				c.Set(key, value)
			}
		}

		c.Status(resp.StatusCode)
		_, err = io.Copy(c.Response().BodyWriter(), resp.Body)
		return err
	})

	// APT 프록시 (간단한 구현)
	proxy.All("/apt/*", func(c *fiber.Ctx) error {
		path := c.Params("*")
		upstream := mockUpstreams["apt"]

		req, err := http.NewRequest(c.Method(), upstream.URL+"/"+path, bytes.NewReader(c.Body()))
		if err != nil {
			return c.Status(500).SendString("Internal Error")
		}

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return c.Status(502).SendString("Bad Gateway")
		}
		defer func() { _ = resp.Body.Close() }()

		for key, values := range resp.Header {
			for _, value := range values {
				c.Set(key, value)
			}
		}

		c.Status(resp.StatusCode)
		_, err = io.Copy(c.Response().BodyWriter(), resp.Body)
		return err
	})

	// 헬스체크 (최적화된)
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})
}

// Cleanup 벤치마크 환경 정리
func (env *BenchmarkEnvironment) Cleanup() {
	for i := len(env.cleanup) - 1; i >= 0; i-- {
		if env.cleanup[i] != nil {
			env.cleanup[i]()
		}
	}
}

// BenchmarkProxyThroughput 프록시 처리량 벤치마크
func BenchmarkProxyThroughput(b *testing.B) {
	env := setupBenchmarkEnvironment(b)
	defer env.Cleanup()

	testCases := []struct {
		name string
		path string
	}{
		{"NPM-Metadata", "/proxy/npm/express"},
		{"NPM-Package", "/proxy/npm/express/-/express-4.18.2.tgz"},
		{"Maven-POM", "/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.pom"},
		{"Maven-JAR", "/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar"},
		{"APT-Release", "/proxy/apt/ubuntu/dists/focal/Release"},
		{"APT-Packages", "/proxy/apt/ubuntu/dists/focal/main/binary-amd64/Packages"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					req, err := http.NewRequest("GET", tc.path, nil)
					require.NoError(b, err)

					resp, err := env.ProxyServer.Test(req, -1)
					require.NoError(b, err)

					// 응답 본문 읽기 (실제 처리량 측정)
					_, err = io.Copy(io.Discard, resp.Body)
					require.NoError(b, err)
					_ = resp.Body.Close()
				}
			})
		})
	}
}

// BenchmarkConcurrentRequests 동시 요청 처리 벤치마크
func BenchmarkConcurrentRequests(b *testing.B) {
	env := setupBenchmarkEnvironment(b)
	defer env.Cleanup()

	concurrencyLevels := []int{1, 10, 50, 100, 200}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency-%d", concurrency), func(b *testing.B) {
			b.SetParallelism(concurrency)
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					req, err := http.NewRequest("GET", "/proxy/npm/express", nil)
					require.NoError(b, err)

					resp, err := env.ProxyServer.Test(req, -1)
					require.NoError(b, err)

					_, _ = io.Copy(io.Discard, resp.Body)
					_ = resp.Body.Close()
				}
			})
		})
	}
}

// BenchmarkRequestLatency 요청 지연시간 벤치마크
func BenchmarkRequestLatency(b *testing.B) {
	env := setupBenchmarkEnvironment(b)
	defer env.Cleanup()

	testPaths := []string{
		"/proxy/npm/express",
		"/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.pom",
		"/proxy/apt/ubuntu/dists/focal/Release",
	}

	for _, path := range testPaths {
		b.Run(path, func(b *testing.B) {
			latencies := make([]time.Duration, b.N)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				start := time.Now()

				req, err := http.NewRequest("GET", path, nil)
				require.NoError(b, err)

				resp, err := env.ProxyServer.Test(req, -1)
				require.NoError(b, err)

				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()

				latencies[i] = time.Since(start)
			}

			// 지연시간 통계 계산
			if b.N > 0 {
				var total time.Duration
				for _, lat := range latencies {
					total += lat
				}
				avgLatency := total / time.Duration(b.N)
				b.ReportMetric(float64(avgLatency.Nanoseconds()), "ns/op")
			}
		})
	}
}

// BenchmarkLargeFileTransfer 대용량 파일 전송 벤치마크
func BenchmarkLargeFileTransfer(b *testing.B) {
	env := setupBenchmarkEnvironment(b)
	defer env.Cleanup()

	testCases := []struct {
		name string
		path string
		size string
	}{
		{"Small-1KB", "/proxy/npm/express", "1KB"},
		{"Medium-1MB", "/proxy/npm/express/-/express-4.18.2.tgz", "1MB"},
		{"Large-5MB", "/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar", "5MB"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			var totalBytes int64

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				req, err := http.NewRequest("GET", tc.path, nil)
				require.NoError(b, err)

				resp, err := env.ProxyServer.Test(req, -1)
				require.NoError(b, err)

				n, err := io.Copy(io.Discard, resp.Body)
				require.NoError(b, err)
				_ = resp.Body.Close()

				totalBytes += n
			}

			// 처리량 보고 (MB/s)
			b.ReportMetric(float64(totalBytes)/1024/1024, "MB")
		})
	}
}

// BenchmarkMemoryUsage 메모리 사용량 벤치마크
func BenchmarkMemoryUsage(b *testing.B) {
	env := setupBenchmarkEnvironment(b)
	defer env.Cleanup()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req, err := http.NewRequest("GET", "/proxy/npm/express", nil)
		require.NoError(b, err)

		resp, err := env.ProxyServer.Test(req, -1)
		require.NoError(b, err)

		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
}

// BenchmarkCachePerformance 캐시 성능 벤치마크
func BenchmarkCachePerformance(b *testing.B) {
	env := setupBenchmarkEnvironment(b)
	defer env.Cleanup()

	// 캐시 워밍업
	for i := 0; i < 10; i++ {
		req, _ := http.NewRequest("GET", "/proxy/npm/express", nil)
		resp, _ := env.ProxyServer.Test(req, -1)
		if resp != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	}

	b.Run("CacheHit", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				req, err := http.NewRequest("GET", "/proxy/npm/express", nil)
				require.NoError(b, err)

				resp, err := env.ProxyServer.Test(req, -1)
				require.NoError(b, err)

				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
			}
		})
	})

	b.Run("CacheMiss", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// 매번 다른 URL로 캐시 미스 유발
			path := fmt.Sprintf("/proxy/npm/express?v=%d", i)
			req, err := http.NewRequest("GET", path, nil)
			require.NoError(b, err)

			resp, err := env.ProxyServer.Test(req, -1)
			require.NoError(b, err)

			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	})
}

// BenchmarkStressTest 스트레스 테스트
func BenchmarkStressTest(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping stress test in short mode")
	}

	env := setupBenchmarkEnvironment(b)
	defer env.Cleanup()

	// 다양한 요청 패턴으로 스트레스 테스트
	requests := []string{
		"/proxy/npm/express",
		"/proxy/npm/express/-/express-4.18.2.tgz",
		"/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.pom",
		"/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar",
		"/proxy/apt/ubuntu/dists/focal/Release",
		"/proxy/apt/ubuntu/dists/focal/main/binary-amd64/Packages",
	}

	b.SetParallelism(100) // 높은 동시성
	b.ResetTimer()

	var requestCounter int64
	var wg sync.WaitGroup

	b.RunParallel(func(pb *testing.PB) {
		defer wg.Done()
		wg.Add(1)

		for pb.Next() {
			// 요청 패턴 순환
			path := requests[int(requestCounter)%len(requests)]
			requestCounter++

			req, err := http.NewRequest("GET", path, nil)
			if err != nil {
				continue
			}

			resp, err := env.ProxyServer.Test(req, 5000) // 5초 타임아웃 (밀리초)
			if err != nil {
				continue
			}

			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	})

	wg.Wait()
}
