package stripe

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

// TestCredentials calls GET /v1/balance, which requires a valid secret key.
func (a *Adapter) TestCredentials(ctx context.Context, creds providers.Credentials) error {
	if _, err := a.call(ctx, http.MethodGet, "/v1/balance", nil, "", creds, true); err != nil {
		return err
	}
	return nil
}

// CreatePayment creates a hosted Stripe Checkout Session and returns its URL.
// The reference is the idempotency key, so a retried create cannot charge twice.
func (a *Adapter) CreatePayment(ctx context.Context, creds providers.Credentials,
	req providers.PaymentRequest) (*providers.PaymentResult, error) {
	successURL, cancelURL, err := checkoutURLs(req.CallbackURL)
	if err != nil {
		return nil, err
	}
	form, _ := checkoutForm(req, successURL, cancelURL, a.cfg.ManagedPayments,
		a.cfg.DefaultTaxCode)

	result, err := a.call(ctx, http.MethodPost, "/v1/checkout/sessions", form,
		"checkout:"+req.Reference, creds, false)
	if err != nil {
		return nil, err
	}
	var created session
	if err := json.Unmarshal(result.Body, &created); err != nil {
		return nil, apperrors.ProviderUnavailable(providerName, err)
	}
	if created.URL == "" || created.ID == "" {
		return nil, apperrors.ProviderRequestFailed(providerName,
			errors.New("Stripe returned a checkout session without a URL"))
	}
	if created.AmountTotal == 0 {
		created.AmountTotal = req.AmountMinor
	}
	if created.ClientReferenceID == "" {
		created.ClientReferenceID = req.Reference
	}
	return resultFromSession(created), nil
}

// GetPayment fetches the current state of a Checkout Session.
func (a *Adapter) GetPayment(ctx context.Context, creds providers.Credentials, reference string) (*providers.PaymentResult, error) {
	return a.VerifyPayment(ctx, creds, reference)
}

// VerifyPayment retrieves the Checkout Session by its provider id. The payments
// domain stores that id on the payment as soon as it is created, so
// reconciliation never trusts a local status.
func (a *Adapter) VerifyPayment(ctx context.Context, creds providers.Credentials, reference string) (*providers.PaymentResult, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return nil, apperrors.Validation("stripe_reference_required",
			"A Stripe payment is verified with its checkout session id (cs_…).")
	}
	result, err := a.call(ctx, http.MethodGet, "/v1/checkout/sessions/"+url.PathEscape(reference), nil, "", creds, true)
	if err != nil {
		return nil, err
	}
	var retrieved session
	if err := json.Unmarshal(result.Body, &retrieved); err != nil {
		return nil, apperrors.ProviderUnavailable(providerName, err)
	}
	return resultFromSession(retrieved), nil
}

// RefundPayment refunds all or part of a settled Checkout Session. Stripe
// refunds against the PaymentIntent (or Charge), so the session is resolved to
// its PaymentIntent first.
func (a *Adapter) RefundPayment(ctx context.Context, creds providers.Credentials,
	req providers.RefundRequest) (*providers.RefundResult, error) {
	field, target, err := a.refundTarget(ctx, creds, req.ProviderReference)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set(field, target)
	if req.AmountMinor > 0 {
		form.Set("amount", strconv.FormatInt(req.AmountMinor, 10))
	}
	if req.Reason != "" {
		form.Set("metadata[reason]", req.Reason)
	}
	if req.PaymentReference != "" {
		form.Set("metadata[payment_reference]", req.PaymentReference)
	}
	idempotencyKey := "refund:" + firstNonEmpty(req.ProviderReference, req.PaymentReference) + ":" + strconv.FormatInt(req.AmountMinor, 10)
	result, err := a.call(ctx, http.MethodPost, "/v1/refunds", form, idempotencyKey, creds, false)
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
		AmountMinor:      created.Amount,
		Currency:         strings.ToUpper(created.Currency),
	}
	if status == string(constants.RefundFailed) {
		out.FailureMessage = firstNonEmpty(created.Failure, "Stripe rejected the refund.")
	}
	if out.AmountMinor == 0 {
		out.AmountMinor = req.AmountMinor
	}
	return out, nil
}

