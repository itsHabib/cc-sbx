# Sync Engine Dev -- Plan Document

## You are in PLAN MODE.

### Project
I want to build a **skill-sync CLI tool**.

**Goal:** build a **sync and diff engine** in which we **read skills from a source provider, write them to target providers, and detect drift between source and targets**.

### Role + Scope (fill in)
- **Role:** Sync Engine Dev
- **Scope:** I own `internal/sync/engine.go`, `internal/sync/diff.go`, and `internal/sync/engine_test.go`. I do NOT own the Provider interface, the registry, any specific provider implementation, config parsing, or CLI commands.
- **File you will write:** `/docs/core/plans/sync-engine.md`
- **No-touch zones:** `internal/provider/*`, `internal/config/*`, `cmd/*`, `main.go`, `go.mod`

---

## Functional Requirements
- FR1: `SyncEngine.Sync(skillFilter)` reads skills from the source provider and writes each to every target provider, returning a structured `SyncResult`.
- FR2: When `skillFilter` is non-empty, only skills whose names appear in the filter are synced.
- FR3: `DiffEngine.Status()` compares source skills against all targets and returns a `DriftReport` with per-skill status: `InSync`, `Modified`, `MissingInTarget`, `ExtraInTarget`.
- FR4: `DiffEngine.Diff(targetName)` produces a `DetailedDiff` containing unified diff strings for each modified skill in a specific target.
- FR5: Sync continues on per-skill errors (does not abort the entire sync on a single WriteSkill failure); errors are collected into the result.
- Tests required: unit tests covering all sync and diff behaviors (table-driven).
- Metrics required: N/A -- no Prometheus in this phase; success is measured by test pass rate and correct result structs.

## Non-Functional Requirements
- Language/runtime: Go 1.22+
- Local dev: `go test ./internal/sync/...`
- Observability: N/A for this package; engine returns structured results that CLI layer can render.
- Safety: All errors wrapped with `fmt.Errorf("context: %w", err)`. No panics. Partial failures collected, not swallowed.
- Documentation: Exported types and functions will have GoDoc comments.
- Performance: N/A for MVP -- skill counts are small (tens, not thousands).

---

## Assumptions / System Model
- Deployment environment: Local CLI binary; no containers or network involved.
- Failure modes: Provider `ListSkills`/`ReadSkill`/`WriteSkill` can return errors (missing dir, permission denied, corrupt file). The sync engine must handle these gracefully per-skill.
- Delivery guarantees: Sync is idempotent -- running it twice with unchanged source produces the same target state.
- Multi-tenancy: None. Single user, single config.

---

## Data Model (as relevant to your role)

### SyncResult
```go
type SyncStatus string
const (
    SyncSuccess SyncStatus = "success"
    SyncError   SyncStatus = "error"
)

type SyncDetail struct {
    SkillName string
    Target    string
    Status    SyncStatus
    Error     error // nil on success
}

type SyncResult struct {
    TotalSynced  int
    TotalErrored int
    Details      []SyncDetail
}
```

### DriftReport / SkillDrift
```go
type DriftStatus string
const (
    InSync          DriftStatus = "in_sync"
    Modified        DriftStatus = "modified"
    MissingInTarget DriftStatus = "missing_in_target"
    ExtraInTarget   DriftStatus = "extra_in_target"
)

type SkillDrift struct {
    SkillName   string
    Status      DriftStatus
    UnifiedDiff string // populated only when Status == Modified
}

type DriftReport struct {
    Results map[string][]SkillDrift // key = target provider name
}
```

### DetailedDiff
```go
type DetailedDiff struct {
    TargetName string
    Diffs      []SkillDrift // only Modified skills, with UnifiedDiff populated
}
```

- Validation: `SyncEngine` constructor validates source and targets are non-nil. `Diff(targetName)` returns error if target not found.
- Versioning: N/A -- no persistence; results are ephemeral per invocation.
- Persistence: None -- engines are stateless; all state comes from providers.

---

## APIs (as relevant to your role)

### SyncEngine API

