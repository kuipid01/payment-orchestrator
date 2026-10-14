// Package database opens the single Postgres handle used by every repository
// and applies the versioned goose migrations. Nothing else in the application
// opens a connection or mutates the schema.
package database

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/kuipid01/payconnect/internal/database/migrations"
)

// DB wraps the GORM handle so callers do not construct their own.
type DB struct {
	Gorm *gorm.DB
	log  *slog.Logger
}

// Connect opens the connection pool and verifies it is reachable.
func Connect(ctx context.Context, dsn string, log *slog.Logger) (*DB, error) {
	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                 sqlLogger(),
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return nil, fmt.Errorf("database: open: %w", err)
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("database: handle: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("database: ping: %w", err)
	}
	if log != nil {
		log.Info("database connected")
	}
	return &DB{Gorm: gormDB, log: log}, nil
}

// sqlLogger builds the GORM logger. Only warnings and slower-than-a-second
// queries are surfaced, and a missing row is not one of them: a lookup that
// finds nothing is a normal outcome, and the caller already logs it with the
// context ("webhook rejected: connection not found"). Left at the default, GORM
// prints every ErrRecordNotFound as an error line with no service or env
// attributes, which reads as a failure nobody can act on.
func sqlLogger() gormlogger.Interface { return newSQLLogger(os.Stdout) }

// newSQLLogger builds that logger over an explicit writer so tests can assert
// on what it does and does not emit.
func newSQLLogger(w io.Writer) gormlogger.Interface {
	return gormlogger.New(log.New(w, "", log.LstdFlags), gormlogger.Config{
		SlowThreshold:             time.Second,
		LogLevel:                  gormlogger.Warn,
		IgnoreRecordNotFoundError: true,
		Colorful:                  false,
	})
}

// Migrate applies every embedded goose migration. Schema changes only ever
// happen here — never through automatic model sync.
func (d *DB) Migrate(ctx context.Context) error {
	sqlDB, err := d.Gorm.DB()
	if err != nil {
		return fmt.Errorf("database: handle: %w", err)
	}
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("database: goose dialect: %w", err)
	}
	if err := goose.UpContext(ctx, sqlDB, "sql"); err != nil {
		return fmt.Errorf("database: goose up: %w", err)
	}
	if d.log != nil {
		d.log.Info("database migrations applied")
	}
	return nil
}

// MigrateDown rolls every migration back one step.
func (d *DB) MigrateDown(ctx context.Context) error {
	sqlDB, err := d.Gorm.DB()
	if err != nil {
		return fmt.Errorf("database: handle: %w", err)
	}
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("database: goose dialect: %w", err)
	}
	return goose.DownContext(ctx, sqlDB, "sql")
}

// MigrationStatus reports the current migration version.
func (d *DB) MigrationStatus(ctx context.Context) error {
	sqlDB, err := d.Gorm.DB()
	if err != nil {
		return fmt.Errorf("database: handle: %w", err)
	}
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("database: goose dialect: %w", err)
	}
	return goose.StatusContext(ctx, sqlDB, "sql")
}

// Ping reports whether the database is reachable (used by readiness checks).
func (d *DB) Ping(ctx context.Context) error {
	if d == nil || d.Gorm == nil {
		return fmt.Errorf("database: not initialised")
	}
	sqlDB, err := d.Gorm.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Close releases the pool.
func (d *DB) Close() error {
	if d == nil || d.Gorm == nil {
		return nil
	}
	sqlDB, err := d.Gorm.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
