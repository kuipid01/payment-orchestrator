package monnify

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/kuipid01/payconnect/internal/constants"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/providers"
)

var (
	errSignature = apperrors.WebhookSignatureInvalid()
	errPayload   = apperrors.WebhookEventInvalid("The Monnify webhook payload could not be parsed.")
)

func (a *Adapter) Name() constants.Provider { return constants.ProviderMonnify }

// TestCredentials validates the API key and secret by requesting an access token.
func (a *Adapter) TestCredentials(ctx context.Context, creds providers.Credentials) error {
	_, err := a.authenticate(ctx, creds)
	return err
}

// CreatePayment initializes a hosted Monnify checkout.
func (a *Adapter) CreatePayment(ctx context.Context, creds providers.Credentials, req providers.PaymentRequest) (*providers.PaymentResult, error) {
	description := strings.TrimSpace(req.Description)
	if description == "" {
		description = "Payment " + req.Reference
	}
	payload := initializeRequest{
		Amount:             json.Number(providers.FormatMajor(req.AmountMinor, req.Currency)),
		CustomerName:       req.Name,
		CustomerEmail:      req.Email,
		PaymentReference:   req.Reference,
		PaymentDescription: description,
		CurrencyCode:       req.Currency,
		ContractCode:       creds.ContractCode,
		RedirectURL:        req.CallbackURL,
		Metadata:           req.Metadata,
	}

	result, err := a.call(ctx, http.MethodPost, "/api/v1/merchant/transactions/init-transaction", payload, false, creds)
	if err != nil {
		return nil, err
	}
	envelope, err := decodeEnvelope(result.Body)
	if err != nil {
		return nil, err
	}
	if !envelope.RequestSuccessful {
		return nil, apperrors.ProviderRequestFailed(providerName, errors.New(envelope.ResponseMessage))
	}
	var body transactionBody
	if err := json.Unmarshal(envelope.ResponseBody, &body); err != nil {
		return nil, apperrors.ProviderUnavailable(providerName, err)
	}
	status := string(constants.PaymentPending)
	if body.PaymentStatus != "" {
		status = mapPaymentStatus(body.PaymentStatus)
	}
	return &providers.PaymentResult{
		ProviderReference: firstNonEmpty(body.TransactionReference, req.Reference),
		Status:            status,
		AmountMinor:       amountFrom(body.Amount, req.Currency, req.AmountMinor),
		Currency:          firstNonEmpty(body.Currency, req.Currency),
		AuthorizationURL:  body.CheckoutURL,
		RawStatus:         body.PaymentStatus,
		ProviderResponse: map[string]any{
			"transaction_reference": body.TransactionReference,
			"payment_reference":     firstNonEmpty(body.PaymentReference, req.Reference),
			"payment_status":        body.PaymentStatus,
		},
	}, nil
}

// GetPayment fetches a transaction by its Monnify payment reference.
func (a *Adapter) GetPayment(ctx context.Context, creds providers.Credentials, reference string) (*providers.PaymentResult, error) {
	return a.VerifyPayment(ctx, creds, reference)
}

// VerifyPayment reconciles a transaction using Monnify's transaction query API.
func (a *Adapter) VerifyPayment(ctx context.Context, creds providers.Credentials, reference string) (*providers.PaymentResult, error) {
	path := "/api/v2/merchant/transactions/query?paymentReference=" + url.QueryEscape(reference)
	result, err := a.call(ctx, http.MethodGet, path, nil, true, creds)
	if err != nil {
		return nil, err
	}
	envelope, err := decodeEnvelope(result.Body)
	if err != nil {
		return nil, err
	}
	if !envelope.RequestSuccessful {
		return nil, apperrors.ProviderRequestFailed(providerName, errors.New(envelope.ResponseMessage))
	}
	var body transactionBody
	if err := json.Unmarshal(envelope.ResponseBody, &body); err != nil {
		return nil, apperrors.ProviderUnavailable(providerName, err)
	}
	status := mapPaymentStatus(body.PaymentStatus)
	return &providers.PaymentResult{
		ProviderReference: firstNonEmpty(body.TransactionReference, reference),
		Status:            status,
		AmountMinor:       amountFrom(body.AmountPaid, body.Currency, amountFrom(body.Amount, body.Currency, 0)),
		Currency:          firstNonEmpty(body.Currency, constants.DefaultCurrency),
		RawStatus:         body.PaymentStatus,
		ProviderResponse: map[string]any{
			"transaction_reference": body.TransactionReference,
			"payment_reference":     firstNonEmpty(body.PaymentReference, reference),
			"payment_status":        body.PaymentStatus,
		},
	}, nil
}

// RefundPayment submits a Monnify refund request.
func (a *Adapter) RefundPayment(ctx context.Context, creds providers.Credentials, req providers.RefundRequest) (*providers.RefundResult, error) {
	refundRef, err := newRefundReference()
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "Refund requested through PayConnect"
	}
	if len([]rune(reason)) > 64 {
		reason = string([]rune(reason)[:64])
	}
	payload := refundRequest{
		TransactionReference: firstNonEmpty(req.ProviderReference, req.PaymentReference),
		RefundReference:      refundRef,
		RefundReason:         reason,
		CustomerNote:         "PayConnect",
		CurrencyCode:         req.Currency,
		RefundAmount:         json.Number(providers.FormatMajor(req.AmountMinor, req.Currency)),
	}
	result, err := a.call(ctx, http.MethodPost, "/api/v1/refunds/initiate-refund", payload, false, creds)
	if err != nil {
		return nil, err
	}
	envelope, err := decodeEnvelope(result.Body)
	if err != nil {
		return nil, err
	}
	if !envelope.RequestSuccessful {
		return nil, apperrors.RefundFailed(envelope.ResponseMessage, nil)
	}
	var body refundBody
	if err := json.Unmarshal(envelope.ResponseBody, &body); err != nil {
		return nil, apperrors.ProviderUnavailable(providerName, err)
	}
	return &providers.RefundResult{
		ProviderRefundID: body.RefundReference,
		Status:           mapRefundStatus(body.RefundStatus),
		AmountMinor:      amountFrom(body.RefundAmount, req.Currency, req.AmountMinor),
		Currency:         req.Currency,
	}, nil
}

func decodeEnvelope(raw []byte) (responseEnvelope, error) {
	var envelope responseEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return envelope, apperrors.ProviderUnavailable(providerName, err)
	}
	return envelope, nil
}

func newRefundReference() (string, error) {
	var random [12]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("generate Monnify refund reference: %w", err)
	}
	return "pc_" + hex.EncodeToString(random[:]), nil
}
