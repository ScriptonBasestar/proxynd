package routers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/template/html/v2"
	"proxynd/configs"
)

// BaseRouter function will perform all route operations
func BaseRouter() *fiber.App {
	// 템플릿 엔진 설정
	engine := html.New("./templates", ".html")

	// Fiber 앱 생성
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	// 로거 미들웨어 사용
	app.Use(logger.New())

	//Giving access to storage folder
	//app.Static("/storage", "./storage")

	//Giving access to template folder
	//app.Static("/templates", "./templates")

	app.Get("/", func(c *fiber.Ctx) error {
		mvnSite := configs.MavenProxyConfig{}
		mvnSite.ReadConfig()
		aptSite := configs.AptProxyConfig{}
		aptSite.ReadConfig()
		return c.Render("dashboard", fiber.Map{
			"mavenProxy": mvnSite.Proxies,
			"aptProxy":   aptSite.Proxies,
		})
	})

	// CORS 미들웨어 (필요시 활성화)
	//app.Use(func(c *fiber.Ctx) error {
	//	// add header Access-Control-Allow-Origin
	//	c.Set("Content-Type", "application/json")
	//	c.Set("Access-Control-Allow-Origin", "*")
	//	c.Set("Access-Control-Max-Age", "86400")
	//	c.Set("Access-Control-Allow-Methods", "POST, GET, PUT, DELETE, UPDATE")
	//	c.Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Max")
	//	c.Set("Access-Control-Allow-Credentials", "true")
	//
	//	if c.Method() == "OPTIONS" {
	//		return c.SendStatus(200)
	//	}
	//	return c.Next()
	//})

	return app
}
