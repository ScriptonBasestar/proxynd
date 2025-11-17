// Package enterprise provides enterprise license validation middleware
package enterprise

import (
	"github.com/gofiber/fiber/v2"
	"proxynd/internal/enterprise/license"
)

// EnterpriseFeatures interface defines methods for enterprise feature checking
type EnterpriseFeatures interface {
	IsEnabled() bool
	HasFeature(feature string) bool
}

// LicenseMiddleware validates enterprise license for protected routes
type LicenseMiddleware struct {
	features EnterpriseFeatures
}

// NewLicenseMiddleware creates a new license validation middleware
func NewLicenseMiddleware(features EnterpriseFeatures) *LicenseMiddleware {
	return &LicenseMiddleware{
		features: features,
	}
}

// RequireEnterprise validates that user has a valid enterprise license
func (m *LicenseMiddleware) RequireEnterprise() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if m.features == nil || !m.features.IsEnabled() {
			return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":        "LICENSE_REQUIRED",
					"message":     "This endpoint requires an enterprise license",
					"upgrade_url": "https://proxynd.io/pricing",
				},
				"metadata": fiber.Map{
					"timestamp":  fiber.Map{},
					"request_id": c.Locals("requestid"),
				},
			})
		}
		return c.Next()
	}
}

// RequireFeature validates that a specific feature is enabled
func (m *LicenseMiddleware) RequireFeature(feature string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if m.features == nil || !m.features.HasFeature(feature) {
			return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":        "FEATURE_NOT_LICENSED",
					"message":     "This feature requires an enterprise license",
					"feature":     feature,
					"upgrade_url": "https://proxynd.io/pricing",
				},
				"metadata": fiber.Map{
					"timestamp":  fiber.Map{},
					"request_id": c.Locals("requestid"),
				},
			})
		}
		return c.Next()
	}
}

// RequireAnyFeature validates that at least one of the specified features is enabled
func (m *LicenseMiddleware) RequireAnyFeature(features ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if m.features == nil {
			return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "LICENSE_REQUIRED",
					"message": "This endpoint requires an enterprise license",
				},
			})
		}

		for _, feature := range features {
			if m.features.HasFeature(feature) {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":        "FEATURE_NOT_LICENSED",
				"message":     "This endpoint requires one of the specified features",
				"features":    features,
				"upgrade_url": "https://proxynd.io/pricing",
			},
		})
	}
}

// SkipInDevelopment wraps middleware to skip in development mode
func (m *LicenseMiddleware) SkipInDevelopment(handler fiber.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip license check in development mode
		if c.Get("X-Dev-Mode") == "true" {
			return c.Next()
		}
		return handler(c)
	}
}

// NewDevModeLicenseMiddleware creates a license middleware that skips validation in dev mode
// This is useful for local development and testing
func NewDevModeLicenseMiddleware() *LicenseMiddleware {
	// In dev mode, return middleware with nil features
	// which will allow all requests to pass through
	return &LicenseMiddleware{
		features: &mockEnterpriseFeatures{},
	}
}

// mockEnterpriseFeatures is a mock implementation for development
// that always returns true for all feature checks
type mockEnterpriseFeatures struct{}

func (m *mockEnterpriseFeatures) IsEnabled() bool {
	return true
}

func (m *mockEnterpriseFeatures) HasFeature(feature string) bool {
	return true
}

// FeatureMap returns a map of feature IDs to their enabled status
func (m *LicenseMiddleware) FeatureMap() map[string]bool {
	if m.features == nil {
		return map[string]bool{}
	}

	featureMap := make(map[string]bool)
	for id := range license.FeatureCatalog {
		featureMap[id] = m.features.HasFeature(id)
	}
	return featureMap
}
