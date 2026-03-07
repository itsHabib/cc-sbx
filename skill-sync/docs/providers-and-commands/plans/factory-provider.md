# Factory Provider Plan

## You are in PLAN MODE.

### Project
I want to build a **skill-sync CLI tool**.

**Goal:** build a **Factory AI Droid provider** that reads and writes skills from Factory's project-level `.factory/skills/<name>/SKILL.md` directory structure, parsing optional YAML frontmatter, and normalizing into the shared `Skill` model.

### Role + Scope
- **Role:** Factory Provider Dev
- **Scope:** Implement `FactoryProvider` struct + all `Provider` interface methods + tests. I own `internal/provider/factory.go` and `internal/provider/factory_test.go`. I do NOT own the CLI commands, sync engine, diff engine, config, or other providers.
- **File I will write:** `/docs/providers-and-commands/plans/factory-provider.md`
- **No-touch zones:** Do not edit any other files; do not write code.

---

## Functional Requirements
- FR1: `FactoryProvider` implements the `Provider` interface (`Name()`, `ListSkills()`, `ReadSkill()`, `WriteSkill()`, `SkillDir()`).
- FR2: Parse YAML frontmatter (`---` delimited) from `SKILL.md` files, extracting `name`, `description`, and optionally `model` fields.
- FR3: Handle files with NO frontmatter gracefully -- treat the entire file as markdown body, derive description from first `# ` line (same fallback as Claude provider).
- FR4: `WriteSkill()` always writes frontmatter with `name` and `description` fields, followed by the markdown body. The `model` field is omitted.
- FR5: Directory-based structure: each skill lives in `<baseDir>/<skill.Name>/SKILL.md`.
- Tests required: unit tests covering frontmatter parsing, no-frontmatter fallback, round-trip, empty file, and edge cases.
- Metrics required: N/A -- CLI tool, no Prometheus.

## Non-Functional Requirements
- Language/runtime: Go 1.22+
- Local dev: `go test ./internal/provider/...`
- Observability: N/A -- single binary CLI
- Safety: All errors wrapped with `fmt.Errorf("factory: context: %w", err)`. Non-existent directories produce clear errors.
- Documentation: CLAUDE.md + EXPLAIN.md contributions (proposed below).
- Performance: N/A -- file I/O on local disk, no hot path.

---

## Assumptions / System Model
- Deployment environment: Local CLI binary, no containers.
- Failure modes: Missing baseDir, missing `SKILL.md` in subdirectory, malformed frontmatter YAML, empty files, permission errors.
- Delivery guarantees: N/A -- not a networked system.
- Multi-tenancy: N/A.

---

## Data Model

### On-disk format: `.factory/skills/<name>/SKILL.md`

```
<baseDir>/
  worker/
    SKILL.md
  reviewer/
    SKILL.md
```

Each `SKILL.md` has optional YAML frontmatter:

```yaml
---
name: worker
description: >-
  General-purpose worker droid for delegating tasks.
model: inherit
---
# Worker Droid

Markdown prompt body here.
```

### Frontmatter struct (internal, not exported)

```go
type factoryFrontmatter struct {
    Name        string `yaml:"name"`
    Description string `yaml:"description"`
    Model       string `yaml:"model,omitempty"`
}
```

### Mapping to Skill model

| Factory field | Skill field | Fallback |
|---|---|---|
| Frontmatter `name` | `Skill.Name` | Directory name |
| Frontmatter `description` | `Skill.Description` | First `# ` line, or empty |
| Markdown body after `---` | `Skill.Content` | Entire file content |
| File path | `Skill.SourcePath` | Always set on read |
| N/A | `Skill.Arguments` | Always `nil` (Factory has no argument syntax) |

### Validation rules
- Frontmatter is optional. If present, it must be valid YAML between `---` delimiters.
- `name` in frontmatter overrides directory name for `Skill.Name`.
- On write, `name` and `description` are always written to frontmatter. `model` is omitted.

### Versioning strategy
- N/A -- files are overwritten on sync. No versioning needed for MVP.

---

## APIs

### Provider interface implementation

```
Name() string                    -> "factory"
ListSkills() ([]Skill, error)    -> scan baseDir for subdirs, read SKILL.md in each
ReadSkill(name string) (*Skill, error) -> read <baseDir>/<name>/SKILL.md
WriteSkill(skill Skill) error    -> write <baseDir>/<skill.Name>/SKILL.md
SkillDir() string                -> return baseDir
```