```go
// NewSyncEngine creates a sync engine with a source and target providers.
func NewSyncEngine(source provider.Provider, targets []provider.Provider) *SyncEngine

// Sync reads skills from source and writes to all targets.
// If skillFilter is non-empty, only matching skill names are synced.
// Returns aggregate results. Does not abort on individual skill errors.
func (e *SyncEngine) Sync(skillFilter []string) (*SyncResult, error)
```

**Sync behavior:**
1. Call `source.ListSkills()`. If this fails, return error (fatal -- no skills to sync).
2. If `skillFilter` is non-empty, filter the list to only matching names.
3. For each skill in the list, call `source.ReadSkill(skill.Name)` to get full content.
4. For each target, call `target.WriteSkill(skill)`.
5. On per-skill or per-target error, record in `SyncDetail` with `SyncError` status; continue.
6. Return `SyncResult` with aggregated counts.

**Error semantics:**
- Returns `error` only for fatal failures (source ListSkills fails).
- Per-skill/per-target errors are captured in `SyncResult.Details`.

### DiffEngine API

```go
// NewDiffEngine creates a diff engine with a source and target providers.
func NewDiffEngine(source provider.Provider, targets []provider.Provider) *DiffEngine

// Status compares source skills against all targets.
// Returns a DriftReport with per-skill status for each target.
func (e *DiffEngine) Status() (*DriftReport, error)

// Diff returns detailed unified diffs for a specific target.
// Returns error if targetName is not in the engine's target list.
func (e *DiffEngine) Diff(targetName string) (*DetailedDiff, error)
```

**Status behavior:**
1. Call `source.ListSkills()` to get source skill names.
2. For each target, call `target.ListSkills()` to get target skill names.
3. For each source skill:
   - If not present in target: `MissingInTarget`.
   - If present: compare `Content` fields (normalize trailing whitespace). If equal: `InSync`. If different: `Modified`.
4. For each target skill not in source: `ExtraInTarget`.

**Diff behavior:**
1. Find target by name in engine's target list. Error if not found.
2. Run status comparison for that target only.
3. For each `Modified` skill, generate a unified diff (line-by-line comparison).
4. Return `DetailedDiff` with only the modified skills and their diffs.

**Unified diff approach:**
- Simple line-by-line diff using a minimal Go implementation (no external dependency).
- Format: standard unified diff with `---`/`+++` headers and `@@` hunks.
- Keep it simple: a basic LCS-based or Myers diff is acceptable, or even a naive "show all changed lines" for MVP. Prefer a small vendored/internal helper over shelling out to `diff`.

---

## Architecture / Component Boundaries (as relevant)

```
internal/sync/
├── engine.go       # SyncEngine: orchestrates source -> targets writes
├── diff.go         # DiffEngine: compares source vs targets, produces diffs
└── engine_test.go  # Tests for both engines
```

**Component responsibilities:**
- `SyncEngine` depends on `provider.Provider` interface only. It does not know about specific providers, config, or CLI.
- `DiffEngine` depends on `provider.Provider` interface only. Same isolation.
- Both engines are constructed with explicit provider instances (dependency injection via constructors). No global state, no registry lookups.
- The `internal/sync` package imports `internal/provider` for the interface and Skill type only.

**Boundary with other roles:**
- Provider Architect defines `Provider` interface and `Skill` struct -- I consume them.
- Claude Provider Dev implements `ClaudeProvider` -- I use it in tests via `t.TempDir()`.
- Config & CLI Foundation wires engines up in CLI commands -- I provide the API they call.

---

## Correctness Invariants (must be explicit)

1. **Idempotency:** Running `Sync` twice with unchanged source produces identical target state.
2. **Filter correctness:** When `skillFilter` is `["a", "b"]` and source has `["a", "b", "c"]`, only `a` and `b` are synced.
3. **Error isolation:** A `WriteSkill` failure for skill X on target Y does not prevent skill X from being written to target Z, nor skill W from being written to target Y.
4. **Status completeness:** `Status()` reports on every source skill AND every extra target skill. No skill is silently omitted.
5. **Diff consistency:** `Diff(target)` returns empty diffs for `InSync` skills and non-empty diffs for `Modified` skills. `MissingInTarget` and `ExtraInTarget` skills have no diff (nothing to compare).
6. **Content comparison normalization:** Trailing whitespace/newlines are trimmed before comparison so that provider write format differences don't cause false `Modified` reports.

