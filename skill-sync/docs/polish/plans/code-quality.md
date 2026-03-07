# Code Quality Dev -- Plan: Go Doc Comments for skill-sync

---

## You are in PLAN MODE.

### Project
I want to polish the **skill-sync** CLI tool for open-source readiness.

**Goal:** Add Go doc comments to all exported types, functions, and package declarations across `internal/` packages so that `go doc` output is clear, accurate, and follows Go conventions.

### Role + Scope
- **Role:** Code Quality Dev
- **Scope:** Doc comments on all exported symbols and package declarations in `internal/config`, `internal/provider`, and `internal/sync`. I do NOT own README, CLI help text, architecture diagrams, or any code behavior changes.
- **File I will write:** `/docs/polish/plans/code-quality.md`
- **No-touch zones:** Do not edit any files outside `internal/`; do not write code yet; do not change any behavior.

---

## Functional Requirements
- FR1: Every exported type, function, method, constant, and variable in `internal/` packages has a Go doc comment that follows `go doc` conventions (starts with the name of the thing being documented).
- FR2: Every package has a package-level doc comment in its primary file.
- FR3: Comments document non-obvious behavior, format assumptions, and error conditions -- not the obvious.
- FR4: `go doc ./internal/...` produces useful, complete output for all packages.
- Tests required: `go vet ./...` must pass (it warns on some doc comment issues). No new unit tests needed -- this is documentation only.
- Metrics required: N/A -- documentation change only.

## Non-Functional Requirements
- Language/runtime: Go (doc comments only, no code changes)
- Local dev: N/A -- no infrastructure changes
- Observability: N/A
- Safety: No behavior changes; doc-only edits carry zero regression risk
- Documentation: This IS the documentation task
- Performance: N/A

---

## Assumptions / System Model
- Deployment environment: N/A -- doc comments only
- Failure modes: The only failure is a doc comment that is wrong or misleading. Mitigated by reading each file before writing docs.
- Delivery guarantees: N/A
- Multi-tenancy: N/A

---

## Data Model

N/A -- not in scope for this role. The data model is already defined in `internal/provider/provider.go` (Skill struct, SkillStatus enum) and `internal/config/config.go` (Config struct). Doc comments will describe these, not change them.

---

## APIs

N/A -- not in scope for this role. The API surface is defined by the exported types and functions in `internal/`. Doc comments will describe them, not change them.

---

## Architecture / Component Boundaries

N/A -- not in scope for this role. No architectural changes. Doc comments span three packages:

1. **`internal/config`** -- Config loading and validation. Primary file: `config.go`.
2. **`internal/provider`** -- Provider interface, Skill model, registry, and 4 provider implementations. Primary file: `provider.go`.
3. **`internal/sync`** -- Sync engine and diff/drift engine. Primary file: `engine.go`.

---

## Correctness Invariants
- Every exported symbol has a doc comment after this task.
- No doc comment contradicts the actual behavior of the code.
- `go vet ./...` continues to pass.
- `go test ./...` continues to pass (doc comments cannot break tests, but we verify anyway).

---

## Current State Assessment

After reading all source files, here is the current doc comment coverage:

### Already well-documented (minor gaps only):
- **`internal/provider/provider.go`** -- Package doc present. All exported types (`Skill`, `Provider`, `SkillStatus`), all fields, all constants, and `String()` method are documented. **No changes needed.**
- **`internal/provider/registry.go`** -- `Register`, `Get`, `List` all have accurate doc comments. Missing: no package doc needed here (package doc is in `provider.go`, which is correct Go convention). **No changes needed.**
- **`internal/provider/claude.go`** -- `Option`, `WithBaseDir`, `ClaudeProvider`, `NewClaudeProvider`, all methods documented. **No changes needed.**
- **`internal/provider/copilot.go`** -- Same pattern as claude.go, all exports documented. **No changes needed.**
- **`internal/provider/gemini.go`** -- All exports documented including helper types. **No changes needed.**
- **`internal/provider/factory.go`** -- All exports documented. **No changes needed.**
- **`internal/sync/engine.go`** -- Package doc present (`// Package sync provides engines for syncing and diffing skills across providers.`). All exported types and functions documented. **No changes needed.**

