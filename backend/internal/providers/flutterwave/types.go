// Package flutterwave implements the PayConnect adapter for Flutterwave.
//
// These types describe Flutterwave's wire format and never leave the package.
package flutterwave

import "encoding/json"

// createRequest is the payload for POST /payments. Flutterwave expects the
// amount in MAJOR units, so the adapter converts from minor units without
// floating point arithmetic.
type createRequest struct {
	TxRef          string         `json:"tx_ref"`
	Amount         string         `json:"amount"`
	Currency       string         `json:"currency"`
	RedirectURL    string         `json:"redirect_url,omitempty"`
	PaymentOptions string         `json:"payment_options,omitempty"`
	Customer       customer       `json:"customer"`
	Meta           map[string]any `json:"meta,omitempty"`
}

// customer is the payer block.
type customer struct {
	Email       string `json:"email"`
	Name        string `json:"name,omitempty"`
	PhoneNumber string `json:"phonenumber,omitempty"`
}

// envelope is Flutterwave's standard response wrapper. Data is kept raw because
// Flutterwave is not consistent about its shape: most endpoints return an object,
// but some (GET /balances) return an array. Decoding straight into
// map[string]any made a perfectly valid success response fail to unmarshal, which
// surfaced as a bogus "provider unavailable".
type envelope struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// object decodes data as a JSON object, returning nil when the payload holds a
// list or scalar instead (GET /balances returns an array).
func (e envelope) object() map[string]any {
	if len(e.Data) == 0 {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(e.Data, &out); err != nil {
		return nil
	}
	return out
}

// transaction is the transaction payload returned by verification.
type transaction struct {
	ID                int64  `json:"id"`
	TxRef             string `json:"tx_ref"`
	Status            string `json:"status"`
	Amount            any    `json:"amount"`
	Currency          string `json:"currency"`
	FlwRef            string `json:"flw_ref"`
	ChargedAmount     any    `json:"charged_amount"`
	AppFee            any    `json:"app_fee"`
	ProcessorResponse string `json:"processor_response"`
}

// refundRequest is the payload for POST /refunds.
type refundRequest struct {
	ID       string `json:"id,omitempty"`
	TxRef    string `json:"tx_ref,omitempty"`
	Amount   string `json:"amount,omitempty"`
	Comments string `json:"comments,omitempty"`
}

// webhookBody is the inbound webhook payload.
type webhookBody struct {
	Event  string         `json:"event"`
	Data   map[string]any `json:"data"`
	TxRef  string         `json:"txRef"`
	ID     any            `json:"id"`
	Status string         `json:"status"`
}
