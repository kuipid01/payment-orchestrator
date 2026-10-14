package redis

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

// newRedisStore starts an in-process Redis and returns a store bound to it,
// along with the server so tests can inspect raw keys and advance TTLs.
func newRedisStore(t *testing.T) (*RedisStore, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	store, err := NewRedisStore(Options{
		URL:      "redis://" + mr.Addr(),
		Prefix:   "payconnect:",
		PoolSize: 4,
		Timeout:  time.Second,
	})
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store, mr
}

// TestStoreContract runs the interface contract against both implementations, so
// the Redis store cannot drift from the behaviour the memory store established.
func TestStoreContract(t *testing.T) {
	ctx := context.Background()

	stores := map[string]Store{"memory": NewMemoryStore()}
	redisStore, _ := newRedisStore(t)
	stores["redis"] = redisStore

	for name, store := range stores {
		t.Run(name, func(t *testing.T) {
			// A missing key is not an error.
			if value, ok, err := store.Get(ctx, "absent"); err != nil || ok || value != "" {
				t.Fatalf("Get(absent) = %q, %v, %v; want %q, false, nil", value, ok, err, "")
			}

			// Incr starts at one and counts up.
			for want := int64(1); want <= 3; want++ {
				got, err := store.Incr(ctx, "counter", time.Minute)
				if err != nil {
					t.Fatalf("Incr: %v", err)
				}
				if got != want {
					t.Fatalf("Incr = %d; want %d", got, want)
				}
			}
			if value, ok, err := store.Get(ctx, "counter"); err != nil || !ok || value != "3" {
				t.Fatalf("Get(counter) = %q, %v, %v; want %q, true, nil", value, ok, err, "3")
			}

			// Set/Get round-trips.
			if err := store.Set(ctx, "greeting", "hello", time.Minute); err != nil {
				t.Fatalf("Set: %v", err)
			}
			if value, ok, err := store.Get(ctx, "greeting"); err != nil || !ok || value != "hello" {
				t.Fatalf("Get(greeting) = %q, %v, %v; want %q, true, nil", value, ok, err, "hello")
			}

			// SetNX only writes when absent.
			if created, err := store.SetNX(ctx, "lock", "a", time.Minute); err != nil || !created {
				t.Fatalf("SetNX(first) = %v, %v; want true, nil", created, err)
			}
			if created, err := store.SetNX(ctx, "lock", "b", time.Minute); err != nil || created {
				t.Fatalf("SetNX(second) = %v, %v; want false, nil", created, err)
			}

			// Del removes.
			if err := store.Del(ctx, "greeting"); err != nil {
				t.Fatalf("Del: %v", err)
			}
			if _, ok, err := store.Get(ctx, "greeting"); err != nil || ok {
				t.Fatalf("Get after Del = ok %v, err %v; want false, nil", ok, err)
			}
		})
	}
}

// TestRedisStoreAppliesPrefix proves the namespace is attached centrally: the
// logical key never hits Redis bare.
func TestRedisStoreAppliesPrefix(t *testing.T) {
	ctx := context.Background()
	store, mr := newRedisStore(t)

	if _, err := store.Incr(ctx, "rl:auth:ip_1:100", time.Minute); err != nil {
		t.Fatalf("Incr: %v", err)
	}
	if !mr.Exists("payconnect:rl:auth:ip_1:100") {
		t.Fatal("expected the prefixed key to exist in Redis")
	}
	if mr.Exists("rl:auth:ip_1:100") {
		t.Fatal("key was written without the configured prefix")
	}
}

