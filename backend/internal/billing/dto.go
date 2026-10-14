package billing

import (
	"strconv"
	"time"
)

/* ---------------------------------- catalog --------------------------------- */

// CatalogResponse is the public price list: every band, every plan, and the one
// rule that spans them all — the volume fee.
//
// It is served so that no page has to hard-code a price. The marketing page and
// the in-app subscription page read the same bytes, which is the only version
// of this that cannot recreate two pages disagreeing about the same tier.
type CatalogResponse struct {
	TrialDays int               `json:"trial_days"`
	VolumeFee VolumeFeeResponse `json:"volume_fee"`
	Bands     []BandResponse    `json:"bands"`
}

// VolumeFeeResponse states the fee in full: the rate, what it applies to, what
// is included free, the ceiling, and how it is collected.
type VolumeFeeResponse struct {
	RateBps           int    `json:"rate_bps"`
	RateLabel         string `json:"rate_label"`
	AllowanceMultiple int    `json:"allowance_multiple"`
	CapMultiple       int    `json:"cap_multiple"`
	Scope             string `json:"scope"`
	Billed            string `json:"billed"`
	Promise           string `json:"promise"`
}

// BandResponse is one pricing geography.
type BandResponse struct {
	Band      string         `json:"band"`
	Currency  string         `json:"currency"`
	Countries []string       `json:"countries"`
	Plans     []PlanResponse `json:"plans"`
}

// PlanResponse is one priced plan.
type PlanResponse struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Tagline string `json:"tagline"`
	// Negotiated plans are quoted per merchant; their price is a "from".
	Negotiated  bool   `json:"negotiated"`
	Highlighted bool   `json:"highlighted"`
	Band        string `json:"band"`
	Currency    string `json:"currency"`
	// All amounts are minor units in Currency. Quarterly is three months at the
	// full price; annual is eleven months (one free). Both are derived from the
	// monthly price, so they cannot disagree with it.
	MonthlyMinor   int64                `json:"monthly_minor"`
	QuarterlyMinor int64                `json:"quarterly_minor"`
	AnnualMinor    int64                `json:"annual_minor"`
	FromMinor      int64                `json:"from_minor,omitempty"`
	Entitlements   EntitlementsResponse `json:"entitlements"`
}

// EntitlementsResponse is what a plan grants. -1 means no ceiling.
type EntitlementsResponse struct {
	Applications               int    `json:"applications"`
	ExtraApplicationMinor      int64  `json:"extra_application_minor"`
	Seats                      int    `json:"seats"`
	ExtraSeatMinor             int64  `json:"extra_seat_minor"`
	PaymentsPerMonth           int    `json:"payments_per_month"`
	RoutedVolumeAllowanceMinor int64  `json:"routed_volume_allowance_minor"`
	RoutedVolumeCapMinor       int64  `json:"routed_volume_cap_minor"`
	RoutedVolumeCeilingMinor   int64  `json:"routed_volume_ceiling_minor,omitempty"`
	WebhookEndpoints           int    `json:"webhook_endpoints"`
	RoutingRules               bool   `json:"routing_rules"`
	PriorityDispatch           bool   `json:"priority_dispatch"`
	ProviderConnections        string `json:"provider_connections"`
	WebhookReplay              bool   `json:"webhook_replay"`
	DedicatedRegion            bool   `json:"dedicated_region"`
	LogRetentionDays           int    `json:"log_retention_days"`
	AuditExport                bool   `json:"audit_export"`
	SlaCredits                 bool   `json:"sla_credits"`
	Support                    string `json:"support"`
	UptimeTarget               string `json:"uptime_target"`
}

// ToCatalogResponse renders the whole catalog.
func ToCatalogResponse(bands []BandCatalog) CatalogResponse {
	out := CatalogResponse{
		TrialDays: TrialDays,
		VolumeFee: VolumeFeeResponse{
			RateBps:           VolumeFeeRateBps,
			RateLabel:         rateLabel(VolumeFeeRateBps),
			AllowanceMultiple: AllowanceMultiple,
			CapMultiple:       CapMultiple,
			Scope:             "Only volume PayConnect routed for you — successful live payments in our own records, net of refunds. Volume you send straight to your provider is never counted.",
			Billed:            "Measured monthly, itemised on one invoice. We never deduct it from a settlement and never take it out of a customer's payment.",
			Promise:           "Your PayConnect bill never exceeds 4× your plan price.",
		},
		Bands: make([]BandResponse, 0, len(bands)),
	}
	for i := range bands {
		band := &bands[i]
		response := BandResponse{
			Band:      string(band.Band),
			Currency:  band.Currency,
			Countries: band.Countries,
			Plans:     make([]PlanResponse, 0, len(band.Plans)),
		}
		for j := range band.Plans {
			response.Plans = append(response.Plans, ToPlanResponse(band.Plans[j]))
		}
		out.Bands = append(out.Bands, response)
	}
	return out
}

