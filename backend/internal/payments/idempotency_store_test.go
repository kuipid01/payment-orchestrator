package payments

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/platform/gormx"
	"github.com/kuipid01/payconnect/internal/redis"
)

func ptrInt(v int) *int { return &v }

func newCacheService() *Service { return &Service{Store: redis.NewMemoryStore()} }

// TestIdempotencyCacheRoundTripsSuccess proves a completed payment can be
// replayed from the shared store without consulting the database.
func TestIdempotencyCacheRoundTripsSuccess(t *testing.T) {
	ctx := context.Background()
	svc := newCacheService()

	svc.rememberIdempotency(ctx, &IdempotencyRecord{
		MerchantID:     "mch_1",
		IdempotencyKey: "key-1",
		RequestHash:    "hash-1",
		StatusCode:     ptrInt(201),
		ResponseBody:   gormx.JSONB[map[string]any]{Data: map[string]any{"id": "pay_1", "status": "pending"}},
	})

	cached, ok := svc.cachedIdempotency(ctx, "mch_1", "key-1")
	if !ok {
		t.Fatal("expected a cache hit")
	}
	if cached.RequestHash != "hash-1" {
		t.Fatalf("RequestHash = %q; want hash-1", cached.RequestHash)
	}
	if appErr := storedError(cached); appErr != nil {
		t.Fatalf("storedError = %v; want nil for a success record", appErr)
	}
	response := replayResponse(cached)
	if response == nil || response.ID != "pay_1" {
		t.Fatalf("replayResponse = %+v; want payment pay_1", response)
	}
}

// TestIdempotencyCacheReplaysStoredError proves a failed request replays the
// same typed error, not a success body.
func TestIdempotencyCacheReplaysStoredError(t *testing.T) {
	ctx := context.Background()
	svc := newCacheService()

	svc.rememberIdempotency(ctx, &IdempotencyRecord{
		MerchantID:     "mch_1",
		IdempotencyKey: "key-err",
		RequestHash:    "hash-err",
		StatusCode:     ptrInt(402),
		ResponseBody: gormx.JSONB[map[string]any]{Data: map[string]any{
			"error": map[string]any{
				"type":    "api_error",
				"code":    "provider_declined",
				"message": "Insufficient funds.",
			},
		}},
	})

	cached, ok := svc.cachedIdempotency(ctx, "mch_1", "key-err")
	if !ok {
		t.Fatal("expected a cache hit")
	}
	stored := storedError(cached)
	if stored == nil {
		t.Fatal("storedError = nil; want the stored error")
	}
	if !apperrors.Is(stored, "provider_declined") {
		t.Fatalf("storedError code = %q; want provider_declined", apperrors.From(stored).Code)
	}
	if status := apperrors.From(stored).HTTPStatus; status != 402 {
		t.Fatalf("storedError status = %d; want 402", status)
	}
}

// TestIdempotencyCacheMissAndNilStore covers the two ways the pre-check declines:
// an empty cache and no store at all.
func TestIdempotencyCacheMissAndNilStore(t *testing.T) {
	ctx := context.Background()

	svc := newCacheService()
	if _, ok := svc.cachedIdempotency(ctx, "mch_1", "unknown"); ok {
		t.Fatal("expected a cache miss on an empty store")
	}

	noStore := &Service{}
	if _, ok := noStore.cachedIdempotency(ctx, "mch_1", "key"); ok {
		t.Fatal("expected no cache hit without a store")
	}
}

// TestIdempotencyCacheFallsBackWhenStoreErrors proves a broken store is treated
// as a miss rather than an error, so idempotency keeps working via the database.
func TestIdempotencyCacheFallsBackWhenStoreErrors(t *testing.T) {
	ctx := context.Background()
	svc := &Service{Store: failingStore{}}

	if _, ok := svc.cachedIdempotency(ctx, "mch_1", "key"); ok {
		t.Fatal("a store error must not be reported as a cache hit")
	}
	// Writes are best effort: they must not panic or block the caller.
	svc.rememberIdempotency(ctx, &IdempotencyRecord{
		MerchantID: "mch_1", IdempotencyKey: "key", ResponseBody: gormx.JSONB[map[string]any]{Data: map[string]any{}},
	})
}

