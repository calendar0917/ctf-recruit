package middleware

import (
	apperrors "ctf-recruit/backend/internal/errors"
	"ctf-recruit/backend/internal/modules/auth"

	"github.com/gofiber/fiber/v2"
)

func RequireRoles(roles ...auth.Role) fiber.Handler {
	allowed := make(map[auth.Role]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(c *fiber.Ctx) error {
		role, ok := c.Locals(RoleKey).(auth.Role)
		if !ok {
			return apperrors.Forbidden("AUTH_FORBIDDEN", "Forbidden")
		}
		if _, exists := allowed[role]; !exists {
			return apperrors.Forbidden("AUTH_FORBIDDEN", "Forbidden")
		}
		return c.Next()
	}
}