// ToPlanResponse maps a priced plan onto the public DTO.
func ToPlanResponse(plan Plan) PlanResponse {
	return PlanResponse{
		Code:           plan.Code,
		Name:           plan.Name,
		Tagline:        plan.Tagline,
		Negotiated:     plan.Negotiated,
		Highlighted:    plan.Highlighted,
		Band:           string(plan.Band),
		Currency:       plan.Currency,
		MonthlyMinor:   plan.MonthlyMinor,
		QuarterlyMinor: plan.QuarterlyMinor,
		AnnualMinor:    plan.AnnualMinor,
		FromMinor:      plan.FromMinor,
		Entitlements: EntitlementsResponse{
			Applications:               plan.Entitlements.Applications,
			ExtraApplicationMinor:      plan.Entitlements.ExtraApplication,
			Seats:                      plan.Entitlements.Seats,
			ExtraSeatMinor:             plan.Entitlements.ExtraSeat,
			PaymentsPerMonth:           plan.Entitlements.PaymentsPerMonth,
			RoutedVolumeAllowanceMinor: plan.Entitlements.RoutedVolumeAllowance,
			RoutedVolumeCapMinor:       plan.Entitlements.RoutedVolumeCap,
			RoutedVolumeCeilingMinor:   plan.Entitlements.RoutedVolumeCeiling,
			WebhookEndpoints:           plan.Entitlements.WebhookEndpoints,
			RoutingRules:               plan.Entitlements.RoutingRules,
			PriorityDispatch:           plan.Entitlements.PriorityDispatch,
			ProviderConnections:        plan.Entitlements.ProviderConnections,
			WebhookReplay:              plan.Entitlements.WebhookReplay,
			DedicatedRegion:            plan.Entitlements.DedicatedRegion,
			LogRetentionDays:           plan.Entitlements.LogRetentionDays,
			AuditExport:                plan.Entitlements.AuditExport,
			SlaCredits:                 plan.Entitlements.SlaCredits,
			Support:                    plan.Entitlements.Support,
			UptimeTarget:               plan.Entitlements.UptimeTarget,
		},
	}
}

/* ------------------------------- subscription ------------------------------- */

// SubscriptionResponse is the merchant's own billing state, including the full
// price list for their band so the subscription page never guesses a price.
type SubscriptionResponse struct {
	Plan               PlanResponse `json:"plan"`
	Band               string       `json:"band"`
	Status             string       `json:"status"`
	Interval           string       `json:"interval"`
	CurrentPeriodStart *string      `json:"current_period_start,omitempty"`
	CurrentPeriodEnd   *string      `json:"current_period_end,omitempty"`
	TrialEndsAt        *string      `json:"trial_ends_at,omitempty"`
	CancelAtPeriodEnd  bool         `json:"cancel_at_period_end"`
	TrialUsed          bool         `json:"trial_used"`
	Provider           string       `json:"provider,omitempty"`
	// CheckoutEnabled is false until a collection provider is wired: a chosen
	// plan is recorded, never charged, and the notice says so.
	CheckoutEnabled bool   `json:"checkout_enabled"`
	Notice          string `json:"notice,omitempty"`
	// Options are the other plans in the merchant's own band, so the chooser
	// prices are the same numbers the catalog serves.
	Options []PlanResponse `json:"options"`
}

// ToViewResponse renders a merchant's billing state.
func ToViewResponse(view *View) SubscriptionResponse {
	out := SubscriptionResponse{
		Plan:               ToPlanResponse(view.Plan),
		Band:               string(view.Plan.Band),
		Status:             view.Status,
		Interval:           view.Interval,
		CurrentPeriodStart: formatTime(view.CurrentPeriodStart),
		CurrentPeriodEnd:   formatTime(view.CurrentPeriodEnd),
		TrialEndsAt:        formatTime(view.TrialEndsAt),
		CancelAtPeriodEnd:  view.CancelAtPeriodEnd,
		TrialUsed:          view.TrialUsed,
		Provider:           view.Provider,
		CheckoutEnabled:    view.CheckoutEnabled,
		Notice:             view.Notice,
	}
	if options, err := PlansIn(view.Plan.Band); err == nil {
		out.Options = make([]PlanResponse, 0, len(options))
		for i := range options {
			out.Options = append(out.Options, ToPlanResponse(options[i]))
		}
	}
	return out
}

