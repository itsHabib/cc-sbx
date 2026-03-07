# skill-sync End-to-End Test Plan

A practical test plan for validating skill-sync CLI commands, provider interactions, drift detection, and cross-provider format translation. Estimated completion time: 15-20 minutes.

---

## 1. Prerequisites

### Build the Binary

```bash
cd <project-root>
go mod tidy
go build -o skill-sync .
```

Verify: `./skill-sync --help` prints usage information with `init`, `sync`, `status`, and `diff` subcommands.

### Create a Test Project Directory

All tests run inside an isolated directory to avoid polluting your real workspace.

```bash
export TEST_DIR=$(mktemp -d)
cp ./skill-sync "$TEST_DIR/"
cd "$TEST_DIR"
```

### Set Up Provider Directories

skill-sync providers use hardcoded base directories. For project-level providers (Copilot, Factory), we create them inside `$TEST_DIR`. For user-level providers (Claude, Gemini), we use the real home directories with backup/restore.

**Back up existing user-level skills (if any):**

```bash
# Claude
if [ -d ~/.claude/commands ]; then
  cp -r ~/.claude/commands ~/.claude/commands.bak
fi

# Gemini
if [ -d ~/.gemini/commands ]; then
  cp -r ~/.gemini/commands ~/.gemini/commands.bak
fi
```

**Create required directories:**

```bash
mkdir -p ~/.claude/commands
mkdir -p ~/.gemini/commands
mkdir -p "$TEST_DIR/.github/prompts"
mkdir -p "$TEST_DIR/.factory/skills"
```

### Seed Source Skills

Create test skills in the Claude provider directory (source). Use `_test-` prefixed names to avoid collisions with real skills.

```bash
# Skill 1: simple content
cat > ~/.claude/commands/_test-deploy.md << 'EOF'
# Deploy to production
Run the deploy pipeline for the current branch.
Check all tests pass before deploying.
EOF

# Skill 2: content with argument placeholders
cat > ~/.claude/commands/_test-search.md << 'EOF'
# Search codebase
Search for $ARGUMENTS in ${PROJECT} across all files.
Return matching lines with context.
EOF

# Skill 3: plain content (no description header)
cat > ~/.claude/commands/_test-lint.md << 'EOF'
Run the linter on all changed files.
Fix any auto-fixable issues.
Report remaining warnings.
EOF
```

---

## 2. Test Scenarios

### TC-01: Initialize Config

**Precondition:** No `.skill-sync.yaml` exists in `$TEST_DIR`.

**Steps:**

```bash
cd "$TEST_DIR"
./skill-sync init --source claude --targets copilot,gemini,factory
```

**Expected:**
- Exit code 0
- Prints: `Created .skill-sync.yaml (source: claude, targets: [copilot gemini factory])`
- File `.skill-sync.yaml` exists with `source: claude` and `targets: [copilot, gemini, factory]`

**Verify:**

```bash
cat .skill-sync.yaml
```

---

### TC-02: Init Refuses Overwrite

**Precondition:** `.skill-sync.yaml` already exists from TC-01.

**Steps:**

```bash
./skill-sync init --source claude --targets copilot
```

**Expected:**
- Exit code 1
- Error message contains: `.skill-sync.yaml already exists`

---

### TC-03: Init Rejects Unknown Provider

**Precondition:** No `.skill-sync.yaml` (remove if present: `rm -f .skill-sync.yaml`).

**Steps:**

```bash
rm -f .skill-sync.yaml
./skill-sync init --source unknown --targets copilot
```

**Expected:**
- Exit code 1
- Error message contains: `unknown source provider "unknown"`

**Restore config for remaining tests:**

```bash
./skill-sync init --source claude --targets copilot,gemini,factory
```

---

### TC-04: Sync All Skills

**Precondition:** `.skill-sync.yaml` exists (from TC-01/TC-03 restore). Source skills seeded.

**Steps:**

```bash
./skill-sync sync
```

**Expected:**
- Exit code 0
- Table output shows 3 skills x 3 targets = 9 rows, all with status `synced`
- Summary line: `Synced: 9  Errors: 0`

**Verify target files exist:**

```bash
# Copilot
ls "$TEST_DIR/.github/prompts/"
# Expected: _test-deploy.prompt.md  _test-lint.prompt.md  _test-search.prompt.md

# Gemini
ls ~/.gemini/commands/
# Expected: _test-deploy.toml  _test-lint.toml  _test-search.toml

# Factory
ls "$TEST_DIR/.factory/skills/"
# Expected: _test-deploy/  _test-lint/  _test-search/  (each containing SKILL.md)
```

---

### TC-05: Sync Dry Run

**Precondition:** Config exists. Source skills seeded.

**Steps:**

```bash
./skill-sync sync --dry-run
```

**Expected:**
- Exit code 0
- Table output shows each skill/target pair with status `would sync`
- Summary line: `Would sync: 3 skill(s) to 3 target(s)`
- No files are modified on disk (verify by checking timestamps or content of an existing target file)

---

### TC-06: Sync with Skill Filter

**Precondition:** Config exists. Clean target dirs (or accept existing state).

