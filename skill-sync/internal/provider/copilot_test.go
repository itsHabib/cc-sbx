package provider

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopilotName(t *testing.T) {
	p := newTestProvider("copilot", t.TempDir())
	if got := p.Name(); got != "copilot" {
		t.Errorf("Name() = %q, want %q", got, "copilot")
	}
}

func TestCopilotSkillDir(t *testing.T) {
	dir := t.TempDir()
	p := newTestProvider("copilot", dir)
	if got := p.SkillDir(); got != dir {
		t.Errorf("SkillDir() = %q, want %q", got, dir)
	}
}

func TestCopilotListSkills_MultipleSkills(t *testing.T) {
	dir := t.TempDir()
	writeTestSkill(t, dir, "alpha", "# Alpha skill\nDo alpha things")
	writeTestSkill(t, dir, "beta", "# Beta skill\nDo beta things")
	writeTestSkill(t, dir, "gamma", "Just gamma content")

	p := newTestProvider("copilot", dir)
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

func TestCopilotListSkills_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	p := newTestProvider("copilot", dir)
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
	p := newTestProvider("copilot", dir)
	_, err := p.ListSkills()
	if err == nil {
		t.Error("ListSkills() expected error for nonexistent dir, got nil")
	}
}

func TestCopilotReadSkill_WithDescription(t *testing.T) {
	dir := t.TempDir()
	content := "# Review the code\nPlease review the following code changes"
	writeTestSkill(t, dir, "review-code", content)

	p := newTestProvider("copilot", dir)
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

func TestCopilotReadSkill_WithFrontmatter(t *testing.T) {
	dir := t.TempDir()
	content := "---\nname: review-code\ndescription: Review code changes\n---\n# Review\n\nPlease review.\n"
	writeTestSkill(t, dir, "review-code", content)

	p := newTestProvider("copilot", dir)
	skill, err := p.ReadSkill("review-code")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}
	if skill.Description != "Review code changes" {
		t.Errorf("Description = %q, want %q", skill.Description, "Review code changes")
	}
}

func TestCopilotReadSkill_NonExistent(t *testing.T) {
	dir := t.TempDir()
	p := newTestProvider("copilot", dir)
	_, err := p.ReadSkill("nonexistent")
	if err == nil {
		t.Error("ReadSkill() expected error for nonexistent skill, got nil")
	}
}

func TestCopilotWriteSkill_Basic(t *testing.T) {
	dir := t.TempDir()
	p := newTestProvider("copilot", dir)

	content := "# My Skill\nDo the thing"
	skill := Skill{
		Name:    "my-skill",
		Content: content,
	}
	if err := p.WriteSkill(skill); err != nil {
		t.Fatalf("WriteSkill() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "my-skill", "SKILL.md"))
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if string(data) != content {
		t.Errorf("file content = %q, want %q", string(data), content)
	}
}

func TestCopilotWriteSkill_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := newTestProvider("copilot", dir)

	original := Skill{
		Name:    "roundtrip",
		Content: "# Round Trip Test\nMore content here",
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
