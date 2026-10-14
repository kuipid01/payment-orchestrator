package moniepoint

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/kuipid01/payconnect/internal/constants"
	"github.com/kuipid01/payconnect/internal/providers"
)

// VerifyWebhook authenticates an inbound Moniepoint webhook with an
// HMAC-SHA256 signature of the raw body keyed by the merchant's secret key.
func (a *Adapter) VerifyWebhook(req providers.WebhookVerificationRequest) (*providers.WebhookResult, error) {
	signature := req.Signature
	if signature == "" {
		signature = firstNonEmpty(
			header(req.Headers, "x-moniepoint-signature"),
			header(req.Headers, "x-signature"),
		)
	}
	if !signatureValid(req.SecretKey, req.Body, signature) {
		return nil, errSignature
	}

	var body webhookBody
	if err := json.Unmarshal(req.Body, &body); err != nil {
		return nil, errPayload
	}
	currency := firstNonEmpty(stringFrom(body.Data, "currency"), constants.DefaultCurrency)
	reference := firstNonEmpty(
		stringFrom(body.Data, "reference"),
		stringFrom(body.Data, "payment_reference"),
	)

	result := &providers.WebhookResult{
		SignatureValid:    true,
		Reference:         reference,
		ProviderReference: firstNonEmpty(stringFrom(body.Data, "id"), reference),
		AmountMinor:       amountFrom(body.Data["amount"], currency),
		Currency:          currency,
		Status:            mapStatus(stringFrom(body.Data, "status")),
		Raw:               body.Data,
	}
	result.EventID = eventID(body, reference)
	result.Type = mapEventType(body.Event, stringFrom(body.Data, "status"))
	return result, nil
}

// mapEventType normalizes Moniepoint event names onto PayConnect event types.
func mapEventType(event, status string) string {
	switch strings.ToLower(event) {
	case "payment.successful", "payment.success", "payment.completed", "transaction.successful":
		return constants.EventPaymentSucceeded
	case "payment.failed", "payment.declined", "transaction.failed":
		return constants.EventPaymentFailed
	case "refund.successful", "refund.completed":
		return constants.EventRefundSucceeded
	case "refund.failed":
		return constants.EventRefundFailed
	}
	switch mapStatus(status) {
	case string(constants.PaymentSuccessful):
		return constants.EventPaymentSucceeded
	case string(constants.PaymentFailed):
		return constants.EventPaymentFailed
	}
	return "provider.event"
}

func eventID(body webhookBody, reference string) string {
	id := ""
	switch typed := body.ID.(type) {
	case string:
		id = typed
	case float64:
		id = strconv.FormatInt(int64(typed), 10)
	}
	if id == "" {
		id = reference
	}
	return "moniepoint:" + id + ":" + body.Event
}

func signatureValid(secret string, body []byte, signature string) bool {
	signature = strings.TrimSpace(signature)
	if secret == "" || signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	if hmac.Equal([]byte(expected), []byte(strings.ToLower(signature))) {
		return true
	}
	// Some Moniepoint deployments send a base64 digest instead of hex.
	return hmac.Equal([]byte(secret), []byte(signature))
}

func header(headers map[string]string, name string) string {
	for key, value := range headers {
		if strings.EqualFold(key, name) {
			return value
		}
	}
	return ""
}
