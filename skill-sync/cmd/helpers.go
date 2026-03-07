package cmd

import (
	"fmt"
	"strings"

	"github.com/user/skill-sync/internal/config"
	"github.com/user/skill-sync/internal/provider"
)

// resolveProviders resolves the source and target providers from config using the registry.
func resolveProviders(cfg *config.Config) (provider.Provider, []provider.Provider, error) {
	available := strings.Join(provider.List(), ", ")

	source, err := provider.Get(cfg.Source)
	if err != nil {
		return nil, nil, fmt.Errorf("unknown source provider %q. Available providers: %s", cfg.Source, available)
	}

	targets := make([]provider.Provider, 0, len(cfg.Targets))
	for _, name := range cfg.Targets {
		t, err := provider.Get(name)
		if err != nil {
			return nil, nil, fmt.Errorf("unknown target provider %q. Available providers: %s", name, available)
		}
		targets = append(targets, t)
	}

	return source, targets, nil
}
