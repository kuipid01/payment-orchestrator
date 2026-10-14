// Package routing decides which provider handles a payment. Rules are
// merchant-scoped and evaluated per (currency, country, amount); when no rule
// matches, the merchant's default connected provider is used.
package routing

import (
	"time"

	"github.com/kuipid01/payconnect/internal/platform/gormx"
)

// Preference is one provider entry inside a rule.
type Preference struct {
	Provider string `json:"provider"`
	Priority int    `json:"priority"`
}

// Rule maps a payment shape onto an ordered list of providers.
type Rule struct {
	ID         string                    `gorm:"primaryKey;size:64" json:"id"`
	MerchantID string                    `gorm:"column:merchant_id" json:"-"`
	Name       string                    `gorm:"column:name" json:"name"`
	Currency   string                    `gorm:"column:currency" json:"currency"`
	Country    string                    `gorm:"column:country" json:"country"`
	MinAmount  int64                     `gorm:"column:min_amount" json:"min_amount"`
	MaxAmount  int64                     `gorm:"column:max_amount" json:"max_amount"`
	Providers  gormx.JSONB[[]Preference] `gorm:"column:providers" json:"providers"`
	Priority   int                       `gorm:"column:priority" json:"priority"`
	Enabled    bool                      `gorm:"column:enabled" json:"enabled"`
	CreatedAt  time.Time                 `gorm:"column:created_at" json:"created_at"`
	UpdatedAt  time.Time                 `gorm:"column:updated_at" json:"updated_at"`
}

// TableName binds the model to the routing_rules table.
func (Rule) TableName() string { return "routing_rules" }

// Matches reports whether a rule applies to a payment shape.
func (r *Rule) Matches(currency string, amountMinor int64) bool {
	if !r.Enabled {
		return false
	}
	if r.Currency != "" && r.Currency != currency {
		return false
	}
	if r.MinAmount > 0 && amountMinor < r.MinAmount {
		return false
	}
	if r.MaxAmount > 0 && amountMinor > r.MaxAmount {
		return false
	}
	return true
}

// OrderedProviders returns the rule's providers sorted by priority.
func (r *Rule) OrderedProviders() []string {
	preferences := append([]Preference(nil), r.Providers.Data...)
	for i := 1; i < len(preferences); i++ {
		for j := i; j > 0 && preferences[j-1].Priority > preferences[j].Priority; j-- {
			preferences[j-1], preferences[j] = preferences[j], preferences[j-1]
		}
	}
	out := make([]string, 0, len(preferences))
	for _, preference := range preferences {
		if preference.Provider != "" {
			out = append(out, preference.Provider)
		}
	}
	return out
}
