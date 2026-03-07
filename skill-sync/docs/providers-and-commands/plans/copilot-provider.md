# Copilot Provider Plan

## You are in PLAN MODE.

### Project
I want to implement the **GitHub Copilot provider** for skill-sync.

**Goal:** build a **CopilotProvider** that reads and writes `.prompt.md` files from `.github/prompts/`, implementing the Provider interface so skill-sync can sync skills to/from GitHub Copilot prompt files.

### Role + Scope
- **Role:** Copilot Provider Dev
- **Scope:** `internal/provider/copilot.go` and `internal/provider/copilot_test.go`. I own only the Copilot provider implementation and its tests. I do NOT own the Provider interface, registry, sync engine, diff engine, config, CLI commands, or other providers.
- **File you will write:** `/docs/providers-and-commands/plans/copilot-provider.md`
- **No-touch zones:** do not edit any other files; do not write code.

---

## Functional Requirements
- FR1: `CopilotProvider` implements the `Provider` interface (`Name()`, `ListSkills()`, `ReadSkill()`, `WriteSkill()`, `SkillDir()`)
- FR2: Reads `.prompt.md` files from a configurable base directory (default: `.github/prompts/`)
- FR3: Parses skill name from filename by stripping `.prompt.md` extension (e.g., `review-code.prompt.md` -> `review-code`)
- FR4: Extracts description from first `# ` line (same convention as Claude provider)
- FR5: Preserves `#file:` references and all other content verbatim in `Content` field
- FR6: `WriteSkill` writes `skill.Content` directly to `<name>.prompt.md` (Content already includes `# description` line if present — same as Claude provider)
- FR7: Registers itself as `"copilot"` via `init()` function
- Tests required: unit tests covering all Provider interface methods, edge cases, and round-trip fidelity
- Metrics required: N/A -- CLI tool, no Prometheus

## Non-Functional Requirements
- Language/runtime: Go 1.22+
- Local dev: `go test ./internal/provider/...`
- Observability: N/A -- provider is a library package
- Safety: All errors wrapped with `fmt.Errorf("copilot: context: %w", err)`; graceful handling of missing directories, empty files
- Documentation: CLAUDE.md + EXPLAIN.md contributions (proposed below)
- Performance: N/A -- file I/O on local disk, no hot path

---

## Assumptions / System Model
- Deployment environment: local CLI tool (no docker, no k8s)
- Failure modes: directory doesn't exist, file read/write permission errors, malformed filenames
- Delivery guarantees: N/A -- local filesystem operations
- Multi-tenancy: N/A

---

## Data Model (as relevant to this role)

The Copilot provider maps between on-disk `.prompt.md` files and the normalized `Skill` struct:

| Skill field | Copilot source |
|---|---|
| `Name` | Filename minus `.prompt.md` extension |
| `Description` | First line if it starts with `# ` (strip prefix) |
| `Content` | Full file content (including description line) |
| `Arguments` | Not extracted (Copilot uses `#file:` references, not `$ARGUMENTS` -- different concept) |
| `SourcePath` | Absolute path to the `.prompt.md` file |

Validation rules:
- Skill name must be non-empty (derived from filename)
- `.prompt.md` extension is required -- `.md` files without `.prompt` prefix are ignored
- Empty files are valid (empty Content, no Description)

Versioning: N/A -- files are overwritten on sync (no versioning strategy needed).

---

## APIs (as relevant to this role)

The Copilot provider exposes the standard `Provider` interface. No new API surface is introduced.

```go
// Constructor
func NewCopilotProvider(opts ...CopilotOption) *CopilotProvider

// Provider interface implementation
func (p *CopilotProvider) Name() string              // returns "copilot"
func (p *CopilotProvider) ListSkills() ([]Skill, error)
func (p *CopilotProvider) ReadSkill(name string) (*Skill, error)
func (p *CopilotProvider) WriteSkill(skill Skill) error
func (p *CopilotProvider) SkillDir() string
```

The `Option` type is reused from the Claude provider (`WithBaseDir`). Since `Option` is defined as `func(*ClaudeProvider)`, the Copilot provider needs its own option type or we need a shared approach. Looking at the existing code, `Option` is `func(*ClaudeProvider)` -- so Copilot will define `CopilotOption func(*CopilotProvider)` and a `WithCopilotBaseDir(dir string) CopilotOption` constructor, OR we generalize. Given the pattern in claude.go where Option is Claude-specific, each provider will have its own option type to keep them decoupled.

**Decision:** Define `CopilotOption func(*CopilotProvider)` and `WithCopilotBaseDir` to stay consistent with the Claude provider's pattern of provider-specific option types. This avoids coupling providers to each other.

---

## Architecture / Component Boundaries

