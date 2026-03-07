# Provider Architect Plan -- skill-sync Core Phase

## You are in PLAN MODE.

### Project
I want to build a **skill-sync CLI tool**.

**Goal:** build a **Go CLI** in which we **sync AI assistant skills from a primary provider to all others, with drift detection** -- starting with the core abstractions and contracts that every other component depends on.

### Role + Scope
- **Role:** Provider Architect
- **Scope:** I own the Provider interface, Skill model, SkillStatus enum, and the provider registry. I do NOT own any concrete provider implementations (Claude, Copilot, etc.), the sync/diff engines, config parsing, or CLI commands.
- **File I will write:** `/docs/core/plans/provider-architect.md`
- **No-touch zones:** do not edit any other files; do not write code.

---

## Functional Requirements
- **FR1:** Define a `Skill` struct that normalizes skills across all providers: Name, Description, Content, Arguments, SourcePath.
- **FR2:** Define a `Provider` interface with methods: `Name()`, `ListSkills()`, `ReadSkill(name)`, `WriteSkill(skill)`, `SkillDir()`.
- **FR3:** Define a `SkillStatus` enum type for drift reporting: `InSync`, `Modified`, `MissingInTarget`, `ExtraInTarget`.
- **FR4:** Implement a provider registry: `Register(provider)`, `Get(name)`, `List()`.
- **Tests required:** Unit tests for registry operations (register, get, list, get-unknown).
- **Metrics required:** N/A -- no runtime metrics for type definitions and registry.

## Non-Functional Requirements
- Language/runtime: Go 1.22+
- Local dev: `go test ./internal/provider/...`
- Observability: N/A -- pure library code, no HTTP endpoints
- Safety: Registry `Get()` returns a clear error for unknown provider names. No panics.
- Documentation: Exported types and functions get GoDoc comments.
- Performance: N/A -- registry is a simple map lookup, no benchmarks needed.

---

## Assumptions / System Model
- Deployment environment: Library code consumed by other packages in-process; no separate deployment.
- Failure modes: Only meaningful failure is requesting an unregistered provider name from the registry.
- Delivery guarantees: N/A -- no async or network operations.
- Multi-tenancy: N/A.

---

## Data Model

### Skill struct
```go
type Skill struct {
    Name        string   // derived from filename (no extension)
    Description string   // first-line description if present (e.g., "# Deploy to prod")
    Content     string   // full skill content (complete markdown body including description line)
    Arguments   []string // extracted argument placeholders (e.g., "$ARGUMENTS", "${PROJECT}")
    SourcePath  string   // absolute path to the source file on disk
}
```

**Validation rules:**
- `Name` must be non-empty (enforced by providers, not the struct itself -- we keep the model dumb).
- `Content` is the raw file content; it is the provider's job to parse/generate it.
- `Arguments` may be nil or empty (skill has no parameters).
- `SourcePath` may be empty for skills constructed programmatically (e.g., in tests).

**Versioning strategy:** None for MVP. Skills are compared by content equality. No version field.

**Persistence:** None. Skill is an in-memory value type. Providers handle file I/O.

### SkillStatus enum
```go
type SkillStatus int

const (
    InSync         SkillStatus = iota // source and target content match
    Modified                          // both exist but content differs
    MissingInTarget                   // exists in source, absent in target
    ExtraInTarget                     // absent in source, exists in target
)
```

A `String()` method on SkillStatus for human-readable output.

### Registry (internal state)
- `map[string]Provider` keyed by provider name (lowercase).
- Not persisted. Populated at init time by each provider package's `init()` function.

---

## APIs

### Package `internal/provider` -- exported API surface

#### Types

| Type | Kind | Description |
|------|------|-------------|
| `Skill` | struct | Normalized skill representation |
| `Provider` | interface | Contract for reading/writing skills |
| `SkillStatus` | int enum | Drift status between source and target |

#### Provider interface

```go
type Provider interface {
    // Name returns the provider's unique identifier (e.g., "claude", "copilot").
    Name() string

    // ListSkills returns all skills found in the provider's skill directory.
    ListSkills() ([]Skill, error)

    // ReadSkill reads a single skill by name. Returns nil, error if not found.
    ReadSkill(name string) (*Skill, error)

    // WriteSkill writes (or overwrites) a skill to the provider's skill directory.
    WriteSkill(skill Skill) error

    // SkillDir returns the absolute path to the provider's skill directory.
    SkillDir() string
}
```

