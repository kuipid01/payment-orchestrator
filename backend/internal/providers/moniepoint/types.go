// Package moniepoint implements the PayConnect adapter for Moniepoint.
//
// NOTE: Moniepoint does not publish a single canonical public API surface the
// way Paystack does. The endpoints below follow the documented REST conventions
// (bearer token, JSON, minor-unit amounts) and are isolated in this package so
// that confirming them against the merchant's Moniepoint API documentation only
// requires changing this file.
package moniepoint

// createRequest is the payload for POST /v1/payments. Moniepoint takes amounts
// in minor units, matching PayConnect's internal representation.
type createRequest struct {
	Reference   string         `json:"reference"`
	Amount      int64          `json:"amount"`
	Currency    string         `json:"currency"`
	Description string         `json:"description,omitempty"`
	CallbackURL string         `json:"callback_url,omitempty"`
	Customer    customer       `json:"customer"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// customer is the payer block.
type customer struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
	Phone string `json:"phone,omitempty"`
}

// envelope is the standard Moniepoint response wrapper.
type envelope struct {
	Status  string         `json:"status"`
	Success *bool          `json:"success"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

// refundRequest is the payload for POST /v1/refunds.
type refundRequest struct {
	PaymentReference string `json:"payment_reference"`
	Amount           int64  `json:"amount,omitempty"`
	Reason           string `json:"reason,omitempty"`
}

// webhookBody is the inbound webhook payload.
type webhookBody struct {
	Event string         `json:"event"`
	ID    any            `json:"id"`
	Data  map[string]any `json:"data"`
}
