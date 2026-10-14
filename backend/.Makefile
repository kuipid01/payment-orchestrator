run:
	go run ./cmd/api

dev:
	air

test:
	go test ./...

tidy:
	go mod tidy

docker-up:
	docker compose up -d

docker-down:
	docker compose down

# A local receiver to paste into a webhook endpoint's URL while developing.
# See scripts/webhook-sink/README.md.
webhook-sink:
	node scripts/webhook-sink/server.mjs

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	ALLOW_MIGRATION_DOWN=true go run ./cmd/migrate down

migrate-status:
	go run ./cmd/migrate status

migrate-test:
	go test ./internal/database -run "TestVersionedMigrationsBuildFreshDatabase|TestGooseAdoptsLegacyAutoMigrateDatabase" -v
.PHONY: financial-test financial-test-race

# The money-handling domains, run serially: these are the packages where a
# regression costs a merchant real money.
FINANCIAL_PACKAGES = ./internal/payments ./internal/refunds ./internal/transactions ./internal/webhooks ./internal/providers ./internal/routing ./internal/auth

financial-test:
	go test -p 1 $(FINANCIAL_PACKAGES) -v

financial-test-race:
	go test -race -p 1 $(FINANCIAL_PACKAGES)