**Error semantics:**
- `ListSkills`: returns `(nil, err)` if the directory doesn't exist or can't be read.
- `ReadSkill`: returns `(nil, err)` if the skill file doesn't exist. The error should wrap `os.ErrNotExist` so callers can check with `errors.Is`.
- `WriteSkill`: returns `err` if the directory can't be created or the file can't be written.

#### Registry functions

```go
// Register adds a provider to the global registry. Panics if name is already registered
// (this is a programming error, caught at init time).
func Register(p Provider)

// Get returns the provider registered under the given name.
// Returns an error if no provider is registered with that name.
func Get(name string) (Provider, error)

// List returns the names of all registered providers, sorted alphabetically.
func List() []string
```

#### SkillStatus methods

```go
// String returns the human-readable status label.
func (s SkillStatus) String() string
```

Maps: InSync -> "in-sync", Modified -> "modified", MissingInTarget -> "missing-in-target", ExtraInTarget -> "extra-in-target".

---

## Architecture / Component Boundaries

### What I own
- `internal/provider/provider.go` -- Skill struct, Provider interface, SkillStatus enum
- `internal/provider/registry.go` -- package-level registry (Register, Get, List)
- `internal/provider/registry_test.go` -- tests for the registry

### What I do NOT own
- Any concrete provider (claude.go, copilot.go, etc.) -- owned by Provider Devs
- Sync/diff engines (`internal/sync/`) -- owned by Sync Engine Dev
- Config parsing (`internal/config/`) -- owned by Config & CLI Foundation
- CLI commands (`cmd/`) -- owned by Config & CLI Foundation / CLI Commands Dev

### How components interact
1. Each provider package imports `internal/provider` and calls `provider.Register()` in its `init()`.
2. Config parsing resolves provider names via `provider.Get(name)`.
3. Sync engine accepts `Provider` interface values -- it never imports concrete providers.
4. Diff engine uses `SkillStatus` to report drift results.

### Config propagation
- N/A for this scope. The registry is populated at init time via Go's `init()` mechanism. No config watches.

### Concurrency model
- Registry is populated at init time (single-threaded by Go spec). No mutex needed for reads after init.
- If concurrent registration were ever needed, a `sync.RWMutex` could be added, but this is not needed for MVP.

### Backpressure
- N/A. No async operations.

---

## Correctness Invariants

1. **Registry uniqueness:** Registering two providers with the same name panics immediately (caught at init, not runtime).
2. **Get unknown returns error:** `Get("nonexistent")` always returns a non-nil error, never a nil Provider.
3. **List is sorted:** `List()` returns provider names in alphabetical order (deterministic iteration).
4. **SkillStatus String coverage:** Every SkillStatus constant has a defined String() output (no `<unknown>` for valid values).
5. **Skill is a value type:** Skill has no pointer fields that could cause aliasing bugs. Copying a Skill is safe.

---

## Tests

### Unit tests -- `internal/provider/registry_test.go`

| Test | What it verifies |
|------|-----------------|
| `TestRegister_And_Get` | Register a mock provider, Get by name returns it |
| `TestGet_Unknown` | Get with unregistered name returns error |
| `TestRegister_Duplicate_Panics` | Registering same name twice panics (use `recover`) |
| `TestList_Empty` | List returns empty slice when nothing registered |
| `TestList_Sorted` | Register providers in random order, List returns alphabetically sorted |
| `TestSkillStatus_String` | Each SkillStatus constant produces expected string |

All tests are table-driven where applicable.

### Integration tests
- N/A for this scope. The registry is a pure in-memory map. Integration testing happens when concrete providers register.

### Property/fuzz tests
- N/A. The surface area is too small to benefit from fuzzing.

### Failure injection tests
- N/A. No I/O or network calls.

### Commands
```
go test ./internal/provider/... -v -count=1
```

---

## Benchmarks + "Success"

N/A -- The registry is a `map[string]Provider` with O(1) lookup. Benchmarking a map read adds no value. Success is defined by all unit tests passing and the types being usable by downstream consumers (Claude Provider Dev, Sync Engine Dev).

---

## Engineering Decisions & Tradeoffs

### Decision 1: Package-level registry vs. passed-by-value registry
- **Decision:** Use a package-level (global) registry with `Register()`/`Get()`/`List()` functions.
- **Alternatives considered:** Pass a `*Registry` struct explicitly through constructors.
- **Why:** The provider set is fixed at compile time. Go's `init()` pattern is idiomatic for registering drivers/providers (see `database/sql`, `image`). A global registry avoids threading a registry parameter through every constructor.
- **Tradeoff acknowledged:** Harder to test in isolation (global state). Mitigated by resetting the registry in tests or using a separate test helper.

