package unit

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"proxynd/middlewares"
)

// TestIPFilterMiddleware IP 필터 미들웨어 테스트
func TestIPFilterMiddleware(t *testing.T) {
	app := fiber.New()

	t.Run("Allow Specific IP", func(t *testing.T) {
		config := middlewares.IPFilterConfig{
			AllowedIPs: []string{"127.0.0.1", "192.168.1.100"},
			DenyAll:    true,
		}

		app.Use(middlewares.IPFilterMiddleware(config))
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		// 허용된 IP에서 요청
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Real-IP", "127.0.0.1")
		
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		// 허용되지 않은 IP에서 요청
		req = httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Real-IP", "10.0.0.1")
		
		resp, err = app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode)
	})

	t.Run("Allow CIDR Range", func(t *testing.T) {
		app := fiber.New()
		
		config := middlewares.IPFilterConfig{
			AllowedCIDRs: []string{"192.168.1.0/24", "10.0.0.0/8"},
			DenyAll:      true,
		}

		app.Use(middlewares.IPFilterMiddleware(config))
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		// CIDR 범위 내 IP
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Real-IP", "192.168.1.50")
		
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		// CIDR 범위 외 IP
		req = httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Real-IP", "172.16.0.1")
		
		resp, err = app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode)
	})

	t.Run("Default Allow Mode", func(t *testing.T) {
		app := fiber.New()
		
		config := middlewares.IPFilterConfig{
			AllowedIPs: []string{"127.0.0.1"},
			DenyAll:    false, // 기본적으로 허용
		}

		app.Use(middlewares.IPFilterMiddleware(config))
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		// 허용되지 않은 IP도 통과해야 함
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Real-IP", "10.0.0.1")
		
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}

// TestBasicAuthMiddleware BasicAuth 미들웨어 테스트
func TestBasicAuthMiddleware(t *testing.T) {
	t.Run("Valid Credentials", func(t *testing.T) {
		app := fiber.New()
		
		config := middlewares.BasicAuthConfig{
			Users: map[string]string{
				"admin": "password123",
				"user":  "secret",
			},
			Realm: "Test Realm",
		}

		app.Use(middlewares.BasicAuthMiddleware(config))
		app.Get("/protected", func(c *fiber.Ctx) error {
			username := c.Locals("username").(string)
			return c.SendString("Hello " + username)
		})

		// 올바른 인증 정보
		credentials := base64.StdEncoding.EncodeToString([]byte("admin:password123"))
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Basic "+credentials)
		
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "Hello admin", string(body))
	})

	t.Run("Invalid Credentials", func(t *testing.T) {
		app := fiber.New()
		
		config := middlewares.BasicAuthConfig{
			Users: map[string]string{
				"admin": "password123",
			},
		}

		app.Use(middlewares.BasicAuthMiddleware(config))
		app.Get("/protected", func(c *fiber.Ctx) error {
			return c.SendString("Protected")
		})

		// 잘못된 비밀번호
		credentials := base64.StdEncoding.EncodeToString([]byte("admin:wrongpassword"))
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Basic "+credentials)
		
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("WWW-Authenticate"), "Basic realm=")
	})

	t.Run("No Authorization Header", func(t *testing.T) {
		app := fiber.New()
		
		config := middlewares.BasicAuthConfig{
			Users: map[string]string{"admin": "password123"},
		}

		app.Use(middlewares.BasicAuthMiddleware(config))
		app.Get("/protected", func(c *fiber.Ctx) error {
			return c.SendString("Protected")
		})

		req := httptest.NewRequest("GET", "/protected", nil)
		
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("Invalid Authorization Format", func(t *testing.T) {
		app := fiber.New()
		
		config := middlewares.BasicAuthConfig{
			Users: map[string]string{"admin": "password123"},
		}

		app.Use(middlewares.BasicAuthMiddleware(config))
		app.Get("/protected", func(c *fiber.Ctx) error {
			return c.SendString("Protected")
		})

		// Bearer 토큰 (Basic이 아님)
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer some-token")
		
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
	})
}

// TestPermissionMiddleware 권한 미들웨어 테스트
func TestPermissionMiddleware(t *testing.T) {
	t.Run("Read Permission", func(t *testing.T) {
		app := fiber.New()
		
		config := middlewares.PermissionConfig{
			UserPermissions: map[string]middlewares.Permission{
				"reader": middlewares.PermissionRead,
				"writer": middlewares.PermissionRead | middlewares.PermissionWrite,
			},
			DefaultPermission: 0, // 권한 없음
		}

		app.Use(middlewares.PermissionMiddleware(config))
		app.Get("/data", func(c *fiber.Ctx) error {
			return c.SendString("Data")
		})

		// 읽기 권한 있는 사용자
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("username", "reader")
			return c.Next()
		})

		req := httptest.NewRequest("GET", "/data", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("Write Permission Denied", func(t *testing.T) {
		app := fiber.New()
		
		config := middlewares.PermissionConfig{
			UserPermissions: map[string]middlewares.Permission{
				"reader": middlewares.PermissionRead,
			},
			DefaultPermission: 0,
		}

		// 사용자 설정
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("username", "reader")
			return c.Next()
		})

		app.Use(middlewares.PermissionMiddleware(config))
		app.Post("/data", func(c *fiber.Ctx) error {
			return c.SendString("Created")
		})

		req := httptest.NewRequest("POST", "/data", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode)
	})

	t.Run("Default Permission", func(t *testing.T) {
		app := fiber.New()
		
		config := middlewares.PermissionConfig{
			UserPermissions:   map[string]middlewares.Permission{},
			DefaultPermission: middlewares.PermissionRead,
		}

		app.Use(middlewares.PermissionMiddleware(config))
		app.Get("/public", func(c *fiber.Ctx) error {
			return c.SendString("Public")
		})

		// 인증되지 않은 사용자 (기본 권한 사용)
		req := httptest.NewRequest("GET", "/public", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}

// TestPackageFilterMiddleware 패키지 필터 미들웨어 테스트
func TestPackageFilterMiddleware(t *testing.T) {
	t.Run("Allowed Package", func(t *testing.T) {
		app := fiber.New()
		
		config := middlewares.PackageFilter{
			AllowedPackages: map[string][]string{
				"npm": {"express", "lodash", "@types/*"},
				"pip": {"requests", "numpy"},
			},
			DenyAll: true,
		}

		app.Use(middlewares.PackageFilterMiddleware(config))
		app.Get("/proxy/:type/*", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		// 허용된 패키지
		req := httptest.NewRequest("GET", "/proxy/npm/express", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		// 와일드카드 패턴 매칭
		req = httptest.NewRequest("GET", "/proxy/npm/@types/node", nil)
		resp, err = app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("Denied Package", func(t *testing.T) {
		app := fiber.New()
		
		config := middlewares.PackageFilter{
			AllowedPackages: map[string][]string{
				"npm": {"express"},
			},
			DenyAll: true,
		}

		app.Use(middlewares.PackageFilterMiddleware(config))
		app.Get("/proxy/:type/*", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		// 허용되지 않은 패키지
		req := httptest.NewRequest("GET", "/proxy/npm/lodash", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode)

		// 허용되지 않은 타입
		req = httptest.NewRequest("GET", "/proxy/maven/com.example", nil)
		resp, err = app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode)
	})

	t.Run("Default Allow Mode", func(t *testing.T) {
		app := fiber.New()
		
		config := middlewares.PackageFilter{
			AllowedPackages: map[string][]string{
				"npm": {"express"},
			},
			DenyAll: false, // 기본적으로 허용
		}

		app.Use(middlewares.PackageFilterMiddleware(config))
		app.Get("/proxy/:type/*", func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		// 목록에 없는 패키지도 허용되어야 함
		req := httptest.NewRequest("GET", "/proxy/npm/lodash", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}

// TestSecurityMiddleware 보안 미들웨어 테스트
func TestSecurityMiddleware(t *testing.T) {
	t.Run("Valid Hash", func(t *testing.T) {
		app := fiber.New()
		
		config := middlewares.SecurityConfig{
			EnableHashVerification: true,
			RequiredHashHeaders:    []string{"Content-SHA256"},
			FailOnHashMismatch:     true,
		}

		app.Use(middlewares.SecurityMiddleware(config))
		app.Post("/upload", func(c *fiber.Ctx) error {
			verified := c.Locals("hashVerified").(bool)
			if verified {
				return c.SendString("Hash verified")
			}
			return c.SendString("No hash verification")
		})

		// 올바른 해시가 포함된 요청
		data := []byte("test data")
		expectedHash := "916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9" // SHA256 of "test data"
		
		req := httptest.NewRequest("POST", "/upload", bytes.NewReader(data))
		req.Header.Set("Content-SHA256", expectedHash)
		
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("Invalid Hash", func(t *testing.T) {
		app := fiber.New()
		
		config := middlewares.SecurityConfig{
			EnableHashVerification: true,
			RequiredHashHeaders:    []string{"Content-SHA256"},
			FailOnHashMismatch:     true,
		}

		app.Use(middlewares.SecurityMiddleware(config))
		app.Post("/upload", func(c *fiber.Ctx) error {
			return c.SendString("Upload successful")
		})

		// 잘못된 해시가 포함된 요청
		data := []byte("test data")
		wrongHash := "invalid-hash"
		
		req := httptest.NewRequest("POST", "/upload", bytes.NewReader(data))
		req.Header.Set("Content-SHA256", wrongHash)
		
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})

	t.Run("No Hash Header", func(t *testing.T) {
		app := fiber.New()
		
		config := middlewares.SecurityConfig{
			EnableHashVerification: true,
			FailOnHashMismatch:     false, // 해시 없어도 통과
		}

		app.Use(middlewares.SecurityMiddleware(config))
		app.Post("/upload", func(c *fiber.Ctx) error {
			return c.SendString("Upload successful")
		})

		// 해시 헤더 없는 요청
		data := []byte("test data")
		req := httptest.NewRequest("POST", "/upload", bytes.NewReader(data))
		
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("Digest Header Format", func(t *testing.T) {
		app := fiber.New()
		
		config := middlewares.SecurityConfig{
			EnableHashVerification: true,
			RequiredHashHeaders:    []string{"Digest"},
			FailOnHashMismatch:     true,
		}

		app.Use(middlewares.SecurityMiddleware(config))
		app.Post("/upload", func(c *fiber.Ctx) error {
			return c.SendString("Upload successful")
		})

		// Digest 헤더 형식의 해시
		data := []byte("test data")
		expectedHash := "916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"
		
		req := httptest.NewRequest("POST", "/upload", bytes.NewReader(data))
		req.Header.Set("Digest", "SHA-256="+expectedHash)
		
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("GET Request Skip", func(t *testing.T) {
		app := fiber.New()
		
		config := middlewares.SecurityConfig{
			EnableHashVerification: true,
			FailOnHashMismatch:     true,
		}

		app.Use(middlewares.SecurityMiddleware(config))
		app.Get("/data", func(c *fiber.Ctx) error {
			return c.SendString("Data")
		})

		// GET 요청은 해시 검증을 건너뛰어야 함
		req := httptest.NewRequest("GET", "/data", nil)
		
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}