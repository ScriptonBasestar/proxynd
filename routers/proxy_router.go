package routers

import (
	"github.com/gofiber/fiber/v2"
	"proxynd/configs"
	proxynd "proxynd/handlers/proxy"
)

func ProxyRouter(app *fiber.App) {
	// 전역 설정 읽기
	globalConfig := configs.GlobalConfig{}
	globalConfig.ReadConfig()

	// 통합 프록시 라우터 설정
	// /proxy/:type/*path 형식으로 모든 프록시 요청을 처리
	proxyGroup := app.Group("/proxy")
	
	// 통합 프록시 핸들러로 모든 프록시 타입 처리
	proxyGroup.Get("/:type/*", proxynd.UnifiedProxyHandler)
	
	// 기존 개별 라우트는 하위 호환성을 위해 유지 (선택사항)
	// 향후 제거 가능
}