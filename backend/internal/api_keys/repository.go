package api_keys

import (
	"time"

	"gorm.io/gorm"
)

// Repository owns API key persistence.
type Repository struct{ DB *gorm.DB }

// NewRepository builds the API key repository.
func NewRepository(db *gorm.DB) *Repository { return &Repository{DB: db} }

// ListFilter narrows a key listing.
type ListFilter struct {
	MerchantID  string
	Type        string
	Environment string
	// DeveloperAccessID, when set, restricts the listing to keys bound to one
	// developer grant. "general" narrows it to team-shared keys instead.
	DeveloperAccessID string
	Limit             int
}

// Create inserts a key.
func (r *Repository) Create(k *Key) error { return r.DB.Create(k).Error }

// ByID loads a key scoped to its merchant.
func (r *Repository) ByID(merchantID, id string) (*Key, error) {
	var k Key
	if err := r.DB.Where("merchant_id = ?", merchantID).First(&k, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &k, nil
}

// ByHash loads a key by its secret hash (authentication path).
func (r *Repository) ByHash(hash string) (*Key, error) {
	var k Key
	if err := r.DB.First(&k, "key_hash = ?", hash).Error; err != nil {
		return nil, err
	}
	return &k, nil
}

// List returns a merchant's keys, newest first.
func (r *Repository) List(f ListFilter) ([]Key, error) {
	q := r.DB.Where("merchant_id = ?", f.MerchantID)
	if f.Type != "" {
		q = q.Where("type = ?", f.Type)
	}
	if f.Environment != "" {
		q = q.Where("environment = ?", f.Environment)
	}
	switch f.DeveloperAccessID {
	case "":
	case ScopeGeneral:
		q = q.Where("developer_access_id IS NULL")
	default:
		q = q.Where("developer_access_id = ?", f.DeveloperAccessID)
	}
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var rows []Key
	if err := q.Order("created_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// Revoke marks a key revoked.
func (r *Repository) Revoke(merchantID, id string, at time.Time) (int64, error) {
	res := r.DB.Model(&Key{}).
		Where("merchant_id = ? AND id = ? AND revoked_at IS NULL", merchantID, id).
		Updates(map[string]any{"revoked_at": at, "updated_at": at})
	return res.RowsAffected, res.Error
}

// DeleteRevoked permanently removes a revoked key, scoped to its merchant.
func (r *Repository) DeleteRevoked(merchantID, id string) (int64, error) {
	res := r.DB.Where("merchant_id = ? AND id = ? AND revoked_at IS NOT NULL", merchantID, id).
		Delete(&Key{})
	return res.RowsAffected, res.Error
}

// RevokeByApplication revokes every active key in one application scope.
func (r *Repository) RevokeByApplication(merchantID, applicationID string, at time.Time) (int64, error) {
	res := r.DB.Model(&Key{}).
		Where("merchant_id = ? AND application_id = ? AND revoked_at IS NULL", merchantID, applicationID).
		Updates(map[string]any{"revoked_at": at, "updated_at": at})
	return res.RowsAffected, res.Error
}

// RevokeByDeveloperAccess revokes every active key bound to one developer
// grant, leaving the application's shared keys untouched.
func (r *Repository) RevokeByDeveloperAccess(merchantID, accessID string, at time.Time) (int64, error) {
	res := r.DB.Model(&Key{}).
		Where("merchant_id = ? AND developer_access_id = ? AND revoked_at IS NULL", merchantID, accessID).
		Updates(map[string]any{"revoked_at": at, "updated_at": at})
	return res.RowsAffected, res.Error
}

// TouchLastUsed records key usage (best effort on the hot path).
func (r *Repository) TouchLastUsed(id string, at time.Time) error {
	return r.DB.Model(&Key{}).Where("id = ?", id).Update("last_used_at", at).Error
}
