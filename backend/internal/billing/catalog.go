// Package billing owns subscriptions: the plan catalog, the entitlements a
// merchant's plan grants, and the usage measured against it.
//
// The catalog is code, not a table. Prices change rarely, every change is a
// deploy, and keeping them here means the pricing page, the billing page and
// (later) the enforcement gates all read one Go value — instead of three copies
// of a number that quietly drift apart, which is exactly how the two pricing
// pages ended up claiming different free-tier limits.
package billing

import (
	"fmt"
	"strings"
)

/* ----------------------------------- bands ---------------------------------- */

// Band is a location-based pricing geography. It is derived from the
// organization's billing country and can never be chosen by the merchant, so
// there is nothing to game — a cheaper band is not a setting.
type Band string

const (
	// BandA is Nigeria: the band the headline prices are set in.
	BandA Band = "A"
	// BandB is the African PPP peers.
	BandB Band = "B"
	// BandC is the emerging-market middle.
	BandC Band = "C"
	// BandD is the margin band — it carries the cheaper ones.
	BandD Band = "D"
)

// Plan codes.
const (
	PlanFree       = "free"
	PlanLite       = "lite"
	PlanPro        = "pro"
	PlanEnterprise = "enterprise"
)

// Billing intervals.
const (
	IntervalMonthly   = "monthly"
	IntervalQuarterly = "quarterly"
	IntervalAnnual    = "annual"
	// IntervalNone is Free: nothing is billed, so there is no cadence.
	IntervalNone = "none"
)

// Subscription statuses.
const (
	// StatusFree means no paid plan: no row, or one that has ended.
	StatusFree = "free"
	// StatusTrialing is the 14-day Pro trial.
	StatusTrialing = "trialing"
	// StatusActive is a paid plan inside its period.
	StatusActive = "active"
	// StatusPastDue is a plan whose period ended (or whose trial ended) and
	// whose renewal has not been confirmed. The plan stays usable: dunning
	// degrades account management, never money movement.
	StatusPastDue = "past_due"
	// StatusPendingPayment is a chosen plan awaiting a checkout provider.
	StatusPendingPayment = "pending_payment"
	// StatusCancelled is a subscription ended by the merchant.
	StatusCancelled = "cancelled"
)

// Volume-fee shape. The fee is a value price on routed volume, never a cut of a
// settlement: it is measured from our own payments table and invoiced monthly,
// so it can never be deducted from a merchant's money.
const (
	// VolumeFeeRateBps is 0.05% in basis points — 50 kobo per ₦1,000.
	VolumeFeeRateBps = 5
	// AllowanceMultiple is the routed volume included free: 500 × the plan's
	// own monthly price, which keeps the ladder consistent across every band
	// and currency with no extra numbers to maintain.
	AllowanceMultiple = 500
	// CapMultiple bounds the fee at 3 × the plan's monthly price, so a bill can
	// never exceed 4 × the plan.
	CapMultiple = 3
	// TrialDays is the no-card Pro trial length.
	TrialDays = 14
	// Unlimited marks a dimension with no ceiling.
	Unlimited = -1
)

/* ---------------------------------- prices ---------------------------------- */

// price is one plan's price inside one band, in that band's currency, in minor
// units. Quarterly and annual are derived from the monthly price, never typed a
// second time: quarterly is the full price three times over, and annual is
// eleven months (one free). The relationships cannot drift if they are not
// separate numbers.
type price struct {
	Monthly int64
	// From is the "starts at" price of a negotiated plan (Enterprise).
	From int64
	// ExtraApplication and ExtraSeat are the per-unit add-on prices.
	ExtraApplication int64
	ExtraSeat        int64
}

// bandDef is one pricing geography: its currency, the countries that resolve to
// it, and what every plan costs there.
type bandDef struct {
	Band      Band
	Currency  string
	Countries []string
	Prices    map[string]price
}

