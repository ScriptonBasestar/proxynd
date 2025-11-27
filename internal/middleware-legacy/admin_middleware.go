package middlewares

import (
	"github.com/gofiber/fiber/v2"
)

// AdminOnlyMiddleware ensures only authenticated users with admin role can access the route
// This middleware combines JWT authentication check with admin role verification
//
// Usage:
//
//	adminGroup.Use(AdminOnlyMiddleware())
func AdminOnlyMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if user is authenticated (JWT claims should be in context)
		_, _, roles, ok := GetUserFromContext(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required",
				"code":  "AUTH_REQUIRED",
			})
		}

		// Check if user has admin role
		hasAdmin := false
		for _, role := range roles {
			if role == "admin" {
				hasAdmin = true
				break
			}
		}

		if !hasAdmin {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Admin role required",
				"code":  "AUTH_ADMIN_REQUIRED",
			})
		}

		return c.Next()
	}
}

// AdminOnlyMiddlewareWithJWT combines JWT authentication and admin role check in one middleware
// This is a convenience function that applies both JWT verification and admin role check
//
// Usage:
//
//	adminGroup.Use(AdminOnlyMiddlewareWithJWT(jwtConfig))
func AdminOnlyMiddlewareWithJWT(config JWTConfig) fiber.Handler {
	// First apply JWT middleware, then admin check
	jwtMiddleware := JWTMiddleware(config)

	return func(c *fiber.Ctx) error {
		// First, verify JWT token
		if err := jwtMiddleware(c); err != nil {
			return err
		}

		// Then, verify admin role
		adminMiddleware := AdminOnlyMiddleware()
		return adminMiddleware(c)
	}
}
