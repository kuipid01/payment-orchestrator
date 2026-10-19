package config

import "testing"

func TestMonnifyBaseURLDefaults(t *testing.T) {
	t.Setenv("MONNIFY_TEST_BASE_URL", "")
	t.Setenv("MONNIFY_LIVE_BASE_URL", "")

	cfg := Load()
	if cfg.Providers.MonnifyTestBaseURL != "https://sandbox.monnify.com" {
		t.Errorf("test base URL = %q", cfg.Providers.MonnifyTestBaseURL)
	}
	if cfg.Providers.MonnifyLiveBaseURL != "https://api.monnify.com" {
		t.Errorf("live base URL = %q", cfg.Providers.MonnifyLiveBaseURL)
	}
}
