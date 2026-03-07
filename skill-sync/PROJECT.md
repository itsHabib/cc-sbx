# skill-sync

> Sync AI assistant skills from your primary provider to all others, with drift detection.

---

## Problem & Motivation

Developers write custom skills/prompts for their preferred CLI AI assistant (Claude Code, GitHub Copilot, Gemini CLI, Factory AI Droid). When they switch between tools or use multiple, their skills are siloed — each provider has its own format, directory structure, and conventions. Manually copying and translating skills is tedious and error-prone, and there's no way to know when skills have drifted out of sync.

skill-sync solves this by letting you declare one source provider and syncing skills to all targets, with format translation and drift detection built in.

---

## Definition of Done

- `skill-sync init` creates a `.skill-sync.yaml` config declaring source + targets
- `skill-sync sync` reads skills from the source provider, translates format, and writes to all target providers
- `skill-sync status` shows drift between source and each target (in-sync, modified, missing)
- `skill-sync diff <provider>` shows the exact diff for a specific target
- Supports 4 providers: Claude Code, GitHub Copilot, Gemini CLI, Factory AI Droid
- Clean CLI UX with helpful `--help` text and error messages

---

## Key Components

- **Provider interface** — common abstraction for reading/writing skills from any provider
- **Provider registry** — registers available providers, looked up by name
- **Skill model** — normalized representation of a skill (name, description, content, arguments)
- **Sync engine** — orchestrates reading from source, translating, writing to targets
- **Diff engine** — compares source skills against target state, reports drift
- **Config** — reads `.skill-sync.yaml` (source, targets, optional skill filter)
- **CLI commands** — `init`, `sync`, `status`, `diff` via cobra

---

## Tech Stack

- Go 1.22+
- cobra (CLI framework)
- yaml.v3 (config parsing)
- No external dependencies beyond stdlib + cobra + yaml

---

## Non-Goals

- Bidirectional sync (no merge conflicts in prompt files)
- New canonical skill format (your source provider IS the canonical format)
- GUI or web interface
- Plugin system for custom providers (hardcoded 4 providers is fine)
- Watching for changes / auto-sync (manual CLI invocation only)

---

## Constraints

- Provider skill formats need research/verification during implementation — Claude Code's format is well-known, others require checking current docs
- Each provider has user-level AND project-level skill locations; start with user-level only
- Skills are text files (markdown, yaml, etc.) — no binary file support needed

---

## Provider Format Reference

### Claude Code (VERIFIED)
- **User-level:** `~/.claude/skills/<name>/SKILL.md`
- **Project-level:** `.claude/skills/<name>/SKILL.md`
- **Format:** Pure Markdown. No frontmatter. Directory-per-skill layout.
  - First line starting with `# ` = description (strip `# ` prefix)
  - `$ARGUMENTS` placeholder for user input
  - `${UPPER_CASE}` named argument placeholders
  - Directory name = skill name
- **Skill name:** derived from directory name (e.g., `deploy/SKILL.md` → skill name `deploy`)
- **Invocation:** `/deploy` in Claude Code CLI

### GitHub Copilot (VERIFIED)
- **User-level:** No user-level custom prompts directory (instructions only via VS Code settings)
- **Project-level prompt files:** `.github/prompts/*.prompt.md`
  - Become slash commands in VS Code/JetBrains (e.g., `/TESTPROMPT`)
  - Pure Markdown, no frontmatter
  - Can reference files with `#file:../../path/to/file.ts` or `[label](../../path/to/file.ts)`
  - Currently in public preview (VS Code, Visual Studio, JetBrains only)
- **Project-level instructions:** `.github/copilot-instructions.md` (repo-wide)
- **Path-specific instructions:** `.github/instructions/*.instructions.md`
  - YAML frontmatter with `applyTo` glob pattern and optional `excludeAgent` field
  ```yaml
  ---
  applyTo: "**/*.ts"
  excludeAgent: "code-review"
  ---
  ```
- **For skill-sync scope:** We sync prompt files (`.prompt.md`), NOT instruction files (which are project-specific context, not reusable skills)
- **Default baseDir:** `.github/prompts/` (project-level)
- **Skill name:** derived from filename (e.g., `review-code.prompt.md` → skill name `review-code`)

### Gemini CLI (VERIFIED)
- **User-level:** `~/.gemini/commands/*.toml`
- **Project-level:** `<project>/.gemini/commands/*.toml`
- **Format:** TOML (NOT Markdown)
  ```toml
  description = "Brief description for /help menu"
  prompt = """
  Your prompt content here.
  Use {{args}} for argument injection.
  Use !{shell command} for shell command injection.
  Use @{path/to/file} for file content injection.
  """
  ```
