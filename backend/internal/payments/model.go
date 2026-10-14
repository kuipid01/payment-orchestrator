// Package payments owns the payment domain: intents, provider execution,
// idempotency, verification and the payment state machine.
package payments

import (
	"time"

	"github.com/kuipid01/payconnect/internal/platform/gormx"
)

// Payment is a PayConnect payment intent and its provider outcome. Provider
// statuses never appear here: the status field uses the public vocabulary.
type Payment struct {
	ID            string  `gorm:"primaryKey;size:64" json:"id"`
	MerchantID    string  `gorm:"column:merchant_id" json:"-"`
	ApplicationID *string `gorm:"column:application_id" json:"application_id,omitempty"`
	// APIKeyID is the credential that created the payment. Nil for rows written
	// before attribution existed, or for a payment created outside a key.
	APIKeyID *string `gorm:"column:api_key_id" json:"-"`
	// DeveloperAccessID names the developer grant whose key created the payment.
	// Nil means a general (team-shared) key, which is what the "general"
	// developer filter selects.
	DeveloperAccessID *string                     `gorm:"column:developer_access_id" json:"developer_access_id,omitempty"`
	Environment       string                      `gorm:"column:environment" json:"environment"`
	Reference         string                      `gorm:"column:reference" json:"reference"`
	Provider          string                      `gorm:"column:provider" json:"provider"`
	ProviderPaymentID *string                     `gorm:"column:provider_payment_id" json:"-"`
	AmountMinor       int64                       `gorm:"column:amount_minor" json:"amount"`
	Currency          string                      `gorm:"column:currency" json:"currency"`
	Status            string                      `gorm:"column:status" json:"status"`
	CustomerID        *string                     `gorm:"column:customer_id" json:"customer_id,omitempty"`
	Description       string                      `gorm:"column:description" json:"description,omitempty"`
	AuthorizationURL  string                      `gorm:"column:authorization_url" json:"authorization_url,omitempty"`
	IdempotencyKey    string                      `gorm:"column:idempotency_key" json:"-"`
	FailureCode       string                      `gorm:"column:failure_code" json:"failure_code,omitempty"`
	FailureMessage    string                      `gorm:"column:failure_message" json:"failure_message,omitempty"`
	Metadata          gormx.JSONB[map[string]any] `gorm:"column:metadata" json:"metadata,omitempty"`
	ProviderResponse  gormx.JSONB[map[string]any] `gorm:"column:provider_response" json:"-"`
	FeeMinor          *int64                      `gorm:"column:fee_minor" json:"fee_minor,omitempty"`
	Timeline          gormx.JSONB[[]Event]        `gorm:"column:timeline" json:"timeline,omitempty"`
	CreatedAt         time.Time                   `gorm:"column:created_at" json:"created_at"`
	UpdatedAt         time.Time                   `gorm:"column:updated_at" json:"updated_at"`
}

// TableName binds the model to the payments table.
func (Payment) TableName() string { return "payments" }

// Event is one entry in a payment's audit timeline.
type Event struct {
	ID      string    `json:"id"`
	Type    string    `json:"type"`
	Message string    `json:"message"`
	At      time.Time `json:"at"`
}

// AppendEvent adds a timeline entry. Cross-domain flows (refunds) use this to
// keep the payment's audit trail complete without reaching into its fields.
func (p *Payment) AppendEvent(eventType, message string, at time.Time) {
	p.appendEvent(eventType, message, at)
}

// appendEvent adds a timeline entry.
func (p *Payment) appendEvent(eventType, message string, at time.Time) {
	p.Timeline.Data = append(p.Timeline.Data, Event{
		ID:      gormx.NewID("evt"),
		Type:    eventType,
		Message: message,
		At:      at,
	})
}

// IdempotencyRecord stores the outcome of one idempotent request so a retry can
// be answered without re-executing the provider call.
type IdempotencyRecord struct {
	ID             string                      `gorm:"primaryKey;size:64"`
	MerchantID     string                      `gorm:"column:merchant_id"`
	IdempotencyKey string                      `gorm:"column:idempotency_key"`
	Endpoint       string                      `gorm:"column:endpoint"`
	RequestHash    string                      `gorm:"column:request_hash"`
	Status         string                      `gorm:"column:status"` // in_progress | completed
	ResponseBody   gormx.JSONB[map[string]any] `gorm:"column:response_body"`
	StatusCode     *int                        `gorm:"column:status_code"`
	PaymentID      *string                     `gorm:"column:payment_id"`
	CreatedAt      time.Time                   `gorm:"column:created_at"`
	UpdatedAt      time.Time                   `gorm:"column:updated_at"`
}

// TableName binds the record to the idempotency_keys table.
func (IdempotencyRecord) TableName() string { return "idempotency_keys" }

// Idempotency states.
const (
	idempotencyInProgress = "in_progress"
	idempotencyCompleted  = "completed"
)
