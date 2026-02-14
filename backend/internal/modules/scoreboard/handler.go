package scoreboard

import (
	apperrors "ctf-recruit/backend/internal/errors"
	"ctf-recruit/backend/internal/middleware"
	"ctf-recruit/backend/internal/modules/auth"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(v1 fiber.Router, authService *auth.Service) {
	scoreboard := v1.Group("/scoreboard", middleware.Auth(authService))
	scoreboard.Get("", h.List)
}

func (h *Handler) List(c *fiber.Ctx) error {
	limit, err := parseQueryInt(c.Query("limit"), defaultLimit)
	if err != nil {
		return apperrors.BadRequest("SCOREBOARD_INVALID_QUERY", "limit must be an integer")
	}
	offset, err := parseQueryInt(c.Query("offset"), 0)
	if err != nil {
		return apperrors.BadRequest("SCOREBOARD_INVALID_QUERY", "offset must be an integer")
	}

	resp, err := h.service.List(c.UserContext(), limit, offset)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

func parseQueryInt(value string, defaultValue int) (int, error) {
	if value == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}
