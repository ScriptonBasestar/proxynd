package routers

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"

	"proxynd/internal/logging"
)

// SetupWebUIRoutes configures WebUI static file serving for Enterprise/Cloud editions
func SetupWebUIRoutes(app *fiber.App, edition string) {
	logger := logging.GetLogger()

	// Only serve WebUI for Enterprise and Cloud editions
	if edition != "enterprise" && edition != "cloud" {
		logger.Info("Skipping WebUI setup (Core edition uses templates)")
		return
	}

	// Get WebUI path from environment or use default
	webuiPath := os.Getenv("WEBUI_PATH")
	if webuiPath == "" {
		webuiPath = "../proxynd-webui/apps/admin/dist"
	}

	// Check if WebUI dist directory exists
	absPath, err := filepath.Abs(webuiPath)
	if err != nil {
		logger.Warn("Failed to resolve WebUI path",
			logging.String("path", webuiPath))
		return
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		logger.Warn("WebUI dist directory not found",
			logging.String("path", absPath),
			logging.String("hint", "Run 'cd proxynd-webui && pnpm build'"))
		return
	}

	logger.Info("Setting up WebUI static file serving",
		logging.String("edition", edition),
		logging.String("path", absPath))

	// Serve WebUI static files at /dashboard to avoid conflicting with API routes
	app.Use("/dashboard", filesystem.New(filesystem.Config{
		Root:         http.Dir(absPath),
		Index:        "index.html",
		NotFoundFile: "index.html",
		Browse:       false,
	}))

	// Redirect root to dashboard for convenience
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/dashboard", fiber.StatusMovedPermanently)
	})

	logger.Info("WebUI routes configured successfully",
		logging.String("mount_path", "/dashboard"))
}
