# Claude Provider Dev -- Plan Document

## You are in PLAN MODE.

### Project
I want to build **skill-sync**, a Go CLI that syncs AI assistant skills from a primary provider to all others with drift detection.

**Goal:** build the **Claude Code provider** -- the first and most important provider implementation, since Claude Code is the likely source of truth for users' skills.

### Role + Scope
- **Role:** Claude Provider Dev
- **Scope:** I own `internal/provider/claude.go` and `internal/provider/claude_test.go`. I implement the `Provider` interface for Claude Code's skill format. I do NOT own the Provider interface definition, the registry, the sync engine, config parsing, or CLI commands.
- **File you will write:** `/docs/core/plans/claude-provider.md`
- **No-touch zones:** do not edit any files outside `internal/provider/claude.go` and `internal/provider/claude_test.go`. Do not write code in this phase.

---

## Functional Requirements
- **FR1:** `ListSkills()` reads all `*.md` files from the Claude Code commands directory (`~/.claude/commands/`) and returns them as `[]Skill`.
- **FR2:** `ReadSkill(name)` reads a single `<name>.md` file, parses it into a `Skill` struct with name, description, content, arguments, and source path.
- **FR3:** `WriteSkill(skill)` writes a `Skill` to `<baseDir>/<skill.Name>.md` in the correct Claude Code format (description as `# ` first line, then content body).
- **FR4:** Parsing extracts the first line as description if it starts with `# `, scans for `$ARGUMENTS` and `${ARG_NAME}` patterns to populate the Arguments field.
- **FR5:** Provider self-registers via `init()` using `registry.Register("claude", ...)`.
- **Tests required:** Unit tests covering all public methods, edge cases (empty files, no description, no arguments, multiple arguments, special characters in filenames).

