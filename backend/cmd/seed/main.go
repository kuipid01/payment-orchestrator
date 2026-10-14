// Seed command populates the database with demo data matching the dashboard
// mock dataset, so the UI renders with realistic content.
//
//	go run ./cmd/seed
//
// It is idempotent: if the demo owner already exists it does nothing.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/kuipid01/payconnect/internal/api_keys"
	"github.com/kuipid01/payconnect/internal/audit"
	"github.com/kuipid01/payconnect/internal/config"
	"github.com/kuipid01/payconnect/internal/constants"
	"github.com/kuipid01/payconnect/internal/customers"
	"github.com/kuipid01/payconnect/internal/database"
	"github.com/kuipid01/payconnect/internal/merchants"
	"github.com/kuipid01/payconnect/internal/payments"
	"github.com/kuipid01/payconnect/internal/platform/gormx"
	"github.com/kuipid01/payconnect/internal/platform/identity"
	"github.com/kuipid01/payconnect/internal/platform/vault"
	"github.com/kuipid01/payconnect/internal/providers"
	"github.com/kuipid01/payconnect/internal/refunds"
	"github.com/kuipid01/payconnect/internal/routing"
	"github.com/kuipid01/payconnect/internal/transactions"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	db, err := database.Connect(ctx, cfg.Database.DSN(), nil)
	if err != nil {
		log.Fatalf("seed database: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := db.Migrate(ctx); err != nil {
		log.Fatalf("seed migrate: %v", err)
	}

	vlt, err := vault.New(cfg.Security.EncryptionKey)
	if err != nil {
		log.Fatalf("seed vault: %v", err)
	}

	merchantRepo := merchants.NewRepository(db.Gorm)
	if _, err := merchantRepo.UserByEmail("john@acme.dev"); err == nil {
		log.Println("seed: demo data already present, skipping")
		return
	}

	now := time.Now().UTC()
	merchant := &merchants.Merchant{
		ID:              gormx.NewID("mch"),
		BusinessName:    "Acme Inc",
		Email:           "john@acme.dev",
		Country:         "NG",
		Status:          constants.MerchantActive,
		DefaultCurrency: constants.DefaultCurrency,
		CreatedAt:       now.AddDate(0, -1, 0),
		UpdatedAt:       now,
	}
	ownerHash, err := identity.HashPassword(cfg.Security.SeedPassword)
	if err != nil {
		log.Fatalf("seed password: %v", err)
	}
	owner := &merchants.User{
		ID: gormx.NewID("usr"), Name: "John Merchant",
		Email: "john@acme.dev", PasswordHash: ownerHash, EmailVerified: true,
		Status: "active", CreatedAt: merchant.CreatedAt,
	}
	developer := &merchants.User{
		ID: gormx.NewID("usr"), Name: "jane",
		Email: "jane@acme.dev", EmailVerified: true,
		Status: "active", CreatedAt: merchant.CreatedAt,
	}
	if err := merchantRepo.CreateMerchantAndUser(merchant, owner); err != nil {
		log.Fatalf("seed merchant: %v", err)
	}
	if err := merchantRepo.CreateUser(developer); err != nil {
		log.Fatalf("seed developer identity: %v", err)
	}
	if err := merchantRepo.CreateUserMerchant(&merchants.UserMerchant{
		ID: gormx.NewID("um"), UserID: developer.ID, MerchantID: merchant.ID,
		Role: "Developer", Status: "active", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		log.Fatalf("seed developer membership: %v", err)
	}
	if err := merchantRepo.CreateDeveloper(&merchants.Developer{
		ID: gormx.NewID("dev"), UserID: developer.ID, Name: "Jane Developer",
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		log.Fatalf("seed developer account: %v", err)
	}

	seedConnections(db.Gorm, vlt, merchant.ID, cfg, now)
	seedKeys(db.Gorm, vlt, merchant.ID, cfg, owner.Name, now)
	customerIDs := seedCustomers(db.Gorm, merchant.ID, now)
	seedPayments(db.Gorm, merchant.ID, customerIDs, now)
	seedRefund(db.Gorm, merchant.ID, now)
	seedRouting(db.Gorm, merchant.ID, now)
	seedLogs(db.Gorm, merchant.ID, now)

	log.Printf("seed complete — sign in with john@acme.dev / %s", cfg.Security.SeedPassword)
}

func seedConnections(db *gorm.DB, vlt *vault.Vault, merchantID string, cfg config.Config, now time.Time) {
	repo := providers.NewRepository(db)
	for i, kind := range []constants.Provider{constants.ProviderPaystack, constants.ProviderFlutterwave} {
		secret := "sk_test_seed_" + gormx.NewID("") + "9d21"
		encrypted, err := vlt.Encrypt(`{"secret_key":"` + secret + `"}`)
		if err != nil {
			log.Fatalf("seed encrypt: %v", err)
		}
		connection := &providers.Connection{
			ID:                   gormx.NewID("conn"),
			MerchantID:           merchantID,
			Provider:             kind.String(),
			Environment:          constants.EnvironmentTest,
			DisplayName:          kind.String() + " test",
			CredentialsEncrypted: encrypted,
			Status:               constants.ConnectionConnected,
			IsDefault:            i == 0,
			SecretLast4:          secret[len(secret)-4:],
			ConnectedAt:          &now,
			LastVerifiedAt:       &now,
			Currency:             constants.DefaultCurrency,
			CreatedAt:            now.AddDate(0, -1, 0),
			UpdatedAt:            now,
		}
		if err := repo.Create(connection); err != nil {
			log.Fatalf("seed connection %s: %v", kind, err)
		}
	}
}

func seedKeys(db *gorm.DB, vlt *vault.Vault, merchantID string, cfg config.Config, createdBy string, now time.Time) {
	repo := api_keys.NewRepository(db)
	prefix := cfg.Security.APIKeyPrefix
	if prefix == "" {
		prefix = constants.KeyPrefixDefault
	}
	for _, spec := range []struct {
		name      string
		keyType   constants.APIKeyType
		env       string
		createdAt time.Time
	}{
		{"Default secret key", constants.APIKeySecret, constants.EnvironmentTest, now.AddDate(0, -1, 0)},
		{"Webhook signing secret", constants.APIKeyWebhook, constants.EnvironmentTest, now.AddDate(0, -1, 0)},
		{"Public key", constants.APIKeyPublic, constants.EnvironmentTest, now.AddDate(0, -1, 0)},
	} {
		full, masked, err := vault.GenerateKey(prefix, spec.env, constants.KeyTypeMarker(spec.keyType))
		if err != nil {
			log.Fatalf("seed key: %v", err)
		}
		var encrypted []byte
		if vlt != nil && spec.env == constants.EnvironmentTest {
			if enc, encErr := vlt.Encrypt(full); encErr == nil {
				encrypted = enc
			}
		}
		key := &api_keys.Key{
			ID: gormx.NewID("key"), MerchantID: merchantID,
			Environment: spec.env, Type: string(spec.keyType), Prefix: prefix, Name: spec.name,
			KeyHash: vault.HashKey(full), SecretMasked: masked, CreatedBy: createdBy,
			SecretEncrypted: encrypted,
			CreatedAt: spec.createdAt, UpdatedAt: spec.createdAt,
		}
		if err := repo.Create(key); err != nil {
			log.Fatalf("seed key %s: %v", spec.name, err)
		}
	}
}

func seedCustomers(db *gorm.DB, merchantID string, now time.Time) []string {
	repo := customers.NewRepository(db)
	specs := []customers.Customer{
		{Email: "john@example.com", Name: "John Doe", Phone: "+2348012345678"},
		{Email: "jane@example.com", Name: "Jane Doe", Phone: "+2348023456789"},
		{Email: "mike@example.com", Name: "Mike Doe", Phone: "+2348034567890"},
	}
	ids := make([]string, 0, len(specs))
	for i := range specs {
		customer := specs[i]
		customer.ID = gormx.NewID("cus")
		customer.MerchantID = merchantID
		customer.Status = "active"
		customer.CreatedAt = now.AddDate(0, -1, 0)
		customer.UpdatedAt = now
		if err := repo.Create(&customer); err != nil {
			log.Fatalf("seed customer: %v", err)
		}
		ids = append(ids, customer.ID)
	}
	return ids
}

func seedPayments(db *gorm.DB, merchantID string, customerIDs []string, now time.Time) {
	repo := payments.NewRepository(db)
	ledger := transactions.NewService(transactions.NewRepository(db))
	statuses := []string{"successful", "successful", "successful", "pending", "failed"}
	amounts := []int64{2_000_000, 1_500_000, 800_000, 3_200_000, 1_200_000}
	providers := []string{"paystack", "flutterwave"}

	for i := 0; i < 60; i++ {
		status := statuses[i%len(statuses)]
		amount := amounts[i%len(amounts)]
		created := now.AddDate(0, 0, -(i % 29)).Add(-time.Duration(i%17) * time.Hour)
		payment := &payments.Payment{
			ID:          gormx.NewID("pay"),
			MerchantID:  merchantID,
			Environment: constants.EnvironmentTest,
			Reference:   fmt.Sprintf("ref_seed_%04d", i),
			Provider:    providers[i%len(providers)],
			AmountMinor: amount,
			Currency:    constants.DefaultCurrency,
			Status:      status,
			CreatedAt:   created,
			UpdatedAt:   created,
		}
		if customerIDs != nil {
			customerID := customerIDs[i%len(customerIDs)]
			payment.CustomerID = &customerID
		}
		if status == "successful" {
			providerRef := "ps_seed_" + gormx.NewID("")
			payment.ProviderPaymentID = &providerRef
			// Provider fees are a percentage of the amount, in minor units.
			fee := amount * 15 / 1000
			payment.FeeMinor = &fee
		}
		if err := repo.Create(payment); err != nil {
			log.Fatalf("seed payment %d: %v", i, err)
		}
		if err := ledger.RecordPayment(context.Background(), payment); err != nil {
			log.Fatalf("seed ledger %d: %v", i, err)
		}
	}
}

func seedRefund(db *gorm.DB, merchantID string, now time.Time) {
	var payment payments.Payment
	if err := db.Where("merchant_id = ? AND status = 'successful'", merchantID).
		Order("created_at").First(&payment).Error; err != nil {
		return
	}
	providerRefundID := "rf_seed_" + gormx.NewID("")
	refund := &refunds.Refund{
		ID:               refunds.NewID(),
		MerchantID:       merchantID,
		PaymentID:        payment.ID,
		Provider:         payment.Provider,
		ProviderRefundID: &providerRefundID,
		AmountMinor:      payment.AmountMinor / 2,
		Currency:         payment.Currency,
		Status:           string(constants.RefundSuccessful),
		Reason:           "customer_request",
		CreatedAt:        now.Add(-2 * time.Hour),
		UpdatedAt:        now.Add(-2 * time.Hour),
	}
	if err := refunds.NewRepository(db).Create(refund); err != nil {
		log.Fatalf("seed refund: %v", err)
	}
	_ = transactions.NewService(transactions.NewRepository(db)).RecordRefund(&transactions.RefundRecord{
		ID: refund.ID, MerchantID: merchantID, PaymentID: payment.ID, Provider: payment.Provider,
		Environment: payment.Environment, AmountMinor: refund.AmountMinor, Currency: refund.Currency,
		Status: refund.Status, Reference: payment.ID,
	})
}

func seedRouting(db *gorm.DB, merchantID string, now time.Time) {
	rule := &routing.Rule{
		ID:         gormx.NewID("route"),
		MerchantID: merchantID,
		Name:       "NGN primary",
		Currency:   "NGN",
		Country:    "NG",
		Providers: gormx.JSONB[[]routing.Preference]{Data: []routing.Preference{
			{Provider: constants.ProviderPaystack.String(), Priority: 1},
			{Provider: constants.ProviderFlutterwave.String(), Priority: 2},
		}},
		Priority:  1,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := routing.NewRepository(db).Create(rule); err != nil {
		log.Fatalf("seed routing: %v", err)
	}
}

func seedLogs(db *gorm.DB, merchantID string, now time.Time) {
	methods := []string{"POST", "POST", "GET", "POST", "GET"}
	paths := []string{"/v1/payments", "/v1/payments", "/v1/payments/pay_1", "/v1/payments", "/v1/payments/pay_2"}
	codes := []int{200, 200, 200, 500, 200}
	for i := 0; i < 40; i++ {
		requestID := gormx.NewID("req")
		if err := db.Exec(`INSERT INTO api_logs
			(id, merchant_id, environment, method, path, status_code, duration_ms, source_ip, request_id, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			gormx.NewID("log"), merchantID, constants.EnvironmentTest,
			methods[i%len(methods)], paths[i%len(paths)], codes[i%len(codes)],
			40+i%160, fmt.Sprintf("102.89.34.%d", i%250), requestID,
			now.Add(-time.Duration(i)*37*time.Minute)).Error; err != nil {
			log.Fatalf("seed api log: %v", err)
		}
	}

	entries := []audit.Log{
		{ActorName: "John Merchant", ActorRole: "Owner", Action: "Provider connected", Detail: "paystack (test)", Severity: "critical"},
		{ActorName: "John Merchant", ActorRole: "Owner", Action: "Provider connected", Detail: "flutterwave (test)", Severity: "critical"},
		{ActorName: "John Merchant", ActorRole: "Owner", Action: "API key created", Detail: "default secret key", Severity: "critical"},
		{ActorName: "John Merchant", ActorRole: "Owner", Action: "Routing rule created", Detail: "NGN primary", Severity: "warning"},
		{ActorName: "John Merchant", ActorRole: "Owner", Action: "Webhook endpoint created", Detail: "https://example.com/webhooks", Severity: "critical"},
	}
	for i := range entries {
		entry := entries[i]
		entry.ID = gormx.NewID("aud")
		entry.MerchantID = merchantID
		entry.CreatedAt = now.Add(-time.Duration(i+1) * 3 * time.Hour)
		if err := db.Create(&entry).Error; err != nil {
			log.Fatalf("seed audit log: %v", err)
		}
	}
}
