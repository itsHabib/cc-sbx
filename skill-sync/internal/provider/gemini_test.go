package provider

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeTestTOML(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestGeminiName(t *testing.T) {
	p := NewGeminiProvider(WithGeminiBaseDir(t.TempDir()))
	if got := p.Name(); got != "gemini" {
		t.Errorf("Name() = %q, want %q", got, "gemini")
	}
}

func TestGeminiSkillDir(t *testing.T) {
	dir := t.TempDir()
	p := NewGeminiProvider(WithGeminiBaseDir(dir))
	if got := p.SkillDir(); got != dir {
		t.Errorf("SkillDir() = %q, want %q", got, dir)
	}
}

func TestGeminiReadSkill_WithDescription(t *testing.T) {
	dir := t.TempDir()
	tomlContent := "description = \"Deploy to production\"\nprompt = \"Run the deploy script\"\n"
	writeTestTOML(t, dir, "deploy.toml", tomlContent)

	p := NewGeminiProvider(WithGeminiBaseDir(dir))
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
	if skill.Content != "Run the deploy script" {
		t.Errorf("Content = %q, want %q", skill.Content, "Run the deploy script")
	}
}

func TestGeminiReadSkill_NoDescription(t *testing.T) {
	dir := t.TempDir()
	tomlContent := "prompt = \"Just do the thing\"\n"
	writeTestTOML(t, dir, "simple.toml", tomlContent)

	p := NewGeminiProvider(WithGeminiBaseDir(dir))
	skill, err := p.ReadSkill("simple")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}
	if skill.Description != "" {
		t.Errorf("Description = %q, want empty", skill.Description)
	}
	if skill.Content != "Just do the thing" {
		t.Errorf("Content = %q, want %q", skill.Content, "Just do the thing")
	}
}

func TestGeminiReadSkill_WithArgs(t *testing.T) {
	dir := t.TempDir()
	tomlContent := "prompt = \"Search for {{args}} in the codebase\"\n"
	writeTestTOML(t, dir, "search.toml", tomlContent)

	p := NewGeminiProvider(WithGeminiBaseDir(dir))
	skill, err := p.ReadSkill("search")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}
	want := []string{"{{args}}"}
	if !reflect.DeepEqual(skill.Arguments, want) {
		t.Errorf("Arguments = %v, want %v", skill.Arguments, want)
	}
}

func TestGeminiReadSkill_EmptyPrompt(t *testing.T) {
	dir := t.TempDir()
	tomlContent := "description = \"Has no prompt\"\nprompt = \"\"\n"
	writeTestTOML(t, dir, "empty-prompt.toml", tomlContent)

	p := NewGeminiProvider(WithGeminiBaseDir(dir))
	_, err := p.ReadSkill("empty-prompt")
	if err == nil {
		t.Error("ReadSkill() expected error for empty prompt, got nil")
	}
}

func TestGeminiReadSkill_MissingPrompt(t *testing.T) {
	dir := t.TempDir()
	tomlContent := "description = \"No prompt field at all\"\n"
	writeTestTOML(t, dir, "no-prompt.toml", tomlContent)

	p := NewGeminiProvider(WithGeminiBaseDir(dir))
	_, err := p.ReadSkill("no-prompt")
	if err == nil {
		t.Error("ReadSkill() expected error for missing prompt, got nil")
	}
}

func TestGeminiReadSkill_MultiLinePrompt(t *testing.T) {
	dir := t.TempDir()
	tomlContent := "prompt = \"\"\"\\nLine one\\nLine two\\nLine three\\n\"\"\"\n"
	// Write a proper multi-line TOML string
	rawContent := "prompt = \"\"\"\nLine one\nLine two\nLine three\n\"\"\"\n"
	writeTestTOML(t, dir, "multiline.toml", rawContent)

	p := NewGeminiProvider(WithGeminiBaseDir(dir))
	skill, err := p.ReadSkill("multiline")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}
	_ = tomlContent // used only for documentation
	if !strings.Contains(skill.Content, "Line one") {
		t.Errorf("Content should contain 'Line one', got %q", skill.Content)
	}
	if !strings.Contains(skill.Content, "Line three") {
		t.Errorf("Content should contain 'Line three', got %q", skill.Content)
	}
}

func TestGeminiReadSkill_NonExistent(t *testing.T) {
	dir := t.TempDir()
	p := NewGeminiProvider(WithGeminiBaseDir(dir))
	_, err := p.ReadSkill("nonexistent")
	if err == nil {
		t.Error("ReadSkill() expected error for nonexistent skill, got nil")
	}
}

