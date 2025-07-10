package main

import (
	"log"

	"github.com/gofiber/fiber/v2"

	"proxynd/middlewares"
)

// Example showing how to integrate the security middlewares
func setupSecurityMiddlewares(app *fiber.App) {
	// 1. 보안 헤더 적용 (모든 요청)
	app.Use(middlewares.SecurityHeaders())

	// 2. Rate Limiting 적용 (모든 요청)
	app.Use(middlewares.RateLimit())

	// 3. API 그룹에 입력 검증 적용
	api := app.Group("/api")
	api.Use(middlewares.InputValidation())

	// 4. 프록시 그룹에 더 엄격한 검증 적용
	proxy := api.Group("/proxy")
	proxy.Use(middlewares.InputValidation(middlewares.StrictValidationConfig()))
	proxy.Use(middlewares.RateLimit(middlewares.StrictRateLimitConfig()))

	// 예제 라우트들
	api.Get("/status", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	proxy.Get("/:type/*", func(c *fiber.Ctx) error {
		proxyType := c.Params("type")
		packagePath := c.Params("*")

		return c.JSON(fiber.Map{
			"proxy_type": proxyType,
			"path":       packagePath,
			"message":    "Proxy request handled successfully",
		})
	})
}

func main() {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// 보안 미들웨어 설정
	setupSecurityMiddlewares(app)

	// 서버 시작
	log.Println("Starting server on :8080")
	log.Println("Test endpoints:")
	log.Println("  GET /api/status")
	log.Println("  GET /api/proxy/npm/express")
	log.Println("  GET /api/proxy/maven/org.springframework/spring-core/5.3.0")

	if err := app.Listen(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
