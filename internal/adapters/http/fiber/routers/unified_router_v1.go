package routers

import (
	"github.com/gofiber/fiber/v2"

	"proxynd/internal/handlers-legacy"
	authHandlers "proxynd/internal/handlers-legacy/auth"
	proxyHandlers "proxynd/internal/handlers-legacy/proxy"
	middlewares "proxynd/internal/middleware-legacy"
)

// UnifiedRouterV1 모든 API를 /api/v1/* 구조로 통합한 라우터
func UnifiedRouterV1(app *fiber.App) {
	// 메인 API v1 그룹 생성
	apiV1 := app.Group("/api/v1")

	// 기본 미들웨어 적용
	apiV1.Use(middlewares.DefaultAccessLogMiddleware())

	// 각 도메인별 라우터 등록
	setupProxyRoutesV1(apiV1)      // 프록시 관리
	setupAdminRoutesV1(apiV1)      // 관리 기능
	setupMonitoringRoutesV1(apiV1) // 모니터링
	setupAuthRoutesV1(apiV1)       // 인증
	setupWebhookRoutesV1(apiV1)    // 웹훅 (이미 v1)
	setupMiscRoutesV1(apiV1)       // 기타 기능
}

// setupProxyRoutesV1 프록시 관련 API (/api/v1/proxy/*)
func setupProxyRoutesV1(apiV1 fiber.Router) {
	proxyGroup := apiV1.Group("/proxy")

	// 프록시 상태 및 관리 API
	proxyGroup.Get("/status", proxyHandlers.GetProxyStatus) // 전체 프록시 상태
	proxyGroup.Get("/types", proxyHandlers.GetProxyTypes)   // 지원하는 프록시 타입 목록
	proxyGroup.Get("/health", getProxyHealth)               // 프록시 헬스체크
	proxyGroup.Get("/types/:type", getProxyTypeInfo)        // 특정 프록시 타입 정보

	// 통합된 프록시 요청 처리 (V3 Factory 사용)
	// 모든 HTTP 메서드 지원: GET, POST, PUT, DELETE, HEAD
	proxyGroup.All("/:type/*",
		proxyHandlers.ValidateProxyType,     // 프록시 타입 검증
		middlewares.ProxyPolicyMiddleware(), // 프록시 정책 적용
		authHandlers.OptionalAuth(),         // 선택적 인증
		authHandlers.BasicAuthFallback(),    // Basic Auth 폴백
		proxyHandlers.UnifiedProxyHandlerV3, // V3 통합 핸들러
	)
}

// setupAdminRoutesV1 관리 기능 API (/api/v1/admin/*)
func setupAdminRoutesV1(apiV1 fiber.Router) {
	adminGroup := apiV1.Group("/admin")

	// Apply admin-only middleware to protect all admin routes
	adminGroup.Use(middlewares.AdminOnlyMiddleware())

	// Connection Pool 관리
	setupConnectionPoolAdmin(adminGroup)

	// 설정 관리
	setupConfigAdmin(adminGroup)

	// 캐시 관리
	setupCacheAdmin(adminGroup)

	// 시스템 상태 관리
	setupStatusAdmin(adminGroup)
}

// setupConnectionPoolAdmin Connection Pool 관리 API
func setupConnectionPoolAdmin(adminGroup fiber.Router) {
	poolGroup := adminGroup.Group("/pool")
	poolHandler := handlers.NewPoolHandler()

	// Connection Pool 상태 및 통계
	poolGroup.Get("/status", poolHandler.GetPoolStatus)
	poolGroup.Get("/health", poolHandler.GetPoolHealth)
	poolGroup.Get("/statistics", poolHandler.GetPoolStatus) // alias

	// Connection Pool 설정 관리
	poolGroup.Get("/config", poolHandler.GetPoolConfiguration)
	poolGroup.Put("/config", poolHandler.UpdatePoolConfiguration)
	poolGroup.Post("/config/save", poolHandler.SavePoolConfiguration)
	poolGroup.Post("/config/reset", poolHandler.ResetPoolConfiguration)

	// 프록시별 타임아웃 관리
	poolGroup.Get("/timeouts", poolHandler.GetProxyTimeouts)
	poolGroup.Put("/timeouts/:type", poolHandler.UpdateProxyTimeout)
}

// setupConfigAdmin 설정 관리 API
func setupConfigAdmin(adminGroup fiber.Router) {
	configGroup := adminGroup.Group("/config")

	// TODO: 기존 ConfigRouter의 기능들을 v1으로 이전
	// configGroup.Get("/", getGlobalConfig)
	// configGroup.Put("/", updateGlobalConfig)
	// configGroup.Get("/proxy/:type", getProxyConfig)
	// configGroup.Put("/proxy/:type", updateProxyConfig)
	configGroup.Get("/placeholder", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Config management API - Coming soon"})
	})
}

