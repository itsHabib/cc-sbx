package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".skill-sync.yaml")
	content := `source: claude
targets:
  - copilot
  - gemini
skills:
  - deploy
  - test
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Source != "claude" {
		t.Errorf("Source = %q, want %q", cfg.Source, "claude")
	}
	if len(cfg.Targets) != 2 || cfg.Targets[0] != "copilot" || cfg.Targets[1] != "gemini" {
		t.Errorf("Targets = %v, want [copilot gemini]", cfg.Targets)
	}
	if len(cfg.Skills) != 2 || cfg.Skills[0] != "deploy" || cfg.Skills[1] != "test" {
		t.Errorf("Skills = %v, want [deploy test]", cfg.Skills)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/.skill-sync.yaml")
	if err == nil {
		t.Fatal("Load() expected error for missing file")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected os.ErrNotExist, got: %v", err)
	}
}

func TestLoad_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".skill-sync.yaml")
	if err := os.WriteFile(path, []byte(":\n  :\n    - [invalid"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() expected error for malformed YAML")
	}
}

func TestLoad_EmptySkills(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".skill-sync.yaml")
	content := `source: claude
targets:
  - copilot
skills: []
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Skills == nil {
		t.Error("Skills should be empty slice, got nil")
	}
	if len(cfg.Skills) != 0 {
		t.Errorf("Skills = %v, want empty", cfg.Skills)
	}
}

func TestValidate_ValidNames(t *testing.T) {
	cfg := &Config{
		Source:  "claude",
		Targets: []string{"copilot", "gemini"},
	}
	err := cfg.Validate([]string{"claude", "copilot", "gemini"})
	if err != nil {
		t.Errorf("Validate() unexpected error: %v", err)
	}
}

func TestValidate_UnknownSource(t *testing.T) {
	cfg := &Config{
		Source:  "unknown",
		Targets: []string{"copilot"},
	}
	err := cfg.Validate([]string{"claude", "copilot"})
	if err == nil {
		t.Fatal("Validate() expected error for unknown source")
	}
	if got := err.Error(); !contains(got, "unknown") {
		t.Errorf("error should mention unknown provider, got: %v", got)
	}
}

func TestValidate_UnknownTarget(t *testing.T) {
	cfg := &Config{
		Source:  "claude",
		Targets: []string{"nonexistent"},
	}
	err := cfg.Validate([]string{"claude", "copilot"})
	if err == nil {
		t.Fatal("Validate() expected error for unknown target")
	}
	if got := err.Error(); !contains(got, "nonexistent") {
		t.Errorf("error should mention unknown target, got: %v", got)
	}
}

func TestValidate_SourceInTargets(t *testing.T) {
	cfg := &Config{
		Source:  "claude",
		Targets: []string{"claude", "copilot"},
	}
	err := cfg.Validate([]string{"claude", "copilot"})
	if err == nil {
		t.Fatal("Validate() expected error when source appears in targets")
	}
	if got := err.Error(); !contains(got, "must not appear in targets") {
		t.Errorf("error should mention source in targets, got: %v", got)
	}
}

func TestValidate_EmptySource(t *testing.T) {
	cfg := &Config{
		Source:  "",
		Targets: []string{"copilot"},
	}
	err := cfg.Validate([]string{"claude", "copilot"})
	if err == nil {
		t.Fatal("Validate() expected error for empty source")
	}
}

func TestValidate_EmptyTargets(t *testing.T) {
	cfg := &Config{
		Source:  "claude",
		Targets: []string{},
	}
	err := cfg.Validate([]string{"claude", "copilot"})
	if err == nil {
		t.Fatal("Validate() expected error for empty targets")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstr(s, substr)
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
