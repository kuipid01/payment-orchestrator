package refunds

import "gorm.io/gorm"

// Repository owns refund persistence.
type Repository struct{ DB *gorm.DB }

// NewRepository builds the refund repository.
func NewRepository(db *gorm.DB) *Repository { return &Repository{DB: db} }

// Filter narrows a refund listing.
type Filter struct {
	MerchantID  string
	PaymentID   string
	Status      string
	Environment string
	Offset      int
	Limit       int
}

// Create inserts a refund.
func (r *Repository) Create(refund *Refund) error { return r.DB.Create(refund).Error }

// Save persists refund changes.
func (r *Repository) Save(refund *Refund) error { return r.DB.Save(refund).Error }

// ByID loads a refund scoped to its merchant.
func (r *Repository) ByID(merchantID, id string) (*Refund, error) {
	var refund Refund
	if err := r.DB.Where("merchant_id = ?", merchantID).First(&refund, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &refund, nil
}

// ByIdempotencyKey loads a refund previously created with the same key, so a
// retried request replays instead of refunding twice.
func (r *Repository) ByIdempotencyKey(merchantID, key string) (*Refund, error) {
	var refund Refund
	if err := r.DB.Where("merchant_id = ? AND idempotency_key = ?", merchantID, key).
		First(&refund).Error; err != nil {
		return nil, err
	}
	return &refund, nil
}

// List returns a page of refunds; the environment filter joins the payment.
func (r *Repository) List(f Filter) ([]Refund, error) {
	q := r.DB.Model(&Refund{}).Where("refunds.merchant_id = ?", f.MerchantID)
	if f.Environment != "" {
		q = q.Joins("JOIN payments ON payments.id = refunds.payment_id").
			Where("payments.environment = ?", f.Environment)
	}
	if f.PaymentID != "" {
		q = q.Where("refunds.payment_id = ?", f.PaymentID)
	}
	if f.Status != "" {
		q = q.Where("refunds.status = ?", f.Status)
	}
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	var rows []Refund
	if err := q.Order("refunds.created_at DESC").Offset(f.Offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// Count returns the number of refunds matching a filter.
func (r *Repository) Count(f Filter) (int64, error) {
	q := r.DB.Model(&Refund{}).Where("merchant_id = ?", f.MerchantID)
	if f.PaymentID != "" {
		q = q.Where("payment_id = ?", f.PaymentID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// RefundedMinor sums the non-failed refunds already recorded for a payment.
func (r *Repository) RefundedMinor(paymentID string) (int64, error) {
	var total int64
	if err := r.DB.Model(&Refund{}).
		Where("payment_id = ? AND status <> ?", paymentID, "failed").
		Select("COALESCE(SUM(amount_minor), 0)").
		Scan(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}
