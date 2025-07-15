package middlewares

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

// TestInputValidation 입력 검증 미들웨어 테스트
func TestInputValidation(t *testing.T) {
	app := fiber.New()
	app.Use(InputValidation())
	app.Get("/test/:package", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})
	app.Get("/proxy/:type/*", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	tests := []struct {
		name           string
		url            string
		userAgent      string
		expectedStatus int
		description    string
	}{
		{
			name:           "Valid package name",
			url:            "/test/my-package",
			userAgent:      "npm/6.14.4 node/v14.15.0",
			expectedStatus: 200,
			description:    "Normal package name should pass",
		},
		{
			name:           "Valid proxy request",
			url:            "/proxy/npm/express/4.17.1",
			userAgent:      "npm/6.14.4 node/v14.15.0",
			expectedStatus: 200,
			description:    "Normal proxy request should pass",
		},
		{
			name:           "XSS attempt in package name",
			url:            "/test/<script>alert(1)</script>",
			userAgent:      "Mozilla/5.0",
			expectedStatus: 400,
			description:    "XSS attempt should be blocked",
		},
		{
			name:           "Path traversal attempt",
			url:            "/test/../../../etc/passwd",
			userAgent:      "curl/7.68.0",
			expectedStatus: 400,
			description:    "Path traversal should be blocked",
		},
		{
			name:           "Long package name",
			url:            "/test/" + strings.Repeat("a", 300),
			userAgent:      "test-client/1.0",
			expectedStatus: 400,
			description:    "Excessive long package name should be blocked",
		},
		{
			name:           "Invalid proxy type",
			url:            "/proxy/invalid-type/package",
			userAgent:      "curl/7.68.0",
			expectedStatus: 400,
			description:    "Invalid proxy type should be blocked",
		},
		{
			name:           "Missing user agent",
			url:            "/test/valid-package",
			userAgent:      "",
			expectedStatus: 400,
			description:    "Missing User-Agent should be blocked",
		},
		{
			name:           "Short user agent",
			url:            "/test/valid-package",
			userAgent:      "a",
			expectedStatus: 400,
			description:    "Too short User-Agent should be blocked",
		},
		{
			name:           "Malicious user agent",
			url:            "/test/valid-package",
			userAgent:      "<script>alert('xss')</script>",
			expectedStatus: 400,
			description:    "Malicious User-Agent should be blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			if tt.userAgent != "" {
				req.Header.Set("User-Agent", tt.userAgent)
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to perform request: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("%s: expected status %d, got %d",
					tt.description, tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

// TestInputValidationWithConfig 설정이 있는 입력 검증 테스트
func TestInputValidationWithConfig(t *testing.T) {
	app := fiber.New()

	// 엄격한 설정 적용
	strictConfig := ValidationConfig{
		MaxFileSize:     1024, // 1KB
		MaxQueryLength:  50,
		MaxParamLength:  20,
		AllowedFileExts: []string{".jar"},
		EnableLogging:   false, // 테스트에서는 로깅 비활성화
	}

	app.Use(InputValidation(strictConfig))
	app.Get("/test/:package", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})
	app.Post("/upload", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	tests := []struct {
		name           string
		method         string
		url            string
		contentLength  int64
		headers        map[string]string
		expectedStatus int
	}{
		{
			name:           "Valid request within limits",
			method:         "GET",
			url:            "/test/pkg",
			expectedStatus: 200,
		},
		{
			name:           "Package name too long",
			method:         "GET",
			url:            "/test/" + strings.Repeat("a", 25),
			expectedStatus: 400,
		},
		{
			name:           "File upload too large",
			method:         "POST",
			url:            "/upload",
			contentLength:  2048, // 2KB > 1KB limit
			headers:        map[string]string{"X-Filename": "test.jar"},
			expectedStatus: 413,
		},
		{
			name:           "Invalid file extension",
			method:         "POST",
			url:            "/upload",
			headers:        map[string]string{"X-Filename": "test.exe"},
			expectedStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Header.Set("User-Agent", "test-client/1.0")

			if tt.contentLength > 0 {
				req.ContentLength = tt.contentLength
			}

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to perform request: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

// TestRateLimit Rate Limiting 미들웨어 테스트
func TestRateLimit(t *testing.T) {
	app := fiber.New()

	// 테스트용 낮은 제한 설정
	rateLimitConfig := RateLimitConfig{
		RequestsPerWindow: 3,
		WindowSize:        time.Second * 5,
		EnableLogging:     false, // 테스트에서는 로깅 비활성화
	}

	app.Use(RateLimit(rateLimitConfig))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// 제한 내 요청들은 성공해야 함
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "test-client/1.0")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i+1, err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("Request %d: expected status 200, got %d", i+1, resp.StatusCode)
		}
	}

	// 4번째 요청은 실패해야 함
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-client/1.0")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Rate limit test request failed: %v", err)
	}
	if resp.StatusCode != 429 {
		t.Errorf("Expected rate limit status 429, got %d", resp.StatusCode)
	}

	// Rate limit 헤더 확인
	if resp.Header.Get("Retry-After") == "" {
		t.Error("Expected Retry-After header to be set")
	}
}

// TestRateLimitDifferentIPs 다른 IP에서의 Rate Limiting 테스트
func TestRateLimitDifferentIPs(t *testing.T) {
	app := fiber.New()

	rateLimitConfig := RateLimitConfig{
		RequestsPerWindow: 2,
		WindowSize:        time.Second * 5,
		EnableLogging:     false,
	}

	app.Use(RateLimit(rateLimitConfig))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// IP1에서 2번 요청 (제한에 도달)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "test-client/1.0")
		req.Header.Set("X-Forwarded-For", "192.168.1.1")
		resp, _ := app.Test(req)
		if resp.StatusCode != 200 {
			t.Errorf("IP1 request %d should succeed", i+1)
		}
	}

	// IP1에서 3번째 요청 (실패해야 함)
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.Header.Set("User-Agent", "test-client/1.0")
	req1.Header.Set("X-Forwarded-For", "192.168.1.1")
	resp1, _ := app.Test(req1)
	if resp1.StatusCode != 429 {
		t.Error("IP1 should be rate limited")
	}

	// IP2에서 요청 (성공해야 함)
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("User-Agent", "test-client/1.0")
	req2.Header.Set("X-Forwarded-For", "192.168.1.2")
	resp2, _ := app.Test(req2)
	if resp2.StatusCode != 200 {
		t.Error("IP2 should not be rate limited")
	}
}

// TestSecurityHeaders 보안 헤더 미들웨어 테스트
func TestSecurityHeaders(t *testing.T) {
	app := fiber.New()
	app.Use(SecurityHeaders())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})
	app.Get("/api/auth/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	tests := []struct {
		name            string
		url             string
		expectedHeaders map[string]string
	}{
		{
			name: "Basic security headers",
			url:  "/test",
			expectedHeaders: map[string]string{
				"X-XSS-Protection":       "1; mode=block",
				"X-Content-Type-Options": "nosniff",
				"X-Frame-Options":        "DENY",
				"Server":                 "ProxyND",
				"Referrer-Policy":        "strict-origin-when-cross-origin",
			},
		},
		{
			name: "Private endpoint with cache headers",
			url:  "/api/auth/test",
			expectedHeaders: map[string]string{
				"X-XSS-Protection":       "1; mode=block",
				"X-Content-Type-Options": "nosniff",
				"X-Frame-Options":        "DENY",
				"Cache-Control":          "no-store, no-cache, must-revalidate, private",
				"Pragma":                 "no-cache",
				"Expires":                "0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			req.Header.Set("User-Agent", "test-client/1.0")
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to perform request: %v", err)
			}

			if resp.StatusCode != 200 {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}

			for expectedHeader, expectedValue := range tt.expectedHeaders {
				actualValue := resp.Header.Get(expectedHeader)
				if actualValue != expectedValue {
					t.Errorf("Header %s: expected '%s', got '%s'",
						expectedHeader, expectedValue, actualValue)
				}
			}
		})
	}
}

