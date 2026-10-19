package monnify

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

// VerifyWebhook validates Monnify's production signature and normalizes events.
// The official sandbox does not include the signature header.
func (a *Adapter) VerifyWebhook(req providers.WebhookVerificationRequest) (*providers.WebhookResult, error) {
	signature := header(req.Headers, "monnify-signature")
	if signature == "" {
		signature = req.Signature
	}
	if signature == "" && req.Environment != constants.EnvironmentTest {
		return nil, errSignature
	}
	if signature != "" && !signatureValid(req.SecretKey, req.Body, signature) {
		return nil, errSignature
	}

	var body webhookBody
	if err := json.Unmarshal(req.Body, &body); err != nil {
		return nil, errPayload
	}
	data := body.EventData
	reference := stringFrom(data, "paymentReference")
	currency := firstNonEmpty(stringFrom(data, "currency"), constants.DefaultCurrency)
	status := stringFrom(data, "paymentStatus")
	if status == "" {
		status = body.EventType
	}
	result := &providers.WebhookResult{
		SignatureValid:    true,
		Reference:         reference,
		ProviderReference: firstNonEmpty(stringFrom(data, "transactionReference"), reference),
		AmountMinor:       amountFrom(data["amountPaid"], currency, amountFrom(data["amount"], currency, 0)),
		Currency:          currency,
		Status:            mapPaymentStatus(status),
		Raw:               data,
	}
	result.Type = mapEventType(body.EventType, status)
	result.EventID = eventID(body.EventType, data, reference)
	return result, nil
}

func mapEventType(event, status string) string {
	switch strings.ToUpper(event) {
	case "SUCCESSFUL_TRANSACTION", "COMPLETED_OFFLINE_PAYMENT":
		return constants.EventPaymentSucceeded
	case "REJECTED_PAYMENT":
		return constants.EventPaymentFailed
	case "SUCCESSFUL_REFUND":
		return constants.EventRefundSucceeded
	case "FAILED_REFUND":
		return constants.EventRefundFailed
	}
	switch mapPaymentStatus(status) {
	case string(constants.PaymentSuccessful):
		return constants.EventPaymentSucceeded
	case string(constants.PaymentFailed):
		return constants.EventPaymentFailed
	default:
		return "provider.event"
	}
}

func eventID(event string, data map[string]any, reference string) string {
	id := firstNonEmpty(stringFrom(data, "transactionReference"), stringFrom(data, "refundReference"), reference)
	if numeric, ok := data["transactionReference"].(float64); ok {
		id = strconv.FormatInt(int64(numeric), 10)
	}
	return "monnify:" + id + ":" + event
}

func signatureValid(secret string, body []byte, signature string) bool {
	signature = strings.TrimSpace(signature)
	if secret == "" || signature == "" {
		return false
	}
	secretBody := make([]byte, 0, len(secret)+len(body))
	secretBody = append(secretBody, secret...)
	secretBody = append(secretBody, body...)
	digest := sha512.Sum512(secretBody)
	expected := hex.EncodeToString(digest[:])
	if hmac.Equal([]byte(expected), []byte(strings.ToLower(signature))) {
		return true
	}

	mac := hmac.New(sha512.New, []byte(secret))
	_, _ = mac.Write(body)
	hmacDigest := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(hmacDigest), []byte(strings.ToLower(signature)))
}

func stringFrom(data map[string]any, key string) string {
	value, _ := data[key].(string)
	return value
}

func header(headers map[string]string, name string) string {
	for key, value := range headers {
		if strings.EqualFold(key, name) {
			return value
		}
	}
	return ""
}
