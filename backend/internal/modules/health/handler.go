package health

import "github.com/gofiber/fiber/v2"

type Handler struct{}

func NewHandler() Handler {
	return Handler{}
}

func (h Handler) GetHealth(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "ok",
		"service": "ctf-api",
	})
}
