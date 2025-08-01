package middlewares

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// JWTConfig JWT 인증 설정
type JWTConfig struct {
	SecretKey     string        `json:"secretKey" yaml:"secret_key"`
	TokenDuration time.Duration `json:"tokenDuration" yaml:"token_duration"`
	Issuer        string        `json:"issuer" yaml:"issuer"`
	SkipPaths     []string      `json:"skipPaths" yaml:"skip_paths"` // 인증을 건너뛸 경로들
}

// JWTClaims JWT 토큰 클레임
type JWTClaims struct {
	UserID    string   `json:"user_id"`
	Username  string   `json:"username"`
	Roles     []string `json:"roles"`
	IssuedAt  int64    `json:"iat"`
	ExpiresAt int64    `json:"exp"`
	jwt.RegisteredClaims
}

// JWTMiddleware JWT 인증 미들웨어 (간단한 버전)
func JWTMiddleware(config JWTConfig) fiber.Handler {
	// 기본값 설정
	if config.TokenDuration == 0 {
		config.TokenDuration = 24 * time.Hour
	}
	if config.Issuer == "" {
		config.Issuer = "proxynd"
	}
	if config.SecretKey == "" {
		// 개발 환경에서만 기본 키 사용 (운영에서는 에러)
		config.SecretKey = "dev-secret-key-change-this-in-production"
	}

	return func(c *fiber.Ctx) error {
		// Skip paths 확인
		path := c.Path()
		for _, skipPath := range config.SkipPaths {
			if strings.HasPrefix(path, skipPath) {
				return c.Next()
			}
		}

		// Authorization 헤더에서 토큰 추출
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing authorization header",
				"code":  "AUTH_MISSING_HEADER",
			})
		}

		// Bearer 토큰 형식 확인
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authorization header format",
				"code":  "AUTH_INVALID_FORMAT",
			})
		}

		tokenString := parts[1]

		// JWT 토큰 파싱 및 검증
		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			// 서명 방법 확인
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(config.SecretKey), nil
		})
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token",
				"code":  "AUTH_INVALID_TOKEN",
			})
		}

		// 클레임 정보 확인
		if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
			// 토큰 만료 확인
			if time.Unix(claims.ExpiresAt, 0).Before(time.Now()) {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Token expired",
					"code":  "AUTH_TOKEN_EXPIRED",
				})
			}

			// 사용자 정보를 컨텍스트에 저장
			c.Locals("user_id", claims.UserID)
			c.Locals("username", claims.Username)
			c.Locals("roles", claims.Roles)
			c.Locals("jwt_claims", claims)

			return c.Next()
		}

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid token claims",
			"code":  "AUTH_INVALID_CLAIMS",
		})
	}
}

// GenerateJWTToken JWT 토큰 생성
func GenerateJWTToken(userID, username string, roles []string, config JWTConfig) (string, error) {
	now := time.Now()
	expiresAt := now.Add(config.TokenDuration)

	claims := JWTClaims{
		UserID:    userID,
		Username:  username,
		Roles:     roles,
		IssuedAt:  now.Unix(),
		ExpiresAt: expiresAt.Unix(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    config.Issuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.SecretKey))
}

// ValidateJWTToken JWT 토큰 검증 (미들웨어 외부에서 사용)
func ValidateJWTToken(tokenString string, config JWTConfig) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(config.SecretKey), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		// 토큰 만료 확인
		if time.Unix(claims.ExpiresAt, 0).Before(time.Now()) {
			return nil, fmt.Errorf("token expired")
		}
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// GetUserFromContext 컨텍스트에서 사용자 정보 추출
func GetUserFromContext(c *fiber.Ctx) (userID, username string, roles []string, ok bool) {
	userIDVal := c.Locals("user_id")
	usernameVal := c.Locals("username")
	rolesVal := c.Locals("roles")

	if userIDVal == nil || usernameVal == nil {
		return "", "", nil, false
	}

	userID, userIDOk := userIDVal.(string)
	username, usernameOk := usernameVal.(string)
	roles, rolesOk := rolesVal.([]string)

	if !userIDOk || !usernameOk || !rolesOk {
		return "", "", nil, false
	}

	return userID, username, roles, true
}

// RequireRole 특정 역할을 요구하는 미들웨어
func RequireRole(requiredRole string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, _, roles, ok := GetUserFromContext(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required",
				"code":  "AUTH_REQUIRED",
			})
		}

		// 역할 확인
		hasRole := false
		for _, role := range roles {
			if role == requiredRole || role == "admin" { // admin은 모든 권한 보유
				hasRole = true
				break
			}
		}

		if !hasRole {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Insufficient permissions",
				"code":  "AUTH_INSUFFICIENT_PERMISSIONS",
			})
		}

		return c.Next()
	}
}

// isAuthenticated 기존 미들웨어와의 호환성을 위한 간단한 인증 확인
func isAuthenticated(c *fiber.Ctx) bool {
	_, _, _, ok := GetUserFromContext(c)
	return ok
}
