package routers

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"proxynd/logging"
)

// LegacyCompatibilityRouter 기존 엔드포인트의 호환성을 유지하는 라우터
func LegacyCompatibilityRouter(app *fiber.App) {
	logger := logging.GetLogger()

	// 프록시 엔드포인트 호환성 유지
	setupProxyCompatibility(app, logger)

	// API 엔드포인트 호환성 유지
	setupAPICompatibility(app, logger)

	// 인증 엔드포인트 호환성 유지
	setupAuthCompatibility(app, logger)
}

// setupProxyCompatibility 프록시 엔드포인트 호환성 설정
func setupProxyCompatibility(app *fiber.App, logger logging.Logger) {
	// 기존 /proxy/:type/* 엔드포인트들을 /api/v1/proxy/:type/*로 리디렉트
	app.All("/proxy/:type/*", func(c *fiber.Ctx) error {
		proxyType := c.Params("type")
		path := c.Params("*")

		// Deprecation 헤더 추가
		c.Set("X-API-Deprecated", "true")
		c.Set("X-API-Deprecated-Info", "Use /api/v1/proxy/:type/* instead")
		c.Set("X-API-Migration-Guide", "https://docs.proxynd.io/api/migration")

		// 로깅
		logger.Warn("레거시 프록시 엔드포인트 사용됨",
			logging.F("original_path", c.OriginalURL()),
			logging.F("proxy_type", proxyType),
			logging.F("path", path),
			logging.F("method", c.Method()),
			logging.F("user_agent", c.Get("User-Agent")),
			logging.F("ip", c.IP()),
		)

		// 새로운 v1 엔드포인트로 내부 리디렉트 (프록시)
		newPath := "/api/v1/proxy/" + proxyType + "/" + path

		// 내부 요청 생성하여 새 엔드포인트로 전달
		return c.Redirect(newPath, fiber.StatusMovedPermanently)
	})

	// V3 프록시 엔드포인트는 deprecated 표시만 하고 유지
	app.All("/v3/proxy/:type/*", func(c *fiber.Ctx) error {
		// Deprecation 헤더 추가
		c.Set("X-API-Deprecated", "true")
		c.Set("X-API-Deprecated-Info", "Use /api/v1/proxy/:type/* instead")
		c.Set("X-API-Migration-Guide", "https://docs.proxynd.io/api/migration")

		// 로깅
		logger.Warn("V3 프록시 엔드포인트 사용됨",
			logging.F("original_path", c.OriginalURL()),
			logging.F("method", c.Method()),
			logging.F("ip", c.IP()),
		)

		// 다음 핸들러로 계속 (기존 V3 라우터가 처리)
		return c.Next()
	})
}

// setupAPICompatibility API 엔드포인트 호환성 설정
func setupAPICompatibility(app *fiber.App, logger logging.Logger) {
	// 설정 API 호환성
	setupConfigAPICompatibility(app, logger)

	// 캐시 API 호환성
	setupCacheAPICompatibility(app, logger)

	// 상태 API 호환성
	setupStatusAPICompatibility(app, logger)

	// 메트릭 API 호환성
	setupMetricsAPICompatibility(app, logger)

	// 사용자 API 호환성
	setupUserAPICompatibility(app, logger)

	// APK API 호환성
	setupAPKAPICompatibility(app, logger)

	// 기타 API 호환성
	setupMiscAPICompatibility(app, logger)
}

// setupConfigAPICompatibility 설정 API 호환성
func setupConfigAPICompatibility(app *fiber.App, logger logging.Logger) {
	app.All("/api/config/*", createCompatibilityHandler(
		"/api/v1/admin/config",
		"Config API has moved to /api/v1/admin/config/*",
		logger,
	))
}

// setupCacheAPICompatibility 캐시 API 호환성
func setupCacheAPICompatibility(app *fiber.App, logger logging.Logger) {
	app.All("/api/cache/*", createCompatibilityHandler(
		"/api/v1/admin/cache",
		"Cache API has moved to /api/v1/admin/cache/*",
		logger,
	))
}

// setupStatusAPICompatibility 상태 API 호환성
func setupStatusAPICompatibility(app *fiber.App, logger logging.Logger) {
	app.All("/api/status/*", createCompatibilityHandler(
		"/api/v1/admin/status",
		"Status API has moved to /api/v1/admin/status/*",
		logger,
	))
}

// setupMetricsAPICompatibility 메트릭 API 호환성
func setupMetricsAPICompatibility(app *fiber.App, logger logging.Logger) {
	// /metrics (Prometheus)는 유지
	// /api/metrics/* 만 이전
	app.All("/api/metrics/*", createCompatibilityHandler(
		"/api/v1/monitoring/metrics",
		"Metrics API has moved to /api/v1/monitoring/metrics/*",
		logger,
	))
}

// setupUserAPICompatibility 사용자 API 호환성
func setupUserAPICompatibility(app *fiber.App, logger logging.Logger) {
	app.All("/api/user/*", createCompatibilityHandler(
		"/api/v1/users",
		"User API has moved to /api/v1/users/*",
		logger,
	))
}

