// Package refunds owns refunds against settled payments: validation, provider
// execution, the payment status transition and the ledger entry.
package refunds

import (
	"time"

	"github.com/kuipid01/payconnect/internal/platform/gormx"
)

// Refund is a full or partial reversal of one payment.
type Refund struct {
	ID               string    `gorm:"primaryKey;size:64" json:"id"`
	MerchantID       string    `gorm:"column:merchant_id" json:"-"`
	PaymentID        string    `gorm:"column:payment_id" json:"payment_id"`
	Provider         string    `gorm:"column:provider" json:"provider"`
	ProviderRefundID *string   `gorm:"column:provider_refund_id" json:"-"`
	AmountMinor      int64     `gorm:"column:amount_minor" json:"amount"`
	Currency         string    `gorm:"column:currency" json:"currency"`
	Status           string    `gorm:"column:status" json:"status"`
	Reason           string    `gorm:"column:reason" json:"reason"`
	FailureMessage   string    `gorm:"column:failure_message" json:"failure_message,omitempty"`
	IdempotencyKey   string    `gorm:"column:idempotency_key" json:"-"`
	CreatedAt        time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName binds the model to the refunds table.
func (Refund) TableName() string { return "refunds" }

// SetProviderRefundID records the provider-side refund identifier.
func (r *Refund) SetProviderRefundID(id string) {
	if id == "" {
		return
	}
	r.ProviderRefundID = &id
}

// NewID mints a refund identifier.
func NewID() string { return gormx.NewID("ref") }
