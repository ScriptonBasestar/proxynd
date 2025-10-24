package routers

import (
	"github.com/gofiber/fiber/v2"

	"proxynd/internal/adapters/http/fiber/handlers/ansible"
)

// RegisterAnsibleRoutes registers Ansible Galaxy v3 API routes
func RegisterAnsibleRoutes(app *fiber.App, handler *ansible.AnsibleHandler) {
	// Galaxy v3 API base path
	v3 := app.Group("/api/v3")

	// Collection upload (requires authentication in production)
	// POST /api/v3/collections/
	v3.Post("/collections/", handler.UploadCollection)

	// Collection download
	// GET /api/v3/collections/{namespace}/{name}/versions/{version}/
	v3.Get("/collections/:namespace/:name/versions/:version/", handler.DownloadCollection)

	// Collection list
	// GET /api/v3/collections/
	v3.Get("/collections/", handler.ListCollections)

	// Optional: Collection detail (metadata only, not download)
	// GET /api/v3/collections/{namespace}/{name}/
	// v3.Get("/collections/:namespace/:name/", handler.GetCollectionDetail)

	// Optional: Delete collection version (requires admin auth)
	// DELETE /api/v3/collections/{namespace}/{name}/versions/{version}/
	// v3.Delete("/collections/:namespace/:name/versions/:version/", authMiddleware, handler.DeleteCollection)
}
