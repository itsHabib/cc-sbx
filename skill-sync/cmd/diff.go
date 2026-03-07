package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/user/skill-sync/internal/sync"
)

var diffCmd = &cobra.Command{
	Use:   "diff [provider]",
	Short: "Show unified diffs for drifted skills",
	Long: `Prints unified diffs for skills that differ between your source and a
target provider. If no provider is specified, shows diffs for all targets.

Like 'git diff' -- informational only, always exits 0.`,
	Example: `  # Show diffs for a specific target
  skill-sync diff copilot

  # Show diffs for only specific skills
  skill-sync diff copilot --skill deploy

  # Show diffs for all targets
  skill-sync diff

  # Use inline providers
  skill-sync diff gemini --source claude --targets gemini`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDiff,
}

var diffSkills []string

func init() {
	diffCmd.Flags().StringSliceVar(&diffSkills, "skill", nil, "show diffs for only named skills (repeatable)")
	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	source, targets, err := resolveProviders(Cfg)
	if err != nil {
		return fmt.Errorf("diff: %w", err)
	}

	engine := sync.NewDiffEngine(source, targets)
	w := cmd.OutOrStdout()

	if len(args) == 1 {
		return doDiff(w, engine, args[0], diffSkills)
	}

	return doDiffAll(w, engine, Cfg.Targets, diffSkills)
}

func doDiff(w io.Writer, engine *sync.DiffEngine, targetName string, skillFilter []string) error {
	result, err := engine.Diff(targetName)
	if err != nil {
		return fmt.Errorf("diff: %w", err)
	}

	filterSet := make(map[string]bool, len(skillFilter))
	for _, name := range skillFilter {
		filterSet[name] = true
	}

	for _, d := range result.Diffs {
		if len(filterSet) > 0 && !filterSet[d.SkillName] {
			continue
		}
		fmt.Fprint(w, d.UnifiedDiff)
	}

	return nil
}

func doDiffAll(w io.Writer, engine *sync.DiffEngine, targetNames []string, skillFilter []string) error {
	for _, name := range targetNames {
		result, err := engine.Diff(name)
		if err != nil {
			return fmt.Errorf("diff: %w", err)
		}

		filterSet := make(map[string]bool, len(skillFilter))
		for _, f := range skillFilter {
			filterSet[f] = true
		}

		for _, d := range result.Diffs {
			if len(filterSet) > 0 && !filterSet[d.SkillName] {
				continue
			}
			fmt.Fprint(w, d.UnifiedDiff)
		}
	}

	return nil
}
