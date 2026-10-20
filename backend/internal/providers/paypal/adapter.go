package paypal

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/kuipid01/payconnect/internal/constants"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/providers"
)

// TestCredentials mints an OAuth token: PayPal only issues one for a valid
// client id and secret pair.
func (a *Adapter) TestCredentials(ctx context.Context, creds providers.Credentials) error {
	_, err := a.token(ctx, creds)
	return err
}

// CreatePayment creates a CAPTURE-intent order on the merchant's own PayPal
// account and returns the hosted approval URL.
func (a *Adapter) CreatePayment(ctx context.Context, creds providers.Credentials,
	req providers.PaymentRequest) (*providers.PaymentResult, error) {
	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" {
		currency = "USD"
	}
	if !supportsCurrency(currency) {
		return nil, apperrors.Validation("paypal_currency_unsupported",
			"PayPal cannot charge in "+currency+". Connect Stripe or a local provider for this currency.")
	}
	callback := strings.TrimSpace(req.CallbackURL)
	if callback == "" {
		return nil, apperrors.Validation("checkout_callback_required",
			"A callback_url is required for PayPal Checkout so the customer can return to the store.")
	}

	description := strings.TrimSpace(req.Description)
	if description == "" {
		description = "Payment " + req.Reference
	}
	payload := map[string]any{
		"intent": "CAPTURE",
		"purchase_units": []map[string]any{{
			"reference_id": req.Reference,
			"custom_id":    req.Reference,
			"invoice_id":   req.Reference,
			"description":  description,
			"amount": map[string]string{
				"currency_code": currency,
				"value":         providers.FormatMajor(req.AmountMinor, currency),
			},
		}},
		"application_context": map[string]string{
			"return_url":          appendQuery(callback, "reference="+url.QueryEscape(req.Reference)),
			"cancel_url":          appendQuery(callback, "payment=cancelled"),
			"user_action":         "PAY_NOW",
			"shipping_preference": "NO_SHIPPING",
		},
	}

	result, err := a.call(ctx, http.MethodPost, "/v2/checkout/orders", payload, false, creds,
		"order:"+req.Reference)
	if err != nil {
		return nil, err
	}
	var created order
	if err := json.Unmarshal(result.Body, &created); err != nil {
		return nil, apperrors.ProviderUnavailable(providerName, err)
	}
	approval := approvalURL(created)
	if created.ID == "" || approval == "" {
		return nil, apperrors.ProviderRequestFailed(providerName,
			errors.New("PayPal did not return an approval link for the order"))
	}
	out := resultFromOrder(created)
	out.AuthorizationURL = approval
	if out.AmountMinor == 0 {
		out.AmountMinor = req.AmountMinor
	}
	if out.Currency == "" {
		out.Currency = currency
	}
	out.Status = string(constants.PaymentPending)
	return out, nil
}

// GetPayment fetches an order and captures it if the customer approved it.
func (a *Adapter) GetPayment(ctx context.Context, creds providers.Credentials, reference string) (*providers.PaymentResult, error) {
	return a.VerifyPayment(ctx, creds, reference)
}

// VerifyPayment is the authoritative reconciliation path: it reads the order
// and, because PayPal requires an explicit capture, captures an APPROVED order
// before reporting success.
func (a *Adapter) VerifyPayment(ctx context.Context, creds providers.Credentials, reference string) (*providers.PaymentResult, error) {
	id := strings.TrimSpace(reference)
	if id == "" {
		return nil, apperrors.Validation("paypal_reference_required",
			"A PayPal payment is verified with its order id.")
	}
	fetched, err := a.fetchOrder(ctx, creds, id)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(fetched.Status, "APPROVED") {
		captured, captureErr := a.captureOrder(ctx, creds, id)
		if captureErr != nil {
			// A concurrent confirmation may already have captured the order.
			// The latest order state decides; otherwise report the capture
			// failure so nothing is marked paid on an unknown outcome.
			latest, latestErr := a.fetchOrder(ctx, creds, id)
			if latestErr == nil && strings.EqualFold(latest.Status, "COMPLETED") {
				fetched = latest
			} else {
				return nil, captureErr
			}
		} else {
			fetched = captured
		}
	}
	return resultFromOrder(*fetched), nil
}

