package constants

// PaymentStatus is the PayConnect public payment status. Provider-specific
// statuses must never appear here.
type PaymentStatus string

const (
	PaymentPending           PaymentStatus = "pending"
	PaymentProcessing        PaymentStatus = "processing"
	PaymentSuccessful        PaymentStatus = "successful"
	PaymentFailed            PaymentStatus = "failed"
	PaymentCancelled         PaymentStatus = "cancelled"
	PaymentExpired           PaymentStatus = "expired"
	PaymentRefunded          PaymentStatus = "refunded"
	PaymentPartiallyRefunded PaymentStatus = "partially_refunded"
)

// IsTerminal reports whether no further provider transition is expected.
func (s PaymentStatus) IsTerminal() bool {
	switch s {
	case PaymentSuccessful, PaymentFailed, PaymentCancelled, PaymentExpired,
		PaymentRefunded, PaymentPartiallyRefunded:
		return true
	}
	return false
}

// IsSettled reports whether the payment collected money.
func (s PaymentStatus) IsSettled() bool {
	return s == PaymentSuccessful || s == PaymentRefunded || s == PaymentPartiallyRefunded
}

// RefundStatus is the PayConnect public refund status.
type RefundStatus string

const (
	RefundPending    RefundStatus = "pending"
	RefundProcessing RefundStatus = "processing"
	RefundSuccessful RefundStatus = "successful"
	RefundFailed     RefundStatus = "failed"
	RefundCancelled  RefundStatus = "cancelled"
)

// TransactionStatus is the ledger-facing state of a financial event.
type TransactionStatus string

const (
	TransactionPending    TransactionStatus = "pending"
	TransactionSuccessful TransactionStatus = "successful"
	TransactionFailed     TransactionStatus = "failed"
	TransactionReversed   TransactionStatus = "reversed"
)
