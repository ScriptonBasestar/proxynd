package benchmark

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"

	"proxynd/internal/app"
)

// HandlerBenchmarkEnvironment 핸들러별 벤치마크 환경
type HandlerBenchmarkEnvironment struct {
	ProxyServer   *fiber.App
	MockUpstreams map[string]*httptest.Server
	ConfigDir     string
	CacheDir      string
	cleanup       []func()
}

// setupHandlerBenchmarkEnvironment 핸들러별 벤치마크 환경 설정
func setupHandlerBenchmarkEnvironment(b *testing.B) *HandlerBenchmarkEnvironment {
	env := &HandlerBenchmarkEnvironment{
		MockUpstreams: make(map[string]*httptest.Server),
		cleanup:       make([]func(), 0),
	}

	// 모든 프록시 타입의 Mock upstream 서버들 설정
	env.setupAllMockUpstreams(b)

	// ProxyND 서버 설정
	env.setupHandlerProxyServer(b)

	return env
}

// setupAllMockUpstreams 모든 프록시 타입의 Mock upstream 서버들 설정
func (env *HandlerBenchmarkEnvironment) setupAllMockUpstreams(_ *testing.B) {
	// NPM Mock Server
	npmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, ".tgz"):
			// 타르볼 파일 (2MB 모의 데이터)
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Length", "2097152")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(make([]byte, 2097152)) // 2MB
		case strings.Contains(r.URL.Path, "express"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"name":"express","version":"4.18.2","description":"Fast web framework","dist":{"tarball":"` + r.Host + `/express/-/express-4.18.2.tgz"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	env.MockUpstreams["npm"] = npmServer
	env.cleanup = append(env.cleanup, npmServer.Close)

	// Maven Mock Server
	mavenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, ".jar"):
			// JAR 파일 (10MB 모의 데이터)
			w.Header().Set("Content-Type", "application/java-archive")
			w.Header().Set("Content-Length", "10485760")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(make([]byte, 10485760)) // 10MB
		case strings.HasSuffix(r.URL.Path, ".pom"):
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			xml := `<?xml version="1.0"?><project><groupId>org.springframework</groupId><artifactId>spring-core</artifactId><version>5.3.21</version></project>`
			_, _ = w.Write([]byte(xml))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	env.MockUpstreams["maven"] = mavenServer
	env.cleanup = append(env.cleanup, mavenServer.Close)

	// PIP Mock Server
	pipServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/simple/"):
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<!DOCTYPE html><html><head><title>Links for requests</title></head><body><h1>Links for requests</h1><a href="requests-2.28.1-py3-none-any.whl">requests-2.28.1-py3-none-any.whl</a></body></html>`))
		case strings.Contains(r.URL.Path, "/pypi/") && strings.HasSuffix(r.URL.Path, "/json"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"info":{"name":"requests","version":"2.28.1","summary":"Python HTTP for Humans."}}`))
		case strings.HasSuffix(r.URL.Path, ".whl"):
			// Wheel 파일 (3MB 모의 데이터)
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Length", "3145728")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(make([]byte, 3145728)) // 3MB
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	env.MockUpstreams["pip"] = pipServer
	env.cleanup = append(env.cleanup, pipServer.Close)

	// Docker Mock Server
	dockerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v2/":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		case strings.Contains(r.URL.Path, "/manifests/"):
			w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
			w.WriteHeader(http.StatusOK)
			manifest := `{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":7023,"digest":"sha256:config-digest"},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":50000000,"digest":"sha256:layer-digest"}]}`
			_, _ = w.Write([]byte(manifest))
		case strings.Contains(r.URL.Path, "/blobs/"):
			// Docker 레이어 (50MB 모의 데이터)
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Length", "52428800")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(make([]byte, 52428800)) // 50MB
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	env.MockUpstreams["docker"] = dockerServer
	env.cleanup = append(env.cleanup, dockerServer.Close)

	// YUM Mock Server
	yumServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "repomd.xml"):
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			xml := `<?xml version="1.0" encoding="UTF-8"?><repomd xmlns="http://linux.duke.edu/metadata/repo"><revision>1640995200</revision><data type="primary"><location href="repodata/primary.xml.gz"/></data></repomd>`
			_, _ = w.Write([]byte(xml))
		case strings.Contains(r.URL.Path, "primary.xml.gz"):
			// 압축된 primary metadata (5MB 모의 데이터)
			w.Header().Set("Content-Type", "application/x-gzip")
			w.Header().Set("Content-Length", "5242880")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(make([]byte, 5242880)) // 5MB
		case strings.HasSuffix(r.URL.Path, ".rpm"):
			// RPM 패키지 (20MB 모의 데이터)
			w.Header().Set("Content-Type", "application/x-rpm")
			w.Header().Set("Content-Length", "20971520")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(make([]byte, 20971520)) // 20MB
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	env.MockUpstreams["yum"] = yumServer
	env.cleanup = append(env.cleanup, yumServer.Close)

	// APK Mock Server
	apkServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "APKINDEX.tar.gz"):
			// APK 인덱스 (2MB 모의 데이터)
			w.Header().Set("Content-Type", "application/x-gzip")
			w.Header().Set("Content-Length", "2097152")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(make([]byte, 2097152)) // 2MB
		case strings.HasSuffix(r.URL.Path, ".apk"):
			// APK 패키지 (15MB 모의 데이터)
			w.Header().Set("Content-Type", "application/vnd.alpine.apk")
			w.Header().Set("Content-Length", "15728640")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(make([]byte, 15728640)) // 15MB
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	env.MockUpstreams["apk"] = apkServer
	env.cleanup = append(env.cleanup, apkServer.Close)

	// APT Mock Server (기존과 동일하지만 최적화)
	aptServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "Release"):
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Origin: Ubuntu\nSuite: jammy\nComponents: main universe\nArchitectures: amd64 arm64\n"))
		case strings.Contains(r.URL.Path, "Packages"):
			// 큰 패키지 목록 (8MB 모의 데이터)
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Content-Length", "8388608")
			w.WriteHeader(http.StatusOK)
			packageData := strings.Repeat("Package: test-package\nVersion: 1.0.0\nArchitecture: amd64\nDescription: Test package\n\n", 16384)
			_, _ = w.Write([]byte(packageData[:8388608]))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	env.MockUpstreams["apt"] = aptServer
	env.cleanup = append(env.cleanup, aptServer.Close)
}

