package middlewares

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/internal/auth/jwt"
	"proxynd/internal/auth/mfa"
	"proxynd/logging"
)

// MFARequiredResponse MFA 필요 응답
type MFARequiredResponse struct {
	Error             string   `json:"error"`
	MFARequired       bool     `json:"mfa_required"`
	AvailableMethods  []string `json:"available_methods"`
	ChallengeRequired bool     `json:"challenge_required"`
	Message           string   `json:"message"`
}

// MFAChallengeResponse MFA 챌린지 응답
type MFAChallengeResponse struct {
	ChallengeID      string `json:"challenge_id"`
	Method           string `json:"method"`
	ExpiresIn        int    `json:"expires_in"`
	Message          string `json:"message"`
	QRCodeURL        string `json:"qr_code_url,omitempty"`        // TOTP 설정 시
	BackupCodesCount int    `json:"backup_codes_count,omitempty"` // 백업 코드 개수
}

// MFAVerificationRequest MFA 검증 요청
type MFAVerificationRequest struct {
	ChallengeID string `json:"challenge_id"`
	Code        string `json:"code"`
	Method      string `json:"method,omitempty"`
}

// MFAMiddlewareConfig MFA 미들웨어 설정
type MFAMiddlewareConfig struct {
	MFAService        *mfa.MFAService
	JWTService        *jwt.JWTService
	SkipPaths         []string      // MFA를 건너뛸 경로들
	RequiredPaths     []string      // MFA가 필수인 경로들
	GracePeriod       time.Duration // MFA 설정 유예 기간
	EnableForAllUsers bool          // 모든 사용자에게 MFA 필수 여부
}

// DefaultMFAMiddlewareConfig 기본 MFA 미들웨어 설정
func DefaultMFAMiddlewareConfig() *MFAMiddlewareConfig {
	return &MFAMiddlewareConfig{
		SkipPaths: []string{
			"/health",
			"/metrics",
			"/auth/login",
			"/auth/callback",
			"/auth/mfa/setup",
			"/auth/mfa/verify",
		},
		RequiredPaths: []string{
			"/admin",
			"/proxy/*/upload", // 업로드 요청은 MFA 필수
		},
		GracePeriod:       7 * 24 * time.Hour, // 7일 유예
		EnableForAllUsers: false,
	}
}

