// Package auth provides authentication handlers and middleware
package auth

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/auth/jwt"
	"proxynd/internal/config"
	"proxynd/internal/logging"
)

// RequireAuth 인증 필수 미들웨어 (세션 기반)
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

// RequireJWTAuth JWT 토큰 기반 인증 미들웨어
func RequireJWTAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Authorization 헤더에서 Bearer 토큰 추출
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			// 헤더가 없으면 세션에서 JWT 토큰 확인
			return requireJWTFromSession(c)
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authorization header format",
			})
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		return validateJWTToken(c, token)
	}
}

// requireJWTFromSession 세션에서 JWT 토큰 확인
func requireJWTFromSession(c *fiber.Ctx) error {
	sess, err := getSession(c)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get session",
		})
	}

	jwtAccessToken, exists := sess["jwt_access_token"]
	if !exists || jwtAccessToken == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	tokenStr, ok := jwtAccessToken.(string)
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid token type",
		})
	}
	return validateJWTToken(c, tokenStr)
}

// validateJWTToken JWT 토큰 검증
func validateJWTToken(c *fiber.Ctx, token string) error {
	// OAuth2 설정 로드
	oauth2Config := &config.OAuth2Config{}
	if err := oauth2Config.ReadConfig(); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Authentication configuration not available",
		})
	}

	// JWT 서비스 생성
	jwtService := jwt.NewJWTService(oauth2Config)

	// 토큰 검증
	claims, err := jwtService.ValidateAccessToken(token)
	if err != nil {
		logging.GetLogger().Warn("JWT token validation failed",
			logging.F("error", err),
			logging.F("ip", c.IP()))
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired token",
		})
	}

	// 토큰이 곧 만료되는지 확인
	if jwtService.IsTokenExpiringSoon(claims) {
		c.Set("X-Token-Expires-Soon", "true")
	}

	// 사용자 정보를 컨텍스트에 저장
	userInfo := jwtService.ExtractUserInfo(claims)
	c.Locals("user", userInfo)
	c.Locals("jwt_claims", claims)

	return c.Next()
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

		email, ok := userMap["email"].(string)
		if !ok {
			email = "unknown"
		}
		logging.GetLogger().Warn("Access denied",
			logging.F("email", email),
			logging.F("role", userRole),
			logging.F("path", c.Path()))

		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error":          "Insufficient privileges",
			"required_roles": requiredRoles,
			"user_role":      userRole,
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
				delete(sess, "jwt_access_token")
				delete(sess, "jwt_refresh_token")

				if email, ok := userMap["email"].(string); ok {
					logging.GetLogger().Info("Session expired", logging.F("email", email))
				}

				return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
					"error": "Session expired",
				})
			}
		}

		// JWT 토큰 만료 시간 확인
		if expiresAt, ok := userMap["expires_at"].(time.Time); ok {
			// 토큰이 10분 이내 만료될 경우 헤더 설정
			timeToExpiry := time.Until(expiresAt)
			if timeToExpiry < 10*time.Minute && timeToExpiry > 0 {
				c.Set("X-Token-Expires-Soon", "true")
				c.Set("X-Token-Expires-In", fmt.Sprintf("%d", int(timeToExpiry.Seconds())))
			}

			if time.Now().After(expiresAt) {
				// 토큰 만료, 자동 갱신 시도
				if jwtRefreshToken, exists := sess["jwt_refresh_token"]; exists && jwtRefreshToken != nil {
					// 토큰 갱신 로직
					if err := autoRefreshToken(c, sess, userMap); err != nil {
						if email, ok := userMap["email"].(string); ok {
							logging.GetLogger().Warn("Auto token refresh failed", logging.F("email", email), logging.F("error", err))
						}

						// 갱신 실패 시 세션 정리
						delete(sess, "user")
						delete(sess, "jwt_access_token")
						delete(sess, "jwt_refresh_token")

						return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
							"error":      "Token expired and refresh failed",
							"error_code": "token_refresh_failed",
							"action":     "login_required",
						})
					}

					// 갱신 성공 시 응답 헤더 설정
					c.Set("X-Token-Refreshed", "true")
				} else {
					return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
						"error":      "Token expired",
						"error_code": "token_expired",
						"action":     "login_required",
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
		logger := logging.GetLogger()
		logger.Info("BasicAuthFallback middleware called",
			logging.String("path", c.Path()))

		// OAuth2 인증이 실패한 경우 BasicAuth 시도
		userData := c.Locals("user")
		if userData != nil {
			// 이미 OAuth2로 인증됨
			logger.Info("User already authenticated via OAuth2")
			return c.Next()
		}

		// 인증 설정 확인
		globalConfig := &config.GlobalConfig{}
		if err := globalConfig.ReadConfig(); err != nil {
			// 설정 로드 실패 시 인증 없이 통과 (개발 모드)
			logger.Info("Failed to load global config - skipping auth", logging.ErrorField(err))
			return c.Next()
		}
		logger.Info("Global config loaded successfully")

		// 전체 인증이 비활성화된 경우 인증 없이 통과
		if globalConfig.Authentication == nil {
			logger.Info("Authentication config is nil - skipping auth")
			return c.Next()
		}

		if globalConfig.Authentication.Enabled != nil && !*globalConfig.Authentication.Enabled {
			logger.Info("Authentication is globally disabled - skipping auth")
			return c.Next()
		}

		logger.Info("Authentication is enabled",
			logging.Bool("global_enabled", globalConfig.Authentication.Enabled == nil || *globalConfig.Authentication.Enabled))

		// BasicAuth가 설정되어 있지 않거나 비활성화된 경우 인증 없이 통과
		if globalConfig.Authentication.BasicAuth == nil {
			logger.Info("BasicAuth config is nil - skipping auth")
			return c.Next()
		}

		if globalConfig.Authentication.BasicAuth.Enabled != nil && !*globalConfig.Authentication.BasicAuth.Enabled {
			logger.Info("BasicAuth is disabled - skipping auth")
			return c.Next()
		}

		logger.Info("BasicAuth is enabled - requiring authentication")

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

// autoRefreshToken 자동 JWT 토큰 갱신 (실시간 권한 동기화 포함)
func autoRefreshToken(_ *fiber.Ctx, sess, userMap fiber.Map) error {
	jwtRefreshToken, exists := sess["jwt_refresh_token"]
	if !exists || jwtRefreshToken == nil {
		return errors.New("no JWT refresh token available")
	}

	// OAuth2 설정 로드
	oauth2Config := &config.OAuth2Config{}
	if err := oauth2Config.ReadConfig(); err != nil {
		return err
	}

	// JWT 서비스 생성
	jwtService := jwt.NewJWTService(oauth2Config)

	// OAuth2 토큰이 있는 경우 권한 동기화를 위해 사용
	oauth2AccessToken, ok := userMap["oauth2_access_token"].(string)
	if !ok {
		oauth2AccessToken = ""
	}

	// 실시간 권한 동기화를 위해 OAuth2 제공자 정보 조회
	if providerName, providerOk := userMap["provider"].(string); providerOk && oauth2AccessToken != "" {
		// 제공자별 인터페이스 구현체 생성 (실제 구현은 추후 추가)
		_ = createOAuth2Provider(providerName, oauth2Config)
	}

	// JWT 토큰 갱신 (권한 동기화 포함)
	var newTokenPair *jwt.TokenPair
	var err error

	// 현재는 기존 권한 유지 (실시간 동기화는 추후 구현)
	refreshToken, ok := jwtRefreshToken.(string)
	if !ok {
		return errors.New("invalid refresh token type")
	}
	newTokenPair, err = jwtService.RefreshAccessToken(refreshToken)
	if err != nil {
		return errors.New("failed to refresh JWT token: " + err.Error())
	}

	// 새 JWT 토큰으로 사용자 정보 추출
	newAccessClaims, err := jwtService.ValidateAccessToken(newTokenPair.AccessToken)
	if err != nil {
		return errors.New("failed to validate new access token: " + err.Error())
	}

	// 세션 업데이트
	newSessionData := jwtService.ExtractUserInfo(newAccessClaims)
	// 기존 정보 보존 (avatar, login_time, oauth2 토큰 등)
	if avatar, ok := userMap["avatar"]; ok {
		newSessionData["avatar"] = avatar
	}
	if loginTime, ok := userMap["login_time"]; ok {
		newSessionData["login_time"] = loginTime
	}
	if oauth2AccessToken, ok := userMap["oauth2_access_token"]; ok {
		newSessionData["oauth2_access_token"] = oauth2AccessToken
	}
	if oauth2RefreshToken, ok := userMap["oauth2_refresh_token"]; ok {
		newSessionData["oauth2_refresh_token"] = oauth2RefreshToken
	}

	sess["user"] = newSessionData
	sess["jwt_access_token"] = newTokenPair.AccessToken
	sess["jwt_refresh_token"] = newTokenPair.RefreshToken

	return nil
}

// createOAuth2Provider OAuth2 제공자별 인터페이스 구현체 생성
func createOAuth2Provider(_ string, _ *config.OAuth2Config) interface{} {
	// 실제 구현에서는 각 제공자별로 인터페이스를 구현하여 반환
	// 현재는 nil 반환 (추후 구현)
	return nil
}

// handleBasicAuth BasicAuth 처리
func handleBasicAuth(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Basic ") {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid Basic Auth format",
		})
	}

	// Base64 디코딩
	payload := authHeader[6:] // "Basic " 제거
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid Base64 encoding",
		})
	}

	// username:password 분리
	credentials := string(decoded)
	parts := strings.SplitN(credentials, ":", 2)
	if len(parts) != 2 {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid credentials format",
		})
	}

	username := parts[0]
	password := parts[1]

	// BasicAuth 사용자 인증 (설정에서 로드)
	if err := validateBasicAuthUser(username, password); err != nil {
		logging.GetLogger().Warn("BasicAuth failed",
			logging.F("username", username),
			logging.F("ip", c.IP()),
			logging.F("error", err))
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid credentials",
		})
	}

	// 인증 성공 - 사용자 정보를 컨텍스트에 저장 (OAuth2와 호환되는 형태)
	userInfo := fiber.Map{
		"email":      username,
		"username":   username,
		"role":       "viewer", // BasicAuth 사용자는 기본적으로 viewer 역할
		"auth_type":  "basic",
		"login_time": time.Now(),
	}

	c.Locals("user", userInfo)
	return c.Next()
}

