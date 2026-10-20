package stripe

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kuipid01/payconnect/internal/constants"
	"github.com/kuipid01/payconnect/internal/providers"
)

const providerName = "stripe"

// Config configures the Stripe adapter. Test and live keys share one host: the
// key prefix, not the URL, decides which mode a request runs in.
type Config struct {
	BaseURL         string
	Client          *http.Client
	MaxRetries      int
	UserAgent       string
	ManagedPayments bool
	DefaultTaxCode  string
}

// Adapter speaks to the Stripe API with the merchant's own secret key.
type Adapter struct{ cfg Config }

// New builds the Stripe adapter.
func New(cfg Config) *Adapter {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.stripe.com"
	}
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Adapter{cfg: cfg}
}

// Name identifies the provider.
func (a *Adapter) Name() constants.Provider { return constants.ProviderStripe }

// call executes a Stripe API request. Stripe accepts form-encoded bodies and
// returns JSON; an Idempotency-Key makes a retried write safe.
func (a *Adapter) call(ctx context.Context, method, path string, form url.Values, idempotencyKey string,
	creds providers.Credentials, retrySafe bool) (*providers.HTTPResult, error) {
	headers := map[string]string{"Authorization": "Bearer " + creds.SecretKey}
	if idempotencyKey != "" {
		headers["Idempotency-Key"] = idempotencyKey
	}
	opts := providers.HTTPOptions{
		Method:     method,
		URL:        strings.TrimRight(a.cfg.BaseURL, "/") + path,
		Headers:    headers,
		RetrySafe:  retrySafe,
		MaxRetries: a.cfg.MaxRetries,
		Provider:   providerName,
		Client:     a.cfg.Client,
		UserAgent:  a.cfg.UserAgent,
	}
	if form != nil {
		opts.Form = form
	}
	return providers.DoJSON(ctx, opts)
}

var _ providers.Adapter = (*Adapter)(nil)

// ProviderName is exposed for tests and logging.
func ProviderName() constants.Provider { return constants.ProviderStripe }