// setupHandlerProxyServer 핸들러별 ProxyND 서버 설정
func (env *HandlerBenchmarkEnvironment) setupHandlerProxyServer(b *testing.B) {
	appConfig := &app.Config{
		Port:        "0",
		Version:     "benchmark",
		BuildTime:   "benchmark",
		CommitSHA:   "benchmark",
		StorageDir:  b.TempDir(),
		ConfigDir:   b.TempDir(),
		CacheMaxAge: time.Hour,
	}

	container := app.NewContainer(appConfig)

	// 최적화된 Fiber 설정
	fiberApp := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		Prefork:               false,
		ServerHeader:          "",
		ReadTimeout:           60 * time.Second,
		WriteTimeout:          60 * time.Second,
		IdleTimeout:           120 * time.Second,
		ReadBufferSize:        8192,
		WriteBufferSize:       8192,
		BodyLimit:             500 * 1024 * 1024, // 500MB
		Concurrency:           512 * 1024,        // 매우 높은 동시성
	})

	// 최적화된 라우터 설정
	env.setupOptimizedHandlerRoutes(fiberApp, container, env.MockUpstreams)

	env.ProxyServer = fiberApp
	env.cleanup = append(env.cleanup, func() {
		if container != nil {
			_ = container.Close()
		}
	})
}

// setupOptimizedHandlerRoutes 최적화된 핸들러별 라우터 설정
func (env *HandlerBenchmarkEnvironment) setupOptimizedHandlerRoutes(app *fiber.App, container *app.Container, mockUpstreams map[string]*httptest.Server) {
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("container", container)
		return c.Next()
	})

	proxy := app.Group("/proxy")

	// 각 프록시 타입별 최적화된 핸들러
	for proxyType, upstream := range mockUpstreams {
		env.setupProxyHandler(proxy, proxyType, upstream)
	}

	// 헬스체크
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// 메트릭 엔드포인트 (간단한 구현)
	app.Get("/metrics", func(c *fiber.Ctx) error {
		return c.SendString("# HELP http_requests_total Total number of HTTP requests\n# TYPE http_requests_total counter\nhttp_requests_total 1000\n")
	})
}

