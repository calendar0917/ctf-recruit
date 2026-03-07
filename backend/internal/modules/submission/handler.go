package submission

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

func (h *Handler) RegisterRoutes(v1 fiber.Router, authService *auth.Service, rateLimiters ...fiber.Handler) {
	submissions := v1.Group("/submissions", middleware.Auth(authService))
	if len(rateLimiters) > 0 && rateLimiters[0] != nil {
		submissions.Post("", rateLimiters[0], h.Create)
	} else {
		submissions.Post("", h.Create)
	}
	submissions.Get("/me", h.ListMine)
	submissions.Get("/challenge/:id", h.ListMineByChallenge)
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

func (h *Handler) ListMine(c *fiber.Ctx) error {
	userID, ok := c.Locals(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		return apperrors.Unauthorized("AUTH_INVALID_TOKEN", "Invalid token")
	}

	limit, err := parseQueryInt(c.Query("limit"), defaultLimit)
	if err != nil {
		return apperrors.BadRequest("SUBMISSION_INVALID_QUERY", "limit must be an integer")
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	offset, err := parseQueryInt(c.Query("offset"), 0)
	if err != nil {
		return apperrors.BadRequest("SUBMISSION_INVALID_QUERY", "offset must be an integer")
	}

	resp, err := h.service.ListMine(c.UserContext(), userID, limit, offset)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

func (h *Handler) ListMineByChallenge(c *fiber.Ctx) error {
	userID, ok := c.Locals(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		return apperrors.Unauthorized("AUTH_INVALID_TOKEN", "Invalid token")
	}

	challengeID := c.Params("id")
	if challengeID == "" {
		return apperrors.BadRequest("SUBMISSION_VALIDATION_ERROR", "challengeId is required")
	}

	limit, err := parseQueryInt(c.Query("limit"), defaultLimit)
	if err != nil {
		return apperrors.BadRequest("SUBMISSION_INVALID_QUERY", "limit must be an integer")
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	offset, err := parseQueryInt(c.Query("offset"), 0)
	if err != nil {
		return apperrors.BadRequest("SUBMISSION_INVALID_QUERY", "offset must be an integer")
	}

	resp, err := h.service.ListMineByChallenge(c.UserContext(), userID, challengeID, limit, offset)
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