func TestGeminiReadSkill_InvalidTOML(t *testing.T) {
	dir := t.TempDir()
	writeTestTOML(t, dir, "bad.toml", "this is not valid toml {{{{")

	p := NewGeminiProvider(WithGeminiBaseDir(dir))
	_, err := p.ReadSkill("bad")
	if err == nil {
		t.Error("ReadSkill() expected error for invalid TOML, got nil")
	}
}

func TestGeminiReadSkill_Namespaced(t *testing.T) {
	dir := t.TempDir()
	tomlContent := "description = \"Commit changes\"\nprompt = \"Stage and commit with {{args}}\"\n"
	writeTestTOML(t, dir, "git/commit.toml", tomlContent)

	p := NewGeminiProvider(WithGeminiBaseDir(dir))
	skill, err := p.ReadSkill("git:commit")
	if err != nil {
		t.Fatalf("ReadSkill() error: %v", err)
	}
	if skill.Name != "git:commit" {
		t.Errorf("Name = %q, want %q", skill.Name, "git:commit")
	}
	if skill.Description != "Commit changes" {
		t.Errorf("Description = %q, want %q", skill.Description, "Commit changes")
	}
	want := []string{"{{args}}"}
	if !reflect.DeepEqual(skill.Arguments, want) {
		t.Errorf("Arguments = %v, want %v", skill.Arguments, want)
	}
}

func TestGeminiWriteSkill_Basic(t *testing.T) {
	dir := t.TempDir()
	p := NewGeminiProvider(WithGeminiBaseDir(dir))

	skill := Skill{
		Name:        "deploy",
		Description: "Deploy to production",
		Content:     "Run the deploy script",
	}
	if err := p.WriteSkill(skill); err != nil {
		t.Fatalf("WriteSkill() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "deploy.toml"))
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "Deploy to production") {
		t.Errorf("file should contain description, got: %s", content)
	}
	if !strings.Contains(content, "Run the deploy script") {
		t.Errorf("file should contain prompt, got: %s", content)
	}
}

func TestGeminiWriteSkill_NoDescription(t *testing.T) {
	dir := t.TempDir()
	p := NewGeminiProvider(WithGeminiBaseDir(dir))

	skill := Skill{
		Name:    "plain",
		Content: "Just do it",
	}
	if err := p.WriteSkill(skill); err != nil {
		t.Fatalf("WriteSkill() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "plain.toml"))
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	content := string(data)
	// description should be omitted (omitempty)
	if strings.Contains(content, "description") {
		t.Errorf("file should not contain description field when empty, got: %s", content)
	}
	if !strings.Contains(content, "Just do it") {
		t.Errorf("file should contain prompt, got: %s", content)
	}
}

func TestGeminiWriteSkill_CreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new", "nested", "dir")
	p := NewGeminiProvider(WithGeminiBaseDir(dir))

	skill := Skill{
		Name:    "test",
		Content: "Hello world",
	}
	if err := p.WriteSkill(skill); err != nil {
		t.Fatalf("WriteSkill() error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "test.toml")); err != nil {
		t.Errorf("expected file to exist: %v", err)
	}
}

func TestGeminiWriteSkill_Namespaced(t *testing.T) {
	dir := t.TempDir()
	p := NewGeminiProvider(WithGeminiBaseDir(dir))

	skill := Skill{
		Name:        "git:commit",
		Description: "Commit changes",
		Content:     "Stage and commit",
	}
	if err := p.WriteSkill(skill); err != nil {
		t.Fatalf("WriteSkill() error: %v", err)
	}

	path := filepath.Join(dir, "git", "commit.toml")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file at %s: %v", path, err)
	}
}

func TestGeminiWriteSkill_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := NewGeminiProvider(WithGeminiBaseDir(dir))

	original := Skill{
		Name:        "roundtrip",
		Description: "Test round trip",
		Content:     "Search for {{args}} in the codebase",
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
	wantArgs := []string{"{{args}}"}
	if !reflect.DeepEqual(got.Arguments, wantArgs) {
		t.Errorf("Arguments = %v, want %v", got.Arguments, wantArgs)
	}
	if got.SourcePath == "" {
		t.Error("SourcePath should not be empty after ReadSkill")
	}
}

func TestGeminiWriteSkill_RoundTrip_Namespaced(t *testing.T) {
	dir := t.TempDir()
	p := NewGeminiProvider(WithGeminiBaseDir(dir))

	original := Skill{
		Name:        "git:status",
		Description: "Show git status",
		Content:     "Run git status {{args}}",
	}
	if err := p.WriteSkill(original); err != nil {
		t.Fatalf("WriteSkill() error: %v", err)
	}

	got, err := p.ReadSkill("git:status")
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

	// Verify file is in the correct subdirectory
	expectedPath := filepath.Join(dir, "git", "status.toml")
	if !strings.HasSuffix(got.SourcePath, filepath.Join("git", "status.toml")) {
		t.Errorf("SourcePath = %q, want to end with git/status.toml", got.SourcePath)
	}
	if _, err := os.Stat(expectedPath); err != nil {
		t.Errorf("expected file at %s: %v", expectedPath, err)
	}
}

func TestGeminiListSkills_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	writeTestTOML(t, dir, "alpha.toml", "prompt = \"Alpha prompt\"\n")
	writeTestTOML(t, dir, "beta.toml", "prompt = \"Beta prompt\"\n")
	writeTestTOML(t, dir, "gamma.toml", "prompt = \"Gamma prompt\"\n")

	p := NewGeminiProvider(WithGeminiBaseDir(dir))
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
	p := NewGeminiProvider(WithGeminiBaseDir(dir))
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
	p := NewGeminiProvider(WithGeminiBaseDir(dir))
	_, err := p.ListSkills()
	if err == nil {
		t.Error("ListSkills() expected error for nonexistent dir, got nil")
	}
}

func TestGeminiListSkills_WithSubdirs(t *testing.T) {
	dir := t.TempDir()
	writeTestTOML(t, dir, "git/commit.toml", "prompt = \"Commit changes\"\n")
	writeTestTOML(t, dir, "git/push.toml", "prompt = \"Push to remote\"\n")

	p := NewGeminiProvider(WithGeminiBaseDir(dir))
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
	if !names["git:commit"] {
		t.Error("ListSkills() missing skill 'git:commit'")
	}
	if !names["git:push"] {
		t.Error("ListSkills() missing skill 'git:push'")
	}
}

func TestGeminiListSkills_MixedDepths(t *testing.T) {
	dir := t.TempDir()
	writeTestTOML(t, dir, "deploy.toml", "prompt = \"Deploy\"\n")
	writeTestTOML(t, dir, "git/commit.toml", "prompt = \"Commit\"\n")
	writeTestTOML(t, dir, "docker/build/image.toml", "prompt = \"Build image\"\n")

	p := NewGeminiProvider(WithGeminiBaseDir(dir))
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
	for _, want := range []string{"deploy", "git:commit", "docker:build:image"} {
		if !names[want] {
			t.Errorf("ListSkills() missing skill %q", want)
		}
	}
}

// Path traversal protection tests

func TestGeminiReadSkill_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	p := NewGeminiProvider(WithGeminiBaseDir(dir))

	names := []string{"../../etc:passwd", "..:secret", "foo:..:..:bar"}
	for _, name := range names {
		_, err := p.ReadSkill(name)
		if err == nil {
			t.Errorf("ReadSkill(%q) expected error for path traversal, got nil", name)
		}
	}
}

