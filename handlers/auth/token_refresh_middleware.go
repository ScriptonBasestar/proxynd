package auth

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/config"
	"proxynd/internal/auth/jwt"
	"proxynd/logging"
)

// RefreshRequest 토큰 갱신 요청 추적
type RefreshRequest struct {
	UserID     string
	Timestamp  time.Time
	InProgress bool
}

// TokenRefreshManager 토큰 갱신 관리자 (동시성 제어)
type TokenRefreshManager struct {
	requests map[string]*RefreshRequest
	mutex    sync.RWMutex
}

// 전역 토큰 갱신 관리자
var refreshManager = &TokenRefreshManager{
	requests: make(map[string]*RefreshRequest),
}

// EnhancedSessionTimeout 개선된 세션 타임아웃 미들웨어
func EnhancedSessionTimeout() fiber.Handler {
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

		// 1. 세션 타임아웃 체크 (24시간)
		if loginTime, ok := userMap["login_time"].(time.Time); ok {
			if time.Since(loginTime) > 24*time.Hour {
				clearSession(sess, userMap)
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error":          "Session expired",
					"error_code":     "SESSION_EXPIRED",
					"login_required": true,
				})
			}
		}

		// 2. JWT 토큰 만료 체크
		expiresAt, hasExpiresAt := userMap["expires_at"].(time.Time)
		if !hasExpiresAt {
			// expires_at가 없으면 JWT 토큰에서 추출
			if jwtToken, exists := sess["jwt_access_token"].(string); exists {
				expiresAt, err = extractTokenExpiration(jwtToken)
				if err != nil {
					logging.GetLogger().Warn("Failed to extract token expiration", logging.F("error", err))
					return c.Next()
				}
				// 캐시를 위해 세션에 저장
				userMap["expires_at"] = expiresAt
				sess["user"] = userMap
			} else {
				return c.Next()
			}
		}

		timeUntilExpiry := time.Until(expiresAt)

		// 3. 토큰이 만료되었거나 곧 만료될 예정인지 확인
		if timeUntilExpiry <= 0 {
			// 토큰이 이미 만료됨 - 자동 갱신 시도
			if err := attemptTokenRefresh(c, sess, userMap); err != nil {
				clearSession(sess, userMap)
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error":            "Token expired and refresh failed",
					"error_code":       "TOKEN_EXPIRED",
					"refresh_endpoint": "/auth/refresh",
					"login_required":   true,
				})
			}
		} else if timeUntilExpiry < 10*time.Minute {
			// 토큰이 곧 만료됨 - 클라이언트에 알림
			c.Set("X-Token-Expires-Soon", "true")
			c.Set("X-Token-Expires-In", timeUntilExpiry.String())
			c.Set("X-Refresh-Endpoint", "/auth/refresh")
		}

		return c.Next()
	}
}

// attemptTokenRefresh 토큰 갱신 시도 (동시성 제어 포함)
func attemptTokenRefresh(c *fiber.Ctx, sess, userMap fiber.Map) error {
	userID, ok := userMap["user_id"].(string)
	if !ok {
		return fiber.NewError(fiber.StatusInternalServerError, "Invalid user ID")
	}

	// 동시성 제어 - 같은 사용자의 중복 갱신 요청 방지
	refreshManager.mutex.Lock()

	if req, exists := refreshManager.requests[userID]; exists {
		if req.InProgress && time.Since(req.Timestamp) < 30*time.Second {
			// 이미 갱신 진행 중이고 30초 이내라면 대기
			refreshManager.mutex.Unlock()
			time.Sleep(100 * time.Millisecond)
			return nil
		}
	}

	// 갱신 요청 등록
	refreshManager.requests[userID] = &RefreshRequest{
		UserID:     userID,
		Timestamp:  time.Now(),
		InProgress: true,
	}
	refreshManager.mutex.Unlock()

	// 갱신 완료 후 정리
	defer func() {
		refreshManager.mutex.Lock()
		delete(refreshManager.requests, userID)
		refreshManager.mutex.Unlock()
	}()

	// 실제 토큰 갱신 수행
	return performTokenRefresh(c, sess, userMap)
}

