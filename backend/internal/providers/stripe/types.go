// Package stripe implements the PayConnect adapter for Stripe Checkout.
//
// The types in this file describe Stripe's wire format and are private to this
// package: nothing outside ever sees them.
package stripe

import "encoding/json"

// session is a Stripe Checkout Session. Only the fields PayConnect acts on are
// decoded; the rest of Stripe's payload is ignored.
type session struct {
	ID                string            `json:"id"`
	Object            string            `json:"object"`
	Status            string            `json:"status"`
	PaymentStatus     string            `json:"payment_status"`
	AmountTotal       int64             `json:"amount_total"`
	Currency          string            `json:"currency"`
	URL               string            `json:"url"`
	ClientReferenceID string            `json:"client_reference_id"`
	PaymentIntent     string            `json:"payment_intent"`
	Metadata          map[string]string `json:"metadata"`
	ExpiresAt         int64             `json:"expires_at"`
}

// refund is a Stripe Refund.
type refund struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	Failure  string `json:"failure_reason"`
}

// charge is the subset of a Stripe Charge PayConnect reads from refund and
// dispute webhooks.
type charge struct {
	ID             string            `json:"id"`
	PaymentIntent  string            `json:"payment_intent"`
	Amount         int64             `json:"amount"`
	AmountRefunded int64             `json:"amount_refunded"`
	Currency       string            `json:"currency"`
	Metadata       map[string]string `json:"metadata"`
}

// webhookEvent is Stripe's event envelope.
type webhookEvent struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Object json.RawMessage `json:"object"`
	} `json:"data"`
}