### Decision 2: Panic on duplicate registration vs. return error
- **Decision:** `Register()` panics if a provider name is already registered.
- **Alternatives considered:** Return an error from `Register()`.
- **Why:** Duplicate registration is always a programming error (two providers claiming the same name). Panicking at init time surfaces the bug immediately. This matches `database/sql.Register` behavior. Returning an error would require every `init()` to handle an error that should never happen.
- **Tradeoff acknowledged:** Panics are harsh, but they only happen at startup and indicate a real bug.

### Decision 3: SkillStatus as int enum vs. string constants
- **Decision:** Use `type SkillStatus int` with `iota` constants.
- **Alternatives considered:** `type SkillStatus string` with string constants like `"in-sync"`.
- **Why:** Int enums are more efficient for comparisons (switch statements in diff engine), prevent typo bugs, and provide exhaustiveness checking in IDE tooling. The `String()` method gives human-readable output when needed.
- **Tradeoff acknowledged:** Requires a `String()` method. Slightly more code than bare string constants.

---

## Risks & Mitigations

### Risk 1: Interface is too narrow for future providers
- **Risk:** A provider (e.g., Factory AI Droid) might need methods not on the current interface (e.g., `DeleteSkill`, `ValidateSkill`).
- **Impact:** Would require changing the interface, breaking all existing implementations.
- **Mitigation:** Keep the interface minimal (5 methods). If new capabilities are needed, use optional interfaces (type assertion) rather than expanding the base interface. Review Factory provider requirements before finalizing.
- **Validation time:** 5 minutes -- scan Factory docs for any unusual skill management patterns.

### Risk 2: Skill model doesn't capture provider-specific metadata
- **Risk:** Some providers use YAML frontmatter (Copilot) or other metadata that doesn't fit Name/Description/Content/Arguments.
- **Impact:** Information loss during sync; skills may not round-trip perfectly.
- **Mitigation:** `Content` stores the raw file body, so provider-specific parsing happens at the provider level. If metadata needs to survive translation, we can add a `Metadata map[string]string` field later without breaking the interface.
- **Validation time:** 10 minutes -- check Copilot and Gemini skill file formats.

### Risk 3: Global registry state leaks between tests
- **Risk:** Tests that register mock providers pollute the global registry, causing flaky tests.
- **Impact:** Tests pass individually but fail when run together.
- **Mitigation:** Add an unexported `resetRegistry()` function called in test setup (`TestMain` or per-test). This is internal-only and not part of the public API.
- **Validation time:** 2 minutes -- verify with `go test -count=10`.

### Risk 4: Content comparison semantics unclear
- **Risk:** "Modified" status depends on comparing Content fields, but whitespace normalization rules aren't defined at the model level.
- **Impact:** Skills that differ only in trailing newlines could be falsely reported as modified.
- **Mitigation:** This is the diff engine's responsibility, not the Skill model's. Document that `Content` is raw/verbatim and the diff engine should normalize before comparing.
- **Validation time:** 3 minutes -- write a test case with trailing whitespace differences.

---

## Recommended API Surface

### Functions/Methods (6 total)

1. `Register(p Provider)` -- register a provider by name (panics on duplicate)
2. `Get(name string) (Provider, error)` -- look up provider by name
3. `List() []string` -- list all registered provider names (sorted)
4. `(SkillStatus).String() string` -- human-readable status label

### Types (3 total)

1. `Skill` struct -- 5 fields (Name, Description, Content, Arguments, SourcePath)
2. `Provider` interface -- 5 methods (Name, ListSkills, ReadSkill, WriteSkill, SkillDir)
3. `SkillStatus` int enum -- 4 constants (InSync, Modified, MissingInTarget, ExtraInTarget)

---

## Folder Structure

```
internal/provider/
    provider.go       # Skill struct, Provider interface, SkillStatus enum + String()
    registry.go       # Register(), Get(), List(), internal map
    registry_test.go  # unit tests for registry + SkillStatus
```

I own all three files. Concrete providers (claude.go, copilot.go, etc.) are added by other roles in the same package.

---

## Step-by-step task plan (small commits)

See "Tighten the plan" section below.

---

## Benchmark plan

N/A -- No performance-critical code in scope. Success = all tests pass + types compile and are usable by downstream consumers.

---

# Tighten the plan into 4-7 small tasks