// performTokenRefresh 실제 토큰 갱신 수행
func performTokenRefresh(_ *fiber.Ctx, sess, userMap fiber.Map) error {
	jwtRefreshToken, exists := sess["jwt_refresh_token"]
	if !exists || jwtRefreshToken == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "No refresh token available")
	}

	// OAuth2 설정 로드
	oauth2Config := &config.OAuth2Config{}
	if err := oauth2Config.ReadConfig(); err != nil {
		return err
	}

	// JWT 서비스 생성
	jwtService := jwt.NewJWTService(oauth2Config)

	// 권한 동기화를 위한 OAuth2 토큰 확인
	oauth2AccessToken, ok := userMap["oauth2_access_token"].(string)
	if !ok {
		oauth2AccessToken = ""
	}
	providerName, ok := userMap["provider"].(string)
	if !ok {
		providerName = ""
	}

	var newTokenPair *jwt.TokenPair
	var err error

	// OAuth2 토큰이 있고 권한 동기화가 필요한 경우
	if oauth2AccessToken != "" && providerName != "" {
		// 실시간 권한 동기화 포함 갱신 (구현 필요)
		refreshTokenStr, refreshOk := jwtRefreshToken.(string)
		if !refreshOk {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid refresh token type")
		}
		newTokenPair, err = jwtService.RefreshAccessToken(refreshTokenStr)
	} else {
		// 기본 갱신 (기존 권한 유지)
		refreshTokenStr, refreshOk := jwtRefreshToken.(string)
		if !refreshOk {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid refresh token type")
		}
		newTokenPair, err = jwtService.RefreshAccessToken(refreshTokenStr)
	}

	if err != nil {
		return err
	}

	// 새 JWT 토큰으로 사용자 정보 추출
	newAccessClaims, err := jwtService.ValidateAccessToken(newTokenPair.AccessToken)
	if err != nil {
		return err
	}

	// 세션 업데이트
	newSessionData := jwtService.ExtractUserInfo(newAccessClaims)

	// 기존 정보 보존
	preservedFields := []string{"avatar", "login_time", "oauth2_access_token", "oauth2_refresh_token"}
	for _, field := range preservedFields {
		if value, exists := userMap[field]; exists {
			newSessionData[field] = value
		}
	}

	sess["user"] = newSessionData
	sess["jwt_access_token"] = newTokenPair.AccessToken
	sess["jwt_refresh_token"] = newTokenPair.RefreshToken

	// 성공 로그
	email, ok := newSessionData["email"].(string)
	if !ok {
		email = "unknown"
	}
	logging.GetLogger().Info("Token auto-refreshed",
		logging.F("email", email),
		logging.F("user_id", newAccessClaims.UserID),
		logging.F("expires_at", newTokenPair.ExpiresAt))

	return nil
}

// extractTokenExpiration JWT 토큰에서 만료 시간 추출
func extractTokenExpiration(tokenString string) (time.Time, error) {
	oauth2Config := &config.OAuth2Config{}
	if err := oauth2Config.ReadConfig(); err != nil {
		return time.Time{}, err
	}

	jwtService := jwt.NewJWTService(oauth2Config)
	claims, err := jwtService.ValidateAccessToken(tokenString)
	if err != nil {
		return time.Time{}, err
	}

	return claims.ExpiresAt.Time, nil
}

// clearSession 세션 정리
func clearSession(sess, userMap fiber.Map) {
	if email, ok := userMap["email"].(string); ok {
		logging.GetLogger().Info("Session cleared", logging.F("email", email))
	}

	// 세션 데이터 삭제
	keysToDelete := []string{
		"user", "jwt_access_token", "jwt_refresh_token",
		"oauth2_access_token", "oauth2_refresh_token",
	}
	for _, key := range keysToDelete {
		delete(sess, key)
	}
}
