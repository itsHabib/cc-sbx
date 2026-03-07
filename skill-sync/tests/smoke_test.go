//go:build smoke

package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/user/skill-sync/internal/provider"
	"github.com/user/skill-sync/internal/sync"
)

// fixtureNames lists the testdata skill filenames (without extension).
var fixtureNames = []string{"simple", "deploy", "search"}

// copyFixtures copies testdata/*.md files into the given directory as <name>/SKILL.md.
func copyFixtures(t *testing.T, dstDir string) {
	t.Helper()
	for _, name := range fixtureNames {
		src := filepath.Join("testdata", name+".md")
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("read fixture %s: %v", name, err)
		}
		skillDir := filepath.Join(dstDir, name)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatalf("create skill dir %s: %v", name, err)
		}
		dst := filepath.Join(skillDir, "SKILL.md")
		if err := os.WriteFile(dst, data, 0644); err != nil {
			t.Fatalf("write fixture %s: %v", name, err)
		}
	}
}

// setupProviders creates temp directories and returns a Claude source provider
// plus Copilot and Gemini target providers.
func setupProviders(t *testing.T) (provider.Provider, []provider.Provider, string, string) {
	t.Helper()

	claudeDir := filepath.Join(t.TempDir(), "claude-skills")
	copilotDir := filepath.Join(t.TempDir(), "copilot-prompts")
	geminiDir := filepath.Join(t.TempDir(), "gemini-commands")

	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("create claude dir: %v", err)
	}

	source := provider.NewClaudeProvider(provider.WithBaseDir(claudeDir))
	targets := []provider.Provider{
		provider.NewCopilotProvider(provider.WithCopilotBaseDir(copilotDir)),
		provider.NewGeminiProvider(provider.WithGeminiBaseDir(geminiDir)),
	}

	return source, targets, copilotDir, geminiDir
}

func TestSmoke_FullFlow(t *testing.T) {
	// -- Phase 1: Setup --
	source, targets, copilotDir, geminiDir := setupProviders(t)
	copyFixtures(t, source.SkillDir())

	// -- Phase 2: Sync all skills --
	syncEng := sync.NewSyncEngine(source, targets)
	result, err := syncEng.Sync(nil, true)
	if err != nil {
		t.Fatalf("Phase 2: Sync failed: %v", err)
	}

	if result.TotalErrored != 0 {
		for _, d := range result.Details {
			if d.Error != nil {
				t.Logf("sync error: %s -> %s: %v", d.SkillName, d.Target, d.Error)
			}
		}
		t.Fatalf("Phase 2: expected 0 errors, got %d", result.TotalErrored)
	}

	// 3 skills x 2 targets = 6
	if result.TotalSynced != 6 {
		t.Fatalf("Phase 2: expected 6 synced, got %d", result.TotalSynced)
	}

	// Verify Copilot files exist
	for _, name := range fixtureNames {
		path := filepath.Join(copilotDir, name+".prompt.md")
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("Phase 2: Copilot file missing: %s", path)
		}
	}

	// Verify Gemini files exist
	for _, name := range fixtureNames {
		path := filepath.Join(geminiDir, name+".toml")
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("Phase 2: Gemini file missing: %s", path)
		}
	}

	// -- Phase 3: Status (all in-sync) --
	diffEng := sync.NewDiffEngine(source, targets)
	report, err := diffEng.Status()
	if err != nil {
		t.Fatalf("Phase 3: Status failed: %v", err)
	}

	for targetName, drifts := range report.Results {
		for _, d := range drifts {
			if d.Status != provider.InSync {
				t.Errorf("Phase 3: expected InSync for %s/%s, got %s",
					targetName, d.SkillName, d.Status)
			}
		}
	}

	// -- Phase 4: Introduce drift --
	driftFile := filepath.Join(copilotDir, "deploy.prompt.md")
	if err := os.WriteFile(driftFile, []byte("# MODIFIED deploy\nThis has been changed."), 0644); err != nil {
		t.Fatalf("Phase 4: write drift file: %v", err)
	}

	report, err = diffEng.Status()
	if err != nil {
		t.Fatalf("Phase 4: Status failed: %v", err)
	}

	copilotDrifts := report.Results["copilot"]
	var modifiedCount, inSyncCount int
	for _, d := range copilotDrifts {
		switch d.Status {
		case provider.Modified:
			modifiedCount++
			if d.SkillName != "deploy" {
				t.Errorf("Phase 4: expected 'deploy' to be Modified, got %s", d.SkillName)
			}
		case provider.InSync:
			inSyncCount++
		default:
			t.Errorf("Phase 4: unexpected status %s for copilot/%s", d.Status, d.SkillName)
		}
	}
	if modifiedCount != 1 {
		t.Errorf("Phase 4: expected 1 Modified in copilot, got %d", modifiedCount)
	}
	if inSyncCount != 2 {
		t.Errorf("Phase 4: expected 2 InSync in copilot, got %d", inSyncCount)
	}

	// Gemini should still be all in-sync
	geminiDrifts := report.Results["gemini"]
	for _, d := range geminiDrifts {
		if d.Status != provider.InSync {
			t.Errorf("Phase 4: expected Gemini InSync for %s, got %s", d.SkillName, d.Status)
		}
	}

	// -- Phase 5: Diff --
	detailed, err := diffEng.Diff("copilot")
	if err != nil {
		t.Fatalf("Phase 5: Diff failed: %v", err)
	}

	if len(detailed.Diffs) != 1 {
		t.Fatalf("Phase 5: expected 1 diff entry, got %d", len(detailed.Diffs))
	}

	d := detailed.Diffs[0]
	if d.SkillName != "deploy" {
		t.Errorf("Phase 5: expected diff for 'deploy', got %s", d.SkillName)
	}
	if d.UnifiedDiff == "" {
		t.Error("Phase 5: UnifiedDiff is empty")
	}
	if !strings.Contains(d.UnifiedDiff, "---") || !strings.Contains(d.UnifiedDiff, "+++") {
		t.Error("Phase 5: UnifiedDiff missing --- / +++ headers")
	}
}

func TestSmoke_SkillFilter(t *testing.T) {
	source, targets, copilotDir, geminiDir := setupProviders(t)
	copyFixtures(t, source.SkillDir())

	syncEng := sync.NewSyncEngine(source, targets)
	result, err := syncEng.Sync([]string{"deploy"}, true)
	if err != nil {
		t.Fatalf("Sync with filter failed: %v", err)
	}

	if result.TotalErrored != 0 {
		t.Fatalf("expected 0 errors, got %d", result.TotalErrored)
	}

	// 1 skill x 2 targets = 2
	if result.TotalSynced != 2 {
		t.Fatalf("expected 2 synced with filter, got %d", result.TotalSynced)
	}

	// "deploy" should exist in both targets
	if _, err := os.Stat(filepath.Join(copilotDir, "deploy.prompt.md")); err != nil {
		t.Error("deploy.prompt.md should exist in copilot dir")
	}
	if _, err := os.Stat(filepath.Join(geminiDir, "deploy.toml")); err != nil {
		t.Error("deploy.toml should exist in gemini dir")
	}

	// "simple" and "search" should NOT exist in targets
	for _, name := range []string{"simple", "search"} {
		if _, err := os.Stat(filepath.Join(copilotDir, name+".prompt.md")); err == nil {
			t.Errorf("%s.prompt.md should NOT exist in copilot dir", name)
		}
		if _, err := os.Stat(filepath.Join(geminiDir, name+".toml")); err == nil {
			t.Errorf("%s.toml should NOT exist in gemini dir", name)
		}
	}
}
