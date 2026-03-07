package provider

import (
	"os"
	"path/filepath"
	"testing"
)

func writeCopilotSkill(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".prompt.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestCopilotName(t *testing.T) {
	p := NewCopilotProvider(WithCopilotBaseDir(t.TempDir()))
	if got := p.Name(); got != "copilot" {
		t.Errorf("Name() = %q, want %q", got, "copilot")
	}
}

func TestCopilotSkillDir(t *testing.T) {
	dir := t.TempDir()
	p := NewCopilotProvider(WithCopilotBaseDir(dir))
	if got := p.SkillDir(); got != dir {
		t.Errorf("SkillDir() = %q, want %q", got, dir)
	}
}

func TestCopilotListSkills_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	writeCopilotSkill(t, dir, "alpha", "# Alpha skill\nDo alpha things")
	writeCopilotSkill(t, dir, "beta", "# Beta skill\nDo beta things")
	writeCopilotSkill(t, dir, "gamma", "Just gamma content")

	p := NewCopilotProvider(WithCopilotBaseDir(dir))
	skills, err := p.ListSkills()
	if err != nil {
		t.Fatalf("ListSkills() error: %v", err)
	}
	if len(skills) != 3 {
		t.Fatalf("ListSkills() returned %d skills, want 3", len(skills))
	}

	names := map[string]bool{}
	for _, s := range skills {
		names[s.Name] = true
	}
	for _, want := range []string{"alpha", "beta", "gamma"} {
		if !names[want] {
			t.Errorf("ListSkills() missing skill %q", want)
		}
	}
}

func TestCopilotListSkills_IgnoresNonPromptMd(t *testing.T) {
	dir := t.TempDir()
	// Write a .prompt.md file (should be found)
	writeCopilotSkill(t, dir, "real-skill", "# Real\nThis is a real prompt")
	// Write a plain .md file (should be ignored)
	if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte("# Notes\nJust notes"), 0644); err != nil {
		t.Fatal(err)
	}
	// Write a .txt file (should be ignored)
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("readme"), 0644); err != nil {
		t.Fatal(err)
	}

	p := NewCopilotProvider(WithCopilotBaseDir(dir))
	skills, err := p.ListSkills()
	if err != nil {
		t.Fatalf("ListSkills() error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("ListSkills() returned %d skills, want 1", len(skills))
	}
	if skills[0].Name != "real-skill" {
		t.Errorf("skill name = %q, want %q", skills[0].Name, "real-skill")
	}
}

func TestCopilotListSkills_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	p := NewCopilotProvider(WithCopilotBaseDir(dir))
	skills, err := p.ListSkills()
	if err != nil {
		t.Fatalf("ListSkills() error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("ListSkills() returned %d skills, want 0", len(skills))
	}
}

func TestCopilotListSkills_NonExistentDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent")
	p := NewCopilotProvider(WithCopilotBaseDir(dir))
	_, err := p.ListSkills()
	if err == nil {
		t.Error("ListSkills() expected error for nonexistent dir, got nil")
	}
}

func TestCopilotReadSkill_WithDescription(t *testing.T) {
	dir := t.TempDir()
	content := "# Review the code\nPlease review the following code changes"
	writeCopilotSkill(t, dir, "review-code", content)

	p := NewCopilotProvider(WithCopilotBaseDir(dir))
	skill, err := p.ReadSkill("review-code")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}
	if skill.Name != "review-code" {
		t.Errorf("Name = %q, want %q", skill.Name, "review-code")
	}
	if skill.Description != "Review the code" {
		t.Errorf("Description = %q, want %q", skill.Description, "Review the code")
	}
	if skill.Content != content {
		t.Errorf("Content = %q, want %q", skill.Content, content)
	}
}

func TestCopilotReadSkill_NoDescription(t *testing.T) {
	dir := t.TempDir()
	content := "Just some regular content\nNo description here"
	writeCopilotSkill(t, dir, "plain", content)

	p := NewCopilotProvider(WithCopilotBaseDir(dir))
	skill, err := p.ReadSkill("plain")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}
	if skill.Description != "" {
		t.Errorf("Description = %q, want empty", skill.Description)
	}
}

