# Smoke Test Dev Plan

## You are in PLAN MODE.

### Project
I want to build a **Go integration smoke test** for skill-sync.

**Goal:** build a **smoke test suite** in which we **exercise the full sync/status/diff flow using real provider implementations against temp directories, verifying the happy path end-to-end**.

### Role + Scope
- **Role:** Smoke Test Dev
- **Scope:** Write `tests/smoke_test.go` and `tests/testdata/` sample skills. I own the integration smoke test and sample fixture files. I do NOT own the manual test plan (QE Lead), edge case documentation (Edge Case Analyst), or any production code changes.
- **File you will write:** `/docs/validation/plans/smoke-test.md`
- **No-touch zones:** do not edit any other files; do not write code.

---

## Functional Requirements
- FR1: A Go test file (`tests/smoke_test.go`) that exercises the full Sync -> Status -> Diff flow using real provider implementations with temp directories.
- FR2: Sample skill fixtures in `tests/testdata/` covering: simple markdown, markdown with `# ` description header, markdown with `$ARGUMENTS` / `${NAME}` placeholders.
- FR3: The smoke test must verify: (a) skills are synced to all target provider directories in correct format, (b) drift detection works when a target skill is modified, (c) diff output is non-empty for modified skills.
- FR4: The smoke test must use `t.TempDir()` for all file I/O -- no cleanup code needed, no side effects on the host system.
- Tests required: The smoke test IS the test. It runs via `go test ./tests/ -v -run TestSmoke`.
- Metrics required: N/A -- this is a test artifact, not a service.

## Non-Functional Requirements
- Language/runtime: Go 1.22+
- Local dev: `go test ./tests/ -v` from the repo root
- Observability: N/A
- Safety: All file operations use `t.TempDir()` -- no writes to real provider directories
- Documentation: This plan doc; test code is self-documenting via descriptive assertions
- Performance: N/A -- smoke test should complete in < 2 seconds

---

## Assumptions / System Model
- Deployment environment: Local developer machine, CI runner
- Failure modes: Provider directory doesn't exist (handled by `t.TempDir()`), file permission issues (not tested in smoke test -- edge case doc covers this)
- Delivery guarantees: N/A
- Multi-tenancy: N/A

---

## Data Model (as relevant to your role)

### Test Fixtures (`tests/testdata/`)

Three sample Claude Code skill files:

- **`simple.md`** -- Plain markdown content, no description header, no arguments.
  - Content: `"Just do the thing.\nNo description, no arguments."`

- **`deploy.md`** -- Has a `# ` description header.
  - Content: `"# Deploy to production\nRun the deploy pipeline for the current branch."`

- **`search.md`** -- Has description + `$ARGUMENTS` and `${PROJECT}` placeholders.
  - Content: `"# Search codebase\nSearch for $ARGUMENTS in ${PROJECT} across all files."`

### Provider Directory Structure (created at test time)

| Provider | Base Dir Pattern | File Pattern |
|----------|-----------------|--------------|
| Claude (source) | `<tmpdir>/claude-commands/` | `<name>.md` |
| Copilot (target) | `<tmpdir>/copilot-prompts/` | `<name>.prompt.md` |
| Gemini (target) | `<tmpdir>/gemini-commands/` | `<name>.toml` |

Validation rules:
- Copilot files must exist at `<base>/<name>.prompt.md`
- Gemini files must exist at `<base>/<name>.toml` and contain valid TOML with a `prompt` field
- Skill names derived from source filenames

Versioning strategy: N/A -- test fixtures are static.

---

## APIs (as relevant to your role)

The smoke test calls these internal APIs directly (no CLI binary execution):

### Provider Construction
- `provider.NewClaudeProvider(provider.WithBaseDir(dir))` -- source provider
- `provider.NewCopilotProvider(provider.WithCopilotBaseDir(dir))` -- target provider
- `provider.NewGeminiProvider(provider.WithGeminiBaseDir(dir))` -- target provider

### Sync Engine
- `sync.NewSyncEngine(source, targets)` -- construct engine
- `engine.Sync(nil)` -- sync all skills (no filter)
- Returns `*SyncResult` with `TotalSynced`, `TotalErrored`, `Details []SyncDetail`

### Diff Engine
- `sync.NewDiffEngine(source, targets)` -- construct engine
- `engine.Status()` -- returns `*DriftReport` with `Results map[string][]SkillDrift`
- `engine.Diff(targetName)` -- returns `*DetailedDiff` with `Diffs []SkillDrift`

### Key Types
- `provider.Skill{Name, Description, Content, Arguments, SourcePath}`
- `provider.SkillStatus` enum: `InSync`, `Modified`, `MissingInTarget`, `ExtraInTarget`
- `sync.SkillDrift{SkillName, Status, UnifiedDiff}`

---

