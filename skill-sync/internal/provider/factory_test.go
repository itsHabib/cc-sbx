package provider

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// writeFactorySkill creates a SKILL.md file inside <dir>/<name>/SKILL.md.
func writeFactorySkill(t *testing.T, dir, name, content string) {
	t.Helper()
	skillDir := filepath.Join(dir, name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestFactoryName(t *testing.T) {
	p := NewFactoryProvider(WithFactoryBaseDir(t.TempDir()))
	if got := p.Name(); got != "factory" {
		t.Errorf("Name() = %q, want %q", got, "factory")
	}
}

func TestFactorySkillDir(t *testing.T) {
	dir := t.TempDir()
	p := NewFactoryProvider(WithFactoryBaseDir(dir))
	if got := p.SkillDir(); got != dir {
		t.Errorf("SkillDir() = %q, want %q", got, dir)
	}
}

func TestFactoryReadSkill_WithFrontmatter(t *testing.T) {
	dir := t.TempDir()
	content := "---\nname: worker\ndescription: General-purpose worker droid\n---\n# Worker Droid\n\nMarkdown prompt body here.\n"
	writeFactorySkill(t, dir, "worker", content)

	p := NewFactoryProvider(WithFactoryBaseDir(dir))
	skill, err := p.ReadSkill("worker")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}
	if skill.Name != "worker" {
		t.Errorf("Name = %q, want %q", skill.Name, "worker")
	}
	if skill.Description != "General-purpose worker droid" {
		t.Errorf("Description = %q, want %q", skill.Description, "General-purpose worker droid")
	}
	wantBody := "# Worker Droid\n\nMarkdown prompt body here.\n"
	if skill.Content != wantBody {
		t.Errorf("Content = %q, want %q", skill.Content, wantBody)
	}
	if skill.SourcePath == "" {
		t.Error("SourcePath should not be empty after ReadSkill")
	}
}

func TestFactoryReadSkill_WithFrontmatterAndModel(t *testing.T) {
	dir := t.TempDir()
	content := "---\nname: reviewer\ndescription: Code review droid\nmodel: inherit\n---\nReview code carefully.\n"
	writeFactorySkill(t, dir, "reviewer", content)

	p := NewFactoryProvider(WithFactoryBaseDir(dir))
	skill, err := p.ReadSkill("reviewer")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}
	if skill.Name != "reviewer" {
		t.Errorf("Name = %q, want %q", skill.Name, "reviewer")
	}
	if skill.Description != "Code review droid" {
		t.Errorf("Description = %q, want %q", skill.Description, "Code review droid")
	}
	wantBody := "Review code carefully.\n"
	if skill.Content != wantBody {
		t.Errorf("Content = %q, want %q", skill.Content, wantBody)
	}
}

func TestFactoryReadSkill_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	content := "# Simple Droid\n\nJust a plain markdown file.\n"
	writeFactorySkill(t, dir, "simple", content)

	p := NewFactoryProvider(WithFactoryBaseDir(dir))
	skill, err := p.ReadSkill("simple")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}
	if skill.Name != "simple" {
		t.Errorf("Name = %q, want %q (from directory name)", skill.Name, "simple")
	}
	if skill.Description != "Simple Droid" {
		t.Errorf("Description = %q, want %q", skill.Description, "Simple Droid")
	}
	if skill.Content != content {
		t.Errorf("Content = %q, want %q", skill.Content, content)
	}
}

func TestFactoryReadSkill_NoFrontmatterNoDescription(t *testing.T) {
	dir := t.TempDir()
	content := "Just plain text with no heading.\n"
	writeFactorySkill(t, dir, "plain", content)

	p := NewFactoryProvider(WithFactoryBaseDir(dir))
	skill, err := p.ReadSkill("plain")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}
	if skill.Name != "plain" {
		t.Errorf("Name = %q, want %q", skill.Name, "plain")
	}
	if skill.Description != "" {
		t.Errorf("Description = %q, want empty", skill.Description)
	}
}

