package paystack

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/kuipid01/payconnect/internal/constants"
	"github.com/kuipid01/payconnect/internal/providers"
)

// VerifyWebhook validates the X-Paystack-Signature header (HMAC-SHA512 of the
// raw body keyed with the merchant's secret key) and normalizes the event.
func (a *Adapter) VerifyWebhook(req providers.WebhookVerificationRequest) (*providers.WebhookResult, error) {
	signature := req.Signature
	if signature == "" {
		signature = header(req.Headers, "X-Paystack-Signature")
	}
	if !validSignature(req.SecretKey, req.Body, signature) {
		return nil, errSignature
	}

	var event webhookEvent
	if err := json.Unmarshal(req.Body, &event); err != nil {
		return nil, errPayload
	}
	result := &providers.WebhookResult{
		SignatureValid:    true,
		Reference:         event.Data.Reference,
		ProviderReference: event.Data.Reference,
		AmountMinor:       event.Data.Amount,
		Currency:          event.Data.Currency,
		Status:            mapStatus(event.Data.Status),
		Raw: map[string]any{
			"event":    event.Event,
			"id":       event.Data.ID,
			"status":   event.Data.Status,
			"currency": event.Data.Currency,
		},
	}
	result.EventID = eventID(event)
	result.Type = mapEventType(event.Event, event.Data.Status)
	return result, nil
}

// mapEventType normalizes Paystack event names onto PayConnect event types.
func mapEventType(event, status string) string {
	switch strings.ToLower(event) {
	case "charge.success", "transaction.success":
		return constants.EventPaymentSucceeded
	case "charge.failed", "transaction.failed":
		return constants.EventPaymentFailed
	case "refund.processed", "refund.success":
		return constants.EventRefundSucceeded
	case "refund.failed":
		return constants.EventRefundFailed
	case "charge.dispute.create", "charge.dispute.remind":
		return constants.EventChargebackCreated
	default:
		switch mapStatus(status) {
		case string(constants.PaymentSuccessful):
			return constants.EventPaymentSucceeded
		case string(constants.PaymentFailed):
			return constants.EventPaymentFailed
		}
		return "provider.event"
	}
}

// eventID builds a stable dedupe key for the provider event.
func eventID(event webhookEvent) string {
	if event.Data.ID != 0 {
		return "paystack:" + strconv.FormatInt(event.Data.ID, 10) + ":" + event.Event
	}
	return "paystack:" + event.Data.Reference + ":" + event.Event
}

func validSignature(secret string, body []byte, signature string) bool {
	if secret == "" || signature == "" {
		return false
	}
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(strings.ToLower(strings.TrimSpace(signature))))
}

func header(headers map[string]string, name string) string {
	for key, value := range headers {
		if strings.EqualFold(key, name) {
			return value
		}
	}
	return ""
}
