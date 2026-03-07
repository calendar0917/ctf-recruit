package recruitment

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
	recruitmentGroup := v1.Group("/recruitments", middleware.Auth(authService))

	recruitmentGroup.Post("", h.Create)
	recruitmentGroup.Get("", middleware.RequireRoles(auth.RoleAdmin), h.List)
	recruitmentGroup.Get("/:id", middleware.RequireRoles(auth.RoleAdmin), h.Get)
}

func (h *Handler) Create(c *fiber.Ctx) error {
	var req CreateSubmissionRequest
	if err := c.BodyParser(&req); err != nil {
		return apperrors.BadRequest("RECRUITMENT_INVALID_PAYLOAD", "Invalid request payload")
	}

	userID, _ := c.Locals(middleware.UserIDKey).(string)
	resp, err := h.service.Create(c.UserContext(), userID, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *Handler) List(c *fiber.Ctx) error {
	limit, err := parseQueryInt(c.Query("limit"), defaultLimit)
	if err != nil {
		return apperrors.BadRequest("RECRUITMENT_INVALID_QUERY", "limit must be an integer")
	}
	offset, err := parseQueryInt(c.Query("offset"), 0)
	if err != nil {
		return apperrors.BadRequest("RECRUITMENT_INVALID_QUERY", "offset must be an integer")
	}

	resp, err := h.service.List(c.UserContext(), limit, offset)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

func (h *Handler) Get(c *fiber.Ctx) error {
	resp, err := h.service.Get(c.UserContext(), c.Params("id"))
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
