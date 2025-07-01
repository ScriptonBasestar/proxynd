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
	
	// 인증 상태 확인
	authGroup.Get("/status", auth.GetAuthStatus)
}