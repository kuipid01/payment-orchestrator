package providers

import (
	"time"

	"gorm.io/gorm"
)

// Repository owns provider-connection persistence.
type Repository struct{ DB *gorm.DB }

// NewRepository builds the provider repository.
func NewRepository(db *gorm.DB) *Repository { return &Repository{DB: db} }

// List returns a merchant's connections.
func (r *Repository) List(merchantID string) ([]Connection, error) {
	var rows []Connection
	if err := r.DB.Where("merchant_id = ?", merchantID).Order("created_at").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ByID loads a connection scoped to its merchant.
func (r *Repository) ByID(merchantID, id string) (*Connection, error) {
	var c Connection
	if err := r.DB.Where("merchant_id = ?", merchantID).First(&c, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// ByIDGlobal loads a connection by id regardless of merchant. Only the provider
// webhook gateway uses it: the connection id in the webhook URL already
// identifies the merchant, and the signature is verified before anything else.
func (r *Repository) ByIDGlobal(id string) (*Connection, error) {
	var c Connection
	if err := r.DB.First(&c, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// ByProviderEnv loads the connection for a provider in one environment.
func (r *Repository) ByProviderEnv(merchantID, provider, environment string) (*Connection, error) {
	var c Connection
	if err := r.DB.Where("merchant_id = ? AND provider = ? AND environment = ?",
		merchantID, provider, environment).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// Create inserts a connection.
func (r *Repository) Create(c *Connection) error { return r.DB.Create(c).Error }

// Save persists changes to a connection.
func (r *Repository) Save(c *Connection) error { return r.DB.Save(c).Error }

// Delete removes a connection row.
func (r *Repository) Delete(merchantID, id string) error {
	return r.DB.Where("merchant_id = ? AND id = ?", merchantID, id).Delete(&Connection{}).Error
}

// TouchWebhookReceived stamps the last inbound webhook time on a connection,
// regardless of merchant. Only the webhook gateway calls it, after a signature
// check has already authenticated the caller.
func (r *Repository) TouchWebhookReceived(id string, at time.Time) error {
	return r.DB.Model(&Connection{}).Where("id = ?", id).
		UpdateColumn("webhook_last_event_at", at).Error
}

// ClearDefault unsets the default flag for a provider environment.
func (r *Repository) ClearDefault(merchantID, environment string) error {
	return r.DB.Model(&Connection{}).
		Where("merchant_id = ? AND environment = ?", merchantID, environment).
		Update("is_default", false).Error
}
