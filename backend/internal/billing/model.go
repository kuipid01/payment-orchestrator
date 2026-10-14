package billing

import (
	"time"

	"github.com/kuipid01/payconnect/internal/platform/gormx"
)

// Subscription is one organization's billing state. There is exactly one row
// per merchant (unique index), because a merchant switching band — or paying by
// card on a Naira organization — does not get a second subscription.
//
// Note what is *not* here: merchants has no plan column. Entitlements come
// through this table and nowhere else, which is the only reason the pricing and
// billing pages can never disagree about what a merchant is on.
type Subscription struct {
	ID         string `gorm:"primaryKey;size:64" json:"id"`
	MerchantID string `gorm:"column:merchant_id;uniqueIndex;size:64" json:"merchant_id"`
	// Plan is a catalog code (free | lite | pro | enterprise).
	Plan string `gorm:"column:plan;size:32" json:"plan"`
	// Status is the subscription lifecycle state (see StatusFree …).
	Status string `gorm:"column:status;size:32" json:"status"`
	// Interval is monthly | quarterly | annual (none on Free).
	Interval string `gorm:"column:interval;size:16" json:"interval"`
	// Band and Currency are recorded, not re-derived on read: a merchant who
	// moves country keeps the price they agreed to until the next term.
	Band     string `gorm:"column:band;size:2" json:"band"`
	Currency string `gorm:"column:currency;size:8" json:"currency"`
	// Provider names who collects (paystack | stripe), and
	// ProviderSubscriptionID is their handle for it. Both are empty until
	// checkout is wired.
	Provider               string     `gorm:"column:provider;size:32" json:"provider"`
	ProviderSubscriptionID string     `gorm:"column:provider_subscription_id;size:128" json:"provider_subscription_id"`
	CurrentPeriodStart     *time.Time `gorm:"column:current_period_start" json:"current_period_start,omitempty"`
	CurrentPeriodEnd       *time.Time `gorm:"column:current_period_end" json:"current_period_end,omitempty"`
	CancelAtPeriodEnd      bool       `gorm:"column:cancel_at_period_end" json:"cancel_at_period_end"`
	TrialEndsAt            *time.Time `gorm:"column:trial_ends_at" json:"trial_ends_at,omitempty"`
	// TrialUsed is permanent: the trial is once per organization, even after a
	// cancellation, and survives a return to Free.
	TrialUsed bool      `gorm:"column:trial_used" json:"trial_used"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName binds the model to the subscriptions table.
func (Subscription) TableName() string { return "merchant_subscriptions" }

// IsEnded reports whether a row represents no plan at all.
func (s *Subscription) IsEnded() bool {
	return s == nil || s.Status == StatusFree || s.Status == StatusCancelled
}

// Event is one subscription state change. Every plan change is a billing fact,
// so the trail is append-only and never edited.
type Event struct {
	ID         string `gorm:"primaryKey;size:64" json:"id"`
	MerchantID string `gorm:"column:merchant_id;size:64;index" json:"merchant_id"`
	// Type is subscribe | change_plan | trial_started | trial_ended | cancel |
	// resume | checkout_requested | payment_failed | reconciled.
	Type string `gorm:"column:type;size:32" json:"type"`
	// FromPlan/ToPlan are the catalog codes on either side of the change.
	FromPlan string `gorm:"column:from_plan;size:32" json:"from_plan"`
	ToPlan   string `gorm:"column:to_plan;size:32" json:"to_plan"`
	// Source is api | provider_webhook | admin | system.
	Source string `gorm:"column:source;size:32" json:"source"`
	// Payload holds redacted detail (interval, band, provider event id). It must
	// never contain a credential.
	Payload   gormx.JSONB[map[string]any] `gorm:"column:payload" json:"payload,omitempty"`
	ActorName string                      `gorm:"column:actor_name;size:128" json:"actor_name"`
	ActorRole string                      `gorm:"column:actor_role;size:64" json:"actor_role"`
	Detail    string                      `gorm:"column:detail" json:"detail"`
	CreatedAt time.Time                   `gorm:"column:created_at" json:"created_at"`
}

// TableName binds the model to the subscription_events table.
func (Event) TableName() string { return "subscription_events" }

// Event types.
const (
	EventSubscribe         = "subscribe"
	EventChangePlan        = "change_plan"
	EventTrialStarted      = "trial_started"
	EventTrialEnded        = "trial_ended"
	EventCancel            = "cancel"
	EventResume            = "resume"
	EventCheckoutRequested = "checkout_requested"
	EventPaymentFailed     = "payment_failed"
	EventReconciled        = "reconciled"
	// EventDowngraded is the end of the dunning path: an unpaid plan whose grace
	// period ran out falls back to Free rather than being suspended.
	EventDowngraded = "downgraded"
)

// Event sources.
const (
	SourceAPI      = "api"
	SourceProvider = "provider_webhook"
	SourceAdmin    = "admin"
	SourceSystem   = "system"
)

// NewSubscriptionID mints a subscription identifier.
func NewSubscriptionID() string { return gormx.NewID("sub") }

// NewEventID mints a subscription-event identifier.
func NewEventID() string { return gormx.NewID("subev") }
