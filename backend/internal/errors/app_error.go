package apperrors

import "fmt"

type AppError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Status  int         `json:"-"`
	Cause   error       `json:"-"`
	Details interface{} `json:"details,omitempty"`
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func New(status int, code, message string, cause error) *AppError {
	return &AppError{Status: status, Code: code, Message: message, Cause: cause}
}