func TestCopilotReadSkill_HashWithoutSpace(t *testing.T) {
	dir := t.TempDir()
	content := "#notadescription\nSome content"
	writeCopilotSkill(t, dir, "hashno", content)

	p := NewCopilotProvider(WithCopilotBaseDir(dir))
	skill, err := p.ReadSkill("hashno")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}
	if skill.Description != "" {
		t.Errorf("Description = %q, want empty (no space after #)", skill.Description)
	}
}

func TestCopilotReadSkill_DoubleHash(t *testing.T) {
	dir := t.TempDir()
	content := "## Section header\nSome content"
	writeCopilotSkill(t, dir, "doublehash", content)

	p := NewCopilotProvider(WithCopilotBaseDir(dir))
	skill, err := p.ReadSkill("doublehash")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}
	if skill.Description != "" {
		t.Errorf("Description = %q, want empty (## is not a description)", skill.Description)
	}
}

func TestCopilotReadSkill_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	writeCopilotSkill(t, dir, "empty", "")

	p := NewCopilotProvider(WithCopilotBaseDir(dir))
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

func TestCopilotReadSkill_NonExistent(t *testing.T) {
	dir := t.TempDir()
	p := NewCopilotProvider(WithCopilotBaseDir(dir))
	_, err := p.ReadSkill("nonexistent")
	if err == nil {
		t.Error("ReadSkill() expected error for nonexistent skill, got nil")
	}
}

func TestCopilotReadSkill_WithFileReferences(t *testing.T) {
	dir := t.TempDir()
	content := "# Review prompt\nPlease review the changes in #file:src/main.go and #file:src/util.go"
	writeCopilotSkill(t, dir, "review", content)

	p := NewCopilotProvider(WithCopilotBaseDir(dir))
	skill, err := p.ReadSkill("review")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}
	if skill.Content != content {
		t.Errorf("Content = %q, want %q (file references should be preserved)", skill.Content, content)
	}
}

func TestCopilotWriteSkill_Basic(t *testing.T) {
	dir := t.TempDir()
	p := NewCopilotProvider(WithCopilotBaseDir(dir))

	content := "# My Skill\nDo the thing"
	skill := Skill{
		Name:    "my-skill",
		Content: content,
	}
	if err := p.WriteSkill(skill); err != nil {
		t.Fatalf("WriteSkill() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "my-skill.prompt.md"))
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if string(data) != content {
		t.Errorf("file content = %q, want %q", string(data), content)
	}
}

func TestCopilotWriteSkill_CreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new", "nested", "dir")
	p := NewCopilotProvider(WithCopilotBaseDir(dir))

	skill := Skill{
		Name:    "test",
		Content: "# Test\nHello",
	}
	if err := p.WriteSkill(skill); err != nil {
		t.Fatalf("WriteSkill() error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "test.prompt.md")); err != nil {
		t.Errorf("expected file to exist: %v", err)
	}
}

func TestCopilotWriteSkill_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := NewCopilotProvider(WithCopilotBaseDir(dir))

	original := Skill{
		Name:    "roundtrip",
		Content: "# Round Trip Test\nReview #file:src/main.go\nMore content here",
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
	if got.Description != "Round Trip Test" {
		t.Errorf("Description = %q, want %q", got.Description, "Round Trip Test")
	}
	if got.Content != original.Content {
		t.Errorf("Content = %q, want %q", got.Content, original.Content)
	}
	if got.SourcePath == "" {
		t.Error("SourcePath should not be empty after ReadSkill")
	}
}

func TestCopilotWriteSkill_RoundTrip_NoDescription(t *testing.T) {
	dir := t.TempDir()
	p := NewCopilotProvider(WithCopilotBaseDir(dir))

	original := Skill{
		Name:    "nodesc",
		Content: "No description line here\nJust content",
	}
	if err := p.WriteSkill(original); err != nil {
		t.Fatalf("WriteSkill() error: %v", err)
	}

	got, err := p.ReadSkill("nodesc")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}

	if got.Description != "" {
		t.Errorf("Description = %q, want empty", got.Description)
	}
	if got.Content != original.Content {
		t.Errorf("Content = %q, want %q", got.Content, original.Content)
	}
}
