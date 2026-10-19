package providers

import (
	"errors"
	"strconv"
	"strings"

	"github.com/kuipid01/payconnect/internal/constants"
)

// CurrencyDecimals reports how many minor units make one major unit.
func CurrencyDecimals(currency string) int {
	switch strings.ToUpper(currency) {
	case "JPY", "KRW", "VND":
		return 0
	default:
		return 2
	}
}

// FormatMajor renders integer minor units as a decimal string for providers
// that expect major units. It uses integer arithmetic only — no floats touch
// money anywhere in this codebase.
func FormatMajor(amountMinor int64, currency string) string {
	decimals := CurrencyDecimals(currency)
	if decimals == 0 {
		return strconv.FormatInt(amountMinor, 10)
	}
	negative := amountMinor < 0
	value := amountMinor
	if negative {
		value = -value
	}
	scale := int64(1)
	for i := 0; i < decimals; i++ {
		scale *= 10
	}
	whole := value / scale
	frac := value % scale
	out := strconv.FormatInt(whole, 10) + "." + padLeft(strconv.FormatInt(frac, 10), decimals)
	if negative {
		return "-" + out
	}
	return out
}

// ParseMajor converts a provider's decimal amount string into integer minor
// units without floating point.
func ParseMajor(value, currency string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errors.New("providers: empty amount")
	}
	negative := strings.HasPrefix(value, "-")
	value = strings.TrimPrefix(value, "-")

	parts := strings.SplitN(value, ".", 2)
	decimals := CurrencyDecimals(currency)
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	frac := int64(0)
	if len(parts) == 2 && decimals > 0 {
		digits := parts[1]
		if len(digits) > decimals {
			digits = digits[:decimals]
		}
		digits = padRight(digits, decimals)
		frac, err = strconv.ParseInt(digits, 10, 64)
		if err != nil {
			return 0, err
		}
	}
	scale := int64(1)
	for i := 0; i < decimals; i++ {
		scale *= 10
	}
	total := whole*scale + frac
	if negative {
		total = -total
	}
	return total, nil
}

// DefaultCurrencyFor returns the settlement currency for a provider.
func DefaultCurrencyFor(provider constants.Provider) string {
	switch provider {
	case constants.ProviderFlutterwave, constants.ProviderMonnify:
		return "NGN"
	default:
		return "NGN"
	}
}

func padLeft(value string, width int) string {
	for len(value) < width {
		value = "0" + value
	}
	return value
}

func padRight(value string, width int) string {
	for len(value) < width {
		value += "0"
	}
	return value
}
