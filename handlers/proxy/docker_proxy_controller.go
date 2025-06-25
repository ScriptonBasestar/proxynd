package proxy

import (
	"github.com/gofiber/fiber/v2"
	"log"
)

// DockerProxy Docker 레지스트리 프록시 핸들러
func DockerProxy(c *fiber.Ctx) error {
	log.Printf("Access proxy docker\n")
	
	// TODO: Docker 레지스트리 v2 프록시 구현
	return c.Status(fiber.StatusNotImplemented).SendString("Docker proxy not implemented yet")
}