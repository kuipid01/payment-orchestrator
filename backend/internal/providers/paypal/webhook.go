package paypal

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/providers"
)

var (
	errSignature = apperrors.WebhookSignatureInvalid()
	errPayload   = apperrors.WebhookEventInvalid("The PayPal webhook payload could not be parsed.")
)

// VerifyWebhook authenticates a delivery the way PayPal requires: a postback to
// PayPal's verify-webhook-signature endpoint, signed with the merchant's own
// credentials and naming the webhook registration id.
//
// Nothing in the body is trusted before that postback returns SUCCESS, so a
// forged event cannot move a payment.
func (a *Adapter) VerifyWebhook(req providers.WebhookVerificationRequest) (*providers.WebhookResult, error) {
	transmissionID := header(req.Headers, "paypal-transmission-id")
	transmissionTime := header(req.Headers, "paypal-transmission-time")
	certURL := header(req.Headers, "paypal-cert-url")
	authAlgo := header(req.Headers, "paypal-auth-algo")
	transmissionSig := header(req.Headers, "paypal-transmission-sig")
	if transmissionID == "" || transmissionTime == "" || certURL == "" || authAlgo == "" || transmissionSig == "" {
		return nil, errSignature
	}
	// WebhookSecret carries the webhook registration id configured in the
	// merchant's PayPal app.
	if strings.TrimSpace(req.WebhookSecret) == "" || strings.TrimSpace(req.PublicKey) == "" {
		return nil, errSignature
	}

	creds := providers.Credentials{
		SecretKey:   req.SecretKey,
		PublicKey:   req.PublicKey,
		Environment: req.Environment,
	}
	verification := map[string]any{
		"auth_algo":         authAlgo,
		"cert_url":          certURL,
		"transmission_id":   transmissionID,
		"transmission_sig":  transmissionSig,
		"transmission_time": transmissionTime,
		"webhook_id":        req.WebhookSecret,
		"webhook_event":     json.RawMessage(req.Body),
	}
	result, err := a.call(context.Background(), http.MethodPost,
		"/v1/notifications/verify-webhook-signature", verification, true, creds, "")
	if err != nil {
		return nil, err
	}
	var verdict verifyWebhookResponse
	if err := json.Unmarshal(result.Body, &verdict); err != nil {
		return nil, apperrors.ProviderUnavailable(providerName, err)
	}
	if !strings.EqualFold(verdict.VerificationStatus, "SUCCESS") {
		return nil, errSignature
	}

	var event webhookEvent
	if err := json.Unmarshal(req.Body, &event); err != nil {
		return nil, errPayload
	}
	var resource webhookResource
	if err := json.Unmarshal(event.Resource, &resource); err != nil {
		return nil, errPayload
	}

	reference := resource.CustomID
	providerReference := resource.ID
	amount := amountMinor(resource.Amount)
	currency := strings.ToUpper(resource.Amount.CurrencyCode)
	if len(resource.PurchaseUnits) > 0 {
		unit := resource.PurchaseUnits[0]
		if reference == "" {
			reference = unit.CustomID
		}
		if amount == 0 {
			amount = amountMinor(unit.Amount)
		}
		if currency == "" {
			currency = strings.ToUpper(unit.Amount.CurrencyCode)
		}
	}
	// A capture event identifies the capture; the payment PayConnect stored is
	// keyed by the order id it relates to.
	if related := resource.SupplementaryData.RelatedIDs.OrderID; related != "" {
		providerReference = related
	}

	// PayPal authorizes an order and requires an explicit capture. The customer
	// return path captures too, but a buyer who closes the tab after approving
	// would otherwise leave the money uncaptured, so capture here as well.
	// PayPal-Request-Id makes both paths idempotent with each other.
	eventType := event.EventType
	if strings.EqualFold(eventType, "CHECKOUT.ORDER.APPROVED") && providerReference != "" {
		captured, captureErr := a.captureOrder(context.Background(), creds, providerReference)
		if captureErr == nil && strings.EqualFold(captured.Status, "COMPLETED") {
			eventType = "CHECKOUT.ORDER.COMPLETED"
			if len(captured.PurchaseUnits) > 0 {
				unit := captured.PurchaseUnits[0]
				if amount == 0 {
					amount = amountMinor(unit.Amount)
				}
				if currency == "" {
					currency = strings.ToUpper(unit.Amount.CurrencyCode)
				}
				if reference == "" {
					reference = unit.CustomID
				}
			}
		}
	}
	normalizedType, status := mapWebhookEvent(eventType)
	eventID := event.ID
	if eventID == "" {
		eventID = event.EventType + ":" + providerReference
	}
	return &providers.WebhookResult{
		EventID:           "paypal:" + eventID,
		Type:              normalizedType,
		Reference:         reference,
		ProviderReference: providerReference,
		AmountMinor:       amount,
		Currency:          currency,
		Status:            status,
		SignatureValid:    true,
		Raw: map[string]any{
			"event":  event.EventType,
			"id":     resource.ID,
			"status": resource.Status,
		},
	}, nil
}

func header(headers map[string]string, name string) string {
	for key, value := range headers {
		if strings.EqualFold(key, name) {
			return value
		}
	}
	return ""
}
