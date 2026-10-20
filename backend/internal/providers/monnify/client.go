package monnify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/kuipid01/payconnect/internal/constants"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/providers"
)

const providerName = "monnify"

// Config configures the Monnify API endpoints and transport.
type Config struct {
	TestBaseURL string
	LiveBaseURL string
	Client      *http.Client
	MaxRetries  int
	UserAgent   string
}

type responseEnvelope struct {
	RequestSuccessful bool            `json:"requestSuccessful"`
	ResponseMessage   string          `json:"responseMessage"`
	ResponseCode      string          `json:"responseCode"`
	ResponseBody      json.RawMessage `json:"responseBody"`
}

type authResponseBody struct {
	AccessToken string `json:"accessToken"`
}

func (a *Adapter) baseURL(environment string) string {
	baseURL := a.cfg.TestBaseURL
	if environment == constants.EnvironmentLive {
		baseURL = a.cfg.LiveBaseURL
	}
	return strings.TrimRight(baseURL, "/")
}

func (a *Adapter) authenticate(ctx context.Context, creds providers.Credentials) (string, error) {
	auth := base64.StdEncoding.EncodeToString([]byte(creds.APIKey + ":" + creds.SecretKey))
	result, err := providers.DoJSON(ctx, providers.HTTPOptions{
		Method:     http.MethodPost,
		URL:        a.baseURL(creds.Environment) + "/api/v1/auth/login",
		Headers:    map[string]string{"Authorization": "Basic " + auth},
		RetrySafe:  true,
		MaxRetries: a.cfg.MaxRetries,
		Provider:   providerName,
		Client:     a.cfg.Client,
		UserAgent:  a.cfg.UserAgent,
	})
	if err != nil {
		return "", err
	}
	var envelope responseEnvelope
	if err := json.Unmarshal(result.Body, &envelope); err != nil {
		return "", apperrors.ProviderUnavailable(providerName, err)
	}
	if !envelope.RequestSuccessful {
		return "", apperrors.ProviderAuthenticationFailed(providerName, errors.New(envelope.ResponseMessage))
	}
	var body authResponseBody
	if err := json.Unmarshal(envelope.ResponseBody, &body); err != nil {
		return "", apperrors.ProviderUnavailable(providerName, err)
	}
	if body.AccessToken == "" {
		return "", apperrors.ProviderUnavailable(providerName, errors.New("Monnify authentication response did not contain an access token"))
	}
	return body.AccessToken, nil
}

func (a *Adapter) call(ctx context.Context, method, path string, body any, retrySafe bool, creds providers.Credentials) (*providers.HTTPResult, error) {
	token, err := a.authenticate(ctx, creds)
	if err != nil {
		return nil, err
	}
	return providers.DoJSON(ctx, providers.HTTPOptions{
		Method:     method,
		URL:        a.baseURL(creds.Environment) + path,
		Headers:    map[string]string{"Authorization": "Bearer " + token},
		Body:       body,
		RetrySafe:  retrySafe,
		MaxRetries: a.cfg.MaxRetries,
		Provider:   providerName,
		Client:     a.cfg.Client,
		UserAgent:  a.cfg.UserAgent,
	})
}

// Adapter speaks to the Monnify API.
type Adapter struct{ cfg Config }

// New builds the Monnify adapter.
func New(cfg Config) *Adapter {
	if cfg.TestBaseURL == "" {
		cfg.TestBaseURL = "https://sandbox.monnify.com"
	}
	if cfg.LiveBaseURL == "" {
		cfg.LiveBaseURL = "https://api.monnify.com"
	}
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Adapter{cfg: cfg}
}

func (a *Adapter) timeout() time.Duration { return a.cfg.Client.Timeout }

var _ providers.Adapter = (*Adapter)(nil)

// ProviderName is exposed for tests and logging.
func ProviderName() constants.Provider { return constants.ProviderMonnify }