// MFAMiddleware MFA 인증 미들웨어
func MFAMiddleware(config *MFAMiddlewareConfig) fiber.Handler {
	if config == nil {
		config = DefaultMFAMiddlewareConfig()
	}

	logger := logging.GetLogger()

	return func(c *fiber.Ctx) error {
		// MFA를 건너뛸 경로 확인
		path := c.Path()
		for _, skipPath := range config.SkipPaths {
			if strings.HasPrefix(path, skipPath) {
				return c.Next()
			}
		}

		// JWT 토큰에서 사용자 정보 추출
		authHeader := c.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			// 인증되지 않은 사용자는 기본 인증 미들웨어에서 처리
			return c.Next()
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := config.JWTService.ValidateAccessToken(tokenString)
		if err != nil {
			// 유효하지 않은 토큰은 기본 인증 미들웨어에서 처리
			return c.Next()
		}

		// MFA 상태 확인
		userMFA, err := config.MFAService.GetUserMFAStatus(claims.UserID)
		if err != nil {
			// MFA가 설정되지 않은 사용자
			if config.EnableForAllUsers || s.isMFARequiredPath(path, config.RequiredPaths) {
				return c.Status(fiber.StatusUnauthorized).JSON(MFARequiredResponse{
					Error:             "MFA setup required",
					MFARequired:       true,
					AvailableMethods:  []string{"totp"},
					ChallengeRequired: false,
					Message:           "Multi-factor authentication must be set up to access this resource",
				})
			}
			return c.Next()
		}

		// MFA가 활성화되지 않은 경우
		if !userMFA.Enabled {
			// 유예 기간 확인
			if time.Since(userMFA.CreatedAt) > config.GracePeriod ||
				config.EnableForAllUsers ||
				s.isMFARequiredPath(path, config.RequiredPaths) {
				return c.Status(fiber.StatusUnauthorized).JSON(MFARequiredResponse{
					Error:             "MFA setup incomplete",
					MFARequired:       true,
					AvailableMethods:  []string{"totp"},
					ChallengeRequired: false,
					Message:           "Multi-factor authentication setup must be completed",
				})
			}
			return c.Next()
		}

		// MFA 인증 상태 확인 (세션에서)
		mfaVerified := c.Locals("mfa_verified")
		if mfaVerified == true {
			return c.Next()
		}

		// JWT에 MFA 인증 정보가 있는지 확인 (토큰 발급 시 MFA가 완료된 경우)
		if mfaInfo, exists := claims.RegisteredClaims.ExtraFields["mfa_verified"]; exists {
			if verified, ok := mfaInfo.(bool); ok && verified {
				// MFA 검증된 토큰의 경우 시간 제한 확인
				if mfaTime, exists := claims.RegisteredClaims.ExtraFields["mfa_time"]; exists {
					if mfaTimeFloat, ok := mfaTime.(float64); ok {
						mfaTimestamp := time.Unix(int64(mfaTimeFloat), 0)
						// MFA 인증이 1시간 이내인 경우만 유효
						if time.Since(mfaTimestamp) < time.Hour {
							c.Locals("mfa_verified", true)
							return c.Next()
						}
					}
				}
			}
		}

		// MFA 챌린지가 필요한 경우
		availableMethods := s.getAvailableMethods(userMFA)
		if len(availableMethods) == 0 {
			return c.Status(fiber.StatusUnauthorized).JSON(MFARequiredResponse{
				Error:             "No verified MFA methods available",
				MFARequired:       true,
				AvailableMethods:  []string{},
				ChallengeRequired: false,
				Message:           "Please set up a multi-factor authentication method",
			})
		}

		logger.Info("MFA challenge required",
			logging.F("user_id", claims.UserID),
			logging.F("email", claims.Email),
			logging.F("path", path))

		return c.Status(fiber.StatusUnauthorized).JSON(MFARequiredResponse{
			Error:             "MFA verification required",
			MFARequired:       true,
			AvailableMethods:  availableMethods,
			ChallengeRequired: true,
			Message:           "Multi-factor authentication verification required",
		})
	}
}

// MFASetupHandler MFA 설정 핸들러
func MFASetupHandler(mfaService *mfa.MFAService, jwtService *jwt.JWTService) fiber.Handler {
	logger := logging.GetLogger()

	return func(c *fiber.Ctx) error {
		// JWT에서 사용자 정보 추출
		claims, err := extractClaimsFromContext(c, jwtService)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required",
			})
		}

		switch c.Method() {
		case "POST":
			// MFA 활성화 및 TOTP 설정 시작
			return handleMFASetup(c, mfaService, claims, logger)
		case "GET":
			// MFA 상태 조회
			return handleMFAStatus(c, mfaService, claims)
		case "DELETE":
			// MFA 비활성화
			return handleMFADisable(c, mfaService, claims, logger)
		default:
			return c.Status(fiber.StatusMethodNotAllowed).JSON(fiber.Map{
				"error": "Method not allowed",
			})
		}
	}
}

// MFAChallengeHandler MFA 챌린지 핸들러
func MFAChallengeHandler(mfaService *mfa.MFAService, jwtService *jwt.JWTService) fiber.Handler {
	logger := logging.GetLogger()

	return func(c *fiber.Ctx) error {
		// JWT에서 사용자 정보 추출
		claims, err := extractClaimsFromContext(c, jwtService)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required",
			})
		}

		switch c.Method() {
		case "POST":
			// 챌린지 시작
			return handleChallengeStart(c, mfaService, claims, logger)
		default:
			return c.Status(fiber.StatusMethodNotAllowed).JSON(fiber.Map{
				"error": "Method not allowed",
			})
		}
	}
}

