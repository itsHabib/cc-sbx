package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/user/skill-sync/internal/provider"
	"github.com/user/skill-sync/internal/sync"
)

func TestDiffSpecificTarget(t *testing.T) {
	source := newMockSource(
		provider.Skill{Name: "review", Content: "line1\noriginal line\nline3\n"},
	)
	target := &mockProvider{
		name:   "copilot",
		skills: []provider.Skill{{Name: "review"}},
		readMap: map[string]*provider.Skill{
			"review": {Name: "review", Content: "line1\nmodified line\nline3\n"},
		},
	}

	engine := sync.NewDiffEngine(source, []provider.Provider{target})

	var buf bytes.Buffer
	err := doDiff(&buf, engine, "copilot", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "--- a/review") {
		t.Error("expected '--- a/review' in diff output")
	}
	if !strings.Contains(output, "+++ b/review") {
		t.Error("expected '+++ b/review' in diff output")
	}
	if !strings.Contains(output, "-original line") {
		t.Error("expected '-original line' in diff output")
	}
	if !strings.Contains(output, "+modified line") {
		t.Error("expected '+modified line' in diff output")
	}
}

func TestDiffAllTargets(t *testing.T) {
	source := newMockSource(
		provider.Skill{Name: "review", Content: "original\n"},
	)
	target1 := &mockProvider{
		name:   "copilot",
		skills: []provider.Skill{{Name: "review"}},
		readMap: map[string]*provider.Skill{
			"review": {Name: "review", Content: "modified1\n"},
		},
	}
	target2 := &mockProvider{
		name:   "gemini",
		skills: []provider.Skill{{Name: "review"}},
		readMap: map[string]*provider.Skill{
			"review": {Name: "review", Content: "modified2\n"},
		},
	}

	engine := sync.NewDiffEngine(source, []provider.Provider{target1, target2})

	var buf bytes.Buffer
	err := doDiffAll(&buf, engine, []string{"copilot", "gemini"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "--- a/review") {
		t.Error("expected diff output for review skill")
	}
}

func TestDiffUnknownTarget(t *testing.T) {
	source := newMockSource()
	engine := sync.NewDiffEngine(source, nil)

	var buf bytes.Buffer
	err := doDiff(&buf, engine, "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for unknown target")
	}
}

func TestDiffNoModifiedSkills(t *testing.T) {
	source := newMockSource(
		provider.Skill{Name: "deploy", Content: "deploy content"},
	)
	target := &mockProvider{
		name:   "copilot",
		skills: []provider.Skill{{Name: "deploy"}},
		readMap: map[string]*provider.Skill{
			"deploy": {Name: "deploy", Content: "deploy content"},
		},
	}

	engine := sync.NewDiffEngine(source, []provider.Provider{target})

	var buf bytes.Buffer
	err := doDiff(&buf, engine, "copilot", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if output != "" {
		t.Errorf("expected empty diff output for in-sync skills, got: %s", output)
	}
}
