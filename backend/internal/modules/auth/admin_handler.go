package auth

import (
	apperrors "ctf-recruit/backend/internal/errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func (h *Handler) AdminListUsers(c *fiber.Ctx) error {
	limit, err := parseQueryInt(c.Query("limit"), defaultAdminListLimit)
	if err != nil {
		return apperrors.BadRequest("AUTH_INVALID_QUERY", "limit must be an integer")
	}
	offset, err := parseQueryInt(c.Query("offset"), 0)
	if err != nil {
		return apperrors.BadRequest("AUTH_INVALID_QUERY", "offset must be an integer")
	}

	resp, err := h.service.ListUsers(c.UserContext(), limit, offset)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

func (h *Handler) AdminUpdateUser(c *fiber.Ctx) error {
	operatorID, _ := c.Locals("userId").(string)
	if operatorID == "" {
		return apperrors.Unauthorized("AUTH_INVALID_TOKEN", "Invalid token")
	}

	var req AdminUpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return apperrors.BadRequest("AUTH_INVALID_PAYLOAD", "Invalid request payload")
	}

	resp, err := h.service.AdminUpdateUser(c.UserContext(), operatorID, c.Params("id"), req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

func (h *Handler) AdminSample(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "admin access granted",
	})
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
