// Package events is the internal event bus: domains publish domain events and
// subscribers (webhook delivery, transactions ledger) react to them.
package events

import (
	"encoding/json"
	"time"

	"github.com/kuipid01/payconnect/internal/platform/gormx"
)

// Event is a normalized domain event, independent of any provider.
type Event struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	MerchantID  string          `json:"-"`
	Environment string          `json:"environment"`
	Reference   string          `json:"reference,omitempty"`
	PaymentID   string          `json:"payment_id,omitempty"`
	RefundID    string          `json:"refund_id,omitempty"`
	Data        json.RawMessage `json:"data,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

// New builds an event with an id and timestamp.
func New(eventType, merchantID, environment string) Event {
	return Event{
		ID:          gormx.NewID("evt"),
		Type:        eventType,
		MerchantID:  merchantID,
		Environment: environment,
		CreatedAt:   time.Now().UTC(),
	}
}
