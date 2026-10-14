package billing

import (
	"context"

	"github.com/kuipid01/payconnect/internal/constants"
)

// The gates: the one place a plan is allowed to say no.
//
// A gate measures a dimension a plan caps — applications, seats, webhook
// endpoints, routing rules — and the answer it produces is deliberately not a
// boolean the caller can act on by itself. Refusal() is the only way to turn a
// gate into an error, and it returns nil unless the gate is explicitly enforced
// in this deployment. That is the log-only contract: a gate computes what it
// *would* block, records it, and a caller that follows the pattern below cannot
// block anything at all until someone flips the switch.
//
//	if err := gates.CheckApplications(ctx, merchantID).Refusal(); err != nil {
//		return nil, err
//	}
//
// Two rules keep this from becoming a liability on the money path:
//
//   - Nothing that moves money is gated. Payments, refunds, provider webhooks
//     and delivery are absent from the gate list by design, and their absence is
//     the enforcement matrix, not an oversight.
//   - Every read fails open. A plan that cannot be resolved or a count that
//     cannot be taken allows the request and logs the failure, because a billing
//     lookup must never be the reason a merchant cannot work.
type Gate string

// Decision is one gate's conclusion, kept whole rather than reduced to a
// boolean so a log line and a test see exactly the same arithmetic.
type Decision struct {
	Gate    Gate
	Plan    Plan
	Metric  string
	Limit   int64
	Current int64
	// Blocked reports that the plan's ceiling is already reached: the caller
	// asked for one more of something and this plan does not include it.
	Blocked bool
	// Enforced reports whether this gate is allowed to refuse in this
	// deployment. It is false everywhere until BILLING_ENFORCED names the gate.
	Enforced bool
}

// Refusal is the error to hand back when the plan does not stretch this far.
//
// It is nil in log-only mode — the default — and that is the whole point: the
// code path that refuses exists on every call site and only fires where it has
// been deliberately turned on.
func (d Decision) Refusal() error {
	if !d.Blocked || !d.Enforced {
		return nil
	}
	return UpgradeRequired(Limit{
		Plan:      d.Plan.Code,
		PlanName:  d.Plan.Name,
		Metric:    d.Metric,
		Limit:     int(d.Limit),
		Current:   int(d.Current),
		UpgradeTo: upgradeTarget(d.Plan.Code),
	})
}

/* ------------------------------ gate vocabulary ------------------------------ */

// ceilings maps each gate to the entitlement it reads, and metrics to the noun a
// refusal names. They are tables rather than code inside each check so a test can
// prove every gate has both — a gate added to the list without a ceiling would
// otherwise allow everything in silence.
var (
	ceilings = map[Gate]func(Plan) int64{
		Gate(constants.GateApplications): func(plan Plan) int64 {
			return int64(plan.Entitlements.Applications)
		},
		Gate(constants.GateSeats): func(plan Plan) int64 {
			return int64(plan.Entitlements.Seats)
		},
		Gate(constants.GateWebhookEndpoints): func(plan Plan) int64 {
			return int64(plan.Entitlements.WebhookEndpoints)
		},
		// Routing rules are a capability rather than a count: a plan that includes
		// them has no ceiling here (rules are limited by their usefulness, not by
		// the plan) and a plan that does not is blocked on the first one.
		Gate(constants.GateRoutingRules): func(plan Plan) int64 {
			if plan.Entitlements.RoutingRules {
				return unlimited
			}
			return 0
		},
	}

	gateMetrics = map[Gate]string{
		Gate(constants.GateApplications):     "application",
		Gate(constants.GateSeats):            "developer seat",
		Gate(constants.GateWebhookEndpoints): "webhook endpoint",
		Gate(constants.GateRoutingRules):     "routing rules",
	}
)

// ceilingFor returns the reader for a gate's ceiling. An unknown gate is a
// programming error, and panicking at wiring time is the loudest way to say so.
func ceilingFor(gate Gate) func(Plan) int64 {
	ceiling, ok := ceilings[gate]
	if !ok {
		panic("billing: gate " + string(gate) + " has no ceiling")
	}
	return ceiling
}

/* ---------------------------------- checks ---------------------------------- */

// CheckApplications reports whether the organization may create another
// application. Free includes one; the wall is the product, not an accident.
func (s *Service) CheckApplications(ctx context.Context, merchantID string) Decision {
	return s.checkCounted(ctx, merchantID, Gate(constants.GateApplications), s.Repo.CountApplications)
}

// CheckSeats reports whether the organization may take on another developer.
// Pending invitations count as seats, so the limit cannot be outrun by inviting
// a crowd and letting none of them accept.
func (s *Service) CheckSeats(ctx context.Context, merchantID string) Decision {
	return s.checkCounted(ctx, merchantID, Gate(constants.GateSeats), s.Repo.CountSeats)
}

// CheckWebhookEndpoints reports whether another delivery endpoint fits.
func (s *Service) CheckWebhookEndpoints(ctx context.Context, merchantID string) Decision {
	return s.checkCounted(ctx, merchantID, Gate(constants.GateWebhookEndpoints), s.Repo.CountWebhookEndpoints)
}

