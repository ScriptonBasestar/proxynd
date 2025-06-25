package middlewares

import "github.com/gofiber/fiber/v2"

/*
UserMiddlewares function to add auth
*/
func UserMiddlewares() fiber.Handler {
	return func(c *fiber.Ctx) error {

		//Code for middlewares

		return c.Next()
	}
}

func ProxyMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {

		//Code for middlewares

		return c.Next()
	}
}