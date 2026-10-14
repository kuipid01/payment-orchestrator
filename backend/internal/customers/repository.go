package customers

import (
	"time"

	"gorm.io/gorm"
)

// Repository owns customer persistence.
type Repository struct{ DB *gorm.DB }

// NewRepository builds the customer repository.
func NewRepository(db *gorm.DB) *Repository { return &Repository{DB: db} }

// ListFilter narrows a customer listing.
type ListFilter struct {
	MerchantID string
	Email      string
	Search     string
	Offset     int
	Limit      int
}

// Create inserts a customer.
func (r *Repository) Create(c *Customer) error { return r.DB.Create(c).Error }

// Save persists a customer.
func (r *Repository) Save(c *Customer) error { return r.DB.Save(c).Error }

// ByID loads a live customer scoped to its merchant.
func (r *Repository) ByID(merchantID, id string) (*Customer, error) {
	var c Customer
	if err := r.DB.Where("merchant_id = ? AND deleted_at IS NULL", merchantID).
		First(&c, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// Refs loads several live customers in one query. ids that no longer exist are
// simply absent from the result: a payment keeps its history either way.
func (r *Repository) Refs(merchantID string, ids []string) ([]Customer, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []Customer
	if err := r.DB.Where("merchant_id = ? AND deleted_at IS NULL AND id IN ?", merchantID, ids).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ByEmail loads a live customer by email (case-insensitive).
func (r *Repository) ByEmail(merchantID, email string) (*Customer, error) {
	var c Customer
	if err := r.DB.Where("merchant_id = ? AND deleted_at IS NULL AND lower(email) = lower(?)",
		merchantID, email).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// List returns a page of customers.
func (r *Repository) List(f ListFilter) ([]Customer, error) {
	q := r.DB.Where("merchant_id = ? AND deleted_at IS NULL", f.MerchantID)
	if f.Email != "" {
		q = q.Where("lower(email) = lower(?)", f.Email)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("lower(email) LIKE lower(?) OR lower(name) LIKE lower(?)", like, like)
	}
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	var rows []Customer
	if err := q.Order("created_at DESC").Offset(f.Offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// Count returns the number of live customers.
func (r *Repository) Count(merchantID string) (int64, error) {
	var total int64
	if err := r.DB.Model(&Customer{}).
		Where("merchant_id = ? AND deleted_at IS NULL", merchantID).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// SoftDelete marks a customer deleted without losing payment history.
func (r *Repository) SoftDelete(merchantID, id string, at time.Time) (int64, error) {
	res := r.DB.Model(&Customer{}).
		Where("merchant_id = ? AND id = ? AND deleted_at IS NULL", merchantID, id).
		Updates(map[string]any{"deleted_at": at, "updated_at": at})
	return res.RowsAffected, res.Error
}
