// Package customers owns the payer directory. Customers are merchant-scoped and
// reusable across payments; the payment itself keeps only a customer id.
package customers

import (
	"time"

	"github.com/kuipid01/payconnect/internal/platform/gormx"
)

// Customer is a payer known to a merchant.
type Customer struct {
	ID         string                      `gorm:"primaryKey;size:64" json:"id"`
	MerchantID string                      `gorm:"column:merchant_id" json:"-"`
	Email      string                      `gorm:"column:email" json:"email"`
	Name       string                      `gorm:"column:name" json:"name"`
	Phone      string                      `gorm:"column:phone" json:"phone"`
	Metadata   gormx.JSONB[map[string]any] `gorm:"column:metadata" json:"metadata,omitempty"`
	Status     string                      `gorm:"column:status" json:"status"`
	CreatedAt  time.Time                   `gorm:"column:created_at" json:"created_at"`
	UpdatedAt  time.Time                   `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt  *time.Time                  `gorm:"column:deleted_at" json:"-"`
}

// TableName binds the model to the customers table.
func (Customer) TableName() string { return "customers" }
