package monnify

import "encoding/json"

type initializeRequest struct {
	Amount             json.Number    `json:"amount"`
	CustomerName       string         `json:"customerName"`
	CustomerEmail      string         `json:"customerEmail"`
	PaymentReference   string         `json:"paymentReference"`
	PaymentDescription string         `json:"paymentDescription"`
	CurrencyCode       string         `json:"currencyCode"`
	ContractCode       string         `json:"contractCode"`
	RedirectURL        string         `json:"redirectUrl,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
}

type transactionBody struct {
	TransactionReference string `json:"transactionReference"`
	PaymentReference     string `json:"paymentReference"`
	CheckoutURL          string `json:"checkoutUrl"`
	PaymentStatus        string `json:"paymentStatus"`
	Amount               any    `json:"amount"`
	AmountPaid           any    `json:"amountPaid"`
	Currency             string `json:"currency"`
}

type refundRequest struct {
	TransactionReference string      `json:"transactionReference"`
	RefundReference      string      `json:"refundReference"`
	RefundReason         string      `json:"refundReason"`
	CustomerNote         string      `json:"customerNote"`
	CurrencyCode         string      `json:"currencyCode"`
	RefundAmount         json.Number `json:"refundAmount"`
}

type refundBody struct {
	RefundReference string `json:"refundReference"`
	RefundStatus    string `json:"refundStatus"`
	RefundAmount    any    `json:"refundAmount"`
}

type webhookBody struct {
	EventType string         `json:"eventType"`
	EventData map[string]any `json:"eventData"`
}
