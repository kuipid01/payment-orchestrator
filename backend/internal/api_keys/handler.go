package api_keys

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/kuipid01/payconnect/internal/constants"
	apihttp "github.com/kuipid01/payconnect/internal/http"
	"github.com/kuipid01/payconnect/internal/merchants"
	"github.com/kuipid01/payconnect/internal/platform/identity"
)

// Handler maps HTTP requests onto the API key service.
type Handler struct {
	svc       *Service
	responder *apihttp.Responder
}

// NewHandler builds the API key handler.
func NewHandler(svc *Service, responder *apihttp.Responder) *Handler {
	return &Handler{svc: svc, responder: responder}
}

// Create handles POST /v1/api-keys. The full secret is returned exactly once.
func (h *Handler) Create(c *fiber.Ctx) error {
	principal := identity.From(c)
	var req struct {
		Name              string `json:"name"`
		Type              string `json:"type"`
		Environment       string `json:"environment"`
		ApplicationID     string `json:"application_id"`
		DeveloperAccessID string `json:"developer_access_id"`
		ExpiresInDays     int    `json:"expires_in_days"`
	}
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	var applicationID *string
	if req.ApplicationID != "" {
		applicationID = &req.ApplicationID
	}
	var developerAccessID *string
	if req.DeveloperAccessID != "" {
		developerAccessID = &req.DeveloperAccessID
	}
	created, err := h.svc.Create(CreateInput{
		MerchantID:        principal.MerchantID,
		ActorName:         actorName(principal),
		ActorRole:         actorRole(principal),
		Name:              req.Name,
		Type:              parseType(req.Type),
		Environment:       req.Environment,
		ApplicationID:     applicationID,
		DeveloperAccessID: developerAccessID,
		ExpiresIn:         time.Duration(req.ExpiresInDays) * 24 * time.Hour,
	})
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.CreatedCompat(c,
		CreatedResponse{Key: ToResponse(created.Key, h.svc.Now()), Secret: created.Secret},
		fiber.Map{"apiKey": created.Secret, "key": ToResponse(created.Key, h.svc.Now())})
}

// List handles GET /v1/api-keys.
func (h *Handler) List(c *fiber.Ctx) error {
	principal := identity.From(c)
	rows, err := h.svc.List(principal.MerchantID, c.Query("type"), c.Query("environment"), c.Query("developer_access_id"))
	if err != nil {
		return h.responder.Error(c, err)
	}
	now := h.svc.Now()
	out := make([]Response, 0, len(rows))
	for i := range rows {
		out = append(out, ToResponse(&rows[i], now))
	}
	return h.responder.Success(c, out)
}

// Rotate handles POST /v1/api-keys/{id}/rotate.
func (h *Handler) Rotate(c *fiber.Ctx) error {
	principal := identity.From(c)
	created, err := h.svc.Rotate(principal.MerchantID, actorName(principal), actorRole(principal), c.Params("id"))
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.SuccessCompat(c,
		CreatedResponse{Key: ToResponse(created.Key, h.svc.Now()), Secret: created.Secret},
		fiber.Map{"apiKey": created.Secret, "key": ToResponse(created.Key, h.svc.Now())})
}

// Reveal handles POST /v1/api-keys/{id}/reveal. It returns the plaintext of a
// test key so it can be copied again; live keys answer 403 by design.
func (h *Handler) Reveal(c *fiber.Ctx) error {
	principal := identity.From(c)
	secret, err := h.svc.Reveal(principal.MerchantID, actorName(principal), actorRole(principal), c.Params("id"))
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, fiber.Map{"secret": secret})
}

// Revoke handles DELETE /v1/api-keys/{id}.
func (h *Handler) Revoke(c *fiber.Ctx) error {
	principal := identity.From(c)
	if err := h.svc.Revoke(principal.MerchantID, actorName(principal), actorRole(principal), c.Params("id")); err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.SuccessCompat(c, fiber.Map{"revoked": true}, fiber.Map{"ok": true})
}

// DeleteRevoked handles DELETE /v1/api-keys/{id}/purge.
func (h *Handler) DeleteRevoked(c *fiber.Ctx) error {
	principal := identity.From(c)
	if err := h.svc.DeleteRevoked(principal.MerchantID, actorName(principal), actorRole(principal), c.Params("id")); err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.SuccessCompat(c, fiber.Map{"deleted": true}, fiber.Map{"ok": true})
}

// parseType maps the request value onto a known key type.
func parseType(value string) constants.APIKeyType {
	switch constants.APIKeyType(value) {
	case constants.APIKeyPublic:
		return constants.APIKeyPublic
	case constants.APIKeyWebhook:
		return constants.APIKeyWebhook
	default:
		return constants.APIKeySecret
	}
}

func actorName(principal *identity.Principal) string {
	if u, ok := principal.User.(*merchants.User); ok {
		return u.Name
	}
	return "system"
}

func actorRole(principal *identity.Principal) string {
	if u, ok := principal.User.(*merchants.User); ok {
		return u.Role
	}
	return "system"
}
