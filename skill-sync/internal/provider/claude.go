package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// argPattern matches $ARGUMENTS (literal) and ${UPPER_CASE_IDENTIFIER} patterns.
var argPattern = regexp.MustCompile(`\$ARGUMENTS|\$\{([A-Z_][A-Z0-9_]*)\}`)

// Option configures a ClaudeProvider.
type Option func(*ClaudeProvider)

// WithBaseDir returns an Option that sets the base directory for skill files.
func WithBaseDir(dir string) Option {
	return func(p *ClaudeProvider) {
		p.baseDir = dir
	}
}

// ClaudeProvider reads and writes skills from Claude Code's skills directory.
// Skills use a directory-per-skill layout: <baseDir>/<name>/SKILL.md.
type ClaudeProvider struct {
	baseDir string
}

// NewClaudeProvider creates a ClaudeProvider. By default it uses ~/.claude/skills/.
func NewClaudeProvider(opts ...Option) *ClaudeProvider {
	home, _ := os.UserHomeDir()
	p := &ClaudeProvider{
		baseDir: filepath.Join(home, ".claude", "skills"),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Name returns "claude".
func (p *ClaudeProvider) Name() string {
	return "claude"
}

// SkillDir returns the base directory where Claude Code skills are stored.
func (p *ClaudeProvider) SkillDir() string {
	return p.baseDir
}

// ListSkills reads all <name>/SKILL.md directories under the base directory and returns them as skills.
// Returns an empty slice (not an error) if the directory exists but has no skill subdirectories.
func (p *ClaudeProvider) ListSkills() ([]Skill, error) {
	entries, err := os.ReadDir(p.baseDir)
	if err != nil {
		return nil, fmt.Errorf("claude: read skill directory: %w", err)
	}

	var skills []Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		skillPath := filepath.Join(p.baseDir, name, "SKILL.md")
		if _, err := os.Stat(skillPath); err != nil {
			continue // skip directories without SKILL.md
		}
		skill, err := p.readSkillFile(name, skillPath)
		if err != nil {
			return nil, fmt.Errorf("claude: list skills: %w", err)
		}
		skills = append(skills, *skill)
	}
	return skills, nil
}

// ReadSkill reads a single skill by name from <baseDir>/<name>/SKILL.md.
func (p *ClaudeProvider) ReadSkill(name string) (*Skill, error) {
	path := filepath.Join(p.baseDir, name, "SKILL.md")
	skill, err := p.readSkillFile(name, path)
	if err != nil {
		return nil, fmt.Errorf("claude: read skill %q: %w", name, err)
	}
	return skill, nil
}

// WriteSkill writes a skill to <baseDir>/<name>/SKILL.md. Creates directories if needed.
func (p *ClaudeProvider) WriteSkill(skill Skill) error {
	skillDir := filepath.Join(p.baseDir, skill.Name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("claude: create skill directory: %w", err)
	}

	path := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(path, []byte(skill.Content), 0644); err != nil {
		return fmt.Errorf("claude: write skill %q: %w", skill.Name, err)
	}
	return nil
}

// readSkillFile reads and parses a single .md file into a Skill.
// If the file has YAML frontmatter (--- delimited), it is stripped from Content
// and the description field is extracted from it.
func (p *ClaudeProvider) readSkillFile(name, path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	raw := string(data)
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve absolute path: %w", err)
	}

	// Extract description from frontmatter if present, but keep content intact.
	_, description := stripFrontmatter(raw)

	skill := &Skill{
		Name:        name,
		Content:     raw,
		Description: description,
		SourcePath:  absPath,
	}

	// If no description from frontmatter, try first markdown heading.
	if skill.Description == "" {
		if lines := strings.SplitN(raw, "\n", 2); len(lines) > 0 {
			firstLine := lines[0]
			if strings.HasPrefix(firstLine, "# ") {
				skill.Description = strings.TrimPrefix(firstLine, "# ")
			}
		}
	}

	// Extract arguments.
	skill.Arguments = extractArguments(raw)

	return skill, nil
}

// stripFrontmatter removes YAML frontmatter (--- delimited) from content.
// Returns the content without frontmatter and the description if found.
func stripFrontmatter(raw string) (content string, description string) {
	if !strings.HasPrefix(raw, "---\n") {
		return raw, ""
	}

	// Find closing ---
	end := strings.Index(raw[4:], "\n---")
	if end < 0 {
		return raw, ""
	}

	frontmatter := raw[4 : 4+end]
	content = strings.TrimLeft(raw[4+end+4:], "\n")

	// Extract description from frontmatter lines.
	for _, line := range strings.Split(frontmatter, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "description:") {
			description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			// Remove surrounding quotes if present.
			description = strings.Trim(description, "\"'")
			break
		}
	}

	return content, description
}

// extractArguments finds all $ARGUMENTS and ${UPPER_CASE} patterns, deduplicated, in order.
func extractArguments(content string) []string {
	matches := argPattern.FindAllString(content, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	var args []string
	for _, m := range matches {
		if _, ok := seen[m]; !ok {
			seen[m] = struct{}{}
			args = append(args, m)
		}
	}
	return args
}

func init() {
	Register(NewClaudeProvider())
}
