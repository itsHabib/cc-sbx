package cmd

import (
	"fmt"
	"strings"

	"github.com/user/skill-sync/internal/config"
	"github.com/user/skill-sync/internal/provider"
)

// resolveProviders creates the source and target providers from config,
// applying any directory overrides from config or CLI flags.
func resolveProviders(cfg *config.Config) (provider.Provider, []provider.Provider, error) {
	available := strings.Join(provider.List(), ", ")

	source, err := provider.New(cfg.Source, cfg.SourceDir)
	if err != nil {
		return nil, nil, fmt.Errorf("unknown source provider %q. Available providers: %s", cfg.Source, available)
	}

	targets := make([]provider.Provider, 0, len(cfg.Targets))
	for _, name := range cfg.Targets {
		dir := ""
		if cfg.TargetDirs != nil {
			dir = cfg.TargetDirs[name]
		}
		t, err := provider.New(name, dir)
		if err != nil {
			return nil, nil, fmt.Errorf("unknown target provider %q. Available providers: %s", name, available)
		}
		targets = append(targets, t)
	}

	return source, targets, nil
}
