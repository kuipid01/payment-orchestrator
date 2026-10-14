package providers

import (
	"context"

	"github.com/kuipid01/payconnect/internal/constants"
)

// Adapter is the single contract every payment provider implements. Adapters
// translate PayConnect domain objects to provider payloads and back — no
// provider response struct ever crosses this boundary.
type Adapter interface {
	// Name identifies the provider this adapter speaks to.
	Name() constants.Provider

	// TestCredentials validates stored credentials without moving money.
	TestCredentials(ctx context.Context, creds Credentials) error

	// CreatePayment starts a charge and returns the provider's normalized result.
	CreatePayment(ctx context.Context, creds Credentials, req PaymentRequest) (*PaymentResult, error)

	// GetPayment fetches the provider's current view of a payment.
	GetPayment(ctx context.Context, creds Credentials, reference string) (*PaymentResult, error)

	// VerifyPayment is the reconciliation path used when a create call timed out
	// or returned an unknown state. It must never be skipped in that case.
	VerifyPayment(ctx context.Context, creds Credentials, reference string) (*PaymentResult, error)

	// RefundPayment refunds all or part of a settled payment.
	RefundPayment(ctx context.Context, creds Credentials, req RefundRequest) (*RefundResult, error)

	// VerifyWebhook authenticates and normalizes an inbound provider webhook.
	VerifyWebhook(req WebhookVerificationRequest) (*WebhookResult, error)
}
