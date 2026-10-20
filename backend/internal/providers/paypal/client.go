package paypal

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/kuipid01/payconnect/internal/constants"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/providers"
)

const providerName = "paypal"

// Config configures the PayPal API endpoints and transport. Sandbox and live
// are separate hosts; the connection's environment picks one.
type Config struct {
	TestBaseURL string
	LiveBaseURL string
	Client      *http.Client
	MaxRetries  int
	UserAgent   string
}

// Adapter speaks to the PayPal REST API with the merchant's own app
// credentials. OAuth access tokens are cached in memory per credential set.
type Adapter struct {
	cfg    Config
	mu     sync.Mutex
	tokens map[string]cachedToken
}

type cachedToken struct {
	value     string
	expiresAt time.Time
}

// New builds the PayPal adapter.
func New(cfg Config) *Adapter {
	if cfg.TestBaseURL == "" {
		cfg.TestBaseURL = "https://api-m.sandbox.paypal.com"
	}
	if cfg.LiveBaseURL == "" {
		cfg.LiveBaseURL = "https://api-m.paypal.com"
	}
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Adapter{cfg: cfg, tokens: make(map[string]cachedToken)}
}

// Name identifies the provider.
func (a *Adapter) Name() constants.Provider { return constants.ProviderPayPal }

func (a *Adapter) baseURL(environment string) string {
	baseURL := a.cfg.TestBaseURL
	if environment == constants.EnvironmentLive {
		baseURL = a.cfg.LiveBaseURL
	}
	return strings.TrimRight(baseURL, "/")
}

// token returns a cached OAuth access token, minting a new one when the cache
// is cold or the token is close to expiry.
func (a *Adapter) token(ctx context.Context, creds providers.Credentials) (string, error) {
	if strings.TrimSpace(creds.PublicKey) == "" || strings.TrimSpace(creds.SecretKey) == "" {
		return "", apperrors.ProviderAuthenticationFailed(providerName,
			errors.New("PayPal needs both the REST app client ID and its secret"))
	}
	cacheKey := tokenCacheKey(creds)
	a.mu.Lock()
	cached, ok := a.tokens[cacheKey]
	a.mu.Unlock()
	if ok && time.Now().Before(cached.expiresAt) {
		return cached.value, nil
	}

	basic := base64.StdEncoding.EncodeToString([]byte(creds.PublicKey + ":" + creds.SecretKey))
	result, err := providers.DoJSON(ctx, providers.HTTPOptions{
		Method:     http.MethodPost,
		URL:        a.baseURL(creds.Environment) + "/v1/oauth2/token",
		Headers:    map[string]string{"Authorization": "Basic " + basic},
		Form:       url.Values{"grant_type": {"client_credentials"}},
		RetrySafe:  true,
		MaxRetries: a.cfg.MaxRetries,
		Provider:   providerName,
		Client:     a.cfg.Client,
		UserAgent:  a.cfg.UserAgent,
	})
	if err != nil {
		return "", err
	}
	var response tokenResponse
	if err := json.Unmarshal(result.Body, &response); err != nil {
		return "", apperrors.ProviderUnavailable(providerName, err)
	}
	if response.AccessToken == "" {
		return "", apperrors.ProviderAuthenticationFailed(providerName,
			errors.New("PayPal did not return an access token"))
	}
	// PayPal tokens last about nine hours. Renew a minute early so a request is
	// never sent with a token that expires in flight.
	ttl := time.Duration(response.ExpiresIn)*time.Second - time.Minute
	if ttl < time.Minute {
		ttl = time.Minute
	}
	a.mu.Lock()
	a.tokens[cacheKey] = cachedToken{value: response.AccessToken, expiresAt: time.Now().Add(ttl)}
	a.mu.Unlock()
	return response.AccessToken, nil
}

// tokenCacheKey identifies one credential set and host without storing the
// secret itself in the map key.
func tokenCacheKey(creds providers.Credentials) string {
	digest := sha256.Sum256([]byte(creds.PublicKey + "\x00" + creds.SecretKey))
	return creds.Environment + ":" + hex.EncodeToString(digest[:])
}

// call executes an authenticated PayPal API request. requestID, when set, is
// sent as PayPal-Request-Id, which makes a retried write idempotent.
func (a *Adapter) call(ctx context.Context, method, path string, body any, retrySafe bool,
	creds providers.Credentials, requestID string) (*providers.HTTPResult, error) {
	token, err := a.token(ctx, creds)
	if err != nil {
		return nil, err
	}
	headers := map[string]string{"Authorization": "Bearer " + token}
	if requestID != "" {
		headers["PayPal-Request-Id"] = requestID
	}
	return providers.DoJSON(ctx, providers.HTTPOptions{
		Method:     method,
		URL:        a.baseURL(creds.Environment) + path,
		Headers:    headers,
		Body:       body,
		RetrySafe:  retrySafe,
		MaxRetries: a.cfg.MaxRetries,
		Provider:   providerName,
		Client:     a.cfg.Client,
		UserAgent:  a.cfg.UserAgent,
	})
}

var _ providers.Adapter = (*Adapter)(nil)

// ProviderName is exposed for tests and logging.
func ProviderName() constants.Provider { return constants.ProviderPayPal }