### Constructor

```
NewFactoryProvider(opts ...FactoryOption) *FactoryProvider
```

Uses provider-specific `FactoryOption func(*FactoryProvider)` and `WithFactoryBaseDir(dir string) FactoryOption` — Claude's `Option` type operates on `*ClaudeProvider` and is not reusable (see Engineering Decisions below).

### Error semantics
- `ListSkills()`: returns error if baseDir does not exist. Returns empty slice if baseDir exists but has no subdirectories with `SKILL.md`.
- `ReadSkill(name)`: returns `os.ErrNotExist`-wrapping error if `<baseDir>/<name>/SKILL.md` does not exist.
- `WriteSkill(skill)`: creates `<baseDir>/<skill.Name>/` directory via `os.MkdirAll` if needed.

---

## Architecture / Component Boundaries

### Components I touch
1. **`internal/provider/factory.go`** -- `FactoryProvider` struct, constructor, all interface methods, frontmatter parsing/serializing helpers.
2. **`internal/provider/factory_test.go`** -- table-driven tests for all behaviors.

### Components I do NOT touch
- `provider.go` (interface + Skill model) -- already defined, no changes needed.
- `registry.go` -- I call `Register()` from `init()`, no modifications.
- `claude.go` -- reference only, no changes.
- `sync/`, `config/`, `cmd/` -- out of scope.

### How config changes propagate
- N/A -- provider is configured at construction time via functional options.

### Concurrency model
- No goroutines. All I/O is sequential file reads/writes.

### Backpressure strategy
- N/A.

---

## Correctness Invariants

1. **Frontmatter round-trip:** A skill written with `WriteSkill` and read back with `ReadSkill` must produce identical `Name`, `Description`, and `Content` fields.
2. **No-frontmatter safety:** A plain markdown file (no `---` delimiters) must parse without error, with the entire content as `Skill.Content`.
3. **Name derivation:** If frontmatter has a `name` field, it takes precedence over the directory name. If no frontmatter, the directory name is used.
4. **Description fallback:** If no frontmatter and no `# ` line, `Skill.Description` is empty string (not an error).
5. **Directory isolation:** Each skill is in its own subdirectory. `ListSkills` only considers directories containing `SKILL.md`.
6. **Registry uniqueness:** `init()` registers exactly one provider named `"factory"`. Duplicate registration panics (enforced by registry).

---

## Tests

### Unit tests (`internal/provider/factory_test.go`)

| Test | What it verifies |
|---|---|
| `TestFactoryName` | `Name()` returns `"factory"` |
| `TestFactorySkillDir` | `SkillDir()` returns configured baseDir |
| `TestFactoryReadSkill_WithFrontmatter` | Parses name, description, body from frontmatter file |
| `TestFactoryReadSkill_WithFrontmatterAndModel` | Parses frontmatter including optional `model` field (model is ignored in Skill but parsing must not fail) |
| `TestFactoryReadSkill_NoFrontmatter` | Plain markdown: name from dir, description from `# ` line |
| `TestFactoryReadSkill_NoFrontmatterNoDescription` | Plain markdown with no `# ` line: description is empty |
| `TestFactoryReadSkill_EmptyFile` | Empty `SKILL.md`: name from dir, empty description + content |
| `TestFactoryReadSkill_FrontmatterOnly` | Frontmatter with no body: content is empty |
| `TestFactoryReadSkill_NonExistent` | Returns error wrapping `os.ErrNotExist` |
| `TestFactoryListSkills_Multiple` | Lists multiple skills from subdirectories |
| `TestFactoryListSkills_Empty` | Empty baseDir returns empty slice, no error |
| `TestFactoryListSkills_NonExistentDir` | Returns error for missing baseDir |
| `TestFactoryListSkills_SkipsNonSkillDirs` | Subdirs without `SKILL.md` are silently skipped |
| `TestFactoryWriteSkill_Basic` | Writes `SKILL.md` with frontmatter + body |
| `TestFactoryWriteSkill_CreatesDir` | Creates subdirectory if needed |
| `TestFactoryWriteSkill_RoundTrip` | Write then read produces matching fields |
| `TestFactoryWriteSkill_RoundTrip_NoDescription` | Round-trip with empty description |
| `TestFactoryReadSkill_MalformedYAML` | Frontmatter delimiters present but invalid YAML → returns error (not silent fallback) |

