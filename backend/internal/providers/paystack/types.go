// Package paystack implements the PayConnect adapter for Paystack.
//
// The types in this file describe Paystack's wire format and are private to this
// package: nothing outside ever sees them.
package paystack

// initializeRequest is the payload for POST /transaction/initialize.
type initializeRequest struct {
	Email       string         `json:"email"`
	Amount      int64          `json:"amount"` // Paystack takes minor units
	Currency    string         `json:"currency,omitempty"`
	Reference   string         `json:"reference,omitempty"`
	CallbackURL string         `json:"callback_url,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// transactionData is the shared transaction payload for initialize/verify.
type transactionData struct {
	ID               int64          `json:"id"`
	Reference        string         `json:"reference"`
	Status           string         `json:"status"`
	Amount           int64          `json:"amount"`
	Currency         string         `json:"currency"`
	GatewayResponse  string         `json:"gateway_response"`
	Channel          string         `json:"channel"`
	Fees             *int64         `json:"fees"`
	AuthorizationURL string         `json:"authorization_url"`
	Customer         map[string]any `json:"customer"`
	Metadata         map[string]any `json:"metadata"`
}

// envelope is Paystack's standard response shape.
type envelope struct {
	Status  bool            `json:"status"`
	Message string          `json:"message"`
	Data    transactionData `json:"data"`
}

// balanceEnvelope is the response of GET /balance.
type balanceEnvelope struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
}

// refundRequest is the payload for POST /refund.
type refundRequest struct {
	Transaction  string `json:"transaction"`
	Amount       int64  `json:"amount,omitempty"`
	MerchantNote string `json:"merchant_note,omitempty"`
}

// refundData is the response payload for POST /refund.
type refundData struct {
	ID       int64  `json:"id"`
	Status   string `json:"status"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	Message  string `json:"message"`
}

// refundEnvelope wraps the refund response.
type refundEnvelope struct {
	Status  bool       `json:"status"`
	Message string     `json:"message"`
	Data    refundData `json:"data"`
}

// webhookEvent is the inbound webhook payload.
type webhookEvent struct {
	Event string          `json:"event"`
	Data  transactionData `json:"data"`
}
