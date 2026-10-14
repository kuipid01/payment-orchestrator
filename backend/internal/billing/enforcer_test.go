package billing

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kuipid01/payconnect/internal/constants"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
)

// appError unwraps the typed error a refusal returns, failing the test when the
// gate produced something the response layer could not render.
func appError(t *testing.T, err error) *apperrors.AppError {
	t.Helper()
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error %v is not an *AppError", err)
	}
	return appErr
}

// allGates is the vocabulary every test below iterates, read from the same list
// configuration validates against.
func allGates() []Gate {
	names := constants.EnforceableGates()
	out := make([]Gate, 0, len(names))
	for _, name := range names {
		out = append(out, Gate(name))
	}
	return out
}

// decideFor runs the production arithmetic for one gate on one plan. Only the
// measurement is substituted; the ceiling comes from the same table the service
// uses, so a test cannot pass against a limit the product does not have.
func decideFor(gate Gate, plan Plan, current int64, enforced bool) Decision {
	return decide(gateRequest{
		gate:    gate,
		metric:  gateMetrics[gate],
		current: current,
		ceiling: ceilingFor(gate),
	}, plan, enforced)
}

// TestEveryGateHasACeilingAndAName is the guard against adding a gate to the
// configuration vocabulary and forgetting that it needs a measurement and a way
// to be described. A gate with no ceiling would allow everything in silence.
func TestEveryGateHasACeilingAndAName(t *testing.T) {
	for _, gate := range allGates() {
		if _, ok := ceilings[gate]; !ok {
			t.Errorf("gate %q has no ceiling", gate)
		}
		if gateMetrics[gate] == "" {
			t.Errorf("gate %q has no metric name", gate)
		}
	}
	if len(ceilings) != len(allGates()) || len(gateMetrics) != len(allGates()) {
		t.Errorf("the gate tables are %d/%d entries for %d gates", len(ceilings), len(gateMetrics), len(allGates()))
	}
}

// TestUnknownGatePanics covers the failure a table-driven gate list is meant to
// make loud: a ceiling is asked for by name, and a name nobody defined is a
// programming error rather than a limit of "everything fits".
func TestUnknownGatePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("ceilingFor accepted a gate that does not exist")
		}
	}()
	ceilingFor(Gate("invented"))
}

// TestGateBlocksOnePastTheCeiling checks the arithmetic against the real
// catalog: Free includes one application, so the second one is the wall.
func TestGateBlocksOnePastTheCeiling(t *testing.T) {
	free := FreePlan(BandA)
	for _, tc := range []struct {
		current int64
		blocked bool
	}{
		{0, false}, // the first application is what Free includes
		{1, true},  // the second is what it does not
		{2, true},
	} {
		decision := decideFor(Gate(constants.GateApplications), free, tc.current, false)
		if decision.Blocked != tc.blocked {
			t.Errorf("Free with %d applications: blocked = %v, want %v", tc.current, decision.Blocked, tc.blocked)
		}
		if got := decision.Limit; got != int64(free.Entitlements.Applications) {
			t.Errorf("Free applications limit = %d, want %d", got, free.Entitlements.Applications)
		}
	}
}

// TestLogOnlyNeverRefuses is the contract the whole rollout rests on: a blocked
// gate in log-only mode reports the blockage and still returns no error, so a
// call site that would refuse cannot do so until enforcement is turned on.
func TestLogOnlyNeverRefuses(t *testing.T) {
	pro, err := PlanFor(BandA, PlanPro)
	if err != nil {
		t.Fatal(err)
	}
	decision := decideFor(Gate(constants.GateSeats), pro, int64(pro.Entitlements.Seats), false)
	if !decision.Blocked {
		t.Fatal("the 26th seat on Pro should be blocked")
	}
	if decision.Enforced {
		t.Error("nothing is enforced until BILLING_ENFORCED names the gate")
	}
	if err := decision.Refusal(); err != nil {
		t.Fatalf("log-only mode returned an error: %v", err)
	}
}

