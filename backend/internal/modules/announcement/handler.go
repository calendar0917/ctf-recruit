package announcement

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
	announcementGroup := v1.Group("/announcements", middleware.Auth(authService))

	announcementGroup.Get("", h.List)
	announcementGroup.Get("/:id", h.Get)

	announcementGroup.Post("", middleware.RequireRoles(auth.RoleAdmin), h.Create)
	announcementGroup.Put("/:id", middleware.RequireRoles(auth.RoleAdmin), h.Update)
	announcementGroup.Delete("/:id", middleware.RequireRoles(auth.RoleAdmin), h.Delete)
}

func (h *Handler) Create(c *fiber.Ctx) error {
	var req CreateAnnouncementRequest
	if err := c.BodyParser(&req); err != nil {
		return apperrors.BadRequest("ANNOUNCEMENT_INVALID_PAYLOAD", "Invalid request payload")
	}

	resp, err := h.service.Create(c.UserContext(), req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *Handler) Update(c *fiber.Ctx) error {
	var req UpdateAnnouncementRequest
	if err := c.BodyParser(&req); err != nil {
		return apperrors.BadRequest("ANNOUNCEMENT_INVALID_PAYLOAD", "Invalid request payload")
	}

	resp, err := h.service.Update(c.UserContext(), c.Params("id"), req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	if err := h.service.Delete(c.UserContext(), c.Params("id")); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) List(c *fiber.Ctx) error {
	publishedOnly := true
	if role, ok := c.Locals(middleware.RoleKey).(auth.Role); ok && role == auth.RoleAdmin {
		publishedOnly = false
	}

	limit, err := parseQueryInt(c.Query("limit"), defaultLimit)
	if err != nil {
		return apperrors.BadRequest("ANNOUNCEMENT_INVALID_QUERY", "limit must be an integer")
	}
	offset, err := parseQueryInt(c.Query("offset"), 0)
	if err != nil {
		return apperrors.BadRequest("ANNOUNCEMENT_INVALID_QUERY", "offset must be an integer")
	}

	resp, err := h.service.List(c.UserContext(), publishedOnly, limit, offset)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

func (h *Handler) Get(c *fiber.Ctx) error {
	publishedOnly := true
	if role, ok := c.Locals(middleware.RoleKey).(auth.Role); ok && role == auth.RoleAdmin {
		publishedOnly = false
	}

	resp, err := h.service.Get(c.UserContext(), c.Params("id"), publishedOnly)
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
