package provider

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGeminiName(t *testing.T) {
	p := newTestProvider("gemini", t.TempDir())
	if got := p.Name(); got != "gemini" {
		t.Errorf("Name() = %q, want %q", got, "gemini")
	}
}

func TestGeminiSkillDir(t *testing.T) {
	dir := t.TempDir()
	p := newTestProvider("gemini", dir)
	if got := p.SkillDir(); got != dir {
		t.Errorf("SkillDir() = %q, want %q", got, dir)
	}
}

func TestGeminiListSkills_MultipleSkills(t *testing.T) {
	dir := t.TempDir()
	writeTestSkill(t, dir, "alpha", "# Alpha\nAlpha prompt")
	writeTestSkill(t, dir, "beta", "# Beta\nBeta prompt")
	writeTestSkill(t, dir, "gamma", "Just gamma content")

	p := newTestProvider("gemini", dir)
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

func TestGeminiListSkills_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	p := newTestProvider("gemini", dir)
	skills, err := p.ListSkills()
	if err != nil {
		t.Fatalf("ListSkills() error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("ListSkills() returned %d skills, want 0", len(skills))
	}
}

func TestGeminiListSkills_NonExistentDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent")
	p := newTestProvider("gemini", dir)
	_, err := p.ListSkills()
	if err == nil {
		t.Error("ListSkills() expected error for nonexistent dir, got nil")
	}
}

func TestGeminiReadSkill_WithFrontmatter(t *testing.T) {
	dir := t.TempDir()
	content := "---\nname: deploy\ndescription: Deploy to production\n---\n# Deploy\n\nRun the deploy script.\n"
	writeTestSkill(t, dir, "deploy", content)

	p := newTestProvider("gemini", dir)
	skill, err := p.ReadSkill("deploy")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}
	if skill.Name != "deploy" {
		t.Errorf("Name = %q, want %q", skill.Name, "deploy")
	}
	if skill.Description != "Deploy to production" {
		t.Errorf("Description = %q, want %q", skill.Description, "Deploy to production")
	}
}

func TestGeminiReadSkill_WithHeading(t *testing.T) {
	dir := t.TempDir()
	content := "# Deploy to production\nRun the deploy script"
	writeTestSkill(t, dir, "deploy", content)

	p := newTestProvider("gemini", dir)
	skill, err := p.ReadSkill("deploy")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}
	if skill.Description != "Deploy to production" {
		t.Errorf("Description = %q, want %q", skill.Description, "Deploy to production")
	}
}

func TestGeminiReadSkill_NonExistent(t *testing.T) {
	dir := t.TempDir()
	p := newTestProvider("gemini", dir)
	_, err := p.ReadSkill("nonexistent")
	if err == nil {
		t.Error("ReadSkill() expected error for nonexistent skill, got nil")
	}
}

func TestGeminiWriteSkill_Basic(t *testing.T) {
	dir := t.TempDir()
	p := newTestProvider("gemini", dir)

	content := "# Deploy\nRun the deploy script"
	skill := Skill{
		Name:    "deploy",
		Content: content,
	}
	if err := p.WriteSkill(skill); err != nil {
		t.Fatalf("WriteSkill() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "deploy", "SKILL.md"))
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if string(data) != content {
		t.Errorf("file content = %q, want %q", string(data), content)
	}
}

func TestGeminiWriteSkill_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := newTestProvider("gemini", dir)

	original := Skill{
		Name:    "roundtrip",
		Content: "---\nname: roundtrip\ndescription: Test round trip\n---\n# Round Trip\n\nContent here.\n",
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
	if got.Description != "Test round trip" {
		t.Errorf("Description = %q, want %q", got.Description, "Test round trip")
	}
	if got.Content != original.Content {
		t.Errorf("Content = %q, want %q", got.Content, original.Content)
	}
	if got.SourcePath == "" {
		t.Error("SourcePath should not be empty after ReadSkill")
	}
}
