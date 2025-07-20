package benchmark

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/stretchr/testify/require"

	"proxynd/logging"
)

// Constants for benchmark tests
const (
	httpMethodPOST = "POST"
)

// BenchmarkMiddlewareStack 미들웨어 스택 성능 벤치마크
func BenchmarkMiddlewareStack(b *testing.B) {
	testCases := []struct {
		name        string
		setupApp    func() *fiber.App
		description string
	}{
		{
			name: "NoMiddleware",
			setupApp: func() *fiber.App {
				app := fiber.New(fiber.Config{DisableStartupMessage: true})
				app.Get("/test", func(c *fiber.Ctx) error {
					return c.SendString("OK")
				})
				return app
			},
			description: "기본 핸들러만 사용",
		},
		{
			name: "BasicMiddleware",
			setupApp: func() *fiber.App {
				app := fiber.New(fiber.Config{DisableStartupMessage: true})
				app.Use(recover.New())
				app.Use(logger.New())
				app.Get("/test", func(c *fiber.Ctx) error {
					return c.SendString("OK")
				})
				return app
			},
			description: "기본 미들웨어 (Recovery + Logger)",
		},
		{
			name: "FullMiddleware",
			setupApp: func() *fiber.App {
				app := fiber.New(fiber.Config{DisableStartupMessage: true})

				// 모든 미들웨어 적용
				app.Use(recover.New())
				app.Use(logger.New(logger.Config{
					Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
				}))
				app.Use(cors.New(cors.Config{
					AllowOrigins: "*",
					AllowMethods: "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
				}))
				app.Use(compress.New(compress.Config{
					Level: compress.LevelBestSpeed,
				}))
				app.Use(limiter.New(limiter.Config{
					Max:        1000,
					Expiration: time.Minute,
				}))

				app.Get("/test", func(c *fiber.Ctx) error {
					return c.SendString("OK")
				})
				return app
			},
			description: "전체 미들웨어 스택",
		},
		{
			name: "ProxyNDMiddleware",
			setupApp: func() *fiber.App {
				app := fiber.New(fiber.Config{DisableStartupMessage: true})

				// ProxyND 실제 미들웨어 스택
				app.Use(logging.RequestLogger())
				app.Use(logging.New())
				app.Use(logging.ErrorLogger())
				app.Use(logging.RecoveryLogger())

				app.Get("/test", func(c *fiber.Ctx) error {
					return c.SendString("OK")
				})
				return app
			},
			description: "ProxyND 미들웨어 스택",
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			app := tc.setupApp()

			b.ResetTimer()
			b.ReportAllocs()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					req, err := http.NewRequest("GET", "/test", nil)
					require.NoError(b, err)

					resp, err := app.Test(req, -1)
					require.NoError(b, err)

					_, _ = io.Copy(io.Discard, resp.Body)
					_ = resp.Body.Close()
				}
			})
		})
	}
}

// BenchmarkLoggingMiddleware 로깅 미들웨어 성능 벤치마크
func BenchmarkLoggingMiddleware(b *testing.B) {
	loggerConfigs := []struct {
		name   string
		config logger.Config
	}{
		{
			name:   "DefaultLogger",
			config: logger.Config{},
		},
		{
			name: "CustomLogger",
			config: logger.Config{
				Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
			},
		},
		{
			name: "DetailedLogger",
			config: logger.Config{
				Format: "[${time}] ${pid} ${locals:requestid} ${status} - ${method} ${path} ${ip} ${latency} ${body}\n",
			},
		},
		{
			name: "MinimalLogger",
			config: logger.Config{
				Format: "${status} ${method} ${path}\n",
			},
		},
	}

	for _, lc := range loggerConfigs {
		b.Run(lc.name, func(b *testing.B) {
			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Use(logger.New(lc.config))
			app.Get("/test", func(c *fiber.Ctx) error {
				return c.SendString("OK")
			})

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				req, err := http.NewRequest("GET", "/test", nil)
				require.NoError(b, err)

				resp, err := app.Test(req, -1)
				require.NoError(b, err)

				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
			}
		})
	}
}