---

## Tests

### Unit tests (`internal/sync/engine_test.go`)

**Mock provider approach:** Create a `mockProvider` struct implementing `provider.Provider` with in-memory skill storage (map). This avoids filesystem dependencies and makes tests fast and deterministic.

**SyncEngine tests (table-driven):**
| Test case | Setup | Assertion |
|-----------|-------|-----------|
| Sync all skills | Source: 3 skills, 1 target | Target receives all 3; TotalSynced=3, TotalErrored=0 |
| Sync with filter | Source: 3 skills, filter=["a","b"] | Target receives 2; TotalSynced=2 |
| Sync with empty source | Source: 0 skills | TotalSynced=0, no errors |
| Sync multi-target | Source: 2 skills, 2 targets | Each target gets 2; TotalSynced=4 |
| Sync with write error | Target WriteSkill returns error for 1 skill | TotalErrored=1, other skills still synced |
| Sync source list error | Source ListSkills returns error | Returns error (fatal) |

**DiffEngine tests (table-driven):**
| Test case | Setup | Assertion |
|-----------|-------|-----------|
| All in sync | Source and target have same 3 skills with same content | All 3 report InSync |
| Modified skill | Target has skill "a" with different content | Reports Modified with non-empty UnifiedDiff |
| Missing in target | Source has "a","b","c"; target has "a","b" | "c" reports MissingInTarget |
| Extra in target | Source has "a"; target has "a","b" | "b" reports ExtraInTarget |
| Mixed statuses | Combination of all statuses | Each skill gets correct status |
| Diff for specific target | 2 targets, 1 modified in each | Diff("target1") returns only target1's diffs |
| Diff for unknown target | Diff("nonexistent") | Returns error |
| Whitespace normalization | Same content but trailing newline differs | Reports InSync, not Modified |

**Commands:**
```bash
go test ./internal/sync/... -v
go test ./internal/sync/... -race
go test ./internal/sync/... -count=1  # no cache
```

---

## Benchmarks + "Success"

N/A -- The sync engine operates on small collections of text files (typical user has <50 skills). Benchmarking is not meaningful at this scale. Success is defined by:
- All tests pass with `-race` flag.
- `SyncResult` and `DriftReport` structs are correct and complete for all tested scenarios.
- The API is clean enough that the CLI Commands Dev can wire it up without confusion.

---

## Engineering Decisions & Tradeoffs (REQUIRED)

### Decision 1: Mock providers in tests vs. real ClaudeProvider with t.TempDir()

- **Decision:** Use mock providers (in-memory map-based) as the primary test strategy.
- **Alternatives considered:** Using `ClaudeProvider` with `t.TempDir()` for realistic filesystem-based tests.
- **Why:** Mock providers make tests faster, more deterministic, and independent of any specific provider implementation. The sync engine should be tested against the Provider interface contract, not against Claude-specific behavior. This also means sync engine tests don't break if the Claude provider changes its parsing logic.
- **Tradeoff acknowledged:** We lose integration-level confidence that the engine works with real providers. Mitigation: the QE phase will add end-to-end tests with real providers.

### Decision 2: Simple line-by-line diff vs. shelling out to `diff` command

- **Decision:** Implement a simple internal line-by-line diff algorithm (basic LCS/Myers) within `diff.go`.
- **Alternatives considered:** (a) Shelling out to `diff -u` via `os/exec`. (b) Using a third-party Go diff library.
- **Why:** Shelling out to `diff` introduces a platform dependency (may not exist on Windows) and subprocess overhead. Third-party libraries add dependency weight. A simple diff is ~50-80 lines of Go and sufficient for comparing markdown skill files.
- **Tradeoff acknowledged:** The internal diff won't handle all edge cases as well as GNU diff (e.g., large files, binary content). This is acceptable since skills are small text files.

### Decision 3: Separate SyncEngine and DiffEngine vs. single unified engine

