# Master Plan: CLI Commands Dev

## You are in PLAN MODE.

### Project
I want to build **skill-sync**, a Go CLI that syncs AI assistant skills from a primary provider to others with drift detection.

**Goal:** build the three remaining CLI commands (`sync`, `status`, `diff`) so that the tool is fully functional end-to-end.

### Role + Scope
- **Role:** CLI Commands Dev
- **Scope:** I own `cmd/sync.go`, `cmd/status.go`, `cmd/diff.go` and their corresponding test files. I wire up the existing `SyncEngine` and `DiffEngine` from `internal/sync/` with providers resolved from `internal/provider/registry.go` using config from `cmd.Cfg` (loaded by `root.go`'s `PersistentPreRunE`). I do NOT own provider implementations, the sync/diff engines, config parsing, or the root/init commands.
- **File I will write:** `/docs/providers-and-commands/plans/cli-commands.md`
- **No-touch zones:** do not edit any other files; do not write code.

---

## Functional Requirements
- **FR1:** `skill-sync sync` reads source skills, writes them to all targets, prints a results table, exits 1 on errors.
- **FR2:** `skill-sync status` computes drift between source and all targets, prints a drift report table, exits 1 if any drift detected.
- **FR3:** `skill-sync diff [provider]` prints unified diffs for modified skills in a specific target (or all targets if no argument given).
- **FR4:** `sync` supports `--dry-run` (show what would sync without writing) and `--skill` (filter to specific skill names) flags.
- **Tests required:** unit tests for each command using mock providers (table-driven).

## Non-Functional Requirements
- Language/runtime: Go 1.22+, cobra CLI framework
- Local dev: `go build && ./skill-sync <cmd>`
- Safety: all errors wrapped with `fmt.Errorf("context: %w", err)`; `RunE` (not `Run`) for error propagation
- Documentation: CLAUDE.md + EXPLAIN.md contributions proposed below
- Performance: N/A — CLI commands are thin wiring layers; performance is in the engines

---

## Assumptions / System Model
- **Config is always loaded** before sync/status/diff run (handled by `root.go` `PersistentPreRunE`). `cmd.Cfg` is non-nil.
- **Provider registry is populated** by the time commands run (providers register via `init()` or explicit registration in `main.go`).
- **SyncEngine.Sync()** returns fatal error only if source `ListSkills` fails; per-skill errors are in `SyncResult.Details`.
- **DiffEngine.Status()** returns fatal error if source `ListSkills` or any target comparison fails.
- **DiffEngine.Diff(targetName)** returns error if `targetName` is unknown.
- Exit code 1 for any drift (status) or any sync errors (sync) enables CI usage.

---

## Data Model
N/A — not in scope for this role. CLI commands consume existing `config.Config`, `sync.SyncResult`, `sync.DriftReport`, and `sync.DetailedDiff` types.

---

## APIs (CLI Command Surface)

### `skill-sync sync`
```
Usage: skill-sync sync [flags]

Flags:
  --dry-run         Show what would be synced without writing
  --skill strings   Filter to specific skill names (repeatable)

Output (stdout):
  SKILL        TARGET     STATUS
  deploy       copilot    synced
  deploy       gemini     synced
  review       copilot    error: permission denied

  Synced: 2  Errors: 1

Exit code: 0 if all synced, 1 if any errors
```

### `skill-sync status`
```
Usage: skill-sync status

Output (stdout):
  Target: copilot
  SKILL        STATUS
  deploy       [ok] in-sync
  review       [!] modified
  build        [-] missing

  Target: gemini
  SKILL        STATUS
  deploy       [ok] in-sync
  extra-skill  [+] extra

Exit code: 0 if all in-sync, 1 if any drift
```

### `skill-sync diff [provider]`
```
Usage: skill-sync diff [provider]

Args:
  provider   Target provider name (optional; if omitted, show all targets)

Output (stdout):
  --- a/review
  +++ b/review
  @@ -1,3 +1,3 @@
   line1
  -original line
  +modified line
   line3

Exit code: 0 always (informational command)
```

---

## Architecture / Component Boundaries

Each command file follows the same pattern:

1. **Resolve providers:** Use `provider.Get(cmd.Cfg.Source)` for source, loop `cmd.Cfg.Targets` for targets via `provider.Get()`.
2. **Build engine:** Construct `sync.NewSyncEngine()` or `sync.NewDiffEngine()` with resolved providers.
3. **Call engine method:** `Sync()`, `Status()`, or `Diff()`.
4. **Format output:** Print human-readable table to stdout.
5. **Set exit code:** Return error from `RunE` to trigger non-zero exit (cobra handles this).

For `--dry-run` in sync: resolve providers and list source skills (applying `--skill` filter), but skip the `WriteSkill` calls. Implementation approach: call `source.ListSkills()` directly, apply filter, print the plan table showing each skill x target with status "would sync", then return. This avoids needing a dry-run flag on the engine itself.

Concurrency model: sequential (commands are thin wiring; engines are synchronous).

---

## Correctness Invariants
1. **Config is validated before command runs** — guaranteed by `PersistentPreRunE`. Commands can assume `Cfg` is valid.
2. **Unknown provider names in config produce errors at validation time**, not at command time.
3. **`sync --dry-run` never writes to disk** — must not call `SyncEngine.Sync()` or any `WriteSkill`.
4. **`status` exits 1 if ANY skill in ANY target is not `InSync`** — useful for CI gating.
5. **`diff` with no args iterates all targets** — no target is silently skipped.
6. **`diff` with unknown provider arg returns error** — propagated from `DiffEngine.Diff()`.

---

## Tests

### Unit Tests (table-driven, mock providers)

**`cmd/sync_test.go`:**
- Test sync with no flags — all skills synced, table output contains "synced"
- Test sync with `--dry-run` — output contains "would sync", no writes to target
- Test sync with `--skill` filter — only filtered skills appear in output
- Test sync with write errors — output contains "error", exit code 1
- Test sync with empty source — no output rows, exit 0

**`cmd/status_test.go`:**
- Test all in-sync — all rows show `[ok]`, exit 0
- Test mixed drift — correct symbols per status, exit 1
- Test empty source with extras — shows `[+]` for extras, exit 1
- Test multiple targets — each target has its own section

**`cmd/diff_test.go`:**
- Test diff with specific target — shows unified diff for modified skills only
- Test diff with no args — shows diffs for all targets
- Test diff with unknown target — returns error
- Test diff with no modified skills — no diff output, exit 0

### Commands
```bash
go test ./cmd/... -v
go test ./cmd/... -run TestSync
go test ./cmd/... -run TestStatus
go test ./cmd/... -run TestDiff
```

Tests do NOT go through the global provider registry (it uses unexported `resetRegistry()`). Instead, each `runX` function is structured to accept resolved providers or an engine, and tests construct `SyncEngine`/`DiffEngine` directly with mock `provider.Provider` implementations. For cobra integration, tests use `rootCmd.SetArgs()` and capture stdout via `bytes.Buffer` on `rootCmd.SetOut()`.

---

## Benchmarks + "Success"
N/A — CLI commands are thin wiring layers over the sync/diff engines. The engines already have their own benchmarks. The CLI layer adds negligible overhead (string formatting for table output). Success is defined by correctness (tests pass) and UX (output is clear and consistent).

---

## Engineering Decisions & Tradeoffs

### Decision 1: Dry-run implemented in the command layer, not the engine
- **Decision:** `--dry-run` lists source skills and prints a plan table without calling `SyncEngine.Sync()`.
- **Alternatives considered:** Adding a `DryRun bool` field to `SyncEngine` that skips `WriteSkill` calls internally.
- **Why:** The engine is already built and tested. Adding dry-run logic to it would require modifying `internal/sync/engine.go` (outside my scope) and would complicate the engine's single responsibility. The command layer can implement dry-run by simply listing skills and formatting output.
- **Tradeoff acknowledged:** Dry-run output won't perfectly mirror sync output (e.g., it can't predict per-target write errors). This is acceptable — dry-run shows intent, not prediction.

