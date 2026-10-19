package providers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/kuipid01/payconnect/internal/constants"
	apihttp "github.com/kuipid01/payconnect/internal/http"
	"github.com/kuipid01/payconnect/internal/merchants"
	"github.com/kuipid01/payconnect/internal/platform/identity"
)

// Handler maps HTTP requests onto the provider service.
type Handler struct {
	svc       *Service
	responder *apihttp.Responder
}

// NewHandler builds the provider handler.
func NewHandler(svc *Service, responder *apihttp.Responder) *Handler {
	return &Handler{svc: svc, responder: responder}
}

// Supported handles GET /v1/providers.
func (h *Handler) Supported(c *fiber.Ctx) error {
	principal := identity.From(c)
	connected, err := h.svc.ConnectedProviders(principal.MerchantID, environmentOrDefault(principal.Environment))
	if err != nil {
		return h.responder.Error(c, err)
	}
	connectedSet := make(map[string]bool, len(connected))
	for _, provider := range connected {
		connectedSet[provider.String()] = true
	}
	out := make([]fiber.Map, 0)
	for _, descriptor := range Descriptors() {
		out = append(out, fiber.Map{
			"provider":                 descriptor.Provider,
			"default_currency":         descriptor.Default,
			"connected":                connectedSet[descriptor.Provider],
			"webhook_secret_required":  descriptor.WebhookSecretRequired,
			"webhook_secret_label":     descriptor.WebhookSecretLabel,
			"webhook_signature_header": descriptor.WebhookSignatureHeader,
			"webhook_dashboard_path":   descriptor.WebhookDashboardPath,
		})
	}
	return h.responder.Success(c, out)
}

// ListConnections handles GET /v1/providers/connections.
func (h *Handler) ListConnections(c *fiber.Ctx) error {
	principal := identity.From(c)
	rows, err := h.svc.List(principal.MerchantID)
	if err != nil {
		return h.responder.Error(c, err)
	}
	out := make([]ConnectionResponse, 0, len(rows))
	for i := range rows {
		out = append(out, h.svc.ToResponse(&rows[i]))
	}
	return h.responder.Success(c, out)
}

// GetConnection handles GET /v1/providers/connections/{id}.
func (h *Handler) GetConnection(c *fiber.Ctx) error {
	principal := identity.From(c)
	connection, err := h.svc.Get(principal.MerchantID, c.Params("id"))
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, h.svc.ToResponse(connection))
}

// CreateConnection handles POST /v1/providers/connections.
func (h *Handler) CreateConnection(c *fiber.Ctx) error {
	principal := identity.From(c)
	var req struct {
		Provider      string `json:"provider"`
		Environment   string `json:"environment"`
		DisplayName   string `json:"display_name"`
		WebhookSecret string `json:"webhook_secret"`
		IsDefault     bool   `json:"is_default"`
		Credentials   struct {
			SecretKey    string `json:"secret_key"`
			PublicKey    string `json:"public_key"`
			APIKey       string `json:"api_key"`
			ContractCode string `json:"contract_code"`
		} `json:"credentials"`
	}
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	connection, err := h.svc.Connect(c.UserContext(), ConnectInput{
		MerchantID:    principal.MerchantID,
		ActorName:     actorName(principal),
		ActorRole:     actorRole(principal),
		Provider:      constants.Provider(req.Provider),
		Environment:   req.Environment,
		DisplayName:   req.DisplayName,
		SecretKey:     req.Credentials.SecretKey,
		PublicKey:     req.Credentials.PublicKey,
		APIKey:        req.Credentials.APIKey,
		ContractCode:  req.Credentials.ContractCode,
		WebhookSecret: req.WebhookSecret,
		IsDefault:     req.IsDefault,
	})
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Created(c, h.svc.ToResponse(connection))
}

// UpdateConnection handles PATCH /v1/providers/connections/{id}.
func (h *Handler) UpdateConnection(c *fiber.Ctx) error {
	principal := identity.From(c)
	var req struct {
		DisplayName *string `json:"display_name"`
		IsDefault   *bool   `json:"is_default"`
		Credentials *struct {
			SecretKey    string `json:"secret_key"`
			PublicKey    string `json:"public_key"`
			APIKey       string `json:"api_key"`
			ContractCode string `json:"contract_code"`
		} `json:"credentials"`
	}
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	in := UpdateInput{DisplayName: req.DisplayName, IsDefault: req.IsDefault}
	if req.Credentials != nil {
		in.SecretKey = &req.Credentials.SecretKey
		in.PublicKey = &req.Credentials.PublicKey
		in.APIKey = &req.Credentials.APIKey
		in.ContractCode = &req.Credentials.ContractCode
	}
	connection, err := h.svc.Update(principal.MerchantID, actorName(principal), actorRole(principal), c.Params("id"), in)
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, h.svc.ToResponse(connection))
}

// SetWebhookSecret handles POST /v1/providers/connections/{id}/webhook-secret.
// It stores the inbound webhook signing value for providers whose signature key
// is not the API secret key.
func (h *Handler) SetWebhookSecret(c *fiber.Ctx) error {
	principal := identity.From(c)
	var req struct {
		WebhookSecret string `json:"webhook_secret"`
	}
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	connection, err := h.svc.SetWebhookSecret(principal.MerchantID,
		actorName(principal), actorRole(principal), c.Params("id"), req.WebhookSecret)
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, h.svc.ToResponse(connection))
}

// DeleteConnection handles DELETE /v1/providers/connections/{id}.
func (h *Handler) DeleteConnection(c *fiber.Ctx) error {
	principal := identity.From(c)
	if err := h.svc.Delete(principal.MerchantID, actorName(principal), actorRole(principal), c.Params("id")); err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, fiber.Map{"deleted": true})
}

// TestConnection handles POST /v1/providers/connections/{id}/test.
func (h *Handler) TestConnection(c *fiber.Ctx) error {
	principal := identity.From(c)
	connection, err := h.svc.Test(c.UserContext(), principal.MerchantID, actorName(principal), actorRole(principal), c.Params("id"))
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, fiber.Map{
		"status":   connection.Status,
		"provider": connection.Provider,
		"verified": connection.LastVerifiedAt != nil,
	})
}

func environmentOrDefault(environment string) string {
	if environment == "" {
		return constants.EnvironmentTest
	}
	return environment
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
