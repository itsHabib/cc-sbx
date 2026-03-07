package sync

import (
	"errors"
	"strings"
	"testing"

	"github.com/user/skill-sync/internal/provider"
)

// mockProvider implements provider.Provider with in-memory skill storage.
type mockProvider struct {
	name      string
	skills    map[string]provider.Skill
	writeErr  map[string]error // per-skill write errors
	listErr   error            // if set, ListSkills returns this error
	readErr   map[string]error // per-skill read errors
}

func newMockProvider(name string, skills ...provider.Skill) *mockProvider {
	m := &mockProvider{
		name:     name,
		skills:   make(map[string]provider.Skill),
		writeErr: make(map[string]error),
		readErr:  make(map[string]error),
	}
	for _, s := range skills {
		m.skills[s.Name] = s
	}
	return m
}

func (m *mockProvider) Name() string { return m.name }

func (m *mockProvider) ListSkills() ([]provider.Skill, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var skills []provider.Skill
	for _, s := range m.skills {
		skills = append(skills, s)
	}
	return skills, nil
}

func (m *mockProvider) ReadSkill(name string) (*provider.Skill, error) {
	if err, ok := m.readErr[name]; ok {
		return nil, err
	}
	s, ok := m.skills[name]
	if !ok {
		return nil, errors.New("skill not found: " + name)
	}
	return &s, nil
}

func (m *mockProvider) WriteSkill(skill provider.Skill) error {
	if err, ok := m.writeErr[skill.Name]; ok {
		return err
	}
	m.skills[skill.Name] = skill
	return nil
}

func (m *mockProvider) SkillDir() string { return "/mock/" + m.name }

// --- SyncEngine Tests ---

