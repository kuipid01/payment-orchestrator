package developer

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/kuipid01/payconnect/internal/database"
	"github.com/kuipid01/payconnect/internal/merchants"
	"github.com/kuipid01/payconnect/internal/platform/gormx"
)

func TestListIncludesAcceptedDeveloper(t *testing.T) {
	dsn := os.Getenv("PAYCONNECT_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set PAYCONNECT_TEST_DATABASE_URL to run the Postgres integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := database.Connect(ctx, dsn, nil)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	defer db.Close()
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	tx := db.Gorm.Begin()
	if tx.Error != nil {
		t.Fatalf("begin test transaction: %v", tx.Error)
	}
	t.Cleanup(func() { tx.Rollback() })

	repo := merchants.NewRepository(tx)
	now := time.Now().UTC()
	ownerID := gormx.NewID("usr")
	userID := gormx.NewID("usr")
	developerID := gormx.NewID("dev")
	merchantID := gormx.NewID("mch")
	accessID := gormx.NewID("dma")

	user := &merchants.User{
		ID: userID, Name: "Invited Developer", Email: userID + "@example.test",
		PasswordHash: "test-hash", Status: "active", CreatedAt: now,
	}
	if err := repo.CreateUser(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	owner := &merchants.User{
		ID: ownerID, Name: "Merchant Owner", Email: ownerID + "@example.test",
		PasswordHash: "test-hash", Status: "active", CreatedAt: now,
	}
	if err := repo.CreateUser(owner); err != nil {
		t.Fatalf("create owner: %v", err)
	}
	if err := repo.CreateMerchant(&merchants.Merchant{
		ID: merchantID, OwnerUserID: ownerID, BusinessName: "Test Merchant",
		Email: "merchant-" + merchantID + "@example.test", Country: "NG",
		Status: "active", DefaultCurrency: "NGN", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create merchant: %v", err)
	}
	if err := repo.CreateDeveloper(&merchants.Developer{
		ID: developerID, UserID: userID, Name: user.Name, Status: "active",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create developer profile: %v", err)
	}
	if err := repo.CreateDeveloperMerchantAccess(&merchants.DeveloperMerchantAccess{
		ID: accessID, DeveloperID: &developerID, MerchantID: merchantID,
		InviteeEmail: user.Email, Status: "active",
		Permissions:         gormx.JSONB[[]string]{Data: []string{"payments.view"}},
		InvitedBy:           ownerID,
		InvitationTokenHash: accessID,
		AcceptedAt:          &now,
		CreatedAt:           now,
		UpdatedAt:           now,
	}); err != nil {
		t.Fatalf("create accepted access: %v", err)
	}
	pendingAccessID := gormx.NewID("inv")
	if err := repo.CreateDeveloperMerchantAccess(&merchants.DeveloperMerchantAccess{
		ID:                  pendingAccessID,
		MerchantID:          merchantID,
		InviteeEmail:        "another-" + userID + "@example.test",
		Status:              "pending",
		Permissions:         gormx.JSONB[[]string]{Data: []string{"payments.view"}},
		InvitedBy:           userID,
		InvitationTokenHash: pendingAccessID,
		CreatedAt:           now,
		UpdatedAt:           now,
	}); err != nil {
		t.Fatalf("create developer's own invitation: %v", err)
	}

	service := NewService(repo, nil, "", nil)
	listed, err := service.List(merchantID, ownerID, "Owner")
	if err != nil {
		t.Fatalf("list developers: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("owner listed %d developer grants, want 2", len(listed))
	}
	if listed[0].ID != pendingAccessID && listed[1].ID != pendingAccessID {
		t.Fatalf("owner list does not include pending invite %q: %+v", pendingAccessID, listed)
	}

	var accepted *DeveloperResponse
	for i := range listed {
		if listed[i].AccessID == accessID {
			accepted = &listed[i]
			break
		}
	}
	if accepted == nil {
		t.Fatalf("owner list does not include accepted developer grant %q: %+v", accessID, listed)
	}
	if accepted.ID != developerID || accepted.DeveloperID != developerID {
		t.Fatalf("accepted developer identifiers = (%q, %q), want (%q, %q)",
			accepted.ID, accepted.DeveloperID, developerID, developerID)
	}
	if accepted.Name != user.Name || accepted.Email != user.Email || accepted.Status != "active" {
		t.Fatalf("listed developer = %+v, want active %q <%s>", accepted, user.Name, user.Email)
	}

	listed, err = service.List(merchantID, userID, "Developer")
	if err != nil {
		t.Fatalf("list developer's own invitations: %v", err)
	}
	if len(listed) != 1 || listed[0].AccessID != pendingAccessID {
		t.Fatalf("developer listed grants %+v, want only own pending grant %q", listed, pendingAccessID)
	}
}
