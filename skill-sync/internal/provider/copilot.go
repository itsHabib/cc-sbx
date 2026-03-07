package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CopilotOption configures a CopilotProvider.
type CopilotOption func(*CopilotProvider)

// WithCopilotBaseDir returns a CopilotOption that sets the base directory for prompt files.
func WithCopilotBaseDir(dir string) CopilotOption {
	return func(p *CopilotProvider) {
		p.baseDir = dir
	}
}

// CopilotProvider reads and writes skills from GitHub Copilot's prompt files directory.
type CopilotProvider struct {
	baseDir string
}

// NewCopilotProvider creates a CopilotProvider. By default it uses .github/prompts/ (project-level).
func NewCopilotProvider(opts ...CopilotOption) *CopilotProvider {
	p := &CopilotProvider{
		baseDir: filepath.Join(".github", "prompts"),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Name returns "copilot".
func (p *CopilotProvider) Name() string {
	return "copilot"
}

// SkillDir returns the base directory where Copilot prompt files are stored.
func (p *CopilotProvider) SkillDir() string {
	return p.baseDir
}

// ListSkills reads all *.prompt.md files in the base directory and returns them as skills.
func (p *CopilotProvider) ListSkills() ([]Skill, error) {
	pattern := filepath.Join(p.baseDir, "*.prompt.md")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("copilot: glob skills: %w", err)
	}

	if len(matches) == 0 {
		if _, err := os.Stat(p.baseDir); err != nil {
			return nil, fmt.Errorf("copilot: read skill directory: %w", err)
		}
		return []Skill{}, nil
	}

	skills := make([]Skill, 0, len(matches))
	for _, path := range matches {
		name := strings.TrimSuffix(filepath.Base(path), ".prompt.md")
		skill, err := p.readSkillFile(name, path)
		if err != nil {
			return nil, fmt.Errorf("copilot: list skills: %w", err)
		}
		skills = append(skills, *skill)
	}
	return skills, nil
}

// ReadSkill reads a single skill by name from the base directory.
func (p *CopilotProvider) ReadSkill(name string) (*Skill, error) {
	path := filepath.Join(p.baseDir, name+".prompt.md")
	skill, err := p.readSkillFile(name, path)
	if err != nil {
		return nil, fmt.Errorf("copilot: read skill %q: %w", name, err)
	}
	return skill, nil
}

// WriteSkill writes a skill to the base directory. Creates the directory if needed.
func (p *CopilotProvider) WriteSkill(skill Skill) error {
	if err := os.MkdirAll(p.baseDir, 0755); err != nil {
		return fmt.Errorf("copilot: create skill directory: %w", err)
	}

	path := filepath.Join(p.baseDir, skill.Name+".prompt.md")
	if err := os.WriteFile(path, []byte(skill.Content), 0644); err != nil {
		return fmt.Errorf("copilot: write skill %q: %w", skill.Name, err)
	}
	return nil
}

// readSkillFile reads and parses a single .prompt.md file into a Skill.
func (p *CopilotProvider) readSkillFile(name, path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve absolute path: %w", err)
	}

	skill := &Skill{
		Name:       name,
		Content:    content,
		SourcePath: absPath,
	}

	// Extract description from first line if it starts with "# ".
	if lines := strings.SplitN(content, "\n", 2); len(lines) > 0 {
		firstLine := lines[0]
		if strings.HasPrefix(firstLine, "# ") {
			skill.Description = strings.TrimPrefix(firstLine, "# ")
		}
	}

	return skill, nil
}

func init() {
	Register(NewCopilotProvider())
}