### Task 1: Define Skill struct and Provider interface
- **Outcome:** `provider.go` exists with exported `Skill` struct and `Provider` interface, both with GoDoc comments.
- **Files to create:** `internal/provider/provider.go`
- **Verification:**
  ```
  go build ./internal/provider/...
  ```
- **Commit message:** `feat(provider): define Skill struct and Provider interface`

### Task 2: Define SkillStatus enum with String method
- **Outcome:** `SkillStatus` type, 4 constants, and `String()` method added to `provider.go`.
- **Files to modify:** `internal/provider/provider.go`
- **Verification:**
  ```
  go build ./internal/provider/...
  ```
- **Commit message:** `feat(provider): add SkillStatus enum with String method`

### Task 3: Implement provider registry
- **Outcome:** `registry.go` with `Register()`, `Get()`, `List()`, and unexported `resetRegistry()` for tests.
- **Files to create:** `internal/provider/registry.go`
- **Verification:**
  ```
  go build ./internal/provider/...
  ```
- **Commit message:** `feat(provider): implement provider registry`

### Task 4: Write registry and SkillStatus unit tests
- **Outcome:** `registry_test.go` with 6 table-driven tests covering register, get, duplicate panic, list empty, list sorted, and SkillStatus strings.
- **Files to create:** `internal/provider/registry_test.go`
- **Verification:**
  ```
  go test ./internal/provider/... -v -count=1
  ```
- **Commit message:** `test(provider): add registry and SkillStatus unit tests`

### Task 5: Verify integration with downstream stubs
- **Outcome:** Manually verify that a stub provider (in a test) can implement the interface and register itself. Confirm no import cycles.
- **Files to modify:** `internal/provider/registry_test.go` (add a `mockProvider` that implements `Provider`)
- **Verification:**
  ```
  go test ./internal/provider/... -v -count=10
  go vet ./internal/provider/...
  ```
- **Commit message:** `test(provider): add mock provider integration test`

---

# CLAUDE.md contributions (do NOT write the file; propose content)

## From Provider Architect

### Coding style rules
- All exported types in `internal/provider/` must have GoDoc comments.
- Error messages use lowercase with `fmt.Errorf("provider: context: %w", err)` wrapping.
- No global mutable state except the provider registry (which is write-once at init time).
- SkillStatus constants use `iota`; always add new statuses at the end to preserve ordering.

### Dev commands
```bash
# Build provider package
go build ./internal/provider/...

# Run provider tests
go test ./internal/provider/... -v -count=1

# Vet provider package
go vet ./internal/provider/...
```

### Before you commit checklist
- [ ] `go build ./internal/provider/...` passes
- [ ] `go test ./internal/provider/... -count=1` passes
- [ ] `go vet ./internal/provider/...` reports no issues
- [ ] No new exported types without GoDoc comments

### Guardrails
- Do NOT add methods to the `Provider` interface without discussing with the team. Interface changes break all providers.
- Do NOT store pointers in the `Skill` struct. It is a value type by design.
- Do NOT add a mutex to the registry unless concurrent registration is actually needed (it isn't for MVP).

---

# EXPLAIN.md contributions (do NOT write the file; propose outline bullets)

### Flow / Architecture
- The `Provider` interface is the central contract: every AI assistant (Claude, Copilot, Gemini, Factory) implements it.
- The provider registry uses Go's `init()` pattern (like `database/sql`) -- importing a provider package auto-registers it.
- `Skill` is a simple value type that normalizes skills across all providers. Raw content is preserved; format translation is the provider's job.

### Key Engineering Decisions + Tradeoffs
- Global registry (simple, idiomatic) over dependency-injected registry (more testable but more plumbing). Reset function provided for tests.
- Panic on duplicate registration (fail-fast at startup) over error return (silent failures).
- Int enum for SkillStatus (type-safe, efficient) over string constants (simpler but error-prone).
- Minimal interface (5 methods) -- extensibility via optional interfaces rather than a fat base interface.

### Limits of MVP + Next Steps
- No `DeleteSkill` method -- sync is additive only. Could add via optional `Deleter` interface later.
- No `Metadata` field on Skill -- if providers need to carry extra data, this can be added without breaking changes.
- Registry has no thread safety for writes -- fine because registration only happens in `init()`.

### How to Run Locally + How to Validate
- `go test ./internal/provider/... -v` runs all provider-layer tests.
- `go vet ./internal/provider/...` checks for common mistakes.
- A mock provider in the test file demonstrates how to implement the interface.

---

## READY FOR APPROVAL
