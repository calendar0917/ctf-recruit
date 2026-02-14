package middleware

import (
	apperrors "ctf-recruit/backend/internal/errors"
	"ctf-recruit/backend/internal/modules/auth"

	"github.com/gofiber/fiber/v2"
)

const (
	UserIDKey = "userId"
	RoleKey   = "role"
)

func Auth(service *auth.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := auth.ExtractBearerToken(c.Get("Authorization"))
		if token == "" {
			return apperrors.Unauthorized("AUTH_MISSING_TOKEN", "Missing authorization token")
		}

		claims, err := service.ParseAccessToken(token)
		if err != nil {
			return err
		}

		c.Locals(UserIDKey, claims.UserID)
		c.Locals(RoleKey, claims.Role)
		return c.Next()
	}
}