### Test commands

```bash
go test ./internal/provider/ -v -run TestFactory
go test ./internal/provider/ -v
go test ./...
```

### Integration tests
- N/A -- provider is a pure file I/O abstraction. Unit tests with `t.TempDir()` are sufficient.

### Property/fuzz tests
- Optional future: fuzz frontmatter parsing with arbitrary YAML. Not in scope for this phase.

### Failure injection
- N/A -- no network I/O. File permission errors are implicitly tested by OS behavior.

---

## Benchmarks + "Success"

N/A -- File I/O on local disk with small markdown files. No performance-sensitive path. The provider will handle at most dozens of files. Benchmarking is not relevant.

---

## Engineering Decisions & Tradeoffs

### Decision 1: Provider-specific option types

- **Decision:** Define `FactoryOption func(*FactoryProvider)` rather than reusing Claude's `Option func(*ClaudeProvider)`.
- **Alternatives considered:** (a) A shared generic `Option` type using interfaces or type assertions. (b) Reusing Claude's `Option` type directly.
- **Why:** Claude's `Option` type is `func(*ClaudeProvider)` which is not compatible with `*FactoryProvider`. A shared generic option would require either generics (adding complexity) or interface-based options (losing type safety). Per-provider option types are simple, explicit, and match the existing pattern.
- **Tradeoff acknowledged:** Slight code duplication (each provider defines its own `WithBaseDir` variant). This is acceptable for 4 providers and avoids premature abstraction.

### Decision 2: Always write frontmatter on WriteSkill

- **Decision:** `WriteSkill()` always emits YAML frontmatter with `name` and `description`, even if the source skill had no frontmatter.
- **Alternatives considered:** (a) Write frontmatter only if description is non-empty. (b) Preserve original format (frontmatter vs. plain) via a metadata flag.
- **Why:** Consistency and round-trip safety. Every Factory skill written by skill-sync will have a predictable format. Reading back always finds frontmatter, avoiding ambiguity.
- **Tradeoff acknowledged:** Skills that were originally plain markdown (no frontmatter) will gain frontmatter after a sync. This is acceptable because skill-sync is the canonical writer for target providers.

### Decision 3: Frontmatter parsing with `---` splitting + `yaml.v3`

- **Decision:** Split file content on `---` delimiters manually, then parse the YAML block with `gopkg.in/yaml.v3`.
- **Alternatives considered:** (a) Use a dedicated frontmatter parsing library (e.g., `github.com/adrg/frontmatter`). (b) Regex-based extraction.
- **Why:** `yaml.v3` is already a dependency. Manual splitting on `---` is straightforward (split on first two `---` lines). No new dependency needed. Regex is fragile for multi-line YAML.
- **Tradeoff acknowledged:** Manual splitting requires careful handling of edge cases (e.g., `---` appearing in the body). Mitigation: only the FIRST pair of `---` lines is treated as frontmatter delimiters.

---

## Risks & Mitigations

### Risk 1: Ambiguous `---` in markdown body
- **Risk:** A markdown file could contain `---` (horizontal rule) in the body, which might be mistaken for frontmatter delimiters.
- **Impact:** Body content could be truncated or misinterpreted as YAML.
- **Mitigation:** Only treat `---` as frontmatter if the file STARTS with `---` on the very first line. The closing `---` is the next occurrence. Everything after the second `---` is body. This matches the standard frontmatter convention used by Jekyll, Hugo, etc.
- **Validation time:** < 5 minutes -- write a test with `---` in the body.

### Risk 2: Frontmatter with unexpected fields
- **Risk:** A Factory droid file has YAML fields we don't expect (e.g., `tags`, `version`, custom fields).
- **Impact:** `yaml.Unmarshal` into our struct silently ignores unknown fields, so no parse error -- but data is lost on round-trip.
- **Mitigation:** This is acceptable for MVP. We only extract `name`, `description`, and `model`. Unknown fields are not preserved. Document this limitation.
- **Validation time:** < 5 minutes -- write a test with extra fields.

