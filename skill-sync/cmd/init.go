package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/user/skill-sync/internal/config"
	"github.com/user/skill-sync/internal/provider"
	"gopkg.in/yaml.v3"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a .skill-sync.yaml config file",
	Long: `Generates a .skill-sync.yaml in the current directory declaring which
provider is your source of truth and which providers to sync to.

Run this once per project. Requires --source and --targets flags.`,
	Example: `  # Initialize with Claude as source, sync to Copilot and Gemini
  skill-sync init --source claude --targets copilot,gemini

  # Initialize with all targets
  skill-sync init --source claude --targets copilot,gemini,factory

  # Use a custom config path
  skill-sync init --source claude --targets copilot --config my-config.yaml`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	if InlineSource == "" {
		return fmt.Errorf("--source is required. Specify your source provider: skill-sync init --source claude --targets copilot,gemini")
	}
	if len(InlineTargets) == 0 {
		return fmt.Errorf("--targets is required. Specify one or more target providers: --targets copilot,gemini,factory")
	}

	// Check if config already exists
	if _, err := os.Stat(cfgPath); err == nil {
		return fmt.Errorf("%s already exists; remove it first or use a different --config path", cfgPath)
	}

	// Validate provider names against the registry
	registered := provider.List()
	cfg := &config.Config{
		Source:  InlineSource,
		Targets: InlineTargets,
		Skills:  []string{},
	}
	if err := cfg.Validate(registered); err != nil {
		return fmt.Errorf("init: %w", err)
	}

	// Marshal and write config
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("init: marshaling config: %w", err)
	}
	if err := os.WriteFile(cfgPath, data, 0644); err != nil {
		return fmt.Errorf("init: writing config: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Created %s (source: %s, targets: %v)\n", cfgPath, cfg.Source, cfg.Targets)
	return nil
}