func TestFactoryReadSkill_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	writeFactorySkill(t, dir, "empty", "")

	p := NewFactoryProvider(WithFactoryBaseDir(dir))
	skill, err := p.ReadSkill("empty")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}
	if skill.Name != "empty" {
		t.Errorf("Name = %q, want %q", skill.Name, "empty")
	}
	if skill.Description != "" {
		t.Errorf("Description = %q, want empty", skill.Description)
	}
	if skill.Content != "" {
		t.Errorf("Content = %q, want empty", skill.Content)
	}
}

func TestFactoryReadSkill_FrontmatterOnly(t *testing.T) {
	dir := t.TempDir()
	content := "---\nname: stub\ndescription: A stub skill\n---\n"
	writeFactorySkill(t, dir, "stub", content)

	p := NewFactoryProvider(WithFactoryBaseDir(dir))
	skill, err := p.ReadSkill("stub")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}
	if skill.Name != "stub" {
		t.Errorf("Name = %q, want %q", skill.Name, "stub")
	}
	if skill.Description != "A stub skill" {
		t.Errorf("Description = %q, want %q", skill.Description, "A stub skill")
	}
	if skill.Content != "" {
		t.Errorf("Content = %q, want empty", skill.Content)
	}
}

func TestFactoryReadSkill_NonExistent(t *testing.T) {
	dir := t.TempDir()
	p := NewFactoryProvider(WithFactoryBaseDir(dir))
	_, err := p.ReadSkill("nonexistent")
	if err == nil {
		t.Error("ReadSkill() expected error for nonexistent skill, got nil")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected error wrapping os.ErrNotExist, got: %v", err)
	}
}

func TestFactoryReadSkill_FrontmatterNameOverridesDir(t *testing.T) {
	dir := t.TempDir()
	content := "---\nname: custom-name\ndescription: Overridden name\n---\nBody.\n"
	writeFactorySkill(t, dir, "dirname", content)

	p := NewFactoryProvider(WithFactoryBaseDir(dir))
	skill, err := p.ReadSkill("dirname")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}
	if skill.Name != "custom-name" {
		t.Errorf("Name = %q, want %q (frontmatter overrides dir)", skill.Name, "custom-name")
	}
}