// addOn builds a price row from a monthly figure and its per-unit add-ons. The
// quarterly and annual prices are derived by planFor, so no band repeats them.
func addOn(monthly, extraApplication, extraSeat int64) price {
	return price{Monthly: monthly, ExtraApplication: extraApplication, ExtraSeat: extraSeat}
}

// bands is the price list. The Naira prices are set independently of the dollar
// ones (the Naira is not converted from them), and every number is whole.
var bands = []bandDef{
	{
		Band:      BandA,
		Currency:  "NGN",
		Countries: []string{"NG"},
		Prices: map[string]price{
			PlanFree: {Monthly: 0},
			// ₦3,500 a month, ₦1,250 per extra application, ₦400 per extra seat.
			PlanLite: addOn(350_000, 125_000, 40_000),
			// ₦9,500 a month, ₦900 per extra application, ₦300 per extra seat.
			PlanPro:        addOn(950_000, 90_000, 30_000),
			PlanEnterprise: {From: 14_900_000}, // from ₦149,000
		},
	},
	{
		Band:      BandB,
		Currency:  "USD",
		Countries: []string{"GH", "KE", "ZA", "EG", "RW"},
		Prices: map[string]price{
			PlanFree:       {Monthly: 0},
			PlanLite:       addOn(400, 400, 200),
			PlanPro:        addOn(1_100, 300, 100),
			PlanEnterprise: {From: 49_900},
		},
	},
	{
		Band:      BandC,
		Currency:  "USD",
		Countries: []string{"IN", "PK", "ID", "PH", "BR", "VN"},
		Prices: map[string]price{
			PlanFree:       {Monthly: 0},
			PlanLite:       addOn(600, 400, 200),
			PlanPro:        addOn(1_700, 300, 100),
			PlanEnterprise: {From: 49_900},
		},
	},
	{
		Band:      BandD,
		Currency:  "USD",
		Countries: []string{"US", "CA", "GB", "IE", "DE", "FR", "ES", "IT", "NL", "PT", "SE", "NO", "DK", "AU", "NZ", "SG", "AE", "JP", "KR"},
		Prices: map[string]price{
			PlanFree:       {Monthly: 0},
			PlanLite:       addOn(900, 400, 200),
			PlanPro:        addOn(2_900, 300, 100),
			PlanEnterprise: {From: 49_900},
		},
	},
}

/* --------------------------------- plan shape -------------------------------- */

// planShape is what a plan grants, independent of what it costs. Split from the
// price table so a price change never touches an entitlement and vice versa.
type planShape struct {
	Code        string
	Name        string
	Tagline     string
	Highlighted bool
	// Negotiated plans are quoted per merchant: their published price is a
	// "from", their volume fee is flat, and nothing about them is self-serve.
	Negotiated bool
	// Applications, Seats, PaymentsPerMonth and WebhookEndpoints use
	// Unlimited (-1) for "no ceiling".
	Applications     int
	Seats            int
	PaymentsPerMonth int
	WebhookEndpoints int
	RoutingRules     bool
	PriorityDispatch bool
	// ProviderConnections is a supported shape, not a gate: the catalog states
	// it so the comparison table does not have to know one plan from another.
	ProviderConnections string
	WebhookReplay       bool
	DedicatedRegion     bool
	LogRetentionDays    int
	AuditExport         bool
	SlaCredits          bool
	Support             string
	UptimeTarget        string
}

