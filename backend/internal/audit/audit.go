// Package audit records security-relevant events. Modules depend on the
// Recorder interface; the implementation writes to audit_logs.
package audit

import (
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/kuipid01/payconnect/internal/platform/gormx"
)

// Entry is one audit record. Detail must never contain secrets.
type Entry struct {
	MerchantID string
	ActorName  string
	ActorRole  string
	Action     string
	Detail     string
	Severity   string // info | warning | critical
}

// Recorder is the sink every mutating service receives.
type Recorder interface {
	Record(e Entry)
}

// Store persists audit entries.
type Store struct {
	DB  *gorm.DB
	Log *slog.Logger
}

// NewStore builds the audit store.
func NewStore(db *gorm.DB, log *slog.Logger) *Store { return &Store{DB: db, Log: log} }

// Record writes an entry. Audit failures are logged, never propagated: they must
// not roll back the business transaction that triggered them.
func (s *Store) Record(e Entry) {
	if e.Severity == "" {
		e.Severity = "info"
	}
	if s == nil || s.DB == nil {
		return
	}
	row := Log{
		ID:         gormx.NewID("aud"),
		MerchantID: e.MerchantID,
		ActorName:  e.ActorName,
		ActorRole:  e.ActorRole,
		Action:     e.Action,
		Detail:     e.Detail,
		Severity:   e.Severity,
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.DB.Create(&row).Error; err != nil && s.Log != nil {
		s.Log.Error("audit write failed", "action", e.Action, "error", err)
	}
}

// Log is the persisted audit row.
type Log struct {
	ID         string    `gorm:"primaryKey;size:64" json:"id"`
	MerchantID string    `gorm:"column:merchant_id" json:"-"`
	ActorName  string    `gorm:"column:actor_name" json:"actorName"`
	ActorRole  string    `gorm:"column:actor_role" json:"actorRole"`
	Action     string    `json:"action"`
	Detail     string    `json:"detail"`
	Severity   string    `json:"severity"`
	CreatedAt  time.Time `json:"at"`
}

// TableName binds the row to the audit_logs table.
func (Log) TableName() string { return "audit_logs" }
