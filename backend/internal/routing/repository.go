package routing

import "gorm.io/gorm"

// Repository owns routing-rule persistence.
type Repository struct{ DB *gorm.DB }

// NewRepository builds the routing repository.
func NewRepository(db *gorm.DB) *Repository { return &Repository{DB: db} }

// List returns a merchant's rules in evaluation order.
func (r *Repository) List(merchantID string) ([]Rule, error) {
	var rows []Rule
	if err := r.DB.Where("merchant_id = ?", merchantID).
		Order("priority ASC, created_at ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ByID loads a rule scoped to its merchant.
func (r *Repository) ByID(merchantID, id string) (*Rule, error) {
	var rule Rule
	if err := r.DB.Where("merchant_id = ?", merchantID).First(&rule, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

// Create inserts a rule.
func (r *Repository) Create(rule *Rule) error { return r.DB.Create(rule).Error }

// Save persists a rule.
func (r *Repository) Save(rule *Rule) error { return r.DB.Save(rule).Error }

// Delete removes a rule.
func (r *Repository) Delete(merchantID, id string) (int64, error) {
	res := r.DB.Where("merchant_id = ? AND id = ?", merchantID, id).Delete(&Rule{})
	return res.RowsAffected, res.Error
}
