package logs

import (
	"gorm.io/gorm"

	"github.com/kuipid01/payconnect/internal/audit"
)

// Repository owns read access to the log projections.
type Repository struct{ DB *gorm.DB }

// NewRepository builds the logs repository.
func NewRepository(db *gorm.DB) *Repository { return &Repository{DB: db} }

// Filter narrows an API log listing.
type Filter struct {
	MerchantID    string
	ApplicationID string
	Environment   string
	Method        string
	StatusClass   string // 2xx | 4xx | 5xx
	RequestID     string
	Offset        int
	Limit         int
}

// APILogs returns a page of request logs.
func (r *Repository) APILogs(f Filter) ([]APILog, error) {
	q := r.DB.Where("merchant_id = ?", f.MerchantID)
	if f.ApplicationID != "" && f.ApplicationID != "all" {
		q = q.Where("application_id = ?", f.ApplicationID)
	}
	if f.Environment != "" {
		q = q.Where("environment = ?", f.Environment)
	}
	if f.Method != "" {
		q = q.Where("method = ?", f.Method)
	}
	if f.RequestID != "" {
		q = q.Where("request_id = ?", f.RequestID)
	}
	switch f.StatusClass {
	case "2xx":
		q = q.Where("status_code >= 200 AND status_code < 300")
	case "4xx":
		q = q.Where("status_code >= 400 AND status_code < 500")
	case "5xx":
		q = q.Where("status_code >= 500")
	}
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	var rows []APILog
	if err := q.Order("created_at DESC").Offset(f.Offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// CountAPILogs returns the number of request logs matching a filter.
func (r *Repository) CountAPILogs(f Filter) (int64, error) {
	q := r.DB.Model(&APILog{}).Where("merchant_id = ?", f.MerchantID)
	if f.Environment != "" {
		q = q.Where("environment = ?", f.Environment)
	}
	if f.ApplicationID != "" && f.ApplicationID != "all" {
		q = q.Where("application_id = ?", f.ApplicationID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// APILogByID loads one request log scoped to its merchant.
func (r *Repository) APILogByID(merchantID, id string) (*APILog, error) {
	var row APILog
	if err := r.DB.Where("merchant_id = ?", merchantID).First(&row, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// AuditLogs returns a page of the security audit trail.
func (r *Repository) AuditLogs(merchantID, severity string, offset, limit int) ([]audit.Log, error) {
	q := auditQuery(r.DB, merchantID, severity)
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var rows []audit.Log
	if err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// CountAuditLogs returns the number of matching audit events.
func (r *Repository) CountAuditLogs(merchantID, severity string) (int64, error) {
	var total int64
	err := auditQuery(r.DB.Model(&audit.Log{}), merchantID, severity).Count(&total).Error
	return total, err
}

// AllAuditLogs returns matching records for CSV export.
func (r *Repository) AllAuditLogs(merchantID, severity string) ([]audit.Log, error) {
	var rows []audit.Log
	err := auditQuery(r.DB, merchantID, severity).Order("created_at DESC").Find(&rows).Error
	return rows, err
}

func auditQuery(db *gorm.DB, merchantID, severity string) *gorm.DB {
	q := db.Where("merchant_id = ?", merchantID)
	if severity != "" {
		q = q.Where("severity = ?", severity)
	}
	return q
}
