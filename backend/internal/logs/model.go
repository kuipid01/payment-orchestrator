// Package logs serves the observability surfaces: developer API request logs and
// the security audit trail. Both are read-only projections of tables written by
// other packages.
//
// Request logs hold metadata only — method, path, status, duration and identity.
// Bodies, authorization headers, provider credentials and customer data are
// never persisted here.
package logs

import "time"

// APILog mirrors the api_logs table (written by the logging middleware).
type APILog struct {
	ID            string    `gorm:"primaryKey;size:64" json:"id"`
	MerchantID    string    `gorm:"column:merchant_id" json:"-"`
	ApplicationID string    `gorm:"column:application_id" json:"application_id,omitempty"`
	Environment   string    `gorm:"column:environment" json:"environment,omitempty"`
	Method        string    `gorm:"column:method" json:"method"`
	Path          string    `gorm:"column:path" json:"path"`
	StatusCode    int       `gorm:"column:status_code" json:"status_code"`
	DurationMs    int       `gorm:"column:duration_ms" json:"duration_ms"`
	SourceIP      string    `gorm:"column:source_ip" json:"source_ip,omitempty"`
	RequestID     string    `gorm:"column:request_id" json:"request_id,omitempty"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
}

// TableName binds the model to the api_logs table.
func (APILog) TableName() string { return "api_logs" }
