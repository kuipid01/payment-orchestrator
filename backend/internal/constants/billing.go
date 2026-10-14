package constants

// Gate names one dimension a plan can cap. These strings are the vocabulary of
// the BILLING_ENFORCED setting, so they live in constants rather than in the
// billing package: configuration validates the names before the billing service
// ever sees them, and a typo in a deployment's environment must fail at startup
// rather than silently leaving a gate in log-only mode.
const (
	// GateApplications is how many applications an organization may have.
	GateApplications = "applications"
	// GateSeats is how many developers may hold a seat.
	GateSeats = "seats"
	// GateWebhookEndpoints is how many delivery endpoints may exist.
	GateWebhookEndpoints = "webhook_endpoints"
	// GateRoutingRules is whether routing rules and failover are included at all.
	GateRoutingRules = "routing_rules"
)

// BillingGateAll is the shorthand for every enforceable gate.
const BillingGateAll = "all"

// EnforceableGates lists every gate this build knows how to enforce. Only the
// gates whose verdict is allowed to refuse an action appear here: nothing that
// moves money is ever refused, so payments and volume are absent by design.
func EnforceableGates() []string {
	return []string{GateApplications, GateSeats, GateWebhookEndpoints, GateRoutingRules}
}

// IsEnforceableGate reports whether name is a gate that may be enforced.
func IsEnforceableGate(name string) bool {
	for _, gate := range EnforceableGates() {
		if gate == name {
			return true
		}
	}
	return false
}
