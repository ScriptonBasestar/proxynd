package routers

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"

	"proxynd/cache"
	"proxynd/internal/config"
	configService "proxynd/internal/services/config"
)

// Global cache manager reference (set by main app)
var globalCacheManager *cache.Manager

// SetCacheManager sets the global cache manager for API access
func SetCacheManager(cm *cache.Manager) {
	globalCacheManager = cm
}

// SetupAPIv1Routes sets up API v1 routes for WebUI integration
func SetupAPIv1Routes(app *fiber.App, cfg interface{}) {
	api := app.Group("/api/v1")

	// Try to cast config to RootConfig for real data access
	var rootCfg *config.RootConfig
	if cfg != nil {
		if rc, ok := cfg.(*config.RootConfig); ok {
			rootCfg = rc
		}
	}

	// System information
	api.Get("/system/info", func(c *fiber.Ctx) error {
		version := os.Getenv("PROXYND_VERSION")
		if version == "" {
			version = "1.0.0"
		}

		return c.JSON(fiber.Map{
			"version": version,
			"edition": getEdition(),
			"license": fiber.Map{
				"type":       getLicenseType(),
				"expires_at": "",
				"features":   getFeatures(),
			},
		})
	})

	// Statistics (real data if available, fallback to mock)
	api.Get("/stats", func(c *fiber.Ctx) error {
		stats := getSystemCacheStats()
		return c.JSON(stats)
	})

	// Package Managers (from config if available)
	api.Get("/pm", func(c *fiber.Ctx) error {
		pms := getPackageManagers(rootCfg)
		return c.JSON(pms)
	})

	api.Post("/pm/:name/toggle", func(c *fiber.Ctx) error {
		return togglePackageManager(c, rootCfg)
	})
}

