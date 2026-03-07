package provider

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// writeTestSkill creates a skill at <dir>/<name>/SKILL.md.
func writeTestSkill(t *testing.T, dir, name, content string) {
	t.Helper()
	skillDir := filepath.Join(dir, name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestName(t *testing.T) {
	p := NewClaudeProvider(WithBaseDir(t.TempDir()))
	if got := p.Name(); got != "claude" {
		t.Errorf("Name() = %q, want %q", got, "claude")
	}
}

func TestSkillDir(t *testing.T) {
	dir := t.TempDir()
	p := NewClaudeProvider(WithBaseDir(dir))
	if got := p.SkillDir(); got != dir {
		t.Errorf("SkillDir() = %q, want %q", got, dir)
	}
}

func TestListSkills_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	writeTestSkill(t, dir, "alpha", "# Alpha skill\nDo alpha things")
	writeTestSkill(t, dir, "beta", "# Beta skill\nDo beta things")
	writeTestSkill(t, dir, "gamma", "Just gamma content")

	p := NewClaudeProvider(WithBaseDir(dir))
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

func TestListSkills_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	p := NewClaudeProvider(WithBaseDir(dir))
	skills, err := p.ListSkills()
	if err != nil {
		t.Fatalf("ListSkills() error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("ListSkills() returned %d skills, want 0", len(skills))
	}
}

func TestListSkills_NonExistentDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent")
	p := NewClaudeProvider(WithBaseDir(dir))
	_, err := p.ListSkills()
	if err == nil {
		t.Error("ListSkills() expected error for nonexistent dir, got nil")
	}
}

func TestListSkills_SkipsDirsWithoutSkillMD(t *testing.T) {
	dir := t.TempDir()
	writeTestSkill(t, dir, "real", "# Real skill\nContent")
	// Create a directory without SKILL.md
	os.MkdirAll(filepath.Join(dir, "not-a-skill"), 0755)
	os.WriteFile(filepath.Join(dir, "not-a-skill", "README.md"), []byte("not a skill"), 0644)

	p := NewClaudeProvider(WithBaseDir(dir))
	skills, err := p.ListSkills()
	if err != nil {
		t.Fatalf("ListSkills() error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("ListSkills() returned %d skills, want 1 (should skip dirs without SKILL.md)", len(skills))
	}
	if skills[0].Name != "real" {
		t.Errorf("skill name = %q, want %q", skills[0].Name, "real")
	}
}

func TestReadSkill_WithDescription(t *testing.T) {
	dir := t.TempDir()
	content := "# Deploy to prod\nRun the deploy script with $ARGUMENTS"
	writeTestSkill(t, dir, "deploy", content)

	p := NewClaudeProvider(WithBaseDir(dir))
	skill, err := p.ReadSkill("deploy")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}
	if skill.Name != "deploy" {
		t.Errorf("Name = %q, want %q", skill.Name, "deploy")
	}
	if skill.Description != "Deploy to prod" {
		t.Errorf("Description = %q, want %q", skill.Description, "Deploy to prod")
	}
	if skill.Content != content {
		t.Errorf("Content = %q, want %q", skill.Content, content)
	}
}

func TestReadSkill_NoDescription(t *testing.T) {
	dir := t.TempDir()
	content := "Just some regular content\nNo description here"
	writeTestSkill(t, dir, "plain", content)

	p := NewClaudeProvider(WithBaseDir(dir))
	skill, err := p.ReadSkill("plain")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}
	if skill.Description != "" {
		t.Errorf("Description = %q, want empty", skill.Description)
	}
}

func TestReadSkill_HashWithoutSpace(t *testing.T) {
	dir := t.TempDir()
	content := "#notadescription\nSome content"
	writeTestSkill(t, dir, "hashno", content)

	p := NewClaudeProvider(WithBaseDir(dir))
	skill, err := p.ReadSkill("hashno")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}
	if skill.Description != "" {
		t.Errorf("Description = %q, want empty (no space after #)", skill.Description)
	}
}

func TestReadSkill_DoubleHash(t *testing.T) {
	dir := t.TempDir()
	content := "## Section header\nSome content"
	writeTestSkill(t, dir, "doublehash", content)

	p := NewClaudeProvider(WithBaseDir(dir))
	skill, err := p.ReadSkill("doublehash")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}
	if skill.Description != "" {
		t.Errorf("Description = %q, want empty (## is not a description)", skill.Description)
	}
}

