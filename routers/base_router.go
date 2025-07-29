package routers

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"

	"proxynd/handlers"
	"proxynd/internal/config"
	"proxynd/logging"
)

// BaseRouter function will perform all route operations
func BaseRouter() *fiber.App {
	// Setup template engine
	engine := html.New("./templates", ".html")

	// Add template helper functions
	engine.AddFunc("sub", func(a, b int) int {
		return a - b
	})
	engine.AddFunc("add", func(a, b int) int {
		return a + b
	})
	engine.AddFunc("hasSuffix", func(s, suffix string) bool {
		return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
	})

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
		mvnSite := config.MavenProxyConfig{}
		if err := mvnSite.ReadConfig(); err != nil {
			log.Printf("Warning: Failed to read Maven config for dashboard: %v", err)
		}
		aptSite := config.AptProxyConfig{}
		if err := aptSite.ReadConfig(); err != nil {
			log.Printf("Warning: Failed to read APT config for dashboard: %v", err)
		}

		// 요청의 실제 호스트와 포트 정보를 기반으로 baseURL 구성
		scheme := "http"
		if c.Protocol() == "https" {
			scheme = "https"
		}

		// 환경변수에서 포트를 가져오거나 기본값 사용
		port := "8080" // 기본 포트
		if envPort := os.Getenv("SERVER_PORT"); envPort != "" {
			port = envPort
		}

		// 간단하고 확실한 방법: localhost:port 형식으로 고정
		baseURL := scheme + "://localhost:" + port

		// Maven과 APT 프록시 URL을 올바르게 구성
		mavenProxyURL := baseURL + "/proxy/maven"
		aptProxyURL := baseURL + "/proxy/apt"

		return c.Render("dashboard", fiber.Map{
			"mavenProxy":    mvnSite.Proxies,
			"aptProxy":      aptSite.Proxies,
			"mavenPath":     mvnSite.Path,
			"aptPath":       aptSite.Path,
			"baseURL":       baseURL,
			"mavenProxyURL": mavenProxyURL,
			"aptProxyURL":   aptProxyURL,
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