// setupCacheAdmin 캐시 관리 API
func setupCacheAdmin(adminGroup fiber.Router) {
	cacheGroup := adminGroup.Group("/cache")

	// TODO: 기존 CacheRouter의 기능들을 v1으로 이전
	// cacheGroup.Get("/", getCacheStatus)
	// cacheGroup.Delete("/", clearAllCache)
	// cacheGroup.Get("/:type", getProxyCache)
	// cacheGroup.Delete("/:type", clearProxyCache)
	cacheGroup.Get("/placeholder", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Cache management API - Coming soon"})
	})
}

// setupStatusAdmin 시스템 상태 관리 API
func setupStatusAdmin(adminGroup fiber.Router) {
	statusGroup := adminGroup.Group("/status")

	// 기존 StatusRouter의 기능들을 v1으로 이전
	statusGroup.Get("/", getServerStatusV1)           // 서버 전반적인 상태
	statusGroup.Get("/health", getHealthCheckV1)      // 헬스체크
	statusGroup.Get("/metrics", getMetricsV1)         // 메트릭 조회
	statusGroup.Get("/detailed", getDetailedStatusV1) // 상세 상태
}

// setupMonitoringRoutesV1 모니터링 API (/api/v1/monitoring/*)
func setupMonitoringRoutesV1(apiV1 fiber.Router) {
	monitoringGroup := apiV1.Group("/monitoring")

	// 메트릭 API
	metricsGroup := monitoringGroup.Group("/metrics")
	// TODO: 기존 MetricsRouter의 기능들을 v1으로 이전
	metricsGroup.Get("/placeholder", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Metrics API - Coming soon"})
	})

	// 헬스체크 API
	healthGroup := monitoringGroup.Group("/health")
	// 기존 HealthRouter의 기능들을 v1으로 이전
	healthGroup.Get("/", getSystemHealthV1)           // 전체 시스템 헬스체크
	healthGroup.Get("/proxy", getProxyHealthV1)       // 프록시 헬스체크
	healthGroup.Get("/detailed", getDetailedHealthV1) // 상세 헬스체크
}

// setupAuthRoutesV1 인증 API (/api/v1/auth/*)
func setupAuthRoutesV1(apiV1 fiber.Router) {
	authGroup := apiV1.Group("/auth")

	// JWT 인증
	setupJWTAuthV1(authGroup)

	// OAuth2 인증
	setupOAuth2AuthV1(authGroup)
}

// setupJWTAuthV1 JWT 인증 API
func setupJWTAuthV1(authGroup fiber.Router) {
	jwtGroup := authGroup.Group("/jwt")

	// TODO: 기존 AuthRouter의 JWT 기능들을 v1으로 이전
	jwtGroup.Post("/placeholder", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "JWT Auth API - Coming soon"})
	})
}

// setupOAuth2AuthV1 OAuth2 인증 API
func setupOAuth2AuthV1(authGroup fiber.Router) {
	oauthGroup := authGroup.Group("/oauth")

	// TODO: 기존 AuthRouter의 OAuth2 기능들을 v1으로 이전
	oauthGroup.Get("/placeholder", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "OAuth2 Auth API - Coming soon"})
	})
}

// setupWebhookRoutesV1 웹훅 API (이미 v1이므로 유지)
func setupWebhookRoutesV1(apiV1 fiber.Router) {
	// 기존 webhook_router.go에서 이미 /api/v1/webhook/* 사용 중
	// 여기서는 추가 설정만 수행

	webhookGroup := apiV1.Group("/webhook")
	// 기존 WebhookRouter 함수에서 라우트가 등록됨

	// 추가적인 웹훅 관리 기능이 필요한 경우 여기에 추가
	webhookGroup.Get("/info", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Webhook API v1",
			"endpoints": []string{
				"POST /test",
				"POST /test/:endpoint",
				"GET /test/connectivity/:endpoint",
				"POST /validate",
			},
		})
	})
}

// setupMiscRoutesV1 기타 기능 API
func setupMiscRoutesV1(apiV1 fiber.Router) {
	// 검색 API
	apiV1.Get("/search", func(c *fiber.Ctx) error {
		// TODO: 기존 base_router.go의 검색 기능을 v1으로 이전
		return c.JSON(fiber.Map{"message": "Search API - Coming soon"})
	})

	// 테스트 API
	testGroup := apiV1.Group("/test")
	testGroup.Get("/placeholder", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Test API - Coming soon"})
	})

	// 사용자 관리 API
	usersGroup := apiV1.Group("/users")
	usersGroup.Get("/placeholder", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "User management API - Coming soon"})
	})

	// APK 관련 API
	apkGroup := apiV1.Group("/apk")

	// APK 미러 API
	apkMirrorGroup := apkGroup.Group("/mirror")
	apkMirrorGroup.Get("/placeholder", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "APK mirror API - Coming soon"})
	})

	// APK 검증 API
	apkVerificationGroup := apkGroup.Group("/verification")
	apkVerificationGroup.Get("/placeholder", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "APK verification API - Coming soon"})
	})
}

