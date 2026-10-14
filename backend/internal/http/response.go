// Package apihttp owns the HTTP envelope: one success shape, one error shape.
// Handlers map domain results onto it; nothing else builds responses.
package apihttp

import (
	"github.com/gofiber/fiber/v2"

	apperrors "github.com/kuipid01/payconnect/internal/errors"
)

// Responder renders the standard API envelope.
type Responder struct{}

func NewResponder() *Responder { return &Responder{} }

// Envelope meta carries cross-cutting request metadata.
type Meta struct {
	RequestID string `json:"request_id"`
}

func metaFor(c *fiber.Ctx) Meta {
	id, _ := c.Locals("requestID").(string)
	if id == "" {
		id = c.Get(fiber.HeaderXRequestID)
	}
	return Meta{RequestID: id}
}

// Success writes {"success":true,"data":…,"meta":{…}}.
func (r *Responder) Success(c *fiber.Ctx, data any) error {
	return c.JSON(fiber.Map{
		"success": true,
		"data":    data,
		"meta":    metaFor(c),
	})
}

// Created is Success with a 201 status.
func (r *Responder) Created(c *fiber.Ctx, data any) error {
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    data,
		"meta":    metaFor(c),
	})
}

// Error renders the standard error envelope. The legacy `error.message` key is
// preserved so existing dashboard clients keep working unchanged.
func (r *Responder) Error(c *fiber.Ctx, err error) error {
	appErr := apperrors.From(err)
	if appErr == nil {
		appErr = apperrors.Internal(nil)
	}
	if appErr.Type == "" {
		appErr.Type = "internal_error"
	}
	return c.Status(appErr.HTTPStatus).JSON(fiber.Map{
		"success": false,
		"error": fiber.Map{
			"type":    appErr.Type,
			"code":    appErr.Code,
			"message": appErr.Message,
			"details": appErr.Details,
		},
		"meta": metaFor(c),
	})
}

// Paginated writes a list envelope with pagination metadata.
func (r *Responder) Paginated(c *fiber.Ctx, data any, page Page) error {
	return c.JSON(fiber.Map{
		"success":    true,
		"data":       data,
		"pagination": page,
		"meta":       metaFor(c),
	})
}

// SuccessCompat writes the standard envelope and additionally merges legacy
// top-level fields. The existing Next.js dashboard reads `token`, `user`,
// `verified` and `ok` at the root, so those endpoints keep working unchanged
// while new clients can read the same values from `data`.
func (r *Responder) SuccessCompat(c *fiber.Ctx, data any, legacy fiber.Map) error {
	body := fiber.Map{
		"success": true,
		"data":    data,
		"meta":    metaFor(c),
	}
	for k, v := range legacy {
		if _, exists := body[k]; !exists {
			body[k] = v
		}
	}
	return c.JSON(body)
}

// CreatedCompat is SuccessCompat with a 201 status.
func (r *Responder) CreatedCompat(c *fiber.Ctx, data any, legacy fiber.Map) error {
	body := fiber.Map{
		"success": true,
		"data":    data,
		"meta":    metaFor(c),
	}
	for k, v := range legacy {
		if _, exists := body[k]; !exists {
			body[k] = v
		}
	}
	return c.Status(fiber.StatusCreated).JSON(body)
}