**Steps:**

```bash
./skill-sync sync --skill _test-deploy
```

**Expected:**
- Exit code 0
- Table output shows only `_test-deploy` synced to 3 targets (3 rows)
- Summary: `Synced: 3  Errors: 0`
- Only `_test-deploy` files are created/updated in targets; other skills remain unchanged

---

### TC-07: Status -- All In Sync

**Precondition:** All skills have been synced (run `./skill-sync sync` if not already done).

**Steps:**

```bash
./skill-sync sync
./skill-sync status
```

**Expected:**
- Exit code 0
- Output shows each target with all skills marked `[ok] in-sync`

---

### TC-08: Status -- Detect Drift (Modified)

**Precondition:** All skills synced from TC-07.

**Steps:**

```bash
# Modify a target skill to introduce drift
echo "MODIFIED CONTENT" > "$TEST_DIR/.github/prompts/_test-deploy.prompt.md"

./skill-sync status
```

**Expected:**
- Exit code 1 (drift detected)
- Copilot target shows `_test-deploy` as `[!] modified`
- Other skills in copilot remain `[ok] in-sync`
- Gemini and Factory targets show all `[ok] in-sync`

---

### TC-09: Status -- Detect Drift (Missing in Target)

**Precondition:** Skills synced. Then remove a target file.

**Steps:**

```bash
# Remove a target skill
rm "$TEST_DIR/.github/prompts/_test-lint.prompt.md"

./skill-sync status
```

**Expected:**
- Exit code 1 (drift detected)
- Copilot target shows `_test-lint` as `[-] missing`
- `_test-deploy` still shows `[!] modified` (from TC-08)

---

### TC-10: Status -- Detect Drift (Extra in Target)

**Precondition:** Continue from TC-09 state.

**Steps:**

```bash
# Add an extra skill to a target that doesn't exist in source
echo "Extra skill content" > "$TEST_DIR/.github/prompts/_test-bonus.prompt.md"

./skill-sync status
```

**Expected:**
- Exit code 1 (drift detected)
- Copilot target shows `_test-bonus` as `[+] extra`

---

### TC-11: Diff -- Show Unified Diff

**Precondition:** Drift exists in copilot from TC-08 (modified `_test-deploy`).

**Steps:**

```bash
./skill-sync diff copilot
```

**Expected:**
- Exit code 0
- Output contains unified diff with `---` and `+++` headers
- Shows the content difference for `_test-deploy`

**Also test diff with no arguments (shows all targets):**

```bash
./skill-sync diff
```

**Expected:**
- Shows diffs for all targets that have modified skills

---

### TC-12: Re-sync Restores In-Sync State

**Precondition:** Drift exists from TC-08/TC-09.

**Steps:**

```bash
./skill-sync sync
./skill-sync status
```

**Expected:**
- `sync` exits 0 with all skills synced successfully
- `status` exits 0 with all skills `[ok] in-sync` across all targets
- Note: the extra `_test-bonus` skill still shows `[+] extra` because sync does not delete extra skills

---

## 3. Provider-Specific Verification

After running TC-04 (sync all), verify format correctness for each provider.

### Claude Code (Source)

| Check | Expected |
|-------|----------|
| Directory | `~/.claude/commands/` |
| File pattern | `_test-deploy.md`, `_test-search.md`, `_test-lint.md` |
| Format | Pure Markdown, no frontmatter |
| Description | First line starting with `# ` (e.g., `# Deploy to production`) |
| Arguments | `$ARGUMENTS`, `${PROJECT}` as literal text |

### GitHub Copilot (Target)

| Check | Expected |
|-------|----------|
| Directory | `$TEST_DIR/.github/prompts/` |
| File pattern | `*.prompt.md` |
| Format | Pure Markdown, no frontmatter |
| Content | Matches source content verbatim |

**Verify:**

```bash
cat "$TEST_DIR/.github/prompts/_test-deploy.prompt.md"
# Should match source content exactly
```

### Gemini CLI (Target)

| Check | Expected |
|-------|----------|
| Directory | `~/.gemini/commands/` |
| File pattern | `*.toml` |
| Format | TOML with `description` and `prompt` fields |
| `description` field | Extracted from `# ` header line (e.g., `"Deploy to production"`) |
| `prompt` field | Full skill content including the `# ` line |

**Verify:**

```bash
cat ~/.gemini/commands/_test-deploy.toml
# Should contain:
#   description = "Deploy to production"
#   prompt = "# Deploy to production\n..."
```

### Factory AI Droid (Target)

| Check | Expected |
|-------|----------|
| Directory | `$TEST_DIR/.factory/skills/` |
| File pattern | `<name>/SKILL.md` (subdirectory per skill) |
| Format | Markdown with YAML frontmatter (`---` delimited) |
| Frontmatter | Contains `name` and `description` fields |
| Body | Skill content as Markdown after frontmatter |

**Verify:**

```bash
cat "$TEST_DIR/.factory/skills/_test-deploy/SKILL.md"
# Should contain:
#   ---
#   name: _test-deploy
#   description: Deploy to production
#   ---
#   <skill content>
```

---

