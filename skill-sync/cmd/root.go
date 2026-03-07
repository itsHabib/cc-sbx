package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/user/skill-sync/internal/config"
	"github.com/user/skill-sync/internal/provider"
)

var (
	cfgPath       string
	Cfg           *config.Config
	InlineSource  string
	InlineTargets []string
)

var rootCmd = &cobra.Command{
	Use:   "skill-sync",
	Short: "Sync AI skills across providers",
	Long: `skill-sync reads skills from a source AI assistant (Claude Code, Copilot,
Gemini CLI, Factory) and syncs them to target providers with format
translation and drift detection.

Configure once with 'skill-sync init', then run 'skill-sync sync' to
keep all your providers in lockstep.`,
	Example: `  # Quick start: init + sync
  skill-sync init --source claude --targets copilot,gemini
  skill-sync sync

  # Check if targets have drifted
  skill-sync status

  # See exactly what changed
  skill-sync diff copilot`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for the init command
		if cmd.Name() == "init" {
			return nil
		}

		// If --source is provided inline, build config from flags
		if InlineSource != "" {
			if len(InlineTargets) == 0 {
				return fmt.Errorf("--targets is required when using --source. Example: --source claude --targets copilot,gemini")
			}
			cfg := &config.Config{
				Source:  InlineSource,
				Targets: InlineTargets,
			}
			if err := cfg.Validate(provider.List()); err != nil {
				return fmt.Errorf("validating config: %w", err)
			}
			Cfg = cfg
			return nil
		}

		// Otherwise load from config file
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		if err := cfg.Validate(provider.List()); err != nil {
			return fmt.Errorf("validating config: %w", err)
		}
		Cfg = cfg
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", ".skill-sync.yaml", "path to config file")
	rootCmd.PersistentFlags().StringVar(&InlineSource, "source", "", "source provider to read skills from (overrides config file)")
	rootCmd.PersistentFlags().StringSliceVar(&InlineTargets, "targets", nil, "target providers to sync skills to (overrides config file)")
}

func Execute() {
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
