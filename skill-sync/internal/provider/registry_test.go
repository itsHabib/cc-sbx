package provider

import (
	"testing"
)

// mockProvider implements Provider for testing.
type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string                        { return m.name }
func (m *mockProvider) ListSkills() ([]Skill, error)        { return nil, nil }
func (m *mockProvider) ReadSkill(name string) (*Skill, error) { return nil, nil }
func (m *mockProvider) WriteSkill(skill Skill) error        { return nil }
func (m *mockProvider) SkillDir() string                    { return "/tmp/" + m.name }

func TestRegister_And_Get(t *testing.T) {
	resetRegistry()

	p := &mockProvider{name: "test-provider"}
	Register(p)

	got, err := Get("test-provider")
	if err != nil {
		t.Fatalf("Get returned unexpected error: %v", err)
	}
	if got.Name() != "test-provider" {
		t.Errorf("got Name() = %q, want %q", got.Name(), "test-provider")
	}
}

func TestGet_Unknown(t *testing.T) {
	resetRegistry()

	_, err := Get("nonexistent")
	if err == nil {
		t.Fatal("Get should return error for unknown provider")
	}
}

func TestRegister_Duplicate_Panics(t *testing.T) {
	resetRegistry()

	Register(&mockProvider{name: "dup"})

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("Register should panic on duplicate name")
		}
	}()

	Register(&mockProvider{name: "dup"})
}

func TestList_Empty(t *testing.T) {
	resetRegistry()

	names := List()
	if len(names) != 0 {
		t.Errorf("List should return empty slice, got %v", names)
	}
}

func TestList_Sorted(t *testing.T) {
	resetRegistry()

	Register(&mockProvider{name: "zebra"})
	Register(&mockProvider{name: "alpha"})
	Register(&mockProvider{name: "middle"})

	names := List()
	expected := []string{"alpha", "middle", "zebra"}

	if len(names) != len(expected) {
		t.Fatalf("List returned %d names, want %d", len(names), len(expected))
	}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("List()[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

func TestSkillStatus_String(t *testing.T) {
	tests := []struct {
		status SkillStatus
		want   string
	}{
		{InSync, "in-sync"},
		{Modified, "modified"},
		{MissingInTarget, "missing-in-target"},
		{ExtraInTarget, "extra-in-target"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.status.String()
			if got != tt.want {
				t.Errorf("SkillStatus(%d).String() = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestMockProvider_Implements_Interface(t *testing.T) {
	resetRegistry()

	// Verify mockProvider satisfies the Provider interface at compile time.
	var _ Provider = &mockProvider{}

	p := &mockProvider{name: "integration-test"}
	Register(p)

	got, err := Get("integration-test")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	skills, err := got.ListSkills()
	if err != nil {
		t.Fatalf("ListSkills returned error: %v", err)
	}
	if skills != nil {
		t.Errorf("expected nil skills from mock, got %v", skills)
	}

	skill, err := got.ReadSkill("anything")
	if err != nil {
		t.Fatalf("ReadSkill returned error: %v", err)
	}
	if skill != nil {
		t.Errorf("expected nil skill from mock, got %v", skill)
	}

	err = got.WriteSkill(Skill{Name: "test"})
	if err != nil {
		t.Fatalf("WriteSkill returned error: %v", err)
	}

	dir := got.SkillDir()
	if dir != "/tmp/integration-test" {
		t.Errorf("SkillDir() = %q, want %q", dir, "/tmp/integration-test")
	}
}
