package auth

import (
	"crypto/subtle"
	"time"

	"github.com/gofiber/fiber/v2"

	middlewares "proxynd/internal/adapters/http/fiber/middleware"
)

// LoginRequest 로그인 요청 구조체
type LoginRequest struct {
	Username string `json:"username" validate:"required,min=1,max=100"`
	Password string `json:"password" validate:"required,min=1"`
}

// LoginResponse 로그인 응답 구조체
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Roles     []string  `json:"roles"`
}

// RefreshRequest 토큰 갱신 요청 구조체
type RefreshRequest struct {
	Token string `json:"token" validate:"required"`
}

// JWTHandler JWT 인증 핸들러
type JWTHandler struct {
	jwtConfig middlewares.JWTConfig
	users     map[string]UserInfo // 임시 사용자 저장소 (실제로는 데이터베이스 사용)
}

// UserInfo 사용자 정보
type UserInfo struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Password string   `json:"password"` // 해시된 비밀번호
	Roles    []string `json:"roles"`
	Enabled  bool     `json:"enabled"`
}

// NewJWTHandler JWT 핸들러 생성자
func NewJWTHandler(jwtConfig middlewares.JWTConfig) *JWTHandler {
	// 임시 사용자 데이터 (실제로는 데이터베이스에서 로드)
	users := map[string]UserInfo{
		"admin": {
			UserID:   "1",
			Username: "admin",
			Password: "admin123", // 실제로는 bcrypt 해시 사용
			Roles:    []string{"admin"},
			Enabled:  true,
		},
		"user": {
			UserID:   "2",
			Username: "user",
			Password: "user123", // 실제로는 bcrypt 해시 사용
			Roles:    []string{"user"},
			Enabled:  true,
		},
		"proxy_manager": {
			UserID:   "3",
			Username: "proxy_manager",
			Password: "proxy123", // 실제로는 bcrypt 해시 사용
			Roles:    []string{"proxy_manager", "user"},
			Enabled:  true,
		},
	}

	return &JWTHandler{
		jwtConfig: jwtConfig,
		users:     users,
	}
}

// Login 로그인 핸들러
func (h *JWTHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request format",
			"code":  "INVALID_REQUEST",
		})
	}

	// 사용자 인증 확인
	user, valid := h.authenticateUser(req.Username, req.Password)
	if !valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid username or password",
			"code":  "AUTH_FAILED",
		})
	}

	// 사용자 활성화 상태 확인
	if !user.Enabled {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "User account is disabled",
			"code":  "USER_DISABLED",
		})
	}

	// JWT 토큰 생성
	token, err := middlewares.GenerateJWTToken(user.UserID, user.Username, user.Roles, h.jwtConfig)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate authentication token",
			"code":  "TOKEN_GENERATION_FAILED",
		})
	}

	expiresAt := time.Now().Add(h.jwtConfig.TokenDuration)

	return c.JSON(LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		UserID:    user.UserID,
		Username:  user.Username,
		Roles:     user.Roles,
	})
}

// RefreshToken 토큰 갱신 핸들러
func (h *JWTHandler) RefreshToken(c *fiber.Ctx) error {
	var req RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request format",
			"code":  "INVALID_REQUEST",
		})
	}

	// 기존 토큰 검증 (만료된 토큰도 클레임 정보 추출을 위해 허용)
	claims, err := middlewares.ValidateJWTToken(req.Token, h.jwtConfig)
	if err != nil && !isTokenExpiredError(err) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid token",
			"code":  "INVALID_TOKEN",
		})
	}

	// 사용자 정보 재확인
	user, exists := h.users[claims.Username]
	if !exists || !user.Enabled {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User not found or disabled",
			"code":  "USER_INVALID",
		})
	}

	// 새 토큰 생성
	newToken, err := middlewares.GenerateJWTToken(user.UserID, user.Username, user.Roles, h.jwtConfig)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to refresh token",
			"code":  "TOKEN_REFRESH_FAILED",
		})
	}

	expiresAt := time.Now().Add(h.jwtConfig.TokenDuration)

	return c.JSON(LoginResponse{
		Token:     newToken,
		ExpiresAt: expiresAt,
		UserID:    user.UserID,
		Username:  user.Username,
		Roles:     user.Roles,
	})
}

// Logout 로그아웃 핸들러 (토큰 무효화)
func (h *JWTHandler) Logout(c *fiber.Ctx) error {
	// 실제로는 토큰을 블랙리스트에 추가하거나 Redis 등에서 무효화
	// 현재는 클라이언트에서 토큰을 삭제하도록 응답만 반환

	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
		"code":    "LOGOUT_SUCCESS",
	})
}

// GetUserInfo 현재 사용자 정보 조회
func (h *JWTHandler) GetUserInfo(c *fiber.Ctx) error {
	userID, username, roles, ok := middlewares.GetUserFromContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
			"code":  "AUTH_REQUIRED",
		})
	}

	user, exists := h.users[username]
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
			"code":  "USER_NOT_FOUND",
		})
	}

	return c.JSON(fiber.Map{
		"user_id":  userID,
		"username": username,
		"roles":    roles,
		"enabled":  user.Enabled,
	})
}

// authenticateUser 사용자 인증 (실제로는 bcrypt 사용)
func (h *JWTHandler) authenticateUser(username, password string) (UserInfo, bool) {
	user, exists := h.users[username]
	if !exists {
		return UserInfo{}, false
	}

	// 실제로는 bcrypt.CompareHashAndPassword 사용
	// 현재는 단순 비교 (개발용)
	if subtle.ConstantTimeCompare([]byte(user.Password), []byte(password)) == 1 {
		return user, true
	}

	return UserInfo{}, false
}

// isTokenExpiredError 토큰 만료 에러인지 확인
func isTokenExpiredError(err error) bool {
	return err != nil && (err.Error() == "token expired" ||
		err.Error() == "Token is expired")
}
