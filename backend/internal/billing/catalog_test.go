package billing

import "testing"

// TestCatalogMatchesDecidedPrices pins the numbers the pricing decisions were
// made on. If a price or a multiple changes, this test is the thing that has to
// be changed deliberately — the catalog is the single source of truth, so a
// silent edit here would show up on the marketing page, the billing page and
// every merchant's invoice at once.
func TestCatalogMatchesDecidedPrices(t *testing.T) {
	cases := []struct {
		band                    Band
		code                    string
		monthly, allowance, cap int64
		quarterly, annual       int64
		currency                string
	}{
		{BandA, PlanFree, 0, 0, 0, 0, 0, "NGN"},
		{BandA, PlanLite, 350_000, 175_000_000, 1_050_000, 1_050_000, 3_850_000, "NGN"},
		{BandA, PlanPro, 950_000, 475_000_000, 2_850_000, 2_850_000, 10_450_000, "NGN"},
		// Band D is the published dollar ladder: $9 / $29 with a $4,500
		// allowance and a $27 cap on Lite.
		{BandD, PlanLite, 900, 450_000, 2_700, 2_700, 9_900, "USD"},
		{BandD, PlanPro, 2_900, 1_450_000, 8_700, 8_700, 31_900, "USD"},
	}
	for _, tc := range cases {
		plan, err := PlanFor(tc.band, tc.code)
		if err != nil {
			t.Fatalf("PlanFor(%s, %s): %v", tc.band, tc.code, err)
		}
		if plan.Currency != tc.currency {
			t.Errorf("%s/%s currency = %s, want %s", tc.band, tc.code, plan.Currency, tc.currency)
		}
		if plan.MonthlyMinor != tc.monthly {
			t.Errorf("%s/%s monthly = %d, want %d", tc.band, tc.code, plan.MonthlyMinor, tc.monthly)
		}
		if plan.QuarterlyMinor != tc.quarterly {
			t.Errorf("%s/%s quarterly = %d, want %d (three months at full price)",
				tc.band, tc.code, plan.QuarterlyMinor, tc.quarterly)
		}
		if plan.AnnualMinor != tc.annual {
			t.Errorf("%s/%s annual = %d, want %d (eleven months)",
				tc.band, tc.code, plan.AnnualMinor, tc.annual)
		}
		if plan.Entitlements.RoutedVolumeAllowance != tc.allowance {
			t.Errorf("%s/%s allowance = %d, want %d (500 × monthly)",
				tc.band, tc.code, plan.Entitlements.RoutedVolumeAllowance, tc.allowance)
		}
		if plan.Entitlements.RoutedVolumeCap != tc.cap {
			t.Errorf("%s/%s cap = %d, want %d (3 × monthly)",
				tc.band, tc.code, plan.Entitlements.RoutedVolumeCap, tc.cap)
		}
	}
}

// TestAllowanceAndCapAreMultiplesAcrossEveryBand is the property that makes the
// ladder consistent: the allowance and the cap are multiples of a tier's own
// price, so no band needs its own constants to maintain.
func TestAllowanceAndCapAreMultiplesAcrossEveryBand(t *testing.T) {
	for _, band := range Catalog() {
		for _, plan := range band.Plans {
			if plan.MonthlyMinor == 0 {
				continue
			}
			if !plan.Negotiated && plan.Entitlements.RoutedVolumeAllowance != plan.MonthlyMinor*AllowanceMultiple {
				t.Errorf("%s/%s allowance is not %d × its price", band.Band, plan.Code, AllowanceMultiple)
			}
			if !plan.Negotiated && plan.Entitlements.RoutedVolumeCap != plan.MonthlyMinor*CapMultiple {
				t.Errorf("%s/%s cap is not %d × its price", band.Band, plan.Code, CapMultiple)
			}
			if plan.QuarterlyMinor != plan.MonthlyMinor*3 {
				t.Errorf("%s/%s quarterly is not three months", band.Band, plan.Code)
			}
			if plan.AnnualMinor != plan.MonthlyMinor*11 {
				t.Errorf("%s/%s annual is not eleven months", band.Band, plan.Code)
			}
		}
	}
}

// TestFreeIsNeverInvoicedAndItsCeilingMatchesLite checks the two properties that
// keep Free honest: a zero cap means no fee, and a free merchant's routed volume
// ceiling is exactly what a Lite merchant gets included.
func TestFreeIsNeverInvoicedAndItsCeilingMatchesLite(t *testing.T) {
	for _, band := range Catalog() {
		var free, lite Plan
		for _, plan := range band.Plans {
			switch plan.Code {
			case PlanFree:
				free = plan
			case PlanLite:
				lite = plan
			}
		}
		if free.Entitlements.RoutedVolumeAllowance != 0 || free.Entitlements.RoutedVolumeCap != 0 {
			t.Fatalf("band %s: Free should have a zero allowance and cap", band.Band)
		}
		if got, want := free.Entitlements.RoutedVolumeCeiling, lite.Entitlements.RoutedVolumeAllowance; got != want {
			t.Errorf("band %s: Free ceiling = %d, want Lite's allowance %d", band.Band, got, want)
		}
		// A hundred million naira of volume on Free still costs nothing.
		if fee := VolumeFee(100_000_000_000, free); fee != 0 {
			t.Errorf("band %s: Free was invoiced a volume fee of %d", band.Band, fee)
		}
	}
}

