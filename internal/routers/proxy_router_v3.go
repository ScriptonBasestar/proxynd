package routers

import (
	"github.com/gofiber/fiber/v2"

	authHandlers "proxynd/internal/handlers-legacy/auth"
	proxyHandlers "proxynd/internal/handlers-legacy/proxy"
	"proxynd/internal/middleware-legacy"
)

// ProxyRouterV3 새로운 V3 프록시 라우터 - 단순화된 Factory 패턴 사용
func ProxyRouterV3(app *fiber.App) {
	// V3 프록시 라우트 그룹 생성
	proxyGroup := app.Group("/v3/proxy")

	// 프록시 상태 API 엔드포인트
	proxyGroup.Get("/status", proxyHandlers.GetProxyStatus)
	proxyGroup.Get("/types", proxyHandlers.GetProxyTypes)

	// 프록시 요청 처리 (모든 HTTP 메서드 지원)
	// 미들웨어 체인:
	// 1. 프록시 타입 유효성 검사
	// 2. 프록시 정책 미들웨어
	// 3. 액세스 로그 미들웨어
	// 4. 인증 미들웨어 (선택적)
	// 5. V3 통합 프록시 핸들러
	proxyGroup.All("/:type/*",
		createV3DeprecationMiddleware(),          // V3 Deprecation warning
		proxyHandlers.ValidateProxyType,          // 프록시 타입 검증
		middlewares.ProxyPolicyMiddleware(),      // 프록시 정책 적용
		middlewares.DefaultAccessLogMiddleware(), // 액세스 로깅
		authHandlers.OptionalAuth(),              // 선택적 OAuth2/JWT 인증
		authHandlers.BasicAuthFallback(),         // Basic Auth 폴백
		proxyHandlers.UnifiedProxyHandlerV3,      // V3 통합 핸들러
	)
}

// UpdateExistingProxyRouter 기존 프록시 라우터를 V3 핸들러로 업데이트
func UpdateExistingProxyRouter(app *fiber.App) {
	// 기존 /proxy/:type/* 라우트들을 V3 핸들러로 업데이트
	// 기존 미들웨어 체인 유지하면서 핸들러만 교체

	// GET 요청
	app.Get("/proxy/:type/*",
		proxyHandlers.ValidateProxyType, // 프록시 타입 검증 추가
		middlewares.ProxyPolicyMiddleware(),
		middlewares.DefaultAccessLogMiddleware(),
		authHandlers.OptionalAuth(),
		authHandlers.BasicAuthFallback(),
		proxyHandlers.UnifiedProxyHandlerV3, // V3 핸들러로 교체
	)

	// POST 요청
	app.Post("/proxy/:type/*",
		proxyHandlers.ValidateProxyType,
		middlewares.ProxyPolicyMiddleware(),
		middlewares.DefaultAccessLogMiddleware(),
		authHandlers.OptionalAuth(),
		authHandlers.BasicAuthFallback(),
		proxyHandlers.UnifiedProxyHandlerV3,
	)

	// PUT 요청
	app.Put("/proxy/:type/*",
		proxyHandlers.ValidateProxyType,
		middlewares.ProxyPolicyMiddleware(),
		middlewares.DefaultAccessLogMiddleware(),
		authHandlers.OptionalAuth(),
		authHandlers.BasicAuthFallback(),
		proxyHandlers.UnifiedProxyHandlerV3,
	)

	// DELETE 요청 (일부 프록시 타입에서 지원)
	app.Delete("/proxy/:type/*",
		proxyHandlers.ValidateProxyType,
		middlewares.ProxyPolicyMiddleware(),
		middlewares.DefaultAccessLogMiddleware(),
		authHandlers.OptionalAuth(),
		authHandlers.BasicAuthFallback(),
		proxyHandlers.UnifiedProxyHandlerV3,
	)

	// HEAD 요청 (메타데이터 확인용)
	app.Head("/proxy/:type/*",
		proxyHandlers.ValidateProxyType,
		middlewares.ProxyPolicyMiddleware(),
		middlewares.DefaultAccessLogMiddleware(),
		authHandlers.OptionalAuth(),
		authHandlers.BasicAuthFallback(),
		proxyHandlers.UnifiedProxyHandlerV3,
	)
}

// RegisterProxyAPI 프록시 관련 API 엔드포인트 등록
func RegisterProxyAPI(app *fiber.App) {
	apiGroup := app.Group("/api/v1/proxy")

	// 프록시 상태 조회
	apiGroup.Get("/status", proxyHandlers.GetProxyStatus)

	// 지원하는 프록시 타입 목록
	apiGroup.Get("/types", proxyHandlers.GetProxyTypes)

	// 특정 프록시 타입의 상세 정보
	apiGroup.Get("/types/:type", func(c *fiber.Ctx) error {
		proxyType := c.Params("type")
		factory := proxyHandlers.GetGlobalFactoryV3()

		if !factory.IsSupported(proxyType) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "지원하지 않는 프록시 타입",
				"type":  proxyType,
			})
		}

		info, err := factory.GetHandlerInfo(proxyType)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "핸들러 정보 조회 실패",
				"type":  proxyType,
			})
		}

		return c.JSON(fiber.Map{
			"type": proxyType,
			"info": info,
		})
	})

	// 프록시 헬스체크
	apiGroup.Get("/health", func(c *fiber.Ctx) error {
		factory := proxyHandlers.GetGlobalFactoryV3()
		healthStatus := factory.HealthCheck()

		allHealthy := true
		for _, healthy := range healthStatus {
			if !healthy {
				allHealthy = false
				break
			}
		}

		status := fiber.StatusOK
		if !allHealthy {
			status = fiber.StatusServiceUnavailable
		}

		return c.Status(status).JSON(fiber.Map{
			"overall_status": map[string]interface{}{
				"healthy": allHealthy,
				"status":  map[bool]string{true: "healthy", false: "unhealthy"}[allHealthy],
			},
			"proxy_status": healthStatus,
			"timestamp":    c.Context().Time().Format("2006-01-02T15:04:05Z07:00"),
		})
	})
}

// createV3DeprecationMiddleware creates a middleware that adds V3 deprecation headers
func createV3DeprecationMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Add deprecation headers
		c.Set("X-API-Deprecated", "true")
		c.Set("X-API-Deprecated-Info", "Use /api/v1/proxy/:type/* instead")
		c.Set("X-API-Migration-Guide", "https://docs.proxynd.io/api/migration")

		return c.Next()
	}
}
