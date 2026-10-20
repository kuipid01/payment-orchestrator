package factory

import (
	"testing"

	"github.com/kuipid01/payconnect/internal/constants"
)

// The factory is the only place that knows every adapter, so this is the test
// that fails if a provider is implemented but never wired.
func TestRegistryResolvesEveryAdvertisedProvider(t *testing.T) {
	registry := NewRegistry(Config{})

	for _, provider := range constants.AllProviders() {
		adapter, err := registry.Get(provider)
		if err != nil {
			t.Fatalf("Get(%s) = %v", provider, err)
		}
		if adapter.Name() != provider {
			t.Fatalf("Get(%s) returned adapter for %s", provider, adapter.Name())
		}
	}
	if got := len(registry.List()); got != len(constants.AllProviders()) {
		t.Fatalf("registry has %d adapters, want %d", got, len(constants.AllProviders()))
	}
}

func TestRegistryRejectsAutoAndUnknownProviders(t *testing.T) {
	registry := NewRegistry(Config{})
	for _, provider := range []constants.Provider{constants.ProviderAuto, "not-a-provider"} {
		if _, err := registry.Get(provider); err == nil {
			t.Fatalf("Get(%s) succeeded, want an error", provider)
		}
	}
}
