package middleware

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const RequestIDKey = "requestId"

func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := c.Get("X-Request-Id")
		if requestID == "" {
			requestID = fmt.Sprintf("req_%s", uuid.NewString())
		}

		c.Locals(RequestIDKey, requestID)
		c.Set("X-Request-Id", requestID)
		return c.Next()
	}
}
