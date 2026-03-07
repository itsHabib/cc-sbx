package cmd

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/user/skill-sync/internal/provider"
	"github.com/user/skill-sync/internal/sync"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show sync drift between providers",
	Long: `Compares skills in your source provider against all targets and reports
which skills are in sync, modified, missing, or extra.

Exits with code 1 if any drift is detected -- useful for CI checks.`,
	Example: `  # Check drift across all targets
  skill-sync status

  # Check only specific skills
  skill-sync status --skill deploy --skill review

  # Use inline providers (no config file)
  skill-sync status --source claude --targets copilot,gemini`,
	RunE: runStatus,
}

var statusSkills []string

func init() {
	statusCmd.Flags().StringSliceVar(&statusSkills, "skill", nil, "check only named skills (repeatable)")
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	source, targets, err := resolveProviders(Cfg)
	if err != nil {
		return fmt.Errorf("status: %w", err)
	}

	engine := sync.NewDiffEngine(source, targets)
	return doStatus(cmd.OutOrStdout(), engine, statusSkills)
}

func doStatus(w io.Writer, engine *sync.DiffEngine, skillFilter []string) error {
	report, err := engine.Status()
	if err != nil {
		return fmt.Errorf("status: %w", err)
	}

	filterSet := make(map[string]bool, len(skillFilter))
	for _, name := range skillFilter {
		filterSet[name] = true
	}

	hasDrift := false

	for targetName, drifts := range report.Results {
		fmt.Fprintf(w, "Target: %s\n", targetName)
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "SKILL\tSTATUS")

		for _, d := range drifts {
			if len(filterSet) > 0 && !filterSet[d.SkillName] {
				continue
			}
			symbol := statusSymbol(d.Status)
			fmt.Fprintf(tw, "%s\t%s\n", d.SkillName, symbol)
			if d.Status != provider.InSync {
				hasDrift = true
			}
		}

		tw.Flush()
		fmt.Fprintln(w)
	}

	if hasDrift {
		return fmt.Errorf("drift detected")
	}

	return nil
}

func statusSymbol(s provider.SkillStatus) string {
	switch s {
	case provider.InSync:
		return "[ok] in-sync"
	case provider.Modified:
		return "[!] modified"
	case provider.MissingInTarget:
		return "[-] missing"
	case provider.ExtraInTarget:
		return "[+] extra"
	default:
		return "[?] unknown"
	}
}
