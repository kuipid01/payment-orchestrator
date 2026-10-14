// Package server owns application composition: dependency wiring, the HTTP app
// and its lifecycle. cmd/api is intentionally tiny because everything else lives
// here.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/kuipid01/payconnect/internal/config"
	"github.com/kuipid01/payconnect/internal/database"
	"github.com/kuipid01/payconnect/internal/observability"
	"github.com/kuipid01/payconnect/internal/redis"
)

// Server bundles the dependency graph, the Fiber app and the background workers.
type Server struct {
	app  *fiber.App
	deps *Dependencies
	log  *slog.Logger
	addr string
}

// NewServer builds the whole application from configuration.
func NewServer(ctx context.Context, cfg config.Config) (*Server, error) {
	logger := observability.NewLoggerWithLevel(cfg.App.Environment, cfg.App.Name, cfg.Log.Level)
	observability.SetDefault(logger)

	db, err := database.Connect(ctx, cfg.Database.DSN(), logger)
	if err != nil {
		return nil, err
	}
	if err := db.Migrate(ctx); err != nil {
		return nil, err
	}

	// The shared store backs rate limiting, lockouts and readiness. A configured
	// Redis is shared across replicas; without REDIS_URL the in-process store is
	// used, which is only correct for a single instance.
	store, err := newStore(ctx, cfg, logger)
	if err != nil {
		return nil, err
	}

	deps, err := Build(cfg, db, store, logger)
	if err != nil {
		return nil, err
	}

	app := NewApp(deps)
	return &Server{
		app:  app,
		deps: deps,
		log:  logger,
		addr: ":" + cfg.Server.Port,
	}, nil
}

// newStore builds the shared key-value store from configuration.
//
// With REDIS_URL set the Redis-backed store is used and must answer a PING: a
// production deployment that cannot reach its store must fail to boot rather
// than silently fall back to per-instance state, which would disable rate limits
// fleet-wide. Outside production an unreachable Redis degrades to the in-process
// store so local development is not blocked by a missing dependency.
func newStore(ctx context.Context, cfg config.Config, logger *slog.Logger) (redis.Store, error) {
	if !cfg.Redis.Enabled() {
		logger.Info("REDIS_URL is unset; using the in-process store (single-instance only)")
		return redis.NewMemoryStore(), nil
	}
	store, err := redis.NewRedisStore(redis.Options{
		URL:      cfg.Redis.URL,
		Prefix:   cfg.Redis.Prefix,
		PoolSize: cfg.Redis.PoolSize,
		Timeout:  cfg.Redis.Timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("redis: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, cfg.Redis.Timeout)
	defer cancel()
	if err := store.Ping(pingCtx); err != nil {
		_ = store.Close()
		if cfg.IsProduction() {
			return nil, fmt.Errorf("redis: configured but unreachable: %w", err)
		}
		logger.Warn("REDIS_URL is set but unreachable; falling back to the in-process store", "error", err)
		return redis.NewMemoryStore(), nil
	}
	logger.Info("using the Redis-backed store", "prefix", cfg.Redis.Prefix, "pool_size", cfg.Redis.PoolSize)
	return store, nil
}

// Start begins listening and blocks until the context is cancelled, then shuts
// the server down gracefully.
func (s *Server) Start(ctx context.Context) error {
	cfg := s.deps.Cfg
	s.deps.Dispatcher.Start(ctx, 30*time.Second)

	// Fiber's timeouts are configured at the listener level.
	s.app.Server().ReadTimeout = cfg.Server.ReadTimeout
	s.app.Server().WriteTimeout = cfg.Server.WriteTimeout
	s.app.Server().IdleTimeout = cfg.Server.IdleTimeout

	listenErr := make(chan error, 1)
	go func() {
		s.log.Info("payconnect api listening", "addr", s.addr, "env", cfg.App.Environment)
		listenErr <- s.app.Listen(s.addr)
	}()

	select {
	case err := <-listenErr:
		if err != nil {
			return fmt.Errorf("server: listen: %w", err)
		}
		return nil
	case <-ctx.Done():
		s.log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := s.app.ShutdownWithContext(shutdownCtx); err != nil {
			s.log.Error("graceful shutdown failed", "error", err)
		}
		if closer, ok := s.deps.Store.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
		if err := s.deps.DB.Close(); err != nil {
			s.log.Error("database close failed", "error", err)
		}
		return nil
	}
}

// Run is the convenience entrypoint used by cmd/api: it installs the signal
// handler and starts the server.
func Run(cfg config.Config) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server, err := NewServer(ctx, cfg)
	if err != nil {
		return err
	}
	return server.Start(ctx)
}