// TestIdempotencyLockAdmitsOneHolder proves the lock is mutually exclusive and
// namespaced apart from the cache entry.
func TestIdempotencyLockAdmitsOneHolder(t *testing.T) {
	ctx := context.Background()
	svc := newCacheService()

	token, locked, usable := svc.lockIdempotency(ctx, "mch_1", "key")
	if !usable || !locked {
		t.Fatalf("first lock = locked %v, usable %v; want true, true", locked, usable)
	}
	if token == "" {
		t.Fatal("expected a non-empty lock token")
	}

	if _, locked, usable := svc.lockIdempotency(ctx, "mch_1", "key"); !usable || locked {
		t.Fatalf("second lock = locked %v, usable %v; want false, true", locked, usable)
	}

	// The lock and the cache live under distinct keys.
	if _, ok, _ := svc.Store.Get(ctx, idempotencyLockKey("mch_1", "key")); !ok {
		t.Fatal("expected the lock key to exist")
	}
	if _, ok, _ := svc.Store.Get(ctx, idempotencyCacheKey("mch_1", "key")); ok {
		t.Fatal("the lock must not occupy the cache key")
	}
}

// TestIdempotencyUnlockReleasesOwnLock proves a completed request frees the key.
func TestIdempotencyUnlockReleasesOwnLock(t *testing.T) {
	ctx := context.Background()
	svc := newCacheService()

	token, _, _ := svc.lockIdempotency(ctx, "mch_1", "key")
	svc.unlockIdempotency(ctx, "mch_1", "key", token)

	if _, locked, _ := svc.lockIdempotency(ctx, "mch_1", "key"); !locked {
		t.Fatal("expected the key to be lockable again after release")
	}
}

// TestIdempotencyUnlockIgnoresForeignToken proves a stale holder cannot release
// a lock that a newer request now owns.
func TestIdempotencyUnlockIgnoresForeignToken(t *testing.T) {
	ctx := context.Background()
	svc := newCacheService()

	if _, locked, usable := svc.lockIdempotency(ctx, "mch_1", "key"); !usable || !locked {
		t.Fatal("expected to hold the lock before testing a foreign release")
	}
	svc.unlockIdempotency(ctx, "mch_1", "key", "some-other-token")

	if _, locked, _ := svc.lockIdempotency(ctx, "mch_1", "key"); locked {
		t.Fatal("a foreign token must not release the lock")
	}
}

// TestIdempotencyLockFallsBackWhenStoreErrors proves an unreachable store
// disables the lock rather than the endpoint, leaving the database in charge.
func TestIdempotencyLockFallsBackWhenStoreErrors(t *testing.T) {
	ctx := context.Background()
	svc := &Service{Store: failingStore{}}

	if _, locked, usable := svc.lockIdempotency(ctx, "mch_1", "key"); usable || locked {
		t.Fatalf("lockIdempotency = locked %v, usable %v; want false, false", locked, usable)
	}
}

// TestCreateHashIsStableWhenReferenceIsOmitted pins the fix that lets a retry
// without an explicit reference be recognised: the server still mints a
// reference for the payment, but the idempotency hash must use the client's
// (empty) reference so repeated attempts hash the same.
func TestCreateHashIsStableWhenReferenceIsOmitted(t *testing.T) {
	request := CreateRequest{
		AmountMinor: 50_000,
		Currency:    "NGN",
		Customer:    CustomerInput{Email: "payer@example.com"},
	}

	first, amount, currency, mintedFirst, err := normalizeCreate(CreateInput{Request: request})
	if err != nil {
		t.Fatalf("normalizeCreate: %v", err)
	}
	second, _, _, mintedSecond, err := normalizeCreate(CreateInput{Request: request})
	if err != nil {
		t.Fatalf("normalizeCreate: %v", err)
	}
	// The server mints a fresh reference each time...
	if mintedFirst == mintedSecond {
		t.Fatal("expected distinct server-minted references")
	}
	// ...but the client supplied none, so the request must still carry none.
	if first.Reference != "" {
		t.Fatal("normalizeCreate must not write the minted reference back onto the request")
	}
	hashFirst := hashRequest(amount, currency, strings.TrimSpace(first.Reference), first)
	hashSecond := hashRequest(amount, currency, strings.TrimSpace(second.Reference), second)
	if hashFirst != hashSecond {
		t.Fatal("the idempotency hash must be stable across retries that omit the reference")
	}
}

// failingStore is a redis.Store that is always unavailable.
type failingStore struct{}

var errStoreDown = errors.New("store down")

func (failingStore) Incr(context.Context, string, time.Duration) (int64, error) {
	return 0, errStoreDown
}
func (failingStore) Get(context.Context, string) (string, bool, error) {
	return "", false, errStoreDown
}
func (failingStore) Set(context.Context, string, string, time.Duration) error {
	return errStoreDown
}
func (failingStore) SetNX(context.Context, string, string, time.Duration) (bool, error) {
	return false, errStoreDown
}
func (failingStore) Del(context.Context, string) error { return errStoreDown }
func (failingStore) Close() error                      { return nil }

var _ redis.Store = failingStore{}