### Gaps found (need doc comment additions):
- **`internal/config/config.go`** -- MISSING package doc comment. Exported types (`Config`) and functions (`Load`, `Validate`) have doc comments but `Config` struct fields (`Source`, `Targets`, `Skills`) lack field-level documentation.
- **`internal/sync/diff.go`** -- `SkillDrift`, `DriftReport`, `DetailedDiff` structs have type-level doc comments but their fields lack documentation. `DiffEngine`, `NewDiffEngine`, `Status`, `Diff` methods are documented. Unexported helpers `normalizeContent`, `splitLines`, `computeLCS`, `unifiedDiff` are documented (good practice).

### Summary of actual work needed:
The codebase is **already well-documented**. The remaining gaps are:
1. Package doc comment for `internal/config`
2. Field-level comments on `Config` struct (3 fields)
3. Field-level comments on `SyncDetail` struct (4 fields)
4. Field-level comments on `SyncResult` struct (3 fields)
5. Field-level comments on `SkillDrift` struct (3 fields)
6. Field-level comments on `DriftReport` struct (1 field)
7. Field-level comments on `DetailedDiff` struct (2 fields)

---

## Tests
- **Verification commands:**
  - `go vet ./...` -- must pass (catches some doc comment issues)
  - `go test ./...` -- must still pass (regression check)
  - `go doc ./internal/config` -- verify package doc appears
  - `go doc ./internal/config.Config` -- verify field docs appear
  - `go doc ./internal/sync.SyncDetail` -- verify field docs appear
  - `go doc ./internal/sync.SkillDrift` -- verify field docs appear

No new unit tests, integration tests, or fuzz tests needed. This is a documentation-only change.

---

## Benchmarks + "Success"
N/A -- documentation change. No performance implications. Success = all `go doc` output is complete and accurate.

---

## Engineering Decisions & Tradeoffs (REQUIRED)

### Decision 1: Add field-level comments only where they add value
- **Decision:** Add doc comments to struct fields that are not self-explanatory (e.g., `Skills` in `Config` which has a non-obvious `flow` YAML tag, `UnifiedDiff` in `SkillDrift` which is conditionally populated).
- **Alternatives considered:** (A) Add a comment to every single field, even obvious ones like `Error error`. (B) Skip field comments entirely and rely on type-level docs.
- **Why:** Go doc convention says "don't state the obvious." `Error error` needs no comment. `UnifiedDiff string` does because it is only populated for Modified status.
- **Tradeoff acknowledged:** Some fields will remain undocumented. A contributor might need to read the code to understand `TotalSynced int`. This is acceptable because the name is self-descriptive.

### Decision 2: Package doc in primary file only
- **Decision:** Place the package doc comment in the "primary" file of each package (`provider.go`, `engine.go`, `config.go`) rather than in a separate `doc.go` file.
- **Alternatives considered:** (A) Create `doc.go` files for each package. (B) Put package docs in every file (Go only uses one).
- **Why:** The codebase already uses this pattern (`provider.go` and `engine.go` have package docs). Adding a `doc.go` would be inconsistent with the existing style. For small packages with 1-3 files, inline package docs are standard.
- **Tradeoff acknowledged:** If a package grows large, a `doc.go` might be clearer. For the current codebase size, inline is fine.

### Decision 3: Do not add doc comments to cmd/ package
- **Decision:** Scope is strictly `internal/` packages. The `cmd/` package has exported vars (`Cfg`, `InlineSource`, `InlineTargets`) and a function (`Execute`) that lack doc comments, but these are cobra wiring, not library API.
- **Alternatives considered:** Document `cmd/` exports too.
- **Why:** The task scope explicitly says "all internal/ packages." The `cmd/` package is not meant to be imported by other Go code -- it is a CLI entry point. Cobra-generated code has a different documentation model (help text, not Go docs).
- **Tradeoff acknowledged:** `go doc ./cmd` will have sparse output. This is acceptable for a CLI package.

---

## Risks & Mitigations (REQUIRED)

### Risk 1: Doc comments that contradict actual behavior
- **Risk:** Writing a doc comment that describes behavior incorrectly (e.g., saying a function returns nil on error when it actually returns a wrapped error).
- **Impact:** Misleading documentation is worse than no documentation.
- **Mitigation:** Read each function's implementation before writing its doc comment. Cross-reference with test files for expected behavior.
- **Validation time:** ~5 minutes per file (already done during plan research).

