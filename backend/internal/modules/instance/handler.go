package instance

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
	group := v1.Group("/instances", middleware.Auth(authService))
	group.Post("/start", h.Start)
	group.Post("/stop", h.Stop)
	group.Get("/me", h.Me)
	group.Post("/:id/transition", h.Transition)
}

func (h *Handler) Start(c *fiber.Ctx) error {
	var req StartInstanceRequest
	if err := c.BodyParser(&req); err != nil {
		return apperrors.BadRequest("INSTANCE_INVALID_PAYLOAD", "Invalid request payload")
	}

	userID, _ := c.Locals(middleware.UserIDKey).(string)
	resp, err := h.service.Start(c.UserContext(), userID, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *Handler) Transition(c *fiber.Ctx) error {
	var req TransitionRequest
	if err := c.BodyParser(&req); err != nil {
		return apperrors.BadRequest("INSTANCE_INVALID_PAYLOAD", "Invalid request payload")
	}

	userID, _ := c.Locals(middleware.UserIDKey).(string)
	resp, err := h.service.Transition(c.UserContext(), userID, c.Params("id"), req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

func (h *Handler) Stop(c *fiber.Ctx) error {
	var req StopInstanceRequest
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return apperrors.BadRequest("INSTANCE_INVALID_PAYLOAD", "Invalid request payload")
		}
	}

	userID, _ := c.Locals(middleware.UserIDKey).(string)
	resp, err := h.service.Stop(c.UserContext(), userID, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

func (h *Handler) Me(c *fiber.Ctx) error {
	userID, _ := c.Locals(middleware.UserIDKey).(string)
	resp, err := h.service.Me(c.UserContext(), userID)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}