## Architecture / Component Boundaries

### Test File Layout
```
tests/
  smoke_test.go       # Main smoke test file
  testdata/
    simple.md          # Fixture: plain markdown
    deploy.md          # Fixture: with description header
    search.md          # Fixture: with $ARGUMENTS placeholders
```

### Test Flow (single function: `TestSmoke_FullFlow`)

```
Phase 1: Setup
  - Create temp dirs for Claude source, Copilot target, Gemini target
  - Copy testdata fixtures into Claude source dir

Phase 2: Sync
  - Construct providers with temp base dirs
  - Run SyncEngine.Sync(nil)
  - Assert TotalSynced == 6 (3 skills x 2 targets), TotalErrored == 0
  - Assert Copilot dir contains 3 .prompt.md files
  - Assert Gemini dir contains 3 .toml files

Phase 3: Status (all in-sync)
  - Run DiffEngine.Status()
  - Assert all skills report InSync for both targets

Phase 4: Introduce Drift
  - Overwrite one Copilot target file with modified content
  - Run DiffEngine.Status() again
  - Assert that one skill shows Modified for copilot, others still InSync
  - Assert Gemini target still shows all InSync

Phase 5: Diff
  - Run DiffEngine.Diff("copilot")
  - Assert Diffs slice has exactly 1 entry
  - Assert UnifiedDiff is non-empty and contains --- / +++ headers

Phase 6: Skill Filter
  - Create fresh providers and SyncEngine
  - Run Sync([]string{"deploy"}) with filter
  - Assert only "deploy" was synced (TotalSynced == 2, one per target)
```

Concurrency model: Single-goroutine test -- no concurrency needed.
Backpressure: N/A.

---

## Correctness Invariants

These invariants will be verified by assertions in the smoke test:

1. After `Sync(nil)`, every source skill has a corresponding file in every target directory, in that provider's format.
2. After sync with no modifications, `Status()` reports `InSync` for all skills in all targets.
3. After modifying exactly one target skill file, `Status()` reports `Modified` for that skill in that target, and `InSync` for all others.
4. `Diff(targetName)` returns a non-empty `UnifiedDiff` for every `Modified` skill.
5. `Sync(filter)` with a single-skill filter syncs only that skill -- `TotalSynced` equals the number of targets.
6. Copilot files use `.prompt.md` extension; Gemini files use `.toml` extension.

---

## Tests

### Unit tests
N/A -- existing unit tests in `internal/provider/` and `internal/sync/` cover individual components. This smoke test is the integration layer on top.

### Integration tests
- **File:** `tests/smoke_test.go`
- **Package:** `package tests` (external test package, imports internal packages)
- **Test function:** `TestSmoke_FullFlow` -- exercises the complete sync -> status -> diff pipeline
- **Optional sub-test:** `TestSmoke_SkillFilter` -- verifies filtered sync

### Commands
```bash
# Run smoke test
go test ./tests/ -v -run TestSmoke

# Run all tests (unit + smoke)
go test ./...

# Run with race detector
go test ./tests/ -v -race -run TestSmoke
```

### Failure injection tests
N/A -- out of scope for smoke test. Edge cases doc covers failure scenarios.

### Property/fuzz tests
N/A -- not applicable for a smoke test.

---

## Benchmarks + "Success"

N/A -- this is a correctness test, not a performance test. The smoke test passes if all assertions hold. There is no throughput or latency target.

Success criteria: `go test ./tests/ -v -run TestSmoke` exits with `PASS` and zero test failures.

---

## Engineering Decisions & Tradeoffs

### Decision 1: Use real providers, not mocks
- **Decision:** Instantiate real `ClaudeProvider`, `CopilotProvider`, `GeminiProvider` with custom base dirs pointing to `t.TempDir()` subdirectories.
- **Alternatives considered:** Using the `mockProvider` from `engine_test.go` (already exists in the unit tests).
- **Why:** The whole point of a smoke test is to exercise real I/O -- file creation, format translation (especially Gemini TOML), and directory structure. Mocks would duplicate what unit tests already cover. We want to catch integration bugs (e.g., Gemini TOML encoding producing invalid files, Copilot `.prompt.md` extension handling, etc.).
- **Tradeoff acknowledged:** Real file I/O makes the test slightly slower than mock-based tests and introduces filesystem dependencies. Mitigated by `t.TempDir()` isolation.

### Decision 2: Single composite test function vs. separate test functions
- **Decision:** Use one primary test function (`TestSmoke_FullFlow`) with sequential phases, plus one separate function for skill filter (`TestSmoke_SkillFilter`).
- **Alternatives considered:** Separate test functions for each phase (sync, status, diff) with independent setup.
- **Why:** The phases are inherently sequential -- you can't test drift detection without first syncing. A single function avoids redundant setup and makes the test read like a scenario walkthrough. The skill filter test is separate because it has independent setup.
- **Tradeoff acknowledged:** A failure in Phase 2 (Sync) will cause later phases to fail as well, making it harder to isolate which phase broke. Mitigated by clear `t.Fatalf` messages at each phase boundary.