// planOrder is the display and comparison order. Free first, Enterprise last:
// the ladder reads left to right.
var planOrder = []planShape{
	{
		Code:    PlanFree,
		Name:    "Free",
		Tagline: "For testing the waters",
		// Free includes live (with a 100-payment wall), because a test-only
		// free tier teaches a merchant nothing and sends them to a competitor.
		Applications:        1,
		Seats:               1,
		PaymentsPerMonth:    100,
		WebhookEndpoints:    1,
		ProviderConnections: "1 per environment",
		LogRetentionDays:    3,
		Support:             "Community + docs",
		UptimeTarget:        "99.9%",
	},
	{
		Code:                PlanLite,
		Name:                "Lite",
		Tagline:             "For your first real product",
		Highlighted:         false,
		Applications:        3,
		Seats:               5,
		PaymentsPerMonth:    Unlimited,
		WebhookEndpoints:    5,
		RoutingRules:        true,
		PriorityDispatch:    true,
		ProviderConnections: "All three, both environments",
		WebhookReplay:       true,
		LogRetentionDays:    30,
		Support:             "Email, 2 business days",
		UptimeTarget:        "99.9%",
	},
	{
		Code:                PlanPro,
		Name:                "Pro",
		Tagline:             "For growing products and teams",
		Highlighted:         true,
		Applications:        10,
		Seats:               25,
		PaymentsPerMonth:    Unlimited,
		WebhookEndpoints:    Unlimited,
		RoutingRules:        true,
		PriorityDispatch:    true,
		ProviderConnections: "All three, multiple accounts each",
		WebhookReplay:       true,
		LogRetentionDays:    90,
		AuditExport:         true,
		Support:             "Priority email, 1 business day",
		UptimeTarget:        "99.9%",
	},
	{
		Code:                PlanEnterprise,
		Name:                "Enterprise",
		Tagline:             "For platform teams at scale",
		Negotiated:          true,
		Applications:        Unlimited,
		Seats:               Unlimited,
		PaymentsPerMonth:    Unlimited,
		WebhookEndpoints:    Unlimited,
		RoutingRules:        true,
		PriorityDispatch:    true,
		ProviderConnections: "All three, dedicated region",
		WebhookReplay:       true,
		DedicatedRegion:     true,
		AuditExport:         true,
		SlaCredits:          true,
		Support:             "Named solutions engineer",
		UptimeTarget:        "99.99%",
	},
}

/* ------------------------------- entitlements ------------------------------- */

// Entitlements is what one plan grants one merchant, resolved into the band's
// own currency. Every amount is in minor units of Plan.Currency.
type Entitlements struct {
	Applications     int
	ExtraApplication int64
	Seats            int
	ExtraSeat        int64
	PaymentsPerMonth int
	// RoutedVolumeAllowance is included free every month; the fee starts above
	// it. Zero for Free, so a free merchant is never invoiced a volume fee.
	RoutedVolumeAllowance int64
	// RoutedVolumeCap is the most the fee can reach.
	RoutedVolumeCap int64
	// RoutedVolumeCeiling is Free's own limit on routed volume: without it a
	// free merchant could route a hundred million naira for nothing. It equals
	// the allowance a Lite merchant gets free, which keeps the ladder honest.
	RoutedVolumeCeiling int64
	WebhookEndpoints    int
	RoutingRules        bool
	PriorityDispatch    bool
	// ProviderConnections is a supported shape rather than a gate: it is stated
	// here so the comparison table never has to know one plan from another.
	ProviderConnections string
	// WebhookReplay is replay plus retry tuning.
	WebhookReplay    bool
	DedicatedRegion  bool
	LogRetentionDays int
	AuditExport      bool
	SlaCredits       bool
	Support          string
	UptimeTarget     string
}

/* ---------------------------------- catalog --------------------------------- */

// Plan is one plan priced inside one band.
type Plan struct {
	Code    string
	Name    string
	Tagline string
	// Band and Currency say which price list this instance came from.
	Band     Band
	Currency string
	// MonthlyMinor is the headline price. QuarterlyMinor is three months at the
	// full price; AnnualMinor is eleven months (one free).
	MonthlyMinor   int64
	QuarterlyMinor int64
	AnnualMinor    int64
	// FromMinor is the published "starts at" price of a negotiated plan.
	FromMinor    int64
	Negotiated   bool
	Highlighted  bool
	Entitlements Entitlements
}

