package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// factoryFrontmatter is the YAML structure for Factory skill frontmatter.
type factoryFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Model       string `yaml:"model,omitempty"`
}

// FactoryOption configures a FactoryProvider.
type FactoryOption func(*FactoryProvider)

// WithFactoryBaseDir sets the base directory for Factory skill files.
func WithFactoryBaseDir(dir string) FactoryOption {
	return func(p *FactoryProvider) {
		p.baseDir = dir
	}
}

// FactoryProvider reads and writes skills from Factory AI Droid's skill directory.
type FactoryProvider struct {
	baseDir string
}

// NewFactoryProvider creates a FactoryProvider.
// Default baseDir: .factory/skills/ (project-level, relative to cwd).
func NewFactoryProvider(opts ...FactoryOption) *FactoryProvider {
	p := &FactoryProvider{
		baseDir: filepath.Join(".factory", "skills"),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Name returns "factory".
func (p *FactoryProvider) Name() string {
	return "factory"
}

// SkillDir returns the base directory where Factory skills are stored.
func (p *FactoryProvider) SkillDir() string {
	return p.baseDir
}

// ListSkills scans baseDir for subdirectories containing SKILL.md.
func (p *FactoryProvider) ListSkills() ([]Skill, error) {
	entries, err := os.ReadDir(p.baseDir)
	if err != nil {
		return nil, fmt.Errorf("factory: read skill directory: %w", err)
	}

	var skills []Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillPath := filepath.Join(p.baseDir, entry.Name(), "SKILL.md")
		if _, err := os.Stat(skillPath); err != nil {
			continue // skip directories without SKILL.md
		}
		skill, err := p.readSkillFile(entry.Name(), skillPath)
		if err != nil {
			return nil, fmt.Errorf("factory: list skills: %w", err)
		}
		skills = append(skills, *skill)
	}

	if skills == nil {
		return []Skill{}, nil
	}
	return skills, nil
}

// ReadSkill reads <baseDir>/<name>/SKILL.md.
func (p *FactoryProvider) ReadSkill(name string) (*Skill, error) {
	path := filepath.Join(p.baseDir, name, "SKILL.md")
	skill, err := p.readSkillFile(name, path)
	if err != nil {
		return nil, fmt.Errorf("factory: read skill %q: %w", name, err)
	}
	return skill, nil
}

// WriteSkill writes <baseDir>/<skill.Name>/SKILL.md with YAML frontmatter.
func (p *FactoryProvider) WriteSkill(skill Skill) error {
	dir := filepath.Join(p.baseDir, skill.Name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("factory: create skill directory: %w", err)
	}

	content := serializeFrontmatter(skill.Name, skill.Description, skill.Content)
	path := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("factory: write skill %q: %w", skill.Name, err)
	}
	return nil
}

// readSkillFile reads and parses a single SKILL.md file into a Skill.
func (p *FactoryProvider) readSkillFile(dirName, path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve absolute path: %w", err)
	}

	fm, body, err := parseFrontmatter(content)
	if err != nil {
		return nil, err
	}

	skill := &Skill{
		Name:       dirName,
		Content:    body,
		SourcePath: absPath,
	}

	if fm != nil {
		if fm.Name != "" {
			skill.Name = fm.Name
		}
		skill.Description = fm.Description
	} else {
		// No frontmatter: extract description from first "# " line.
		if lines := strings.SplitN(body, "\n", 2); len(lines) > 0 {
			if strings.HasPrefix(lines[0], "# ") {
				skill.Description = strings.TrimPrefix(lines[0], "# ")
			}
		}
	}

	return skill, nil
}

// parseFrontmatter splits content into a frontmatter struct and body string.
// Returns (nil, fullContent, nil) if no frontmatter is present.
// Returns error if frontmatter delimiters are found but YAML parsing fails.
func parseFrontmatter(content string) (*factoryFrontmatter, string, error) {
	if !strings.HasPrefix(content, "---") {
		return nil, content, nil
	}

	// Find the closing --- delimiter.
	// The first line is "---", so look for the next "---" line.
	rest := content[3:]
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}

	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		// No closing delimiter found -- treat entire content as body (no frontmatter).
		return nil, content, nil
	}

	yamlContent := rest[:idx]
	body := rest[idx+4:] // skip "\n---"
	if len(body) > 0 && body[0] == '\n' {
		body = body[1:]
	}

	var fm factoryFrontmatter
	if err := yaml.Unmarshal([]byte(yamlContent), &fm); err != nil {
		return nil, "", fmt.Errorf("factory: parse frontmatter: %w", err)
	}

	return &fm, body, nil
}

// serializeFrontmatter produces the ---/yaml/--- block followed by body content.
func serializeFrontmatter(name, description, body string) string {
	fm := factoryFrontmatter{
		Name:        name,
		Description: description,
	}
	yamlBytes, _ := yaml.Marshal(&fm)

	var b strings.Builder
	b.WriteString("---\n")
	b.Write(yamlBytes)
	b.WriteString("---\n")
	if body != "" {
		b.WriteString(body)
	}
	return b.String()
}

func init() {
	Register(NewFactoryProvider())
}
