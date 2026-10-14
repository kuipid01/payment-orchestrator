package billing

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/kuipid01/payconnect/internal/constants"
)

// Repository owns subscription persistence and the usage aggregates. It never
// writes a plan onto the merchants table: entitlements live here.
type Repository struct{ DB *gorm.DB }

// NewRepository builds the billing repository.
func NewRepository(db *gorm.DB) *Repository { return &Repository{DB: db} }

// SubscriptionByMerchant loads a merchant's subscription row.
func (r *Repository) SubscriptionByMerchant(merchantID string) (*Subscription, error) {
	var row Subscription
	if err := r.DB.Where("merchant_id = ?", merchantID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// Create inserts a subscription row.
func (r *Repository) Create(sub *Subscription) error { return r.DB.Create(sub).Error }

// Save persists a subscription row.
func (r *Repository) Save(sub *Subscription) error { return r.DB.Save(sub).Error }

// RecordEvent appends one state change to the trail.
func (r *Repository) RecordEvent(event *Event) error { return r.DB.Create(event).Error }

// Events returns a merchant's most recent state changes, newest first.
func (r *Repository) Events(merchantID string, limit int) ([]Event, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var rows []Event
	if err := r.DB.Where("merchant_id = ?", merchantID).
		Order("created_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

/* ---------------------------------- counts --------------------------------- */

// The counts below answer one question each — "how much of this does the
// organization have right now?" — and they are the measurements the plan gates
// refuse on. They count what exists, not what was created this month: a plan
// limit is about the shape of an account, not about a calendar.

// CountApplications returns the organization's live applications.
func (r *Repository) CountApplications(merchantID string) (int64, error) {
	return r.count(r.DB.Model(&applicationRow{}).
		Where("merchant_id = ? AND deleted_at IS NULL", merchantID))
}

// CountSeats returns the people who hold a seat: active memberships plus the
// invitations that are still pending. A pending invitation is counted because it
// will become a seat the moment it is accepted — otherwise the seat limit could
// be outrun by inviting a crowd and having none of them accept.
func (r *Repository) CountSeats(merchantID string) (int64, error) {
	members, err := r.count(r.DB.Model(&membershipRow{}).
		Where("merchant_id = ? AND status <> ?", merchantID, "revoked"))
	if err != nil {
		return 0, err
	}
	pending, err := r.count(r.DB.Model(&invitationRow{}).
		Where("merchant_id = ? AND status = ?", merchantID, "pending"))
	if err != nil {
		return 0, err
	}
	return members + pending, nil
}

// CountWebhookEndpoints returns the organization's delivery endpoints.
func (r *Repository) CountWebhookEndpoints(merchantID string) (int64, error) {
	return r.count(r.DB.Model(&endpointRow{}).Where("merchant_id = ?", merchantID))
}

// CountRoutingRules returns the organization's routing rules.
func (r *Repository) CountRoutingRules(merchantID string) (int64, error) {
	return r.count(r.DB.Model(&ruleRow{}).Where("merchant_id = ?", merchantID))
}

// count runs one COUNT against a prepared query.
func (r *Repository) count(query *gorm.DB) (int64, error) {
	var total int64
	return total, query.Count(&total).Error
}

/* ---------------------------------- usage ---------------------------------- */

// UsageSnapshot is one calendar month of measured usage, straight from the
// tables that own the facts: applications, memberships, payments and refunds.
//
// Usage is derived, never counted into a separate ledger. A counter that can
// disagree with the payments table is a support ticket we would deserve.
type UsageSnapshot struct {
	Applications int64
	Seats        int64
	Payments     int64
	// RoutedVolume is successful live volume this month, net of successful
	// refunds, in the merchant's billing currency (minor units).
	RoutedVolume int64
	// RefundedVolume is the part of that netting; reported so the dashboard can
	// show why the volume moved.
	RefundedVolume int64
}

// UsageFor measures a merchant's month. `from` is inclusive, `to` exclusive.
//
// Only the live environment is counted: test traffic is not routed money, and
// billing it would be indefensible. Only the merchant's own billing currency is
// summed, because adding naira to dollars produces a number with no meaning —
// a merchant routing two currencies is measured in their billing currency
// until per-currency conversion exists.
func (r *Repository) UsageFor(merchantID, currency string, from, to time.Time) (UsageSnapshot, error) {
	var snap UsageSnapshot

	// Applications and seats are counted by the same methods the plan gates use,
	// so the meter a merchant reads and the limit that refuses them are one
	// number rather than two that drift apart.
	applications, err := r.CountApplications(merchantID)
	if err != nil {
		return snap, err
	}
	snap.Applications = applications

	seats, err := r.CountSeats(merchantID)
	if err != nil {
		return snap, err
	}
	snap.Seats = seats

	live := "merchant_id = ? AND environment = ? AND currency = ? AND created_at >= ? AND created_at < ?"
	liveArgs := []any{merchantID, constants.EnvironmentLive, currency, from, to}

	if err := r.DB.Model(&paymentRow{}).
		Where(live, liveArgs...).
		Count(&snap.Payments).Error; err != nil {
		return snap, err
	}

	// Volume counts money that was actually routed, and a refund rewrites the
	// payment's status — a refunded payment is no longer "successful". So the
	// summed set is every payment that collected money, including the ones a
	// refund has since marked, and the refund rows below take back the part
	// that was returned. Counting only status='successful' and then subtracting
	// refunds would take a full refund off twice.
	var volume struct{ Total *int64 }
	if err := r.DB.Model(&paymentRow{}).
		Select("SUM(amount_minor) AS total").
		Where(live+" AND status IN ?", append(liveArgs, []string{
			string(constants.PaymentSuccessful),
			string(constants.PaymentRefunded),
			string(constants.PaymentPartiallyRefunded),
		})...).
		Scan(&volume).Error; err != nil {
		return snap, err
	}

	// Refunds are joined to their payment so a refund of a test payment never
	// nets against live volume. The refunds table has no environment column of
	// its own — the payment is the environment.
	var refunded struct{ Total *int64 }
	if err := r.DB.Table("refunds AS r").
		Select("SUM(r.amount_minor) AS total").
		Joins("JOIN payments AS p ON p.id = r.payment_id").
		Where("r.merchant_id = ? AND r.status = ? AND p.environment = ? AND r.currency = ? AND r.created_at >= ? AND r.created_at < ?",
			merchantID, string(constants.RefundSuccessful), constants.EnvironmentLive, currency, from, to).
		Scan(&refunded).Error; err != nil {
		return snap, err
	}

	snap.RoutedVolume = valueOrZero(volume.Total)
	snap.RefundedVolume = valueOrZero(refunded.Total)
	snap.RoutedVolume -= snap.RefundedVolume
	if snap.RoutedVolume < 0 {
		snap.RoutedVolume = 0
	}
	return snap, nil
}

func valueOrZero(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

// IsNotFound reports whether an error is GORM's record-not-found, so callers
// outside this package do not have to import gorm just to ask.
func IsNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }

// The aggregates scan into the smallest possible row shapes rather than the
// owning domain's model, so billing never depends on another package's struct
// layout — only on the column names it reads.
type applicationRow struct{}

func (applicationRow) TableName() string { return "applications" }

type membershipRow struct{}

func (membershipRow) TableName() string { return "user_merchants" }

type invitationRow struct{}

func (invitationRow) TableName() string { return "developer_merchant_access" }

type endpointRow struct{}

func (endpointRow) TableName() string { return "webhook_endpoints" }

type ruleRow struct{}

func (ruleRow) TableName() string { return "routing_rules" }

type paymentRow struct{}

func (paymentRow) TableName() string { return "payments" }