// setupProxyHandler 개별 프록시 핸들러 설정
func (env *HandlerBenchmarkEnvironment) setupProxyHandler(proxy fiber.Router, proxyType string, upstream *httptest.Server) {
	proxy.All("/"+proxyType+"/*", func(c *fiber.Ctx) error {
		path := c.Params("*")

		// 최적화된 HTTP 클라이언트
		client := &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		}

		req, err := http.NewRequest(c.Method(), upstream.URL+"/"+path, bytes.NewReader(c.Body()))
		if err != nil {
			return c.Status(500).SendString("Internal Error")
		}

		// 필수 헤더만 복사
		if userAgent := c.Get("User-Agent"); userAgent != "" {
			req.Header.Set("User-Agent", userAgent)
		}
		if accept := c.Get("Accept"); accept != "" {
			req.Header.Set("Accept", accept)
		}

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

		// 스트리밍 응답
		_, err = io.Copy(c.Response().BodyWriter(), resp.Body)
		return err
	})
}

// Cleanup 벤치마크 환경 정리
func (env *HandlerBenchmarkEnvironment) Cleanup() {
	for i := len(env.cleanup) - 1; i >= 0; i-- {
		if env.cleanup[i] != nil {
			env.cleanup[i]()
		}
	}
}

// BenchmarkAllHandlers 모든 핸들러 성능 벤치마크
func BenchmarkAllHandlers(b *testing.B) {
	env := setupHandlerBenchmarkEnvironment(b)
	defer env.Cleanup()

	handlerTests := []struct {
		name     string
		path     string
		dataType string
		size     string
	}{
		// NPM 핸들러
		{"NPM-Metadata", "/proxy/npm/express", "JSON", "1KB"},
		{"NPM-Package", "/proxy/npm/express/-/express-4.18.2.tgz", "Binary", "2MB"},

		// Maven 핸들러
		{"Maven-POM", "/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.pom", "XML", "1KB"},
		{"Maven-JAR", "/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar", "Binary", "10MB"},

		// PIP 핸들러
		{"PIP-Metadata", "/proxy/pip/pypi/requests/json", "JSON", "1KB"},
		{"PIP-Simple", "/proxy/pip/simple/requests/", "HTML", "1KB"},
		{"PIP-Package", "/proxy/pip/simple/requests/requests-2.28.1-py3-none-any.whl", "Binary", "3MB"},

		// Docker 핸들러
		{"Docker-API", "/proxy/docker/v2/", "JSON", "1KB"},
		{"Docker-Manifest", "/proxy/docker/v2/library/nginx/manifests/latest", "JSON", "1KB"},
		{"Docker-Layer", "/proxy/docker/v2/library/nginx/blobs/sha256:layer-digest", "Binary", "50MB"},

		// YUM 핸들러
		{"YUM-Metadata", "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/repomd.xml", "XML", "1KB"},
		{"YUM-Primary", "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/primary.xml.gz", "Binary", "5MB"},
		{"YUM-Package", "/proxy/yum/centos/8/BaseOS/x86_64/os/Packages/nginx-1.20.1-1.el8.x86_64.rpm", "Binary", "20MB"},

		// APK 핸들러
		{"APK-Index", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz", "Binary", "2MB"},
		{"APK-Package", "/proxy/apk/alpine/v3.16/main/x86_64/nginx-1.22.0-r1.apk", "Binary", "15MB"},

		// APT 핸들러
		{"APT-Release", "/proxy/apt/ubuntu/dists/jammy/Release", "Text", "1KB"},
		{"APT-Packages", "/proxy/apt/ubuntu/dists/jammy/main/binary-amd64/Packages", "Text", "8MB"},
	}

	for _, test := range handlerTests {
		b.Run(test.name, func(b *testing.B) {
			// 첫 번째 요청으로 워밍업
			req, _ := http.NewRequest("GET", test.path, nil)
			resp, err := env.ProxyServer.Test(req, -1)
			if err == nil && resp != nil {
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
			}

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					req, err := http.NewRequest("GET", test.path, nil)
					require.NoError(b, err)

					resp, err := env.ProxyServer.Test(req, -1)
					require.NoError(b, err)

					bytesRead, err := io.Copy(io.Discard, resp.Body)
					require.NoError(b, err)
					_ = resp.Body.Close()

					// 처리량 보고
					b.SetBytes(bytesRead)
				}
			})
		})
	}
}

