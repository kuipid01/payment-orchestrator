// Package webhooks owns both directions of webhook traffic:
//
//   - inbound provider notifications (receive → verify → dedupe → reconcile),
//   - outbound merchant deliveries (sign → POST → retry → dead-letter).
//
// Provider webhook secrets and merchant signing secrets are encrypted at rest;
// neither is ever returned by an API.
package webhooks

import (
	"time"

	"github.com/kuipid01/payconnect/internal/platform/gormx"
)

// ProviderEvent is a deduplicated inbound provider notification. The unique
// (provider, provider_event_id) constraint is what makes receiver retries safe.
type ProviderEvent struct {
	ID              string                      `gorm:"primaryKey;size:64" json:"id"`
	MerchantID      string                      `gorm:"column:merchant_id" json:"-"`
	ConnectionID    string                      `gorm:"column:connection_id" json:"connection_id"`
	Provider        string                      `gorm:"column:provider" json:"provider"`
	ProviderEventID string                      `gorm:"column:provider_event_id" json:"provider_event_id"`
	EventType       string                      `gorm:"column:event_type" json:"event_type"`
	Payload         gormx.JSONB[map[string]any] `gorm:"column:payload" json:"payload,omitempty"`
	ProcessedAt     *time.Time                  `gorm:"column:processed_at" json:"processed_at,omitempty"`
	CreatedAt       time.Time                   `gorm:"column:created_at" json:"created_at"`
}

// TableName binds the model to the provider_events table.
func (ProviderEvent) TableName() string { return "provider_events" }

// Endpoint is a merchant-configured delivery target.
type Endpoint struct {
	ID                     string                `gorm:"primaryKey;size:64" json:"id"`
	MerchantID             string                `gorm:"column:merchant_id" json:"-"`
	ApplicationID          *string               `gorm:"column:application_id" json:"application_id,omitempty"`
	URL                    string                `gorm:"column:url" json:"url"`
	Environment            string                `gorm:"column:environment" json:"environment"`
	Status                 string                `gorm:"column:status" json:"status"`
	Events                 gormx.JSONB[[]string] `gorm:"column:events" json:"events"`
	SigningSecretEncrypted []byte                `gorm:"column:signing_secret_encrypted" json:"-"`
	SigningSecretMasked    string                `gorm:"column:signing_secret_masked" json:"signing_secret_masked"`
	FailureCount           int                   `gorm:"column:failure_count" json:"failure_count"`
	CreatedAt              time.Time             `gorm:"column:created_at" json:"created_at"`
	UpdatedAt              time.Time             `gorm:"column:updated_at" json:"updated_at"`
}

// TableName binds the model to the webhook_endpoints table.
func (Endpoint) TableName() string { return "webhook_endpoints" }

// Subscribes reports whether the endpoint wants an event type.
func (e *Endpoint) Subscribes(eventType string) bool {
	if len(e.Events.Data) == 0 {
		return true
	}
	for _, candidate := range e.Events.Data {
		if candidate == eventType {
			return true
		}
	}
	return false
}

// Delivery is one attempt record for an outbound merchant webhook.
type Delivery struct {
	ID           string `gorm:"primaryKey;size:64" json:"id"`
	MerchantID   string `gorm:"column:merchant_id" json:"-"`
	EndpointID   string `gorm:"column:endpoint_id" json:"endpoint_id"`
	Event        string `gorm:"column:event" json:"event"`
	Status       string `gorm:"column:status" json:"status"`
	Attempts     int    `gorm:"column:attempts" json:"attempts"`
	ResponseCode *int   `gorm:"column:response_code" json:"response_code,omitempty"`
	// ResponseBody is the head of the endpoint's reply from the most recent
	// attempt that reached it, so a merchant can see why it was rejected.
	ResponseBody  string                      `gorm:"column:response_body" json:"response_body,omitempty"`
	Payload       gormx.JSONB[map[string]any] `gorm:"column:payload" json:"payload"`
	LastError     string                      `gorm:"column:last_error" json:"last_error,omitempty"`
	NextAttemptAt *time.Time                  `gorm:"column:next_attempt_at" json:"next_attempt_at,omitempty"`
	CreatedAt     time.Time                   `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time                   `gorm:"column:updated_at" json:"updated_at"`
	DeliveredAt   *time.Time                  `gorm:"column:delivered_at" json:"delivered_at,omitempty"`
}

// TableName binds the model to the webhook_deliveries table.
func (Delivery) TableName() string { return "webhook_deliveries" }

// IsTerminal reports whether a delivery has stopped being retried.
func (d *Delivery) IsTerminal() bool {
	return d.Status == "delivered" || d.Status == "dead"
}
