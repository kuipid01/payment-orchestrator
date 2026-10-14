package transactions

import "gorm.io/gorm"

// Repository owns ledger persistence.
type Repository struct{ DB *gorm.DB }

// NewRepository builds the ledger repository.
func NewRepository(db *gorm.DB) *Repository { return &Repository{DB: db} }

// Filter narrows a ledger listing.
type Filter struct {
	MerchantID  string
	Environment string
	Type        string
	Status      string
	PaymentID   string
	Offset      int
	Limit       int
}

// Create inserts a ledger row.
func (r *Repository) Create(t *Transaction) error { return r.DB.Create(t).Error }

// ByID loads a ledger row scoped to its merchant.
func (r *Repository) ByID(merchantID, id string) (*Transaction, error) {
	var t Transaction
	if err := r.DB.Where("merchant_id = ?", merchantID).First(&t, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// ExistsForPayment reports whether a payment already has a ledger row of a type.
// This is what makes ledger writes idempotent when provider webhooks repeat.
func (r *Repository) ExistsForPayment(merchantID, paymentID, txType string) (bool, error) {
	var count int64
	if err := r.DB.Model(&Transaction{}).
		Where("merchant_id = ? AND payment_id = ? AND type = ?", merchantID, paymentID, txType).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// UpdateForPayment converges a payment's existing ledger row onto its latest
// terminal status. A payment that failed and was later confirmed successful must
// not leave a stale "failed" row behind in the ledger.
func (r *Repository) UpdateForPayment(merchantID, paymentID, txType, status, reference string) (int64, error) {
	res := r.DB.Model(&Transaction{}).
		Where("merchant_id = ? AND payment_id = ? AND type = ?", merchantID, paymentID, txType).
		Updates(map[string]any{"status": status, "reference": reference})
	return res.RowsAffected, res.Error
}

// ExistsForRefund reports whether a refund already has a ledger row.
func (r *Repository) ExistsForRefund(refundID string) (bool, error) {
	var count int64
	if err := r.DB.Model(&Transaction{}).Where("refund_id = ?", refundID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// List returns a page of ledger rows.
func (r *Repository) List(f Filter) ([]Transaction, error) {
	q := r.DB.Where("merchant_id = ?", f.MerchantID)
	if f.Environment != "" {
		q = q.Where("environment = ?", f.Environment)
	}
	if f.Type != "" {
		q = q.Where("type = ?", f.Type)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.PaymentID != "" {
		q = q.Where("payment_id = ?", f.PaymentID)
	}
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	var rows []Transaction
	if err := q.Order("created_at DESC").Offset(f.Offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// Count returns the number of ledger rows matching a filter.
func (r *Repository) Count(f Filter) (int64, error) {
	q := r.DB.Model(&Transaction{}).Where("merchant_id = ?", f.MerchantID)
	if f.Environment != "" {
		q = q.Where("environment = ?", f.Environment)
	}
	if f.Type != "" {
		q = q.Where("type = ?", f.Type)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}
