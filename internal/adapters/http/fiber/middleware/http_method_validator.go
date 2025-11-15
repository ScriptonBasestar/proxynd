package middlewares

import (
	"github.com/gofiber/fiber/v2"
)

// ReadOnlyMethodValidator creates a middleware that only allows GET and HEAD methods
// All other HTTP methods (POST, PUT, DELETE, PATCH, etc.) will be rejected with 405 Method Not Allowed
func ReadOnlyMethodValidator() fiber.Handler {
	return func(c *fiber.Ctx) error {
		method := c.Method()

		// Allow only GET and HEAD methods for read-only proxy
		if method == fiber.MethodGet || method == fiber.MethodHead {
			return c.Next()
		}

		// Reject all other methods with 405 Method Not Allowed
		c.Set("Allow", "GET, HEAD")
		return c.Status(fiber.StatusMethodNotAllowed).JSON(fiber.Map{
			"error":   "Method Not Allowed",
			"message": "This proxy only supports GET and HEAD methods",
			"allowed": []string{"GET", "HEAD"},
		})
	}
}
