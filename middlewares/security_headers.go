package middlewares

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// SecurityHeadersConfig 보안 헤더 설정
type SecurityHeadersConfig struct {
	EnableXSSProtection      bool   `json:"enable_xss_protection"`
	EnableContentTypeNoSniff bool   `json:"enable_content_type_nosniff"`
	EnableFrameOptions       bool   `json:"enable_frame_options"`
	EnableHSTS               bool   `json:"enable_hsts"`
	HSTSMaxAge               int    `json:"hsts_max_age"`
	EnableCSP                bool   `json:"enable_csp"`
	CSPDirective             string `json:"csp_directive"`
	ReferrerPolicy           string `json:"referrer_policy"`
	PermissionsPolicy        string `json:"permissions_policy"`
	ServerHeader             string `json:"server_header"`
}

// DefaultSecurityHeadersConfig 기본 보안 헤더 설정
func DefaultSecurityHeadersConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		EnableXSSProtection:      true,
		EnableContentTypeNoSniff: true,
		EnableFrameOptions:       true,
		EnableHSTS:               false,    // HTTPS 환경에서만 활성화
		HSTSMaxAge:               31536000, // 1년
		EnableCSP:                true,
		CSPDirective:             "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self'",
		ReferrerPolicy:           "strict-origin-when-cross-origin",
		PermissionsPolicy:        "geolocation=(), microphone=(), camera=()",
		ServerHeader:             "ProxyND",
	}
}

// ProductionSecurityHeadersConfig 프로덕션용 보안 헤더 설정
func ProductionSecurityHeadersConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		EnableXSSProtection:      true,
		EnableContentTypeNoSniff: true,
		EnableFrameOptions:       true,
		EnableHSTS:               true,
		HSTSMaxAge:               31536000,
		EnableCSP:                true,
		CSPDirective:             "default-src 'self'; object-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; base-uri 'self'; form-action 'self'",
		ReferrerPolicy:           "strict-origin-when-cross-origin",
		PermissionsPolicy:        "geolocation=(), microphone=(), camera=(), payment=(), usb=(), magnetometer=(), gyroscope=(), accelerometer=()",
		ServerHeader:             "ProxyND",
	}
}

// SecurityHeaders 보안 헤더 미들웨어
func SecurityHeaders(config ...SecurityHeadersConfig) fiber.Handler {
	cfg := DefaultSecurityHeadersConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *fiber.Ctx) error {
		// XSS Protection
		if cfg.EnableXSSProtection {
			c.Set("X-XSS-Protection", "1; mode=block")
		}

		// Content Type Options (MIME sniffing 방지)
		if cfg.EnableContentTypeNoSniff {
			c.Set("X-Content-Type-Options", "nosniff")
		}

		// Frame Options (Clickjacking 방지)
		if cfg.EnableFrameOptions {
			c.Set("X-Frame-Options", "DENY")
		}

		// HSTS (HTTP Strict Transport Security)
		if cfg.EnableHSTS {
			hstsValue := fmt.Sprintf("max-age=%d; includeSubDomains", cfg.HSTSMaxAge)
			c.Set("Strict-Transport-Security", hstsValue)
		}

		// Content Security Policy
		if cfg.EnableCSP && cfg.CSPDirective != "" {
			c.Set("Content-Security-Policy", cfg.CSPDirective)
		}

		// Referrer Policy
		if cfg.ReferrerPolicy != "" {
			c.Set("Referrer-Policy", cfg.ReferrerPolicy)
		}

		// Permissions Policy (구 Feature-Policy)
		if cfg.PermissionsPolicy != "" {
			c.Set("Permissions-Policy", cfg.PermissionsPolicy)
		}

		// Server 헤더 (정보 노출 최소화)
		if cfg.ServerHeader != "" {
			c.Set("Server", cfg.ServerHeader)
		}

		// 추가 보안 헤더들
		c.Set("X-Permitted-Cross-Domain-Policies", "none")
		c.Set("Cross-Origin-Embedder-Policy", "require-corp")
		c.Set("Cross-Origin-Opener-Policy", "same-origin")
		c.Set("Cross-Origin-Resource-Policy", "same-origin")

		// Cache Control (민감한 엔드포인트용)
		path := c.Path()
		if isPrivateEndpoint(path) {
			c.Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
			c.Set("Pragma", "no-cache")
			c.Set("Expires", "0")
		}

		return c.Next()
	}
}

// isPrivateEndpoint 민감한 엔드포인트인지 확인
func isPrivateEndpoint(path string) bool {
	privateEndpoints := []string{
		"/api/auth/",
		"/api/user/",
		"/api/config/",
		"/api/cache/",
		"/auth/",
	}

	for _, endpoint := range privateEndpoints {
		if strings.HasPrefix(path, endpoint) {
			return true
		}
	}

	return false
}

// SecureHeaders 가장 기본적인 보안 헤더만 적용
func SecureHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 기본 보안 헤더만 적용
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Server", "ProxyND")

		return c.Next()
	}
}

// CORSSecurityHeaders CORS와 함께 사용할 보안 헤더
func CORSSecurityHeaders(allowedOrigins []string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		origin := c.Get("Origin")

		// Origin 검증 (와일드카드 지원)
		isAllowed := false
		for _, allowedOrigin := range allowedOrigins {
			if allowedOrigin == "*" || origin == allowedOrigin {
				isAllowed = true
				break
			}
		}

		// OPTIONS 요청 처리 (Preflight)
		if c.Method() == "OPTIONS" {
			if isAllowed {
				c.Set("Access-Control-Allow-Origin", origin)
			}
			c.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-API-Key")
			c.Set("Access-Control-Max-Age", "3600") // 1시간으로 단축
			return c.SendStatus(fiber.StatusNoContent)
		}

		if isAllowed {
			c.Set("Access-Control-Allow-Origin", origin)
		} else {
			c.Set("Access-Control-Allow-Origin", "null")
		}

		// CORS 보안 헤더
		c.Set("Access-Control-Allow-Credentials", "true")
		c.Set("Access-Control-Expose-Headers", "X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset")

		// 기본 보안 헤더
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "SAMEORIGIN") // CORS 환경에서는 SAMEORIGIN 사용
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")

		return c.Next()
	}
}

// APISecurityHeaders API 전용 보안 헤더 (더 엄격함)
func APISecurityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// API 전용 보안 헤더
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Referrer-Policy", "no-referrer")
		c.Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
		c.Set("Pragma", "no-cache")
		c.Set("Expires", "0")
		
		// API 응답에는 민감한 정보가 있을 수 있으므로
		c.Set("Cross-Origin-Embedder-Policy", "require-corp")
		c.Set("Cross-Origin-Opener-Policy", "same-origin")
		c.Set("Cross-Origin-Resource-Policy", "same-origin")
		
		// CSP를 더 엄격하게
		c.Set("Content-Security-Policy", "default-src 'none'; script-src 'none'; object-src 'none'")
		
		// 서버 정보 숨김
		c.Set("Server", "ProxyND-API")

		return c.Next()
	}
}

// PublicSecurityHeaders 공개 컨텐츠용 보안 헤더 (덜 엄격함)
func PublicSecurityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 공개 컨텐츠용 보안 헤더
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "SAMEORIGIN")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// 공개 캐시 허용
		c.Set("Cache-Control", "public, max-age=3600")
		
		// 기본 CSP (스크립트와 스타일 허용)
		c.Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
		
		c.Set("Server", "ProxyND")

		return c.Next()
	}
}
