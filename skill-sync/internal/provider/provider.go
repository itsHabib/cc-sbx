// Package provider defines the core abstractions for skill-sync providers.
//
// Every AI assistant provider (Claude Code, GitHub Copilot, Gemini CLI, etc.)
// implements the Provider interface. Skills are normalized into the Skill struct
// for cross-provider comparison and syncing.
package provider

import "fmt"

// Skill is a normalized representation of an AI assistant skill/command.
// It is a value type — copying a Skill is always safe.
type Skill struct {
	// Name is derived from the filename (without extension).
	Name string

	// Description is the first-line description if present (e.g., "Deploy to prod").
	Description string

	// Content is the full skill content including the description line.
	Content string

	// Arguments contains extracted argument placeholders (e.g., "$ARGUMENTS", "${PROJECT}").
	Arguments []string

	// SourcePath is the absolute path to the source file on disk.
	// May be empty for skills constructed programmatically.
	SourcePath string
}

// Provider is the contract for reading and writing skills from an AI assistant.
type Provider interface {
	// Name returns the provider's unique identifier (e.g., "claude", "copilot").
	Name() string

	// ListSkills returns all skills found in the provider's skill directory.
	ListSkills() ([]Skill, error)

	// ReadSkill reads a single skill by name.
	// Returns an error wrapping os.ErrNotExist if the skill is not found.
	ReadSkill(name string) (*Skill, error)

	// WriteSkill writes (or overwrites) a skill to the provider's skill directory.
	WriteSkill(skill Skill) error

	// SkillDir returns the absolute path to the provider's skill directory.
	SkillDir() string
}

// SkillStatus represents the drift status of a skill between source and target.
type SkillStatus int

const (
	// InSync means source and target content match.
	InSync SkillStatus = iota
	// Modified means both exist but content differs.
	Modified
	// MissingInTarget means the skill exists in source but is absent in the target.
	MissingInTarget
	// ExtraInTarget means the skill is absent in source but exists in the target.
	ExtraInTarget
)

// String returns the human-readable status label.
func (s SkillStatus) String() string {
	switch s {
	case InSync:
		return "in-sync"
	case Modified:
		return "modified"
	case MissingInTarget:
		return "missing-in-target"
	case ExtraInTarget:
		return "extra-in-target"
	default:
		return fmt.Sprintf("unknown(%d)", int(s))
	}
}
