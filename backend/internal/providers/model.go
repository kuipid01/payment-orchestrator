// Package providers owns the provider-credential domain and the adapter layer
// that talks to Paystack, Flutterwave and Moniepoint.
//
// Provider-specific payloads never leave this package's adapter subpackages:
// everything crossing the boundary is a PayConnect domain object.
package providers

import (
	"time"
)

// Credentials are the decrypted provider secrets needed to call a provider.
// They exist only in memory for the duration of one request.
type Credentials struct {
	SecretKey string
	PublicKey string
	// WebhookSecret is the provider-specific inbound webhook signing value when
	// it differs from the API secret key (Flutterwave's "verif-hash"). Empty for
	// providers that sign with SecretKey itself.
	WebhookSecret string
	Environment   string
}

// Connection is a merchant's encrypted provider credential set.
type Connection struct {
	ID                   string `gorm:"primaryKey;size:64" json:"id"`
	MerchantID           string `gorm:"column:merchant_id" json:"-"`
	Provider             string `gorm:"column:provider" json:"provider"`
	Environment          string `gorm:"column:environment" json:"environment"`
	DisplayName          string `gorm:"column:display_name" json:"display_name"`
	CredentialsEncrypted []byte `gorm:"column:credentials_encrypted" json:"-"`
	Status               string `gorm:"column:status" json:"status"`
	IsDefault            bool   `gorm:"column:is_default" json:"is_default"`
	SecretLast4          string `gorm:"column:secret_last4" json:"secret_last4"`
	// WebhookSecretEncrypted is the optional inbound webhook signing value for
	// providers whose signature key is not the API secret key.
	WebhookSecretEncrypted []byte `gorm:"column:webhook_secret_encrypted" json:"-"`
	WebhookSecretMasked    string `gorm:"column:webhook_secret_masked" json:"webhook_secret_masked"`
	// WebhookLastEventAt stamps the last verified inbound webhook for health.
	WebhookLastEventAt *time.Time `gorm:"column:webhook_last_event_at" json:"webhook_last_event_at,omitempty"`
	LastVerifiedAt     *time.Time `gorm:"column:last_verified_at" json:"last_verified_at"`
	ConnectedAt        *time.Time `gorm:"column:connected_at" json:"connected_at"`
	Currency           string     `gorm:"column:currency" json:"currency"`
	CreatedAt          time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

// TableName maps the connection onto the provider_connections table.
func (Connection) TableName() string { return "provider_connections" }

// IsConnected reports whether the connection is usable.
func (c *Connection) IsConnected() bool { return c != nil && c.Status == "connected" }

// PaymentRequest is a provider-independent charge request.
type PaymentRequest struct {
	Reference   string
	AmountMinor int64
	Currency    string
	Email       string
	Name        string
	Phone       string
	Description string
	CallbackURL string
	Metadata    map[string]any
}

// PaymentResult is a provider-independent charge (or verification) result.
type PaymentResult struct {
	ProviderReference string
	Status            string // normalized: pending | processing | successful | failed | cancelled | expired
	AmountMinor       int64
	Currency          string
	AuthorizationURL  string
	Channel           string
	FeeMinor          *int64
	FailureCode       string
	FailureMessage    string
	RawStatus         string // provider status, kept for audit only
	ProviderResponse  map[string]any
}

// RefundRequest is a provider-independent refund request.
type RefundRequest struct {
	PaymentReference  string
	ProviderReference string
	AmountMinor       int64
	Currency          string
	Reason            string
	Metadata          map[string]any
}

// RefundResult is a provider-independent refund result.
type RefundResult struct {
	ProviderRefundID string
	Status           string // pending | processing | successful | failed
	AmountMinor      int64
	Currency         string
	FailureMessage   string
}

// WebhookVerificationRequest carries an inbound provider webhook.
type WebhookVerificationRequest struct {
	Headers map[string]string
	Body    []byte
	// SecretKey is the connection's API secret key, the signing key for
	// providers that need no separate webhook secret.
	SecretKey string
	// WebhookSecret, when set, is the provider-specific signing value that takes
	// precedence over SecretKey (Flutterwave's "verif-hash").
	WebhookSecret string
	Signature     string
}

// WebhookResult is a normalized inbound provider event.
type WebhookResult struct {
	EventID           string
	Type              string // payment.succeeded | payment.failed | refund.succeeded | refund.failed | other
	Reference         string
	ProviderReference string
	AmountMinor       int64
	Currency          string
	Status            string
	SignatureValid    bool
	Raw               map[string]any
}