### Decision 2: Table output with symbols rather than plain text status
- **Decision:** Use `[ok]`, `[!]`, `[-]`, `[+]` symbols in status output for quick visual scanning.
- **Alternatives considered:** Color-coded output using ANSI escape codes; plain text labels only.
- **Why:** Symbols work in all terminals including CI logs and piped output. ANSI colors break in non-TTY contexts and add a dependency (or manual escape code handling). Plain text is harder to scan.
- **Tradeoff acknowledged:** Symbols are less discoverable than color for new users. Mitigated by also printing the text label (e.g., `[!] modified`).

### Decision 3: Exit code semantics
- **Decision:** `sync` exits 1 on any error; `status` exits 1 on any drift; `diff` always exits 0.
- **Alternatives considered:** All commands exit 0 and use a `--strict` flag for CI.
- **Why:** The primary use case for `status` is CI drift detection — exit 1 on drift is the expected behavior. `diff` is informational (like `git diff` which exits 0). This matches Unix conventions.
- **Tradeoff acknowledged:** Users who want `status` to be informational-only must use `|| true` in their scripts. This is standard Unix practice.

---

## Risks & Mitigations

### Risk 1: Provider registry not populated when commands run
- **Risk:** If providers register via `init()` but the import is missing from `main.go`, `provider.Get()` fails at runtime.
- **Impact:** All three commands fail with "unknown provider" errors despite valid config.
- **Mitigation:** Verify that `main.go` (or a `cmd/` init file) imports all provider packages for side effects. Add a smoke test that creates a config and runs each command.
- **Validation time:** < 5 minutes (check imports + run one command).