func TestReadSkill_WithArguments(t *testing.T) {
	dir := t.TempDir()
	content := "# Search\nSearch for $ARGUMENTS in ${PROJECT} and ${QUERY}"
	writeTestSkill(t, dir, "search", content)

	p := NewClaudeProvider(WithBaseDir(dir))
	skill, err := p.ReadSkill("search")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}
	want := []string{"$ARGUMENTS", "${PROJECT}", "${QUERY}"}
	if !reflect.DeepEqual(skill.Arguments, want) {
		t.Errorf("Arguments = %v, want %v", skill.Arguments, want)
	}
}

func TestReadSkill_NoArguments(t *testing.T) {
	dir := t.TempDir()
	content := "# Simple\nJust do the thing"
	writeTestSkill(t, dir, "simple", content)

	p := NewClaudeProvider(WithBaseDir(dir))
	skill, err := p.ReadSkill("simple")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}
	if len(skill.Arguments) != 0 {
		t.Errorf("Arguments = %v, want empty", skill.Arguments)
	}
}

func TestReadSkill_DuplicateArguments(t *testing.T) {
	dir := t.TempDir()
	content := "Use $ARGUMENTS then $ARGUMENTS again and ${QUERY} then ${QUERY}"
	writeTestSkill(t, dir, "dupes", content)

	p := NewClaudeProvider(WithBaseDir(dir))
	skill, err := p.ReadSkill("dupes")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}
	want := []string{"$ARGUMENTS", "${QUERY}"}
	if !reflect.DeepEqual(skill.Arguments, want) {
		t.Errorf("Arguments = %v, want %v (deduplicated)", skill.Arguments, want)
	}
}

func TestReadSkill_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	writeTestSkill(t, dir, "empty", "")

	p := NewClaudeProvider(WithBaseDir(dir))
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
	if len(skill.Arguments) != 0 {
		t.Errorf("Arguments = %v, want empty", skill.Arguments)
	}
}

func TestReadSkill_NonExistent(t *testing.T) {
	dir := t.TempDir()
	p := NewClaudeProvider(WithBaseDir(dir))
	_, err := p.ReadSkill("nonexistent")
	if err == nil {
		t.Error("ReadSkill() expected error for nonexistent skill, got nil")
	}
}

func TestWriteSkill_Basic(t *testing.T) {
	dir := t.TempDir()
	p := NewClaudeProvider(WithBaseDir(dir))

	content := "# My Skill\nDo the thing with $ARGUMENTS"
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

func TestWriteSkill_CreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new", "nested", "dir")
	p := NewClaudeProvider(WithBaseDir(dir))

	skill := Skill{
		Name:    "test",
		Content: "# Test\nHello",
	}
	if err := p.WriteSkill(skill); err != nil {
		t.Fatalf("WriteSkill() error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "test", "SKILL.md")); err != nil {
		t.Errorf("expected file to exist: %v", err)
	}
}

func TestWriteSkill_NoDescription(t *testing.T) {
	dir := t.TempDir()
	p := NewClaudeProvider(WithBaseDir(dir))

	content := "Just plain content, no description"
	skill := Skill{
		Name:    "plain",
		Content: content,
	}
	if err := p.WriteSkill(skill); err != nil {
		t.Fatalf("WriteSkill() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "plain", "SKILL.md"))
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if string(data) != content {
		t.Errorf("file content = %q, want %q", string(data), content)
	}
}

func TestWriteSkill_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := NewClaudeProvider(WithBaseDir(dir))

	original := Skill{
		Name:    "roundtrip",
		Content: "# Round Trip Test\nSearch for $ARGUMENTS in ${PROJECT}\nMore content here",
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
	wantArgs := []string{"$ARGUMENTS", "${PROJECT}"}
	if !reflect.DeepEqual(got.Arguments, wantArgs) {
		t.Errorf("Arguments = %v, want %v", got.Arguments, wantArgs)
	}
	if got.SourcePath == "" {
		t.Error("SourcePath should not be empty after ReadSkill")
	}
}

func TestWriteSkill_RoundTrip_NoDescription(t *testing.T) {
	dir := t.TempDir()
	p := NewClaudeProvider(WithBaseDir(dir))

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