// BenchmarkCompressionMiddleware 압축 미들웨어 성능 벤치마크
func BenchmarkCompressionMiddleware(b *testing.B) {
	// 다양한 크기의 응답 데이터
	dataSizes := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
	}

	compressionLevels := []struct {
		name  string
		level compress.Level
	}{
		{"NoCompression", compress.LevelDisabled},
		{"BestSpeed", compress.LevelBestSpeed},
		{"Default", compress.LevelDefault},
		{"BestCompression", compress.LevelBestCompression},
	}

	for _, ds := range dataSizes {
		for _, cl := range compressionLevels {
			b.Run(fmt.Sprintf("%s_%s", ds.name, cl.name), func(b *testing.B) {
				app := fiber.New(fiber.Config{DisableStartupMessage: true})

				if cl.level != compress.LevelDisabled {
					app.Use(compress.New(compress.Config{
						Level: cl.level,
					}))
				}

				// 반복 가능한 텍스트 데이터 생성 (압축률 높음)
				testData := strings.Repeat("Hello, World! This is test data for compression benchmark. ", ds.size/60)
				if len(testData) < ds.size {
					testData += strings.Repeat("X", ds.size-len(testData))
				}
				testData = testData[:ds.size]

				app.Get("/test", func(c *fiber.Ctx) error {
					return c.SendString(testData)
				})

				b.SetBytes(int64(ds.size))
				b.ResetTimer()
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					req, err := http.NewRequest("GET", "/test", nil)
					req.Header.Set("Accept-Encoding", "gzip")
					require.NoError(b, err)

					resp, err := app.Test(req, -1)
					require.NoError(b, err)

					n, err := io.Copy(io.Discard, resp.Body)
					require.NoError(b, err)
					_ = resp.Body.Close()

					// 압축 효율성 체크
					if cl.level != compress.LevelDisabled {
						require.Less(b, n, int64(ds.size), "압축된 데이터가 원본보다 작아야 함")
					}
				}
			})
		}
	}
}

// BenchmarkCORSMiddleware CORS 미들웨어 성능 벤치마크
func BenchmarkCORSMiddleware(b *testing.B) {
	corsConfigs := []struct {
		name   string
		config cors.Config
	}{
		{
			name:   "DefaultCORS",
			config: cors.Config{},
		},
		{
			name: "AllowAll",
			config: cors.Config{
				AllowOrigins: "*",
				AllowMethods: "*",
				AllowHeaders: "*",
			},
		},
		{
			name: "RestrictiveOrigins",
			config: cors.Config{
				AllowOrigins: "https://example.com,https://api.example.com",
				AllowMethods: "GET,POST,PUT,DELETE",
				AllowHeaders: "Origin,Content-Type,Accept,Authorization",
			},
		},
		{
			name: "WithCredentials",
			config: cors.Config{
				AllowOrigins:     "https://example.com",
				AllowMethods:     "GET,POST,PUT,DELETE",
				AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
				AllowCredentials: true,
			},
		},
	}

	requestTypes := []struct {
		name    string
		method  string
		origin  string
		headers map[string]string
	}{
		{
			name:    "SimpleGET",
			method:  "GET",
			origin:  "https://example.com",
			headers: map[string]string{},
		},
		{
			name:    "SimpleGET-NoOrigin",
			method:  "GET",
			origin:  "",
			headers: map[string]string{},
		},
		{
			name:   "PreflightOPTIONS",
			method: "OPTIONS",
			origin: "https://example.com",
			headers: map[string]string{
				"Access-Control-Request-Method":  "PUT",
				"Access-Control-Request-Headers": "Content-Type,Authorization",
			},
		},
		{
			name:   "CrossOrigin",
			method: httpMethodPOST,
			origin: "https://malicious.com",
			headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
	}

	for _, cc := range corsConfigs {
		for _, rt := range requestTypes {
			b.Run(fmt.Sprintf("%s_%s", cc.name, rt.name), func(b *testing.B) {
				app := fiber.New(fiber.Config{DisableStartupMessage: true})
				app.Use(cors.New(cc.config))
				app.All("/test", func(c *fiber.Ctx) error {
					return c.SendString("OK")
				})

				b.ResetTimer()
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					req, err := http.NewRequest(rt.method, "/test", nil)
					require.NoError(b, err)

					if rt.origin != "" {
						req.Header.Set("Origin", rt.origin)
					}

					for key, value := range rt.headers {
						req.Header.Set(key, value)
					}

					resp, err := app.Test(req, -1)
					require.NoError(b, err)

					_, _ = io.Copy(io.Discard, resp.Body)
					_ = resp.Body.Close()
				}
			})
		}
	}
}

