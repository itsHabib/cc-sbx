// Package config handles parsing and validation of .skill-sync.yaml configuration files.
package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the .skill-sync.yaml configuration file.
type Config struct {
	// Source is the provider name to read skills from (e.g., "claude").
	Source string `yaml:"source"`

	// SourceDir optionally overrides the source provider's default skill directory.
	SourceDir string `yaml:"source_dir,omitempty"`

	// Targets lists provider names to sync skills to (e.g., ["copilot", "gemini"]).
	Targets []string `yaml:"targets"`

	// TargetDirs optionally overrides target providers' default skill directories.
	// Keys are provider names, values are directory paths.
	TargetDirs map[string]string `yaml:"target_dirs,omitempty"`

	// Skills optionally restricts syncing to the named skills.
	// An empty list means all skills are synced. Uses YAML flow style for compact inline lists.
	Skills []string `yaml:"skills,flow"`
}

// Load reads and parses a .skill-sync.yaml file from the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: reading file: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parsing yaml: %w", err)
	}
	return &cfg, nil
}

// Validate checks that source and target names exist in the provided list
// of registered provider names.
func (c *Config) Validate(registeredNames []string) error {
	nameSet := make(map[string]bool, len(registeredNames))
	for _, n := range registeredNames {
		nameSet[n] = true
	}

	var errs []string

	if c.Source == "" {
		errs = append(errs, "source must not be empty")
	} else if !nameSet[c.Source] {
		errs = append(errs, fmt.Sprintf("unknown source provider %q", c.Source))
	}

	if len(c.Targets) == 0 {
		errs = append(errs, "targets must have at least one entry")
	}

	for _, t := range c.Targets {
		if !nameSet[t] {
			errs = append(errs, fmt.Sprintf("unknown target provider %q", t))
		}
		if t == c.Source {
			errs = append(errs, fmt.Sprintf("source %q must not appear in targets", c.Source))
		}
	}

	// Validate target_dirs keys reference valid targets.
	for name := range c.TargetDirs {
		found := false
		for _, t := range c.Targets {
			if t == name {
				found = true
				break
			}
		}
		if !found {
			errs = append(errs, fmt.Sprintf("target_dirs: %q is not in targets list", name))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("config: validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}
