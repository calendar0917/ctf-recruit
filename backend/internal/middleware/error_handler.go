package middleware

import (
	apperrors "ctf-recruit/backend/internal/errors"
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"
)

type errorBody struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

type errorResponse struct {
	Error     errorBody `json:"error"`
	RequestID string    `json:"requestId,omitempty"`
}

func ErrorHandler(c *fiber.Ctx, err error) error {
	requestID, _ := c.Locals(RequestIDKey).(string)

	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return c.Status(appErr.Status).JSON(errorResponse{
			Error:     errorBody{Code: appErr.Code, Message: appErr.Message, Details: appErr.Details},
			RequestID: requestID,
		})
	}

	if fiberErr, ok := err.(*fiber.Error); ok {
		return c.Status(fiberErr.Code).JSON(errorResponse{
			Error:     errorBody{Code: "HTTP_ERROR", Message: fiberErr.Message},
			RequestID: requestID,
		})
	}

	slog.Error("unexpected request error", "requestId", requestID, "error", err)
	return c.Status(fiber.StatusInternalServerError).JSON(errorResponse{
		Error:     errorBody{Code: "INTERNAL_ERROR", Message: "Internal server error"},
		RequestID: requestID,
	})
}
