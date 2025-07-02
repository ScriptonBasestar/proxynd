package routers

import (
	"github.com/gofiber/fiber/v2"
	"proxynd/handlers/auth"
)

// AuthRouter OAuth2 인증 관련 라우터
func AuthRouter(app *fiber.App) {
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