Components I touch:
- **CopilotProvider struct** -- holds `baseDir` field, defaults to `.github/prompts/`
- **init() registration** -- registers with the global provider registry as `"copilot"`

How config changes propagate: The provider is instantiated once at startup via `init()`. `baseDir` override is via functional options at construction time.

Concurrency model: N/A -- providers are called sequentially by the sync engine.

Backpressure: N/A -- local filesystem.

Key design note: The Claude provider's `WriteSkill` writes `skill.Content` directly (it includes the `# description` line). The Copilot provider should follow the same convention -- `Content` is the full file content. When writing a skill that originated from a non-Copilot source, the `Content` field already contains the full text (possibly with `# description` as first line from Claude format). Since Copilot also uses `# ` for description, the content passes through unchanged. No format translation of the body is needed for this provider pair.

---

## Correctness Invariants

1. **Round-trip fidelity:** `WriteSkill(s)` followed by `ReadSkill(s.Name)` returns a skill with identical `Content` and correctly parsed `Description` and `Name`.
2. **Extension handling:** Only `*.prompt.md` files are listed -- not `*.md`, not `*.prompt`, not other extensions.
3. **Name derivation:** Skill name is always `filename - ".prompt.md"` (e.g., `foo-bar.prompt.md` -> `foo-bar`).
4. **Description extraction:** Only first line matching `^# ` is treated as description. `##`, `#no-space`, and non-heading first lines produce empty description.
5. **Empty file handling:** Empty `.prompt.md` files produce a valid Skill with empty Content and Description.
6. **Directory creation:** `WriteSkill` creates the base directory if it doesn't exist.
7. **Error wrapping:** All returned errors are wrapped with `"copilot: "` prefix context.

---

## Tests

### Unit tests (`internal/provider/copilot_test.go`)

Following the exact patterns from `claude_test.go`:

| Test | What it verifies |
|---|---|
| `TestCopilotName` | `Name()` returns `"copilot"` |
| `TestCopilotSkillDir` | `SkillDir()` returns configured baseDir |
| `TestCopilotListSkills_MultipleFiles` | Lists all `.prompt.md` files, correct names |
| `TestCopilotListSkills_IgnoresNonPromptMd` | `.md` files without `.prompt` are ignored |
| `TestCopilotListSkills_EmptyDir` | Returns empty slice for empty directory |
| `TestCopilotListSkills_NonExistentDir` | Returns error for missing directory |
| `TestCopilotReadSkill_WithDescription` | Parses `# Description` from first line |
| `TestCopilotReadSkill_NoDescription` | No description when first line isn't `# ` |
| `TestCopilotReadSkill_HashWithoutSpace` | `#notadescription` -> empty description |
| `TestCopilotReadSkill_DoubleHash` | `## Section` -> empty description |
| `TestCopilotReadSkill_EmptyFile` | Empty file -> valid Skill, all fields empty |
| `TestCopilotReadSkill_NonExistent` | Error for missing skill |
| `TestCopilotReadSkill_WithFileReferences` | `#file:path` preserved in Content |
| `TestCopilotWriteSkill_Basic` | Writes correct content to `<name>.prompt.md` |
| `TestCopilotWriteSkill_CreatesDir` | Creates nested directory if needed |
| `TestCopilotWriteSkill_RoundTrip` | Write then read produces identical skill |
| `TestCopilotWriteSkill_RoundTrip_NoDescription` | Round-trip with no description line |

All tests use `t.TempDir()` for isolation.

### Commands
- `go test ./internal/provider/... -v -run TestCopilot`
- `go test ./internal/provider/... -count=1` (no cache)
- `go vet ./internal/provider/...`

### Integration / fuzz / failure injection tests
N/A for this role -- provider is a pure filesystem abstraction. The sync engine integration tests (owned by CLI Commands Dev) will exercise cross-provider scenarios.

---

## Benchmarks + "Success"

N/A -- The Copilot provider performs simple file I/O (glob + read/write). There is no hot path or performance-sensitive operation that warrants benchmarking. The sync engine benchmarks (if any) would cover end-to-end throughput.

---

## Engineering Decisions & Tradeoffs

### Decision 1: Provider-specific option types

- **Decision:** Define `CopilotOption func(*CopilotProvider)` rather than sharing Claude's `Option` type
- **Alternatives considered:** (A) Make `Option` generic/shared across providers; (B) Use a common `WithBaseDir` that works for all
- **Why:** The existing `Option` is `func(*ClaudeProvider)` -- it's Claude-specific. Making it generic would require changing Phase 1 code. Each provider having its own option type is simple, decoupled, and follows the established pattern.
- **Tradeoff acknowledged:** Slight code duplication (each provider defines its own option boilerplate). Acceptable for 4 providers.

### Decision 2: Content is the full file (not body-only)

