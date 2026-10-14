package webhooks

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/kuipid01/payconnect/internal/constants"
	apihttp "github.com/kuipid01/payconnect/internal/http"
	"github.com/kuipid01/payconnect/internal/identityhelp"
	"github.com/kuipid01/payconnect/internal/middleware"
	"github.com/kuipid01/payconnect/internal/platform/identity"
)

// Handler maps HTTP requests onto the webhook service.
type Handler struct {
	svc       *Service
	responder *apihttp.Responder
}

// NewHandler builds the webhook handler.
func NewHandler(svc *Service, responder *apihttp.Responder) *Handler {
	return &Handler{svc: svc, responder: responder}
}

/* ------------------------------ provider inbound ----------------------------- */

// Receive handles POST /v1/webhooks/{provider}/{connection_id}. It is a public
// endpoint: the provider signature, verified against the connection's own
// credentials, is the authentication.
func (h *Handler) Receive(c *fiber.Ctx) error {
	provider := constants.Provider(strings.ToLower(c.Params("provider")))
	headers := map[string]string{}
	c.Request().Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
	})
	requestID := middleware.RequestIDValue(c)
	result, err := h.svc.HandleInbound(c.UserContext(), requestID, provider, c.Params("connection_id"),
		headers, append([]byte(nil), c.Body()...))
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, result)
}

/* --------------------------------- endpoints --------------------------------- */

// ListEndpoints handles GET /v1/webhook-endpoints.
func (h *Handler) ListEndpoints(c *fiber.Ctx) error {
	principal := identity.From(c)
	rows, err := h.svc.ListEndpoints(EndpointFilter{
		MerchantID:  principal.MerchantID,
		Environment: c.Query("environment"),
		Status:      c.Query("status"),
	})
	if err != nil {
		return h.responder.Error(c, err)
	}
	stats, err := h.svc.EndpointDeliveryStats(principal.MerchantID)
	if err != nil {
		return h.responder.Error(c, err)
	}
	out := make([]EndpointResponse, 0, len(rows))
	for i := range rows {
		response := ToEndpointResponse(&rows[i])
		response.WithDeliveryStats(stats[rows[i].ID])
		out = append(out, response)
	}
	return h.responder.Success(c, out)
}

// GetEndpoint handles GET /v1/webhook-endpoints/{id}.
func (h *Handler) GetEndpoint(c *fiber.Ctx) error {
	principal := identity.From(c)
	endpoint, err := h.svc.GetEndpoint(principal.MerchantID, c.Params("id"))
	if err != nil {
		return h.responder.Error(c, err)
	}
	stats, err := h.svc.EndpointDeliveryStats(principal.MerchantID)
	if err != nil {
		return h.responder.Error(c, err)
	}
	response := ToEndpointResponse(endpoint)
	response.WithDeliveryStats(stats[endpoint.ID])
	return h.responder.Success(c, response)
}

// CreateEndpoint handles POST /v1/webhook-endpoints.
func (h *Handler) CreateEndpoint(c *fiber.Ctx) error {
	principal := identity.From(c)
	var req struct {
		URL           string   `json:"url"`
		Environment   string   `json:"environment"`
		Status        string   `json:"status"`
		Events        []string `json:"events"`
		ApplicationID string   `json:"application_id"`
	}
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	var applicationID *string
	if req.ApplicationID != "" {
		applicationID = &req.ApplicationID
	}
	endpoint, secret, err := h.svc.CreateEndpoint(c.UserContext(), principal.MerchantID,
		identityhelp.ActorName(principal), identityhelp.ActorRole(principal), EndpointInput{
			URL:           req.URL,
			Environment:   req.Environment,
			Status:        req.Status,
			Events:        req.Events,
			ApplicationID: applicationID,
		})
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Created(c, CreatedEndpointResponse{
		Endpoint:      ToEndpointResponse(endpoint),
		SigningSecret: secret,
	})
}