// TestRedisStoreIncrSetsTTLOnFirstWriteOnly is the contract that matters for
// fixed-window limiters: the window must not slide forward on every request.
func TestRedisStoreIncrSetsTTLOnFirstWriteOnly(t *testing.T) {
	ctx := context.Background()
	store, mr := newRedisStore(t)

	const window = 2 * time.Minute
	if _, err := store.Incr(ctx, "window", window); err != nil {
		t.Fatalf("Incr: %v", err)
	}
	if ttl := mr.TTL("payconnect:window"); ttl != window {
		t.Fatalf("TTL after first write = %v; want %v", ttl, window)
	}

	// Halfway through the window a second request must not extend it.
	mr.FastForward(30 * time.Second)
	if count, err := store.Incr(ctx, "window", window); err != nil || count != 2 {
		t.Fatalf("second Incr = %d, %v; want 2, nil", count, err)
	}
	if ttl := mr.TTL("payconnect:window"); ttl != window-30*time.Second {
		t.Fatalf("TTL after second write = %v; want %v (unchanged)", ttl, window-30*time.Second)
	}

	// Once the window lapses the counter starts over at one.
	mr.FastForward(window)
	if count, err := store.Incr(ctx, "window", window); err != nil || count != 1 {
		t.Fatalf("Incr after expiry = %d, %v; want 1, nil", count, err)
	}
}

// TestRedisStoreIncrIsAtomic fires concurrent increments at one key: each must
// observe a distinct count and the TTL must be applied exactly once.
func TestRedisStoreIncrIsAtomic(t *testing.T) {
	ctx := context.Background()
	store, mr := newRedisStore(t)

	const writers = 50
	const window = time.Minute

	var (
		mu     sync.Mutex
		seen   = make(map[int64]bool)
		wg     sync.WaitGroup
		failed error
	)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			count, err := store.Incr(ctx, "hot", window)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				failed = err
				return
			}
			if seen[count] {
				failed = fmt.Errorf("count %d observed twice: increments are not atomic", count)
			}
			seen[count] = true
		}()
	}
	wg.Wait()

	if failed != nil {
		t.Fatal(failed)
	}
	for want := int64(1); want <= writers; want++ {
		if !seen[want] {
			t.Fatalf("count %d was never returned; got %d distinct values, want %d", want, len(seen), writers)
		}
	}
	if ttl := mr.TTL("payconnect:hot"); ttl != window {
		t.Fatalf("TTL = %v; want %v (set once, not per increment)", ttl, window)
	}
}

// TestRedisStoreGetDistinguishesMissFromError pins the behaviour the readiness
// probe depends on: an empty key is (false, nil), a broken connection is an error.
func TestRedisStoreGetDistinguishesMissFromError(t *testing.T) {
	ctx := context.Background()
	store, mr := newRedisStore(t)

	if _, ok, err := store.Get(ctx, "nothing"); ok || err != nil {
		t.Fatalf("Get(missing) = ok %v, err %v; want false, nil", ok, err)
	}

	if err := store.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}

	mr.Close()
	if _, _, err := store.Get(ctx, "nothing"); err == nil {
		t.Fatal("Get on a closed server returned nil error; want a connection error")
	}
}

// TestNewRedisStoreRejectsBadConfig covers the constructor's guards.
func TestNewRedisStoreRejectsBadConfig(t *testing.T) {
	if _, err := NewRedisStore(Options{}); err == nil {
		t.Fatal("empty URL should be rejected")
	}
	if _, err := NewRedisStore(Options{URL: "not a url"}); err == nil {
		t.Fatal("malformed URL should be rejected")
	}
}

// TestRedisStoreSetWithoutTTLPersists guards the "0 = no expiry" contract that
// Set relies on.
func TestRedisStoreSetWithoutTTLPersists(t *testing.T) {
	ctx := context.Background()
	store, mr := newRedisStore(t)

	if err := store.Set(ctx, "forever", "value", 0); err != nil {
		t.Fatalf("Set: %v", err)
	}
	mr.FastForward(24 * time.Hour)
	if value, ok, err := store.Get(ctx, "forever"); err != nil || !ok || value != "value" {
		t.Fatalf("Get = %q, %v, %v; want the value to survive without a TTL", value, ok, err)
	}
	if ttl := mr.TTL("payconnect:forever"); ttl != 0 {
		t.Fatalf("TTL = %v; want 0 (no expiry)", ttl)
	}
}
