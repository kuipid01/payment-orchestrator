// Package application owns the merchant-facing product integration.
package application

import "time"

// Application is a product surface that owns API keys, payments, webhooks and logs.
type Application struct {
	ID          string     `gorm:"primaryKey;size:64" json:"id"`
	MerchantID  string     `gorm:"column:merchant_id;size:64" json:"merchant_id"`
	Name        string     `gorm:"column:name" json:"name"`
	Description string     `gorm:"column:description" json:"description"`
	Environment string     `gorm:"column:environment" json:"-"`
	Status      string     `gorm:"column:status" json:"status"`
	LastUsedAt  *time.Time `gorm:"column:last_used_at" json:"last_used_at,omitempty"`
	CreatedAt   time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt   *time.Time `gorm:"column:deleted_at" json:"-"`
}

// TableName binds the model to the applications table.
func (Application) TableName() string { return "applications" }

// Stats contains lifetime usage data exposed by the application list and detail pages.
type Stats struct {
	APIRRequests       int64 `json:"apiRequests"`
	SuccessfulPayments int64 `json:"successfulPayments"`
	FailedPayments     int64 `json:"failedPayments"`
	APIErrors          int64 `json:"apiErrors"`
	SuccessRate        int64 `json:"successRate"`
}

// ApplicationResponse is the public representation returned by the API.
type ApplicationResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	LastUsedAt  *string `json:"lastUsedAt,omitempty"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
	Stats       Stats   `json:"stats"`
}

// ToResponse maps an application and its lifetime statistics to the public DTO.
func ToResponse(app *Application, stats Stats) ApplicationResponse {
	return ApplicationResponse{
		ID:          app.ID,
		Name:        app.Name,
		Description: app.Description,
		Status:      app.Status,
		LastUsedAt:  formatTime(app.LastUsedAt),
		CreatedAt:   app.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   app.UpdatedAt.UTC().Format(time.RFC3339),
		Stats:       stats,
	}
}

// CreatedResponse includes the keys minted with the application, exactly once.
// APIKey mirrors the test key for clients that predate the key pair.
type CreatedResponse struct {
	Application ApplicationResponse  `json:"application"`
	APIKey      string               `json:"api_key"`
	Keys        []CreatedKeyResponse `json:"keys"`
}

// CreatedKeyResponse is one key returned at creation, with its secret.
type CreatedKeyResponse struct {
	ID          string `json:"id"`
	Environment string `json:"environment"`
	Scope       string `json:"scope"`
	Secret      string `json:"secret"`
}

func formatTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}
