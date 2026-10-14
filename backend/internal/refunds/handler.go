package refunds

import (
	"github.com/gofiber/fiber/v2"

	apihttp "github.com/kuipid01/payconnect/internal/http"
	"github.com/kuipid01/payconnect/internal/identityhelp"
	"github.com/kuipid01/payconnect/internal/middleware"
	"github.com/kuipid01/payconnect/internal/platform/identity"
)

// Handler maps HTTP requests onto the refund service.
type Handler struct {
	svc       *Service
	responder *apihttp.Responder
}

// NewHandler builds the refund handler.
func NewHandler(svc *Service, responder *apihttp.Responder) *Handler {
	return &Handler{svc: svc, responder: responder}
}

// Create handles POST /v1/payments/{id}/refund.
func (h *Handler) Create(c *fiber.Ctx) error {
	principal := identity.From(c)
	if !principal.IsAPIKey() {
		return h.responder.Error(c, permissionDeniedSession())
	}
	var req struct {
		Amount int64  `json:"amount"`
		Reason string `json:"reason"`
	}
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	refund, err := h.svc.Refund(c.UserContext(), Input{
		MerchantID:     principal.MerchantID,
		ActorName:      identityhelp.ActorName(principal),
		ActorRole:      identityhelp.ActorRole(principal),
		PaymentID:      c.Params("id"),
		AmountMinor:    req.Amount,
		Reason:         req.Reason,
		IdempotencyKey: middleware.IdempotencyKey(c),
	})
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Created(c, ToResponse(refund))
}

// List handles GET /v1/refunds.
func (h *Handler) List(c *fiber.Ctx) error {
	principal := identity.From(c)
	page := apihttp.ParsePagination(c)
	environment := c.Query("environment")
	if principal.IsAPIKey() {
		environment = principal.Environment
	}
	rows, total, err := h.svc.List(Filter{
		MerchantID:  principal.MerchantID,
		PaymentID:   c.Query("payment_id"),
		Status:      c.Query("status"),
		Environment: environment,
		Offset:      page.Offset(),
		Limit:       page.Limit,
	})
	if err != nil {
		return h.responder.Error(c, err)
	}
	out := make([]Response, 0, len(rows))
	for i := range rows {
		out = append(out, ToResponse(&rows[i]))
	}
	return h.responder.Paginated(c, out, apihttp.Page{
		Page:    page.Page,
		Limit:   page.Limit,
		Total:   &total,
		HasMore: int64(page.Offset()+len(rows)) < total,
	})
}

// Get handles GET /v1/refunds/{id}.
func (h *Handler) Get(c *fiber.Ctx) error {
	principal := identity.From(c)
	refund, err := h.svc.Get(principal.MerchantID, c.Params("id"))
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, ToResponse(refund))
}
