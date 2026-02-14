package apperrors

import "github.com/gofiber/fiber/v2"

func BadRequest(code, message string) *AppError {
	return New(fiber.StatusBadRequest, code, message, nil)
}

func Unauthorized(code, message string) *AppError {
	return New(fiber.StatusUnauthorized, code, message, nil)
}

func Forbidden(code, message string) *AppError {
	return New(fiber.StatusForbidden, code, message, nil)
}

func NotFound(code, message string) *AppError {
	return New(fiber.StatusNotFound, code, message, nil)
}

func Conflict(code, message string) *AppError {
	return New(fiber.StatusConflict, code, message, nil)
}

func Internal(code, message string, cause error) *AppError {
	return New(fiber.StatusInternalServerError, code, message, cause)
}