### Decision 3: Skip Factory provider in smoke test
- **Decision:** Test with Claude (source) -> Copilot + Gemini (targets). Do not include Factory as a target.
- **Alternatives considered:** Including all 4 providers.
- **Why:** Claude + Copilot covers markdown-to-markdown sync. Claude + Gemini covers the most complex translation (markdown-to-TOML). Factory adds YAML frontmatter but the translation pattern is similar to Copilot. Including 2 targets is sufficient to verify multi-target sync behavior without over-testing. Keeps the smoke test focused.
- **Tradeoff acknowledged:** Factory-specific bugs (frontmatter serialization, `<name>/SKILL.md` directory structure) won't be caught by this smoke test. Unit tests in `factory_test.go` cover those.

### Decision 4: Copy fixtures from testdata vs. inline strings
- **Decision:** Store fixtures as files in `tests/testdata/` and copy them into temp dirs at test time using `os.ReadFile` + `os.WriteFile`.
- **Alternatives considered:** Defining skill content as inline string constants in the test file.
- **Why:** Separate fixture files make the test content easier to review and modify. They also serve as example skills for anyone reading the test. The fixtures are small (3 files) so the overhead is negligible.
- **Tradeoff acknowledged:** Adds a dependency on the `testdata/` directory existing relative to the test file. Mitigated by Go's standard `testdata` convention which is well-understood.

---

## Risks & Mitigations

### Risk 1: Gemini TOML round-trip content mismatch
- **Risk:** When Claude content is synced to Gemini, it's stored in the TOML `prompt` field. The Gemini provider's `ReadSkill` trims whitespace from the prompt. The DiffEngine's `normalizeContent` also trims trailing whitespace. But there could be subtle whitespace differences that cause a false `Modified` status after sync.
- **Impact:** Smoke test Phase 3 (verify all-in-sync after sync) would fail, blocking validation.
- **Mitigation:** Read the Gemini TOML file back after sync and compare the `prompt` field value against the original content. If mismatch is found, it's a real bug in the Gemini provider that needs fixing.
- **Validation time:** < 5 minutes -- run the sync phase and inspect Gemini output.

### Risk 2: Import path issues with `tests/` package
- **Risk:** The `tests/` package at the repo root importing `github.com/user/skill-sync/internal/...` packages may have module path resolution issues, since it's not under `internal/` itself.
- **Impact:** `go test ./tests/` would fail to compile.
- **Mitigation:** Verify that Go allows a `tests/` package at module root to import `internal/` packages. Go's `internal/` visibility rule allows access from packages rooted at the parent of `internal/` -- since `tests/` is a sibling of `internal/`, it should work. Validate with `go build ./tests/` before running.
- **Validation time:** < 2 minutes -- `go build ./tests/`.

### Risk 3: Provider `init()` registration conflicts
- **Risk:** Each provider file has an `init()` function that calls `Register()` with default base dirs (e.g., `~/.claude/commands/`). Importing the `provider` package in the smoke test triggers these registrations. This isn't a problem for test execution (we construct providers with custom dirs), but it could panic if run in a weird order.
- **Impact:** Test binary fails to start with "duplicate registration" panic.
- **Mitigation:** The smoke test constructs providers directly via `New*Provider(WithBaseDir(...))` and passes them to `NewSyncEngine` / `NewDiffEngine`. It does NOT use the global registry (`provider.Get`). The `init()` registrations happen once and are harmless. No conflict.
- **Validation time:** < 1 minute -- compile and run the test.

### Risk 4: Testdata fixtures not found at runtime
- **Risk:** When running `go test ./tests/`, the working directory is set to the `tests/` package directory. `testdata/` files are accessed relative to this. If the test is run from a different directory or the testdata path is wrong, fixture loading fails.
- **Impact:** All tests fail at setup phase with "file not found" errors.
- **Mitigation:** Use Go's standard `testdata` convention. Access fixtures via relative path `testdata/<name>.md` which Go test framework resolves relative to the test file's package directory. Verify with a quick `go test ./tests/ -v -run TestSmoke` early.
- **Validation time:** < 2 minutes.

---

## Recommended API Surface

The smoke test does not expose an API. It consumes these internal APIs:

| Package | Function/Method | Purpose in Smoke Test |
|---------|----------------|----------------------|
| `provider` | `NewClaudeProvider(WithBaseDir(dir))` | Create source provider |
| `provider` | `NewCopilotProvider(WithCopilotBaseDir(dir))` | Create target provider |
| `provider` | `NewGeminiProvider(WithGeminiBaseDir(dir))` | Create target provider |
| `sync` | `NewSyncEngine(src, targets)` | Create sync engine |
| `sync` | `engine.Sync(filter []string)` | Execute sync |
| `sync` | `NewDiffEngine(src, targets)` | Create diff engine |
| `sync` | `engine.Status()` | Get drift report |
| `sync` | `engine.Diff(targetName)` | Get unified diffs |

---

## Folder Structure

```
tests/
  smoke_test.go              # Package: tests
  testdata/
    simple.md                 # Plain markdown skill fixture
    deploy.md                 # Skill with # description header
    search.md                 # Skill with $ARGUMENTS / ${PROJECT}
```

Ownership: Smoke Test Dev owns all files under `tests/`.

---

## Step-by-Step Task Plan

### Task 1: Create test fixtures
- **Outcome:** Three sample skill files exist in `tests/testdata/`.
- **Files to create:** `tests/testdata/simple.md`, `tests/testdata/deploy.md`, `tests/testdata/search.md`
- **Verification:** `ls tests/testdata/` shows 3 `.md` files; `cat` each to confirm content.
- **Commit message:** `test: add smoke test fixtures for skill-sync validation`

### Task 2: Scaffold smoke_test.go with setup helpers
- **Outcome:** `tests/smoke_test.go` compiles. Contains helper functions: `copyFixtures(t, srcDir)` to copy testdata into a temp Claude source dir, and `createProviders(t)` to create source + target providers with temp dirs.
- **Files to create:** `tests/smoke_test.go`
- **Verification:** `go build ./tests/` succeeds with no errors.
- **Commit message:** `test: scaffold smoke test with setup helpers`

### Task 3: Implement TestSmoke_FullFlow (sync + status + diff)
- **Outcome:** `TestSmoke_FullFlow` exercises: sync all skills -> verify files exist -> verify all-in-sync status -> introduce drift -> verify Modified status -> verify diff output.
- **Files to modify:** `tests/smoke_test.go`
- **Verification:** `go test ./tests/ -v -run TestSmoke_FullFlow` passes.
- **Commit message:** `test: implement full-flow smoke test for sync, status, and diff`

### Task 4: Implement TestSmoke_SkillFilter
- **Outcome:** `TestSmoke_SkillFilter` verifies that `Sync([]string{"deploy"})` only syncs the "deploy" skill.
- **Files to modify:** `tests/smoke_test.go`
- **Verification:** `go test ./tests/ -v -run TestSmoke_SkillFilter` passes.
- **Commit message:** `test: add skill filter smoke test`

### Task 5: Run full test suite and verify no regressions
- **Outcome:** All existing unit tests and new smoke tests pass together.
- **Files to modify:** None.
- **Verification:** `go test ./... -v` passes. `go test ./tests/ -v -race -run TestSmoke` passes.
- **Commit message:** N/A -- verification only, no code change.

---

## CLAUDE.md contributions (do NOT write the file; propose content)

## From Smoke Test Dev
- **Coding style:** Smoke tests use `t.Fatalf` at phase boundaries to fail fast. Use `t.Errorf` for non-fatal assertions within a phase. Name test functions `TestSmoke_<Scenario>`.
- **Dev commands:**
  ```bash
  # Run smoke tests only
  go test ./tests/ -v -run TestSmoke

  # Run with race detector
  go test ./tests/ -v -race -run TestSmoke

  # Run all tests (unit + smoke)
  go test ./...
  ```
- **Before you commit checklist:**
  - [ ] `go test ./...` passes (includes both unit and smoke tests)
  - [ ] `go vet ./...` clean
  - [ ] Smoke test fixtures in `tests/testdata/` are not modified without updating test assertions
- **Guardrails:** Do not add slow operations (network calls, large file generation) to smoke tests. Smoke tests must complete in < 5 seconds.

---

## EXPLAIN.md contributions (do NOT write the file; propose outline bullets)

- **Smoke test purpose:** Validates that the sync engine, diff engine, and real provider implementations work together end-to-end using temporary directories.
- **Test architecture:** Single-flow test that mirrors a real user workflow: sync skills from Claude -> Copilot + Gemini, verify sync succeeded, introduce drift, verify drift detection, verify diff output.
- **Key decision:** Uses real providers (not mocks) to catch integration bugs like TOML encoding issues or file extension mismatches.
- **Fixture design:** Three sample skills covering plain content, description headers, and argument placeholders -- the most common skill patterns.
- **Limits of MVP:** Does not test Factory provider, CLI binary execution, or error paths. Those are covered by unit tests and the edge case document.
- **How to run:** `go test ./tests/ -v -run TestSmoke` from repo root.
- **How to validate:** All assertions pass, exit code 0. Check verbose output to see each phase complete.

---

## READY FOR APPROVAL
