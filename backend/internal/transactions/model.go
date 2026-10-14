// Package transactions is the financial ledger view: every settled or failed
// money movement is persisted here once, so reporting never re-reads provider
// responses.
package transactions

import "time"

// Transaction is one normalized financial event.
type Transaction struct {
	ID          string    `gorm:"primaryKey;size:64" json:"id"`
	MerchantID  string    `gorm:"column:merchant_id" json:"-"`
	PaymentID   *string   `gorm:"column:payment_id" json:"payment_id,omitempty"`
	RefundID    *string   `gorm:"column:refund_id" json:"refund_id,omitempty"`
	Provider    string    `gorm:"column:provider" json:"provider"`
	Environment string    `gorm:"column:environment" json:"environment"`
	Type        string    `gorm:"column:type" json:"type"` // payment | refund | chargeback
	Direction   string    `gorm:"column:direction" json:"direction"`
	AmountMinor int64     `gorm:"column:amount_minor" json:"amount"`
	Currency    string    `gorm:"column:currency" json:"currency"`
	Status      string    `gorm:"column:status" json:"status"`
	Reference   string    `gorm:"column:reference" json:"reference"`
	Description string    `gorm:"column:description" json:"description"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"created_at"`
}

// TableName binds the model to the transactions table.
func (Transaction) TableName() string { return "transactions" }
