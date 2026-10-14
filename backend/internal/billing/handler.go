package billing

import (
	"github.com/gofiber/fiber/v2"

	apihttp "github.com/kuipid01/payconnect/internal/http"
	"github.com/kuipid01/payconnect/internal/identityhelp"
	"github.com/kuipid01/payconnect/internal/platform/identity"
)

// Handler maps HTTP requests onto the billing service.
type Handler struct {
	svc       *Service
	responder *apihttp.Responder
}

// NewHandler builds the billing handler.
func NewHandler(svc *Service, responder *apihttp.Responder) *Handler {
	return &Handler{svc: svc, responder: responder}
}

// Catalog handles GET /v1/plans: the public price list, no session required,
// which is what lets the marketing page render prices it does not own.
func (h *Handler) Catalog(c *fiber.Ctx) error {
	// The catalog is a pricing constant, so a shared cache is welcome here —
	// unlike every response below, which is one merchant's own state.
	c.Set(fiber.HeaderCacheControl, "public, max-age=300")
	return h.responder.Success(c, ToCatalogResponse(Catalog()))
}

// Plan handles GET /v1/billing/plan: the merchant's plan, its entitlements, and
// when it renews or lapses.
func (h *Handler) Plan(c *fiber.Ctx) error {
	principal := identity.From(c)
	view, err := h.svc.Plan(c.UserContext(), principal.MerchantID)
	if err != nil {
		return h.responder.Error(c, err)
	}
	private(c)
	return h.responder.Success(c, ToViewResponse(view))
}

// Usage handles GET /v1/billing/usage: this month's applications, seats,
// payments and routed volume against the plan, plus the volume fee accrued so
// far.
func (h *Handler) Usage(c *fiber.Ctx) error {
	principal := identity.From(c)
	usage, err := h.svc.Usage(c.UserContext(), principal.MerchantID)
	if err != nil {
		return h.responder.Error(c, err)
	}
	private(c)
	return h.responder.Success(c, ToUsageResponse(usage))
}

// Events handles GET /v1/billing/events: the subscription trail, newest first.
func (h *Handler) Events(c *fiber.Ctx) error {
	principal := identity.From(c)
	events, err := h.svc.Events(c.UserContext(), principal.MerchantID, 20)
	if err != nil {
		return h.responder.Error(c, err)
	}
	out := make([]EventResponse, 0, len(events))
	for i := range events {
		out = append(out, ToEventResponse(&events[i]))
	}
	private(c)
	return h.responder.Success(c, out)
}

// Checkout handles POST /v1/billing/checkout: the merchant's chosen plan and
// cadence. Nothing is charged while no collection provider is wired, and the
// response says exactly that.
func (h *Handler) Checkout(c *fiber.Ctx) error {
	principal := identity.From(c)
	var req struct {
		Plan     string `json:"plan"`
		Interval string `json:"interval"`
	}
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	view, err := h.svc.RequestCheckout(c.UserContext(), principal.MerchantID,
		identityhelp.ActorName(principal), identityhelp.ActorRole(principal), req.Plan, req.Interval)
	if err != nil {
		return h.responder.Error(c, err)
	}
	private(c)
	return h.responder.Success(c, ToViewResponse(view))
}

// StartTrial handles POST /v1/billing/trial: 14 days of Pro, no card, once per
// organization.
func (h *Handler) StartTrial(c *fiber.Ctx) error {
	principal := identity.From(c)
	view, err := h.svc.StartTrial(c.UserContext(), principal.MerchantID,
		identityhelp.ActorName(principal), identityhelp.ActorRole(principal))
	if err != nil {
		return h.responder.Error(c, err)
	}
	private(c)
	return h.responder.Success(c, ToViewResponse(view))
}

// Cancel handles POST /v1/billing/cancel: cancel at period end.
func (h *Handler) Cancel(c *fiber.Ctx) error {
	principal := identity.From(c)
	view, err := h.svc.Cancel(c.UserContext(), principal.MerchantID,
		identityhelp.ActorName(principal), identityhelp.ActorRole(principal))
	if err != nil {
		return h.responder.Error(c, err)
	}
	private(c)
	return h.responder.Success(c, ToViewResponse(view))
}

// Resume handles POST /v1/billing/resume: withdraw a pending cancellation.
func (h *Handler) Resume(c *fiber.Ctx) error {
	principal := identity.From(c)
	view, err := h.svc.Resume(c.UserContext(), principal.MerchantID,
		identityhelp.ActorName(principal), identityhelp.ActorRole(principal))
	if err != nil {
		return h.responder.Error(c, err)
	}
	private(c)
	return h.responder.Success(c, ToViewResponse(view))
}

// private marks a response as one merchant's own state: a shared cache must
// never hand it to anyone else.
func private(c *fiber.Ctx) {
	c.Set(fiber.HeaderCacheControl, "private, no-store")
}
