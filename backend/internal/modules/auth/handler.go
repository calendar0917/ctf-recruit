package auth

import (
	apperrors "ctf-recruit/backend/internal/errors"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return apperrors.BadRequest("AUTH_INVALID_PAYLOAD", "Invalid request payload")
	}

	resp, err := h.service.Register(c.UserContext(), req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *Handler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return apperrors.BadRequest("AUTH_INVALID_PAYLOAD", "Invalid request payload")
	}

	resp, err := h.service.Login(c.UserContext(), req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

func (h *Handler) Me(c *fiber.Ctx) error {
	userID, ok := c.Locals("userId").(string)
	if !ok || userID == "" {
		return apperrors.Unauthorized("AUTH_INVALID_TOKEN", "Invalid token")
	}

	resp, err := h.service.Me(c.UserContext(), userID)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}

func ExtractBearerToken(header string) string {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
