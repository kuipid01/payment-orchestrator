// Package factory wires concrete provider adapters into a registry. It lives in
// its own package because it is the only place that knows about every adapter:
// the providers package itself depends on the adapter contract, never on the
// implementations, which keeps the dependency direction one-way.
package factory

import (
	"net/http"
	"time"

	"github.com/kuipid01/payconnect/internal/providers"
	"github.com/kuipid01/payconnect/internal/providers/flutterwave"
	"github.com/kuipid01/payconnect/internal/providers/monnify"
	"github.com/kuipid01/payconnect/internal/providers/paypal"
	"github.com/kuipid01/payconnect/internal/providers/paystack"
	"github.com/kuipid01/payconnect/internal/providers/stripe"
)

// Config carries the per-provider transport settings.
type Config struct {
	PaystackBaseURL    string
	FlutterwaveBaseURL string
	MonnifyTestBaseURL string
	MonnifyLiveBaseURL string
	StripeBaseURL      string
	PayPalTestBaseURL  string
	PayPalLiveBaseURL  string
	Timeout            time.Duration
	MaxRetries         int
	UserAgent          string
}

// NewRegistry builds the adapter registry for a deployment. Adding a provider
// means adding one adapter and one line here.
func NewRegistry(cfg Config) providers.Registry {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	client := &http.Client{Timeout: timeout}

	return providers.NewRegistry(
		paystack.New(paystack.Config{
			BaseURL:    cfg.PaystackBaseURL,
			Client:     client,
			MaxRetries: cfg.MaxRetries,
			UserAgent:  cfg.UserAgent,
		}),
		flutterwave.New(flutterwave.Config{
			BaseURL:    cfg.FlutterwaveBaseURL,
			Client:     client,
			MaxRetries: cfg.MaxRetries,
			UserAgent:  cfg.UserAgent,
		}),
		monnify.New(monnify.Config{
			TestBaseURL: cfg.MonnifyTestBaseURL,
			LiveBaseURL: cfg.MonnifyLiveBaseURL,
			Client:      client,
			MaxRetries:  cfg.MaxRetries,
			UserAgent:   cfg.UserAgent,
		}),
		stripe.New(stripe.Config{
			BaseURL:    cfg.StripeBaseURL,
			Client:     client,
			MaxRetries: cfg.MaxRetries,
			UserAgent:  cfg.UserAgent,
		}),
		paypal.New(paypal.Config{
			TestBaseURL: cfg.PayPalTestBaseURL,
			LiveBaseURL: cfg.PayPalLiveBaseURL,
			Client:      client,
			MaxRetries:  cfg.MaxRetries,
			UserAgent:   cfg.UserAgent,
		}),
	)
}
