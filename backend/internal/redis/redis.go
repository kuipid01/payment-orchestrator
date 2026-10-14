// Package redis provides the shared key-value store used for rate limiting,
// idempotency locks and short-lived caches.
//
// The Store interface is deliberately small so the in-process implementation
// can be swapped for a real Redis client without touching call sites. When
// REDIS_URL is set, build the client-backed store; otherwise the memory store
// is used (correct for single-instance deployments, not for a fleet).
package redis

import (
	"context"
	"sync"
	"time"
)

// Store is the minimal key-value contract the application depends on.
type Store interface {
	// Incr atomically increments key, setting ttl on first write.
	Incr(ctx context.Context, key string, ttl time.Duration) (int64, error)
	// Get returns the raw value and whether it existed.
	Get(ctx context.Context, key string) (string, bool, error)
	// Set stores a value with a TTL (0 = no expiry).
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	// SetNX stores a value only when the key is absent.
	SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
	// Del removes a key.
	Del(ctx context.Context, key string) error
	// Close releases any resources held by the store.
	Close() error
}

type entry struct {
	value   string
	expires time.Time
}

// MemoryStore is a mutex-guarded TTL map. Fine for tests and single-node dev.
type MemoryStore struct {
	mu     sync.Mutex
	items  map[string]entry
	closed bool
}

// NewMemoryStore builds an empty in-process store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{items: make(map[string]entry)}
}

func (m *MemoryStore) expired(e entry, now time.Time) bool {
	return !e.expires.IsZero() && now.After(e.expires)
}

func (m *MemoryStore) Incr(_ context.Context, key string, ttl time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	cur := m.items[key]
	if cur.value == "" || m.expired(cur, now) {
		var expires time.Time
		if ttl > 0 {
			expires = now.Add(ttl)
		}
		m.items[key] = entry{value: "1", expires: expires}
		return 1, nil
	}
	value := parseInt(cur.value) + 1
	cur.value = itoa(value)
	m.items[key] = cur
	return value, nil
}

func (m *MemoryStore) Get(_ context.Context, key string) (string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.items[key]
	if !ok || m.expired(e, time.Now()) {
		delete(m.items, key)
		return "", false, nil
	}
	return e.value, true, nil
}

func (m *MemoryStore) Set(_ context.Context, key, value string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var expires time.Time
	if ttl > 0 {
		expires = time.Now().Add(ttl)
	}
	m.items[key] = entry{value: value, expires: expires}
	return nil
}

func (m *MemoryStore) SetNX(_ context.Context, key, value string, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	if e, ok := m.items[key]; ok && !m.expired(e, now) {
		return false, nil
	}
	var expires time.Time
	if ttl > 0 {
		expires = now.Add(ttl)
	}
	m.items[key] = entry{value: value, expires: expires}
	return true, nil
}

func (m *MemoryStore) Del(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, key)
	return nil
}

func (m *MemoryStore) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = make(map[string]entry)
	m.closed = true
	return nil
}

func parseInt(v string) int64 {
	var n int64
	for _, ch := range v {
		if ch < '0' || ch > '9' {
			return n
		}
		n = n*10 + int64(ch-'0')
	}
	return n
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// interface guard
var _ Store = (*MemoryStore)(nil)
