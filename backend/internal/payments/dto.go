package payments

import (
	"strings"
	"time"
	"unicode"

	"github.com/kuipid01/payconnect/internal/providers"
)

// CreateRequest is the public payload for POST /v1/payments.
type CreateRequest struct {
	Amount      int64          `json:"amount"`
	AmountMinor int64          `json:"amount_minor"`
	Currency    string         `json:"currency"`
	Reference   string         `json:"reference"`
	Adapter     string         `json:"adapter"`
	Fallback    bool           `json:"fallback"`
	Description string         `json:"description"`
	CallbackURL string         `json:"callback_url"`
	Customer    CustomerInput  `json:"customer"`
	Email       string         `json:"email,omitempty"`
	Metadata    map[string]any `json:"metadata"`
	// Options carries provider-specific settings, namespaced per provider
	// (options.stripe, options.paypal). Supplying options pins the adapter to
	// that provider, disables fallback, and is validated before any provider
	// call — the common fields above stay provider-neutral.
	Options providers.PaymentOptions `json:"options,omitempty"`
}

// CustomerInput is the payer block.
type CustomerInput struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

// CustomerRef is the payer as the dashboard renders it: an id plus the display
// fields. The customer directory resolves it, so the payments domain never
// carries a second copy of the payer's details.
type CustomerRef struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// Response is the canonical public payment representation.
type Response struct {
	ID                string         `json:"id"`
	Reference         string         `json:"reference"`
	Amount            int64          `json:"amount"`
	Currency          string         `json:"currency"`
	Status            string         `json:"status"`
	Provider          string         `json:"provider"`
	Environment       string         `json:"environment"`
	AuthorizationURL  string         `json:"authorization_url,omitempty"`
	Description       string         `json:"description,omitempty"`
	CustomerID        string         `json:"customer_id,omitempty"`
	Customer          *CustomerRef   `json:"customer,omitempty"`
	ApplicationID     string         `json:"application_id,omitempty"`
	APIKeyID          string         `json:"api_key_id,omitempty"`
	DeveloperAccessID string         `json:"developer_access_id,omitempty"`
	FeeMinor          *int64         `json:"fee_minor,omitempty"`
	FailureCode       string         `json:"failure_code,omitempty"`
	FailureMessage    string         `json:"failure_message,omitempty"`
	Metadata          map[string]any `json:"metadata,omitempty"`
	Timeline          []Event        `json:"timeline,omitempty"`
	// ProviderResponse is the provider's own answer with credential-grade
	// material stripped. It is a display convenience: the vault holds the only
	// complete copy, and nothing here is ever written back.
	ProviderResponse map[string]any `json:"provider_response,omitempty"`
	CreatedAt        string         `json:"created_at"`
	UpdatedAt        string         `json:"updated_at"`
}

// ToResponse maps the domain model onto the public DTO. The raw provider
// response is redacted, never copied verbatim.
func ToResponse(p *Payment) Response {
	out := Response{
		ID:               p.ID,
		Reference:        p.Reference,
		Amount:           p.AmountMinor,
		Currency:         p.Currency,
		Status:           p.Status,
		Provider:         p.Provider,
		Environment:      p.Environment,
		AuthorizationURL: p.AuthorizationURL,
		Description:      p.Description,
		FeeMinor:         p.FeeMinor,
		FailureCode:      p.FailureCode,
		FailureMessage:   p.FailureMessage,
		Metadata:         p.Metadata.Data,
		Timeline:         p.Timeline.Data,
		ProviderResponse: RedactProviderResponse(p.ProviderResponse.Data),
		CreatedAt:        p.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:        p.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if p.CustomerID != nil {
		out.CustomerID = *p.CustomerID
	}
	if p.ApplicationID != nil {
		out.ApplicationID = *p.ApplicationID
	}
	if p.APIKeyID != nil {
		out.APIKeyID = *p.APIKeyID
	}
	if p.DeveloperAccessID != nil {
		out.DeveloperAccessID = *p.DeveloperAccessID
	}
	return out
}

// sensitiveKeyTokens are key names that never leave the API, whatever a provider
// chooses to call them. Matching is per word: "authorization_url" is display-safe
// while a nested "authorization_code" is not, and only the latter is caught.
var sensitiveKeyTokens = map[string]struct{}{
	"authorization": {}, "auth": {}, "card": {}, "cvv": {}, "cvc": {},
	"key": {}, "keys": {}, "pan": {}, "password": {}, "pin": {},
	"secret": {}, "signature": {}, "token": {},
}

// credentialPrefixes are PayConnect's own key prefixes; a value carrying one is
// redacted even when its key name looks innocent.
var credentialPrefixes = []string{"pc_live_", "pc_test_", "sk_live_", "sk_test_", "whsec_"}

// RedactProviderResponse returns the provider's answer with credential-grade
// fields masked, so the dashboard can show what the provider said without ever
// echoing secret material. The original is untouched.
func RedactProviderResponse(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	redacted, _ := redactValue(raw, 0).(map[string]any)
	return redacted
}

func redactValue(value any, depth int) any {
	// Providers nest aggressively; the bound keeps a hostile payload from
	// turning a listing into an expensive traversal.
	if depth > 6 {
		return "[truncated]"
	}
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, nested := range typed {
			if isSensitiveKey(key) {
				out[key] = "[redacted]"
				continue
			}
			out[key] = redactValue(nested, depth+1)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, nested := range typed {
			out = append(out, redactValue(nested, depth+1))
		}
		return out
	case string:
		if looksLikeCredential(typed) {
			return "[redacted]"
		}
		return typed
	default:
		return value
	}
}

func isSensitiveKey(key string) bool {
	for _, token := range strings.FieldsFunc(strings.ToLower(key), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if _, sensitive := sensitiveKeyTokens[token]; sensitive {
			return true
		}
	}
	return false
}

func looksLikeCredential(value string) bool {
	for _, prefix := range credentialPrefixes {
		if strings.Contains(value, prefix) {
			return true
		}
	}
	return false
}

// MetricsResponse is the dashboard aggregate payload.
type MetricsResponse struct {
	VolumeMinor  int64         `json:"volume_minor"`
	Currency     string        `json:"currency"`
	Successful   int64         `json:"successful"`
	Failed       int64         `json:"failed"`
	SuccessRate  float64       `json:"success_rate"`
	VolumeSeries []SeriesPoint `json:"volume_series"`
}

// SeriesPoint is one day of the dashboard series.
type SeriesPoint struct {
	Date        string `json:"date"`
	VolumeMinor int64  `json:"volume_minor"`
}