### Risk 3: Empty or malformed YAML in frontmatter
- **Risk:** The YAML between `---` delimiters is invalid or empty.
- **Impact:** `yaml.Unmarshal` returns an error, which propagates up.
- **Mitigation:** `parseFrontmatter` returns an error when frontmatter delimiters are found but YAML is malformed. The caller (`ReadSkill`) wraps and propagates this error rather than silently falling back, so users know their file is broken. Empty YAML between `---` is valid (produces zero-value struct) and is not an error. Test both cases explicitly.
- **Validation time:** < 5 minutes -- write a test with invalid YAML.

### Risk 4: Option type compatibility with other providers
- **Risk:** The `Option` type defined in `claude.go` is `func(*ClaudeProvider)` -- using it for Factory would be a type error.
- **Impact:** Compilation failure if we try to reuse Claude's `Option` type.
- **Mitigation:** Define `FactoryOption` as a separate type. Already addressed in Engineering Decisions.
- **Validation time:** < 2 minutes -- compiler catches this immediately.

---

## Recommended API Surface

### Exported symbols in `internal/provider/factory.go`

```go
// FactoryOption configures a FactoryProvider.
type FactoryOption func(*FactoryProvider)

// WithFactoryBaseDir sets the base directory for Factory skill files.
func WithFactoryBaseDir(dir string) FactoryOption

// FactoryProvider reads and writes skills from Factory AI Droid's skill directory.
type FactoryProvider struct { /* unexported fields */ }

// NewFactoryProvider creates a FactoryProvider.
// Default baseDir: .factory/skills/ (project-level, relative to cwd).
func NewFactoryProvider(opts ...FactoryOption) *FactoryProvider

// Name returns "factory".
func (p *FactoryProvider) Name() string

// ListSkills scans baseDir for subdirectories containing SKILL.md.
func (p *FactoryProvider) ListSkills() ([]Skill, error)

// ReadSkill reads <baseDir>/<name>/SKILL.md.
func (p *FactoryProvider) ReadSkill(name string) (*Skill, error)

// WriteSkill writes <baseDir>/<skill.Name>/SKILL.md with YAML frontmatter.
func (p *FactoryProvider) WriteSkill(skill Skill) error

// SkillDir returns the base directory.
func (p *FactoryProvider) SkillDir() string
```

### Unexported helpers

```go
// factoryFrontmatter is the YAML structure for Factory skill frontmatter.
type factoryFrontmatter struct { ... }

// parseFrontmatter splits content into frontmatter struct + body string.
// Returns (nil, fullContent, nil) if no frontmatter is present.
// Returns error if frontmatter delimiters are found but YAML parsing fails.
func parseFrontmatter(content string) (*factoryFrontmatter, string, error)

// serializeFrontmatter produces the ---/yaml/--- block + body.
func serializeFrontmatter(name, description, body string) string
```

---

## Folder Structure

```
internal/provider/
  provider.go        # (existing) Provider interface + Skill model
  registry.go        # (existing) Global provider registry
  claude.go          # (existing) Claude Code provider -- reference impl
  claude_test.go     # (existing) Claude tests -- reference for test style
  factory.go         # (NEW) Factory AI Droid provider
  factory_test.go    # (NEW) Factory provider tests
```

Ownership: I own `factory.go` and `factory_test.go`. Everything else is read-only reference.

---

## Tighten the plan into 4-7 small tasks

### Task 1: Scaffold FactoryProvider struct + constructor + Name/SkillDir

- **Outcome:** `FactoryProvider` struct with `baseDir` field, `NewFactoryProvider` constructor with `FactoryOption` functional options, `Name()` returns `"factory"`, `SkillDir()` returns `baseDir`, `init()` registers with the registry.
- **Files to create/modify:** `internal/provider/factory.go` (create)
- **Exact verification:**
  ```bash
  go build ./...
  go vet ./...
  ```
- **Suggested commit message:** `feat(provider): scaffold FactoryProvider struct with constructor and registration`

### Task 2: Implement frontmatter parsing helpers

- **Outcome:** `parseFrontmatter()` splits a file's content into a `factoryFrontmatter` struct and body string. Handles: valid frontmatter, no frontmatter, empty content, frontmatter-only (no body).
- **Files to create/modify:** `internal/provider/factory.go` (add helpers)
- **Exact verification:**
  ```bash
  go build ./...
  go vet ./...
  ```
- **Suggested commit message:** `feat(provider): add Factory frontmatter parsing helpers`

### Task 3: Implement ReadSkill and ListSkills