- **Required fields:** `prompt` (string)
- **Optional fields:** `description` (string)
- **Arguments:** `{{args}}` placeholder (auto shell-escaped inside `!{...}` blocks)
- **Namespacing:** subdirectories create namespaced commands (e.g., `git/commit.toml` → `/git:commit`)
- **Default baseDir:** `~/.gemini/commands/` (user-level; project-level is `<project>/.gemini/commands/`)
- **Skill name:** derived from file path relative to commands dir (e.g., `commit.toml` → `/commit`, `git/fix.toml` → `/git:fix`)
- **Reload:** `/commands reload` after changes

### Factory AI Droid (VERIFIED — from real files on disk)
- **User-level droids:** `~/.factory/droids/*.md`
- **Project-level skills:** `.factory/skills/<name>/SKILL.md`
- **Format:** Markdown with YAML frontmatter
  ```yaml
  ---
  name: worker
  description: >-
    General-purpose worker droid for delegating tasks.
  model: inherit
  ---
  # Worker Droid

  Prompt content here as markdown body.
  ```
- **Frontmatter fields:**
  - `name` (string, required): droid identifier
  - `description` (string, required): what the droid does
  - `model` (string, optional): model to use (`inherit` = use session default, or specific model ID)
- **Body:** Markdown prompt/instructions (everything after frontmatter)
- **Note:** Some droids have NO frontmatter (plain markdown only — e.g., `new-reader-writer.md`). The provider should handle both cases.
- **Default baseDir:** `.factory/skills/` (project-level; each skill is `<name>/SKILL.md`)
- **Skill name:** derived from directory name (e.g., `.factory/skills/foo/SKILL.md` → `foo`)

---

## Format Translation Notes

When syncing skills between providers, these translations apply:

| Field | Claude Code | Copilot | Gemini CLI | Factory Droid |
|-------|-------------|---------|------------|---------------|
| **Skill name** | filename | filename (minus `.prompt.md`) | filename/path (minus `.toml`) | frontmatter `name` or filename |
| **Description** | `# ` first line | N/A (no description field) | `description` TOML field | frontmatter `description` |
| **Content** | markdown body | markdown body | `prompt` TOML field | markdown body after frontmatter |
| **Arguments** | `$ARGUMENTS`, `${NAME}` | `#file:` references (different concept) | `{{args}}`, `@{file}`, `!{cmd}` | N/A (no argument syntax) |
| **File extension** | `.md` | `.prompt.md` | `.toml` | `.md` |

**Key challenge:** Gemini uses TOML, all others use Markdown. The Gemini provider must convert between TOML and the internal Skill model.

**Argument translation:** Each provider has its own argument/placeholder syntax. For MVP, arguments are stored verbatim in the Skill model and NOT translated between providers. The `Content` field preserves source format.

---

## Team

| Role | Focus |
|------|-------|
| Provider Architect | Provider interface, registry, skill model |
| Claude Provider Dev | Claude Code skill reader/writer |
| Sync Engine Dev | Sync engine + diff/drift detection |
| Config & CLI Foundation | Config parsing, root + init commands |
| Copilot Provider Dev | GitHub Copilot skill reader/writer |
| Gemini Provider Dev | Gemini CLI skill reader/writer |
| Factory Provider Dev | Factory AI Droid skill reader/writer |
| CLI Commands Dev | sync, status, diff commands |
| QE Lead | End-to-end test plan |
| Smoke Test Dev | Smoke test script |
| Edge Case Analyst | Edge case documentation |
| README Author | README with quickstart + architecture diagram |
| CLI UX Dev | Polished --help text and error messages |
| Code Quality Dev | Docstrings on exported types/funcs |

---

## Phases

| Phase | Config | Goal |
|-------|--------|------|
| core | docs/core/kickoff.yaml | Provider model, sync engine, diff engine, Claude provider |
| providers-and-commands | docs/providers-and-commands/kickoff.yaml | Remaining providers + full CLI commands |
| validation | docs/validation/kickoff.yaml | E2E test plan + smoke tests |
| polish | docs/polish/kickoff.yaml | README, diagrams, CLI help, docstrings |

---

## Usage in Phase Planning

This file is the source of truth for all planning phases.
List it as the first dependency in every phase config:

```yaml
dependencies:
  - PROJECT.md
```