### Risk 2: SyncEngine/DiffEngine API changes from Phase 1
- **Risk:** The engine API I'm coding against could differ from what was actually merged.
- **Impact:** Commands won't compile.
- **Mitigation:** Read the actual source files (`internal/sync/engine.go`, `internal/sync/diff.go`) before writing code — already done. Pin to the exact method signatures: `Sync(skillFilter []string) (*SyncResult, error)`, `Status() (*DriftReport, error)`, `Diff(targetName string) (*DetailedDiff, error)`.
- **Validation time:** < 2 minutes (go build).

### Risk 3: Cobra command registration conflicts
- **Risk:** Multiple `init()` functions in `cmd/` package registering commands could conflict or cause ordering issues.
- **Impact:** Panic at startup or commands not registered.
- **Mitigation:** Follow exact pattern from `cmd/init.go`: define command var, register in `init()` via `rootCmd.AddCommand()`. Each file registers exactly one command.
- **Validation time:** < 2 minutes (go build + `./skill-sync --help`).

### Risk 4: Test isolation with global provider registry
- **Risk:** Tests that register mock providers pollute the global registry, causing flaky tests or panics from duplicate registration. Additionally, `resetRegistry()` in `internal/provider/registry.go` is **unexported** — `cmd/` package tests cannot call it.
- **Impact:** Test failures in CI; inability to isolate provider state between tests.
- **Mitigation:** Do NOT test through the registry. Instead, construct `SyncEngine`/`DiffEngine` directly with mock providers (they accept `provider.Provider` interfaces). Each `runX` function should accept resolved providers or the engine as parameters (thin wrapper pattern), and tests inject mocks at that layer. This avoids the registry entirely and gives full isolation.
- **Validation time:** < 5 minutes (run tests twice in succession).

---

## Recommended API Surface

### Functions/Commands

| Command | File | Cobra Var | RunE Function |
|---------|------|-----------|---------------|
| `sync` | `cmd/sync.go` | `syncCmd` | `runSync(cmd, args)` |
| `status` | `cmd/status.go` | `statusCmd` | `runStatus(cmd, args)` |
| `diff` | `cmd/diff.go` | `diffCmd` | `runDiff(cmd, args)` |

Each `runX` function:
1. Calls `resolveProviders()` (helper in each file or shared) to get source + targets from `Cfg` + registry
2. Builds the appropriate engine
3. Delegates to an internal `doX(engine, writer, flags)` function that does the actual work
4. `doX` calls the engine method, formats output to `io.Writer`, returns error or nil

This separation allows tests to call `doX` directly with mock engines/providers, bypassing the registry entirely. The cobra `RunE` is a thin wrapper that resolves providers and calls `doX`.

### Helper: `resolveProviders`
```
func resolveProviders(cfg *config.Config) (source provider.Provider, targets []provider.Provider, err error)
```
Shared across all three commands. Lives in `cmd/helpers.go` or duplicated inline (simple enough to inline — 10 lines).

---

## Folder Structure

```
cmd/
  root.go          # existing — PersistentPreRunE, config loading
  init.go          # existing — init command
  sync.go          # NEW — sync command + --dry-run, --skill flags
  status.go        # NEW — status command
  diff.go          # NEW — diff command
  helpers.go       # NEW — resolveProviders helper (shared by all 3)
  sync_test.go     # NEW — sync command tests
  status_test.go   # NEW — status command tests
  diff_test.go     # NEW — diff command tests
```

