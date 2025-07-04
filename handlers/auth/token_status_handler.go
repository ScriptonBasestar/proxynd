package auth

import (
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/configs"
	"proxynd/internal/auth/jwt"
	"proxynd/logging"
)

// TokenStatusResponse 토큰 상태 응답
type TokenStatusResponse struct {
	Valid           bool      `json:"valid"`
	ExpiresAt       time.Time `json:"expires_at,omitempty"`
	ExpiresIn       int64     `json:"expires_in,omitempty"` // 초 단위
	ExpiresSoon     bool      `json:"expires_soon"`
	RefreshRequired bool      `json:"refresh_required"`
	RefreshEndpoint string    `json:"refresh_endpoint"`
	UserInfo        fiber.Map `json:"user_info,omitempty"`
}

// GetTokenStatus 토큰 상태 조회
func GetTokenStatus(c *fiber.Ctx) error {
	sess, err := getSession(c)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get session",
		})
	}

	// JWT 액세스 토큰 확인
	jwtAccessToken, exists := sess["jwt_access_token"]
	if !exists || jwtAccessToken == nil {
		return c.JSON(TokenStatusResponse{
			Valid:           false,
			RefreshRequired: true,
			RefreshEndpoint: "/auth/refresh",
		})
	}

	// OAuth2 설정 로드
	oauth2Config := &configs.OAuth2Config{}
	if err := oauth2Config.ReadConfig(); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "OAuth2 configuration not available",
		})
	}

	// JWT 서비스 생성
	jwtService := jwt.NewJWTService(oauth2Config)

	// 토큰 검증
	claims, err := jwtService.ValidateAccessToken(jwtAccessToken.(string))
	if err != nil {
		logging.GetLogger().Debug("Token validation failed", logging.F("error", err))

		return c.JSON(TokenStatusResponse{
			Valid:           false,
			RefreshRequired: true,
			RefreshEndpoint: "/auth/refresh",
		})
	}

	// 토큰 만료 시간 계산
	now := time.Now()
	expiresAt := claims.ExpiresAt.Time
	timeUntilExpiry := expiresAt.Sub(now)

	// 토큰이 곧 만료되는지 확인 (10분 이내)
	expiresSoon := timeUntilExpiry < 10*time.Minute
	refreshRequired := timeUntilExpiry <= 0

	// 사용자 정보 추출
	userInfo := jwtService.ExtractUserInfo(claims)

	// 세션에서 추가 정보 가져오기
	if userData, exists := sess["user"]; exists {
		if userMap, ok := userData.(fiber.Map); ok {
			if avatar, exists := userMap["avatar"]; exists {
				userInfo["avatar"] = avatar
			}
			if loginTime, exists := userMap["login_time"]; exists {
				userInfo["login_time"] = loginTime
			}
		}
	}

	response := TokenStatusResponse{
		Valid:           !refreshRequired,
		ExpiresAt:       expiresAt,
		ExpiresIn:       int64(timeUntilExpiry.Seconds()),
		ExpiresSoon:     expiresSoon,
		RefreshRequired: refreshRequired,
		RefreshEndpoint: "/auth/refresh",
		UserInfo:        userInfo,
	}

	// 토큰이 곧 만료되는 경우 헤더에도 표시
	if expiresSoon {
		c.Set("X-Token-Expires-Soon", "true")
		c.Set("X-Token-Expires-In", timeUntilExpiry.String())
		c.Set("X-Refresh-Endpoint", "/auth/refresh")
	}

	return c.JSON(response)
}

// ValidateTokenEndpoint 토큰 유효성 검증 전용 엔드포인트
func ValidateTokenEndpoint(c *fiber.Ctx) error {
	// Authorization 헤더 또는 쿼리 매개변수에서 토큰 추출
	token := c.Get("Authorization")
	if token == "" {
		token = c.Query("token")
	}

	if token == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Token is required",
		})
	}

	// Bearer 토큰 형식 처리
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	// OAuth2 설정 로드
	oauth2Config := &configs.OAuth2Config{}
	if err := oauth2Config.ReadConfig(); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "OAuth2 configuration not available",
		})
	}

	// JWT 서비스 생성
	jwtService := jwt.NewJWTService(oauth2Config)

	// 토큰 검증
	claims, err := jwtService.ValidateAccessToken(token)
	if err != nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"valid": false,
			"error": "Invalid or expired token",
		})
	}

	// 토큰 만료 시간 계산
	now := time.Now()
	expiresAt := claims.ExpiresAt.Time
	timeUntilExpiry := expiresAt.Sub(now)

	return c.JSON(fiber.Map{
		"valid":        true,
		"expires_at":   expiresAt,
		"expires_in":   int64(timeUntilExpiry.Seconds()),
		"expires_soon": timeUntilExpiry < 10*time.Minute,
		"user_id":      claims.UserID,
		"email":        claims.Email,
		"role":         claims.Role,
	})
}

// RefreshTokenStatus 토큰 갱신 후 상태 조회
func RefreshTokenStatus(c *fiber.Ctx) error {
	// 토큰 갱신 수행
	if err := RefreshToken(c); err != nil {
		return err
	}

	// 갱신 후 상태 조회
	return GetTokenStatus(c)
}

// BatchTokenValidation 배치 토큰 검증 (여러 토큰 동시 검증)
func BatchTokenValidation(c *fiber.Ctx) error {
	type BatchRequest struct {
		Tokens []string `json:"tokens"`
	}

	var req BatchRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if len(req.Tokens) == 0 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "No tokens provided",
		})
	}

	if len(req.Tokens) > 10 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Too many tokens (max 10)",
		})
	}

	// OAuth2 설정 로드
	oauth2Config := &configs.OAuth2Config{}
	if err := oauth2Config.ReadConfig(); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "OAuth2 configuration not available",
		})
	}

	// JWT 서비스 생성
	jwtService := jwt.NewJWTService(oauth2Config)

	results := make([]fiber.Map, len(req.Tokens))

	for i, token := range req.Tokens {
		claims, err := jwtService.ValidateAccessToken(token)
		if err != nil {
			results[i] = fiber.Map{
				"valid": false,
				"error": err.Error(),
			}
			continue
		}

		// 토큰 만료 시간 계산
		now := time.Now()
		expiresAt := claims.ExpiresAt.Time
		timeUntilExpiry := expiresAt.Sub(now)

		results[i] = fiber.Map{
			"valid":        true,
			"expires_at":   expiresAt,
			"expires_in":   int64(timeUntilExpiry.Seconds()),
			"expires_soon": timeUntilExpiry < 10*time.Minute,
			"user_id":      claims.UserID,
			"email":        claims.Email,
			"role":         claims.Role,
		}
	}

	return c.JSON(fiber.Map{
		"results": results,
	})
}
