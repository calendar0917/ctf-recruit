package submission

import (
	apperrors "ctf-recruit/backend/internal/errors"
	"ctf-recruit/backend/internal/middleware"
	"ctf-recruit/backend/internal/modules/auth"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(v1 fiber.Router, authService *auth.Service) {
	submissions := v1.Group("/submissions", middleware.Auth(authService))
	submissions.Post("", h.Create)
}

func (h *Handler) Create(c *fiber.Ctx) error {
	var req CreateSubmissionRequest
	if err := c.BodyParser(&req); err != nil {
		return apperrors.BadRequest("SUBMISSION_INVALID_PAYLOAD", "Invalid request payload")
	}

	userID, ok := c.Locals(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		return apperrors.Unauthorized("AUTH_INVALID_TOKEN", "Invalid token")
	}

	role, ok := c.Locals(middleware.RoleKey).(auth.Role)
	if !ok {
		return apperrors.Unauthorized("AUTH_INVALID_TOKEN", "Invalid token")
	}

	resp, err := h.service.Submit(c.UserContext(), userID, role, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}
