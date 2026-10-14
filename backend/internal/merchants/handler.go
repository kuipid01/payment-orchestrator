package merchants

import (
	"github.com/gofiber/fiber/v2"

	apihttp "github.com/kuipid01/payconnect/internal/http"
	"github.com/kuipid01/payconnect/internal/platform/identity"
)

// Handler maps HTTP requests onto the merchant service.
type Handler struct {
	svc       *Service
	responder *apihttp.Responder
}

// NewHandler builds the merchant handler.
func NewHandler(svc *Service, responder *apihttp.Responder) *Handler {
	return &Handler{svc: svc, responder: responder}
}

// Get handles GET /v1/merchant.
func (h *Handler) Get(c *fiber.Ctx) error {
	principal := identity.From(c)
	m, err := h.svc.Get(principal.MerchantID)
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, ToResponse(m))
}

// Update handles PATCH /v1/merchant.
func (h *Handler) Update(c *fiber.Ctx) error {
	principal := identity.From(c)
	var req struct {
		BusinessName    *string `json:"business_name"`
		Country         *string `json:"country"`
		DefaultCurrency *string `json:"default_currency"`
	}
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	m, err := h.svc.Update(principal.MerchantID, UpdateInput{
		BusinessName:    req.BusinessName,
		Country:         req.Country,
		DefaultCurrency: req.DefaultCurrency,
	})
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, ToResponse(m))
}
