package provider

import (
	"fmt"
	"sort"
	"sync"
)

// ProviderFactory creates a Provider with the given base directory.
// If baseDir is empty, the provider uses its default directory.
type ProviderFactory func(baseDir string) Provider

var (
	mu       sync.RWMutex
	registry = make(map[string]ProviderFactory)
)

// Register adds a provider factory to the global registry.
// It panics if a provider with the same name is already registered.
// This is intended to be called from provider packages' init() functions.
func Register(name string, factory ProviderFactory) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("provider: duplicate registration for %q", name))
	}
	registry[name] = factory
}

// New creates a provider by name with an optional base directory override.
// If baseDir is empty, the provider uses its default directory.
func New(name, baseDir string) (Provider, error) {
	mu.RLock()
	defer mu.RUnlock()
	factory, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("provider: unknown provider %q", name)
	}
	return factory(baseDir), nil
}

// Get returns a provider with its default base directory.
func Get(name string) (Provider, error) {
	return New(name, "")
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
	registry = make(map[string]ProviderFactory)
}
