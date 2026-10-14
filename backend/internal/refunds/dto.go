package refunds

import "time"

// Response is the public refund representation.
type Response struct {
	ID             string `json:"id"`
	PaymentID      string `json:"payment_id"`
	Provider       string `json:"provider"`
	Amount         int64  `json:"amount"`
	Currency       string `json:"currency"`
	Status         string `json:"status"`
	Reason         string `json:"reason,omitempty"`
	FailureMessage string `json:"failure_message,omitempty"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// ToResponse maps the model onto the public DTO.
func ToResponse(r *Refund) Response {
	return Response{
		ID:             r.ID,
		PaymentID:      r.PaymentID,
		Provider:       r.Provider,
		Amount:         r.AmountMinor,
		Currency:       r.Currency,
		Status:         r.Status,
		Reason:         r.Reason,
		FailureMessage: r.FailureMessage,
		CreatedAt:      r.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      r.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
