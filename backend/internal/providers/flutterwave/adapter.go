package flutterwave

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
	errPayload   = apperrors.WebhookEventInvalid("The Flutterwave webhook payload could not be parsed.")
)

// Adapter speaks to the Flutterwave v3 API.
type Adapter struct{ cfg Config }

// New builds the Flutterwave adapter.
func New(cfg Config) *Adapter {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.flutterwave.com/v3"
	}
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Adapter{cfg: cfg}
}

// Name identifies the provider.
func (a *Adapter) Name() constants.Provider { return constants.ProviderFlutterwave }

// TestCredentials calls GET /balances.
func (a *Adapter) TestCredentials(ctx context.Context, creds providers.Credentials) error {
	possibleEndpoints := []string{
		"/balances",
		"/balances?currency=NGN",
		"/transactions",
	}
	result, err := a.call(ctx, http.MethodGet, possibleEndpoints, nil, true, creds.SecretKey)
	if err != nil {
		return err
	}
	var envelope envelope
	if err := json.Unmarshal(result.Body, &envelope); err != nil {
		return apperrors.ProviderUnavailable(providerName, err)
	}
	if envelope.Status != "" && envelope.Status != "success" {
		return apperrors.ProviderAuthenticationFailed(providerName, errors.New(envelope.Message))
	}
	return nil
}

// CreatePayment starts a hosted checkout. Flutterwave takes amounts in major
// units, so the adapter formats them as a decimal string.
func (a *Adapter) CreatePayment(ctx context.Context, creds providers.Credentials, req providers.PaymentRequest) (*providers.PaymentResult, error) {
	possibleEndpoints := []string{
		"/payments",
	}
	payload := createRequest{
		TxRef:       req.Reference,
		Amount:      providers.FormatMajor(req.AmountMinor, req.Currency),
		Currency:    req.Currency,
		RedirectURL: req.CallbackURL,
		Customer: customer{
			Email:       req.Email,
			Name:        req.Name,
			PhoneNumber: req.Phone,
		},
		Meta: req.Metadata,
	}
	result, err := a.call(ctx, http.MethodPost, possibleEndpoints, payload, false, creds.SecretKey)
	if err != nil {
		return nil, err
	}
	var envelope envelope
	if err := json.Unmarshal(result.Body, &envelope); err != nil {
		return nil, apperrors.ProviderUnavailable(providerName, err)
	}
	if envelope.Status != "success" {
		return nil, apperrors.ProviderRequestFailed(providerName, errors.New(envelope.Message))
	}
	link, _ := envelope.object()["link"].(string)
	return &providers.PaymentResult{
		ProviderReference: req.Reference,
		Status:            string(constants.PaymentPending),
		AmountMinor:       req.AmountMinor,
		Currency:          req.Currency,
		AuthorizationURL:  link,
		RawStatus:         envelope.Status,
		ProviderResponse: map[string]any{
			"status": envelope.Status,
			"link":   link,
		},
	}, nil
}

// GetPayment verifies by merchant reference.
func (a *Adapter) GetPayment(ctx context.Context, creds providers.Credentials, reference string) (*providers.PaymentResult, error) {
	return a.VerifyPayment(ctx, creds, reference)
}

// VerifyPayment reconciles a transaction with Flutterwave.
func (a *Adapter) VerifyPayment(ctx context.Context, creds providers.Credentials, reference string) (*providers.PaymentResult, error) {
	possibleEndpoints := []string{
		"/transactions/verify_by_reference?tx_ref=" + url.QueryEscape(reference),
	}
	result, err := a.call(ctx, http.MethodGet, possibleEndpoints, nil, true, creds.SecretKey)
	if err != nil {
		return nil, err
	}
	var envelope envelope
	if err := json.Unmarshal(result.Body, &envelope); err != nil {
		return nil, apperrors.ProviderUnavailable(providerName, err)
	}
	if envelope.Status != "success" {
		return nil, apperrors.ProviderRequestFailed(providerName, errors.New(envelope.Message))
	}
	var tx transaction
	if err := json.Unmarshal(envelope.Data, &tx); err != nil {
		return nil, apperrors.ProviderUnavailable(providerName, err)
	}
	// amount and charged_amount are loosely typed on Flutterwave's side, so keep
	// the raw values when the typed fields did not capture them.
	if data := envelope.object(); data != nil {
		if tx.Amount == nil {
			tx.Amount = data["amount"]
		}
		if tx.ChargedAmount == nil {
			tx.ChargedAmount = data["charged_amount"]
		}
	}
	return resultFromTransaction(tx), nil
}

// RefundPayment refunds all or part of a payment.
func (a *Adapter) RefundPayment(ctx context.Context, creds providers.Credentials, req providers.RefundRequest) (*providers.RefundResult, error) {
	payload := refundRequest{
		TxRef:    req.PaymentReference,
		Comments: req.Reason,
	}
	if req.AmountMinor > 0 {
		payload.Amount = providers.FormatMajor(req.AmountMinor, req.Currency)
	}
	possibleEndpoints := []string{
		"/refunds",
	}
	result, err := a.call(ctx, http.MethodPost, possibleEndpoints, payload, false, creds.SecretKey)
	if err != nil {
		return nil, err
	}
	var envelope envelope
	if err := json.Unmarshal(result.Body, &envelope); err != nil {
		return nil, apperrors.ProviderUnavailable(providerName, err)
	}
	if envelope.Status != "success" {
		return nil, apperrors.RefundFailed(envelope.Message, nil)
	}
	data := envelope.object()
	status := mapRefundStatus(stringFrom(data, "status"))
	out := &providers.RefundResult{
		Status:      status,
		AmountMinor: req.AmountMinor,
		Currency:    req.Currency,
	}
	if id := stringifyID(data["id"]); id != "" {
		out.ProviderRefundID = id
	}
	return out, nil
}

var _ = strconv.Itoa
