package developer

import (
	"time"

	"github.com/kuipid01/payconnect/internal/platform/gormx"
)

type Developer struct {
	ID          string                `gorm:"primaryKey;size:64" json:"id"`
	Name        string                `json:"name"`
	Email       string                `gorm:"uniqueIndex" json:"email"`
	Permissions gormx.JSONB[[]string] `json:"permissions"`
	MerchantID  string                `gorm:"column:merchant_id" json:"merchant_id,omitempty"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
}