// refundTarget resolves whatever provider reference a payment carries into the
// field Stripe's refund endpoint accepts.
func (a *Adapter) refundTarget(ctx context.Context, creds providers.Credentials, reference string) (string, string, error) {
	reference = strings.TrimSpace(reference)
	switch {
	case strings.HasPrefix(reference, "pi_"):
		return "payment_intent", reference, nil
	case strings.HasPrefix(reference, "ch_"):
		return "charge", reference, nil
	case strings.HasPrefix(reference, "cs_"):
		session, err := a.retrieveSession(ctx, creds, reference)
		if err != nil {
			return "", "", err
		}
		if session.PaymentIntent == "" {
			return "", "", apperrors.Validation("stripe_refund_not_captured",
				"Stripe has no PaymentIntent for this checkout session yet, so it cannot be refunded.")
		}
		return "payment_intent", session.PaymentIntent, nil
	default:
		return "", "", apperrors.Validation("stripe_refund_reference_invalid",
			"A Stripe refund needs the checkout session, PaymentIntent or Charge id of the payment.")
	}
}

func (a *Adapter) retrieveSession(ctx context.Context, creds providers.Credentials, id string) (*session, error) {
	result, err := a.call(ctx, http.MethodGet, "/v1/checkout/sessions/"+url.PathEscape(id), nil, "", creds, true)
	if err != nil {
		return nil, err
	}
	var retrieved session
	if err := json.Unmarshal(result.Body, &retrieved); err != nil {
		return nil, apperrors.ProviderUnavailable(providerName, err)
	}
	return &retrieved, nil
}

// checkoutForm builds the Checkout Session creation payload.
func checkoutForm(
	req providers.PaymentRequest,
	successURL, cancelURL string,
	managedPayments bool,
	defaultTaxCode string,
) (url.Values, error) {
	name := strings.TrimSpace(req.Description)
	if name == "" {
		name = "Test payment"
		if strings.TrimSpace(req.Reference) != "" {
			name += " " + req.Reference
		}
	}

	form := url.Values{}
	form.Set("mode", "payment")
	form.Set("success_url", successURL)
	form.Set("cancel_url", cancelURL)
	form.Set("client_reference_id", req.Reference)
	form.Set("line_items[0][quantity]", "1")
	form.Set("line_items[0][price_data][currency]",
		strings.ToLower(req.Currency))
	form.Set("line_items[0][price_data][unit_amount]",
		strconv.FormatInt(req.AmountMinor, 10))
	form.Set("line_items[0][price_data][product_data][name]", name)

	if managedPayments {
		taxCode := strings.TrimSpace(defaultTaxCode)
		if taxCode == "" {
			return nil, errors.New(
				"Stripe Managed Payments requires a configured product tax code",
			)
		}

		form.Set(
			"line_items[0][price_data][product_data][tax_code]",
			taxCode,
		)
	}

	if req.Email != "" {
		form.Set("customer_email", req.Email)
	}

	form.Set("metadata[reference]", req.Reference)
	form.Set("payment_intent_data[metadata][reference]", req.Reference)

	for key, value := range req.Metadata {
		if key == "" || key == "reference" || value == nil {
			continue
		}
		form.Set("metadata["+key+"]", metadataString(value))
	}

	return form, nil
}

// checkoutURLs builds the hosted page's return URLs. Stripe requires both.
func checkoutURLs(callbackURL string) (string, string, error) {
	callbackURL = strings.TrimSpace(callbackURL)
	if callbackURL == "" {
		return "", "", apperrors.Validation("checkout_callback_required",
			"A callback_url is required for Stripe Checkout so the customer can return to the store.")
	}
	success := appendQuery(callbackURL, "session_id={CHECKOUT_SESSION_ID}")
	cancel := appendQuery(callbackURL, "payment=cancelled")
	return success, cancel, nil
}

func appendQuery(baseURL, query string) string {
	separator := "?"
	if strings.Contains(baseURL, "?") {
		separator = "&"
	}
	return baseURL + separator + query
}

// metadataString renders one metadata value for a form-encoded Stripe field.
func metadataString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case bool:
		return strconv.FormatBool(typed)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return string(encoded)
	}
}
