package middlewares

import (
	"crypto/subtle"
	"encoding/base64"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// BasicAuthConfig BasicAuth 설정
type BasicAuthConfig struct {
	Users map[string]string // username -> password 매핑
	Realm string            // 인증 영역 이름
}

// BasicAuthMiddleware BasicAuth 미들웨어
func BasicAuthMiddleware(config BasicAuthConfig) fiber.Handler {
	if config.Realm == "" {
		config.Realm = "Restricted"
	}

	return func(c *fiber.Ctx) error {
		// Authorization 헤더 확인
		auth := c.Get("Authorization")
		if auth == "" {
			return unauthorized(c, config.Realm)
		}

		// Basic 인증 방식 확인
		if !strings.HasPrefix(auth, "Basic ") {
			return unauthorized(c, config.Realm)
		}

		// Base64 디코딩
		payload := auth[6:] // "Basic " 제거
		decoded, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return unauthorized(c, config.Realm)
		}

		// username:password 분리
		credentials := string(decoded)
		parts := strings.SplitN(credentials, ":", 2)
		if len(parts) != 2 {
			return unauthorized(c, config.Realm)
		}

		username, password := parts[0], parts[1]

		// 사용자 인증 확인
		expectedPassword, exists := config.Users[username]
		if !exists {
			return unauthorized(c, config.Realm)
		}

		// 타이밍 공격 방지를 위한 constant time 비교
		if subtle.ConstantTimeCompare([]byte(password), []byte(expectedPassword)) != 1 {
			return unauthorized(c, config.Realm)
		}

		// 인증 성공 - 사용자 정보를 컨텍스트에 저장
		c.Locals("username", username)
		return c.Next()
	}
}

// unauthorized 인증 실패 응답
func unauthorized(c *fiber.Ctx, realm string) error {
	c.Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
	return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
}

// UserMiddlewares function to add auth
func UserMiddlewares() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Code for middlewares
		return c.Next()
	}
}

// ProxyMiddleware exported function ProxyMiddleware should have comment or be unexported
func ProxyMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Code for middlewares
		return c.Next()
	}
}
