package providers

import "testing"

// Money must never round-trip through a float: these cases are exactly where a
// float-based implementation loses a minor unit.
func TestFormatMajor(t *testing.T) {
	cases := []struct {
		minor    int64
		currency string
		want     string
	}{
		{250000, "NGN", "2500.00"},
		{1, "NGN", "0.01"},
		{99, "NGN", "0.99"},
		{100, "NGN", "1.00"},
		{123456789, "NGN", "1234567.89"},
		{0, "NGN", "0.00"},
		{-250000, "NGN", "-2500.00"},
		{1000, "JPY", "1000"},
	}
	for _, tc := range cases {
		if got := FormatMajor(tc.minor, tc.currency); got != tc.want {
			t.Errorf("FormatMajor(%d, %s) = %q, want %q", tc.minor, tc.currency, got, tc.want)
		}
	}
}

func TestParseMajor(t *testing.T) {
	cases := []struct {
		value    string
		currency string
		want     int64
	}{
		{"2500.00", "NGN", 250000},
		{"0.01", "NGN", 1},
		{"1", "NGN", 100},
		{"1234567.89", "NGN", 123456789},
		{"1000", "JPY", 1000},
		{"-2500.00", "NGN", -250000},
	}
	for _, tc := range cases {
		got, err := ParseMajor(tc.value, tc.currency)
		if err != nil {
			t.Fatalf("ParseMajor(%q, %s) returned error: %v", tc.value, tc.currency, err)
		}
		if got != tc.want {
			t.Errorf("ParseMajor(%q, %s) = %d, want %d", tc.value, tc.currency, got, tc.want)
		}
	}
}

// A parsed provider amount must format back to the same value, so verification
// and reporting never drift by a minor unit.
func TestFormatParseRoundTrip(t *testing.T) {
	for _, minor := range []int64{0, 1, 99, 100, 250000, 999999999} {
		formatted := FormatMajor(minor, "NGN")
		parsed, err := ParseMajor(formatted, "NGN")
		if err != nil {
			t.Fatalf("ParseMajor(%q) returned error: %v", formatted, err)
		}
		if parsed != minor {
			t.Errorf("round trip %d -> %q -> %d", minor, formatted, parsed)
		}
	}
}

func TestParseMajorRejectsGarbage(t *testing.T) {
	if _, err := ParseMajor("", "NGN"); err == nil {
		t.Error("expected an error for an empty amount")
	}
	if _, err := ParseMajor("not-a-number", "NGN"); err == nil {
		t.Error("expected an error for a non-numeric amount")
	}
}
