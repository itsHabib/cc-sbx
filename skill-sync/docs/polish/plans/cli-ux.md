# Master Plan: CLI UX Dev

## You are in PLAN MODE.

### Project
I want to polish **skill-sync**, a Go CLI that syncs AI assistant skills from a primary provider to others with drift detection.

**Goal:** make every CLI command's `--help` output genuinely helpful — real examples, clear descriptions, and error messages that tell the user what happened, why, and what to do about it.

### Role + Scope
- **Role:** CLI UX Dev
- **Scope:** I own the UX strings in `cmd/root.go`, `cmd/init.go`, `cmd/sync.go`, `cmd/status.go`, and `cmd/diff.go`. Specifically: cobra `Short`, `Long`, `Example` fields, flag descriptions, and error message formatting. I do NOT own command logic, provider implementations, engines, config, tests, README, or docstrings.
- **File I will write:** `/docs/polish/plans/cli-ux.md`
- **No-touch zones:** do not edit any other files; do not write code.

---

## Functional Requirements
- **FR1:** Every command has a `Short` description under 60 characters that clearly states what the command does.
- **FR2:** Every command has a `Long` description (2-3 sentences) explaining what the command does, when to use it, and key behavior (e.g., exit codes).
- **FR3:** Every command has a cobra `Example` block with 2-4 real usage examples showing common workflows.
- **FR4:** Error messages follow the pattern: "what happened + why it probably happened + what to do about it."
- **FR5:** Flag descriptions are concise but tell the user what effect the flag has, not just what it is.
- **Tests required:** Manual verification via `./skill-sync --help`, `./skill-sync <cmd> --help`, and intentionally triggering error paths.

## Non-Functional Requirements
- Language/runtime: Go 1.22+, cobra CLI framework
- Local dev: `go build ./... && ./skill-sync --help`
- Safety: no logic changes — UX strings only
- Documentation: CLAUDE.md + EXPLAIN.md contributions proposed below
- Performance: N/A — string constants only

---

## Assumptions / System Model
- The 4 registered providers are: `claude`, `copilot`, `gemini`, `factory`
- Config file is `.skill-sync.yaml` by default (overridable with `--config`)
- `PersistentPreRunE` in `root.go` handles config loading before subcommands run; `init` is exempt
- Error wrapping uses `fmt.Errorf("context: %w", err)` throughout
- Exit code semantics: `sync` exits 1 on errors, `status` exits 1 on drift, `diff` exits 0

---

## Data Model
N/A -- not in scope for this role. CLI UX Dev only touches user-facing strings, not data structures.

---

## APIs (CLI Help Surface)

### `skill-sync` (root)
```
Short: "Sync AI skills across providers"
Long:  "skill-sync reads skills from a source AI assistant (Claude Code, Copilot,
        Gemini CLI, Factory) and syncs them to target providers with format
        translation and drift detection.

        Configure once with 'skill-sync init', then run 'skill-sync sync' to
        keep all your providers in lockstep."

Example:
  # Quick start: init + sync
  skill-sync init --source claude --targets copilot,gemini
  skill-sync sync

  # Check if targets have drifted
  skill-sync status

  # See exactly what changed
  skill-sync diff copilot
```

### `skill-sync init`
```
Short: "Create a .skill-sync.yaml config file"
Long:  "Generates a .skill-sync.yaml in the current directory declaring which
        provider is your source of truth and which providers to sync to.

        Run this once per project. Requires --source and --targets flags."

Example:
  # Initialize with Claude as source, sync to Copilot and Gemini
  skill-sync init --source claude --targets copilot,gemini

  # Initialize with all targets
  skill-sync init --source claude --targets copilot,gemini,factory

  # Use a custom config path
  skill-sync init --source claude --targets copilot --config my-config.yaml
```