// CheckRoutingRules reports whether routing rules are included at all.
func (s *Service) CheckRoutingRules(ctx context.Context, merchantID string) Decision {
	return s.checkCounted(ctx, merchantID, Gate(constants.GateRoutingRules), s.Repo.CountRoutingRules)
}

// checkCounted measures one gate's dimension and decides. The counter is passed
// in so each gate names its own measurement in one line, and so the arithmetic
// itself stays testable without a database.
func (s *Service) checkCounted(ctx context.Context, merchantID string, gate Gate, count counter) Decision {
	current, err := count(merchantID)
	if err != nil {
		return s.gateUnavailable(merchantID, gate, err)
	}
	return s.check(ctx, merchantID, gateRequest{
		gate:    gate,
		metric:  gateMetrics[gate],
		current: current,
		ceiling: ceilingFor(gate),
	})
}

// counter measures one dimension an organization holds right now.
type counter func(merchantID string) (int64, error)

/* -------------------------------- enforcement -------------------------------- */

// unlimited marks a dimension this plan does not cap.
const unlimited = -1

// SetEnforcedGates turns enforcement on for the named gates, and off for every
// other gate. It is called once at startup from configuration; nothing else may
// flip a gate, so "is this enforced?" has exactly one answer per deployment.
func (s *Service) SetEnforcedGates(names []string) {
	enforced := make(map[Gate]bool, len(names))
	for _, name := range names {
		enforced[Gate(name)] = true
	}
	s.enforced = enforced
}

// EnforcedGates lists the gates this deployment may refuse on, in the catalog's
// order, so startup can state its own posture instead of leaving an operator to
// infer it. An empty list means log-only.
func (s *Service) EnforcedGates() []string {
	out := make([]string, 0, len(constants.EnforceableGates()))
	for _, name := range constants.EnforceableGates() {
		if s.enforced[Gate(name)] {
			out = append(out, name)
		}
	}
	return out
}

/* --------------------------------- internals --------------------------------- */

// gateRequest is one evaluation: which gate, what it is called when refusing,
// the ceiling its plan grants, and what the organization has now.
type gateRequest struct {
	gate    Gate
	metric  string
	current int64
	ceiling func(Plan) int64
}

// check resolves the plan, decides, and records a refusal that did not happen.
func (s *Service) check(ctx context.Context, merchantID string, req gateRequest) Decision {
	plan, err := s.cachedPlan(ctx, merchantID)
	if err != nil {
		// Fail open, loudly: an unresolvable plan is treated as "no known limit"
		// rather than Free's, so a billing outage can never be the reason a
		// merchant is stopped from working.
		return s.gateUnavailable(merchantID, req.gate, err)
	}
	decision := decide(req, plan, s.enforced[req.gate])
	s.report(decision, merchantID)
	return decision
}

// decide is the whole gate arithmetic, with no I/O in it: the caller asks for one
// more of something, and the plan either includes it or does not.
func decide(req gateRequest, plan Plan, enforced bool) Decision {
	limit := req.ceiling(plan)
	decision := Decision{
		Gate:     req.gate,
		Plan:     plan,
		Metric:   req.metric,
		Limit:    limit,
		Current:  req.current,
		Enforced: enforced,
	}
	if limit == unlimited {
		return decision
	}
	decision.Blocked = req.current+1 > limit
	return decision
}

// gateUnavailable is the fail-open answer: nothing is refused, nothing is
// enforced, and the reason is in the log.
func (s *Service) gateUnavailable(merchantID string, gate Gate, err error) Decision {
	s.warn("billing gate could not be evaluated; allowing the request",
		"gate", string(gate), "merchant_id", merchantID, "error", err)
	return Decision{Gate: gate}
}

// report writes the line that makes log-only mode useful: what the gate would
// have refused, and on which plan. An allowed request is debug-level, so a gate
// that is wired but never reached is still visible while developing.
func (s *Service) report(decision Decision, merchantID string) {
	if s.Logger == nil {
		return
	}
	if !decision.Blocked {
		s.Logger.Debug("billing gate allowed",
			"gate", string(decision.Gate),
			"merchant_id", merchantID,
			"plan", decision.Plan.Code,
			"limit", decision.Limit,
			"current", decision.Current)
		return
	}
	message := "billing gate would refuse: plan limit reached"
	if decision.Enforced {
		message = "billing gate refused: plan limit reached"
	}
	s.Logger.Warn(message,
		"gate", string(decision.Gate),
		"merchant_id", merchantID,
		"plan", decision.Plan.Code,
		"limit", decision.Limit,
		"current", decision.Current,
		"enforced", decision.Enforced)
}

// upgradeTarget names the plan that would lift a limit, so the refusal can point
// somewhere instead of only saying no.
func upgradeTarget(code string) string {
	switch NormalizePlanCode(code) {
	case PlanFree:
		return PlanLite
	case PlanLite:
		return PlanPro
	default:
		return PlanEnterprise
	}
}