- **Outcome:** `ReadSkill(name)` reads `<baseDir>/<name>/SKILL.md`, parses frontmatter, maps to `Skill`. `ListSkills()` scans baseDir subdirectories for `SKILL.md` files.
- **Files to create/modify:** `internal/provider/factory.go` (add methods)
- **Exact verification:**
  ```bash
  go build ./...
  go vet ./...
  ```
- **Suggested commit message:** `feat(provider): implement Factory ReadSkill and ListSkills`

### Task 4: Implement WriteSkill with frontmatter serialization

- **Outcome:** `WriteSkill(skill)` creates `<baseDir>/<skill.Name>/SKILL.md` with YAML frontmatter (`name`, `description`) followed by `Content` as body. Creates directory with `os.MkdirAll`.
- **Files to create/modify:** `internal/provider/factory.go` (add method + serialize helper)
- **Exact verification:**
  ```bash
  go build ./...
  go vet ./...
  ```
- **Suggested commit message:** `feat(provider): implement Factory WriteSkill with frontmatter serialization`

### Task 5: Write comprehensive test suite

- **Outcome:** All 17 test cases from the Tests section pass. Covers frontmatter parsing, no-frontmatter fallback, round-trip, empty files, malformed YAML error, edge cases, list operations.
- **Files to create/modify:** `internal/provider/factory_test.go` (create)
- **Exact verification:**
  ```bash
  go test ./internal/provider/ -v -run TestFactory
  go test ./...
  ```
- **Suggested commit message:** `test(provider): add comprehensive Factory provider tests`

### Task 6: Final integration check -- all tests pass, no regressions

- **Outcome:** `go test ./...` passes with zero failures. `go vet ./...` clean. Factory provider appears in `provider.List()` output.
- **Files to create/modify:** None (verification only).
- **Exact verification:**
  ```bash
  go test ./... -v
  go vet ./...
  ```
- **Suggested commit message:** N/A (verification step, no new code).

---

## CLAUDE.md contributions (do NOT write the file; propose content)

### From Factory Provider Dev

**Coding style rules:**
- Factory provider follows the same pattern as `claude.go`: unexported struct fields, exported constructor with functional options, `init()` registration.
- Frontmatter parsing: only the first `---`...`---` pair is treated as frontmatter. The file must START with `---` on line 1.
- Error wrapping: all errors use `fmt.Errorf("factory: <context>: %w", err)`.
- Option type: `FactoryOption func(*FactoryProvider)` -- do NOT mix with Claude's `Option` type.

**Dev commands:**
```bash
go test ./internal/provider/ -v -run TestFactory   # run Factory tests only
go test ./internal/provider/ -v                     # run all provider tests
go test ./...                                       # run everything
go vet ./...                                        # lint check
```

**Before you commit checklist:**
- [ ] `go test ./...` passes
- [ ] `go vet ./...` clean
- [ ] No import cycles introduced
- [ ] Factory registered as `"factory"` in init()
- [ ] WriteSkill always emits frontmatter (even if source had none)

**Guardrails:**
- Do NOT parse `model` from frontmatter into the Skill model -- it is provider-specific metadata.
- Do NOT translate argument placeholders -- Factory has no argument syntax.
- Do NOT modify `provider.go` or `registry.go`.

---

## EXPLAIN.md contributions (do NOT write the file; propose outline bullets)

### Factory AI Droid Provider

- **Format:** Markdown files with optional YAML frontmatter in `.factory/skills/<name>/SKILL.md`
- **Frontmatter fields:** `name` (string), `description` (string), `model` (string, optional -- ignored by skill-sync)
- **Directory structure:** Each skill is a subdirectory under `.factory/skills/`, containing a `SKILL.md` file
- **Read behavior:**
  - If frontmatter present: `name` and `description` from YAML, body after second `---` is `Content`
  - If no frontmatter: directory name is `Name`, first `# ` line is `Description`, entire file is `Content`
- **Write behavior:** Always writes frontmatter block with `name` and `description`, then body
- **Key tradeoff:** Plain markdown files gain frontmatter after sync -- acceptable since skill-sync is the canonical writer
- **Limitation:** Extra YAML fields in frontmatter (e.g., `tags`, `version`) are not preserved on round-trip
- **How to validate:** `go test ./internal/provider/ -v -run TestFactory`

---

## READY FOR APPROVAL