// BenchmarkRateLimitingMiddleware 속도 제한 미들웨어 성능 벤치마크
func BenchmarkRateLimitingMiddleware(b *testing.B) {
	limitConfigs := []struct {
		name   string
		config limiter.Config
	}{
		{
			name: "HighLimit",
			config: limiter.Config{
				Max:        10000,
				Expiration: time.Minute,
			},
		},
		{
			name: "MediumLimit",
			config: limiter.Config{
				Max:        1000,
				Expiration: time.Minute,
			},
		},
		{
			name: "LowLimit",
			config: limiter.Config{
				Max:        100,
				Expiration: time.Minute,
			},
		},
		{
			name: "VeryLowLimit",
			config: limiter.Config{
				Max:        10,
				Expiration: time.Minute,
			},
		},
	}

	for _, lc := range limitConfigs {
		b.Run(lc.name, func(b *testing.B) {
			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Use(limiter.New(lc.config))
			app.Get("/test", func(c *fiber.Ctx) error {
				return c.SendString("OK")
			})

			b.ResetTimer()
			b.ReportAllocs()

			successCount := 0
			limitedCount := 0

			for i := 0; i < b.N; i++ {
				req, err := http.NewRequest("GET", "/test", nil)
				require.NoError(b, err)

				resp, err := app.Test(req, -1)
				require.NoError(b, err)

				switch resp.StatusCode {
				case 200:
					successCount++
				case 429:
					limitedCount++
				}

				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
			}

			// 결과 리포트
			b.ReportMetric(float64(successCount), "success")
			b.ReportMetric(float64(limitedCount), "limited")
		})
	}
}

// BenchmarkRecoveryMiddleware 복구 미들웨어 성능 벤치마크
func BenchmarkRecoveryMiddleware(b *testing.B) {
	panicTypes := []struct {
		name    string
		handler func(c *fiber.Ctx) error
	}{
		{
			name: "NoPanic",
			handler: func(c *fiber.Ctx) error {
				return c.SendString("OK")
			},
		},
		{
			name: "StringPanic",
			handler: func(_ *fiber.Ctx) error {
				panic("test panic")
			},
		},
		{
			name: "ErrorPanic",
			handler: func(_ *fiber.Ctx) error {
				panic(fmt.Errorf("test error panic"))
			},
		},
		{
			name: "NilPanic",
			handler: func(c *fiber.Ctx) error {
				var ptr *string
				return c.SendString(*ptr) // nil pointer panic
			},
		},
	}

	for _, pt := range panicTypes {
		b.Run(pt.name, func(b *testing.B) {
			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Use(recover.New(recover.Config{
				EnableStackTrace: true,
			}))
			app.Get("/test", pt.handler)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				req, err := http.NewRequest("GET", "/test", nil)
				require.NoError(b, err)

				resp, err := app.Test(req, -1)
				require.NoError(b, err)

				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
			}
		})
	}
}

