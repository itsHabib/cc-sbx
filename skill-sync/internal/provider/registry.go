package provider

import (
	"fmt"
	"sort"
	"sync"
)

var (
	mu       sync.RWMutex
	registry = make(map[string]Provider)
)

// Register adds a provider to the global registry.
// It panics if a provider with the same name is already registered.
// This is intended to be called from provider packages' init() functions.
func Register(p Provider) {
	mu.Lock()
	defer mu.Unlock()
	name := p.Name()
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("provider: duplicate registration for %q", name))
	}
	registry[name] = p
}

// Get returns the provider registered under the given name.
// Returns an error if no provider is registered with that name.
func Get(name string) (Provider, error) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("provider: unknown provider %q", name)
	}
	return p, nil
}

// List returns the names of all registered providers, sorted alphabetically.
func List() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// resetRegistry clears all registered providers. Used only in tests.
func resetRegistry() {
	mu.Lock()
	defer mu.Unlock()
	registry = make(map[string]Provider)
}
