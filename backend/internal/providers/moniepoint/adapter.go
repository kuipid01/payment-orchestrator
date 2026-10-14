package moniepoint

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/kuipid01/payconnect/internal/constants"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/providers"
)

var (
	errSignature = apperrors.WebhookSignatureInvalid()
	errPayload   = apperrors.WebhookEventInvalid("The Moniepoint webhook payload could not be parsed.")
)

// Adapter speaks to the Moniepoint merchant API.
type Adapter struct{ cfg Config }

// New builds the Moniepoint adapter.
func New(cfg Config) *Adapter {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.moniepoint.com"
	}
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Adapter{cfg: cfg}
}

// Name identifies the provider.
func (a *Adapter) Name() constants.Provider { return constants.ProviderMoniepoint }

// TestCredentials calls GET /v1/accounts, which requires a valid bearer token.
func (a *Adapter) TestCredentials(ctx context.Context, creds providers.Credentials) error {
	result, err := a.call(ctx, http.MethodGet, "/v1/accounts", nil, true, creds.SecretKey)
	if err != nil {
		return err
	}
	var envelope envelope
	if err := json.Unmarshal(result.Body, &envelope); err != nil {
		return apperrors.ProviderUnavailable(providerName, err)
	}
	if envelope.Success != nil && !*envelope.Success {
		return apperrors.ProviderAuthenticationFailed(providerName, errors.New(envelope.Message))
	}
	return nil
}

// CreatePayment starts a charge. Moniepoint takes minor units.
func (a *Adapter) CreatePayment(ctx context.Context, creds providers.Credentials, req providers.PaymentRequest) (*providers.PaymentResult, error) {
	payload := createRequest{
		Reference:   req.Reference,
		Amount:      req.AmountMinor,
		Currency:    req.Currency,
		Description: req.Description,
		CallbackURL: req.CallbackURL,
		Customer: customer{
			Email: req.Email,
			Name:  req.Name,
			Phone: req.Phone,
		},
		Metadata: req.Metadata,
	}
	result, err := a.call(ctx, http.MethodPost, "/v1/payments", payload, false, creds.SecretKey)
	if err != nil {
		return nil, err
	}
	var envelope envelope
	if err := json.Unmarshal(result.Body, &envelope); err != nil {
		return nil, apperrors.ProviderUnavailable(providerName, err)
	}
	if envelope.Success != nil && !*envelope.Success {
		return nil, apperrors.ProviderRequestFailed(providerName, errors.New(envelope.Message))
	}
	domain := resultFromData(envelope.Data, req.Reference, req.AmountMinor, req.Currency)
	if domain.AuthorizationURL == "" {
		// Fall back to the provider's own checkout page when it returns an id.
		if id := stringFrom(envelope.Data, "id"); id != "" {
			domain.AuthorizationURL = "https://checkout.moniepoint.com/pay/" + url.PathEscape(id)
		}
	}
	return domain, nil
}

// GetPayment fetches a payment by merchant reference.
func (a *Adapter) GetPayment(ctx context.Context, creds providers.Credentials, reference string) (*providers.PaymentResult, error) {
	return a.VerifyPayment(ctx, creds, reference)
}

// VerifyPayment reconciles a payment with Moniepoint.
func (a *Adapter) VerifyPayment(ctx context.Context, creds providers.Credentials, reference string) (*providers.PaymentResult, error) {
	result, err := a.call(ctx, http.MethodGet, "/v1/payments/"+url.PathEscape(reference), nil, true, creds.SecretKey)
	if err != nil {
		return nil, err
	}
	var envelope envelope
	if err := json.Unmarshal(result.Body, &envelope); err != nil {
		return nil, apperrors.ProviderUnavailable(providerName, err)
	}
	return resultFromData(envelope.Data, reference, 0, constants.DefaultCurrency), nil
}

// RefundPayment refunds all or part of a payment.
func (a *Adapter) RefundPayment(ctx context.Context, creds providers.Credentials, req providers.RefundRequest) (*providers.RefundResult, error) {
	payload := refundRequest{
		PaymentReference: firstNonEmpty(req.ProviderReference, req.PaymentReference),
		Amount:           req.AmountMinor,
		Reason:           req.Reason,
	}
	result, err := a.call(ctx, http.MethodPost, "/v1/refunds", payload, false, creds.SecretKey)
	if err != nil {
		return nil, err
	}
	var envelope envelope
	if err := json.Unmarshal(result.Body, &envelope); err != nil {
		return nil, apperrors.ProviderUnavailable(providerName, err)
	}
	if envelope.Success != nil && !*envelope.Success {
		return nil, apperrors.RefundFailed(envelope.Message, nil)
	}
	status := mapRefundStatus(stringFrom(envelope.Data, "status"))
	out := &providers.RefundResult{
		Status:           status,
		AmountMinor:      req.AmountMinor,
		Currency:         req.Currency,
		ProviderRefundID: stringFrom(envelope.Data, "id"),
	}
	if status == string(constants.RefundFailed) {
		out.FailureMessage = envelope.Message
	}
	return out, nil
}