- **Decision:** Two separate structs (`SyncEngine` and `DiffEngine`) with the same constructor pattern.
- **Alternatives considered:** A single `Engine` struct with `Sync()`, `Status()`, and `Diff()` methods.
- **Why:** Separation of concerns. Sync is a write operation; diff is a read-only operation. Having them separate makes it clear which operations mutate state. It also allows the CLI to construct only what it needs (e.g., `status` command doesn't need sync logic).
- **Tradeoff acknowledged:** Slight code duplication in constructor patterns and provider field storage. The duplication is minimal (2-3 lines) and the clarity benefit outweighs it.

---

## Risks & Mitigations (REQUIRED)

### Risk 1: Provider interface not finalized when I start coding
- **Risk:** The Provider Architect's plan may change the `Provider` interface or `Skill` struct after I've built against a draft.
- **Impact:** Compile errors and rework in engine.go and diff.go.
- **Mitigation:** Use the interface as specified in the kickoff.yaml. Mock providers in tests mean I'm only coupled to the interface, not to implementations. If the interface changes, the fix is mechanical (update mock + call sites).
- **Validation time:** < 5 minutes to update mocks and engine code if the interface changes.

### Risk 2: Content normalization is too aggressive or too loose
- **Risk:** Trimming trailing whitespace may hide real differences, or not trimming enough may cause false positives (e.g., provider A adds a trailing newline on write, provider B doesn't).
- **Impact:** Users see incorrect drift reports (either false "Modified" or missed real changes).
- **Mitigation:** Start with trimming trailing whitespace/newlines only. Document the normalization behavior. Add a `--strict` flag later if users want exact byte comparison.
- **Validation time:** < 10 minutes to write a test case covering whitespace edge cases and verify behavior.

### Risk 3: Unified diff output format is confusing or incomplete
- **Risk:** A hand-rolled diff algorithm may produce output that's hard to read or misses context lines.
- **Impact:** `skill-sync diff` output is unhelpful to users.
- **Mitigation:** Follow standard unified diff format (`---`/`+++`/`@@`). Include 3 lines of context around changes (standard convention). Test with known inputs and verify output matches expected diff format.
- **Validation time:** < 15 minutes to compare output against `diff -u` for a few test cases.

### Risk 4: Skill filter matching semantics are ambiguous
- **Risk:** Should filter match exact names, prefixes, or globs? If a filter name doesn't match any source skill, is that an error or silently ignored?
- **Impact:** User confusion when filter doesn't behave as expected.
- **Mitigation:** Use exact name matching (case-sensitive). If a filter name doesn't match any source skill, include it in the result as a "not found" detail (not a fatal error). Document this behavior.
- **Validation time:** < 5 minutes to write test cases for filter edge cases.

---

## Recommended API Surface

### Functions/Constructors
1. `NewSyncEngine(source provider.Provider, targets []provider.Provider) *SyncEngine`
2. `(*SyncEngine).Sync(skillFilter []string) (*SyncResult, error)`
3. `NewDiffEngine(source provider.Provider, targets []provider.Provider) *DiffEngine`
4. `(*DiffEngine).Status() (*DriftReport, error)`
5. `(*DiffEngine).Diff(targetName string) (*DetailedDiff, error)`

### Exported Types
- `SyncEngine`, `SyncResult`, `SyncDetail`, `SyncStatus`
- `DiffEngine`, `DriftReport`, `DetailedDiff`, `SkillDrift`, `DriftStatus`

## Folder Structure

```
internal/sync/
├── engine.go       # SyncEngine + SyncResult + SyncDetail + SyncStatus
├── diff.go         # DiffEngine + DriftReport + DetailedDiff + SkillDrift + DriftStatus + diff algorithm
└── engine_test.go  # mockProvider + all table-driven tests for both engines
```

Owned by: Sync Engine Dev.
No other roles write to `internal/sync/`.

## Step-by-step task plan (see Tighten section below)

## Benchmark plan
N/A -- see Benchmarks section above.

---

# Tighten the plan into 4-7 small tasks (STRICT)

### Task 1: Define sync engine types and Sync method
- **Outcome:** `SyncEngine` struct, constructor, `SyncResult`, `SyncDetail`, `SyncStatus` types, and `Sync()` method implemented.
- **Files to create/modify:** `internal/sync/engine.go`
- **Exact verification command(s):** `go build ./internal/sync/...` (compiles without error)
- **Suggested commit message:** `feat(sync): add SyncEngine with Sync method and result types`

### Task 2: Define diff engine types and Status/Diff methods
- **Outcome:** `DiffEngine` struct, constructor, `DriftReport`, `SkillDrift`, `DriftStatus`, `DetailedDiff` types, `Status()` and `Diff()` methods implemented. Includes internal unified diff helper.
- **Files to create/modify:** `internal/sync/diff.go`
- **Exact verification command(s):** `go build ./internal/sync/...` (compiles without error)
- **Suggested commit message:** `feat(sync): add DiffEngine with Status, Diff, and unified diff generation`

### Task 3: Implement mock provider and SyncEngine tests
- **Outcome:** `mockProvider` implementing `provider.Provider` with in-memory storage. All 6 SyncEngine test cases passing.
- **Files to create/modify:** `internal/sync/engine_test.go`
- **Exact verification command(s):** `go test ./internal/sync/... -v -run TestSync`
- **Suggested commit message:** `test(sync): add SyncEngine tests with mock provider`

### Task 4: Implement DiffEngine tests
- **Outcome:** All 8 DiffEngine test cases passing (Status and Diff methods).
- **Files to create/modify:** `internal/sync/engine_test.go`
- **Exact verification command(s):** `go test ./internal/sync/... -v -run TestDiff`
- **Suggested commit message:** `test(sync): add DiffEngine tests for status and diff`

### Task 5: Race condition and edge case hardening
- **Outcome:** All tests pass with `-race` flag. Edge cases verified: empty source, empty targets, filter with no matches, large skill content.
- **Files to create/modify:** `internal/sync/engine_test.go` (add edge case tests)
- **Exact verification command(s):** `go test ./internal/sync/... -v -race -count=1`
- **Suggested commit message:** `test(sync): add race and edge case tests for sync and diff engines`

---

# CLAUDE.md contributions (do NOT write the file; propose content)

## From Sync Engine Dev
**Coding style rules:**
- All exported types in `internal/sync/` have GoDoc comments.
- Error wrapping: `fmt.Errorf("sync: <context>: %w", err)` and `fmt.Errorf("diff: <context>: %w", err)`.
- No global state in `internal/sync/`. Engines are constructed with explicit dependencies.

**Dev commands:**
```bash
go test ./internal/sync/... -v           # run sync/diff tests
go test ./internal/sync/... -v -race     # with race detector
go test ./internal/sync/... -run TestSync  # sync engine only
go test ./internal/sync/... -run TestDiff  # diff engine only
```

**Before you commit checklist:**
- [ ] `go test ./internal/sync/... -race` passes
- [ ] `go vet ./internal/sync/...` clean
- [ ] No TODO comments without a linked issue
- [ ] SyncResult and DriftReport fields are fully populated (no zero-value surprises)

**Guardrails:**
- `SyncEngine.Sync()` must never abort on a single skill write failure; always collect errors.
- `DiffEngine.Status()` must report on ALL source and ALL target skills; no silent omissions.
- Content comparison MUST normalize trailing whitespace before comparing.

---

# EXPLAIN.md contributions (do NOT write the file; propose outline bullets)

- **Sync flow:** `SyncEngine` reads skills from source provider via `ListSkills()` + `ReadSkill()`, then calls `WriteSkill()` on each target. Errors are collected per-skill, never fatal.
- **Diff flow:** `DiffEngine.Status()` reads from source and each target, compares by skill name and content. Produces a `DriftReport` with four possible statuses per skill.
- **Unified diff:** `Diff(targetName)` generates standard unified diff output for modified skills, using an internal line-by-line comparison (no external `diff` binary required).
- **Key decision: mock-first testing** -- Engine tests use mock providers to stay decoupled from filesystem behavior. Integration tests with real providers are deferred to the QE phase.
- **Key decision: two engines, not one** -- `SyncEngine` (write) and `DiffEngine` (read-only) are separate structs for clarity and single-responsibility.
- **Limits of MVP:** No concurrent sync (skills processed sequentially). No progress callbacks. Diff algorithm is basic (sufficient for small markdown files, not optimized for large text).
- **How to run locally:** `go test ./internal/sync/... -v -race`
- **How to validate:** Check `SyncResult.TotalSynced` matches expected count. Check `DriftReport.Results` contains entries for all targets and all skills.

---

## READY FOR APPROVAL