func TestFactoryListSkills_Multiple(t *testing.T) {
	dir := t.TempDir()
	writeFactorySkill(t, dir, "alpha", "---\nname: alpha\ndescription: Alpha\n---\nAlpha body\n")
	writeFactorySkill(t, dir, "beta", "# Beta\n\nBeta body\n")

	p := NewFactoryProvider(WithFactoryBaseDir(dir))
	skills, err := p.ListSkills()
	if err != nil {
		t.Fatalf("ListSkills() error: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("ListSkills() returned %d skills, want 2", len(skills))
	}

	names := map[string]bool{}
	for _, s := range skills {
		names[s.Name] = true
	}
	for _, want := range []string{"alpha", "beta"} {
		if !names[want] {
			t.Errorf("ListSkills() missing skill %q", want)
		}
	}
}

func TestFactoryListSkills_Empty(t *testing.T) {
	dir := t.TempDir()
	p := NewFactoryProvider(WithFactoryBaseDir(dir))
	skills, err := p.ListSkills()
	if err != nil {
		t.Fatalf("ListSkills() error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("ListSkills() returned %d skills, want 0", len(skills))
	}
}

func TestFactoryListSkills_NonExistentDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent")
	p := NewFactoryProvider(WithFactoryBaseDir(dir))
	_, err := p.ListSkills()
	if err == nil {
		t.Error("ListSkills() expected error for nonexistent dir, got nil")
	}
}

func TestFactoryListSkills_SkipsNonSkillDirs(t *testing.T) {
	dir := t.TempDir()
	writeFactorySkill(t, dir, "real", "---\nname: real\ndescription: Real skill\n---\nBody\n")
	// Create a directory without SKILL.md
	if err := os.MkdirAll(filepath.Join(dir, "not-a-skill"), 0755); err != nil {
		t.Fatal(err)
	}
	// Create a regular file (not a directory) in baseDir
	if err := os.WriteFile(filepath.Join(dir, "random.txt"), []byte("ignore me"), 0644); err != nil {
		t.Fatal(err)
	}

	p := NewFactoryProvider(WithFactoryBaseDir(dir))
	skills, err := p.ListSkills()
	if err != nil {
		t.Fatalf("ListSkills() error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("ListSkills() returned %d skills, want 1", len(skills))
	}
	if skills[0].Name != "real" {
		t.Errorf("skill name = %q, want %q", skills[0].Name, "real")
	}
}

func TestFactoryWriteSkill_Basic(t *testing.T) {
	dir := t.TempDir()
	p := NewFactoryProvider(WithFactoryBaseDir(dir))

	skill := Skill{
		Name:        "worker",
		Description: "General-purpose worker",
		Content:     "# Worker\n\nDo the work.\n",
	}
	if err := p.WriteSkill(skill); err != nil {
		t.Fatalf("WriteSkill() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "worker", "SKILL.md"))
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	got := string(data)
	want := "---\nname: worker\ndescription: General-purpose worker\n---\n# Worker\n\nDo the work.\n"
	if got != want {
		t.Errorf("file content =\n%s\nwant:\n%s", got, want)
	}
}

func TestFactoryWriteSkill_CreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new", "nested")
	p := NewFactoryProvider(WithFactoryBaseDir(dir))

	skill := Skill{
		Name:        "test",
		Description: "Test skill",
		Content:     "Hello\n",
	}
	if err := p.WriteSkill(skill); err != nil {
		t.Fatalf("WriteSkill() error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "test", "SKILL.md")); err != nil {
		t.Errorf("expected SKILL.md to exist: %v", err)
	}
}

func TestFactoryWriteSkill_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := NewFactoryProvider(WithFactoryBaseDir(dir))

	original := Skill{
		Name:        "roundtrip",
		Description: "Round trip test",
		Content:     "# Round Trip\n\nContent goes here.\n",
	}
	if err := p.WriteSkill(original); err != nil {
		t.Fatalf("WriteSkill() error: %v", err)
	}

	got, err := p.ReadSkill("roundtrip")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}

	if got.Name != original.Name {
		t.Errorf("Name = %q, want %q", got.Name, original.Name)
	}
	if got.Description != original.Description {
		t.Errorf("Description = %q, want %q", got.Description, original.Description)
	}
	if got.Content != original.Content {
		t.Errorf("Content = %q, want %q", got.Content, original.Content)
	}
	if got.SourcePath == "" {
		t.Error("SourcePath should not be empty after ReadSkill")
	}
}

func TestFactoryWriteSkill_RoundTrip_NoDescription(t *testing.T) {
	dir := t.TempDir()
	p := NewFactoryProvider(WithFactoryBaseDir(dir))

	original := Skill{
		Name:        "nodesc",
		Description: "",
		Content:     "Just content, no description.\n",
	}
	if err := p.WriteSkill(original); err != nil {
		t.Fatalf("WriteSkill() error: %v", err)
	}

	got, err := p.ReadSkill("nodesc")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}

	if got.Name != original.Name {
		t.Errorf("Name = %q, want %q", got.Name, original.Name)
	}
	if got.Description != original.Description {
		t.Errorf("Description = %q, want %q", got.Description, original.Description)
	}
	if got.Content != original.Content {
		t.Errorf("Content = %q, want %q", got.Content, original.Content)
	}
}

func TestFactoryReadSkill_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	content := "---\nname: [invalid yaml\n---\nBody\n"
	writeFactorySkill(t, dir, "bad", content)

	p := NewFactoryProvider(WithFactoryBaseDir(dir))
	_, err := p.ReadSkill("bad")
	if err == nil {
		t.Error("ReadSkill() expected error for malformed YAML, got nil")
	}
}
