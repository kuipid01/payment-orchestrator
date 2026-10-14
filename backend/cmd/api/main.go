// PayConnect API server entrypoint.
//
// Intentionally small: load and validate configuration, then hand control to the
// server package, which builds every dependency explicitly and owns the
// lifecycle.
//
//	LoadConfig → ValidateConfig → InitializeLogger → InitializeDatabase →
//	InitializeStore → InitializeRepositories → InitializeProviderRegistry →
//	InitializeServices → InitializeHandlers → RegisterMiddleware →
//	RegisterRoutes → StartServer
package main

import (
	"log"

	"github.com/kuipid01/payconnect/internal/config"
	"github.com/kuipid01/payconnect/internal/server"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("configuration invalid: %v", err)
	}
	if err := server.Run(cfg); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
