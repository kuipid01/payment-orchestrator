package customers

import (
	"github.com/gofiber/fiber/v2"

	apihttp "github.com/kuipid01/payconnect/internal/http"
	"github.com/kuipid01/payconnect/internal/identityhelp"
	"github.com/kuipid01/payconnect/internal/platform/identity"
)

// Handler maps HTTP requests onto the customer service.
type Handler struct {
	svc       *Service
	responder *apihttp.Responder
}

// NewHandler builds the customer handler.
func NewHandler(svc *Service, responder *apihttp.Responder) *Handler {
	return &Handler{svc: svc, responder: responder}
}

// Create handles POST /v1/customers.
func (h *Handler) Create(c *fiber.Ctx) error {
	principal := identity.From(c)
	var req Input
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	customer, err := h.svc.Create(principal.MerchantID, req)
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Created(c, ToResponse(customer))
}

// List handles GET /v1/customers.
func (h *Handler) List(c *fiber.Ctx) error {
	principal := identity.From(c)
	page := apihttp.ParsePagination(c)
	rows, total, err := h.svc.List(ListFilter{
		MerchantID: principal.MerchantID,
		Email:      c.Query("email"),
		Search:     c.Query("search"),
		Offset:     page.Offset(),
		Limit:      page.Limit,
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

// Get handles GET /v1/customers/{id}.
func (h *Handler) Get(c *fiber.Ctx) error {
	principal := identity.From(c)
	customer, err := h.svc.Get(principal.MerchantID, c.Params("id"))
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, ToResponse(customer))
}

// Update handles PATCH /v1/customers/{id}.
func (h *Handler) Update(c *fiber.Ctx) error {
	principal := identity.From(c)
	var req Input
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	customer, err := h.svc.Update(principal.MerchantID, c.Params("id"), req)
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, ToResponse(customer))
}

// Delete handles DELETE /v1/customers/{id}.
func (h *Handler) Delete(c *fiber.Ctx) error {
	principal := identity.From(c)
	if err := h.svc.Delete(principal.MerchantID,
		identityhelp.ActorName(principal), identityhelp.ActorRole(principal), c.Params("id")); err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, fiber.Map{"deleted": true})
}