// TestVolumeFeeLadder checks the published Lite ladder in Band A, which is the
// table a merchant can check against their own invoice.
func TestVolumeFeeLadder(t *testing.T) {
	lite, err := PlanFor(BandA, PlanLite)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		routedMinor  int64
		wantFeeMinor int64
		note         string
	}{
		{1_750_000_000 / 10, 0, "≤ ₦1.75m is included free"},
		{500_000_000, 162_500, "₦5m routed"},
		{1_000_000_000, 412_500, "₦10m routed"},
		{2_000_000_000, 912_500, "₦20m routed"},
		{2_275_000_000, 1_050_000, "the cap binds exactly at ₦22.75m"},
		{5_000_000_000, 1_050_000, "₦50m routed is still capped"},
	}
	for _, tc := range cases {
		if got := VolumeFee(tc.routedMinor, lite); got != tc.wantFeeMinor {
			t.Errorf("VolumeFee(%d) = %d, want %d (%s)", tc.routedMinor, got, tc.wantFeeMinor, tc.note)
		}
	}
	// The uncapped figure is shown beside the capped one, so it must stay
	// larger where the cap binds.
	if got, want := UncappedVolumeFee(4_630_000_000, lite), int64(2_227_500); got != want {
		t.Errorf("UncappedVolumeFee(₦46.3m) = %d, want %d", got, want)
	}
	// At the exact binding point the two agree; past it the capped figure stops
	// growing, which is what the merchant sees on the invoice.
	if got := VolumeFee(2_275_000_000, lite); got != UncappedVolumeFee(2_275_000_000, lite) {
		t.Errorf("at the binding point capped (%d) and uncapped (%d) should agree", got, UncappedVolumeFee(2_275_000_000, lite))
	}
	if UncappedVolumeFee(2_500_000_000, lite) <= VolumeFee(2_500_000_000, lite) {
		t.Error("the cap should be visible above the binding point: uncapped must exceed capped")
	}
}

// TestBillNeverExceedsFourTimesThePlan is the promise on the pricing page, so it
// is asserted rather than assumed.
func TestBillNeverExceedsFourTimesThePlan(t *testing.T) {
	for _, band := range Catalog() {
		for _, plan := range band.Plans {
			if plan.Negotiated || plan.MonthlyMinor == 0 {
				continue
			}
			huge := int64(10_000_000_000_000)
			total := plan.MonthlyMinor + VolumeFee(huge, plan)
			if total > plan.MonthlyMinor*4 {
				t.Errorf("%s/%s: bill %d exceeds 4 × %d", band.Band, plan.Code, total, plan.MonthlyMinor)
			}
		}
	}
}

// TestBandResolution checks that bands are derived and never guessed: an unknown
// country resolves to the margin band, not to the cheapest one.
func TestBandResolution(t *testing.T) {
	cases := map[string]Band{
		"NG": BandA,
		"ng": BandA,
		"":   BandA,
		"KE": BandB,
		"GH": BandB,
		"IN": BandC,
		"BR": BandC,
		"US": BandD,
		"gb": BandD,
		"XZ": BandD, // unknown countries never get the cheap band
	}
	for country, want := range cases {
		if got := BandFor(country); got != want {
			t.Errorf("BandFor(%q) = %s, want %s", country, got, want)
		}
	}
}

// TestCatalogShape checks the catalog is complete and sells exactly four plans
// per band, with Agency retired into Pro.
func TestCatalogShape(t *testing.T) {
	if got := len(Catalog()); got != len(bands) {
		t.Fatalf("catalog has %d bands, want %d", got, len(bands))
	}
	for _, band := range Catalog() {
		if len(band.Plans) != 4 {
			t.Fatalf("band %s sells %d plans, want 4", band.Band, len(band.Plans))
		}
		seen := map[string]bool{}
		for _, plan := range band.Plans {
			if seen[plan.Code] {
				t.Errorf("band %s lists %s twice", band.Band, plan.Code)
			}
			seen[plan.Code] = true
			if plan.Name == "" || plan.Tagline == "" {
				t.Errorf("band %s/%s is missing its name or tagline", band.Band, plan.Code)
			}
		}
		for _, code := range []string{PlanFree, PlanLite, PlanPro, PlanEnterprise} {
			if !seen[code] {
				t.Errorf("band %s is missing the %s plan", band.Band, code)
			}
		}
	}
}

// TestNormalizeRetiresAgency checks an organization stored on the retired Agency
// plan keeps resolving to something real after check-in.
func TestNormalizeRetiresAgency(t *testing.T) {
	cases := map[string]string{
		"agency":     PlanPro,
		"AGENCY":     PlanPro,
		"pro":        PlanPro,
		"lite":       PlanLite,
		"enterprise": PlanEnterprise,
		"free":       PlanFree,
		"nonsense":   PlanFree,
		"":           PlanFree,
	}
	for in, want := range cases {
		if got := NormalizePlanCode(in); got != want {
			t.Errorf("NormalizePlanCode(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestPaidPlansAreSelfServeOnly checks Free and Enterprise are not sold through
// checkout: one costs nothing and the other is quoted by hand.
func TestPaidPlansAreSelfServeOnly(t *testing.T) {
	if IsPaidPlan(PlanFree) {
		t.Error("Free is not a paid plan")
	}
	if IsPaidPlan(PlanEnterprise) {
		t.Error("Enterprise is negotiated, not self-serve")
	}
	if !IsPaidPlan(PlanLite) || !IsPaidPlan(PlanPro) {
		t.Error("Lite and Pro are the self-serve paid plans")
	}
}