// setupAPKAPICompatibility APK API 호환성
func setupAPKAPICompatibility(app *fiber.App, logger logging.Logger) {
	app.All("/api/apk/*", createCompatibilityHandler(
		"/api/v1/apk",
		"APK API has moved to /api/v1/apk/*",
		logger,
	))
}

// setupMiscAPICompatibility 기타 API 호환성
func setupMiscAPICompatibility(app *fiber.App, logger logging.Logger) {
	// 검색 API
	app.All("/api/search", createCompatibilityHandler(
		"/api/v1/search",
		"Search API has moved to /api/v1/search",
		logger,
	))

	// 테스트 API
	app.All("/api/test/*", createCompatibilityHandler(
		"/api/v1/test",
		"Test API has moved to /api/v1/test/*",
		logger,
	))
}

// setupAuthCompatibility 인증 엔드포인트 호환성 설정
func setupAuthCompatibility(app *fiber.App, logger logging.Logger) {
	// JWT 인증 API
	app.All("/auth/jwt/*", createCompatibilityHandler(
		"/api/v1/auth/jwt",
		"JWT Auth API has moved to /api/v1/auth/jwt/*",
		logger,
	))

	// OAuth2 인증 API (기본 /auth/*)
	app.All("/auth/*", func(c *fiber.Ctx) error {
		// OAuth2는 특수 처리 (JWT가 아닌 경우)
		path := c.Path()

		// JWT 경로가 아닌 경우만 처리
		if !strings.Contains(path, "/auth/jwt") {
			return createCompatibilityHandler(
				"/api/v1/auth/oauth",
				"OAuth2 Auth API has moved to /api/v1/auth/oauth/*",
				logger,
			)(c)
		}

		// JWT 경로인 경우 다음 핸들러로
		return c.Next()
	})
}

// createCompatibilityHandler 호환성 핸들러 생성 헬퍼
func createCompatibilityHandler(newBasePath, message string, logger logging.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		originalPath := c.Path()

		// Deprecation 헤더 추가
		c.Set("X-API-Deprecated", "true")
		c.Set("X-API-Deprecated-Info", message)
		c.Set("X-API-Migration-Guide", "https://docs.proxynd.io/api/migration")

		// 로깅
		logger.Warn("레거시 API 엔드포인트 사용됨",
			logging.F("original_path", originalPath),
			logging.F("new_base_path", newBasePath),
			logging.F("method", c.Method()),
			logging.F("user_agent", c.Get("User-Agent")),
			logging.F("ip", c.IP()),
		)

		// 경로 매핑
		newPath := mapLegacyPathToV1(originalPath, newBasePath)

		// 301 리디렉트
		return c.Redirect(newPath, fiber.StatusMovedPermanently)
	}
}

// mapLegacyPathToV1 레거시 경로를 v1 경로로 매핑
func mapLegacyPathToV1(originalPath, newBasePath string) string {
	// 예: /api/config/maven -> /api/v1/admin/config/maven
	// 예: /api/cache/clear -> /api/v1/admin/cache/clear

	// 기존 경로에서 /api/ 이후 부분 추출
	parts := strings.Split(originalPath, "/")
	if len(parts) >= 3 && parts[1] == "api" {
		// /api/config/maven -> config/maven
		subPath := strings.Join(parts[3:], "/")
		if subPath != "" {
			return newBasePath + "/" + subPath
		}
	}

	return newBasePath
}

// LegacyAPIInfo 레거시 API 정보 제공 엔드포인트
func LegacyAPIInfo(app *fiber.App) {
	app.Get("/api/migration-info", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message":         "API 마이그레이션 정보",
			"migration_guide": "https://docs.proxynd.io/api/migration",
			"deprecated_endpoints": map[string]string{
				"/proxy/*":       "/api/v1/proxy/*",
				"/v3/proxy/*":    "/api/v1/proxy/*",
				"/api/config/*":  "/api/v1/admin/config/*",
				"/api/cache/*":   "/api/v1/admin/cache/*",
				"/api/status/*":  "/api/v1/admin/status/*",
				"/api/metrics/*": "/api/v1/monitoring/metrics/*",
				"/api/user/*":    "/api/v1/users/*",
				"/api/apk/*":     "/api/v1/apk/*",
				"/api/search":    "/api/v1/search",
				"/api/test/*":    "/api/v1/test/*",
				"/auth/jwt/*":    "/api/v1/auth/jwt/*",
				"/auth/*":        "/api/v1/auth/oauth/*",
			},
			"current_version": "v1",
			"deprecation_schedule": map[string]string{
				"2025-Q1": "Deprecation warnings added",
				"2025-Q2": "Legacy endpoints still supported with warnings",
				"2025-Q3": "Legacy endpoints return 410 Gone",
				"2025-Q4": "Legacy endpoints removed",
			},
		})
	})
}
