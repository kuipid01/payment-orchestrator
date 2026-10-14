package refunds

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/kuipid01/payconnect/internal/audit"
	"github.com/kuipid01/payconnect/internal/constants"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/events"
	"github.com/kuipid01/payconnect/internal/payments"
	"github.com/kuipid01/payconnect/internal/providers"
)

/* --------------------------------- ports ------------------------------------ */

// PaymentStore is the payment access the refund flow needs.
type PaymentStore interface {
	ByID(merchantID, id string) (*payments.Payment, error)
	Save(payment *payments.Payment) error
}

// CredentialProvider supplies decrypted provider credentials.
type CredentialProvider interface {
	CredentialsFor(merchantID, environment string, provider constants.Provider) (providers.Credentials, error)
}

// Registry resolves a provider to its adapter.
type Registry interface {
	Get(provider constants.Provider) (providers.Adapter, error)
}

// Ledger records the normalized refund movement.
type Ledger interface {
	RecordRefund(refund *RefundRecord) error
}

// RefundRecord is the ledger-facing view of a refund.
type RefundRecord struct {
	ID          string
	MerchantID  string
	PaymentID   string
	Provider    string
	Environment string
	AmountMinor int64
	Currency    string
	Status      string
	Reference   string
}

// Service owns refund rules.
type Service struct {
	Repo      *Repository
	Payments  PaymentStore
	Providers CredentialProvider
	Registry  Registry
	Ledger    Ledger
	Events    events.Publisher
	Audit     audit.Recorder
	Now       func() time.Time
}

// NewService builds the refund service.
func NewService(repo *Repository, paymentStore PaymentStore, credentials CredentialProvider,
	registry Registry, ledger Ledger, bus events.Publisher, recorder audit.Recorder) *Service {
	return &Service{
		Repo: repo, Payments: paymentStore, Providers: credentials, Registry: registry,
		Ledger: ledger, Events: bus, Audit: recorder, Now: time.Now,
	}
}

// Input is the internal, already-authenticated refund request.
type Input struct {
	MerchantID     string
	ActorName      string
	ActorRole      string
	PaymentID      string
	AmountMinor    int64
	Reason         string
	IdempotencyKey string
}

