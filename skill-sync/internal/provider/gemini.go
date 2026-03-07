package provider

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

// geminiArgPattern matches {{args}} placeholders in Gemini prompts.
var geminiArgPattern = regexp.MustCompile(`\{\{[^}]+\}\}`)

// geminiCommand represents the TOML structure of a Gemini CLI command file.
type geminiCommand struct {
	Description string `toml:"description,omitempty"`
	Prompt      string `toml:"prompt"`
}

// GeminiOption configures a GeminiProvider.
type GeminiOption func(*GeminiProvider)

// WithGeminiBaseDir returns a GeminiOption that sets the base directory for command files.
func WithGeminiBaseDir(dir string) GeminiOption {
	return func(p *GeminiProvider) {
		p.baseDir = dir
	}
}

// GeminiProvider reads and writes skills from Gemini CLI's commands directory.
type GeminiProvider struct {
	baseDir string
}

// NewGeminiProvider creates a GeminiProvider. By default it uses ~/.gemini/commands/.
func NewGeminiProvider(opts ...GeminiOption) *GeminiProvider {
	home, _ := os.UserHomeDir()
	p := &GeminiProvider{
		baseDir: filepath.Join(home, ".gemini", "commands"),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Name returns "gemini".
func (p *GeminiProvider) Name() string {
	return "gemini"
}

// SkillDir returns the base directory where Gemini CLI commands are stored.
func (p *GeminiProvider) SkillDir() string {
	return p.baseDir
}

// ListSkills recursively walks the base directory for *.toml files and returns them as skills.
// Returns an empty slice (not an error) if the directory exists but has no .toml files.
func (p *GeminiProvider) ListSkills() ([]Skill, error) {
	info, err := os.Stat(p.baseDir)
	if err != nil {
		return nil, fmt.Errorf("gemini: read skill directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("gemini: read skill directory: %s is not a directory", p.baseDir)
	}

	var skills []Skill
	err = filepath.WalkDir(p.baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".toml" {
			return nil
		}

		name, err := geminiPathToName(p.baseDir, path)
		if err != nil {
			return fmt.Errorf("gemini: derive name from %s: %w", path, err)
		}

		skill, err := p.readSkillFile(name, path)
		if err != nil {
			return fmt.Errorf("gemini: list skills: %w", err)
		}
		skills = append(skills, *skill)
		return nil
	})
	if err != nil {
		return nil, err
	}

	if skills == nil {
		skills = []Skill{}
	}
	return skills, nil
}

// ReadSkill reads a single skill by name from the base directory.
// Converts ":" in name to "/" for path lookup (namespaced commands).
func (p *GeminiProvider) ReadSkill(name string) (*Skill, error) {
	if err := geminiValidateName(name); err != nil {
		return nil, fmt.Errorf("gemini: read skill %q: %w", name, err)
	}

	path := geminiNameToPath(p.baseDir, name)
	skill, err := p.readSkillFile(name, path)
	if err != nil {
		return nil, fmt.Errorf("gemini: read skill %q: %w", name, err)
	}
	return skill, nil
}

// WriteSkill writes a skill as a TOML file. Creates subdirectories if needed.
// Converts ":" in name to "/" for path construction.
func (p *GeminiProvider) WriteSkill(skill Skill) error {
	if err := geminiValidateName(skill.Name); err != nil {
		return fmt.Errorf("gemini: write skill %q: %w", skill.Name, err)
	}

	path := geminiNameToPath(p.baseDir, skill.Name)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("gemini: create skill directory: %w", err)
	}

	cmd := geminiCommand{
		Description: skill.Description,
		Prompt:      skill.Content,
	}

	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(cmd); err != nil {
		return fmt.Errorf("gemini: encode TOML for %q: %w", skill.Name, err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("gemini: write skill %q: %w", skill.Name, err)
	}
	return nil
}

// readSkillFile reads and parses a single .toml file into a Skill.
func (p *GeminiProvider) readSkillFile(name, path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cmd geminiCommand
	if err := toml.Unmarshal(data, &cmd); err != nil {
		return nil, fmt.Errorf("parse TOML: %w", err)
	}

	prompt := strings.TrimSpace(cmd.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("missing or empty prompt field")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve absolute path: %w", err)
	}

	skill := &Skill{
		Name:        name,
		Description: cmd.Description,
		Content:     cmd.Prompt,
		Arguments:   geminiExtractArgs(cmd.Prompt),
		SourcePath:  absPath,
	}

	return skill, nil
}

// geminiNameToPath converts a skill name to a file path.
// "git:commit" -> "<baseDir>/git/commit.toml"
func geminiNameToPath(baseDir, name string) string {
	parts := strings.ReplaceAll(name, ":", string(filepath.Separator))
	return filepath.Join(baseDir, parts+".toml")
}

// geminiPathToName converts a file path to a skill name.
// "<baseDir>/git/commit.toml" -> "git:commit"
func geminiPathToName(baseDir, absPath string) (string, error) {
	rel, err := filepath.Rel(baseDir, absPath)
	if err != nil {
		return "", err
	}
	name := strings.TrimSuffix(rel, ".toml")
	name = strings.ReplaceAll(name, string(filepath.Separator), ":")
	return name, nil
}

// geminiExtractArgs finds all {{...}} placeholders in the prompt, deduplicated, in order.
func geminiExtractArgs(prompt string) []string {
	matches := geminiArgPattern.FindAllString(prompt, -1)
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

// geminiValidateName checks that a skill name is safe (no path traversal).
func geminiValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("skill name must not be empty")
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("skill name must not contain '..'")
	}
	return nil
}

func init() {
	Register(NewGeminiProvider())
}
