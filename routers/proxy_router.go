package routers

import (
	"github.com/gofiber/fiber/v2"
	"proxynd/alerts"
	"proxynd/configs"
	proxynd "proxynd/handlers/proxy"
	"proxynd/middlewares"
)

func ProxyRouter(app *fiber.App) {
	// 전역 설정 읽기
	globalConfig := configs.GlobalConfig{}
	globalConfig.ReadConfig()

	// 알림 관리자 초기화
	alertConfig := loadAlertConfig()
	alertManager := alerts.NewAlertManager(alertConfig)
	
	// 로그 알림 채널 등록
	if logAlerter, err := alerts.NewLogAlerter(alerts.LogAlerterConfig{
		Enabled:    true,
		LogFile:    "./logs/verification-alerts.log",
		JSONFormat: true,
	}); err == nil {
		alertManager.RegisterAlerter(logAlerter)
	}
	
	// 검증 핸들러 생성
	verificationHandler := proxynd.NewVerificationHandler(&globalConfig, alertManager)

	// 통합 프록시 라우터 설정
	// /proxy/:type/*path 형식으로 모든 프록시 요청을 처리
	proxyGroup := app.Group("/proxy")
	
	// 프록시 미들웨어 적용
	proxyGroup.Use(middlewares.ProxyPolicyMiddleware())
	proxyGroup.Use(middlewares.DefaultAccessLogMiddleware())
	
	// 패키지 검증 미들웨어 추가
	proxyGroup.Use(verificationHandler.VerificationMiddleware())
	
	// 통합 프록시 핸들러로 모든 프록시 타입 처리
	proxyGroup.Get("/:type/*", proxynd.UnifiedProxyHandler)
	proxyGroup.Post("/:type/*", proxynd.UnifiedProxyHandler)
	proxyGroup.Put("/:type/*", proxynd.UnifiedProxyHandler)
	
	// 기존 개별 라우트는 하위 호환성을 위해 유지 (선택사항)
	// 향후 제거 가능
}

// loadAlertConfig 알림 설정 로드
func loadAlertConfig() *alerts.AlertConfig {
	// 기본 설정
	return &alerts.AlertConfig{
		Enabled: true,
		RateLimit: alerts.RateLimitConfig{
			Enabled:      true,
			MaxPerMinute: 10,
			MaxPerHour:   100,
			BurstSize:    5,
		},
	}
}