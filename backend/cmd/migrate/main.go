// Migration command: applies, rolls back or reports the goose schema version.
//
//	go run ./cmd/migrate up      # apply every pending migration
//	go run ./cmd/migrate down    # roll back one migration
//	go run ./cmd/migrate status  # report the current version
//
// Rolling back requires ALLOW_MIGRATION_DOWN=true, so a stray invocation in a
// deployed environment cannot undo schema changes by accident.
package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/kuipid01/payconnect/internal/config"
	"github.com/kuipid01/payconnect/internal/database"
	"github.com/kuipid01/payconnect/internal/observability"
)

func main() {
	command := "up"
	if len(os.Args) > 1 {
		command = strings.ToLower(strings.TrimSpace(os.Args[1]))
	}

	cfg := config.Load()
	logger := observability.NewLogger(cfg.App.Environment, cfg.App.Name)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := database.Connect(ctx, cfg.Database.DSN(), logger)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer func() { _ = db.Close() }()

	switch command {
	case "up":
		err = db.Migrate(ctx)
	case "down":
		if !strings.EqualFold(os.Getenv("ALLOW_MIGRATION_DOWN"), "true") {
			log.Fatal("refusing to roll back: set ALLOW_MIGRATION_DOWN=true to confirm")
		}
		err = db.MigrateDown(ctx)
	case "status":
		err = db.MigrationStatus(ctx)
	default:
		log.Fatalf("unknown command %q: expected up, down or status", command)
	}
	if err != nil {
		log.Fatalf("migrate %s: %v", command, err)
	}
	log.Printf("migrate %s: ok", command)
}