// BenchmarkCustomMiddleware 커스텀 미들웨어 성능 벤치마크
func BenchmarkCustomMiddleware(b *testing.B) {
	middlewareTypes := []struct {
		name       string
		middleware func(*fiber.Ctx) error
	}{
		{
			name: "NoOp",
			middleware: func(c *fiber.Ctx) error {
				return c.Next()
			},
		},
		{
			name: "SimpleHeader",
			middleware: func(c *fiber.Ctx) error {
				c.Set("X-Custom-Header", "benchmark")
				return c.Next()
			},
		},
		{
			name: "RequestID",
			middleware: func(c *fiber.Ctx) error {
				requestID := fmt.Sprintf("req-%d", time.Now().UnixNano())
				c.Locals("requestID", requestID)
				c.Set("X-Request-ID", requestID)
				return c.Next()
			},
		},
		{
			name: "TimingHeader",
			middleware: func(c *fiber.Ctx) error {
				start := time.Now()
				err := c.Next()
				duration := time.Since(start)
				c.Set("X-Response-Time", duration.String())
				return err
			},
		},
		{
			name: "BodySizeCheck",
			middleware: func(c *fiber.Ctx) error {
				if len(c.Body()) > 1024*1024 { // 1MB limit
					return c.Status(413).SendString("Request too large")
				}
				return c.Next()
			},
		},
		{
			name: "ContentTypeValidation",
			middleware: func(c *fiber.Ctx) error {
				contentType := c.Get("Content-Type")
				if c.Method() == httpMethodPOST && contentType == "" {
					return c.Status(400).SendString("Content-Type required")
				}
				return c.Next()
			},
		},
	}

	for _, mt := range middlewareTypes {
		b.Run(mt.name, func(b *testing.B) {
			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Use(mt.middleware)
			app.Get("/test", func(c *fiber.Ctx) error {
				return c.SendString("OK")
			})
			app.Post("/test", func(c *fiber.Ctx) error {
				return c.SendString("OK")
			})

			b.ResetTimer()
			b.ReportAllocs()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					// GET과 POST 요청을 번갈아 테스트
					method := "GET"
					body := io.Reader(nil)
					if pb.Next() {
						method = httpMethodPOST
						body = bytes.NewReader([]byte(`{"test": "data"}`))
					}

					req, err := http.NewRequest(method, "/test", body)
					require.NoError(b, err)

					if method == httpMethodPOST {
						req.Header.Set("Content-Type", "application/json")
					}

					resp, err := app.Test(req, -1)
					require.NoError(b, err)

					_, _ = io.Copy(io.Discard, resp.Body)
					_ = resp.Body.Close()
				}
			})
		})
	}
}

// BenchmarkMiddlewareMemoryUsage 미들웨어 메모리 사용량 벤치마크
func BenchmarkMiddlewareMemoryUsage(b *testing.B) {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})

	// 메모리 집약적인 미들웨어들
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${pid} ${locals:requestid} ${status} - ${method} ${path} ${ip} ${latency} ${body}\n",
	}))
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestCompression,
	}))
	app.Use(func(c *fiber.Ctx) error {
		// 큰 로컬 데이터 시뮬레이션
		c.Locals("large_data", make([]byte, 1024)) // 1KB
		return c.Next()
	})

	// 큰 응답 데이터
	largeResponse := strings.Repeat("Large response data for memory benchmark. ", 1000) // ~40KB

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString(largeResponse)
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req, err := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		require.NoError(b, err)

		resp, err := app.Test(req, -1)
		require.NoError(b, err)

		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
}

// BenchmarkMiddlewareChainDepth 미들웨어 체인 깊이 성능 테스트
func BenchmarkMiddlewareChainDepth(b *testing.B) {
	chainDepths := []int{1, 5, 10, 20, 50, 100}

	for _, depth := range chainDepths {
		b.Run(fmt.Sprintf("Depth-%d", depth), func(b *testing.B) {
			app := fiber.New(fiber.Config{DisableStartupMessage: true})

			// 지정된 깊이만큼 미들웨어 체인 생성
			for i := 0; i < depth; i++ {
				app.Use(func(c *fiber.Ctx) error {
					c.Set(fmt.Sprintf("X-Middleware-%d", i), "processed")
					return c.Next()
				})
			}

			app.Get("/test", func(c *fiber.Ctx) error {
				return c.SendString("OK")
			})

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				req, err := http.NewRequest("GET", "/test", nil)
				require.NoError(b, err)

				resp, err := app.Test(req, -1)
				require.NoError(b, err)

				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
			}
		})
	}
}