### Risk 2: Merge conflicts with concurrent README/CLI help changes
- **Risk:** Other team members (README Author, CLI UX Dev) are editing files in the same repo simultaneously.
- **Impact:** Git merge conflicts could block the PR.
- **Mitigation:** This role only touches `internal/` Go files. README Author touches `README.md`. CLI UX Dev touches `cmd/` files. Zero file overlap = zero conflict risk.
- **Validation time:** 0 minutes (no overlap by design).

### Risk 3: go vet or go test regression
- **Risk:** A malformed doc comment could cause `go vet` to fail.
- **Impact:** Build pipeline breaks.
- **Mitigation:** Run `go vet ./...` and `go test ./...` after every file edit. Doc comments in Go are plain comments -- they cannot cause compilation errors. `go vet` checks for mismatched function names in doc comments.
- **Validation time:** ~2 minutes to run both commands.

---

## Recommended API Surface

N/A -- no new APIs. This task adds documentation to existing exports:

### `internal/config` (1 file)
| Symbol | Current Doc | Action |
|--------|------------|--------|
| `package config` | Missing | Add package doc |
| `Config` struct | Has type doc | Add field comments for `Source`, `Targets`, `Skills` |
| `Load` | Has doc | No change |
| `Validate` | Has doc | No change |

### `internal/provider` (6 files)
| Symbol | Current Doc | Action |
|--------|------------|--------|
| `package provider` | Has doc (in provider.go) | No change |
| All exports in provider.go | Fully documented | No change |
| All exports in registry.go | Fully documented | No change |
| All exports in claude.go | Fully documented | No change |
| All exports in copilot.go | Fully documented | No change |
| All exports in gemini.go | Fully documented | No change |
| All exports in factory.go | Fully documented | No change |

### `internal/sync` (2 files)
| Symbol | Current Doc | Action |
|--------|------------|--------|
| `package sync` | Has doc (in engine.go) | No change |
| `SyncStatus` | Has doc | No change |
| `SyncDetail` | Has type doc | Add field comments |
| `SyncResult` | Has type doc | Add field comments |
| `SyncEngine` | Has doc | No change |
| `NewSyncEngine` | Has doc | No change |
| `Sync` | Has doc | No change |
| `SkillDrift` | Has type doc | Add field comments |
| `DriftReport` | Has type doc | Add field comments |
| `DetailedDiff` | Has type doc | Add field comments |
| `DiffEngine` | Has doc | No change |
| `NewDiffEngine` | Has doc | No change |
| `Status` | Has doc | No change |
| `Diff` | Has doc | No change |

---

## Folder Structure

No new files or packages. Edits to existing files only:

```
internal/
├── config/
│   └── config.go          # ADD: package doc, field comments on Config
├── provider/
│   ├── provider.go        # NO CHANGE (already complete)
│   ├── registry.go        # NO CHANGE
│   ├── claude.go          # NO CHANGE
│   ├── copilot.go         # NO CHANGE
│   ├── gemini.go          # NO CHANGE
│   └── factory.go         # NO CHANGE
└── sync/
    ├── engine.go          # ADD: field comments on SyncDetail, SyncResult
    └── diff.go            # ADD: field comments on SkillDrift, DriftReport, DetailedDiff
```

---

## Tighten the plan into 4-7 small tasks (STRICT)

### Task 1: Add package doc and field comments to `internal/config/config.go`
- **Outcome:** Package has a doc comment. `Config` struct fields have doc comments describing their YAML mapping and purpose.
- **Files to create/modify:** `internal/config/config.go`
- **Exact verification commands:**
  - `cd /Users/michaelhabib/dev/teams-sbx/skill-sync && go vet ./internal/config/...`
  - `cd /Users/michaelhabib/dev/teams-sbx/skill-sync && go doc ./internal/config`
  - `cd /Users/michaelhabib/dev/teams-sbx/skill-sync && go doc ./internal/config.Config`
- **Suggested commit message:** `docs: add package doc and field comments to internal/config`