Error improvements:
- `--source is required` -> `Error: --source is required. Specify your source provider: skill-sync init --source claude --targets copilot,gemini`
- `--targets is required` -> `Error: --targets is required. Specify one or more target providers: --targets copilot,gemini,factory`
- `already exists` -> keep existing message (already clear)
- Provider validation -> `Error: unknown provider "foo". Available providers: claude, copilot, gemini, factory`

### `skill-sync sync`
```
Short: "Sync skills from source to all targets"
Long:  "Reads skills from your source provider, translates the format, and writes
        them to every target provider. Use --dry-run to preview without writing.

        Exits with code 1 if any skill fails to sync."

Example:
  # Sync all skills to all targets
  skill-sync sync

  # Preview what would be synced
  skill-sync sync --dry-run

  # Sync only specific skills
  skill-sync sync --skill deploy --skill review

  # Sync without a config file
  skill-sync sync --source claude --targets copilot,gemini
```

Flag improvements:
- `--dry-run`: `"Preview sync without writing to targets"`
- `--skill`: `"Sync only named skills (repeatable)"`

### `skill-sync status`
```
Short: "Show sync drift between providers"
Long:  "Compares skills in your source provider against all targets and reports
        which skills are in sync, modified, missing, or extra.

        Exits with code 1 if any drift is detected -- useful for CI checks."

Example:
  # Check drift across all targets
  skill-sync status

  # Use inline providers (no config file)
  skill-sync status --source claude --targets copilot,gemini
```

### `skill-sync diff [provider]`
```
Short: "Show unified diffs for drifted skills"
Long:  "Prints unified diffs for skills that differ between your source and a
        target provider. If no provider is specified, shows diffs for all targets.

        Like 'git diff' -- informational only, always exits 0."

Example:
  # Show diffs for a specific target
  skill-sync diff copilot

  # Show diffs for all targets
  skill-sync diff

  # Use inline providers
  skill-sync diff gemini --source claude --targets gemini
```

---

## Architecture / Component Boundaries
All changes are to cobra command struct fields (`Short`, `Long`, `Example`) and error message strings in `RunE` functions. No new files, no structural changes.

**Files touched:**
| File | What changes |
|------|-------------|
| `cmd/root.go` | `Short`, `Long`, `Example` on `rootCmd`; flag descriptions |
| `cmd/init.go` | `Short`, `Long`, `Example` on `initCmd`; error message strings in `runInit` |
| `cmd/sync.go` | `Short`, `Long`, `Example` on `syncCmd`; flag descriptions |
| `cmd/status.go` | `Short`, `Long`, `Example` on `statusCmd` |
| `cmd/diff.go` | `Short`, `Long`, `Example` on `diffCmd` |
| `cmd/helpers.go` | Error message strings in `resolveProviders` (add provider list to "unknown provider" error) |

---

## Correctness Invariants
1. **No behavior changes** -- only string constants and error message formatting change. All logic remains identical.
2. **`Short` descriptions are under 60 characters** -- enforced by manual check.
3. **`Example` blocks use cobra's `Example` field** -- not embedded in `Long`. Cobra renders them with proper indentation.
4. **Error messages include actionable guidance** -- every error tells the user what to do next.
5. **Flag descriptions match actual flag behavior** -- no misleading text.
6. **`go build` and `go test` still pass** -- no compilation breakage from string changes.

---

## Tests
- **Manual verification (primary):**
  - `./skill-sync --help` -- verify root help output
  - `./skill-sync init --help` -- verify init help with examples
  - `./skill-sync sync --help` -- verify sync help with examples and flag descriptions
  - `./skill-sync status --help` -- verify status help with examples
  - `./skill-sync diff --help` -- verify diff help with examples
  - `./skill-sync init` -- verify improved error message (missing --source)
  - `./skill-sync init --source foo --targets bar` -- verify improved unknown provider error
- **Automated:**
  - `go build ./...` -- compilation still works
  - `go vet ./...` -- no issues
  - `go test ./...` -- all existing tests still pass

