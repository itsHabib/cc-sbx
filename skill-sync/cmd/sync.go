package cmd

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/user/skill-sync/internal/provider"
	"github.com/user/skill-sync/internal/sync"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync skills from source to all targets",
	Long: `Reads skills from your source provider, translates the format, and writes
them to every target provider. Use --dry-run to preview without writing.

Exits with code 1 if any skill fails to sync.`,
	Example: `  # Sync all skills to all targets
  skill-sync sync

  # Preview what would be synced
  skill-sync sync --dry-run

  # Sync only specific skills
  skill-sync sync --skill deploy --skill review

  # Sync without a config file
  skill-sync sync --source claude --targets copilot,gemini`,
	RunE: runSync,
}

var (
	syncDryRun bool
	syncSkills []string
	syncForce  bool
)

func init() {
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "preview sync without writing to targets")
	syncCmd.Flags().StringSliceVar(&syncSkills, "skill", nil, "sync only named skills (repeatable)")
	syncCmd.Flags().BoolVar(&syncForce, "force", false, "overwrite existing skills in targets (default: skip existing)")
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	source, targets, err := resolveProviders(Cfg)
	if err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	w := cmd.OutOrStdout()

	if syncDryRun {
		return doSyncDryRun(w, source, targets, syncSkills)
	}

	engine := sync.NewSyncEngine(source, targets)
	if err := doSync(w, engine, syncSkills, syncForce); err != nil {
		return err
	}

	// Print source and target locations
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  source:  %s (%s)\n", source.Name(), source.SkillDir())
	for _, t := range targets {
		fmt.Fprintf(w, "  target:  %s (%s)\n", t.Name(), t.SkillDir())
	}

	return nil
}

func doSyncDryRun(w io.Writer, source provider.Provider, targets []provider.Provider, skillFilter []string) error {
	skills, err := source.ListSkills()
	if err != nil {
		return fmt.Errorf("sync: list source skills: %w", err)
	}

	if len(skillFilter) > 0 {
		filterSet := make(map[string]bool, len(skillFilter))
		for _, name := range skillFilter {
			filterSet[name] = true
		}
		var filtered []provider.Skill
		for _, s := range skills {
			if filterSet[s.Name] {
				filtered = append(filtered, s)
			}
		}
		skills = filtered
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SKILL\tTARGET\tSTATUS")
	for _, skill := range skills {
		for _, target := range targets {
			fmt.Fprintf(tw, "%s\t%s\t%s\n", skill.Name, target.Name(), "would sync")
		}
	}
	tw.Flush()

	if len(skills) > 0 {
		fmt.Fprintf(w, "\nWould sync: %d skill(s) to %d target(s)\n", len(skills), len(targets))
	}

	return nil
}

func doSync(w io.Writer, engine *sync.SyncEngine, skillFilter []string, force bool) error {
	result, err := engine.Sync(skillFilter, force)
	if err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SKILL\tTARGET\tSTATUS")
	for _, d := range result.Details {
		status := "synced"
		if d.Status == sync.SyncError {
			status = fmt.Sprintf("error: %v", d.Error)
		} else if d.Status == sync.SyncSkipped {
			status = "skipped (exists)"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", d.SkillName, d.Target, status)
	}
	tw.Flush()

	fmt.Fprintf(w, "\nSynced: %d  Skipped: %d  Errors: %d\n", result.TotalSynced, result.TotalSkipped, result.TotalErrored)

	if result.TotalErrored > 0 {
		return fmt.Errorf("sync completed with %d error(s)", result.TotalErrored)
	}

	return nil
}
