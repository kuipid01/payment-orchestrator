package providers

import (
	"github.com/kuipid01/payconnect/internal/constants"
	apperrors "github.com/kuipid01/payconnect/internal/errors"
)

// Registry resolves a provider name to its adapter.
type Registry interface {
	// Get returns the adapter for a concrete provider.
	Get(provider constants.Provider) (Adapter, error)
	// List returns every registered adapter (used by discovery endpoints).
	List() []Adapter
}

// MapRegistry is the default in-process registry.
type MapRegistry struct {
	adapters map[constants.Provider]Adapter
	order    []constants.Provider
}

// NewRegistry builds a registry from adapters.
func NewRegistry(adapters ...Adapter) *MapRegistry {
	registry := &MapRegistry{adapters: make(map[constants.Provider]Adapter, len(adapters))}
	for _, adapter := range adapters {
		if adapter == nil {
			continue
		}
		registry.adapters[adapter.Name()] = adapter
		registry.order = append(registry.order, adapter.Name())
	}
	return registry
}

// Get resolves an adapter or returns a typed error.
func (r *MapRegistry) Get(provider constants.Provider) (Adapter, error) {
	adapter, ok := r.adapters[provider]
	if !ok {
		return nil, apperrors.ProviderNotSupported(provider.String())
	}
	return adapter, nil
}

// List returns every registered adapter.
func (r *MapRegistry) List() []Adapter {
	out := make([]Adapter, 0, len(r.order))
	for _, provider := range r.order {
		out = append(out, r.adapters[provider])
	}
	return out
}

var _ Registry = (*MapRegistry)(nil)
