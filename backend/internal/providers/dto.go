package providers

import (
	"time"

	"github.com/kuipid01/payconnect/internal/constants"
)

// ConnectionResponse is the public connection representation. Credentials are
// never included — not even masked values beyond the last four characters.
type ConnectionResponse struct {
	ID             string  `json:"id"`
	Provider       string  `json:"provider"`
	Environment    string  `json:"environment"`
	DisplayName    string  `json:"display_name"`
	Status         string  `json:"status"`
	IsDefault      bool    `json:"is_default"`
	Secret         string  `json:"secret_last4"`
	Currency       string  `json:"currency"`
	LastVerifiedAt *string `json:"last_verified_at"`
	ConnectedAt    *string `json:"connected_at"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`

	// Inbound webhook configuration. WebhookURL is the URL the merchant pastes
	// into the provider dashboard; the remaining fields describe what the
	// provider requires so the UI never special-cases a provider by name.
	WebhookURL             string  `json:"webhook_url,omitempty"`
	WebhookSecretRequired  bool    `json:"webhook_secret_required"`
	WebhookSecretLabel     string  `json:"webhook_secret_label,omitempty"`
	WebhookSignatureHeader string  `json:"webhook_signature_header,omitempty"`
	WebhookDashboardPath   string  `json:"webhook_dashboard_path,omitempty"`
	WebhookSecretMasked    string  `json:"webhook_secret_masked,omitempty"`
	WebhookSecretSet       bool    `json:"webhook_secret_set"`
	WebhookLastEventAt     *string `json:"webhook_last_event_at,omitempty"`

	// SupportedOptions lists the payment.options fields this provider accepts.
	// The dashboard renders its inputs from this list, so adding a provider
	// option never requires a frontend special-case.
	SupportedOptions []string `json:"supported_options"`
}

// ToConnectionResponse maps a connection onto the public DTO.
func ToConnectionResponse(c *Connection) ConnectionResponse {
	return ConnectionResponse{
		ID:                  c.ID,
		Provider:            c.Provider,
		Environment:         c.Environment,
		DisplayName:         c.DisplayName,
		Status:              c.Status,
		IsDefault:           c.IsDefault,
		Secret:              c.SecretLast4,
		Currency:            c.Currency,
		LastVerifiedAt:      formatTime(c.LastVerifiedAt),
		ConnectedAt:         formatTime(c.ConnectedAt),
		CreatedAt:           c.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:           c.UpdatedAt.UTC().Format(time.RFC3339),
		WebhookSecretMasked: c.WebhookSecretMasked,
		WebhookLastEventAt:  formatTime(c.WebhookLastEventAt),
		SupportedOptions:    OptionKeys(constants.Provider(c.Provider)),
	}
}

// ProviderDescriptor advertises a supported provider, including what its inbound
// webhooks require, so the dashboard never special-cases a provider by name.
type ProviderDescriptor struct {
	Provider                  string `json:"provider"`
	Default                   string `json:"default_currency"`
	WebhookSecretRequired     bool   `json:"webhook_secret_required"`
	WebhookSecretLabel        string `json:"webhook_secret_label,omitempty"`
	WebhookSecretSelfAssigned bool   `json:"webhook_secret_self_assigned"`
	WebhookSignatureHeader    string `json:"webhook_signature_header,omitempty"`
	WebhookDashboardPath      string `json:"webhook_dashboard_path,omitempty"`
	// SupportedOptions is the payment.options vocabulary this provider accepts.
	SupportedOptions []string `json:"supported_options"`
}

func formatTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	formatted := t.UTC().Format(time.RFC3339)
	return &formatted
}

// Descriptors lists the providers a client may request.
func Descriptors() []ProviderDescriptor {
	out := make([]ProviderDescriptor, 0, len(constants.AllProviders()))
	for _, provider := range constants.AllProviders() {
		support := constants.WebhookSupportFor(provider)
		out = append(out, ProviderDescriptor{
			Provider:                  provider.String(),
			Default:                   DefaultCurrencyFor(provider),
			WebhookSecretRequired:     support.RequiresWebhookSecret,
			WebhookSecretLabel:        support.SecretLabel,
			WebhookSecretSelfAssigned: support.SecretSelfAssigned,
			WebhookSignatureHeader:    support.SignatureHeader,
			WebhookDashboardPath:      support.DashboardPath,
			SupportedOptions:          OptionKeys(provider),
		})
	}
	return out
}
