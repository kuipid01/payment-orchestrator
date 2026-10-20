// Package paypal implements the PayConnect adapter for PayPal Orders v2.
//
// The types in this file describe PayPal's wire format and are private to this
// package: nothing outside ever sees them.
package paypal

import "encoding/json"

// money is PayPal's decimal amount. PayPal takes major units, unlike the minor
// units PayConnect stores, so every amount crosses through ParseMajor or
// FormatMajor.
type money struct {
	CurrencyCode string `json:"currency_code"`
	Value        string `json:"value"`
}

// link is one HATEOAS link on a PayPal resource. The approval URL the customer
// is redirected to is the "approve" (or newer "payer-action") link.
type link struct {
	Rel    string `json:"rel"`
	Href   string `json:"href"`
	Method string `json:"method"`
}

// capture is a settled payment against an order.
type capture struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Amount   money  `json:"amount"`
	CustomID string `json:"custom_id"`
}

// purchaseUnit is the order's line-item aggregate.
type purchaseUnit struct {
	ReferenceID string   `json:"reference_id"`
	CustomID    string   `json:"custom_id"`
	InvoiceID   string   `json:"invoice_id"`
	Amount      money    `json:"amount"`
	Payments    payments `json:"payments"`
}

type payments struct {
	Captures []capture `json:"captures"`
}

// order is a PayPal Orders v2 order.
type order struct {
	ID            string         `json:"id"`
	Status        string         `json:"status"`
	PurchaseUnits []purchaseUnit `json:"purchase_units"`
	Links         []link         `json:"links"`
}

// tokenResponse is the OAuth client_credentials response.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

// refund is a PayPal refund.
type refund struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Amount money  `json:"amount"`
}

// verifyWebhookResponse is PayPal's server-to-server verification verdict.
type verifyWebhookResponse struct {
	VerificationStatus string `json:"verification_status"`
}

// webhookEvent is PayPal's event envelope.
type webhookEvent struct {
	ID        string          `json:"id"`
	EventType string          `json:"event_type"`
	Resource  json.RawMessage `json:"resource"`
}

// webhookResource covers both order resources and capture resources. Order
// events carry purchase_units; capture and refund events carry amount and a
// related order id.
type webhookResource struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	CustomID string `json:"custom_id"`
	Amount   money  `json:"amount"`
	// RelatedIDs maps a capture back to its order, which is the identifier
	// PayConnect stores on the payment.
	SupplementaryData struct {
		RelatedIDs struct {
			OrderID string `json:"order_id"`
		} `json:"related_ids"`
	} `json:"supplementary_data"`
	PurchaseUnits []purchaseUnit `json:"purchase_units"`
}