// MFAVerifyHandler MFA 검증 핸들러
func MFAVerifyHandler(mfaService *mfa.MFAService, jwtService *jwt.JWTService) fiber.Handler {
	logger := logging.GetLogger()

	return func(c *fiber.Ctx) error {
		switch c.Method() {
		case "POST":
			// 챌린지 검증
			return handleChallengeVerify(c, mfaService, jwtService, logger)
		default:
			return c.Status(fiber.StatusMethodNotAllowed).JSON(fiber.Map{
				"error": "Method not allowed",
			})
		}
	}
}

// helper functions

// isMFARequiredPath MFA가 필수인 경로인지 확인
func (s *MFAMiddlewareConfig) isMFARequiredPath(path string, requiredPaths []string) bool {
	for _, requiredPath := range requiredPaths {
		if strings.Contains(requiredPath, "*") {
			// 와일드카드 패턴 매칭
			pattern := strings.Replace(requiredPath, "*", "", -1)
			if strings.Contains(path, pattern) {
				return true
			}
		} else if strings.HasPrefix(path, requiredPath) {
			return true
		}
	}
	return false
}

// getAvailableMethods 사용 가능한 MFA 방법 목록 반환
func (s *MFAMiddlewareConfig) getAvailableMethods(userMFA *mfa.MFAUserConfig) []string {
	var methods []string
	for method, setup := range userMFA.Methods {
		if setup.Enabled && setup.Verified {
			methods = append(methods, string(method))
		}
	}
	if len(userMFA.BackupCodes) > 0 {
		methods = append(methods, "backup_code")
	}
	return methods
}

// extractClaimsFromContext JWT claims 추출
func extractClaimsFromContext(c *fiber.Ctx, jwtService *jwt.JWTService) (*jwt.Claims, error) {
	authHeader := c.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, fmt.Errorf("missing or invalid authorization header")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	return jwtService.ValidateAccessToken(tokenString)
}

// handleMFASetup MFA 설정 처리
func handleMFASetup(c *fiber.Ctx, mfaService *mfa.MFAService, claims *jwt.Claims, logger logging.Logger) error {
	var request struct {
		Method string `json:"method"`
	}

	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	switch request.Method {
	case "totp", "":
		// TOTP 설정
		userConfig, err := mfaService.EnableMFA(claims.UserID, claims.Email)
		if err != nil {
			logger.Warn("Failed to enable MFA",
				logging.F("user_id", claims.UserID),
				logging.F("error", err))
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		totpSetup, qrURL, err := mfaService.SetupTOTP(claims.UserID)
		if err != nil {
			logger.Error("Failed to setup TOTP",
				logging.F("user_id", claims.UserID),
				logging.F("error", err))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to setup TOTP",
			})
		}

		// QR 코드 생성
		qrBytes, err := mfa.GenerateQRCode(qrURL, 256)
		if err != nil {
			logger.Warn("Failed to generate QR code",
				logging.F("user_id", claims.UserID),
				logging.F("error", err))
		}

		response := fiber.Map{
			"method":       "totp",
			"secret":       totpSetup.Secret,
			"qr_url":       qrURL,
			"backup_codes": userConfig.BackupCodes,
			"message":      "TOTP setup initiated. Please verify with your authenticator app.",
		}

		if qrBytes != nil {
			// QR 코드를 base64로 인코딩하여 포함
			c.Set("Content-Type", "application/json")
			response["qr_code"] = fmt.Sprintf("data:image/png;base64,%s",
				string(qrBytes)) // base64 인코딩 필요
		}

		return c.JSON(response)

	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Unsupported MFA method",
		})
	}
}