## 4. Error Scenarios

### ERR-01: Missing Config File

```bash
cd "$(mktemp -d)"
/path/to/skill-sync status
```

**Expected:** Exit code 1, error contains `loading config` or `reading file`.

### ERR-02: Sync Without Config

```bash
cd "$(mktemp -d)"
/path/to/skill-sync sync
```

**Expected:** Exit code 1, error about missing config file.

### ERR-03: Custom Config Path

```bash
cd "$TEST_DIR"
./skill-sync init --source claude --targets copilot --config custom-config.yaml
./skill-sync status --config custom-config.yaml
```

**Expected:** Both commands succeed using the custom config path.

**Cleanup:**

```bash
rm -f "$TEST_DIR/custom-config.yaml"
```

---

## 5. CI Integration

### Using `skill-sync status` as a CI Gate

The `status` command exits with code 0 when all targets are in-sync and code 1 when drift is detected. This makes it suitable as a CI check.

**GitHub Actions example:**

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

**Generic CI script:**

```bash
#!/bin/bash
set -e

# Build
go build -o skill-sync .

# Check drift -- exits non-zero if any target has drifted
./skill-sync status

echo "All skills in sync."
```

### Exit Code Contract

| Command | Exit 0 | Exit 1 |
|---------|--------|--------|
| `init` | Config created successfully | Config already exists, unknown provider, or validation error |
| `sync` | All skills synced successfully | One or more sync errors |
| `status` | All targets are in-sync | Any drift detected (modified, missing, or extra) |
| `diff` | Diffs printed (even if no diffs exist) | Provider resolution or engine error |

---

## 6. Pass/Fail Criteria

### Test Execution Summary

| Test Case | Description | Pass Criteria |
|-----------|-------------|---------------|
| TC-01 | Initialize config | Config file created with correct content |
| TC-02 | Init refuses overwrite | Error returned, no file modification |
| TC-03 | Init rejects unknown provider | Error returned with clear message |
| TC-04 | Sync all skills | 9 skills synced (3 skills x 3 targets), files exist in correct locations |
| TC-05 | Sync dry run | Output shows "would sync", no files written |
| TC-06 | Sync with skill filter | Only filtered skill synced |
| TC-07 | Status all in sync | Exit 0, all skills show `[ok] in-sync` |
| TC-08 | Status detects modified | Exit 1, modified skill shows `[!] modified` |
| TC-09 | Status detects missing | Exit 1, removed skill shows `[-] missing` |
| TC-10 | Status detects extra | Exit 1, extra skill shows `[+] extra` |
| TC-11 | Diff shows changes | Unified diff output with `---`/`+++` headers |
| TC-12 | Re-sync restores state | Status returns to in-sync after re-sync |
| ERR-01 | Missing config | Non-zero exit with config error |
| ERR-02 | Sync without config | Non-zero exit with config error |
| ERR-03 | Custom config path | Commands work with `--config` override |

### Overall Pass Criteria

- All 15 test cases pass
- Provider-specific format checks confirm correct file structure (Section 3)
- No test skill files remain in provider directories after cleanup (Section 7)

### Overall Fail Criteria

- Any test case does not produce the expected exit code
- Any test case does not produce the expected output pattern
- Synced files are in the wrong format for their target provider
- `status` reports false drift immediately after a clean `sync`

---

## 7. Cleanup

After completing all tests, remove test artifacts and restore backups.

```bash
# Remove test skills from user-level provider dirs
rm -f ~/.claude/commands/_test-*.md
rm -f ~/.gemini/commands/_test-*.toml

# Restore backups if they exist
if [ -d ~/.claude/commands.bak ]; then
  rm -rf ~/.claude/commands
  mv ~/.claude/commands.bak ~/.claude/commands
fi

if [ -d ~/.gemini/commands.bak ]; then
  rm -rf ~/.gemini/commands
  mv ~/.gemini/commands.bak ~/.gemini/commands
fi

# Remove test project directory
rm -rf "$TEST_DIR"
```

### Verify Cleanup

```bash
# Confirm no test files remain
ls ~/.claude/commands/_test-* 2>/dev/null && echo "WARN: test files remain" || echo "OK: clean"
ls ~/.gemini/commands/_test-* 2>/dev/null && echo "WARN: test files remain" || echo "OK: clean"
```

---

## 8. Known Limitations

- **Hardcoded provider directories:** The CLI uses hardcoded base paths registered at startup. There is no runtime flag to override provider directories. This means user-level providers (Claude, Gemini) must be tested against real home directory paths.
- **No argument translation:** `$ARGUMENTS` and `${NAME}` from Claude are passed through verbatim to other providers. They will not be functional in Gemini (which uses `{{args}}`) or Factory (no argument syntax).
- **Sync does not delete extras:** The `sync` command only writes source skills to targets. It does not remove extra skills in the target that have no corresponding source. The `status` command reports these as `[+] extra`.
- **Description loss on reverse sync:** When using Gemini or Factory as the source and Claude as a target, the description field is not prepended as a `# ` header line in the Claude output. The description is only preserved in the `Skill.Description` field, not written to the Claude file.