/* ----------------------------------- usage ---------------------------------- */

// UsageResponse is one calendar month measured against the plan.
type UsageResponse struct {
	Period       string         `json:"period"`
	Plan         string         `json:"plan"`
	Currency     string         `json:"currency"`
	Applications MetricResponse `json:"applications"`
	Seats        MetricResponse `json:"seats"`
	Payments     MetricResponse `json:"payments"`
	RoutedVolume VolumeResponse `json:"routed_volume"`
	VolumeFee    FeeResponse    `json:"volume_fee"`
}

// MetricResponse is one metered dimension: what is used against what the plan
// includes (-1 limit means no ceiling).
type MetricResponse struct {
	Used  int64 `json:"used"`
	Limit int   `json:"limit"`
}

// VolumeResponse is the routed-volume meter, in the plan's own currency.
type VolumeResponse struct {
	Used      int64 `json:"used"`
	Refunded  int64 `json:"refunded"`
	Allowance int64 `json:"allowance"`
	Cap       int64 `json:"cap"`
	Ceiling   int64 `json:"ceiling,omitempty"`
}

// FeeResponse is the volume fee accrued so far this month.
type FeeResponse struct {
	Amount   int64 `json:"amount"`
	Uncapped int64 `json:"uncapped"`
	RateBps  int   `json:"rate_bps"`
}

// ToUsageResponse renders a measured month against the plan it was measured on.
func ToUsageResponse(usage *Usage) UsageResponse {
	plan := usage.plan
	return UsageResponse{
		Period:   usage.Period,
		Plan:     usage.PlanCode,
		Currency: usage.Currency,
		Applications: MetricResponse{
			Used:  usage.Applications,
			Limit: plan.Entitlements.Applications,
		},
		Seats: MetricResponse{
			Used:  usage.Seats,
			Limit: plan.Entitlements.Seats,
		},
		Payments: MetricResponse{
			Used:  usage.Payments,
			Limit: plan.Entitlements.PaymentsPerMonth,
		},
		RoutedVolume: VolumeResponse{
			Used:      usage.RoutedVolume,
			Refunded:  usage.RefundedVolume,
			Allowance: usage.Allowance,
			Cap:       usage.Cap,
			Ceiling:   usage.Ceiling,
		},
		VolumeFee: FeeResponse{
			Amount:   usage.Fee,
			Uncapped: usage.UncappedFee,
			RateBps:  VolumeFeeRateBps,
		},
	}
}

// EventResponse is one entry in the subscription trail.
type EventResponse struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	FromPlan  string `json:"from_plan"`
	ToPlan    string `json:"to_plan"`
	Source    string `json:"source"`
	Detail    string `json:"detail"`
	ActorName string `json:"actor_name"`
	CreatedAt string `json:"created_at"`
}

// ToEventResponse maps one trail entry.
func ToEventResponse(event *Event) EventResponse {
	return EventResponse{
		ID:        event.ID,
		Type:      event.Type,
		FromPlan:  event.FromPlan,
		ToPlan:    event.ToPlan,
		Source:    event.Source,
		Detail:    event.Detail,
		ActorName: event.ActorName,
		CreatedAt: event.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// rateLabel renders basis points as the percentage a merchant reads, without a
// float artifact: 5 bps is "0.05%", not "0.05000000000000001%".
func rateLabel(bps int) string {
	whole := bps / 100
	fraction := bps % 100
	if fraction == 0 {
		return strconv.Itoa(whole) + "%"
	}
	label := strconv.Itoa(whole) + "." + pad2(strconv.Itoa(fraction))
	return trimZeros(label) + "%"
}

func pad2(value string) string {
	if len(value) >= 2 {
		return value
	}
	return "0" + value
}

func trimZeros(value string) string {
	for len(value) > 0 && value[len(value)-1] == '0' {
		value = value[:len(value)-1]
	}
	if len(value) > 0 && value[len(value)-1] == '.' {
		value = value[:len(value)-1]
	}
	return value
}

func formatTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}
