package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/user/skill-sync/internal/provider"
	"github.com/user/skill-sync/internal/sync"
)

// mockProvider implements provider.Provider for testing.
type mockProvider struct {
	name      string
	skills    []provider.Skill
	readMap   map[string]*provider.Skill
	writeErr  error
	listErr   error
	written   []provider.Skill
}

func (m *mockProvider) Name() string { return m.name }

func (m *mockProvider) ListSkills() ([]provider.Skill, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.skills, nil
}

func (m *mockProvider) ReadSkill(name string) (*provider.Skill, error) {
	if s, ok := m.readMap[name]; ok {
		return s, nil
	}
	return nil, errors.New("skill not found: " + name)
}

func (m *mockProvider) WriteSkill(skill provider.Skill) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	m.written = append(m.written, skill)
	return nil
}

func (m *mockProvider) SkillDir() string { return "/mock/" + m.name }

func newMockSource(skills ...provider.Skill) *mockProvider {
	readMap := make(map[string]*provider.Skill)
	for i := range skills {
		s := skills[i]
		readMap[s.Name] = &s
	}
	return &mockProvider{
		name:    "source",
		skills:  skills,
		readMap: readMap,
	}
}

func newMockTarget(name string) *mockProvider {
	return &mockProvider{
		name:    name,
		readMap: make(map[string]*provider.Skill),
	}
}

func TestSyncAllSkills(t *testing.T) {
	source := newMockSource(
		provider.Skill{Name: "deploy", Content: "deploy content"},
		provider.Skill{Name: "review", Content: "review content"},
	)
	target := newMockTarget("copilot")

	engine := sync.NewSyncEngine(source, []provider.Provider{target})

	var buf bytes.Buffer
	err := doSync(&buf, engine, nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "deploy") {
		t.Error("expected deploy in output")
	}
	if !strings.Contains(output, "review") {
		t.Error("expected review in output")
	}
	if !strings.Contains(output, "synced") {
		t.Error("expected 'synced' status in output")
	}
	if !strings.Contains(output, "Synced: 2") {
		t.Errorf("expected 'Synced: 2' in output, got: %s", output)
	}
}

func TestSyncDryRun(t *testing.T) {
	source := newMockSource(
		provider.Skill{Name: "deploy", Content: "deploy content"},
	)
	target := newMockTarget("copilot")

	var buf bytes.Buffer
	err := doSyncDryRun(&buf, source, []provider.Provider{target}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "would sync") {
		t.Error("expected 'would sync' in output")
	}
	if len(target.written) != 0 {
		t.Error("dry-run should not write to target")
	}
}

func TestSyncWithSkillFilter(t *testing.T) {
	source := newMockSource(
		provider.Skill{Name: "deploy", Content: "deploy content"},
		provider.Skill{Name: "review", Content: "review content"},
		provider.Skill{Name: "build", Content: "build content"},
	)
	target := newMockTarget("copilot")

	engine := sync.NewSyncEngine(source, []provider.Provider{target})

	var buf bytes.Buffer
	err := doSync(&buf, engine, []string{"deploy"}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "deploy") {
		t.Error("expected deploy in output")
	}
	if strings.Contains(output, "review") {
		t.Error("review should be filtered out")
	}
	if !strings.Contains(output, "Synced: 1") {
		t.Errorf("expected 'Synced: 1' in output, got: %s", output)
	}
}

func TestSyncWithWriteErrors(t *testing.T) {
	source := newMockSource(
		provider.Skill{Name: "deploy", Content: "deploy content"},
	)
	target := &mockProvider{
		name:     "copilot",
		writeErr: errors.New("permission denied"),
		readMap:  make(map[string]*provider.Skill),
	}

	engine := sync.NewSyncEngine(source, []provider.Provider{target})

	var buf bytes.Buffer
	err := doSync(&buf, engine, nil, true)
	if err == nil {
		t.Fatal("expected error for write failure")
	}

	output := buf.String()
	if !strings.Contains(output, "error") {
		t.Error("expected 'error' in output")
	}
	if !strings.Contains(output, "Errors: 1") {
		t.Errorf("expected 'Errors: 1' in output, got: %s", output)
	}
}

func TestSyncEmptySource(t *testing.T) {
	source := newMockSource()
	target := newMockTarget("copilot")

	engine := sync.NewSyncEngine(source, []provider.Provider{target})

	var buf bytes.Buffer
	err := doSync(&buf, engine, nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Synced: 0") {
		t.Errorf("expected 'Synced: 0' in output, got: %s", output)
	}
}

func TestSyncDryRunWithFilter(t *testing.T) {
	source := newMockSource(
		provider.Skill{Name: "deploy", Content: "deploy content"},
		provider.Skill{Name: "review", Content: "review content"},
	)
	target := newMockTarget("copilot")

	var buf bytes.Buffer
	err := doSyncDryRun(&buf, source, []provider.Provider{target}, []string{"deploy"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "deploy") {
		t.Error("expected deploy in output")
	}
	if strings.Contains(output, "review") {
		t.Error("review should be filtered out")
	}
}
