package routers

import (
	"github.com/gofiber/fiber/v2"
)

// SetupAPIv1Routes sets up API v1 routes for WebUI integration
func SetupAPIv1Routes(app *fiber.App, cfg interface{}) {
	api := app.Group("/api/v1")

	// System information
	api.Get("/system/info", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"version": "1.0.0",
			"edition": getEdition(),
			"license": fiber.Map{
				"type":       "community",
				"expires_at": "",
				"features":   getFeatures(),
			},
		})
	})

	// Statistics (mock data for now)
	api.Get("/stats", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"hit_rate":       85.5,
			"total_requests": 12345,
			"cache_size_mb":  1024.0,
			"hits":           10494,
			"misses":         1851,
		})
	})

	// Package Managers (simplified)
	api.Get("/pm", func(c *fiber.Ctx) error {
		return c.JSON([]fiber.Map{
			{"name": "maven", "enabled": true, "url": "/maven", "cache_enabled": true},
			{"name": "npm", "enabled": true, "url": "/npm", "cache_enabled": true},
			{"name": "docker", "enabled": true, "url": "/v2", "cache_enabled": true},
			{"name": "pypi", "enabled": true, "url": "/pypi", "cache_enabled": true},
		})
	})

	api.Post("/pm/:name/toggle", func(c *fiber.Ctx) error {
		name := c.Params("name")
		return c.JSON(fiber.Map{
			"name":    name,
			"enabled": true,
			"message": "Package manager toggled successfully",
		})
	})
}

// getFeatures returns available features based on edition
func getFeatures() []string {
	edition := getEdition()

	switch edition {
	case "enterprise":
		return []string{
			"ldap_auth",
			"saml_auth",
			"rbac",
			"audit_log",
			"advanced_analytics",
		}
	case "cloud":
		return []string{
			"ldap_auth",
			"saml_auth",
			"rbac",
			"audit_log",
			"advanced_analytics",
			"multitenancy",
			"billing",
		}
	default:
		return []string{}
	}
}
