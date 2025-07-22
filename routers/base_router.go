package routers

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"

	"proxynd/configs"
	"proxynd/handlers"
	"proxynd/logging"
)

// BaseRouter function will perform all route operations
func BaseRouter() *fiber.App {
	// Setup template engine
	engine := html.New("./templates", ".html")

	// Create Fiber app
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	// Use structured logging middleware
	app.Use(logging.RequestLogger())
	app.Use(logging.New())
	app.Use(logging.ErrorLogger())
	app.Use(logging.RecoveryLogger())

	app.Get("/", func(c *fiber.Ctx) error {
		mvnSite := configs.MavenProxyConfig{}
		if err := mvnSite.ReadConfig(); err != nil {
			log.Printf("Warning: Failed to read Maven config for dashboard: %v", err)
		}
		aptSite := configs.AptProxyConfig{}
		if err := aptSite.ReadConfig(); err != nil {
			log.Printf("Warning: Failed to read APT config for dashboard: %v", err)
		}

		// 현재 서버의 호스트와 포트 정보 가져오기
		host := c.Hostname()
		if host == "" {
			host = "localhost"
		}
		port := c.Port()
		if port == "" {
			port = "8080"
		}
		baseURL := "http://" + host + ":" + port

		return c.Render("dashboard", fiber.Map{
			"mavenProxy": mvnSite.Proxies,
			"aptProxy":   aptSite.Proxies,
			"mavenPath":  mvnSite.Path,
			"aptPath":    aptSite.Path,
			"baseURL":    baseURL,
		})
	})

	// Add search API
	app.Get("/api/search", handlers.SearchHandler)

	// Add metrics router (use default if no config)
	MetricsRouter(app, nil)

	// Add APK signature verification router
	APKVerificationRouter(app)

	// Add APK mirror selection router
	APKMirrorRouter(app)

	// Add OAuth2 authentication router
	AuthRouter(app)

	return app
}