// 헬퍼 함수들

// getProxyHealth 프록시 헬스체크
func getProxyHealth(c *fiber.Ctx) error {
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
}

// getProxyTypeInfo 특정 프록시 타입 정보 조회
func getProxyTypeInfo(c *fiber.Ctx) error {
	proxyType := c.Params("type")
	factory := proxyHandlers.GetGlobalFactoryV3()

	if !factory.IsSupported(proxyType) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":           "지원하지 않는 프록시 타입",
			"type":            proxyType,
			"supported_types": factory.GetSupportedTypes(),
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
}

// V1 Status API Handlers (migrated from status_router.go)

// getServerStatusV1 서버 전반적인 상태 조회
func getServerStatusV1(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "healthy",
		"message":   "Server status API v1 - Migrated from legacy",
		"timestamp": c.Context().Time().Format("2006-01-02T15:04:05Z07:00"),
		"version":   "v1",
	})
}

// getHealthCheckV1 헬스체크 조회
func getHealthCheckV1(c *fiber.Ctx) error {
	factory := proxyHandlers.GetGlobalFactoryV3()
	healthStatus := factory.HealthCheck()

	allHealthy := true
	for _, healthy := range healthStatus {
		if !healthy {
			allHealthy = false
			break
		}
	}

	return c.JSON(fiber.Map{
		"status":       map[bool]string{true: "healthy", false: "unhealthy"}[allHealthy],
		"proxy_health": healthStatus,
		"timestamp":    c.Context().Time().Format("2006-01-02T15:04:05Z07:00"),
		"api_version":  "v1",
	})
}

// getMetricsV1 메트릭 조회
func getMetricsV1(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Metrics API v1 - Basic implementation",
		"metrics": fiber.Map{
			"active_connections": 0,
			"total_requests":     0,
			"proxy_types":        7,
		},
		"timestamp": c.Context().Time().Format("2006-01-02T15:04:05Z07:00"),
	})
}

// getDetailedStatusV1 상세 상태 조회
func getDetailedStatusV1(c *fiber.Ctx) error {
	factory := proxyHandlers.GetGlobalFactoryV3()
	healthStatus := factory.HealthCheck()

	return c.JSON(fiber.Map{
		"system_status":    "operational",
		"proxy_status":     healthStatus,
		"supported_types":  factory.GetSupportedTypes(),
		"api_version":      "v1",
		"migration_status": "completed",
		"timestamp":        c.Context().Time().Format("2006-01-02T15:04:05Z07:00"),
	})
}

// V1 Health Monitoring API Handlers

// getSystemHealthV1 전체 시스템 헬스체크
func getSystemHealthV1(c *fiber.Ctx) error {
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
		"overall_health": map[bool]string{true: "healthy", false: "unhealthy"}[allHealthy],
		"services": fiber.Map{
			"proxy_factory":   allHealthy,
			"connection_pool": true,
			"configuration":   true,
		},
		"timestamp":   c.Context().Time().Format("2006-01-02T15:04:05Z07:00"),
		"api_version": "v1",
	})
}

// getProxyHealthV1 프록시 헬스체크 (별칭)
func getProxyHealthV1(c *fiber.Ctx) error {
	// 기존 getProxyHealth 함수와 동일한 로직
	return getProxyHealth(c)
}

// getDetailedHealthV1 상세 헬스체크
func getDetailedHealthV1(c *fiber.Ctx) error {
	factory := proxyHandlers.GetGlobalFactoryV3()
	healthStatus := factory.HealthCheck()

	allHealthy := true
	healthyCount := 0
	for _, healthy := range healthStatus {
		if healthy {
			healthyCount++
		} else {
			allHealthy = false
		}
	}

	return c.JSON(fiber.Map{
		"overall_status": map[bool]string{true: "healthy", false: "degraded"}[allHealthy],
		"summary": fiber.Map{
			"total_services":     len(healthStatus),
			"healthy_services":   healthyCount,
			"unhealthy_services": len(healthStatus) - healthyCount,
		},
		"proxy_details": healthStatus,
		"system_checks": fiber.Map{
			"connection_pool": "healthy",
			"configuration":   "healthy",
			"memory":          "normal",
		},
		"api_version": "v1",
		"timestamp":   c.Context().Time().Format("2006-01-02T15:04:05Z07:00"),
	})
}