// BandCatalog is one band's complete price list.
type BandCatalog struct {
	Band      Band
	Currency  string
	Countries []string
	Plans     []Plan
}

// Catalog returns every band with its priced plans, in display order.
func Catalog() []BandCatalog {
	out := make([]BandCatalog, 0, len(bands))
	for i := range bands {
		band := &bands[i]
		plans := make([]Plan, 0, len(planOrder))
		for j := range planOrder {
			plans = append(plans, planFor(band, &planOrder[j]))
		}
		out = append(out, BandCatalog{
			Band:      band.Band,
			Currency:  band.Currency,
			Countries: append([]string(nil), band.Countries...),
			Plans:     plans,
		})
	}
	return out
}

// PlansIn returns one band's priced plans.
func PlansIn(band Band) ([]Plan, error) {
	def := bandDefFor(band)
	if def == nil {
		return nil, fmt.Errorf("billing: unknown band %q", band)
	}
	plans := make([]Plan, 0, len(planOrder))
	for i := range planOrder {
		plans = append(plans, planFor(def, &planOrder[i]))
	}
	return plans, nil
}

// PlanFor resolves one plan in one band.
func PlanFor(band Band, code string) (Plan, error) {
	def := bandDefFor(band)
	if def == nil {
		return Plan{}, fmt.Errorf("billing: unknown band %q", band)
	}
	code = NormalizePlanCode(code)
	for i := range planOrder {
		if planOrder[i].Code == code {
			return planFor(def, &planOrder[i]), nil
		}
	}
	return Plan{}, fmt.Errorf("billing: unknown plan %q", code)
}

// FreePlan is the plan every organization starts on, in the band its country
// resolves to.
func FreePlan(band Band) Plan {
	plan, err := PlanFor(band, PlanFree)
	if err != nil {
		// Only reachable if the band constants and the band table disagree.
		plan, _ = PlanFor(BandA, PlanFree)
	}
	return plan
}

// planFor prices one plan shape inside one band and derives its entitlements.
func planFor(band *bandDef, shape *planShape) Plan {
	priced := band.Prices[shape.Code]
	entitlements := Entitlements{
		Applications:        shape.Applications,
		Seats:               shape.Seats,
		PaymentsPerMonth:    shape.PaymentsPerMonth,
		ExtraApplication:    priced.ExtraApplication,
		ExtraSeat:           priced.ExtraSeat,
		WebhookEndpoints:    shape.WebhookEndpoints,
		RoutingRules:        shape.RoutingRules,
		PriorityDispatch:    shape.PriorityDispatch,
		ProviderConnections: shape.ProviderConnections,
		WebhookReplay:       shape.WebhookReplay,
		DedicatedRegion:     shape.DedicatedRegion,
		LogRetentionDays:    shape.LogRetentionDays,
		AuditExport:         shape.AuditExport,
		SlaCredits:          shape.SlaCredits,
		Support:             shape.Support,
		UptimeTarget:        shape.UptimeTarget,
	}
	if !shape.Negotiated {
		// Allowance and cap are multiples of the plan's own price, not separate
		// constants, so every band and currency gets a consistent ladder.
		entitlements.RoutedVolumeAllowance = priced.Monthly * AllowanceMultiple
		entitlements.RoutedVolumeCap = priced.Monthly * CapMultiple
	}
	if shape.Code == PlanFree {
		// Free's own multiple is zero, so its ceiling is Lite's allowance: what
		// a paying Lite merchant gets included, a free one may not exceed.
		lite := band.Prices[PlanLite]
		entitlements.RoutedVolumeCeiling = lite.Monthly * AllowanceMultiple
	}
	return Plan{
		Code:           shape.Code,
		Name:           shape.Name,
		Tagline:        shape.Tagline,
		Band:           band.Band,
		Currency:       band.Currency,
		MonthlyMinor:   priced.Monthly,
		QuarterlyMinor: priced.Monthly * 3,
		AnnualMinor:    priced.Monthly * 11,
		FromMinor:      priced.From,
		Negotiated:     shape.Negotiated,
		Highlighted:    shape.Highlighted,
		Entitlements:   entitlements,
	}
}