func TestGeminiWriteSkill_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	p := NewGeminiProvider(WithGeminiBaseDir(dir))

	names := []string{"../../etc:passwd", "..:secret", "foo:..:..:bar"}
	for _, name := range names {
		err := p.WriteSkill(Skill{Name: name, Content: "malicious"})
		if err == nil {
			t.Errorf("WriteSkill(%q) expected error for path traversal, got nil", name)
		}
	}
}

func TestGeminiReadSkill_EmptyName(t *testing.T) {
	dir := t.TempDir()
	p := NewGeminiProvider(WithGeminiBaseDir(dir))
	_, err := p.ReadSkill("")
	if err == nil {
		t.Error("ReadSkill(\"\") expected error for empty name, got nil")
	}
}

func TestGeminiWriteSkill_EmptyName(t *testing.T) {
	dir := t.TempDir()
	p := NewGeminiProvider(WithGeminiBaseDir(dir))
	err := p.WriteSkill(Skill{Name: "", Content: "test"})
	if err == nil {
		t.Error("WriteSkill(\"\") expected error for empty name, got nil")
	}
}

func TestGeminiExtractArgs_Deduplication(t *testing.T) {
	prompt := "Use {{args}} then {{args}} and {{query}} then {{query}}"
	got := geminiExtractArgs(prompt)
	want := []string{"{{args}}", "{{query}}"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("geminiExtractArgs() = %v, want %v (deduplicated)", got, want)
	}
}

func TestGeminiExtractArgs_NoArgs(t *testing.T) {
	prompt := "No placeholders here"
	got := geminiExtractArgs(prompt)
	if got != nil {
		t.Errorf("geminiExtractArgs() = %v, want nil", got)
	}
}
