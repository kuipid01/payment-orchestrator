// Package apperrors defines the single typed error shape every layer returns.
// Handlers never build ad-hoc error payloads: they hand an *AppError to the
// response layer, which renders the standard envelope.
package apperrors

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError is a typed application error. It carries a stable machine code, a
// human-readable message safe to expose, an HTTP status, optional public
// details, and an internal cause that is logged but never returned.
type AppError struct {
	Type       string // envelope error class, see constants.ErrorType*
	Code       string
	Message    string
	HTTPStatus int
	Details    any
	Cause      error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap exposes the internal cause to errors.Is/As without leaking it publicly.
func (e *AppError) Unwrap() error { return e.Cause }

// WithCause attaches an internal cause (logged, never serialized).
func (e *AppError) WithCause(cause error) *AppError {
	clone := *e
	clone.Cause = cause
	return &clone
}

// WithDetails attaches public details (safe for clients).
func (e *AppError) WithDetails(details any) *AppError {
	clone := *e
	clone.Details = details
	return &clone
}

// WithMessage overrides the public message.
func (e *AppError) WithMessage(message string) *AppError {
	clone := *e
	clone.Message = message
	return &clone
}

// Public exposes only what a client may see.
func (e *AppError) Public() map[string]any {
	return map[string]any{
		"code":    e.Code,
		"message": e.Message,
		"details": e.Details,
	}
}

// New builds an AppError with an explicit status.
func New(errType, code, message string, status int) *AppError {
	return &AppError{Type: errType, Code: code, Message: message, HTTPStatus: status}
}

// Internal wraps any unexpected error as a non-leaking 500.
func Internal(cause error) *AppError {
	return &AppError{
		Type:       "internal_error",
		Code:       CodeInternalError,
		Message:    "An unexpected error occurred.",
		HTTPStatus: http.StatusInternalServerError,
		Cause:      cause,
	}
}

// From normalizes any error into an *AppError. Unknown errors become 500s and
// never expose their message to clients.
func From(err error) *AppError {
	if err == nil {
		return nil
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return Internal(err)
}

// Is reports whether err carries the given application code.
func Is(err error, code string) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}