### Commands
```bash
go build ./...
go vet ./...
go test ./...
./skill-sync --help
./skill-sync init --help
./skill-sync sync --help
./skill-sync status --help
./skill-sync diff --help
```

---

## Benchmarks + "Success"
N/A -- this role modifies string constants only. There is nothing to benchmark. Success is defined by:
1. Every `--help` output has a clear Short, informative Long, and real Example block.
2. Error messages follow the "what + why + fix" pattern.
3. All existing tests still pass (`go test ./...`).

---

## Engineering Decisions & Tradeoffs

### Decision 1: Use cobra's `Example` field instead of embedding examples in `Long`
- **Decision:** Put usage examples in the cobra `Example` field, not inline in `Long`.
- **Alternatives considered:** Embedding examples directly in the `Long` description string.
- **Why:** Cobra renders `Example` with proper indentation under an "Examples:" header, matching the convention of tools like `kubectl`, `gh`, and `docker`. It also keeps `Long` focused on explaining the command rather than showing usage.
- **Tradeoff acknowledged:** The `Long` + `Example` split means two places to maintain. But cobra handles the formatting, and it follows community convention.

### Decision 2: Actionable error messages with available values listed
- **Decision:** Error messages include the list of available providers (e.g., `Available providers: claude, copilot, gemini, factory`) and a corrective example command.
- **Alternatives considered:** Terse error messages (e.g., `unknown provider "foo"`) that match the existing style.
- **Why:** The primary audience is a developer who just installed the tool. They don't know the valid provider names yet. Listing available providers saves a round trip to `--help`. Including an example command makes the fix copy-pasteable.
- **Tradeoff acknowledged:** Error messages become longer. But they are only shown on error paths, and the extra verbosity directly reduces user frustration.

### Decision 3: Keep `Short` descriptions action-oriented and under 60 chars
- **Decision:** Use imperative verb phrases (e.g., "Sync skills from source to all targets") rather than noun phrases (e.g., "Skill synchronization command").
- **Alternatives considered:** Noun-style descriptions matching some cobra projects.
- **Why:** Action-oriented descriptions immediately tell the user what happens when they run the command. The `--help` listing reads naturally: `init  Create a .skill-sync.yaml config file`, `sync  Sync skills from source to all targets`.
- **Tradeoff acknowledged:** Slightly less formal. But this matches the style of `gh`, `docker`, and other popular CLI tools.

---

## Risks & Mitigations

### Risk 1: Error message changes break existing tests
- **Risk:** Existing unit tests in `cmd/` may assert on exact error message strings. Changing error messages could cause test failures.
- **Impact:** Tests fail; need to update test assertions alongside UX changes.
- **Mitigation:** Before modifying error strings, grep all test files for the exact error text being changed. Update test assertions to match new messages. Run `go test ./cmd/...` after each file change.
- **Validation time:** < 5 minutes (grep + test run).

### Risk 2: cobra `Example` field formatting issues
- **Risk:** Multi-line `Example` strings might render with unexpected indentation depending on the Go raw string literal formatting and cobra's internal rendering.
- **Impact:** Ugly or misaligned help output.
- **Mitigation:** Use Go raw string literals (backtick strings) for `Example` to control whitespace precisely. Verify with `./skill-sync <cmd> --help` for each command after writing.
- **Validation time:** < 2 minutes per command.

### Risk 3: Available provider list in errors becomes stale
- **Risk:** Hardcoding provider names in error messages means they go stale if providers are added/removed.
- **Impact:** Error messages list wrong providers.
- **Mitigation:** Use `provider.List()` dynamically in error messages (it returns registered provider names). The `resolveProviders` helper in `cmd/helpers.go` already has access to the registry. For `init`, the validation already calls `provider.List()` — use its return value in the error string.
- **Validation time:** < 3 minutes (read the existing validation flow).

