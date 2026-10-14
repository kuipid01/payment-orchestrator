package paystack

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/kuipid01/payconnect/internal/constants"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/providers"
)

var (
	errSignature = apperrors.WebhookSignatureInvalid()
	errPayload   = apperrors.WebhookEventInvalid("The Paystack webhook payload could not be parsed.")
)

// Adapter speaks to the Paystack API.
type Adapter struct{ cfg Config }

// New builds the Paystack adapter.
func New(cfg Config) *Adapter {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.paystack.co"
	}
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Adapter{cfg: cfg}
}

// Name identifies the provider.
func (a *Adapter) Name() constants.Provider { return constants.ProviderPaystack }

// TestCredentials calls GET /balance, which requires a valid secret key.
func (a *Adapter) TestCredentials(ctx context.Context, creds providers.Credentials) error {
	result, err := a.call(ctx, http.MethodGet, "/balance", nil, true, creds.SecretKey)
	if err != nil {
		return err
	}
	var envelope balanceEnvelope
	if err := json.Unmarshal(result.Body, &envelope); err != nil {
		return apperrors.ProviderUnavailable(providerName, err)
	}
	if !envelope.Status {
		return apperrors.ProviderAuthenticationFailed(providerName, errors.New(envelope.Message))
	}
	return nil
}

// CreatePayment initializes a transaction and returns the authorization URL.
// Paystack takes amounts in minor units, so no conversion is needed.
func (a *Adapter) CreatePayment(ctx context.Context, creds providers.Credentials, req providers.PaymentRequest) (*providers.PaymentResult, error) {
	payload := initializeRequest{
		Email:       req.Email,
		Amount:      req.AmountMinor,
		Currency:    req.Currency,
		Reference:   req.Reference,
		CallbackURL: req.CallbackURL,
		Metadata:    req.Metadata,
	}
	// Writes are never retried: the reference is the idempotency handle.
	result, err := a.call(ctx, http.MethodPost, "/transaction/initialize", payload, false, creds.SecretKey)
	if err != nil {
		return nil, err
	}
	var envelope envelope
	if err := json.Unmarshal(result.Body, &envelope); err != nil {
		return nil, apperrors.ProviderUnavailable(providerName, err)
	}
	if !envelope.Status {
		return nil, apperrors.ProviderRequestFailed(providerName, errors.New(envelope.Message))
	}
	domain := resultFromData(envelope.Data)
	if domain.ProviderReference == "" {
		domain.ProviderReference = req.Reference
	}
	if domain.Status == "" {
		domain.Status = string(constants.PaymentPending)
	}
	return domain, nil
}

// GetPayment fetches a transaction by reference.
func (a *Adapter) GetPayment(ctx context.Context, creds providers.Credentials, reference string) (*providers.PaymentResult, error) {
	return a.VerifyPayment(ctx, creds, reference)
}

// VerifyPayment is the reconciliation path after a timeout or unknown outcome.
func (a *Adapter) VerifyPayment(ctx context.Context, creds providers.Credentials, reference string) (*providers.PaymentResult, error) {
	result, err := a.call(ctx, http.MethodGet, "/transaction/verify/"+url.PathEscape(reference), nil, true, creds.SecretKey)
	if err != nil {
		return nil, err
	}
	var envelope envelope
	if err := json.Unmarshal(result.Body, &envelope); err != nil {
		return nil, apperrors.ProviderUnavailable(providerName, err)
	}
	if !envelope.Status {
		return nil, apperrors.ProviderRequestFailed(providerName, errors.New(envelope.Message))
	}
	return resultFromData(envelope.Data), nil
}

// RefundPayment refunds all or part of a transaction.
func (a *Adapter) RefundPayment(ctx context.Context, creds providers.Credentials, req providers.RefundRequest) (*providers.RefundResult, error) {
	reference := req.ProviderReference
	if reference == "" {
		reference = req.PaymentReference
	}
	payload := refundRequest{Transaction: reference, Amount: req.AmountMinor, MerchantNote: req.Reason}
	result, err := a.call(ctx, http.MethodPost, "/refund", payload, false, creds.SecretKey)
	if err != nil {
		return nil, err
	}
	var envelope refundEnvelope
	if err := json.Unmarshal(result.Body, &envelope); err != nil {
		return nil, apperrors.ProviderUnavailable(providerName, err)
	}
	if !envelope.Status {
		return nil, apperrors.RefundFailed(envelope.Message, nil)
	}
	status := mapRefundStatus(envelope.Data.Status)
	out := &providers.RefundResult{
		Status:      status,
		AmountMinor: envelope.Data.Amount,
		Currency:    envelope.Data.Currency,
	}
	if envelope.Data.ID != 0 {
		out.ProviderRefundID = strconv.FormatInt(envelope.Data.ID, 10)
	}
	if status == string(constants.RefundFailed) {
		out.FailureMessage = envelope.Data.Message
	}
	return out, nil
}