// UpdateEndpoint handles PATCH /v1/webhook-endpoints/{id}.
func (h *Handler) UpdateEndpoint(c *fiber.Ctx) error {
	principal := identity.From(c)
	var req struct {
		URL         string   `json:"url"`
		Environment string   `json:"environment"`
		Status      string   `json:"status"`
		Events      []string `json:"events"`
	}
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	endpoint, err := h.svc.UpdateEndpoint(principal.MerchantID,
		identityhelp.ActorName(principal), identityhelp.ActorRole(principal), c.Params("id"), EndpointInput{
			URL:         req.URL,
			Environment: req.Environment,
			Status:      req.Status,
			Events:      req.Events,
		})
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, ToEndpointResponse(endpoint))
}

// DeleteEndpoint handles DELETE /v1/webhook-endpoints/{id}.
func (h *Handler) DeleteEndpoint(c *fiber.Ctx) error {
	principal := identity.From(c)
	if err := h.svc.DeleteEndpoint(principal.MerchantID,
		identityhelp.ActorName(principal), identityhelp.ActorRole(principal), c.Params("id")); err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, fiber.Map{"deleted": true})
}

/* ------------------------------ inbound events ------------------------------- */

// ListInboundEvents handles GET /v1/webhook-events — the provider events this
// merchant has received, newest first. It is the inbound counterpart to the
// outbound delivery list.
func (h *Handler) ListInboundEvents(c *fiber.Ctx) error {
	principal := identity.From(c)
	page := apihttp.ParsePagination(c)
	rows, total, err := h.svc.ListInboundEvents(ProviderEventFilter{
		MerchantID:   principal.MerchantID,
		ConnectionID: c.Query("connection"),
		Provider:     c.Query("provider"),
		EventType:    c.Query("event"),
		Offset:       page.Offset(),
		Limit:        page.Limit,
	})
	if err != nil {
		return h.responder.Error(c, err)
	}
	out := make([]ProviderEventResponse, 0, len(rows))
	for i := range rows {
		out = append(out, ToProviderEventResponse(&rows[i]))
	}
	return h.responder.Paginated(c, out, apihttp.Page{
		Page:    page.Page,
		Limit:   page.Limit,
		Total:   &total,
		HasMore: int64(page.Offset()+len(rows)) < total,
	})
}

/* -------------------------------- deliveries --------------------------------- */

// ListDeliveries handles GET /v1/webhook-deliveries.
func (h *Handler) ListDeliveries(c *fiber.Ctx) error {
	principal := identity.From(c)
	page := apihttp.ParsePagination(c)
	rows, total, err := h.svc.ListDeliveries(DeliveryFilter{
		MerchantID: principal.MerchantID,
		EndpointID: c.Query("endpoint"),
		Event:      c.Query("event"),
		Status:     c.Query("status"),
		Offset:     page.Offset(),
		Limit:      page.Limit,
	})
	if err != nil {
		return h.responder.Error(c, err)
	}
	out := make([]DeliveryResponse, 0, len(rows))
	for i := range rows {
		out = append(out, ToDeliveryResponse(&rows[i]))
	}
	return h.responder.Paginated(c, out, apihttp.Page{
		Page:    page.Page,
		Limit:   page.Limit,
		Total:   &total,
		HasMore: int64(page.Offset()+len(rows)) < total,
	})
}

// GetDelivery handles GET /v1/webhook-deliveries/{id}.
func (h *Handler) GetDelivery(c *fiber.Ctx) error {
	principal := identity.From(c)
	delivery, err := h.svc.GetDelivery(principal.MerchantID, c.Params("id"))
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, ToDeliveryResponse(delivery))
}

// TestEndpoint handles POST /v1/webhook-endpoints/{id}/test. It delivers one
// signed sample event — event type webhook.test — so a merchant can see their
// receiver accept or reject a real HTTPS request without waiting for a payment.
// The outcome is returned immediately, including what the endpoint answered.
func (h *Handler) TestEndpoint(c *fiber.Ctx) error {
	principal := identity.From(c)
	result, err := h.svc.TestEndpoint(c.UserContext(), principal.MerchantID, c.Params("id"))
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, ToTestDeliveryResponse(result))
}

// RetryDelivery handles POST /v1/webhook-deliveries/{id}/retry.
func (h *Handler) RetryDelivery(c *fiber.Ctx) error {
	principal := identity.From(c)
	delivery, err := h.svc.RetryDelivery(c.UserContext(), principal.MerchantID, c.Params("id"))
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, ToDeliveryResponse(delivery))
}
