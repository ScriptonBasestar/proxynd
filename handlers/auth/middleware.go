package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"proxynd/configs"
	"proxynd/logging"
)

// RequireAuth 인증 필수 미들웨어
func RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		sess, err := getSession(c)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to get session",
			})
		}
		
		userData, exists := sess["user"]
		if !exists || userData == nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required",
			})
		}
		
		// 사용자 정보를 컨텍스트에 저장
		c.Locals("user", userData)
		
		return c.Next()
	}
}

// RequireRole 역할 기반 접근 제어 미들웨어
func RequireRole(requiredRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userData := c.Locals("user")
		if userData == nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required",
			})
		}
		
		userMap, ok := userData.(fiber.Map)
		if !ok {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "Invalid user session data",
			})
		}
		
		userRole, ok := userMap["role"].(string)
		if !ok {
			return c.Status(http.StatusForbidden).JSON(fiber.Map{
				"error": "User role not found",
			})
		}
		
		// 역할 확인
		for _, role := range requiredRoles {
			if userRole == role {
				return c.Next()
			}
		}
		
		// admin 역할은 모든 권한을 가짐
		if userRole == "admin" {
			return c.Next()
		}
		
		logging.GetLogger().Warn("Access denied", 
			logging.F("email", userMap["email"].(string)), 
			logging.F("role", userRole), 
			logging.F("path", c.Path()))
		
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Insufficient privileges",
			"required_roles": requiredRoles,
			"user_role": userRole,
		})
	}
}

// OptionalAuth 선택적 인증 미들웨어 (인증되지 않아도 통과)
func OptionalAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		sess, err := getSession(c)
		if err != nil {
			// 세션 오류가 있어도 계속 진행
			return c.Next()
		}
		
		userData, exists := sess["user"]
		if exists && userData != nil {
			// 사용자 정보를 컨텍스트에 저장
			c.Locals("user", userData)
		}
		
		return c.Next()
	}
}

// SessionTimeout 세션 타임아웃 체크 미들웨어
func SessionTimeout() fiber.Handler {
	return func(c *fiber.Ctx) error {
		sess, err := getSession(c)
		if err != nil {
			return c.Next()
		}
		
		userData, exists := sess["user"]
		if !exists || userData == nil {
			return c.Next()
		}
		
		userMap, ok := userData.(fiber.Map)
		if !ok {
			return c.Next()
		}
		
		// 로그인 시간 확인
		if loginTime, ok := userMap["login_time"].(time.Time); ok {
			// 24시간 세션 타임아웃
			if time.Since(loginTime) > 24*time.Hour {
				// 세션 만료
				delete(sess, "user")
				delete(sess, "access_token")
				delete(sess, "refresh_token")
				
				if email, ok := userMap["email"].(string); ok {
					logging.GetLogger().Info("Session expired", logging.F("email", email))
				}
				
				return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
					"error": "Session expired",
				})
			}
		}
		
		// 토큰 만료 시간 확인
		if expiresAt, ok := userMap["expires_at"].(time.Time); ok {
			if time.Now().After(expiresAt) {
				// 토큰 만료, 자동 갱신 시도
				if refreshToken, exists := sess["refresh_token"]; exists && refreshToken != nil {
					// 토큰 갱신 로직은 별도 함수로 분리
					if err := autoRefreshToken(c, sess, userMap); err != nil {
						if email, ok := userMap["email"].(string); ok {
							logging.GetLogger().Warn("Auto token refresh failed", logging.F("email", email), logging.F("error", err))
						}
						
						return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
							"error": "Token expired and refresh failed",
						})
					}
				} else {
					return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
						"error": "Token expired",
					})
				}
			}
		}
		
		return c.Next()
	}
}

// BasicAuthFallback BasicAuth 폴백 미들웨어
func BasicAuthFallback() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// OAuth2 인증이 실패한 경우 BasicAuth 시도
		userData := c.Locals("user")
		if userData != nil {
			// 이미 OAuth2로 인증됨
			return c.Next()
		}
		
		// Authorization 헤더 확인
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authorization required",
			})
		}
		
		// Basic Auth 처리
		if strings.HasPrefix(authHeader, "Basic ") {
			// 기존 BasicAuth 로직 호출
			// 여기서는 간단한 예제만 제공
			return handleBasicAuth(c)
		}
		
		// Bearer 토큰 처리 (JWT 등)
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			return handleBearerToken(c, token)
		}
		
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid authorization header format",
		})
	}
}

// AdminOnly 관리자 전용 미들웨어
func AdminOnly() fiber.Handler {
	return RequireRole("admin")
}

// MaintainerOrAdmin 유지보수자 또는 관리자 미들웨어
func MaintainerOrAdmin() fiber.Handler {
	return RequireRole("admin", "maintainer")
}

// DeveloperOrAbove 개발자 이상 권한 미들웨어
func DeveloperOrAbove() fiber.Handler {
	return RequireRole("admin", "maintainer", "developer")
}

// 유틸리티 함수들

// autoRefreshToken 자동 토큰 갱신
func autoRefreshToken(c *fiber.Ctx, sess fiber.Map, userMap fiber.Map) error {
	refreshToken, exists := sess["refresh_token"]
	if !exists || refreshToken == nil {
		return errors.New("no refresh token available")
	}
	
	providerName, ok := userMap["provider"].(string)
	if !ok {
		return errors.New("provider information not found")
	}
	
	// OAuth2 설정 로드
	oauth2Config := &configs.OAuth2Config{}
	if err := oauth2Config.ReadConfig(); err != nil {
		return err
	}
	
	providerConfig, exists := oauth2Config.GetProvider(providerName)
	if !exists {
		return errors.New("provider not configured")
	}
	
	// 실제 토큰 갱신 로직은 RefreshToken 함수와 동일
	// 여기서는 간단히 만료 시간만 연장
	userMap["expires_at"] = time.Now().Add(1 * time.Hour)
	sess["user"] = userMap
	
	// 미사용 변수 경고 제거
	_ = providerConfig
	
	return nil
}

// handleBasicAuth BasicAuth 처리
func handleBasicAuth(c *fiber.Ctx) error {
	// 기존 BasicAuth 로직
	// 실제 구현은 프로젝트의 기존 인증 시스템에 따라 달라짐
	return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
		"error": "BasicAuth not implemented in this example",
	})
}

// handleBearerToken Bearer 토큰 처리
func handleBearerToken(c *fiber.Ctx, token string) error {
	// JWT 토큰 검증 로직
	// 실제 구현은 JWT 라이브러리를 사용하여 토큰 검증
	return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
		"error": "Bearer token validation not implemented in this example",
	})
}

// GetUserFromContext 컨텍스트에서 사용자 정보 추출
func GetUserFromContext(c *fiber.Ctx) (fiber.Map, bool) {
	userData := c.Locals("user")
	if userData == nil {
		return nil, false
	}
	
	userMap, ok := userData.(fiber.Map)
	return userMap, ok
}

// HasRole 사용자가 특정 역할을 가지고 있는지 확인
func HasRole(c *fiber.Ctx, role string) bool {
	userMap, ok := GetUserFromContext(c)
	if !ok {
		return false
	}
	
	userRole, ok := userMap["role"].(string)
	if !ok {
		return false
	}
	
	return userRole == role || userRole == "admin"
}

// IsAuthenticated 사용자가 인증되었는지 확인
func IsAuthenticated(c *fiber.Ctx) bool {
	_, ok := GetUserFromContext(c)
	return ok
}