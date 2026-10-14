package transactions

import (
	"github.com/gofiber/fiber/v2"

	apihttp "github.com/kuipid01/payconnect/internal/http"
	"github.com/kuipid01/payconnect/internal/platform/identity"
)

// Handler maps HTTP requests onto the ledger service.
type Handler struct {
	svc       *Service
	responder *apihttp.Responder
}

// NewHandler builds the ledger handler.
func NewHandler(svc *Service, responder *apihttp.Responder) *Handler {
	return &Handler{svc: svc, responder: responder}
}

// List handles GET /v1/transactions.
func (h *Handler) List(c *fiber.Ctx) error {
	principal := identity.From(c)
	page := apihttp.ParsePagination(c)
	environment := c.Query("environment")
	if principal.IsAPIKey() {
		environment = principal.Environment
	}
	filter := Filter{
		MerchantID:  principal.MerchantID,
		Environment: environment,
		Type:        c.Query("type"),
		Status:      c.Query("status"),
		PaymentID:   c.Query("payment_id"),
		Offset:      page.Offset(),
		Limit:       page.Limit,
	}
	rows, total, err := h.svc.List(filter)
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Paginated(c, rows, apihttp.Page{
		Page:    page.Page,
		Limit:   page.Limit,
		Total:   &total,
		HasMore: int64(page.Offset()+len(rows)) < total,
	})
}

// Get handles GET /v1/transactions/{id}.
func (h *Handler) Get(c *fiber.Ctx) error {
	principal := identity.From(c)
	row, err := h.svc.Get(principal.MerchantID, c.Params("id"))
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, row)
}
