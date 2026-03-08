package provider

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFactoryName(t *testing.T) {
	p := newTestProvider("factory", t.TempDir())
	if got := p.Name(); got != "factory" {
		t.Errorf("Name() = %q, want %q", got, "factory")
	}
}

func TestFactorySkillDir(t *testing.T) {
	dir := t.TempDir()
	p := newTestProvider("factory", dir)
	if got := p.SkillDir(); got != dir {
		t.Errorf("SkillDir() = %q, want %q", got, dir)
	}
}

func TestFactoryReadSkill_WithFrontmatter(t *testing.T) {
	dir := t.TempDir()
	content := "---\nname: worker\ndescription: General-purpose worker droid\n---\n# Worker Droid\n\nMarkdown prompt body here.\n"
	writeTestSkill(t, dir, "worker", content)

	p := newTestProvider("factory", dir)
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
	// Content should be the full raw file content (including frontmatter).
	if skill.Content != content {
		t.Errorf("Content = %q, want %q", skill.Content, content)
	}
	if skill.SourcePath == "" {
		t.Error("SourcePath should not be empty after ReadSkill")
	}
}

func TestFactoryReadSkill_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	content := "# Simple Droid\n\nJust a plain markdown file.\n"
	writeTestSkill(t, dir, "simple", content)

	p := newTestProvider("factory", dir)
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

func TestFactoryReadSkill_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	writeTestSkill(t, dir, "empty", "")

	p := newTestProvider("factory", dir)
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

func TestFactoryReadSkill_NonExistent(t *testing.T) {
	dir := t.TempDir()
	p := newTestProvider("factory", dir)
	_, err := p.ReadSkill("nonexistent")
	if err == nil {
		t.Error("ReadSkill() expected error for nonexistent skill, got nil")
	}
}

func TestFactoryListSkills_Multiple(t *testing.T) {
	dir := t.TempDir()
	writeTestSkill(t, dir, "alpha", "---\nname: alpha\ndescription: Alpha\n---\nAlpha body\n")
	writeTestSkill(t, dir, "beta", "# Beta\n\nBeta body\n")

	p := newTestProvider("factory", dir)
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
	p := newTestProvider("factory", dir)
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
	p := newTestProvider("factory", dir)
	_, err := p.ListSkills()
	if err == nil {
		t.Error("ListSkills() expected error for nonexistent dir, got nil")
	}
}

func TestFactoryListSkills_SkipsNonSkillDirs(t *testing.T) {
	dir := t.TempDir()
	writeTestSkill(t, dir, "real", "---\nname: real\ndescription: Real skill\n---\nBody\n")
	// Create a directory without SKILL.md
	if err := os.MkdirAll(filepath.Join(dir, "not-a-skill"), 0755); err != nil {
		t.Fatal(err)
	}
	// Create a regular file (not a directory) in baseDir
	if err := os.WriteFile(filepath.Join(dir, "random.txt"), []byte("ignore me"), 0644); err != nil {
		t.Fatal(err)
	}

	p := newTestProvider("factory", dir)
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
	p := newTestProvider("factory", dir)

	content := "---\nname: worker\ndescription: General-purpose worker\n---\n# Worker\n\nDo the work.\n"
	skill := Skill{
		Name:    "worker",
		Content: content,
	}
	if err := p.WriteSkill(skill); err != nil {
		t.Fatalf("WriteSkill() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "worker", "SKILL.md"))
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if string(data) != content {
		t.Errorf("file content =\n%s\nwant:\n%s", string(data), content)
	}
}

func TestFactoryWriteSkill_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := newTestProvider("factory", dir)

	original := Skill{
		Name:    "roundtrip",
		Content: "---\nname: roundtrip\ndescription: Round trip test\n---\n# Round Trip\n\nContent goes here.\n",
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
	if got.Description != "Round trip test" {
		t.Errorf("Description = %q, want %q", got.Description, "Round trip test")
	}
	if got.Content != original.Content {
		t.Errorf("Content = %q, want %q", got.Content, original.Content)
	}
	if got.SourcePath == "" {
		t.Error("SourcePath should not be empty after ReadSkill")
	}
}
