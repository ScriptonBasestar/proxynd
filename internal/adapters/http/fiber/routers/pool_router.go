package routers

import (
	"github.com/gofiber/fiber/v2"

	handlers "proxynd/internal/adapters/http/fiber/handlers"
	authHandlers "proxynd/internal/adapters/http/fiber/handlers/auth"
	middlewares "proxynd/internal/adapters/http/fiber/middleware"
)

// PoolRouter Connection Pool 관리 라우터
func PoolRouter(app *fiber.App) {
	// Pool Handler 인스턴스 생성
	poolHandler := handlers.NewPoolHandler()

	// API 그룹 생성 (관리자 인증 필요)
	poolGroup := app.Group("/api/v1/pool")

	// 관리자 인증 미들웨어 적용
	poolGroup.Use(
		middlewares.DefaultAccessLogMiddleware(), // 액세스 로깅
		authHandlers.OptionalAuth(),              // 선택적 인증
		middlewares.AdminOnlyMiddleware(),        // Admin role required
	)

	// Connection Pool 상태 및 통계
	poolGroup.Get("/status", poolHandler.GetPoolStatus)     // Pool 상태 조회
	poolGroup.Get("/health", poolHandler.GetPoolHealth)     // Pool 헬스체크
	poolGroup.Get("/statistics", poolHandler.GetPoolStatus) // 통계 (status와 동일)

	// Connection Pool 설정 관리
	poolGroup.Get("/config", poolHandler.GetPoolConfiguration)          // 설정 조회
	poolGroup.Put("/config", poolHandler.UpdatePoolConfiguration)       // 설정 업데이트
	poolGroup.Post("/config/save", poolHandler.SavePoolConfiguration)   // 설정 저장
	poolGroup.Post("/config/reset", poolHandler.ResetPoolConfiguration) // 설정 리셋

	// 프록시별 타임아웃 관리
	poolGroup.Get("/timeouts", poolHandler.GetProxyTimeouts)         // 모든 타임아웃 조회
	poolGroup.Put("/timeouts/:type", poolHandler.UpdateProxyTimeout) // 특정 프록시 타임아웃 업데이트

	// 읽기 전용 엔드포인트 (인증 불필요)
	publicGroup := app.Group("/api/v1/pool/public")
	publicGroup.Use(middlewares.DefaultAccessLogMiddleware())

	publicGroup.Get("/status", poolHandler.GetPoolStatus)      // 공개 상태 조회
	publicGroup.Get("/health", poolHandler.GetPoolHealth)      // 공개 헬스체크
	publicGroup.Get("/timeouts", poolHandler.GetProxyTimeouts) // 공개 타임아웃 조회
}

// RegisterPoolMiddleware Connection Pool 미들웨어 등록
func RegisterPoolMiddleware(app *fiber.App) {
	// Connection Pool 통계 수집 미들웨어
	app.Use(func(c *fiber.Ctx) error {
		// 요청 시작 시간 기록
		c.Locals("request_start_time", c.Context().Time())

		// 다음 핸들러 실행
		err := c.Next()

		// 응답 후 통계 업데이트 (비동기)
		go func() {
			// Connection Pool 사용 통계 업데이트
			// 실제 구현에서는 메트릭 시스템에 전송
		}()

		return err
	})
}