- **Decision:** `Content` stores the entire file content including the `# description` first line, matching the Claude provider's behavior
- **Alternatives considered:** Store body-only in `Content` and reconstruct the `# description` line on write
- **Why:** The Claude provider stores full file content in `Content`. If we split description out, round-trip fidelity breaks -- writing a Claude skill to Copilot would lose the description line from Content and need reconstruction. Keeping `Content` = full file is simpler and matches the reference implementation.
- **Tradeoff acknowledged:** `Description` is redundant with the first line of `Content`. Callers who modify `Description` must also update the first line of `Content` for consistency. This is the same tradeoff the Claude provider already makes.

### Decision 3: No argument extraction for Copilot

- **Decision:** `Arguments` field is always `nil` for Copilot-sourced skills
- **Alternatives considered:** Extract `#file:` references as "arguments"
- **Why:** Copilot's `#file:` references are contextual file includes, not user-input arguments like Claude's `$ARGUMENTS`. They serve a fundamentally different purpose. The PROJECT.md explicitly states: "Arguments are stored verbatim in the Skill model and NOT translated between providers."
- **Tradeoff acknowledged:** If a Copilot skill is used as a sync source, target providers won't see any arguments. This is correct -- the `#file:` references are preserved in Content and will appear as literal text in the target format.

---

## Risks & Mitigations

### Risk 1: `.prompt.md` glob pattern collides with other `.md` files
- **Risk:** `filepath.Glob("*.prompt.md")` might not correctly filter when other `.md` files exist in the same directory
- **Impact:** Non-prompt files could be incorrectly listed as skills
- **Mitigation:** The glob `*.prompt.md` is specific -- Go's `filepath.Glob` matches the full suffix. Write an explicit test (`TestCopilotListSkills_IgnoresNonPromptMd`) that places both `.md` and `.prompt.md` files in the same directory and verifies only `.prompt.md` files are returned.
- **Validation time:** < 5 minutes (write test, run it)