// TestProductionSecurityHeaders 프로덕션 보안 헤더 테스트
func TestProductionSecurityHeaders(t *testing.T) {
	app := fiber.New()
	app.Use(SecurityHeaders(ProductionSecurityHeadersConfig()))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-client/1.0")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}

	// HSTS 헤더 확인 (프로덕션 설정에서는 활성화)
	hsts := resp.Header.Get("Strict-Transport-Security")
	if hsts == "" {
		t.Error("Expected HSTS header to be set in production config")
	}

	// 더 엄격한 CSP 확인
	csp := resp.Header.Get("Content-Security-Policy")
	if !strings.Contains(csp, "object-src 'none'") {
		t.Error("Expected stricter CSP in production config")
	}
}

// TestBurstRateLimit 버스트 Rate Limiting 테스트
func TestBurstRateLimit(t *testing.T) {
	app := fiber.New()
	app.Use(BurstRateLimit(5, 3, time.Second*10)) // normal: 5/10s, burst: 3/1s
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// 빠른 연속 요청 (버스트 제한 테스트)
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "test-client/1.0")
		resp, _ := app.Test(req)
		if resp.StatusCode != 200 {
			t.Errorf("Burst request %d should succeed", i+1)
		}
	}

	// 4번째 빠른 요청은 버스트 제한에 걸려야 함
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-client/1.0")
	resp, _ := app.Test(req)
	if resp.StatusCode != 429 {
		t.Error("Should hit burst rate limit")
	}
}

// BenchmarkInputValidation 입력 검증 성능 테스트
func BenchmarkInputValidation(b *testing.B) {
	app := fiber.New()
	app.Use(InputValidation())
	app.Get("/test/:package", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test/valid-package-name", nil)
	req.Header.Set("User-Agent", "test-client/1.0")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = app.Test(req)
	}
}

// BenchmarkRateLimit Rate Limiting 성능 테스트
func BenchmarkRateLimit(b *testing.B) {
	app := fiber.New()
	app.Use(RateLimit())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "test-client/1.0")
		req.Header.Set("X-Forwarded-For", fmt.Sprintf("192.168.1.%d", i%254+1))
		_, _ = app.Test(req)
	}
}