// handleMFAStatus MFA 상태 조회 처리
func handleMFAStatus(c *fiber.Ctx, mfaService *mfa.MFAService, claims *jwt.Claims) error {
	userMFA, err := mfaService.GetUserMFAStatus(claims.UserID)
	if err != nil {
		return c.JSON(fiber.Map{
			"enabled": false,
			"methods": []string{},
			"message": "MFA not configured",
		})
	}

	var methods []string
	var methodDetails []fiber.Map
	for method, setup := range userMFA.Methods {
		if setup.Enabled && setup.Verified {
			methods = append(methods, string(method))
			methodDetails = append(methodDetails, fiber.Map{
				"method":     string(method),
				"enabled":    setup.Enabled,
				"verified":   setup.Verified,
				"created_at": setup.CreatedAt,
				"last_used":  setup.LastUsedAt,
			})
		}
	}

	return c.JSON(fiber.Map{
		"enabled":            userMFA.Enabled,
		"methods":            methods,
		"method_details":     methodDetails,
		"backup_codes_count": len(userMFA.BackupCodes),
		"last_used":          userMFA.LastUsedAt,
		"failed_attempts":    userMFA.FailedAttempts,
		"locked_until":       userMFA.LockedUntil,
	})
}

// handleMFADisable MFA 비활성화 처리
func handleMFADisable(c *fiber.Ctx, mfaService *mfa.MFAService, claims *jwt.Claims, logger logging.Logger) error {
	if err := mfaService.DisableMFA(claims.UserID); err != nil {
		logger.Error("Failed to disable MFA",
			logging.F("user_id", claims.UserID),
			logging.F("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to disable MFA",
		})
	}

	logger.Info("MFA disabled by user",
		logging.F("user_id", claims.UserID),
		logging.F("email", claims.Email))

	return c.JSON(fiber.Map{
		"message": "MFA has been disabled",
	})
}

// handleChallengeStart 챌린지 시작 처리
func handleChallengeStart(c *fiber.Ctx, mfaService *mfa.MFAService, claims *jwt.Claims, logger logging.Logger) error {
	var request struct {
		Method string `json:"method"`
	}

	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	method := mfa.MFAMethod(request.Method)
	if method == "" {
		method = mfa.MethodTOTP // 기본값
	}

	challenge, err := mfaService.StartMFAChallenge(claims.UserID, method)
	if err != nil {
		logger.Warn("Failed to start MFA challenge",
			logging.F("user_id", claims.UserID),
			logging.F("method", string(method)),
			logging.F("error", err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	response := MFAChallengeResponse{
		ChallengeID: challenge.ChallengeID,
		Method:      string(challenge.Method),
		ExpiresIn:   int(time.Until(challenge.ExpiresAt).Seconds()),
		Message:     fmt.Sprintf("MFA challenge started using %s", challenge.Method),
	}

	return c.JSON(response)
}

// handleChallengeVerify 챌린지 검증 처리
func handleChallengeVerify(c *fiber.Ctx, mfaService *mfa.MFAService, jwtService *jwt.JWTService, logger logging.Logger) error {
	var request MFAVerificationRequest

	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if request.ChallengeID == "" || request.Code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Challenge ID and code are required",
		})
	}

	// 챌린지 검증
	if err := mfaService.VerifyMFAChallenge(request.ChallengeID, request.Code); err != nil {
		logger.Warn("MFA challenge verification failed",
			logging.F("challenge_id", request.ChallengeID),
			logging.F("error", err))
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid verification code",
		})
	}

	// 세션에 MFA 검증 상태 저장
	c.Locals("mfa_verified", true)
	c.Locals("mfa_verified_at", time.Now())

	logger.Info("MFA challenge verified successfully",
		logging.F("challenge_id", request.ChallengeID))

	return c.JSON(fiber.Map{
		"verified":   true,
		"message":    "MFA verification successful",
		"expires_in": 3600, // 1시간 유효
	})
}