// getLicenseType returns the license type based on edition
func getLicenseType() string {
	edition := getEdition()
	switch edition {
	case "enterprise":
		return "enterprise"
	case "cloud":
		return "cloud"
	default:
		return "community"
	}
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

// getPackageManagers returns list of package managers from config
func getPackageManagers(cfg *config.RootConfig) []fiber.Map {
	// Default package managers if config not available
	defaultPMs := []fiber.Map{
		{"name": "maven", "enabled": true, "url": "/maven", "cache_enabled": true},
		{"name": "npm", "enabled": true, "url": "/npm", "cache_enabled": true},
		{"name": "docker", "enabled": true, "url": "/v2", "cache_enabled": true},
		{"name": "pypi", "enabled": true, "url": "/pypi", "cache_enabled": true},
		{"name": "apt", "enabled": true, "url": "/apt", "cache_enabled": true},
		{"name": "yum", "enabled": true, "url": "/yum", "cache_enabled": true},
		{"name": "apk", "enabled": true, "url": "/apk", "cache_enabled": true},
	}

	// If no config, return defaults
	if cfg == nil {
		return defaultPMs
	}

	// Build list from config
	pms := make([]fiber.Map, 0)

	// Check Maven
	if cfg.Registries.Maven.Enabled {
		pms = append(pms, fiber.Map{
			"name":          "maven",
			"enabled":       true,
			"url":           "/maven",
			"cache_enabled": true,
		})
	}

	// Check NPM
	if cfg.Registries.NPM.Enabled {
		pms = append(pms, fiber.Map{
			"name":          "npm",
			"enabled":       true,
			"url":           "/npm",
			"cache_enabled": true,
		})
	}

	// Check Docker
	if cfg.Registries.Docker.Enabled {
		pms = append(pms, fiber.Map{
			"name":          "docker",
			"enabled":       true,
			"url":           "/v2",
			"cache_enabled": true,
		})
	}

	// Check PyPI
	if cfg.Registries.PyPI.Enabled {
		pms = append(pms, fiber.Map{
			"name":          "pypi",
			"enabled":       true,
			"url":           "/pypi",
			"cache_enabled": true,
		})
	}

	// Check APT
	if cfg.Registries.APT.Enabled {
		pms = append(pms, fiber.Map{
			"name":          "apt",
			"enabled":       true,
			"url":           "/apt",
			"cache_enabled": true,
		})
	}

	// If no PMs are enabled in config, return defaults
	if len(pms) == 0 {
		return defaultPMs
	}

	return pms
}

// togglePackageManager toggles the enabled state of a package manager
func togglePackageManager(c *fiber.Ctx, cfg *config.RootConfig) error {
	name := c.Params("name")

	// Validate package manager name
	validPMs := map[string]bool{
		"maven":  true,
		"npm":    true,
		"docker": true,
		"pypi":   true,
		"apt":    true,
		"yum":    true,
		"apk":    true,
	}

	if !validPMs[name] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_package_manager",
			"message": fmt.Sprintf("Unknown package manager: %s", name),
			"valid":   []string{"maven", "npm", "docker", "pypi", "apt", "yum", "apk"},
		})
	}

	// Get config service from locals
	configSvc, ok := c.Locals("configService").(configService.Service)
	if !ok || configSvc == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error":   "config_service_unavailable",
			"message": "Configuration service is not available",
		})
	}

	// Get RootConfig from config service
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rootCfg, err := configSvc.GetRootConfig(ctx)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error":   "config_load_failed",
			"message": fmt.Sprintf("Failed to load configuration: %v", err),
		})
	}

	// Parse request body for explicit enable/disable (optional)
	type ToggleRequest struct {
		Enabled *bool `json:"enabled"` // Optional: if nil, toggle current state
	}

	var req ToggleRequest
	if err := c.BodyParser(&req); err != nil && len(c.Body()) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid JSON body",
		})
	}

	// Get current state and toggle
	var currentState, newState bool
	var registryPtr *bool

	switch name {
	case "maven":
		registryPtr = &rootCfg.Registries.Maven.Enabled
		currentState = rootCfg.Registries.Maven.Enabled
	case "npm":
		registryPtr = &rootCfg.Registries.NPM.Enabled
		currentState = rootCfg.Registries.NPM.Enabled
	case "docker":
		registryPtr = &rootCfg.Registries.Docker.Enabled
		currentState = rootCfg.Registries.Docker.Enabled
	case "pypi":
		registryPtr = &rootCfg.Registries.PyPI.Enabled
		currentState = rootCfg.Registries.PyPI.Enabled
	case "apt":
		registryPtr = &rootCfg.Registries.APT.Enabled
		currentState = rootCfg.Registries.APT.Enabled
	case "yum", "apk":
		// YUM and APK not yet in RootConfig, return not implemented
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
			"error":   "not_implemented",
			"message": fmt.Sprintf("Package manager '%s' configuration not yet implemented", name),
		})
	}

	// Determine new state
	if req.Enabled != nil {
		newState = *req.Enabled
	} else {
		newState = !currentState
	}

	// Update the state
	*registryPtr = newState

	// Persist the change to disk
	saveCtx, saveCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer saveCancel()

	if err := configSvc.SaveConfig(saveCtx); err != nil {
		// Rollback the change
		*registryPtr = currentState

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "save_failed",
			"message": fmt.Sprintf("Failed to persist configuration: %v", err),
		})
	}

	// TODO: Notify plugin system about the state change
	// This will be implemented in the next step

	return c.JSON(fiber.Map{
		"name":           name,
		"enabled":        newState,
		"previous_state": currentState,
		"message":        fmt.Sprintf("Package manager '%s' %s successfully", name, map[bool]string{true: "enabled", false: "disabled"}[newState]),
		"persisted":      true,
	})
}

// getSystemCacheStats returns system-wide cache statistics (real or mock)
func getSystemCacheStats() fiber.Map {
	// Try to get real cache statistics
	if globalCacheManager != nil {
		stats := globalCacheManager.GetStats()

		totalRequests := stats.Hits + stats.Misses
		sizeMB := float64(stats.Size) / (1024.0 * 1024.0)

		// Calculate hit rate
		hitRate := 0.0
		if totalRequests > 0 {
			hitRate = (float64(stats.Hits) / float64(totalRequests)) * 100.0
		}

		return fiber.Map{
			"hit_rate":       hitRate,
			"total_requests": totalRequests,
			"cache_size_mb":  sizeMB,
			"hits":           stats.Hits,
			"misses":         stats.Misses,
		}
	}

	// Fallback to mock data if cache manager not available
	return fiber.Map{
		"hit_rate":       85.5,
		"total_requests": 12345,
		"cache_size_mb":  1024.0,
		"hits":           10494,
		"misses":         1851,
	}
}
