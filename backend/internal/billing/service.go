package billing

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/kuipid01/payconnect/internal/audit"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/merchants"
	"github.com/kuipid01/payconnect/internal/platform/gormx"
	"github.com/kuipid01/payconnect/internal/redis"
)

// graceDays is how long a plan keeps working after its period ends. Account
// management degrades on this clock; money movement never stops on it.
const (
	graceDays    = 40
	planCacheTTL = 60 * time.Second
)

// MerchantLookup resolves an organization. Billing needs it for the billing
// country, because the price band is derived from it and can never be chosen.
type MerchantLookup interface {
	MerchantByID(id string) (*merchants.Merchant, error)
}

// Service owns subscription state: what plan a merchant is on, what that plan
// grants, and what a change costs.
type Service struct {
	Repo      *Repository
	Merchants MerchantLookup
	Store     redis.Store
	Audit     audit.Recorder
	Logger    *slog.Logger
	// Now is injectable so trial and period boundaries are testable.
	Now func() time.Time
	// CheckoutEnabled says whether a collection provider is wired. It is false
	// until phase 4: a chosen plan is recorded and reported as pending, and
	// nothing is charged.
	CheckoutEnabled bool
	// enforced names the gates allowed to refuse, set once at startup from
	// BILLING_ENFORCED (see enforcer.go). While it is empty every gate is
	// log-only, which is the default and the only state that has been earned.
	enforced map[Gate]bool
}

// NewService builds the billing service.
func NewService(repo *Repository, merchants MerchantLookup, store redis.Store, recorder audit.Recorder, logger *slog.Logger) *Service {
	return &Service{Repo: repo, Merchants: merchants, Store: store, Audit: recorder, Logger: logger}
}

// View is a merchant's resolved billing state: the plan, where it came from and
// when it lapses. It carries no usage; that is measured separately.
type View struct {
	Plan               Plan
	Status             string
	Interval           string
	CurrentPeriodStart *time.Time
	CurrentPeriodEnd   *time.Time
	TrialEndsAt        *time.Time
	CancelAtPeriodEnd  bool
	TrialUsed          bool
	Provider           string
	CheckoutEnabled    bool
	// Notice explains a state the merchant should understand, e.g. that a
	// chosen plan is recorded but not yet billed.
	Notice string
}

// Usage is one calendar month measured against a plan. The plan travels with
// the measurement so a caller can render limits and usage from one value.
type Usage struct {
	plan     Plan
	Period   string
	PlanCode string
	Currency string

	Applications int64
	Seats        int64
	Payments     int64
	// RoutedVolume is live volume we carried this month, net of refunds.
	RoutedVolume   int64
	RefundedVolume int64

	// The month's volume-fee arithmetic, in the plan's currency: what is
	// included free, what the fee reached, and what it would have been with no
	// cap (so the cap can be seen doing something).
	Allowance   int64
	Cap         int64
	Ceiling     int64
	Fee         int64
	UncappedFee int64
}

// resolved is the internal shape both Plan and Usage read from: the row, the
// plan the merchant may actually use, and — when those differ — the plan they
// asked for.
type resolved struct {
	sub *Subscription
	// plan is the effective plan: what entitlements apply right now.
	plan Plan
	// chosen is the plan a pending subscription names, for display only. It is
	// never an entitlement: a plan nobody has paid for is not a plan.
	chosen Plan
}

// Plan resolves a merchant's current plan and billing state.
func (s *Service) Plan(ctx context.Context, merchantID string) (*View, error) {
	merchant, err := s.merchant(merchantID)
	if err != nil {
		return nil, err
	}
	res, err := s.resolve(ctx, merchant)
	if err != nil {
		return nil, err
	}
	return s.view(res), nil
}