// RefundPayment refunds all or part of a captured payment. The stored provider
// reference is the order id, so the capture id is resolved from it first.
func (a *Adapter) RefundPayment(ctx context.Context, creds providers.Credentials,
	req providers.RefundRequest) (*providers.RefundResult, error) {
	reference := strings.TrimSpace(req.ProviderReference)
	if reference == "" {
		return nil, apperrors.Validation("paypal_refund_reference_invalid",
			"A PayPal refund needs the order id of the payment.")
	}
	if req.Currency != "" && !supportsCurrency(req.Currency) {
		return nil, apperrors.Validation("paypal_currency_unsupported",
			"PayPal cannot refund in this currency.")
	}
	captured := ""
	order, err := a.fetchOrder(ctx, creds, reference)
	switch {
	case err == nil:
		captured = captureID(*order)
		if captured == "" {
			return nil, apperrors.Validation("paypal_refund_not_captured",
				"PayPal has not captured this order yet, so it cannot be refunded.")
		}
	case apperrors.Is(err, apperrors.CodeNotFound):
		// The caller already holds a capture id.
		captured = reference
	default:
		return nil, err
	}

	payload := map[string]any{}
	if req.AmountMinor > 0 {
		payload["amount"] = map[string]string{
			"currency_code": strings.ToUpper(req.Currency),
			"value":         providers.FormatMajor(req.AmountMinor, req.Currency),
		}
	}
	if req.Reason != "" {
		payload["note_to_payer"] = req.Reason
	}
	idempotencyKey := "refund:" + captured + ":" + strconv.FormatInt(req.AmountMinor, 10)
	result, err := a.call(ctx, http.MethodPost, "/v2/payments/captures/"+url.PathEscape(captured)+"/refunds",
		payload, false, creds, idempotencyKey)
	if err != nil {
		return nil, err
	}
	var created refund
	if err := json.Unmarshal(result.Body, &created); err != nil {
		return nil, apperrors.ProviderUnavailable(providerName, err)
	}
	status := mapRefundStatus(created.Status)
	out := &providers.RefundResult{
		ProviderRefundID: created.ID,
		Status:           status,
		AmountMinor:      amountMinor(created.Amount),
		Currency:         strings.ToUpper(created.Amount.CurrencyCode),
	}
	if out.AmountMinor == 0 {
		out.AmountMinor = req.AmountMinor
	}
	if status == string(constants.RefundFailed) {
		out.FailureMessage = "PayPal rejected the refund."
	}
	return out, nil
}

func (a *Adapter) fetchOrder(ctx context.Context, creds providers.Credentials, id string) (*order, error) {
	result, err := a.call(ctx, http.MethodGet, "/v2/checkout/orders/"+url.PathEscape(id), nil, true, creds, "")
	if err != nil {
		return nil, err
	}
	var fetched order
	if err := json.Unmarshal(result.Body, &fetched); err != nil {
		return nil, apperrors.ProviderUnavailable(providerName, err)
	}
	return &fetched, nil
}

func (a *Adapter) captureOrder(ctx context.Context, creds providers.Credentials, id string) (*order, error) {
	result, err := a.call(ctx, http.MethodPost, "/v2/checkout/orders/"+url.PathEscape(id)+"/capture",
		map[string]any{}, false, creds, "capture:"+id)
	if err != nil {
		return nil, err
	}
	var captured order
	if err := json.Unmarshal(result.Body, &captured); err != nil {
		return nil, apperrors.ProviderUnavailable(providerName, err)
	}
	if captured.ID == "" {
		captured.ID = id
	}
	return &captured, nil
}

func appendQuery(baseURL, query string) string {
	separator := "?"
	if strings.Contains(baseURL, "?") {
		separator = "&"
	}
	return baseURL + separator + query
}
