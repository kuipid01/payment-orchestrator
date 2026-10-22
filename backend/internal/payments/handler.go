package payments

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/kuipid01/payconnect/internal/constants"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
	apihttp "github.com/kuipid01/payconnect/internal/http"
	"github.com/kuipid01/payconnect/internal/middleware"
	"github.com/kuipid01/payconnect/internal/platform/identity"
	"github.com/kuipid01/payconnect/internal/providers"
)

// Handler maps HTTP requests onto the payment service.
type Handler struct {
	svc       *Service
	responder *apihttp.Responder
}

// NewHandler builds the payment handler.
func NewHandler(svc *Service, responder *apihttp.Responder) *Handler {
	return &Handler{svc: svc, responder: responder}
}

// Create handles POST /v1/payments. It requires a developer API key, and the
// key's environment decides which providers and data the payment can touch.
func (h *Handler) Create(c *fiber.Ctx) error {
	principal := identity.From(c)
	if !principal.IsAPIKey() {
		return h.responder.Error(c, apperrorsForbiddenSession())
	}
	var req CreateRequest
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	response, err := h.svc.Create(c.UserContext(), CreateInput{
		MerchantID:        principal.MerchantID,
		ApplicationID:     principal.ApplicationID,
		Environment:       environmentFor(principal),
		APIKeyID:          principal.APIKeyID,
		DeveloperAccessID: principal.DeveloperAccessID,
		Request:           req,
		IdempotencyKey:    middleware.IdempotencyKey(c),
		RequestID:         middleware.RequestIDValue(c),
	})
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Created(c, response)
}

// Test handles POST /v1/payments/test. It starts a sandbox charge on one
// provider connection so a merchant can walk a payment through the provider and
// watch both webhook directions — the provider's notification arriving and the
// delivery leaving for their own endpoint — without writing any code.
func (h *Handler) Test(c *fiber.Ctx) error {
	principal := identity.From(c)
	var req struct {
		ConnectionID string                   `json:"connection_id"`
		Amount       int64                    `json:"amount"`
		Currency     string                   `json:"currency"`
		Reference    string                   `json:"reference"`
		Description  string                   `json:"description"`
		CallbackURL  string                   `json:"callback_url"`
		Customer     CustomerInput            `json:"customer"`
		Email        string                   `json:"email"`
		Options      providers.PaymentOptions `json:"options"`
	}
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	customer := req.Customer
	// Accept a top-level email as a convenience for the dashboard form, which
	// has no reason to nest a one-field object.
	if strings.TrimSpace(customer.Email) == "" {
		customer.Email = strings.TrimSpace(req.Email)
	}
	response, err := h.svc.CreateTest(c.UserContext(), TestCreateInput{
		MerchantID:   principal.MerchantID,
		ConnectionID: req.ConnectionID,
		Request: CreateRequest{
			Amount:      req.Amount,
			Currency:    req.Currency,
			Reference:   req.Reference,
			Description: req.Description,
			CallbackURL: req.CallbackURL,
			Customer:    customer,
			Options:     req.Options,
		},
		RequestID: middleware.RequestIDValue(c),
	})
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Created(c, response)
}

// List handles GET /v1/payments.
//
// The dashboard's payment table is entirely server-driven: environment,
// application, developer, status, provider, currency, date range and free-text
// search all arrive as query parameters and all narrow the same predicate the
// pagination total is computed from.
func (h *Handler) List(c *fiber.Ctx) error {
	principal := identity.From(c)
	environment, err := listEnvironment(principal, c.Query("environment"))
	if err != nil {
		return h.responder.Error(c, err)
	}
	page := apihttp.ParsePagination(c)
	filter := Filter{
		MerchantID:        principal.MerchantID,
		Environment:       environment,
		ApplicationID:     strings.TrimSpace(c.Query("application_id")),
		DeveloperAccessID: strings.TrimSpace(c.Query("developer_access_id")),
		Status:            c.Query("status"),
		Provider:          c.Query("provider"),
		Currency:          strings.ToUpper(c.Query("currency")),
		Reference:         c.Query("reference"),
		CustomerID:        c.Query("customer_id"),
		Search:            c.Query("q"),
		From:              queryTime(c, "from"),
		To:                queryTime(c, "to"),
		Offset:            page.Offset(),
		Limit:             page.Limit,
	}
	rows, total, summary, err := h.svc.List(filter)
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Paginated(c, rows, apihttp.Page{
		Page:    page.Page,
		Limit:   page.Limit,
		Total:   &total,
		HasMore: int64(page.Offset()+len(rows)) < total,
		Summary: summary,
	})
}

// Get handles GET /v1/payments/{id}.
func (h *Handler) Get(c *fiber.Ctx) error {
	principal := identity.From(c)
	response, err := h.svc.Get(principal.MerchantID, c.Params("id"))
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, response)
}

// Verify handles GET /v1/payments/{id}/verify and GET /v1/payments/verify?reference=…
func (h *Handler) Verify(c *fiber.Ctx) error {
	principal := identity.From(c)
	if reference := c.Query("reference"); reference != "" {
		response, err := h.svc.VerifyByReference(c.UserContext(), principal.MerchantID, environmentFor(principal), reference)
		if err != nil {
			return h.responder.Error(c, err)
		}
		return h.responder.Success(c, response)
	}
	response, err := h.svc.Verify(c.UserContext(), principal.MerchantID, c.Params("id"))
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, response)
}

// Metrics handles GET /v1/dashboard/metrics, scoped to the caller's environment
// so a test key can never observe live volume.
func (h *Handler) Metrics(c *fiber.Ctx) error {
	principal := identity.From(c)
	currency := strings.ToUpper(apihttp.Query(c, "currency", constants.DefaultCurrency))
	metrics := h.svc.Metrics(principal.MerchantID, environmentFor(principal), currency)
	return h.responder.Success(c, metrics)
}

// environmentFor pins the request environment to the credential being used.
// API keys carry their environment; dashboard sessions see live data.
func environmentFor(principal *identity.Principal) string {
	if principal.IsAPIKey() {
		return principal.Environment
	}
	return constants.EnvironmentLive
}

// listEnvironment resolves the *filter* environment for a listing.
//
// An API key stays pinned to its own environment: a test key must never read live
// payments, so its "requested" value is ignored. A dashboard session may narrow
// the listing to one environment, or ask for all of them — the operators of a
// merchant want to see the test payments their developers created alongside live
// money. Anything else is a client bug and is rejected rather than silently
// widened.
func listEnvironment(principal *identity.Principal, requested string) (string, error) {
	if principal.IsAPIKey() {
		return principal.Environment, nil
	}
	switch strings.ToLower(strings.TrimSpace(requested)) {
	case "", "all":
		return "", nil
	case constants.EnvironmentLive, constants.EnvironmentTest:
		return strings.ToLower(strings.TrimSpace(requested)), nil
	default:
		return "", apperrors.Validation(apperrors.CodeValidationError,
			"environment must be one of live, test or all.")
	}
}

func queryTime(c *fiber.Ctx, key string) *time.Time {
	value := c.Query(key)
	if value == "" {
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return &parsed
	}
	return nil
}