func TestSync_AllSkills(t *testing.T) {
	source := newMockProvider("source",
		provider.Skill{Name: "a", Content: "content-a"},
		provider.Skill{Name: "b", Content: "content-b"},
		provider.Skill{Name: "c", Content: "content-c"},
	)
	target := newMockProvider("target1")

	engine := NewSyncEngine(source, []provider.Provider{target})
	result, err := engine.Sync(nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalSynced != 3 {
		t.Errorf("TotalSynced = %d, want 3", result.TotalSynced)
	}
	if result.TotalErrored != 0 {
		t.Errorf("TotalErrored = %d, want 0", result.TotalErrored)
	}
	if len(target.skills) != 3 {
		t.Errorf("target has %d skills, want 3", len(target.skills))
	}
}

func TestSync_WithFilter(t *testing.T) {
	source := newMockProvider("source",
		provider.Skill{Name: "a", Content: "content-a"},
		provider.Skill{Name: "b", Content: "content-b"},
		provider.Skill{Name: "c", Content: "content-c"},
	)
	target := newMockProvider("target1")

	engine := NewSyncEngine(source, []provider.Provider{target})
	result, err := engine.Sync([]string{"a", "b"}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalSynced != 2 {
		t.Errorf("TotalSynced = %d, want 2", result.TotalSynced)
	}
	if len(target.skills) != 2 {
		t.Errorf("target has %d skills, want 2", len(target.skills))
	}
	if _, ok := target.skills["c"]; ok {
		t.Error("skill 'c' should not have been synced")
	}
}

func TestSync_EmptySource(t *testing.T) {
	source := newMockProvider("source")
	target := newMockProvider("target1")

	engine := NewSyncEngine(source, []provider.Provider{target})
	result, err := engine.Sync(nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalSynced != 0 {
		t.Errorf("TotalSynced = %d, want 0", result.TotalSynced)
	}
	if result.TotalErrored != 0 {
		t.Errorf("TotalErrored = %d, want 0", result.TotalErrored)
	}
}

func TestSync_MultiTarget(t *testing.T) {
	source := newMockProvider("source",
		provider.Skill{Name: "a", Content: "content-a"},
		provider.Skill{Name: "b", Content: "content-b"},
	)
	target1 := newMockProvider("target1")
	target2 := newMockProvider("target2")

	engine := NewSyncEngine(source, []provider.Provider{target1, target2})
	result, err := engine.Sync(nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalSynced != 4 {
		t.Errorf("TotalSynced = %d, want 4", result.TotalSynced)
	}
	if len(target1.skills) != 2 {
		t.Errorf("target1 has %d skills, want 2", len(target1.skills))
	}
	if len(target2.skills) != 2 {
		t.Errorf("target2 has %d skills, want 2", len(target2.skills))
	}
}

func TestSync_WriteError(t *testing.T) {
	source := newMockProvider("source",
		provider.Skill{Name: "a", Content: "content-a"},
		provider.Skill{Name: "b", Content: "content-b"},
	)
	target := newMockProvider("target1")
	target.writeErr["a"] = errors.New("permission denied")

	engine := NewSyncEngine(source, []provider.Provider{target})
	result, err := engine.Sync(nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalErrored != 1 {
		t.Errorf("TotalErrored = %d, want 1", result.TotalErrored)
	}
	// "b" should still have been synced
	if _, ok := target.skills["b"]; !ok {
		t.Error("skill 'b' should have been synced despite 'a' error")
	}
}

func TestSync_SourceListError(t *testing.T) {
	source := newMockProvider("source")
	source.listErr = errors.New("directory not found")
	target := newMockProvider("target1")

	engine := NewSyncEngine(source, []provider.Provider{target})
	_, err := engine.Sync(nil, true)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "list source skills") {
		t.Errorf("error = %q, want it to mention 'list source skills'", err.Error())
	}
}

func TestSync_ReadSourceError(t *testing.T) {
	source := newMockProvider("source",
		provider.Skill{Name: "a", Content: "content-a"},
		provider.Skill{Name: "b", Content: "content-b"},
	)
	source.readErr["a"] = errors.New("corrupt file")
	target := newMockProvider("target1")

	engine := NewSyncEngine(source, []provider.Provider{target})
	result, err := engine.Sync(nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalErrored != 1 {
		t.Errorf("TotalErrored = %d, want 1", result.TotalErrored)
	}
	// "b" should still have been synced
	if _, ok := target.skills["b"]; !ok {
		t.Error("skill 'b' should have been synced despite 'a' read error")
	}
}

// --- DiffEngine Tests ---

func TestStatus_AllInSync(t *testing.T) {
	source := newMockProvider("source",
		provider.Skill{Name: "a", Content: "same"},
		provider.Skill{Name: "b", Content: "same-b"},
		provider.Skill{Name: "c", Content: "same-c"},
	)
	target := newMockProvider("target1",
		provider.Skill{Name: "a", Content: "same"},
		provider.Skill{Name: "b", Content: "same-b"},
		provider.Skill{Name: "c", Content: "same-c"},
	)

	engine := NewDiffEngine(source, []provider.Provider{target})
	report, err := engine.Status()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	drifts := report.Results["target1"]
	if len(drifts) != 3 {
		t.Fatalf("got %d drifts, want 3", len(drifts))
	}
	for _, d := range drifts {
		if d.Status != provider.InSync {
			t.Errorf("skill %q status = %v, want InSync", d.SkillName, d.Status)
		}
	}
}

func TestStatus_Modified(t *testing.T) {
	source := newMockProvider("source",
		provider.Skill{Name: "a", Content: "original"},
	)
	target := newMockProvider("target1",
		provider.Skill{Name: "a", Content: "changed"},
	)

	engine := NewDiffEngine(source, []provider.Provider{target})
	report, err := engine.Status()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	drifts := report.Results["target1"]
	if len(drifts) != 1 {
		t.Fatalf("got %d drifts, want 1", len(drifts))
	}
	if drifts[0].Status != provider.Modified {
		t.Errorf("status = %v, want Modified", drifts[0].Status)
	}
	if drifts[0].UnifiedDiff == "" {
		t.Error("expected non-empty UnifiedDiff for modified skill")
	}
}

func TestStatus_MissingInTarget(t *testing.T) {
	source := newMockProvider("source",
		provider.Skill{Name: "a", Content: "a"},
		provider.Skill{Name: "b", Content: "b"},
		provider.Skill{Name: "c", Content: "c"},
	)
	target := newMockProvider("target1",
		provider.Skill{Name: "a", Content: "a"},
		provider.Skill{Name: "b", Content: "b"},
	)

	engine := NewDiffEngine(source, []provider.Provider{target})
	report, err := engine.Status()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	drifts := report.Results["target1"]
	found := false
	for _, d := range drifts {
		if d.SkillName == "c" {
			found = true
			if d.Status != provider.MissingInTarget {
				t.Errorf("skill 'c' status = %v, want MissingInTarget", d.Status)
			}
		}
	}
	if !found {
		t.Error("skill 'c' not found in drift report")
	}
}

func TestStatus_ExtraInTarget(t *testing.T) {
	source := newMockProvider("source",
		provider.Skill{Name: "a", Content: "a"},
	)
	target := newMockProvider("target1",
		provider.Skill{Name: "a", Content: "a"},
		provider.Skill{Name: "b", Content: "b"},
	)

	engine := NewDiffEngine(source, []provider.Provider{target})
	report, err := engine.Status()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	drifts := report.Results["target1"]
	found := false
	for _, d := range drifts {
		if d.SkillName == "b" {
			found = true
			if d.Status != provider.ExtraInTarget {
				t.Errorf("skill 'b' status = %v, want ExtraInTarget", d.Status)
			}
		}
	}
	if !found {
		t.Error("skill 'b' not found in drift report")
	}
}

func TestStatus_MixedStatuses(t *testing.T) {
	source := newMockProvider("source",
		provider.Skill{Name: "in-sync", Content: "same"},
		provider.Skill{Name: "modified", Content: "original"},
		provider.Skill{Name: "missing", Content: "exists"},
	)
	target := newMockProvider("target1",
		provider.Skill{Name: "in-sync", Content: "same"},
		provider.Skill{Name: "modified", Content: "changed"},
		provider.Skill{Name: "extra", Content: "extra"},
	)

	engine := NewDiffEngine(source, []provider.Provider{target})
	report, err := engine.Status()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	drifts := report.Results["target1"]
	statusMap := make(map[string]provider.SkillStatus)
	for _, d := range drifts {
		statusMap[d.SkillName] = d.Status
	}

	expected := map[string]provider.SkillStatus{
		"in-sync":  provider.InSync,
		"modified": provider.Modified,
		"missing":  provider.MissingInTarget,
		"extra":    provider.ExtraInTarget,
	}
	for name, want := range expected {
		got, ok := statusMap[name]
		if !ok {
			t.Errorf("skill %q not found in drift report", name)
			continue
		}
		if got != want {
			t.Errorf("skill %q status = %v, want %v", name, got, want)
		}
	}
}

func TestDiff_SpecificTarget(t *testing.T) {
	source := newMockProvider("source",
		provider.Skill{Name: "a", Content: "original-a"},
		provider.Skill{Name: "b", Content: "same-b"},
	)
	target1 := newMockProvider("target1",
		provider.Skill{Name: "a", Content: "changed-a-t1"},
		provider.Skill{Name: "b", Content: "same-b"},
	)
	target2 := newMockProvider("target2",
		provider.Skill{Name: "a", Content: "changed-a-t2"},
		provider.Skill{Name: "b", Content: "same-b"},
	)

	engine := NewDiffEngine(source, []provider.Provider{target1, target2})
	detailed, err := engine.Diff("target1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if detailed.TargetName != "target1" {
		t.Errorf("TargetName = %q, want target1", detailed.TargetName)
	}
	if len(detailed.Diffs) != 1 {
		t.Fatalf("got %d diffs, want 1", len(detailed.Diffs))
	}
	if detailed.Diffs[0].SkillName != "a" {
		t.Errorf("diff skill = %q, want 'a'", detailed.Diffs[0].SkillName)
	}
	if detailed.Diffs[0].UnifiedDiff == "" {
		t.Error("expected non-empty unified diff")
	}
}

func TestDiff_UnknownTarget(t *testing.T) {
	source := newMockProvider("source")
	target := newMockProvider("target1")

	engine := NewDiffEngine(source, []provider.Provider{target})
	_, err := engine.Diff("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown target, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error = %q, want it to mention 'nonexistent'", err.Error())
	}
}

func TestStatus_WhitespaceNormalization(t *testing.T) {
	source := newMockProvider("source",
		provider.Skill{Name: "a", Content: "content\n"},
	)
	target := newMockProvider("target1",
		provider.Skill{Name: "a", Content: "content\n\n"},
	)

	engine := NewDiffEngine(source, []provider.Provider{target})
	report, err := engine.Status()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	drifts := report.Results["target1"]
	if len(drifts) != 1 {
		t.Fatalf("got %d drifts, want 1", len(drifts))
	}
	if drifts[0].Status != provider.InSync {
		t.Errorf("status = %v, want InSync (whitespace normalization should handle trailing newlines)", drifts[0].Status)
	}
}

// --- Edge Case Tests ---

func TestSync_EmptyTargets(t *testing.T) {
	source := newMockProvider("source",
		provider.Skill{Name: "a", Content: "content-a"},
	)

	engine := NewSyncEngine(source, nil)
	result, err := engine.Sync(nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalSynced != 0 {
		t.Errorf("TotalSynced = %d, want 0", result.TotalSynced)
	}
}

func TestSync_FilterNoMatch(t *testing.T) {
	source := newMockProvider("source",
		provider.Skill{Name: "a", Content: "content-a"},
	)
	target := newMockProvider("target1")

	engine := NewSyncEngine(source, []provider.Provider{target})
	result, err := engine.Sync([]string{"nonexistent"}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalSynced != 0 {
		t.Errorf("TotalSynced = %d, want 0", result.TotalSynced)
	}
}

func TestStatus_EmptySource(t *testing.T) {
	source := newMockProvider("source")
	target := newMockProvider("target1",
		provider.Skill{Name: "extra", Content: "extra"},
	)

	engine := NewDiffEngine(source, []provider.Provider{target})
	report, err := engine.Status()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	drifts := report.Results["target1"]
	if len(drifts) != 1 {
		t.Fatalf("got %d drifts, want 1", len(drifts))
	}
	if drifts[0].Status != provider.ExtraInTarget {
		t.Errorf("status = %v, want ExtraInTarget", drifts[0].Status)
	}
}

func TestStatus_EmptyTargets(t *testing.T) {
	source := newMockProvider("source",
		provider.Skill{Name: "a", Content: "a"},
	)

	engine := NewDiffEngine(source, nil)
	report, err := engine.Status()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(report.Results) != 0 {
		t.Errorf("got %d target results, want 0", len(report.Results))
	}
}

func TestSync_LargeContent(t *testing.T) {
	largeContent := strings.Repeat("line of content\n", 1000)
	source := newMockProvider("source",
		provider.Skill{Name: "large", Content: largeContent},
	)
	target := newMockProvider("target1")

	engine := NewSyncEngine(source, []provider.Provider{target})
	result, err := engine.Sync(nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalSynced != 1 {
		t.Errorf("TotalSynced = %d, want 1", result.TotalSynced)
	}
	if target.skills["large"].Content != largeContent {
		t.Error("large content was not preserved")
	}
}

func TestDiff_UnifiedDiffFormat(t *testing.T) {
	source := newMockProvider("source",
		provider.Skill{Name: "test", Content: "line1\nline2\nline3\n"},
	)
	target := newMockProvider("target1",
		provider.Skill{Name: "test", Content: "line1\nmodified\nline3\n"},
	)

	engine := NewDiffEngine(source, []provider.Provider{target})
	detailed, err := engine.Diff("target1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(detailed.Diffs) != 1 {
		t.Fatalf("got %d diffs, want 1", len(detailed.Diffs))
	}

	diff := detailed.Diffs[0].UnifiedDiff
	if !strings.Contains(diff, "--- a/test") {
		t.Error("diff missing --- header")
	}
	if !strings.Contains(diff, "+++ b/test") {
		t.Error("diff missing +++ header")
	}
	if !strings.Contains(diff, "@@") {
		t.Error("diff missing @@ hunk header")
	}
	if !strings.Contains(diff, "-line2") {
		t.Error("diff missing deleted line")
	}
	if !strings.Contains(diff, "+modified") {
		t.Error("diff missing added line")
	}
}

func TestSync_WriteErrorIsolation(t *testing.T) {
	source := newMockProvider("source",
		provider.Skill{Name: "a", Content: "content-a"},
		provider.Skill{Name: "b", Content: "content-b"},
	)
	target1 := newMockProvider("target1")
	target1.writeErr["a"] = errors.New("target1 write error")
	target2 := newMockProvider("target2")

	engine := NewSyncEngine(source, []provider.Provider{target1, target2})
	result, err := engine.Sync(nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// a fails on target1 but succeeds on target2
	// b succeeds on both
	if result.TotalSynced != 3 {
		t.Errorf("TotalSynced = %d, want 3", result.TotalSynced)
	}
	if result.TotalErrored != 1 {
		t.Errorf("TotalErrored = %d, want 1", result.TotalErrored)
	}
	if _, ok := target2.skills["a"]; !ok {
		t.Error("skill 'a' should have been synced to target2 despite target1 error")
	}
}
