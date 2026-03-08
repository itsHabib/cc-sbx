# skill-sync

Sync AI assistant skills across providers -- write once, use everywhere.

## The Problem

Every AI coding assistant stores custom skills in a different directory: Claude Code uses `~/.claude/skills/`, Copilot uses `~/.copilot/skills/`, Gemini CLI uses `~/.gemini/skills/`, and Factory uses `~/.factory/skills/`. They all use the same `SKILL.md` format (the [Agent Skills open standard](https://docs.github.com/en/copilot/concepts/agents/about-agent-skills)), but if you use more than one tool, you end up maintaining the same skills in multiple places. When one copy drifts, you have no way to know until something breaks.

skill-sync fixes this. Declare one provider as your source of truth, and skill-sync copies your skills to every target you configure. Then use `status` to catch drift before it becomes a problem.

## Quick Start

```bash
# Install
go install github.com/user/skill-sync@latest

# Create a config file
skill-sync init --source claude --targets copilot,gemini,factory

# Sync all skills
skill-sync sync

# Check for drift
skill-sync status
```

Or skip the config file entirely:

```bash
skill-sync sync --source claude --targets copilot,gemini
```

## Usage

### `skill-sync init`

Creates a `.skill-sync.yaml` config file in the current directory.

```bash
skill-sync init --source claude --targets copilot,gemini,factory
```

```
Created .skill-sync.yaml (source: claude, targets: [copilot gemini factory])
```

### `skill-sync sync`

Reads skills from your source provider and writes them to all targets.

```bash
skill-sync sync
```

```
SKILL    TARGET   STATUS
deploy   copilot  synced
deploy   gemini   synced
deploy   factory  synced
search   copilot  synced
search   gemini   synced
search   factory  synced

Synced: 6  Errors: 0
```

Preview what would happen without writing anything:

```bash
skill-sync sync --dry-run
```

Sync only specific skills:

```bash
skill-sync sync --skill deploy --skill review
```

Override directories:

```bash
# Use a custom source directory
skill-sync sync --source-dir /path/to/my/skills

# Use a custom target directory (single target only)
skill-sync sync --source claude --targets copilot --target-dir /path/to/copilot/skills
```

### `skill-sync status`

Compares skills in your source against all targets and reports drift. Exits with code 1 if any drift is detected.

```bash
skill-sync status
```

```
Target: copilot
SKILL    STATUS
deploy   [ok] in-sync
search   [!] modified
lint     [-] missing

Target: gemini
SKILL    STATUS
deploy   [ok] in-sync
search   [ok] in-sync
lint     [ok] in-sync
```

Status symbols:

| Symbol | Meaning |
|--------|---------|
| `[ok] in-sync` | Source and target content match |
| `[!] modified` | Both exist but content differs |
| `[-] missing` | Skill exists in source but not in target |
| `[+] extra` | Skill exists in target but not in source |

### `skill-sync diff [provider]`

Shows unified diffs for skills that differ between source and a target. If no provider is specified, shows diffs for all targets.

```bash
skill-sync diff copilot
```

```
--- a/search
+++ b/search
@@ -1,3 +1,3 @@
 # Search codebase
-Search for $ARGUMENTS in ${PROJECT} across all files.
+Search for $ARGUMENTS in src/ only.
 Return matching lines with context.
```

## Supported Providers

All providers use the [Agent Skills open standard](https://docs.github.com/en/copilot/concepts/agents/about-agent-skills) -- `SKILL.md` files with optional YAML frontmatter, one directory per skill.

| Provider | Default Skill Location | Aliases / Compat |
|----------|----------------------|------------------|
| Claude Code | `~/.claude/skills/<name>/SKILL.md` | — |
| GitHub Copilot | `~/.copilot/skills/<name>/SKILL.md` | Also reads `~/.claude/skills/` |
| Gemini CLI | `~/.gemini/skills/<name>/SKILL.md` | Also reads `~/.agents/skills/` |
| Factory AI | `~/.factory/skills/<name>/SKILL.md` | Also reads `.agent/skills/` |

All defaults are user-level (`$HOME`) directories, so skills follow you across projects.

## Configuration

The `.skill-sync.yaml` file declares your source provider, target providers, and optional directory overrides.

```yaml
# Source of truth -- skills are read from here
source: claude

# Optional: override the source provider's default directory
# source_dir: /custom/path/to/claude/skills

# Target providers -- skills are synced to all of these
targets:
    - copilot
    - gemini
    - factory

# Optional: override target directories (per-provider)
# target_dirs:
#   copilot: /custom/path/to/copilot/skills
#   gemini: /custom/path/to/gemini/skills

# Optional: sync only these skills (empty = sync all)
skills: []
```

### Directory Override Priority

1. CLI flag (`--source-dir` / `--target-dir`) -- highest priority
2. Config file (`source_dir` / `target_dirs`)
3. Provider default (`~/.<provider>/skills/`)

The `--target-dir` flag can only be used when there is a single target. For multiple targets, use `target_dirs` in the config file.

All commands accept `--config` to use a different config file path:

```bash
skill-sync sync --config my-config.yaml
```

You can also skip the config file entirely with `--source` and `--targets`:

```bash
skill-sync status --source claude --targets copilot,gemini
```

## How It Works

Since all four providers now use the same `SKILL.md` format, **sync is a straight copy** -- no format translation needed. skill-sync reads each `<name>/SKILL.md` directory from the source and writes it verbatim to each target's skill directory.

**status** reads skills from both source and all targets, normalizes trailing whitespace, and compares content. It reports each skill as in-sync, modified, missing, or extra. If any drift is detected, it exits with code 1.

**diff** does the same comparison as `status` but produces unified diffs (like `git diff`) for modified skills instead of a status table.

Sync is additive -- it writes source skills to targets but does not delete extra skills found in targets. The `status` command reports extras so you can handle them manually.

## CI Integration

Use `status` as a CI gate to catch skill drift on every push:

```yaml
name: Skill Sync Check
on: [push, pull_request]

jobs:
  skill-drift:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Build skill-sync
        run: go build -o skill-sync .

      - name: Check for skill drift
        run: ./skill-sync status
```

### Exit Codes

| Command | Exit 0 | Exit 1 |
|---------|--------|--------|
| `init` | Config created | Config exists, unknown provider, or validation error |
| `sync` | All skills synced | One or more skills failed to sync |
| `status` | All targets in-sync | Drift detected |
| `diff` | Always (diffs printed) | Provider resolution error |

## Contributing

```bash
# Build
go build -o skill-sync .

# Run all tests
go test ./...

# Run smoke tests
go test -tags smoke ./tests/

# Lint
go vet ./...
```

The codebase is structured as:

```
cmd/           CLI commands (cobra)
internal/
  config/      Config loading and validation
  provider/    Provider interface + shared skillMDProvider
  sync/        Sync engine + diff/drift engine
tests/         Smoke tests
```

When adding a new provider, register a `ProviderFactory` in an `init()` function -- it takes a `baseDir` string and returns a `Provider`. See any of the existing provider files for a one-liner example.

## License

MIT