No changes to `internal/` or any other directories.

---

## Tighten the plan into 4-7 small tasks

### Task 1: Add `resolveProviders` helper
- **Outcome:** Shared helper that resolves source + target providers from config using the registry.
- **Files to create/modify:** `cmd/helpers.go`
- **Verification:**
  ```bash
  go build ./cmd/...
  go vet ./cmd/...
  ```
- **Commit message:** `feat(cmd): add resolveProviders helper for CLI commands`

### Task 2: Implement `sync` command
- **Outcome:** `skill-sync sync` works with `--dry-run` and `--skill` flags, prints results table, exits 1 on errors.
- **Files to create/modify:** `cmd/sync.go`
- **Verification:**
  ```bash
  go build ./...
  go vet ./...
  ./skill-sync sync --help
  ```
- **Commit message:** `feat(cmd): add sync command with --dry-run and --skill flags`

### Task 3: Implement `status` command
- **Outcome:** `skill-sync status` prints drift report with symbols, exits 1 on drift.
- **Files to create/modify:** `cmd/status.go`
- **Verification:**
  ```bash
  go build ./...
  go vet ./...
  ./skill-sync status --help
  ```
- **Commit message:** `feat(cmd): add status command with drift report output`

### Task 4: Implement `diff` command
- **Outcome:** `skill-sync diff [provider]` prints unified diffs for modified skills. No arg = all targets.
- **Files to create/modify:** `cmd/diff.go`
- **Verification:**
  ```bash
  go build ./...
  go vet ./...
  ./skill-sync diff --help
  ```
- **Commit message:** `feat(cmd): add diff command with optional provider argument`

### Task 5: Add unit tests for all three commands
- **Outcome:** Table-driven tests for sync, status, diff commands using mock providers and cobra test harness.
- **Files to create/modify:** `cmd/sync_test.go`, `cmd/status_test.go`, `cmd/diff_test.go`
- **Verification:**
  ```bash
  go test ./cmd/... -v
  go test ./cmd/... -count=2
  ```
- **Commit message:** `test(cmd): add unit tests for sync, status, and diff commands`

---

## CLAUDE.md contributions (do NOT write the file; propose content)

## From CLI Commands Dev
- **Coding style:** All cobra commands use `RunE` (not `Run`). Errors are returned, never printed directly. Use `fmt.Errorf("context: %w", err)` wrapping.
- **Dev commands:**
  - `go build ./...` — verify compilation
  - `go test ./cmd/... -v` — run CLI command tests
  - `go vet ./...` — static analysis
  - `./skill-sync sync --help` — verify flag registration
- **Before you commit:**
  - `go build ./...` passes
  - `go vet ./...` passes
  - `go test ./cmd/... -v` passes
  - All three commands appear in `./skill-sync --help`
- **Guardrails:**
  - Never print errors to stderr directly; return them from `RunE` (cobra handles stderr output)
  - `--dry-run` must NEVER call `WriteSkill` or `SyncEngine.Sync()`
  - Tests must NOT use the global provider registry; inject mock providers directly into engine constructors

---

## EXPLAIN.md contributions (do NOT write the file; propose outline bullets)

### Flow / Architecture
- CLI commands are thin wiring: resolve providers from config + registry, build engine, call method, format output
- `PersistentPreRunE` in `root.go` loads and validates config before any command runs
- Each command is a single file in `cmd/` with a cobra command var, `init()` registration, and `RunE` function

### Key Engineering Decisions + Tradeoffs
- Dry-run is implemented in the command layer (not the engine) to avoid modifying Phase 1 code
- Status exit code 1 on drift enables CI gating without extra flags
- Symbols (`[ok]`, `[!]`, `[-]`, `[+]`) chosen over ANSI colors for universal terminal compatibility

### Limits of MVP + Next Steps
- No `--format json` flag for machine-readable output (could be added later)
- No `--verbose` flag for detailed sync logging
- No `--force` flag to overwrite even if target is newer
- Dry-run can't predict per-target write errors

### How to Run Locally + How to Validate
- `go build ./... && ./skill-sync sync` — run a sync
- `./skill-sync status` — check drift (exit 1 = drift detected)
- `./skill-sync diff copilot` — see diffs for a specific target
- `go test ./cmd/... -v` — run all CLI command tests

---

## READY FOR APPROVAL