### Risk 4: Long description wrapping in narrow terminals
- **Risk:** `Long` descriptions that are too wide may wrap awkwardly in 80-column terminals.
- **Impact:** Ugly help output.
- **Mitigation:** Keep `Long` descriptions to 2-3 sentences. Test with `COLUMNS=80 ./skill-sync --help`. Cobra wraps automatically but explicit line breaks at ~78 chars ensure clean output.
- **Validation time:** < 2 minutes.

---

## Recommended API Surface

### What changes (no new functions/endpoints)

This role modifies **existing string constants** only. No new functions or API surface.

| File | Field | Current | Proposed |
|------|-------|---------|----------|
| `cmd/root.go` | `Short` | "Sync AI assistant skills from a primary provider to all others" | "Sync AI skills across providers" |
| `cmd/root.go` | `Long` | 1 sentence | 2-3 sentences + positioning |
| `cmd/root.go` | `Example` | (none) | Quick-start workflow |
| `cmd/init.go` | `Short` | "Initialize a .skill-sync.yaml config file" | "Create a .skill-sync.yaml config file" |
| `cmd/init.go` | `Long` | 1 sentence | 2 sentences + when to use |
| `cmd/init.go` | `Example` | (none) | 3 examples |
| `cmd/init.go` | errors | terse | actionable with provider list |
| `cmd/sync.go` | `Short` | "Sync skills from source provider to all targets" | "Sync skills from source to all targets" |
| `cmd/sync.go` | `Long` | 1 sentence | 2 sentences + exit code note |
| `cmd/sync.go` | `Example` | (none) | 4 examples |
| `cmd/sync.go` | flags | minimal | descriptive |
| `cmd/status.go` | `Short` | "Show drift status between source and target providers" | "Show sync drift between providers" |
| `cmd/status.go` | `Long` | 1 sentence | 2 sentences + CI note |
| `cmd/status.go` | `Example` | (none) | 2 examples |
| `cmd/diff.go` | `Short` | "Show unified diffs for modified skills in a target provider" | "Show unified diffs for drifted skills" |
| `cmd/diff.go` | `Long` | 1 sentence | 2 sentences + exit code note |
| `cmd/diff.go` | `Example` | (none) | 3 examples |
| `cmd/helpers.go` | error | "unknown provider" | "unknown provider + available list" |

---

## Folder Structure

No new files or directories. All changes are in-place edits to existing files:

```
cmd/
  root.go          # EDIT: Short, Long, Example, flag descriptions
  init.go          # EDIT: Short, Long, Example, error messages
  sync.go          # EDIT: Short, Long, Example, flag descriptions
  status.go        # EDIT: Short, Long, Example
  diff.go          # EDIT: Short, Long, Example
  helpers.go       # EDIT: error message in resolveProviders
```

---

## Tighten the plan into 4-7 small tasks

### Task 1: Root command help text
- **Outcome:** `skill-sync --help` shows a polished Short, Long, and Example block with a quick-start workflow. Flag descriptions for `--config`, `--source`, `--targets` are improved.
- **Files to modify:** `cmd/root.go`
- **Verification:**
  ```bash
  go build ./... && ./skill-sync --help
  go vet ./...
  ```
- **Commit message:** `polish(cmd): improve root command help text with examples`

### Task 2: Init command help and error messages
- **Outcome:** `skill-sync init --help` shows examples. Error messages for missing `--source`, missing `--targets`, and unknown providers include actionable guidance with available provider names.
- **Files to modify:** `cmd/init.go`
- **Verification:**
  ```bash
  go build ./... && ./skill-sync init --help
  ./skill-sync init 2>&1 | grep -q "Available providers"
  go test ./...
  ```
- **Commit message:** `polish(cmd): improve init help text and error messages`

### Task 3: Sync command help and flag descriptions
- **Outcome:** `skill-sync sync --help` shows examples including `--dry-run` and `--skill` usage. Flag descriptions are clear and descriptive.
- **Files to modify:** `cmd/sync.go`
- **Verification:**
  ```bash
  go build ./... && ./skill-sync sync --help
  go vet ./...
  ```