// TestEnforcedGateRefusesWithTheArithmetic checks that an enforced gate returns
// the structured wall the dashboard renders a prompt from: the plan, the metric,
// the limit, what is already used, and where an upgrade leads.
func TestEnforcedGateRefusesWithTheArithmetic(t *testing.T) {
	free := FreePlan(BandA)
	decision := decideFor(Gate(constants.GateApplications), free, 1, true)
	if !decision.Blocked || !decision.Enforced {
		t.Fatalf("decision = %+v, want blocked and enforced", decision)
	}

	err := decision.Refusal()
	if err == nil {
		t.Fatal("an enforced gate did not refuse")
	}
	appErr := appError(t, err)
	if appErr.Code != "upgrade_required" {
		t.Errorf("code = %q, want upgrade_required", appErr.Code)
	}
	if appErr.HTTPStatus != 402 {
		t.Errorf("status = %d, want 402", appErr.HTTPStatus)
	}
	if !strings.Contains(appErr.Message, "1 application") || !strings.Contains(appErr.Message, "already using 1") {
		t.Errorf("message %q does not state the arithmetic", appErr.Message)
	}
	limit, ok := appErr.Details.(Limit)
	if !ok {
		t.Fatalf("details are %T, want billing.Limit", appErr.Details)
	}
	if limit.Plan != PlanFree || limit.Limit != 1 || limit.Current != 1 || limit.Metric != "application" {
		t.Errorf("details = %+v, want Free/1/1/application", limit)
	}
	if limit.UpgradeTo != PlanLite {
		t.Errorf("upgrade_to = %q, want %q", limit.UpgradeTo, PlanLite)
	}
}

// TestCapabilityGateReadsAsProse checks the odd one out: routing rules are not a
// count, so refusing them must not say "includes 0 routing rules".
func TestCapabilityGateReadsAsProse(t *testing.T) {
	free := FreePlan(BandA)
	if free.Entitlements.RoutingRules {
		t.Fatal("this test assumes Free does not include routing rules")
	}
	decision := decideFor(Gate(constants.GateRoutingRules), free, 0, true)
	if !decision.Blocked {
		t.Fatal("the first routing rule on Free should be blocked")
	}
	if err := decision.Refusal(); err == nil {
		t.Fatal("an enforced capability gate did not refuse")
	} else if message := appError(t, err).Message; !strings.Contains(message, "does not include routing rules") {
		t.Errorf("message = %q, want it to name what Free does not include", message)
	}

	pro, err := PlanFor(BandA, PlanPro)
	if err != nil {
		t.Fatal(err)
	}
	if got := decideFor(Gate(constants.GateRoutingRules), pro, 500, true); got.Blocked {
		t.Error("Pro includes routing rules, so no number of rules is a plan limit")
	}
}

// TestUnlimitedDimensionNeverBlocks checks that -1 means what it says on every
// gate, so a negotiated plan cannot be refused by a limit nobody set.
func TestUnlimitedDimensionNeverBlocks(t *testing.T) {
	for _, gate := range allGates() {
		plan := FreePlan(BandA)
		switch gate {
		case Gate(constants.GateApplications):
			plan.Entitlements.Applications = unlimited
		case Gate(constants.GateSeats):
			plan.Entitlements.Seats = unlimited
		case Gate(constants.GateWebhookEndpoints):
			plan.Entitlements.WebhookEndpoints = unlimited
		case Gate(constants.GateRoutingRules):
			plan.Entitlements.RoutingRules = true
		}
		decision := decideFor(gate, plan, 1_000_000, true)
		if decision.Blocked {
			t.Errorf("gate %q blocked an uncapped dimension", gate)
		}
		if err := decision.Refusal(); err != nil {
			t.Errorf("gate %q refused an uncapped dimension: %v", gate, err)
		}
	}
}

