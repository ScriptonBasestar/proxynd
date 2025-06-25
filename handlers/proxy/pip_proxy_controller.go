package proxy

import (
	"github.com/gofiber/fiber/v2"
	"log"
)

// PipProxy pip 패키지 매니저 프록시 핸들러
func PipProxy(c *fiber.Ctx) error {
	log.Printf("Access proxy pip\n")
	
	// TODO: pip 프록시 구현
	return c.Status(fiber.StatusNotImplemented).SendString("PIP proxy not implemented yet")
}