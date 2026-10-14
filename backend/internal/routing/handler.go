package routing

import (
	"github.com/gofiber/fiber/v2"

	"github.com/kuipid01/payconnect/internal/constants"
	apihttp "github.com/kuipid01/payconnect/internal/http"
	"github.com/kuipid01/payconnect/internal/identityhelp"
	"github.com/kuipid01/payconnect/internal/platform/identity"
)

// Handler maps HTTP requests onto the routing service.
type Handler struct {
	svc       *Service
	responder *apihttp.Responder
}

// NewHandler builds the routing handler.
func NewHandler(svc *Service, responder *apihttp.Responder) *Handler {
	return &Handler{svc: svc, responder: responder}
}

// Get handles GET /v1/routing: the merchant's rules plus the provider ordering
// that would be used for the requested environment.
func (h *Handler) Get(c *fiber.Ctx) error {
	principal := identity.From(c)
	environment := environmentFor(c, principal)
	rules, err := h.svc.List(principal.MerchantID)
	if err != nil {
		return h.responder.Error(c, err)
	}
	out := OverviewResponse{Environment: environment}
	if connected, err := h.svc.ConnectedProviders(principal.MerchantID, environment); err == nil {
		for _, provider := range connected {
			out.Connected = append(out.Connected, provider.String())
		}
	}
	if fallback, err := h.svc.DefaultProvider(principal.MerchantID, environment); err == nil {
		out.DefaultProvider = fallback.String()
	}
	out.Rules = make([]Response, 0, len(rules))
	for i := range rules {
		out.Rules = append(out.Rules, ToResponse(&rules[i]))
	}
	return h.responder.Success(c, out)
}

// Update handles PATCH /v1/routing: it changes the environment default provider.
func (h *Handler) Update(c *fiber.Ctx) error {
	principal := identity.From(c)
	var req struct {
		Environment     string `json:"environment"`
		DefaultProvider string `json:"default_provider"`
	}
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	if req.Environment == "" {
		req.Environment = environmentFor(c, principal)
	}
	if err := h.svc.SetDefaultProvider(principal.MerchantID,
		identityhelp.ActorName(principal), identityhelp.ActorRole(principal),
		req.Environment, constants.Provider(req.DefaultProvider)); err != nil {
		return h.responder.Error(c, err)
	}
	return h.Get(c)
}

// ListRules handles GET /v1/routing/rules.
func (h *Handler) ListRules(c *fiber.Ctx) error {
	principal := identity.From(c)
	rules, err := h.svc.List(principal.MerchantID)
	if err != nil {
		return h.responder.Error(c, err)
	}
	out := make([]Response, 0, len(rules))
	for i := range rules {
		out = append(out, ToResponse(&rules[i]))
	}
	return h.responder.Success(c, out)
}

// CreateRule handles POST /v1/routing/rules.
func (h *Handler) CreateRule(c *fiber.Ctx) error {
	principal := identity.From(c)
	var req RuleRequest
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	rule, err := h.svc.CreateRule(c.UserContext(), principal.MerchantID,
		identityhelp.ActorName(principal), identityhelp.ActorRole(principal), req.toInput())
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Created(c, ToResponse(rule))
}

// GetRule handles GET /v1/routing/rules/{id}.
func (h *Handler) GetRule(c *fiber.Ctx) error {
	principal := identity.From(c)
	rule, err := h.svc.Get(principal.MerchantID, c.Params("id"))
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, ToResponse(rule))
}

// UpdateRule handles PATCH /v1/routing/rules/{id}.
func (h *Handler) UpdateRule(c *fiber.Ctx) error {
	principal := identity.From(c)
	var req RuleRequest
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	rule, err := h.svc.UpdateRule(principal.MerchantID,
		identityhelp.ActorName(principal), identityhelp.ActorRole(principal), c.Params("id"), req.toInput())
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, ToResponse(rule))
}

// DeleteRule handles DELETE /v1/routing/rules/{id}.
func (h *Handler) DeleteRule(c *fiber.Ctx) error {
	principal := identity.From(c)
	if err := h.svc.DeleteRule(principal.MerchantID,
		identityhelp.ActorName(principal), identityhelp.ActorRole(principal), c.Params("id")); err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, fiber.Map{"deleted": true})
}

// environmentFor pins the request environment to the credential being used.
func environmentFor(c *fiber.Ctx, principal *identity.Principal) string {
	if principal.IsAPIKey() {
		return principal.Environment
	}
	if requested := c.Query("environment"); constants.IsValidPaymentEnvironment(requested) {
		return requested
	}
	return constants.EnvironmentLive
}