## Non-Functional Requirements
- Language/runtime: Go 1.22+
- No external dependencies beyond stdlib
- Observability: N/A for this component (it's a library, not a service)
- Safety: never silently overwrite files without the caller requesting it; surface all filesystem errors clearly
- Documentation: exported types and functions get clear godoc comments
- Performance: N/A -- filesystem I/O on a handful of small markdown files; no benchmarks needed

---

## Assumptions / System Model
- **Deployment environment:** local CLI tool, runs on user's machine
- **Failure modes:** directory doesn't exist, permission denied, malformed markdown, empty files, filenames with spaces or special characters
- **Delivery guarantees:** N/A (not a distributed system)
- **Multi-tenancy:** none; single user's local filesystem

---

## Data Model

### Skill (defined by Provider Architect, consumed here)
```go
type Skill struct {
    Name        string   // derived from filename without .md extension
    Description string   // first line if it starts with "# " (stripped of prefix)
    Content     string   // full file content (including the description line)
    Arguments   []string // extracted $ARGUMENTS or ${ARG_NAME} placeholders
    SourcePath  string   // absolute path to the .md file
}
```

### ClaudeProvider struct
```go
type ClaudeProvider struct {
    baseDir string // defaults to ~/.claude/commands/
}
```

- **Validation rules:** skill name must not be empty; baseDir must be a valid directory (or creatable on WriteSkill)
- **Versioning:** N/A -- skills are plain files, no versioning beyond what the filesystem provides
- **Persistence:** direct filesystem read/write; no database or caching

---

## APIs

### Constructor
```go
func NewClaudeProvider(opts ...Option) *ClaudeProvider
```
- Returns a `*ClaudeProvider` with baseDir defaulting to `~/.claude/commands/`
- Functional options pattern for overriding baseDir (essential for testing)

### Option type
```go
type Option func(*ClaudeProvider)

func WithBaseDir(dir string) Option
```

### Provider interface methods
```go
func (p *ClaudeProvider) Name() string
// Returns "claude"

func (p *ClaudeProvider) ListSkills() ([]Skill, error)
// Globs baseDir/*.md, parses each file, returns slice
// Returns empty slice (not error) if directory has no .md files
// Returns error if baseDir doesn't exist or is unreadable

func (p *ClaudeProvider) ReadSkill(name string) (*Skill, error)
// Reads baseDir/name.md, parses into Skill
// Returns error if file doesn't exist

func (p *ClaudeProvider) WriteSkill(skill Skill) error
// Writes skill to baseDir/skill.Name.md
// Creates baseDir if it doesn't exist (os.MkdirAll)
// If description is non-empty, writes "# <description>\n" as first line
// Then writes content body

func (p *ClaudeProvider) SkillDir() string
// Returns baseDir
```

### Error semantics
- All errors wrapped with `fmt.Errorf("claude: <context>: %w", err)`
- File-not-found errors propagate as-is (wrapped); callers can check with `os.IsNotExist`
- ListSkills on an empty directory returns `([]Skill{}, nil)`, not an error

---

## Architecture / Component Boundaries

```
internal/provider/
    provider.go    -- Provider interface + Skill model  (owned by Provider Architect)
    registry.go    -- Register/Get functions             (owned by Provider Architect)
    claude.go      -- ClaudeProvider implementation      (MY SCOPE)
    claude_test.go -- Tests for ClaudeProvider            (MY SCOPE)
```

- **ClaudeProvider** depends on: `Provider` interface, `Skill` struct, `registry.Register()`
- **Sync engine** depends on: `ClaudeProvider` via the `Provider` interface (no direct coupling)
- **I do NOT touch:** registry.go, provider.go, engine.go, diff.go, config.go, or any cmd/ files

---

## Correctness Invariants

1. **Round-trip fidelity:** `WriteSkill(skill)` followed by `ReadSkill(skill.Name)` must return a Skill with identical Name, Description, Content, and Arguments.
2. **Name derivation:** Skill.Name always equals the filename without the `.md` extension.
3. **Description extraction:** Only the first line is considered for description, and only if it starts with exactly `# ` (hash + space). Lines starting with `##` or `#word` are NOT descriptions.
4. **Argument extraction:** All `$ARGUMENTS` literals and `${WORD}` patterns (where WORD is `[A-Z_][A-Z0-9_]*`) are captured. Duplicates are deduplicated. Order is stable (first occurrence).
5. **SourcePath accuracy:** SourcePath is always the absolute path to the file on disk.
6. **No data loss on write:** WriteSkill preserves the full content faithfully; no truncation or encoding issues.
7. **Empty directory is not an error:** ListSkills returns an empty slice, not an error, when the directory exists but contains no `.md` files.

---

## Tests

### Unit tests (`internal/provider/claude_test.go`)

All tests use `t.TempDir()` for isolation. Table-driven where applicable.

| Test | What it verifies |
|------|-----------------|
| `TestListSkills_MultipleFiles` | Creates 3 .md files in temp dir, calls ListSkills, verifies count and names |
| `TestListSkills_EmptyDir` | Empty temp dir returns `([]Skill{}, nil)` |
| `TestListSkills_NonExistentDir` | Missing dir returns an error |
| `TestReadSkill_WithDescription` | File starting with `# Desc` -- verifies Description="Desc" |
| `TestReadSkill_NoDescription` | File starting with regular text -- verifies Description="" |
| `TestReadSkill_WithArguments` | File containing `$ARGUMENTS` and `${QUERY}` -- verifies Arguments slice |
| `TestReadSkill_NoArguments` | File with no argument patterns -- verifies Arguments is empty |
| `TestReadSkill_DuplicateArguments` | `$ARGUMENTS` appears twice -- verifies deduplication |
| `TestReadSkill_EmptyFile` | Empty .md file -- verifies no panic, empty fields |
| `TestReadSkill_NonExistent` | Missing file returns error |
| `TestWriteSkill_Basic` | Writes a skill, reads file back, verifies content on disk |
| `TestWriteSkill_CreatesDir` | baseDir doesn't exist yet -- verifies MkdirAll behavior |
| `TestWriteSkill_RoundTrip` | Write then Read, compare all fields for equality |
| `TestWriteSkill_NoDescription` | Skill with empty description -- verifies no `# ` prefix line |
| `TestName` | Verifies `Name()` returns `"claude"` |
| `TestSkillDir` | Verifies `SkillDir()` returns the configured baseDir |

### Commands
```bash
go test ./internal/provider/ -v -run TestClaude
go test ./internal/provider/ -race
```

### Integration tests
N/A -- this is a filesystem-based provider; unit tests with `t.TempDir()` provide sufficient integration coverage. True E2E tests are owned by the QE Lead in the validation phase.

### Property/fuzz tests
Optional future work: fuzz `ReadSkill` with random file contents to verify no panics. Not in scope for this phase.

---

## Benchmarks + "Success"

N/A -- The Claude provider reads a handful of small markdown files from the local filesystem. There is no performance-critical path to benchmark. Success is defined by:
- All unit tests pass
- Round-trip fidelity is proven
- Edge cases (empty file, no description, no arguments) are handled gracefully
- Code integrates cleanly with the Provider interface defined by the Architect

---

## Engineering Decisions & Tradeoffs

### Decision 1: Functional options for constructor vs. config struct
- **Decision:** Use functional options (`WithBaseDir(dir string)`) for `NewClaudeProvider`
- **Alternatives considered:** (a) Pass config struct; (b) Pass baseDir as a direct argument
- **Why:** Functional options are idiomatic Go for optional configuration. They allow adding new options (e.g., `WithProjectLevel()` in the future) without breaking the constructor signature. A direct string argument would be simpler but less extensible.
- **Tradeoff acknowledged:** Slightly more boilerplate (Option type + WithX functions) for a constructor that currently has only one option. Accepted because the pattern pays off when project-level skill dirs are added later.

### Decision 2: Content field stores full file content (including description line)
- **Decision:** `Skill.Content` contains the entire file content, including the `# Description` first line if present.
- **Alternatives considered:** (a) Store Content as body-only (excluding the description line); (b) Store raw bytes instead of string.
- **Why:** Storing full content means `WriteSkill` can write `Content` as-is for providers that don't have a separate description concept. It also ensures no information is lost during round-trips. If Content excluded the description, we'd need to reconstruct it on write, which is error-prone.
- **Tradeoff acknowledged:** The description appears in both `Description` and `Content` fields, which is redundant. But redundancy is safer than lossy transformation. The sync engine can choose which field to use per target provider.

### Decision 3: Argument regex pattern
- **Decision:** Match both `$ARGUMENTS` (literal) and `${IDENTIFIER}` where IDENTIFIER matches `[A-Z_][A-Z0-9_]*`
- **Alternatives considered:** (a) Only match `$ARGUMENTS`; (b) Also match lowercase `${arg_name}` patterns
- **Why:** Claude Code documentation shows `$ARGUMENTS` as the primary placeholder, but `${QUERY}` style is also used. Restricting to uppercase keeps the regex simple and avoids false positives with shell-like `${lowercase}` patterns that might appear in code examples.
- **Tradeoff acknowledged:** Skills that use lowercase argument placeholders (if any exist) won't have their arguments detected. This can be relaxed later if needed.

---

## Risks & Mitigations

### Risk 1: Provider interface not finalized when I start coding
- **Risk:** The Provider Architect hasn't merged the interface definition yet, so `Skill` struct fields or method signatures could change.
- **Impact:** My implementation may not compile against the final interface; rework required.
- **Mitigation:** Read the Architect's plan doc before coding. If the interface is still in flux, stub it locally with the expected shape and reconcile once merged. The interface is well-specified in the kickoff doc, so major surprises are unlikely.
- **Validation time:** < 5 minutes (read Architect's plan, compare against kickoff spec)

### Risk 2: Claude Code skill format has undocumented conventions
- **Risk:** Real-world Claude Code skills may use patterns not covered in the PROJECT.md spec (e.g., YAML frontmatter, nested directories, non-`.md` extensions).
- **Impact:** ListSkills/ReadSkill may miss or mangle some skills.
- **Mitigation:** Examine the actual `~/.claude/commands/` directory on this machine for real skill files. Design parsing to be lenient (don't fail on unexpected content, just skip optional fields). Start with the documented format and extend if real files reveal more patterns.
- **Validation time:** < 5 minutes (ls + cat a few real skill files)

### Risk 3: Filename edge cases
- **Risk:** Skill filenames may contain spaces, unicode characters, or dots (e.g., `my skill.md`, `v2.0-helper.md`).
- **Impact:** Name derivation (`strings.TrimSuffix(filename, ".md")`) could produce unexpected names; WriteSkill could create filenames that don't round-trip.
- **Mitigation:** Use `filepath.Ext()` for extension stripping (handles multiple dots correctly). Add test cases with spaces and special characters. Document that skill names are derived as-is from filenames without sanitization.
- **Validation time:** < 10 minutes (write targeted test cases)

### Risk 4: WriteSkill file permission issues
- **Risk:** On some systems, the `~/.claude/commands/` directory may not exist or may have restrictive permissions.
- **Impact:** WriteSkill fails with a permission error that's confusing to the user.
- **Mitigation:** Use `os.MkdirAll(baseDir, 0755)` before writing. Wrap errors with clear context (`"claude: cannot create skill directory: %w"`). Test with a temp dir that simulates permission issues.
- **Validation time:** < 5 minutes (test on temp dir with restricted perms)

---

## Recommended API Surface

### Exported functions/types (in `internal/provider/claude.go`)

| Symbol | Kind | Behavior |
|--------|------|----------|
| `ClaudeProvider` | struct | Holds `baseDir` field |
| `Option` | type | `func(*ClaudeProvider)` |
| `WithBaseDir(dir string)` | func | Returns Option that sets baseDir |
| `NewClaudeProvider(opts ...Option)` | func | Constructor; defaults baseDir to `~/.claude/commands/` |
| `(p *ClaudeProvider) Name()` | method | Returns `"claude"` |
| `(p *ClaudeProvider) ListSkills()` | method | Globs `*.md`, parses each, returns `[]Skill` |
| `(p *ClaudeProvider) ReadSkill(name)` | method | Reads and parses `<name>.md` |
| `(p *ClaudeProvider) WriteSkill(skill)` | method | Writes skill to `<name>.md` |
| `(p *ClaudeProvider) SkillDir()` | method | Returns baseDir |
| `init()` | func | Registers "claude" provider in registry |

---

## Folder Structure

```
internal/provider/
    provider.go      # Provider interface + Skill model  (Provider Architect)
    registry.go      # Register/Get                       (Provider Architect)
    claude.go        # ClaudeProvider                     (ME)
    claude_test.go   # Tests                              (ME)
```

---

## Tighten the plan into 4-7 small tasks

### Task 1: Scaffold ClaudeProvider struct + constructor + Name/SkillDir
- **Outcome:** `ClaudeProvider` struct with `baseDir` field, `Option` type, `WithBaseDir`, `NewClaudeProvider`, `Name()`, `SkillDir()` all compile.
- **Files to create/modify:** `internal/provider/claude.go`
- **Exact verification:** `go build ./internal/provider/`
- **Suggested commit message:** `feat(provider): scaffold ClaudeProvider struct and constructor`

### Task 2: Implement ReadSkill with parsing logic
- **Outcome:** `ReadSkill(name)` reads a `.md` file and correctly populates `Skill.Name`, `Description`, `Content`, `Arguments`, and `SourcePath`.
- **Files to create/modify:** `internal/provider/claude.go`
- **Exact verification:** `go build ./internal/provider/`
- **Suggested commit message:** `feat(provider): implement ClaudeProvider.ReadSkill with description and argument parsing`

### Task 3: Implement ListSkills
- **Outcome:** `ListSkills()` globs `*.md` in baseDir, calls parse logic for each, returns `[]Skill`.
- **Files to create/modify:** `internal/provider/claude.go`
- **Exact verification:** `go build ./internal/provider/`
- **Suggested commit message:** `feat(provider): implement ClaudeProvider.ListSkills`

### Task 4: Implement WriteSkill
- **Outcome:** `WriteSkill(skill)` creates `<name>.md` with `# Description` first line (if set) followed by content body. Creates baseDir if missing.
- **Files to create/modify:** `internal/provider/claude.go`
- **Exact verification:** `go build ./internal/provider/`
- **Suggested commit message:** `feat(provider): implement ClaudeProvider.WriteSkill`

### Task 5: Add init() registration
- **Outcome:** `init()` calls `registry.Register("claude", NewClaudeProvider())` so the provider is available at import time.
- **Files to create/modify:** `internal/provider/claude.go`
- **Exact verification:** `go build ./internal/provider/`
- **Suggested commit message:** `feat(provider): register Claude provider in init`

### Task 6: Write comprehensive unit tests
- **Outcome:** All 16 test cases listed in the Tests section pass. Covers ListSkills, ReadSkill, WriteSkill, round-trip, edge cases.
- **Files to create/modify:** `internal/provider/claude_test.go`
- **Exact verification:** `go test ./internal/provider/ -v -race -run Test`
- **Suggested commit message:** `test(provider): add comprehensive Claude provider unit tests`

---

## CLAUDE.md contributions (proposed content, do NOT write the file)

### From Claude Provider Dev
- **Coding style:**
  - All errors in `claude.go` wrapped with `fmt.Errorf("claude: <context>: %w", err)`
  - Use `filepath` package for all path operations (not `path` or string concat)
  - Use `os.UserHomeDir()` for `~` expansion; never hardcode home directory paths
  - Functional options pattern for provider constructors
- **Dev commands:**
  - `go test ./internal/provider/ -v -race` -- run provider tests
  - `go test ./internal/provider/ -run TestClaude` -- run only Claude provider tests
- **Before you commit checklist:**
  - [ ] `go vet ./internal/provider/`
  - [ ] `go test ./internal/provider/ -race`
  - [ ] All tests pass, no data races
  - [ ] Round-trip test (WriteSkill -> ReadSkill) still passes
- **Guardrails:**
  - Never use `os.UserHomeDir()` in tests -- always use `t.TempDir()`
  - The `Content` field stores the FULL file content including description line -- do not strip it

---

## EXPLAIN.md contributions (proposed outline bullets)

### Flow / Architecture
- ClaudeProvider reads/writes skills from `~/.claude/commands/*.md`
- Each `.md` file = one skill; filename (minus extension) = skill name
- First line starting with `# ` is extracted as description
- `$ARGUMENTS` and `${UPPER_CASE}` patterns are extracted as arguments

### Key Engineering Decisions + Tradeoffs
- Content stores full file (including description line) -- redundant but lossless
- Functional options constructor -- extensible for future options (e.g., project-level dirs)
- Uppercase-only argument pattern -- avoids false positives in code examples

### Limits of MVP + Next Steps
- User-level skills only (`~/.claude/commands/`); project-level (`.claude/commands/`) is future work
- No file watching or caching; reads from disk on every call
- No support for nested subdirectories within the commands folder

### How to Run Locally + How to Validate
- `go test ./internal/provider/ -v -race` to run all provider tests
- Create a test skill: `echo "# Test\nHello $ARGUMENTS" > ~/.claude/commands/test-skill.md`
- Verify parsing: build and call `NewClaudeProvider().ReadSkill("test-skill")`

---

## READY FOR APPROVAL
