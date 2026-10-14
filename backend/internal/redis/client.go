package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Options configure the Redis-backed Store. They mirror config.RedisConfig but
// are declared here so this package stays independent of the config package and
// can be constructed directly in tests.
type Options struct {
	// URL is a redis:// or rediss:// connection string. Required.
	URL string
	// Prefix namespaces every key so one Redis instance can back several
	// deployments without collisions.
	Prefix string
	// PoolSize caps concurrent connections; zero leaves the client default.
	PoolSize int
	// Timeout bounds dial, read and write operations; zero leaves the default.
	Timeout time.Duration
}

// RedisStore is a Store backed by a real Redis server. Unlike MemoryStore its
// state survives restarts and is shared by every replica, which is what makes
// rate limits and lockouts hold across a fleet.
type RedisStore struct {
	client *goredis.Client
	prefix string
}

// incrScript increments a key and applies the TTL only on the first write.
//
// INCR followed by a separate EXPIRE is not equivalent: under concurrency a
// second caller can INCR between the two commands and the window would be
// extended on every request instead of once. Doing both in one script keeps the
// "ttl on first write" contract atomic.
var incrScript = goredis.NewScript(`
local current = redis.call('INCR', KEYS[1])
if current == 1 then
  local ttl = tonumber(ARGV[1])
  if ttl and ttl > 0 then
    redis.call('PEXPIRE', KEYS[1], ttl)
  end
end
return current
`)

// NewRedisStore parses the URL and builds a client. It does not dial: call Ping
// to confirm the server is actually reachable before relying on it.
func NewRedisStore(opts Options) (*RedisStore, error) {
	if opts.URL == "" {
		return nil, fmt.Errorf("redis: URL must not be empty")
	}
	parsed, err := goredis.ParseURL(opts.URL)
	if err != nil {
		return nil, fmt.Errorf("redis: parse url: %w", err)
	}
	if opts.PoolSize > 0 {
		parsed.PoolSize = opts.PoolSize
	}
	if opts.Timeout > 0 {
		parsed.DialTimeout = opts.Timeout
		parsed.ReadTimeout = opts.Timeout
		parsed.WriteTimeout = opts.Timeout
	}
	return &RedisStore{client: goredis.NewClient(parsed), prefix: opts.Prefix}, nil
}

// Ping verifies the server answers. Startup uses it so a configured but
// unreachable Redis fails loudly instead of silently degrading.
func (s *RedisStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

// key namespaces the caller's logical key under the configured prefix. The
// prefix is applied here, in one place, so no call site has to remember it.
func (s *RedisStore) key(key string) string {
	if s.prefix == "" {
		return key
	}
	return s.prefix + key
}

// Incr atomically increments key, setting ttl on the first write.
func (s *RedisStore) Incr(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	count, err := incrScript.Run(ctx, s.client, []string{s.key(key)}, ttl.Milliseconds()).Int64()
	if err != nil {
		return 0, err
	}
	return count, nil
}

// Get returns the raw value and whether it existed. A missing key is reported as
// ("" , false, nil): redis.Nil is an expected outcome, not an error, so the
// readiness probe and the lockout check never treat an empty cache as a failure.
func (s *RedisStore) Get(ctx context.Context, key string) (string, bool, error) {
	value, err := s.client.Get(ctx, s.key(key)).Result()
	switch {
	case err == goredis.Nil:
		return "", false, nil
	case err != nil:
		return "", false, err
	default:
		return value, true, nil
	}
}

// Set stores a value with a TTL (0 = no expiry).
func (s *RedisStore) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return s.client.Set(ctx, s.key(key), value, ttl).Err()
}

// SetNX stores a value only when the key is absent.
func (s *RedisStore) SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	return s.client.SetNX(ctx, s.key(key), value, ttl).Result()
}

// Del removes a key.
func (s *RedisStore) Del(ctx context.Context, key string) error {
	return s.client.Del(ctx, s.key(key)).Err()
}

// Close releases the connection pool.
func (s *RedisStore) Close() error {
	return s.client.Close()
}

// interface guard
var _ Store = (*RedisStore)(nil)
