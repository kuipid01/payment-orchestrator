package transactions

import (
	"context"
	"time"

	"github.com/kuipid01/payconnect/internal/constants"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/payments"
	"github.com/kuipid01/payconnect/internal/platform/gormx"
)

// Service records normalized financial events and serves the ledger.
type Service struct {
	Repo *Repository
	Now  func() time.Time
}

// NewService builds the ledger service.
func NewService(repo *Repository) *Service {
	return &Service{Repo: repo, Now: time.Now}
}

// RecordPayment persists the ledger entry for a payment. It is idempotent:
// repeated provider notifications never double-count.
func (s *Service) RecordPayment(_ context.Context, payment *payments.Payment) error {
	if payment == nil {
		return nil
	}
	status := constants.PaymentStatus(payment.Status)
	if status != constants.PaymentSuccessful && status != constants.PaymentFailed &&
		status != constants.PaymentRefunded && status != constants.PaymentPartiallyRefunded {
		return nil
	}
	updated, err := s.Repo.UpdateForPayment(payment.MerchantID, payment.ID, "payment",
		ledgerStatus(status), payment.Reference)
	if err != nil {
		return err
	}
	if updated > 0 {
		return nil
	}
	return s.Repo.Create(&Transaction{
		ID:          gormx.NewID("txn"),
		MerchantID:  payment.MerchantID,
		PaymentID:   &payment.ID,
		Provider:    payment.Provider,
		Environment: payment.Environment,
		Type:        "payment",
		Direction:   "credit",
		AmountMinor: payment.AmountMinor,
		Currency:    payment.Currency,
		Status:      ledgerStatus(status),
		Reference:   payment.Reference,
		Description: "Payment via " + payment.Provider,
		CreatedAt:   s.Now().UTC(),
	})
}

// RecordRefund persists the ledger entry for a refund.
func (s *Service) RecordRefund(refund *RefundRecord) error {
	if refund == nil {
		return nil
	}
	exists, err := s.Repo.ExistsForRefund(refund.ID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return s.Repo.Create(&Transaction{
		ID:          gormx.NewID("txn"),
		MerchantID:  refund.MerchantID,
		PaymentID:   &refund.PaymentID,
		RefundID:    &refund.ID,
		Provider:    refund.Provider,
		Environment: refund.Environment,
		Type:        "refund",
		Direction:   "debit",
		AmountMinor: refund.AmountMinor,
		Currency:    refund.Currency,
		Status:      ledgerStatus(constants.PaymentStatus(refund.Status)),
		Reference:   refund.Reference,
		Description: "Refund for " + refund.PaymentID,
		CreatedAt:   s.Now().UTC(),
	})
}

// RefundRecord is the minimal refund view the ledger needs, which keeps the
// transactions package independent from the refunds package.
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

// Get returns one ledger row.
func (s *Service) Get(merchantID, id string) (*Transaction, error) {
	row, err := s.Repo.ByID(merchantID, id)
	if err != nil {
		return nil, apperrors.NotFound("transaction_not_found", "Transaction not found.")
	}
	return row, nil
}

// List returns a page of ledger rows.
func (s *Service) List(f Filter) ([]Transaction, int64, error) {
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

func ledgerStatus(status constants.PaymentStatus) string {
	switch status {
	case constants.PaymentSuccessful:
		return string(constants.TransactionSuccessful)
	case constants.PaymentFailed:
		return string(constants.TransactionFailed)
	case constants.PaymentRefunded, constants.PaymentPartiallyRefunded:
		return string(constants.TransactionReversed)
	default:
		return string(constants.TransactionPending)
	}
}
