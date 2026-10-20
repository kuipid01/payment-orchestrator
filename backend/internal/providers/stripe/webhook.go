package stripe

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/kuipid01/payconnect/internal/constants"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/providers"
)

var (
	errSignature = apperrors.WebhookSignatureInvalid()
	errPayload   = apperrors.WebhookEventInvalid("The Stripe webhook payload could not be parsed.")
)

// signatureTolerance bounds how old a signed webhook may be. Stripe recommends
// five minutes; a replay outside that window is rejected even with a valid MAC.
const signatureTolerance = 5 * time.Minute

// VerifyWebhook validates the Stripe-Signature header (HMAC-SHA256 of
// "<timestamp>.<raw body>" keyed with the endpoint's signing secret) and
// normalizes the event.
func (a *Adapter) VerifyWebhook(req providers.WebhookVerificationRequest) (*providers.WebhookResult, error) {
	signature := req.Signature
	if signature == "" {
		signature = header(req.Headers, "Stripe-Signature")
	}
	if !validSignature(req.WebhookSecret, req.Body, signature, time.Now()) {
		return nil, errSignature
	}

	var event webhookEvent
	if err := json.Unmarshal(req.Body, &event); err != nil {
		return nil, errPayload
	}

	switch strings.ToLower(event.Type) {
	case "charge.refunded":
		var chargeObject charge
		if err := json.Unmarshal(event.Data.Object, &chargeObject); err != nil {
			return nil, errPayload
		}
		result := &providers.WebhookResult{
			EventID:           eventID(event),
			Type:              constants.EventRefundSucceeded,
			Reference:         chargeObject.Metadata["reference"],
			ProviderReference: firstNonEmpty(chargeObject.PaymentIntent, chargeObject.ID),
			AmountMinor:       chargeObject.AmountRefunded,
			Currency:          strings.ToUpper(chargeObject.Currency),
			SignatureValid:    true,
			Raw: map[string]any{
				"event": event.Type,
				"id":    chargeObject.ID,
			},
		}
		return result, nil
	case "charge.dispute.created", "charge.dispute.updated":
		var chargeObject charge
		if err := json.Unmarshal(event.Data.Object, &chargeObject); err != nil {
			return nil, errPayload
		}
		result := &providers.WebhookResult{
			EventID:           eventID(event),
			Type:              constants.EventChargebackCreated,
			Reference:         chargeObject.Metadata["reference"],
			ProviderReference: firstNonEmpty(chargeObject.PaymentIntent, chargeObject.ID),
			AmountMinor:       chargeObject.Amount,
			Currency:          strings.ToUpper(chargeObject.Currency),
			SignatureValid:    true,
			Raw: map[string]any{
				"event": event.Type,
				"id":    chargeObject.ID,
			},
		}
		return result, nil
	}

	var sessionObject session
	if err := json.Unmarshal(event.Data.Object, &sessionObject); err != nil {
		return nil, errPayload
	}
	result := &providers.WebhookResult{
		EventID:           eventID(event),
		Type:              mapEventType(event.Type, sessionObject),
		Reference:         firstNonEmpty(sessionObject.ClientReferenceID, sessionObject.Metadata["reference"]),
		ProviderReference: sessionObject.ID,
		AmountMinor:       sessionObject.AmountTotal,
		Currency:          strings.ToUpper(sessionObject.Currency),
		Status:            mapSessionStatus(sessionObject),
		SignatureValid:    true,
		Raw: map[string]any{
			"event":          event.Type,
			"id":             sessionObject.ID,
			"status":         sessionObject.Status,
			"payment_status": sessionObject.PaymentStatus,
			"currency":       sessionObject.Currency,
		},
	}
	switch result.Type {
	case "provider.event":
		// An event PayConnect does not act on must not move payment state.
		result.Status = ""
	case constants.EventPaymentFailed:
		// checkout.session.async_payment_failed leaves the session unpaid, so
		// its payment_status alone would map to processing. The event says the
		// attempt failed: keep an expired session expired, otherwise fail it.
		if result.Status != string(constants.PaymentExpired) && result.Status != string(constants.PaymentCancelled) {
			result.Status = string(constants.PaymentFailed)
		}
	case constants.EventPaymentProcessing:
		result.Status = string(constants.PaymentProcessing)
	}
	return result, nil
}

// eventID builds a stable dedupe key. Stripe retries the same event id, so
// deduping on it makes redelivery a no-op.
func eventID(event webhookEvent) string {
	if event.ID != "" {
		return "stripe:" + event.ID
	}
	return "stripe:" + event.Type
}

// validSignature verifies the t=…,v1=… header Stripe sends. Multiple v1 values
// are supported because Stripe rotates signing secrets with an overlap.
func validSignature(secret string, body []byte, signature string, now time.Time) bool {
	secret = strings.TrimSpace(secret)
	signature = strings.TrimSpace(signature)
	if secret == "" || signature == "" {
		return false
	}
	timestamp := ""
	digests := make([]string, 0, 2)
	for _, part := range strings.Split(signature, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch key {
		case "t":
			timestamp = value
		case "v1":
			digests = append(digests, value)
		}
	}
	if timestamp == "" || len(digests) == 0 {
		return false
	}
	seconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	if delta := now.Sub(time.Unix(seconds, 0)); delta > signatureTolerance || delta < -signatureTolerance {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	for _, digest := range digests {
		if hmac.Equal([]byte(expected), []byte(strings.ToLower(digest))) {
			return true
		}
	}
	return false
}

func header(headers map[string]string, name string) string {
	for key, value := range headers {
		if strings.EqualFold(key, name) {
			return value
		}
	}
	return ""
}
