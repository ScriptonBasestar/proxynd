package routers

import (
	"github.com/gofiber/fiber/v2"
	"proxynd/logging"
)

// APICompatibilityRouter CLI API 호환성을 위한 라우터
// CLI에서 기대하는 엔드포인트 경로를 기존 구현에 연결
func APICompatibilityRouter(app *fiber.App) {
	logger := logging.GetLogger()

	// CLI에서 기대하는 /api/health -> /api/status/health로 리다이렉트
	app.Get("/api/health", func(c *fiber.Ctx) error {
		logger.Info("Redirecting /api/health to /api/status/health")
		return getHealthCheck(c)
	})

	// CLI에서 기대하는 /api/metrics -> /api/status/metrics로 리다이렉트
	app.Get("/api/metrics", func(c *fiber.Ctx) error {
		logger.Info("Redirecting /api/metrics to /api/status/metrics")
		return getMetrics(c)
	})

	// 추가적인 호환성 엔드포인트들
	api := app.Group("/api")

	// 기존 /healthz를 /api/health 형식으로도 제공
	api.Get("/healthz", func(c *fiber.Ctx) error {
		logger.Info("Serving healthz through /api/healthz")
		// 기존 healthz 핸들러 호출
		return c.Redirect("/healthz", 302)
	})

	// 전역 서버 상태를 /api/status로도 제공 (이미 StatusRouter에서 처리됨)
	logger.Info("API compatibility router initialized")
}

// CLICompatibilityMiddleware CLI 도구와의 호환성을 위한 미들웨어
func CLICompatibilityMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// CLI에서 기대하는 응답 헤더 추가
		c.Set("X-ProxyND-API-Version", "v1")
		c.Set("X-ProxyND-CLI-Compatible", "true")
		
		// CORS 헤더 추가 (CLI 도구가 브라우저에서 실행될 경우 대비)
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Content-Type,Authorization")

		return c.Next()
	}
}