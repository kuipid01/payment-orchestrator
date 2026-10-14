package application

import (
	"github.com/gofiber/fiber/v2"

	apihttp "github.com/kuipid01/payconnect/internal/http"
	"github.com/kuipid01/payconnect/internal/identityhelp"
	"github.com/kuipid01/payconnect/internal/platform/identity"
)

// Handler maps HTTP requests onto the application service.
type Handler struct {
	svc       *Service
	responder *apihttp.Responder
}

// NewHandler builds the application handler.
func NewHandler(svc *Service, responder *apihttp.Responder) *Handler {
	return &Handler{svc: svc, responder: responder}
}

// Create handles POST /v1/applications.
func (h *Handler) Create(c *fiber.Ctx) error {
	principal := identity.From(c)
	var req Input
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	result, err := h.svc.Create(c.UserContext(), principal.MerchantID,
		identityhelp.ActorName(principal), identityhelp.ActorRole(principal), req)
	if err != nil {
		return h.responder.Error(c, err)
	}
	response, err := h.svc.WithStats(result.Application)
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Created(c, CreatedResponse{
		Application: response,
		APIKey:      result.APIKey,
		Keys:        toCreatedKeys(result.Keys),
	})
}

// toCreatedKeys maps the minted keys onto the one-time response DTO.
func toCreatedKeys(keys []CreatedKey) []CreatedKeyResponse {
	out := make([]CreatedKeyResponse, 0, len(keys))
	for _, key := range keys {
		out = append(out, CreatedKeyResponse{
			ID:          key.ID,
			Environment: key.Environment,
			Scope:       key.Scope,
			Secret:      key.Secret,
		})
	}
	return out
}

// List handles GET /v1/applications.
func (h *Handler) List(c *fiber.Ctx) error {
	principal := identity.From(c)
	rows, err := h.svc.List(principal.MerchantID, c.Query("application_id"), principal)
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, rows)
}

// Get handles GET /v1/applications/:id.
func (h *Handler) Get(c *fiber.Ctx) error {
	principal := identity.From(c)
	app, err := h.svc.Get(principal.MerchantID, c.Params("id"), principal)
	if err != nil {
		return h.responder.Error(c, err)
	}
	response, err := h.svc.WithStats(app)
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, response)
}

// Update handles PATCH /v1/applications/:id.
func (h *Handler) Update(c *fiber.Ctx) error {
	principal := identity.From(c)
	var req Input
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	app, err := h.svc.Update(principal.MerchantID, c.Params("id"),
		identityhelp.ActorName(principal), identityhelp.ActorRole(principal), req)
	if err != nil {
		return h.responder.Error(c, err)
	}
	response, err := h.svc.WithStats(app)
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, response)
}

// Delete handles DELETE /v1/applications/:id.
func (h *Handler) Delete(c *fiber.Ctx) error {
	principal := identity.From(c)
	if err := h.svc.Delete(principal.MerchantID, c.Params("id"),
		identityhelp.ActorName(principal), identityhelp.ActorRole(principal)); err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, fiber.Map{"deleted": true})
}