// Refund executes the refund flow: validate → provider → persist → transition.
func (s *Service) Refund(ctx context.Context, in Input) (*Refund, error) {
	if in.IdempotencyKey != "" {
		if existing, err := s.Repo.ByIdempotencyKey(in.MerchantID, in.IdempotencyKey); err == nil {
			return existing, nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	payment, err := s.Payments.ByID(in.MerchantID, in.PaymentID)
	if err != nil {
		return nil, apperrors.PaymentNotFound()
	}
	if !constants.PaymentStatus(payment.Status).IsSettled() {
		return nil, apperrors.PaymentNotRefundable("Only successful payments can be refunded.")
	}

	refunded, err := s.Repo.RefundedMinor(payment.ID)
	if err != nil {
		return nil, err
	}
	remaining := payment.AmountMinor - refunded
	if remaining <= 0 {
		return nil, apperrors.Conflict(apperrors.CodePaymentAlreadyRefunded, "This payment has already been fully refunded.")
	}
	amount := in.AmountMinor
	if amount == 0 {
		amount = remaining
	}
	if amount < 0 || amount > remaining {
		return nil, apperrors.InvalidAmount("The refund amount exceeds the refundable balance.")
	}

	adapter, err := s.Registry.Get(constants.Provider(payment.Provider))
	if err != nil {
		return nil, err
	}
	credentials, err := s.Providers.CredentialsFor(in.MerchantID, payment.Environment, constants.Provider(payment.Provider))
	if err != nil {
		return nil, err
	}

	now := s.Now().UTC()
	refund := &Refund{
		ID:             NewID(),
		MerchantID:     in.MerchantID,
		PaymentID:      payment.ID,
		Provider:       payment.Provider,
		AmountMinor:    amount,
		Currency:       payment.Currency,
		Status:         string(constants.RefundPending),
		Reason:         in.Reason,
		IdempotencyKey: in.IdempotencyKey,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	providerReference := ""
	if payment.ProviderPaymentID != nil {
		providerReference = *payment.ProviderPaymentID
	}

	result, err := adapter.RefundPayment(ctx, credentials, providers.RefundRequest{
		PaymentReference:  payment.Reference,
		ProviderReference: providerReference,
		AmountMinor:       amount,
		Currency:          payment.Currency,
		Reason:            in.Reason,
	})
	if err != nil {
		// The refund row is not created: nothing moved, and the caller may retry.
		return nil, apperrors.RefundFailed("", err)
	}

	refund.Status = normalizeStatus(result.Status)
	refund.SetProviderRefundID(result.ProviderRefundID)
	refund.FailureMessage = result.FailureMessage
	if refund.Status == string(constants.RefundFailed) && refund.FailureMessage == "" {
		refund.FailureMessage = "The provider rejected the refund."
	}
	if err := s.Repo.Create(refund); err != nil {
		return nil, err
	}

	s.applyToPayment(payment, refund, refunded+amount)
	if err := s.Payments.Save(payment); err != nil {
		return nil, err
	}

	if refund.Status == string(constants.RefundSuccessful) || refund.Status == string(constants.RefundProcessing) {
		s.recordLedger(refund, payment.Environment)
		s.publish(ctx, refund, payment)
	}
	s.Audit.Record(audit.Entry{
		MerchantID: in.MerchantID, ActorName: in.ActorName, ActorRole: in.ActorRole,
		Action: "Refund created", Detail: refund.ID + " (" + refund.Status + ")", Severity: "critical",
	})
	return refund, nil
}

// applyToPayment moves the payment to refunded/partially_refunded once the
// provider accepted the reversal. Terminal refund states are never regressed.
func (s *Service) applyToPayment(payment *payments.Payment, refund *Refund, totalRefunded int64) {
	if s.Now == nil {
		s.Now = time.Now
	}
	now := s.Now().UTC()
	switch refund.Status {
	case string(constants.RefundSuccessful), string(constants.RefundProcessing), string(constants.RefundPending):
	default:
		return
	}
	if totalRefunded >= payment.AmountMinor {
		payment.Status = string(constants.PaymentRefunded)
	} else {
		payment.Status = string(constants.PaymentPartiallyRefunded)
	}
	payment.AppendEvent("refund", "Refund "+refund.ID+" ("+refund.Status+")", now)
	payment.UpdatedAt = now
}

// Get returns one refund.
func (s *Service) Get(merchantID, id string) (*Refund, error) {
	refund, err := s.Repo.ByID(merchantID, id)
	if err != nil {
		return nil, apperrors.RefundNotFound()
	}
	return refund, nil
}

// List returns a page of refunds.
func (s *Service) List(f Filter) ([]Refund, int64, error) {
	rows, err := s.Repo.List(f)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.Repo.Count(f)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (s *Service) recordLedger(refund *Refund, environment string) {
	if s.Ledger == nil {
		return
	}
	_ = s.Ledger.RecordRefund(&RefundRecord{
		ID:          refund.ID,
		MerchantID:  refund.MerchantID,
		PaymentID:   refund.PaymentID,
		Provider:    refund.Provider,
		Environment: environment,
		AmountMinor: refund.AmountMinor,
		Currency:    refund.Currency,
		Status:      refund.Status,
		Reference:   refund.PaymentID,
	})
}

func (s *Service) publish(ctx context.Context, refund *Refund, payment *payments.Payment) {
	if s.Events == nil {
		return
	}
	eventType := constants.EventRefundCreated
	switch refund.Status {
	case string(constants.RefundSuccessful):
		eventType = constants.EventRefundSucceeded
	case string(constants.RefundFailed):
		eventType = constants.EventRefundFailed
	}
	event := events.New(eventType, refund.MerchantID, payment.Environment)
	event.RefundID = refund.ID
	event.PaymentID = refund.PaymentID
	event.Reference = payment.Reference
	if payload, err := json.Marshal(ToResponse(refund)); err == nil {
		event.Data = payload
	}
	s.Events.Publish(ctx, event)
}

func normalizeStatus(status string) string {
	switch constants.RefundStatus(status) {
	case constants.RefundSuccessful, constants.RefundProcessing, constants.RefundFailed, constants.RefundCancelled:
		return status
	default:
		return string(constants.RefundPending)
	}
}
