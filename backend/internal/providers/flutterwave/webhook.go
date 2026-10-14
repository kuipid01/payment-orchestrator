package flutterwave

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

// VerifyWebhook authenticates an inbound Flutterwave webhook.
//
// Flutterwave authenticates with a "verif-hash" header that must equal the
// secret hash configured in the merchant's Flutterwave dashboard. That hash is
// chosen by the merchant and is independent of the API secret key, so it is
// stored on the connection as its own webhook secret. When none is stored we
// fall back to the API secret key, so connections configured before webhook
// secrets existed keep working. The remaining path is an HMAC-SHA256 of the raw
// body keyed with the same value, which some Flutterwave-compatible
// integrations use instead of the plain verif-hash.
func (a *Adapter) VerifyWebhook(req providers.WebhookVerificationRequest) (*providers.WebhookResult, error) {
	signature := req.Signature
	if signature == "" {
		signature = header(req.Headers, "verif-hash")
	}
	secret := req.WebhookSecret
	if secret == "" {
		secret = req.SecretKey
	}
	if !signatureValid(secret, req.Body, signature) {
		return nil, errSignature
	}

	var body webhookBody
	if err := json.Unmarshal(req.Body, &body); err != nil {
		return nil, errPayload
	}
	currency := stringFrom(body.Data, "currency")
	if currency == "" {
		currency = constants.DefaultCurrency
	}
	reference := firstNonEmpty(stringFrom(body.Data, "tx_ref"), body.TxRef)
	status := stringFrom(body.Data, "status")
	providerReference := stringFrom(body.Data, "flw_ref")

	result := &providers.WebhookResult{
		SignatureValid:    true,
		Reference:         reference,
		ProviderReference: firstNonEmpty(providerReference, reference),
		AmountMinor:       amountFrom(body.Data["amount"], currency),
		Currency:          currency,
		Status:            mapStatus(status),
		Raw:               body.Data,
	}
	result.EventID = eventID(body, reference)
	result.Type = mapEventType(body.Event, status)
	return result, nil
}

// mapEventType normalizes Flutterwave event names onto PayConnect event types.
func mapEventType(event, status string) string {
	switch strings.ToUpper(event) {
	case "CHARGE.COMPLETED", "TRANSFER.COMPLETED":
		return constants.EventPaymentSucceeded
	case "CHARGE.FAILED":
		return constants.EventPaymentFailed
	case "REFUND.COMPLETED":
		return constants.EventRefundSucceeded
	case "REFUND.FAILED":
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
	if id := stringifyID(body.ID); id != "" {
		return "flutterwave:" + id + ":" + body.Event
	}
	return "flutterwave:" + reference + ":" + body.Event
}

func stringifyID(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	case int64:
		return strconv.FormatInt(typed, 10)
	default:
		return ""
	}
}

func stringFrom(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	if value, ok := data[key].(string); ok {
		return value
	}
	return ""
}

// signatureValid accepts either the configured verif-hash or an HMAC-SHA256
// signature of the body.
func signatureValid(secret string, body []byte, signature string) bool {
	signature = strings.TrimSpace(signature)
	if secret == "" || signature == "" {
		return false
	}
	if hmac.Equal([]byte(secret), []byte(signature)) {
		return true
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(strings.ToLower(signature)))
}

func header(headers map[string]string, name string) string {
	for key, value := range headers {
		if strings.EqualFold(key, name) {
			return value
		}
	}
	return ""
}
