package config

import (
	"strings"
	"testing"

	"github.com/kuipid01/payconnect/internal/constants"
)

// TestBillingEnforcedIsOffByDefault is the posture a deployment gets by saying
// nothing: every gate measures and logs, and none of them refuses.
func TestBillingEnforcedIsOffByDefault(t *testing.T) {
	t.Setenv("BILLING_ENFORCED", "")
	cfg := Load()
	if len(cfg.Billing.EnforcedGates) != 0 {
		t.Fatalf("BILLING_ENFORCED unset produced %v, want log-only", cfg.Billing.EnforcedGates)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("a log-only deployment failed validation: %v", err)
	}
}

// TestBillingEnforcedValues covers the spellings an operator will reach for,
// including the two shorthands the plan document already promised.
func TestBillingEnforcedValues(t *testing.T) {
	cases := map[string][]string{
		"":                    nil,
		"false":               nil,
		"none":                nil,
		"true":                constants.EnforceableGates(),
		"all":                 constants.EnforceableGates(),
		"applications":        {constants.GateApplications},
		"Applications, Seats": {constants.GateApplications, constants.GateSeats},
		"seats,seats":         {constants.GateSeats, constants.GateSeats},
	}
	for spec, want := range cases {
		t.Setenv("BILLING_ENFORCED", spec)
		got := Load().Billing.EnforcedGates
		if len(got) != len(want) {
			t.Errorf("BILLING_ENFORCED=%q gave %v, want %v", spec, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("BILLING_ENFORCED=%q gave %v, want %v", spec, got, want)
				break
			}
		}
	}
}

// TestBillingEnforcedRejectsUnknownGate is the failure that matters most: a
// misspelled gate must not degrade into log-only in silence, because the operator
// would believe the limit is enforced when it is not.
func TestBillingEnforcedRejectsUnknownGate(t *testing.T) {
	t.Setenv("BILLING_ENFORCED", "applicaton,seats")
	cfg := Load()
	err := cfg.Validate()
	if err == nil {
		t.Fatal("a misspelled gate passed validation")
	}
	if !strings.Contains(err.Error(), "applicaton") {
		t.Errorf("validation error does not name the offending gate: %v", err)
	}
	// The recognised gate in the same list must still be loaded, so the operator
	// fixes the typo rather than wondering which half was read.
	if len(cfg.Billing.EnforcedGates) != 2 {
		t.Errorf("enforced gates = %v, want both names kept for the error message", cfg.Billing.EnforcedGates)
	}
}

// TestBillingEnforcedShorthandsPassValidation checks the other half of the
// contract: every spelling an operator might reasonably write is accepted, so the
// typo check above cannot be satisfied by rejecting too much.
func TestBillingEnforcedShorthandsPassValidation(t *testing.T) {
	for _, spec := range []string{
		"",
		"false",
		"all",
		"applications",
		strings.Join(constants.EnforceableGates(), ","),
	} {
		t.Setenv("BILLING_ENFORCED", spec)
		if err := Load().Validate(); err != nil {
			t.Errorf("BILLING_ENFORCED=%q failed validation: %v", spec, err)
		}
	}
}
