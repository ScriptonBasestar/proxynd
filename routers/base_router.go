package routers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"

	"proxynd/configs"
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
		mvnSite.ReadConfig()
		aptSite := configs.AptProxyConfig{}
		aptSite.ReadConfig()
		return c.Render("dashboard", fiber.Map{
			"mavenProxy": mvnSite.Proxies,
			"aptProxy":   aptSite.Proxies,
		})
	})

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
