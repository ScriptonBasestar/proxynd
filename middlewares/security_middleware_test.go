package middlewares

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityMiddlewareConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   SecurityMiddlewareConfig
		expected bool
	}{
		{
			name:     "Default config",
			config:   DefaultSecurityMiddlewareConfig(),
			expected: true,
		},
		{
			name:     "Production config",
			config:   ProductionSecurityMiddlewareConfig(),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config.EnableRateLimit)
			assert.Equal(t, tt.expected, tt.config.EnableInputValidation)
			assert.Equal(t, tt.expected, tt.config.EnableSecurityHeaders)
		})
	}
}

func TestEnhancedInputValidation(t *testing.T) {
	app := fiber.New()
	
	// 테스트용 미들웨어 설정
	app.Use(EnhancedInputValidation())
	
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		headers        map[string]string
		expectedStatus int
	}{
		{
			name:           "Valid request",
			method:         "POST",
			path:           "/test",
			body:           `{"name": "test", "value": "data"}`,
			headers:        map[string]string{"Content-Type": "application/json"},
			expectedStatus: 200,
		},
		{
			name:           "SQL injection in body",
			method:         "POST",
			path:           "/test",
			body:           `{"query": "SELECT * FROM users WHERE id = 1 UNION SELECT password FROM admin"}`,
			headers:        map[string]string{"Content-Type": "application/json"},
			expectedStatus: 400,
		},
		{
			name:           "XSS in body",
			method:         "POST",
			path:           "/test",
			body:           `{"content": "<script>alert('xss')</script>"}`,
			headers:        map[string]string{"Content-Type": "application/json"},
			expectedStatus: 400,
		},
		{
			name:           "Invalid Content-Type",
			method:         "POST",
			path:           "/test",
			body:           `test data`,
			headers:        map[string]string{"Content-Type": "text/html"},
			expectedStatus: 400,
		},
		{
			name:           "Long header value",
			method:         "POST",
			path:           "/test",
			body:           `{"test": "data"}`,
			headers: map[string]string{
				"Content-Type": "application/json",
				"X-Custom":     strings.Repeat("a", 2500), // 너무 긴 헤더
			},
			expectedStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestIPValidation(t *testing.T) {
	app := fiber.New()
	
	// IP 필터링 미들웨어 설정
	blockedCIDRs := []string{"192.168.1.0/24", "10.0.0.1"}
	allowedCIDRs := []string{"127.0.0.1", "::1"}
	
	app.Use(IPValidation(blockedCIDRs, allowedCIDRs))
	
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	tests := []struct {
		name           string
		remoteAddr     string
		expectedStatus int
	}{
		{
			name:           "Allowed IP - localhost",
			remoteAddr:     "127.0.0.1:12345",
			expectedStatus: 200,
		},
		{
			name:           "Blocked IP in CIDR",
			remoteAddr:     "192.168.1.100:12345",
			expectedStatus: 403,
		},
		{
			name:           "Blocked specific IP",
			remoteAddr:     "10.0.0.1:12345",
			expectedStatus: 403,
		},
		{
			name:           "Non-allowed IP",
			remoteAddr:     "8.8.8.8:12345",
			expectedStatus: 403,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestEnhancedRateLimiterIntegration(t *testing.T) {
	app := fiber.New()
	
	// 테스트용 Rate Limiter 설정 (매우 낮은 제한)
	config := EnhancedRateLimitConfig{
		Rate:  "2-M", // 분당 2개 요청
		Burst: 1,
		KeyGenerator: func(_ *fiber.Ctx) string {
			return "test-key" // 모든 요청에 같은 키 사용
		},
		EnableLogging: false, // 테스트 시 로깅 비활성화
	}
	
	app.Use(NewEnhancedRateLimiter(config))
	
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// 첫 번째 요청 - 성공해야 함
	req1 := httptest.NewRequest("GET", "/test", nil)
	resp1, err := app.Test(req1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp1.StatusCode)

	// 두 번째 요청 - 성공해야 함
	req2 := httptest.NewRequest("GET", "/test", nil)
	resp2, err := app.Test(req2)
	require.NoError(t, err)
	assert.Equal(t, 200, resp2.StatusCode)

	// 세 번째 요청 - Rate limit 초과로 실패해야 함
	req3 := httptest.NewRequest("GET", "/test", nil)
	resp3, err := app.Test(req3)
	require.NoError(t, err)
	assert.Equal(t, 429, resp3.StatusCode)
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	app := fiber.New()
	
	app.Use(SecurityHeaders())
	
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	
	assert.Equal(t, 200, resp.StatusCode)
	
	// 보안 헤더 확인
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", resp.Header.Get("X-XSS-Protection"))
	assert.Equal(t, "strict-origin-when-cross-origin", resp.Header.Get("Referrer-Policy"))
	assert.Equal(t, "ProxyND", resp.Header.Get("Server"))
}

func TestCORSSecurityHeaders(t *testing.T) {
	app := fiber.New()
	
	allowedOrigins := []string{"https://example.com", "https://test.com"}
	app.Use(CORSSecurityHeaders(allowedOrigins))
	
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	tests := []struct {
		name           string
		origin         string
		method         string
		expectedOrigin string
		expectedStatus int
	}{
		{
			name:           "Allowed origin",
			origin:         "https://example.com",
			method:         "GET",
			expectedOrigin: "https://example.com",
			expectedStatus: 200,
		},
		{
			name:           "Non-allowed origin",
			origin:         "https://malicious.com",
			method:         "GET",
			expectedOrigin: "null",
			expectedStatus: 200,
		},
		{
			name:           "OPTIONS preflight request",
			origin:         "https://example.com",
			method:         "OPTIONS",
			expectedOrigin: "https://example.com",
			expectedStatus: 204,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/test", nil)
			req.Header.Set("Origin", tt.origin)
			
			resp, err := app.Test(req)
			require.NoError(t, err)
			
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			assert.Equal(t, tt.expectedOrigin, resp.Header.Get("Access-Control-Allow-Origin"))
		})
	}
}

func TestSetupSecurityMiddlewares(t *testing.T) {
	app := fiber.New()
	
	config := DefaultSecurityMiddlewareConfig()
	config.EnableSecurityLogging = false // 테스트 시 로깅 비활성화
	
	// 보안 미들웨어 설정
	SetupSecurityMiddlewares(app, config)
	
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	
	assert.Equal(t, 200, resp.StatusCode)
	
	// Rate limit 헤더 확인
	assert.NotEmpty(t, resp.Header.Get("X-RateLimit-Limit"))
	assert.NotEmpty(t, resp.Header.Get("X-RateLimit-Remaining"))
	
	// 보안 헤더 확인
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
}

func TestAPISecurityMiddlewares(t *testing.T) {
	app := fiber.New()
	
	config := DefaultSecurityMiddlewareConfig()
	config.EnableSecurityLogging = false
	
	// API 그룹 생성
	apiGroup := app.Group("/api")
	SetupAPISecurityMiddlewares(apiGroup, config)
	
	apiGroup.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	
	assert.Equal(t, 200, resp.StatusCode)
	
	// API 전용 보안 헤더 확인
	assert.Equal(t, "ProxyND-API", resp.Header.Get("Server"))
	assert.Equal(t, "no-store, no-cache, must-revalidate, private", resp.Header.Get("Cache-Control"))
}

func BenchmarkSecurityMiddlewares(b *testing.B) {
	app := fiber.New()
	
	config := DefaultSecurityMiddlewareConfig()
	config.EnableSecurityLogging = false
	
	SetupSecurityMiddlewares(app, config)
	
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := app.Test(req)
			if err != nil {
				b.Fatal(err)
			}
			if resp.StatusCode != 200 {
				b.Fatalf("Expected status 200, got %d", resp.StatusCode)
			}
			_ = resp.Body.Close()
		}
	})
}

func TestValidateContentType(t *testing.T) {
	app := fiber.New()
	
	app.Use(EnhancedInputValidation())
	
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	tests := []struct {
		name           string
		contentType    string
		body           []byte
		expectedStatus int
	}{
		{
			name:           "Valid JSON content type",
			contentType:    "application/json",
			body:           []byte(`{"test": "data"}`),
			expectedStatus: 200,
		},
		{
			name:           "Valid XML content type",
			contentType:    "application/xml",
			body:           []byte(`<root><test>data</test></root>`),
			expectedStatus: 200,
		},
		{
			name:           "Valid form content type",
			contentType:    "application/x-www-form-urlencoded",
			body:           []byte(`name=test&value=data`),
			expectedStatus: 200,
		},
		{
			name:           "Invalid content type",
			contentType:    "text/html",
			body:           []byte(`<html><body>test</body></html>`),
			expectedStatus: 400,
		},
		{
			name:           "Empty body with invalid content type",
			contentType:    "text/html",
			body:           []byte{},
			expectedStatus: 200, // 빈 바디는 검증하지 않음
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", bytes.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}