// PlanName returns a plan's display name, for a stored code.
func PlanName(code string) string {
	code = NormalizePlanCode(code)
	for i := range planOrder {
		if planOrder[i].Code == code {
			return planOrder[i].Name
		}
	}
	return planOrder[0].Name
}

// bandDefFor finds a band definition. A nil result means an unknown band, which
// only happens if a stored subscription was written by a newer build.
func bandDefFor(band Band) *bandDef {
	for i := range bands {
		if bands[i].Band == band {
			return &bands[i]
		}
	}
	return nil
}

/* ------------------------------ band resolution ----------------------------- */

// BandFor resolves a billing country to its price band.
//
// An explicit country we do not know resolves to the margin band, because the
// failure that matters is accidentally handing a merchant the cheapest price;
// an empty country is treated as Nigeria, which is what the merchant record
// defaults to and where the product starts. Both choices are recorded here
// rather than at the call site so they cannot drift.
func BandFor(country string) Band {
	code := strings.ToUpper(strings.TrimSpace(country))
	if code == "" {
		return BandA
	}
	for i := range bands {
		for _, candidate := range bands[i].Countries {
			if candidate == code {
				return bands[i].Band
			}
		}
	}
	return BandD
}

// NormalizePlanCode maps a stored or requested plan onto a catalog code. It
// also retires the "agency" plan that Pro replaced, so an old row keeps
// resolving to something real.
func NormalizePlanCode(code string) string {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case PlanLite:
		return PlanLite
	case PlanPro, "agency":
		return PlanPro
	case PlanEnterprise:
		return PlanEnterprise
	default:
		return PlanFree
	}
}

// NormalizeInterval maps a requested cadence onto a catalog interval.
func NormalizeInterval(interval string) string {
	switch strings.ToLower(strings.TrimSpace(interval)) {
	case IntervalQuarterly, "quarter":
		return IntervalQuarterly
	case IntervalAnnual, "yearly", "year":
		return IntervalAnnual
	default:
		return IntervalMonthly
	}
}

// IsPaidPlan reports whether a plan is billed (Free is not).
func IsPaidPlan(code string) bool {
	code = NormalizePlanCode(code)
	return code == PlanLite || code == PlanPro
}

/* ---------------------------------- the fee ---------------------------------- */

// VolumeFee is the fee owed on one month of routed volume, in the plan's own
// minor units.
//
// It is floored (never rounded up), it charges only the volume above the
// allowance, and it never exceeds the cap — which is zero on Free, so a free
// merchant is never invoiced a volume fee at all. That last property is what
// makes "your bill never exceeds 4 × your plan price" true by construction.
func VolumeFee(routedMinor int64, plan Plan) int64 {
	if plan.Negotiated {
		return 0
	}
	allowance := plan.Entitlements.RoutedVolumeAllowance
	if routedMinor <= allowance {
		return 0
	}
	fee := (routedMinor - allowance) * VolumeFeeRateBps / 10_000
	if cap := plan.Entitlements.RoutedVolumeCap; fee > cap {
		return cap
	}
	return fee
}

// UncappedVolumeFee is what the same month would have cost with no cap. It is
// shown next to the capped figure on the invoice, so a merchant can see the
// promise doing something.
func UncappedVolumeFee(routedMinor int64, plan Plan) int64 {
	if plan.Negotiated {
		return 0
	}
	allowance := plan.Entitlements.RoutedVolumeAllowance
	if routedMinor <= allowance {
		return 0
	}
	return (routedMinor - allowance) * VolumeFeeRateBps / 10_000
}
