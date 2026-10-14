package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/platform/identity"
)

// APILogRow mirrors the api_logs table (see migration 00001).
type APILogRow struct {
	ID            string    `gorm:"primaryKey;size:64"`
	MerchantID    string    `gorm:"column:merchant_id"`
	ApplicationID string    `gorm:"column:application_id"`
	Environment   string    `gorm:"column:environment"`
	Method        string    `gorm:"column:method"`
	Path          string    `gorm:"column:path"`
	StatusCode    int       `gorm:"column:status_code"`
	DurationMs    int       `gorm:"column:duration_ms"`
	SourceIP      string    `gorm:"column:source_ip"`
	RequestID     string    `gorm:"column:request_id"`
	CreatedAt     time.Time `gorm:"column:created_at"`
}

// TableName keeps the row bound to the existing table.
func (APILogRow) TableName() string { return "api_logs" }

// Logging writes one api_logs row per authenticated request.
//
// It records metadata only: method, path, status, duration and identity. Bodies,
// authorization headers and provider payloads are never logged.
func Logging(db *gorm.DB, log *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		// An error returned by a handler is rendered by Fiber's error handler
		// *after* this middleware unwinds, so the response status is still 200
		// here. Derive the real status from the typed error instead.
		status := c.Response().StatusCode()
		if err != nil {
			status = apperrors.From(err).HTTPStatus
		}
		principal := identity.From(c)
		if principal == nil {
			// Only authenticated traffic is persisted: it is the merchant-facing audit trail.
			return err
		}

		row := APILogRow{
			ID:            requestIDOrNew(c),
			MerchantID:    principal.MerchantID,
			ApplicationID: principal.ApplicationID,
			Environment:   principal.Environment,
			Method:        c.Method(),
			Path:          c.Path(),
			StatusCode:    status,
			DurationMs:    int(time.Since(start).Milliseconds()),
			SourceIP:      c.IP(),
			RequestID:     RequestIDValue(c),
			CreatedAt:     time.Now().UTC(),
		}
		if db != nil {
			// Best effort: a logging failure must never fail the request.
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				if writeErr := db.WithContext(ctx).Create(&row).Error; writeErr != nil && log != nil {
					log.Warn("api log write failed", "error", writeErr)
				}
			}()
		}
		return err
	}
}

func requestIDOrNew(c *fiber.Ctx) string {
	if id := RequestIDValue(c); id != "" {
		return id + "_log"
	}
	return gormxNewID()
}
