package paystack

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/kuipid01/payconnect/internal/constants"
	"github.com/kuipid01/payconnect/internal/providers"
)

const providerName = "paystack"

// Config configures the Paystack adapter.
type Config struct {
	BaseURL    string
	Client     *http.Client
	MaxRetries int
	UserAgent  string
}

// call executes a Paystack API request with the merchant's secret key.
func (a *Adapter) call(ctx context.Context, method, path string, body any, retrySafe bool, secretKey string) (*providers.HTTPResult, error) {
	baseURL := strings.TrimRight(a.cfg.BaseURL, "/")
	return providers.DoJSON(ctx, providers.HTTPOptions{
		Method:     method,
		URL:        baseURL + path,
		Headers:    map[string]string{"Authorization": "Bearer " + secretKey},
		Body:       body,
		RetrySafe:  retrySafe,
		MaxRetries: a.cfg.MaxRetries,
		Provider:   providerName,
		Client:     a.cfg.Client,
		UserAgent:  a.cfg.UserAgent,
	})
}

func (a *Adapter) timeout() time.Duration {
	if a.cfg.Client != nil {
		return a.cfg.Client.Timeout
	}
	return 10 * time.Second
}

var _ providers.Adapter = (*Adapter)(nil)

// ProviderName is exposed for tests and logging.
func ProviderName() constants.Provider { return constants.ProviderPaystack }