### Task 2: Add field comments to `SyncDetail` and `SyncResult` in `internal/sync/engine.go`
- **Outcome:** `SyncDetail` fields (`SkillName`, `Target`, `Status`, `Error`) and `SyncResult` fields (`TotalSynced`, `TotalErrored`, `Details`) have doc comments where they add value beyond the field name.
- **Files to create/modify:** `internal/sync/engine.go`
- **Exact verification commands:**
  - `cd /Users/michaelhabib/dev/teams-sbx/skill-sync && go vet ./internal/sync/...`
  - `cd /Users/michaelhabib/dev/teams-sbx/skill-sync && go doc ./internal/sync.SyncDetail`
  - `cd /Users/michaelhabib/dev/teams-sbx/skill-sync && go doc ./internal/sync.SyncResult`
- **Suggested commit message:** `docs: add field comments to SyncDetail and SyncResult`

### Task 3: Add field comments to `SkillDrift`, `DriftReport`, and `DetailedDiff` in `internal/sync/diff.go`
- **Outcome:** `SkillDrift.UnifiedDiff` documents that it is only populated when Status is Modified. `DriftReport.Results` documents the map key semantics. `DetailedDiff` fields are documented.
- **Files to create/modify:** `internal/sync/diff.go`
- **Exact verification commands:**
  - `cd /Users/michaelhabib/dev/teams-sbx/skill-sync && go vet ./internal/sync/...`
  - `cd /Users/michaelhabib/dev/teams-sbx/skill-sync && go doc ./internal/sync.SkillDrift`
  - `cd /Users/michaelhabib/dev/teams-sbx/skill-sync && go doc ./internal/sync.DriftReport`
  - `cd /Users/michaelhabib/dev/teams-sbx/skill-sync && go doc ./internal/sync.DetailedDiff`
- **Suggested commit message:** `docs: add field comments to SkillDrift, DriftReport, DetailedDiff`

### Task 4: Final verification across all packages
- **Outcome:** All `go doc` commands produce complete output. `go vet` and `go test` pass.
- **Files to create/modify:** None (verification only)
- **Exact verification commands:**
  - `cd /Users/michaelhabib/dev/teams-sbx/skill-sync && go vet ./...`
  - `cd /Users/michaelhabib/dev/teams-sbx/skill-sync && go test ./...`
  - `cd /Users/michaelhabib/dev/teams-sbx/skill-sync && go doc ./internal/config`
  - `cd /Users/michaelhabib/dev/teams-sbx/skill-sync && go doc ./internal/provider`
  - `cd /Users/michaelhabib/dev/teams-sbx/skill-sync && go doc ./internal/sync`
- **Suggested commit message:** N/A (verification step, no commit)

---

## CLAUDE.md contributions (do NOT write the file; propose content)

## From Code Quality Dev
- **Coding style rules:**
  - All exported types, functions, methods, and constants MUST have Go doc comments
  - Doc comments start with the name of the thing being documented (Go convention)
  - Do not document the obvious (`Error error` needs no comment; `UnifiedDiff string` does because it has conditional population semantics)
  - Package doc comments go in the primary file of the package (e.g., `provider.go` for `package provider`), not in a separate `doc.go`
  - Struct field comments are required when the field name alone does not convey usage constraints or format expectations

- **Dev commands:**
  - `go doc ./internal/config` -- check config package documentation
  - `go doc ./internal/provider` -- check provider package documentation
  - `go doc ./internal/sync` -- check sync package documentation
  - `go vet ./...` -- includes some doc comment validation

- **Before you commit checklist:**
  - [ ] `go vet ./...` passes
  - [ ] `go test ./...` passes
  - [ ] Any new exported symbol has a doc comment
  - [ ] Doc comments do not contradict actual behavior

- **Guardrails:**
  - No exported symbol should be added without a doc comment
  - If you change a function's behavior, update its doc comment in the same commit

---

## EXPLAIN.md contributions (do NOT write the file; propose outline bullets)

- **Doc comment conventions:**
  - All `internal/` packages follow standard Go doc comment conventions
  - Package docs are in the primary file of each package
  - Field-level comments are used where the field name alone is insufficient
  - Run `go doc ./internal/<package>` to see formatted documentation

- **Key decisions:**
  - Package docs inline (not `doc.go`) for consistency with existing codebase style
  - Field comments are selective, not exhaustive -- only where they add clarity
  - `cmd/` package is excluded from doc comment requirements (it is CLI wiring, not a library API)

- **Validation:**
  - `go vet ./...` catches doc comment issues
  - `go doc` commands verify output formatting

---

## READY FOR APPROVAL
