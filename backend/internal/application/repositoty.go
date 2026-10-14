package application

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// Repository owns application persistence.
type Repository struct{ DB *gorm.DB }

// NewRepository builds the application repository.
func NewRepository(db *gorm.DB) *Repository { return &Repository{DB: db} }

// Create inserts an application.
func (r *Repository) Create(app *Application) error { return r.DB.Create(app).Error }

// Save persists an application update.
func (r *Repository) Save(app *Application) error { return r.DB.Save(app).Error }

// ByID loads an application scoped to its merchant and excludes soft-deleted rows.
func (r *Repository) ByID(merchantID, id string) (*Application, error) {
	var app Application
	if err := r.DB.Where("merchant_id = ? AND deleted_at IS NULL", merchantID).
		First(&app, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &app, nil
}

// List returns applications visible to the caller.
func (r *Repository) List(f ListFilter) ([]Application, error) {
	q := r.DB.Where("merchant_id = ? AND deleted_at IS NULL", f.MerchantID)
	if f.ApplicationID != "" {
		q = q.Where("id = ?", f.ApplicationID)
	}
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	var rows []Application
	if err := q.Order("created_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// SoftDelete marks an application unavailable without destroying payment history.
func (r *Repository) SoftDelete(merchantID, id string, at time.Time) (int64, error) {
	result := r.DB.Model(&Application{}).
		Where("merchant_id = ? AND id = ? AND deleted_at IS NULL", merchantID, id).
		Updates(map[string]any{
			"status":     "paused",
			"deleted_at": at,
			"updated_at": at,
		})
	return result.RowsAffected, result.Error
}

// Stats returns lifetime counters for the supplied applications.
func (r *Repository) Stats(applicationIDs []string) (map[string]Stats, error) {
	stats := make(map[string]Stats, len(applicationIDs))
	if len(applicationIDs) == 0 {
		return stats, nil
	}
	placeholders := strings.Repeat("?,", len(applicationIDs))
	placeholders = strings.TrimSuffix(placeholders, ",")
	args := make([]any, 0, len(applicationIDs))
	for _, id := range applicationIDs {
		args = append(args, id)
	}

	var paymentRows []struct {
		ApplicationID string
		Successful    int64
		Failed        int64
	}
	paymentQuery := "SELECT application_id, COUNT(*) FILTER (WHERE status = 'successful') AS successful, COUNT(*) FILTER (WHERE status IN ('failed', 'cancelled')) AS failed FROM payments WHERE application_id IN (" + placeholders + ") GROUP BY application_id"
	if err := r.DB.Raw(paymentQuery, args...).Scan(&paymentRows).Error; err != nil {
		return nil, err
	}
	for _, row := range paymentRows {
		stats[row.ApplicationID] = Stats{SuccessfulPayments: row.Successful, FailedPayments: row.Failed}
	}

	var logRows []struct {
		ApplicationID string
		APIRRequests  int64
		APIErrors     int64
	}
	logQuery := "SELECT application_id, COUNT(*) AS api_requests, COUNT(*) FILTER (WHERE status_code >= 400) AS api_errors FROM api_logs WHERE application_id IN (" + placeholders + ") GROUP BY application_id"
	if err := r.DB.Raw(logQuery, args...).Scan(&logRows).Error; err != nil {
		return nil, err
	}
	for _, row := range logRows {
		entry := stats[row.ApplicationID]
		entry.APIRRequests = row.APIRRequests
		entry.APIErrors = row.APIErrors
		stats[row.ApplicationID] = entry
	}
	for id, entry := range stats {
		if entry.APIRRequests > 0 {
			entry.SuccessRate = entry.SuccessfulPayments * 100 / entry.APIRRequests
			stats[id] = entry
		}
	}
	return stats, nil
}
