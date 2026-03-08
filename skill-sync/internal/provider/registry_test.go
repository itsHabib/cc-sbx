package provider

import (
	"testing"
)

func TestRegister_And_Get(t *testing.T) {
	resetRegistry()

	Register("test-provider", func(baseDir string) Provider {
		return &skillMDProvider{providerName: "test-provider", baseDir: "/tmp/test-provider"}
	})

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

	Register("dup", func(baseDir string) Provider {
		return &skillMDProvider{providerName: "dup", baseDir: "/tmp/dup"}
	})

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("Register should panic on duplicate name")
		}
	}()

	Register("dup", func(baseDir string) Provider {
		return &skillMDProvider{providerName: "dup", baseDir: "/tmp/dup"}
	})
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

	for _, name := range []string{"zebra", "alpha", "middle"} {
		n := name
		Register(n, func(baseDir string) Provider {
			return &skillMDProvider{providerName: n, baseDir: "/tmp/" + n}
		})
	}

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

func TestNew_WithBaseDir(t *testing.T) {
	resetRegistry()

	Register("test", func(baseDir string) Provider {
		if baseDir == "" {
			baseDir = "/default/path"
		}
		return &skillMDProvider{providerName: "test", baseDir: baseDir}
	})

	// Default dir
	p, err := Get("test")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if p.SkillDir() != "/default/path" {
		t.Errorf("SkillDir() = %q, want /default/path", p.SkillDir())
	}

	// Custom dir
	p2, err := New("test", "/custom/path")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if p2.SkillDir() != "/custom/path" {
		t.Errorf("SkillDir() = %q, want /custom/path", p2.SkillDir())
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