// Usage measures the current calendar month against the merchant's plan.
func (s *Service) Usage(ctx context.Context, merchantID string) (*Usage, error) {
	merchant, err := s.merchant(merchantID)
	if err != nil {
		return nil, err
	}
	res, err := s.resolve(ctx, merchant)
	if err != nil {
		return nil, err
	}
	start, end, label := Period(s.now())
	snapshot, err := s.Repo.UsageFor(merchantID, res.plan.Currency, start, end)
	if err != nil {
		return nil, err
	}
	return &Usage{
		plan:           res.plan,
		Period:         label,
		PlanCode:       res.plan.Code,
		Currency:       res.plan.Currency,
		Applications:   snapshot.Applications,
		Seats:          snapshot.Seats,
		Payments:       snapshot.Payments,
		RoutedVolume:   snapshot.RoutedVolume,
		RefundedVolume: snapshot.RefundedVolume,
		Allowance:      res.plan.Entitlements.RoutedVolumeAllowance,
		Cap:            res.plan.Entitlements.RoutedVolumeCap,
		Ceiling:        res.plan.Entitlements.RoutedVolumeCeiling,
		Fee:            VolumeFee(snapshot.RoutedVolume, res.plan),
		UncappedFee:    UncappedVolumeFee(snapshot.RoutedVolume, res.plan),
	}, nil
}

// Entitlements resolves what a merchant's plan grants, for the gates that
// enforce it.
//
// It fails open. If the subscription cannot be read, the merchant is treated as
// Free — never as "over limit" — because a billing read failure must not stop a
// payment, and every gate built on this is log-only until it has proven itself.
func (s *Service) Entitlements(ctx context.Context, merchantID string) Entitlements {
	plan, err := s.cachedPlan(ctx, merchantID)
	if err != nil {
		s.warn("entitlements unavailable; treating the merchant as Free",
			"merchant_id", merchantID, "error", err)
		return FreePlan(BandA).Entitlements
	}
	return plan.Entitlements
}

// EntitlementsFor is the gate-friendly variant: it returns the plan alongside
// its entitlements, so an upgrade prompt can name the plan the merchant is on.
func (s *Service) EntitlementsFor(ctx context.Context, merchantID string) (Plan, error) {
	return s.cachedPlan(ctx, merchantID)
}

// Events returns the merchant's recent subscription state changes.
func (s *Service) Events(ctx context.Context, merchantID string, limit int) ([]Event, error) {
	return s.Repo.Events(merchantID, limit)
}

/* --------------------------------- mutations -------------------------------- */

// StartTrial begins the 14-day Pro trial: no card, once per organization.
//
// The trial hands out Pro, and expiry lands on Lite rather than Free, so the
// merchant keeps routing, webhooks and replay while they decide — a much softer
// landing than losing half the product.
func (s *Service) StartTrial(ctx context.Context, merchantID, actorName, actorRole string) (*View, error) {
	merchant, err := s.merchant(merchantID)
	if err != nil {
		return nil, err
	}
	res, err := s.resolve(ctx, merchant)
	if err != nil {
		return nil, err
	}
	if res.sub != nil && res.sub.TrialUsed {
		return nil, TrialAlreadyUsed()
	}
	if res.plan.Code != PlanFree {
		return nil, apperrors.Conflict("already_subscribed", "This organization is already on a paid plan.")
	}

	now := s.now()
	trialEnds := now.AddDate(0, 0, TrialDays)
	trial, err := PlanFor(res.plan.Band, PlanPro)
	if err != nil {
		return nil, err
	}
	sub := res.sub
	if sub == nil {
		sub = &Subscription{ID: NewSubscriptionID(), MerchantID: merchantID, CreatedAt: now}
	}
	sub.Plan = PlanPro
	sub.Status = StatusTrialing
	sub.Interval = IntervalMonthly
	sub.Band = string(trial.Band)
	sub.Currency = trial.Currency
	sub.CurrentPeriodStart = &now
	sub.CurrentPeriodEnd = &trialEnds
	sub.TrialEndsAt = &trialEnds
	sub.TrialUsed = true
	sub.CancelAtPeriodEnd = false
	sub.UpdatedAt = now
	if err := s.persist(ctx, sub); err != nil {
		return nil, err
	}
	s.record(ctx, merchantID, EventTrialStarted, PlanFree, PlanPro, SourceAPI, actorName, actorRole,
		"14-day Pro trial started; expires "+trialEnds.Format(time.RFC3339),
		map[string]any{"interval": sub.Interval, "band": sub.Band})
	s.clearCache(ctx, merchantID)

	return s.Plan(ctx, merchantID)
}

