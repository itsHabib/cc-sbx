package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/user/skill-sync/internal/provider"
	"github.com/user/skill-sync/internal/sync"
)

func TestStatusAllInSync(t *testing.T) {
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
	err := doStatus(&buf, engine, nil)
	if err != nil {
		t.Fatalf("expected no error for all in-sync, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[ok] in-sync") {
		t.Errorf("expected '[ok] in-sync' in output, got: %s", output)
	}
}

func TestStatusMixedDrift(t *testing.T) {
	source := newMockSource(
		provider.Skill{Name: "deploy", Content: "deploy content"},
		provider.Skill{Name: "review", Content: "review v2"},
	)
	target := &mockProvider{
		name:   "copilot",
		skills: []provider.Skill{{Name: "deploy"}, {Name: "review"}},
		readMap: map[string]*provider.Skill{
			"deploy": {Name: "deploy", Content: "deploy content"},
			"review": {Name: "review", Content: "review v1"},
		},
	}

	engine := sync.NewDiffEngine(source, []provider.Provider{target})

	var buf bytes.Buffer
	err := doStatus(&buf, engine, nil)
	if err == nil {
		t.Fatal("expected error for drift detected")
	}

	output := buf.String()
	if !strings.Contains(output, "[ok] in-sync") {
		t.Error("expected '[ok] in-sync' for deploy")
	}
	if !strings.Contains(output, "[!] modified") {
		t.Error("expected '[!] modified' for review")
	}
}

func TestStatusMissingInTarget(t *testing.T) {
	source := newMockSource(
		provider.Skill{Name: "deploy", Content: "deploy content"},
	)
	target := &mockProvider{
		name:    "copilot",
		skills:  []provider.Skill{},
		readMap: map[string]*provider.Skill{},
	}

	engine := sync.NewDiffEngine(source, []provider.Provider{target})

	var buf bytes.Buffer
	err := doStatus(&buf, engine, nil)
	if err == nil {
		t.Fatal("expected error for drift detected")
	}

	output := buf.String()
	if !strings.Contains(output, "[-] missing") {
		t.Errorf("expected '[-] missing' in output, got: %s", output)
	}
}

func TestStatusExtraInTarget(t *testing.T) {
	source := newMockSource()
	target := &mockProvider{
		name:   "copilot",
		skills: []provider.Skill{{Name: "extra-skill"}},
		readMap: map[string]*provider.Skill{
			"extra-skill": {Name: "extra-skill", Content: "extra"},
		},
	}

	engine := sync.NewDiffEngine(source, []provider.Provider{target})

	var buf bytes.Buffer
	err := doStatus(&buf, engine, nil)
	if err == nil {
		t.Fatal("expected error for drift detected")
	}

	output := buf.String()
	if !strings.Contains(output, "[+] extra") {
		t.Errorf("expected '[+] extra' in output, got: %s", output)
	}
}

func TestStatusMultipleTargets(t *testing.T) {
	source := newMockSource(
		provider.Skill{Name: "deploy", Content: "deploy content"},
	)
	target1 := &mockProvider{
		name:   "copilot",
		skills: []provider.Skill{{Name: "deploy"}},
		readMap: map[string]*provider.Skill{
			"deploy": {Name: "deploy", Content: "deploy content"},
		},
	}
	target2 := &mockProvider{
		name:    "gemini",
		skills:  []provider.Skill{},
		readMap: map[string]*provider.Skill{},
	}

	engine := sync.NewDiffEngine(source, []provider.Provider{target1, target2})

	var buf bytes.Buffer
	err := doStatus(&buf, engine, nil)
	if err == nil {
		t.Fatal("expected error for drift detected")
	}

	output := buf.String()
	if !strings.Contains(output, "Target: copilot") {
		t.Error("expected 'Target: copilot' in output")
	}
	if !strings.Contains(output, "Target: gemini") {
		t.Error("expected 'Target: gemini' in output")
	}
}