- **Commit message:** `polish(cmd): improve sync help text with examples and flag descriptions`

### Task 4: Status and diff command help
- **Outcome:** `skill-sync status --help` and `skill-sync diff --help` both show examples and explain exit code behavior.
- **Files to modify:** `cmd/status.go`, `cmd/diff.go`
- **Verification:**
  ```bash
  go build ./... && ./skill-sync status --help && ./skill-sync diff --help
  go vet ./...
  ```
- **Commit message:** `polish(cmd): improve status and diff help text with examples`

### Task 5: Error messages in helpers and validation
- **Outcome:** `resolveProviders` error messages include the list of available providers. All error paths produce actionable messages.
- **Files to modify:** `cmd/helpers.go`
- **Verification:**
  ```bash
  go build ./...
  go test ./...
  ```
- **Commit message:** `polish(cmd): improve error messages with available provider list`

### Task 6: Final review pass
- **Outcome:** All 5 commands verified end-to-end. Every `--help` output reviewed for consistency (verb tense, capitalization, line length). All tests pass.
- **Files to modify:** any of the above (minor tweaks only)
- **Verification:**
  ```bash
  go build ./... && go vet ./... && go test ./...
  ./skill-sync --help
  ./skill-sync init --help
  ./skill-sync sync --help
  ./skill-sync status --help
  ./skill-sync diff --help
  ```
- **Commit message:** `polish(cmd): final consistency pass on CLI help text`

---

## CLAUDE.md contributions (do NOT write the file; propose content)

## From CLI UX Dev
- **CLI help conventions:**
  - `Short` descriptions: imperative verb, under 60 chars, no trailing period
  - `Long` descriptions: 2-3 sentences, explain what + when + key behavior (exit codes)
  - `Example` blocks: use cobra's `Example` field (not inline in `Long`), 2-4 real examples with comments
  - Flag descriptions: describe the effect, not just the type (e.g., "Preview sync without writing" not "enable dry run mode")
- **Error message style:**
  - Always include: what happened, why, what to do about it
  - When a value is invalid, list the valid values (use `provider.List()` dynamically, not hardcoded)
  - Include a corrective example command when possible
- **Before you commit:**
  - Run `./skill-sync <cmd> --help` for every command to visually verify output
  - Check that `Short` is under 60 characters
  - Check that error messages include actionable guidance
  - `go build ./... && go vet ./... && go test ./...` all pass
- **Guardrails:**
  - Never hardcode provider names in error messages -- use `provider.List()` dynamically
  - Never embed examples in `Long` -- use cobra's `Example` field
  - `Short` must not have a trailing period

---

## EXPLAIN.md contributions (do NOT write the file; propose outline bullets)

### CLI UX Philosophy
- Help text follows the "60-second rule": a new user should understand what each command does and how to use it within 60 seconds of running `--help`
- Error messages are designed for copy-paste debugging: they include the failing value, the valid values, and an example fix
- Cobra's `Example` field is used for all usage examples, keeping `Long` focused on explanation

### Key Engineering Decisions + Tradeoffs
- Chose cobra `Example` field over inline examples in `Long` for consistent formatting and community convention alignment
- Error messages dynamically include available providers via `provider.List()` so they never go stale
- `Short` descriptions are action-oriented imperatives (matching `gh`, `docker`, `kubectl` style)

### How to Validate CLI UX
- `./skill-sync --help` -- verify root with quick-start examples
- `./skill-sync <cmd> --help` -- verify each subcommand has Short, Long, and Examples
- Intentionally trigger errors (e.g., `--source foo`) to verify actionable messages
- `COLUMNS=80 ./skill-sync --help` -- verify clean wrapping in narrow terminals

---

## READY FOR APPROVAL