// RequestCheckout records the plan and cadence a merchant chose.
//
// While no collection provider is configured this is where the flow stops: the
// choice is stored as pending_payment, nothing is charged, and the view says so
// in as many words. When a provider is wired, the session URL is created here
// and returned; the rest of the flow already reads the same row.
func (s *Service) RequestCheckout(ctx context.Context, merchantID, actorName, actorRole, planCode, interval string) (*View, error) {
	merchant, err := s.merchant(merchantID)
	if err != nil {
		return nil, err
	}
	res, err := s.resolve(ctx, merchant)
	if err != nil {
		return nil, err
	}

	code := NormalizePlanCode(planCode)
	switch {
	case code == PlanEnterprise:
		return nil, apperrors.Conflict("plan_not_self_serve",
			"Enterprise is quoted individually — talk to us and we will size it with you.")
	case !IsPaidPlan(code):
		return nil, PlanNotFound()
	}
	cadence := NormalizeInterval(interval)
	chosen, err := PlanFor(res.plan.Band, code)
	if err != nil {
		return nil, err
	}

	now := s.now()
	sub := res.sub
	if sub == nil {
		sub = &Subscription{ID: NewSubscriptionID(), MerchantID: merchantID, CreatedAt: now}
	}
	fromPlan := res.plan.Code
	sub.Plan = chosen.Code
	sub.Status = StatusPendingPayment
	sub.Interval = cadence
	sub.Band = string(chosen.Band)
	sub.Currency = chosen.Currency
	sub.CurrentPeriodStart = &now
	sub.CurrentPeriodEnd = nil
	sub.CancelAtPeriodEnd = false
	sub.UpdatedAt = now
	if err := s.persist(ctx, sub); err != nil {
		return nil, err
	}
	s.record(ctx, merchantID, EventCheckoutRequested, fromPlan, chosen.Code, SourceAPI, actorName, actorRole,
		"Chose "+chosen.Name+" billed "+cadence+"; awaiting a collection provider",
		map[string]any{"interval": cadence, "band": sub.Band, "checkout_enabled": s.CheckoutEnabled})
	s.clearCache(ctx, merchantID)

	view, err := s.Plan(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	if !s.CheckoutEnabled {
		view.Notice = "We have recorded " + chosen.Name + " (" + cadence + "). Card payments are not open yet, so nothing has been charged — your plan stays " + PlanName(fromPlan) + " until billing goes live."
	}
	return view, nil
}

// Cancel ends a subscription. A plan that is billing keeps working to the end
// of the period it was paid for; a plan that is only a recorded intention has
// nothing to keep working, so it ends immediately.
func (s *Service) Cancel(ctx context.Context, merchantID, actorName, actorRole string) (*View, error) {
	merchant, err := s.merchant(merchantID)
	if err != nil {
		return nil, err
	}
	res, err := s.resolve(ctx, merchant)
	if err != nil {
		return nil, err
	}
	if res.sub == nil || res.sub.IsEnded() {
		return nil, SubscriptionNotFound()
	}

	now := s.now()
	fromStatus := res.sub.Status
	if res.sub.Status == StatusPendingPayment {
		res.sub.Plan = PlanFree
		res.sub.Status = StatusCancelled
		res.sub.Interval = IntervalNone
		res.sub.CancelAtPeriodEnd = false
	} else {
		res.sub.CancelAtPeriodEnd = true
	}
	res.sub.UpdatedAt = now
	if err := s.Repo.Save(res.sub); err != nil {
		return nil, err
	}
	detail := "Cancellation requested; access continues to the end of the current period"
	if res.sub.Status == StatusCancelled {
		detail = "Pending checkout cancelled; the organization is on Free"
	}
	s.record(ctx, merchantID, EventCancel, res.plan.Code, res.sub.Plan, SourceAPI, actorName, actorRole, detail,
		map[string]any{"from_status": fromStatus, "cancel_at_period_end": res.sub.CancelAtPeriodEnd})
	s.clearCache(ctx, merchantID)

	return s.Plan(ctx, merchantID)
}

// Resume reverses a cancellation that has not yet taken effect.
func (s *Service) Resume(ctx context.Context, merchantID, actorName, actorRole string) (*View, error) {
	merchant, err := s.merchant(merchantID)
	if err != nil {
		return nil, err
	}
	res, err := s.resolve(ctx, merchant)
	if err != nil {
		return nil, err
	}
	if res.sub == nil || res.sub.IsEnded() {
		return nil, SubscriptionNotFound()
	}
	if !res.sub.CancelAtPeriodEnd {
		return s.view(res), nil
	}

	now := s.now()
	res.sub.CancelAtPeriodEnd = false
	res.sub.UpdatedAt = now
	if err := s.Repo.Save(res.sub); err != nil {
		return nil, err
	}
	s.record(ctx, merchantID, EventResume, res.plan.Code, res.plan.Code, SourceAPI, actorName, actorRole,
		"Cancellation withdrawn", nil)
	s.clearCache(ctx, merchantID)

	return s.Plan(ctx, merchantID)
}

/* --------------------------------- resolution ------------------------------- */

// resolve loads and settles a merchant's subscription, then prices it.
func (s *Service) resolve(ctx context.Context, merchant *merchants.Merchant) (*resolved, error) {
	sub, err := s.Repo.SubscriptionByMerchant(merchant.ID)
	switch {
	case err == nil:
	case IsNotFound(err):
		sub = nil
	default:
		return nil, err
	}

	sub, err = s.settle(ctx, merchant, sub)
	if err != nil {
		return nil, err
	}

	if sub == nil || sub.IsEnded() {
		// The row is kept even when it has ended: it still holds the permanent
		// facts (trial_used) and it is the row a new choice must update, not a
		// second one this merchant is not allowed to have.
		return &resolved{sub: sub, plan: FreePlan(BandFor(merchant.Country))}, nil
	}

	code := NormalizePlanCode(sub.Plan)
	if code == PlanFree {
		return &resolved{sub: sub, plan: FreePlan(BandFor(merchant.Country))}, nil
	}
	// A paid subscription keeps the band it was sold in: a merchant who moves
	// country is not repriced mid-term, and a country edit is not a discount.
	band := BandFor(merchant.Country)
	if sub.Band != "" {
		band = Band(sub.Band)
	}
	plan, err := PlanFor(band, code)
	if err != nil {
		// Only reachable if a newer build wrote a plan this one does not know.
		s.warn("stored plan is not in the catalog; treating the merchant as Free",
			"plan", sub.Plan, "band", sub.Band)
		return &resolved{sub: sub, plan: FreePlan(band)}, nil
	}
	// A chosen plan is not yet a paid plan. Until a provider confirms the
	// payment, the merchant keeps the entitlements they have — which is the
	// whole point of "nothing is charged until billing opens".
	if sub.Status == StatusPendingPayment {
		return &resolved{sub: sub, plan: FreePlan(band), chosen: plan}, nil
	}
	return &resolved{sub: sub, plan: plan}, nil
}

// settle applies the time-based transitions lazily, on read, so no scheduler is
// needed to keep a subscription honest: a trial that has ended, a period that
// has lapsed, a cancellation that has come into effect.
func (s *Service) settle(ctx context.Context, merchant *merchants.Merchant, sub *Subscription) (*Subscription, error) {
	if sub == nil || sub.IsEnded() {
		return sub, nil
	}
	now := s.now()
	eventType, detail := "", ""

	switch {
	case sub.Status == StatusTrialing && sub.TrialEndsAt != nil && !now.Before(*sub.TrialEndsAt):
		if sub.CurrentPeriodEnd == nil {
			sub.CurrentPeriodEnd = sub.TrialEndsAt
		}
		// The trial hands out Pro. Expiry lands on Lite, pending payment.
		sub.Plan = PlanLite
		sub.Status = StatusPastDue
		sub.Interval = IntervalMonthly
		eventType, detail = EventTrialEnded, "Trial ended; the organization moved to Lite pending payment"

	case sub.Status == StatusActive && sub.CurrentPeriodEnd != nil && !now.Before(*sub.CurrentPeriodEnd):
		sub.Status = StatusPastDue
		eventType, detail = EventPaymentFailed, "Period ended without a confirmed renewal"

	case sub.CancelAtPeriodEnd && sub.CurrentPeriodEnd != nil && !now.Before(*sub.CurrentPeriodEnd):
		sub.Plan = PlanFree
		sub.Status = StatusCancelled
		sub.Interval = IntervalNone
		sub.CancelAtPeriodEnd = false
		eventType, detail = EventCancel, "Cancellation took effect at the end of the period"
	}

	// The last step of the dunning path, evaluated on its own rather than inside
	// the switch above: a plan that is already past due must still reach Free
	// when its grace period runs out, and no other branch would fire on that
	// read. An unpaid plan keeps its oldest application live instead of the
	// account being suspended.
	if sub.Status == StatusPastDue && sub.CurrentPeriodEnd != nil &&
		now.After(sub.CurrentPeriodEnd.AddDate(0, 0, graceDays)) {
		sub.Plan = PlanFree
		sub.Status = StatusFree
		sub.Interval = IntervalNone
		sub.CancelAtPeriodEnd = false
		detail += "; grace period elapsed, the organization is back on Free"
		if eventType == "" {
			eventType = EventDowngraded
		}
	}

	if eventType == "" {
		return sub, nil
	}
	sub.UpdatedAt = now
	if err := s.Repo.Save(sub); err != nil {
		return nil, err
	}
	s.record(ctx, merchant.ID, eventType, "", sub.Plan, SourceSystem, "system", "system", strings.TrimPrefix(detail, "; "), nil)
	s.clearCache(ctx, merchant.ID)
	return sub, nil
}

// cachedPlan resolves a merchant's priced plan through the short-lived cache,
// which is what keeps an entitlement check off the database on the hot path.
func (s *Service) cachedPlan(ctx context.Context, merchantID string) (Plan, error) {
	key := "plan:" + merchantID
	if s.Store != nil {
		if raw, ok, err := s.Store.Get(ctx, key); err == nil && ok {
			if code, band, found := splitCacheValue(raw); found {
				if plan, err := PlanFor(band, code); err == nil {
					return plan, nil
				}
			}
		}
	}

	merchant, err := s.merchant(merchantID)
	if err != nil {
		return Plan{}, err
	}
	res, err := s.resolve(ctx, merchant)
	if err != nil {
		return Plan{}, err
	}
	if s.Store != nil {
		_ = s.Store.Set(ctx, key, res.plan.Code+"|"+string(res.plan.Band), planCacheTTL)
	}
	return res.plan, nil
}

// clearCache drops the cached decision after a mutation, so an upgrade is not
// invisible for up to the cache TTL.
func (s *Service) clearCache(ctx context.Context, merchantID string) {
	if s.Store == nil {
		return
	}
	if err := s.Store.Del(ctx, "plan:"+merchantID); err != nil {
		s.warn("could not clear the cached plan", "merchant_id", merchantID, "error", err)
	}
}

// splitCacheValue parses the "<code>|<band>" cache entry.
func splitCacheValue(raw string) (string, Band, bool) {
	parts := strings.SplitN(raw, "|", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], Band(parts[1]), true
}

/* ---------------------------------- helpers --------------------------------- */

func (s *Service) view(res *resolved) *View {
	view := &View{
		Plan:            res.plan,
		Status:          StatusFree,
		Interval:        IntervalNone,
		CheckoutEnabled: s.CheckoutEnabled,
	}
	if res.sub == nil {
		return view
	}
	// The status comes from the row even when the effective plan is Free — a
	// chosen plan awaiting payment is exactly that, and hiding the state would
	// leave the merchant wondering whether their click did anything.
	view.Status = res.sub.Status
	view.Interval = res.sub.Interval
	view.CurrentPeriodStart = res.sub.CurrentPeriodStart
	view.CurrentPeriodEnd = res.sub.CurrentPeriodEnd
	view.TrialEndsAt = res.sub.TrialEndsAt
	view.CancelAtPeriodEnd = res.sub.CancelAtPeriodEnd
	view.TrialUsed = res.sub.TrialUsed
	view.Provider = res.sub.Provider
	if res.sub.Status == StatusPendingPayment {
		view.Notice = "You chose " + PlanName(res.sub.Plan) + " billed " + res.sub.Interval +
			". Nothing is charged yet — you keep " + res.plan.Name + " until billing opens."
	}
	return view
}

func (s *Service) merchant(merchantID string) (*merchants.Merchant, error) {
	if s.Merchants == nil || merchantID == "" {
		return nil, apperrors.ServiceUnavailable("Billing could not resolve the organization.", nil)
	}
	merchant, err := s.Merchants.MerchantByID(merchantID)
	if err != nil {
		if IsNotFound(err) {
			return nil, apperrors.NotFound(apperrors.CodeNotFound, "Organization not found.")
		}
		return nil, err
	}
	return merchant, nil
}

// persist writes a subscription row, creating it on the first write and
// updating the existing one afterwards.
//
// It resolves against the stored row rather than the caller's struct so a
// mutation can never mint a second subscription for one merchant (the column is
// unique, so that would be a silent zero-row UPDATE) and can never resurrect a
// spent trial: trial_used is a permanent fact about the organization.
func (s *Service) persist(ctx context.Context, sub *Subscription) error {
	existing, err := s.Repo.SubscriptionByMerchant(sub.MerchantID)
	switch {
	case err == nil:
		sub.ID = existing.ID
		sub.CreatedAt = existing.CreatedAt
		if existing.TrialUsed {
			sub.TrialUsed = true
		}
		return s.Repo.Save(sub)
	case IsNotFound(err):
		if sub.CreatedAt.IsZero() {
			sub.CreatedAt = s.now()
		}
		return s.Repo.Create(sub)
	default:
		return err
	}
}

// record writes the append-only trail entry and the audit entry. Every plan
// change is a billing fact, so it lands in both.
func (s *Service) record(ctx context.Context, merchantID, eventType, fromPlan, toPlan, source, actorName, actorRole, detail string, payload map[string]any) {
	if payload == nil {
		payload = map[string]any{}
	}
	event := &Event{
		ID:         NewEventID(),
		MerchantID: merchantID,
		Type:       eventType,
		FromPlan:   fromPlan,
		ToPlan:     toPlan,
		Source:     source,
		Payload:    gormx.JSONB[map[string]any]{Data: payload},
		ActorName:  actorName,
		ActorRole:  actorRole,
		Detail:     detail,
		CreatedAt:  s.now(),
	}
	if err := s.Repo.RecordEvent(event); err != nil {
		s.warn("could not record a subscription event", "merchant_id", merchantID, "type", eventType, "error", err)
	}
	if s.Audit == nil {
		return
	}
	severity := "info"
	if fromPlan != toPlan && toPlan != "" {
		severity = "critical"
	}
	s.Audit.Record(audit.Entry{
		MerchantID: merchantID,
		ActorName:  actorName,
		ActorRole:  actorRole,
		Action:     "Subscription " + eventType,
		Detail:     detail,
		Severity:   severity,
	})
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s *Service) warn(message string, attrs ...any) {
	if s.Logger != nil {
		s.Logger.Warn(message, attrs...)
	}
}

// Period returns the UTC calendar month containing t: start inclusive, end
// exclusive, and the YYYY-MM label the invoice and the usage meter share.
//
// Usage months are calendar months rather than billing periods on purpose: a
// merchant's renewal date moves with the cadence they chose, but everyone's
// usage reads the same way.
func Period(t time.Time) (time.Time, time.Time, string) {
	t = t.UTC()
	start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	return start, start.AddDate(0, 1, 0), start.Format("2006-01")
}