// TestEnforcementIsPerGate checks the switch itself: naming one gate must leave
// every other gate in log-only mode.
func TestEnforcementIsPerGate(t *testing.T) {
	service := &Service{}
	service.SetEnforcedGates([]string{constants.GateSeats})
	if !service.enforced[Gate(constants.GateSeats)] {
		t.Error("seats should be enforced")
	}
	if service.enforced[Gate(constants.GateApplications)] {
		t.Error("applications should still be log-only")
	}
	// A decision for an unenforced gate carries no error even when blocked.
	decision := decideFor(Gate(constants.GateApplications), FreePlan(BandA), 5, service.enforced[Gate(constants.GateApplications)])
	if err := decision.Refusal(); err != nil {
		t.Errorf("an unenforced gate refused: %v", err)
	}
}

// TestEnforcedGatesReportsInCatalogOrder keeps the startup log stable and
// readable regardless of the order the environment listed the gates in.
func TestEnforcedGatesReportsInCatalogOrder(t *testing.T) {
	service := &Service{}
	service.SetEnforcedGates([]string{constants.GateRoutingRules, constants.GateApplications})
	got := service.EnforcedGates()
	want := []string{constants.GateApplications, constants.GateRoutingRules}
	if len(got) != len(want) {
		t.Fatalf("enforced = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("enforced = %v, want %v", got, want)
		}
	}
	// Setting the gates replaces the previous posture rather than adding to it.
	service.SetEnforcedGates(nil)
	if got := service.EnforcedGates(); len(got) != 0 {
		t.Errorf("enforced = %v after clearing, want none", got)
	}
}

// TestGatesFailOpenWhenTheCountFails is the money-path guarantee in its first
// form: a measurement that cannot be taken allows the request. This is what keeps
// a database problem from looking like a plan limit.
func TestGatesFailOpenWhenTheCountFails(t *testing.T) {
	service := enforcedEverywhere()
	failing := func(string) (int64, error) { return 0, errors.New("the database is unreachable") }
	for _, gate := range allGates() {
		decision := service.checkCounted(context.Background(), "org_1", gate, failing)
		assertAllows(t, gate, decision)
	}
}

// TestGatesFailOpenWhenThePlanCannotBeResolved is the same guarantee in its
// second form: a merchant whose plan is unknown is treated as having no limit,
// never as being over one.
func TestGatesFailOpenWhenThePlanCannotBeResolved(t *testing.T) {
	service := enforcedEverywhere() // no merchant lookup is wired, so resolution fails
	counted := func(string) (int64, error) { return 99, nil }
	for _, gate := range allGates() {
		decision := service.checkCounted(context.Background(), "org_1", gate, counted)
		assertAllows(t, gate, decision)
	}
}

// enforcedEverywhere builds a service with every gate switched on, which is the
// configuration where a failure to fail open would be most visible.
func enforcedEverywhere() *Service {
	service := &Service{}
	service.SetEnforcedGates(constants.EnforceableGates())
	return service
}

// assertAllows checks that a gate refused nothing and flagged nothing as
// enforced — the two ways a fail-open decision could still cause a refusal.
func assertAllows(t *testing.T, gate Gate, decision Decision) {
	t.Helper()
	if decision.Blocked || decision.Enforced {
		t.Errorf("gate %q: decision = %+v, want an allowing decision", gate, decision)
	}
	if err := decision.Refusal(); err != nil {
		t.Errorf("gate %q refused a request it could not evaluate: %v", gate, err)
	}
}

// TestUpgradeTargetNamesTheNextPlan keeps the CTA honest: the refusal has to
// point at a plan that actually lifts the limit.
func TestUpgradeTargetNamesTheNextPlan(t *testing.T) {
	cases := map[string]string{
		PlanFree:       PlanLite,
		PlanLite:       PlanPro,
		PlanPro:        PlanEnterprise,
		PlanEnterprise: PlanEnterprise,
	}
	for code, want := range cases {
		if got := upgradeTarget(code); got != want {
			t.Errorf("upgradeTarget(%q) = %q, want %q", code, got, want)
		}
	}
}
