package constants

// Money and payment limits. All amounts are integer minor units.
const (
	// MinPaymentAmountMinor is the smallest accepted charge (₦1.00 equivalent).
	MinPaymentAmountMinor int64 = 100
	// MaxPaymentAmountMinor is a sanity ceiling for a single charge.
	MaxPaymentAmountMinor int64 = 500_000_000

	DefaultCurrency = "NGN"
)

// SupportedCurrencies lists currencies PayConnect can settle today.
func SupportedCurrencies() []string {
	return []string{"NGN", "USD", "GHS", "KES", "ZAR"}
}

// IsSupportedCurrency reports whether code is settleable.
func IsSupportedCurrency(code string) bool {
	for _, c := range SupportedCurrencies() {
		if c == code {
			return true
		}
	}
	return false
}
