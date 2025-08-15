package routers

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/handlers/auth"
	"proxynd/logging"
	"proxynd/middlewares"
)

// AuthRouter OAuth2 인증 관련 라우터
func AuthRouter(app *fiber.App) {
	JWTAuthRouter(app)
	OAuth2Router(app)
}

// JWTAuthRouter JWT 인증 라우터
func JWTAuthRouter(app *fiber.App) {
	// JWT 설정 (실제로는 설정 파일에서 로드)
	jwtConfig := middlewares.JWTConfig{
		SecretKey:     "your-secret-key-change-this-in-production",
		TokenDuration: 24 * time.Hour,
		Issuer:        "proxynd",
		SkipPaths: []string{
			"/auth/jwt/login",
			"/auth/jwt/refresh",
			"/health",
			"/metrics",
			"/status",
		},
	}

	// 로거 생성 (사용되지 않지만 향후 확장을 위해 유지)
	_ = logging.GetLogger()

	// JWT 핸들러 생성
	jwtHandler := auth.NewJWTHandler(jwtConfig)

	// JWT 인증 그룹
	jwtGroup := app.Group("/auth/jwt")

	// JWT 로그인 엔드포인트 (인증 불필요)
	jwtGroup.Post("/login", jwtHandler.Login)

	// JWT 토큰 갱신 엔드포인트 (인증 불필요)
	jwtGroup.Post("/refresh", jwtHandler.RefreshToken)

	// JWT 인증이 필요한 엔드포인트들
	protectedGroup := jwtGroup.Use(middlewares.JWTMiddleware(jwtConfig))

	// 로그아웃 엔드포인트
	protectedGroup.Post("/logout", jwtHandler.Logout)

	// 현재 사용자 정보 조회
	protectedGroup.Get("/me", jwtHandler.GetUserInfo)
}

// OAuth2Router OAuth2 인증 라우터 (기존 코드)
func OAuth2Router(app *fiber.App) {
	// OAuth2 인증 그룹
	authGroup := app.Group("/auth")

	// OAuth2 로그인 시작 엔드포인트
	authGroup.Get("/login/:provider", auth.StartOAuth2Login)

	// OAuth2 콜백 엔드포인트
	authGroup.Get("/callback/:provider", auth.HandleOAuth2Callback)

	// OAuth2 로그아웃 엔드포인트
	authGroup.Post("/logout", auth.HandleLogout)

	// 토큰 갱신 엔드포인트
	authGroup.Post("/refresh", auth.RefreshToken)

	// 현재 사용자 정보 조회
	authGroup.Get("/me", auth.GetCurrentUser)

	// 인증 상태 확인 (기존 호환성)
	authGroup.Get("/status", auth.GetAuthStatus)

	// 토큰 상태 관련 새로운 엔드포인트들
	tokenGroup := authGroup.Group("/token")

	// 토큰 상태 상세 조회
	tokenGroup.Get("/status", auth.GetTokenStatus)

	// 토큰 유효성 검증
	tokenGroup.Post("/validate", auth.ValidateTokenEndpoint)

	// 토큰 갱신 후 상태 조회
	tokenGroup.Post("/refresh-status", auth.RefreshTokenStatus)

	// 배치 토큰 검증
	tokenGroup.Post("/batch-validate", auth.BatchTokenValidation)
}