// handleBearerToken Bearer 토큰 처리
func handleBearerToken(c *fiber.Ctx, token string) error {
	return validateJWTToken(c, token)
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

// validateBasicAuthUser BasicAuth 사용자 검증
func validateBasicAuthUser(username, password string) error {
	// 글로벌 설정에서 BasicAuth 사용자 정보 로드
	globalConfig := &config.GlobalConfig{}
	if err := globalConfig.ReadConfig(); err != nil {
		return errors.New("failed to load configuration")
	}

	// 전체 인증 설정 확인
	if globalConfig.Authentication == nil {
		return errors.New("authentication not configured")
	}

	// 전체 인증이 비활성화된 경우
	if globalConfig.Authentication.Enabled != nil && !*globalConfig.Authentication.Enabled {
		return errors.New("authentication is disabled")
	}

	// BasicAuth 설정이 있는지 확인
	if globalConfig.Authentication.BasicAuth == nil {
		return errors.New("BasicAuth not configured")
	}

	// BasicAuth가 비활성화된 경우 인증 불가
	if globalConfig.Authentication.BasicAuth.Enabled != nil && !*globalConfig.Authentication.BasicAuth.Enabled {
		return errors.New("BasicAuth is disabled")
	}

	// 사용자 존재 확인
	expectedPassword, exists := globalConfig.Authentication.BasicAuth.Users[username]
	if !exists {
		return errors.New("user not found")
	}

	// 타이밍 공격 방지를 위한 constant time 비교
	if subtle.ConstantTimeCompare([]byte(password), []byte(expectedPassword)) != 1 {
		return errors.New("invalid password")
	}

	return nil
}
