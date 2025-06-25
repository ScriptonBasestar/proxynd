package routers

import (
	"github.com/gofiber/fiber/v2"
	"os"
)

func HealthRouter(app *fiber.App) {
	// 헬스체크 엔드포인트 설정
	app.Get("/healthz", func(c *fiber.Ctx) error {
		// 환경 변수 체크
		configDir := os.Getenv("CONFIG_DIR")
		storageDir := os.Getenv("STORAGE_DIR")
		
		checks := fiber.Map{
			"status": "ok",
			"checks": fiber.Map{
				"CONFIG_DIR":  configDir != "",
				"STORAGE_DIR": storageDir != "",
			},
		}
		
		// 모든 체크가 통과했는지 확인
		allPassed := configDir != "" && storageDir != ""
		
		if allPassed {
			return c.Status(200).JSON(checks)
		}
		
		checks["status"] = "error"
		return c.Status(503).JSON(checks)
	})
}