### Risk 2: Filename edge cases with `.prompt.md` stripping
- **Risk:** A file named just `.prompt.md` (empty skill name) or `foo.bar.prompt.md` (dots in name) could cause unexpected behavior
- **Impact:** Empty skill name or incorrect name derivation
- **Mitigation:** Use `strings.TrimSuffix(filename, ".prompt.md")` which handles multi-dot filenames correctly (`foo.bar.prompt.md` -> `foo.bar`). For the empty-name edge case, document it as unsupported (same as Claude provider doesn't guard against `.md` with no name).
- **Validation time:** < 5 minutes (test with dotted filenames)

### Risk 3: init() registration conflicts
- **Risk:** If `copilot.go` and `claude.go` both register via `init()`, and tests import the package, both providers register. Tests that call `resetRegistry()` might interfere.
- **Impact:** Test flakiness or panics from duplicate registration
- **Mitigation:** Follow the exact same pattern as `claude.go` -- register in `init()`, and in tests use `NewCopilotProvider(WithCopilotBaseDir(dir))` directly without going through the registry. The registry tests already handle this with `resetRegistry()`.
- **Validation time:** < 5 minutes (run full test suite)

---

## Recommended API Surface

```go
// internal/provider/copilot.go

type CopilotOption func(*CopilotProvider)

func WithCopilotBaseDir(dir string) CopilotOption

type CopilotProvider struct {
    baseDir string
}

func NewCopilotProvider(opts ...CopilotOption) *CopilotProvider
func (p *CopilotProvider) Name() string
func (p *CopilotProvider) ListSkills() ([]Skill, error)
func (p *CopilotProvider) ReadSkill(name string) (*Skill, error)
func (p *CopilotProvider) WriteSkill(skill Skill) error
func (p *CopilotProvider) SkillDir() string

func init() // registers "copilot" provider
```

Exact behavior:
- `NewCopilotProvider()` defaults `baseDir` to `.github/prompts/` (project-level, relative to cwd -- matching the project-level convention)
- `Name()` returns `"copilot"`
- `SkillDir()` returns `baseDir`
- `ListSkills()` globs `*.prompt.md`, strips extension for names, reads each file
- `ReadSkill(name)` reads `<baseDir>/<name>.prompt.md`, parses description from first `# ` line, sets Content to full file text
- `WriteSkill(skill)` creates `baseDir` if needed, writes `skill.Content` directly to `<baseDir>/<skill.Name>.prompt.md` (same as Claude — do NOT prepend `# description` separately, since Content already includes it)

---

## Folder Structure

```
internal/provider/
  provider.go          # Skill model + Provider interface (exists, not touched)
  registry.go          # Global registry (exists, not touched)
  claude.go            # Claude provider (exists, not touched)
  claude_test.go       # Claude tests (exists, not touched)
  copilot.go           # NEW -- CopilotProvider implementation
  copilot_test.go      # NEW -- CopilotProvider tests
```

I own: `copilot.go`, `copilot_test.go`
I do not touch anything else.

---

## Step-by-Step Task Plan

### Task 1: Scaffold CopilotProvider struct and constructor
- **Outcome:** `CopilotProvider` struct with `baseDir` field, `CopilotOption` type, `WithCopilotBaseDir`, `NewCopilotProvider`, `Name()`, `SkillDir()` methods
- **Files to create/modify:** `internal/provider/copilot.go`
- **Verification:** `go vet ./internal/provider/...` passes; `go build ./internal/provider/...` compiles
- **Suggested commit:** `feat(provider): scaffold CopilotProvider struct and constructor`

### Task 2: Implement ListSkills for .prompt.md files
- **Outcome:** `ListSkills()` globs `*.prompt.md`, returns skills with correct names; handles empty dir and nonexistent dir
- **Files to create/modify:** `internal/provider/copilot.go`
- **Verification:** `go test ./internal/provider/... -v -run TestCopilotList`
- **Suggested commit:** `feat(provider): implement CopilotProvider.ListSkills`

### Task 3: Implement ReadSkill with description parsing
- **Outcome:** `ReadSkill(name)` reads `.prompt.md` file, extracts description from `# ` first line, sets Content to full file, sets SourcePath. Handles edge cases: no description, empty file, `#file:` references preserved.
- **Files to create/modify:** `internal/provider/copilot.go`
- **Verification:** `go test ./internal/provider/... -v -run TestCopilotRead`
- **Suggested commit:** `feat(provider): implement CopilotProvider.ReadSkill`

### Task 4: Implement WriteSkill and init() registration
- **Outcome:** `WriteSkill(skill)` writes Content to `<name>.prompt.md`, creates dir if needed. `init()` registers as `"copilot"`.
- **Files to create/modify:** `internal/provider/copilot.go`
- **Verification:** `go test ./internal/provider/... -v -run TestCopilotWrite`
- **Suggested commit:** `feat(provider): implement CopilotProvider.WriteSkill and register`

### Task 5: Complete test suite with round-trip and edge cases
- **Outcome:** Full `copilot_test.go` with all 17 tests listed above passing
- **Files to create/modify:** `internal/provider/copilot_test.go`
- **Verification:** `go test ./internal/provider/... -v -run TestCopilot -count=1` -- all pass; `go vet ./internal/provider/...` clean
- **Suggested commit:** `test(provider): complete CopilotProvider test suite`

---

## CLAUDE.md contributions (proposed content, do NOT write the file)

### From Copilot Provider Dev

**Coding style:**
- Provider-specific option types: `CopilotOption func(*CopilotProvider)`, not shared with other providers
- All errors prefixed with provider name: `fmt.Errorf("copilot: context: %w", err)`
- File extension for Copilot is `.prompt.md` (not `.md`) -- always use the full double extension
- `Content` field stores full file content including `# description` first line (same as Claude)

**Dev commands:**
- `go test ./internal/provider/... -v -run TestCopilot` -- run Copilot tests only
- `go test ./internal/provider/... -count=1` -- run all provider tests (no cache)
- `go vet ./internal/provider/...` -- static analysis

**Before you commit checklist:**
- [ ] `go vet ./internal/provider/...` clean
- [ ] `go test ./internal/provider/... -count=1` all pass
- [ ] No changes to existing files (claude.go, provider.go, registry.go)

**Guardrails:**
- Do NOT extract `#file:` references as Arguments -- they are file includes, not user-input placeholders
- The glob pattern must be `*.prompt.md`, never `*.md` (would collide with Claude files or other markdown)
- `strings.TrimSuffix` with `.prompt.md` handles multi-dot filenames correctly

---

## EXPLAIN.md contributions (proposed outline bullets)

### Copilot Provider
- **Format:** Copilot prompt files are pure Markdown at `.github/prompts/*.prompt.md` (project-level)
- **Flow:** `ListSkills` globs `*.prompt.md` -> `ReadSkill` reads file + parses `# ` description -> `WriteSkill` writes Content to `<name>.prompt.md`
- **Key decision:** `Content` = full file (not body-only) to match Claude provider and ensure round-trip fidelity
- **Key decision:** No argument extraction -- `#file:` references are preserved in Content but not treated as Arguments
- **Tradeoff:** Provider-specific option types cause minor duplication but keep providers fully decoupled
- **Limits:** No user-level prompt directory for Copilot (only project-level); no support for `.instructions.md` files (those are context, not skills)
- **How to validate:** `go test ./internal/provider/... -v -run TestCopilot`

---

## READY FOR APPROVAL
