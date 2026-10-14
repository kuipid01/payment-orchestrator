package routing

import (
	"testing"

	"github.com/kuipid01/payconnect/internal/platform/gormx"
)

func rule(enabled bool, currency string, min, max int64, providers ...Preference) Rule {
	return Rule{
		Enabled:   enabled,
		Currency:  currency,
		MinAmount: min,
		MaxAmount: max,
		Providers: gormx.JSONB[[]Preference]{Data: providers},
	}
}

func TestRuleMatches(t *testing.T) {
	ngnOnly := rule(true, "NGN", 0, 0)
	if !ngnOnly.Matches("NGN", 1000) {
		t.Error("a currency-only rule should match any NGN amount")
	}
	if ngnOnly.Matches("USD", 1000) {
		t.Error("a currency rule must not match another currency")
	}

	banded := rule(true, "NGN", 1000, 5000)
	if banded.Matches("NGN", 999) {
		t.Error("amount below min_amount must not match")
	}
	if !banded.Matches("NGN", 1000) || !banded.Matches("NGN", 5000) {
		t.Error("amount on a band boundary must match")
	}
	if banded.Matches("NGN", 5001) {
		t.Error("amount above max_amount must not match")
	}

	disabled := rule(false, "NGN", 0, 0)
	if disabled.Matches("NGN", 100) {
		t.Error("a disabled rule must never match")
	}
}

func TestOrderedProviders(t *testing.T) {
	r := rule(true, "NGN", 0, 0,
		Preference{Provider: "flutterwave", Priority: 2},
		Preference{Provider: "paystack", Priority: 1},
		Preference{Provider: "", Priority: 3},
	)
	got := r.OrderedProviders()
	want := []string{"paystack", "flutterwave"}
	if len(got) != len(want) {
		t.Fatalf("OrderedProviders() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("OrderedProviders() = %v, want %v", got, want)
		}
	}
}