// BenchmarkHandlerThroughput 핸들러별 처리량 벤치마크
func BenchmarkHandlerThroughput(b *testing.B) {
	env := setupHandlerBenchmarkEnvironment(b)
	defer env.Cleanup()

	proxyTypes := []struct {
		name string
		path string
	}{
		{"NPM", "/proxy/npm/express"},
		{"Maven", "/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.pom"},
		{"PIP", "/proxy/pip/pypi/requests/json"},
		{"Docker", "/proxy/docker/v2/"},
		{"YUM", "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/repomd.xml"},
		{"APK", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz"},
		{"APT", "/proxy/apt/ubuntu/dists/jammy/Release"},
	}

	for _, proxyType := range proxyTypes {
		b.Run(proxyType.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				req, err := http.NewRequest("GET", proxyType.path, nil)
				require.NoError(b, err)

				resp, err := env.ProxyServer.Test(req, -1)
				require.NoError(b, err)

				_, err = io.Copy(io.Discard, resp.Body)
				require.NoError(b, err)
				_ = resp.Body.Close()
			}
		})
	}
}

// BenchmarkHandlerConcurrency 핸들러별 동시성 벤치마크
func BenchmarkHandlerConcurrency(b *testing.B) {
	env := setupHandlerBenchmarkEnvironment(b)
	defer env.Cleanup()

	concurrencyLevels := []int{1, 10, 50, 100}
	handlers := []struct {
		name string
		path string
	}{
		{"NPM", "/proxy/npm/express"},
		{"PIP", "/proxy/pip/pypi/requests/json"},
		{"Docker", "/proxy/docker/v2/"},
	}

	for _, handler := range handlers {
		for _, concurrency := range concurrencyLevels {
			testName := fmt.Sprintf("%s-Concurrency-%d", handler.name, concurrency)
			b.Run(testName, func(b *testing.B) {
				b.SetParallelism(concurrency)
				b.ResetTimer()

				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						req, err := http.NewRequest("GET", handler.path, nil)
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
}

// BenchmarkHandlerLatency 핸들러별 지연시간 벤치마크
func BenchmarkHandlerLatency(b *testing.B) {
	env := setupHandlerBenchmarkEnvironment(b)
	defer env.Cleanup()

	handlers := []struct {
		name string
		path string
	}{
		{"NPM", "/proxy/npm/express"},
		{"Maven", "/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.pom"},
		{"PIP", "/proxy/pip/pypi/requests/json"},
		{"Docker", "/proxy/docker/v2/"},
		{"YUM", "/proxy/yum/centos/8/BaseOS/x86_64/os/repodata/repomd.xml"},
		{"APK", "/proxy/apk/alpine/v3.16/main/x86_64/APKINDEX.tar.gz"},
		{"APT", "/proxy/apt/ubuntu/dists/jammy/Release"},
	}

	for _, handler := range handlers {
		b.Run(handler.name, func(b *testing.B) {
			latencies := make([]time.Duration, b.N)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				start := time.Now()

				req, err := http.NewRequest("GET", handler.path, nil)
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

// BenchmarkHandlerMemoryUsage 핸들러별 메모리 사용량 벤치마크
func BenchmarkHandlerMemoryUsage(b *testing.B) {
	env := setupHandlerBenchmarkEnvironment(b)
	defer env.Cleanup()

	handlers := []struct {
		name string
		path string
	}{
		{"NPM", "/proxy/npm/express"},
		{"PIP", "/proxy/pip/pypi/requests/json"},
		{"Docker", "/proxy/docker/v2/"},
	}

	for _, handler := range handlers {
		b.Run(handler.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				req, err := http.NewRequest("GET", handler.path, nil)
				require.NoError(b, err)

				resp, err := env.ProxyServer.Test(req, -1)
				require.NoError(b, err)

				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
			}
		})
	}
}
