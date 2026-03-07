# Gemini Provider Dev -- Plan Document

## You are in PLAN MODE.

### Project
I want to build **skill-sync**, a Go CLI that syncs AI assistant skills from a primary provider to all others with drift detection.

**Goal:** build the **Gemini CLI provider** -- the TOML-based provider that reads and writes Gemini CLI commands, enabling skill-sync to sync skills to/from Gemini CLI.

### Role + Scope
- **Role:** Gemini Provider Dev
- **Scope:** I own `internal/provider/gemini.go` and `internal/provider/gemini_test.go`. I implement the `Provider` interface for Gemini CLI's TOML-based command format. I handle TOML parsing/writing, namespaced commands (subdirectory support), and `{{args}}` argument extraction. I do NOT own the Provider interface, the registry, the sync engine, diff engine, config parsing, or CLI commands.
- **File you will write:** `/docs/providers-and-commands/plans/gemini-provider.md`
- **No-touch zones:** do not edit any files outside `internal/provider/gemini.go` and `internal/provider/gemini_test.go`. Do not write code in this phase.

---

## Functional Requirements
- **FR1:** `ListSkills()` reads all `*.toml` files from the Gemini commands directory (`~/.gemini/commands/`), including files in subdirectories, and returns them as `[]Skill`.
- **FR2:** `ReadSkill(name)` reads a single command file (handling `:` to `/` path conversion for namespaced commands), parses the TOML, and populates a `Skill` struct with name, description, content, arguments, and source path.
- **FR3:** `WriteSkill(skill)` writes a `Skill` as a TOML file with `description` and `prompt` fields. Creates subdirectories for namespaced command names (e.g., `git:commit` -> `git/commit.toml`).
- **FR4:** Parsing extracts the `prompt` field as `Skill.Content`, the `description` field as `Skill.Description`, and scans the prompt for `{{args}}` to populate `Skill.Arguments`.
- **FR5:** Provider self-registers via `init()` using `registry.Register(NewGeminiProvider())`.
- **FR6:** Namespaced commands are supported: `git/commit.toml` -> skill name `git:commit`, and writing `git:commit` creates `git/commit.toml`.
- **Tests required:** Unit tests covering TOML round-trip, description extraction, argument parsing, namespaced commands, and edge cases.

## Non-Functional Requirements
- Language/runtime: Go 1.22+
- External dependency: `github.com/BurntSushi/toml` for TOML parsing (must be added to go.mod)
- Observability: N/A for this component (library, not a service)
- Safety: never silently overwrite files; surface all filesystem and TOML parsing errors clearly
- Documentation: exported types and functions get clear godoc comments
- Performance: N/A -- filesystem I/O on a handful of small TOML files; no benchmarks needed

---

## Assumptions / System Model
- **Deployment environment:** local CLI tool, runs on user's machine
- **Failure modes:** directory doesn't exist, permission denied, malformed TOML, missing `prompt` field, empty files, filenames with special characters, deeply nested subdirectories
- **Delivery guarantees:** N/A (not a distributed system)
- **Multi-tenancy:** none; single user's local filesystem

---

## Data Model

### Skill (defined by Provider Architect, consumed here)
```go
type Skill struct {
    Name        string   // derived from relative file path (e.g., "commit" or "git:commit")
    Description string   // from TOML `description` field
    Content     string   // from TOML `prompt` field
    Arguments   []string // extracted {{args}} placeholders from prompt
    SourcePath  string   // absolute path to the .toml file
}
```

### GeminiProvider struct
```go
type GeminiProvider struct {
    baseDir string // defaults to ~/.gemini/commands/
}
```

### TOML file structure
```go
// geminiCommand represents the TOML structure of a Gemini CLI command file.
type geminiCommand struct {
    Description string `toml:"description"`
    Prompt      string `toml:"prompt"`
}
```

- **Validation rules:** `prompt` field is required (return error if missing/empty after trim); `description` is optional; skill name must not be empty
- **Versioning:** N/A -- skills are plain files, no versioning beyond what the filesystem provides
- **Persistence:** direct filesystem read/write + TOML marshal/unmarshal; no database or caching

---

## APIs

### Constructor
```go
func NewGeminiProvider(opts ...GeminiOption) *GeminiProvider
```
- Returns a `*GeminiProvider` with baseDir defaulting to `~/.gemini/commands/`
- Uses provider-specific `GeminiOption` type (Claude's `Option` operates on `*ClaudeProvider` — not reusable)

### Option type
```go
type GeminiOption func(*GeminiProvider)

func WithGeminiBaseDir(dir string) GeminiOption
```

### Provider interface methods
```go
func (p *GeminiProvider) Name() string
// Returns "gemini"

func (p *GeminiProvider) ListSkills() ([]Skill, error)
// Recursively walks baseDir for *.toml files, parses each, returns slice
// Subdirectory path becomes namespace: git/commit.toml -> name "git:commit"
// Returns empty slice (not error) if directory exists but has no .toml files
// Returns error if baseDir doesn't exist or is unreadable

func (p *GeminiProvider) ReadSkill(name string) (*Skill, error)
// Converts ":" in name to "/" for path lookup
// Reads baseDir/<path>.toml, parses TOML
// Returns error if file doesn't exist or TOML is malformed

func (p *GeminiProvider) WriteSkill(skill Skill) error
// Converts ":" in name to "/" for path construction
// Creates subdirectories if needed (os.MkdirAll)
// Writes TOML with description (if non-empty) and prompt fields
// Uses multi-line TOML string for prompt field

func (p *GeminiProvider) SkillDir() string
// Returns baseDir
```

### Error semantics
- All errors wrapped with `fmt.Errorf("gemini: <context>: %w", err)`
- File-not-found errors propagate as-is (wrapped); callers can check with `os.IsNotExist`
- Missing `prompt` field in TOML returns a descriptive error
- ListSkills on an empty directory returns `([]Skill{}, nil)`, not an error

---

## Architecture / Component Boundaries

```
internal/provider/
    provider.go      # Provider interface + Skill model  (Provider Architect)
    registry.go      # Register/Get                       (Provider Architect)
    claude.go        # ClaudeProvider                     (Claude Provider Dev)
    claude_test.go   # Claude tests                       (Claude Provider Dev)
    gemini.go        # GeminiProvider                     (MY SCOPE)
    gemini_test.go   # Gemini tests                       (MY SCOPE)
```

- **GeminiProvider** depends on: `Provider` interface, `Skill` struct, `registry.Register()`
- **GeminiProvider** also depends on: `github.com/BurntSushi/toml` (external dependency)
- **Sync engine** depends on: `GeminiProvider` via the `Provider` interface (no direct coupling)
- **I do NOT touch:** registry.go, provider.go, engine.go, diff.go, config.go, claude.go, or any cmd/ files
- **go.mod change:** must add `github.com/BurntSushi/toml` as a dependency

### Key implementation details
- **Option type isolation:** The Claude provider defines `Option func(*ClaudeProvider)`. Gemini needs its own option type `GeminiOption func(*GeminiProvider)` to avoid coupling. This follows Go convention where each type has its own functional options.
- **Recursive directory walk:** Unlike Claude (flat glob), Gemini needs `filepath.WalkDir` to discover `*.toml` files in subdirectories for namespaced commands.
- **Name <-> path conversion:** The `/` to `:` mapping is bidirectional. `nameToPath(name)` converts `git:commit` -> `git/commit.toml`. `pathToName(relPath)` converts `git/commit.toml` -> `git:commit`.
- **TOML writing:** Use `toml.NewEncoder(buf).Encode(cmd)` to produce valid TOML. Multi-line prompt strings will be encoded as TOML multi-line basic strings by the library.

---

## Correctness Invariants

1. **Round-trip fidelity:** `WriteSkill(skill)` followed by `ReadSkill(skill.Name)` must return a Skill with identical Name, Description, Content, and Arguments.
2. **Name derivation from path:** Skill.Name equals the relative path from baseDir with `.toml` stripped and `/` replaced by `:`. A top-level `foo.toml` -> name `foo`. A nested `git/commit.toml` -> name `git:commit`.
3. **Prompt field is required:** ReadSkill returns an error if the TOML file has no `prompt` field or if it is empty after trimming whitespace.
4. **Description is optional:** If the TOML has no `description` field, Skill.Description is empty (not an error).
5. **Argument extraction:** `{{args}}` patterns in the prompt are captured into Skill.Arguments. Deduplicated, stable order.
6. **SourcePath accuracy:** SourcePath is always the absolute path to the `.toml` file on disk.
7. **No data loss on write:** WriteSkill preserves the full prompt content faithfully; TOML encoding does not mangle newlines, special characters, or whitespace.
8. **Empty directory is not an error:** ListSkills returns an empty slice, not an error, when the directory exists but contains no `.toml` files.
9. **Namespace path safety:** Name-to-path conversion never traverses above baseDir (no `../` injection via skill names containing `..`).

---

## Tests

### Unit tests (`internal/provider/gemini_test.go`)

All tests use `t.TempDir()` for isolation. Table-driven where applicable.

| Test | What it verifies |
|------|-----------------|
| `TestGeminiName` | `Name()` returns `"gemini"` |
| `TestGeminiSkillDir` | `SkillDir()` returns the configured baseDir |
| `TestGeminiReadSkill_WithDescription` | TOML with both `description` and `prompt` -> correct Skill fields |
| `TestGeminiReadSkill_NoDescription` | TOML with only `prompt`, no `description` -> Description="" |
| `TestGeminiReadSkill_WithArgs` | Prompt containing `{{args}}` -> Arguments=["{{args}}"] |
| `TestGeminiReadSkill_EmptyPrompt` | TOML with empty/missing `prompt` -> returns error |
| `TestGeminiReadSkill_MultiLinePrompt` | Multi-line prompt content preserved correctly |
| `TestGeminiReadSkill_NonExistent` | Missing file returns error |
| `TestGeminiReadSkill_InvalidTOML` | Malformed TOML returns error |
| `TestGeminiReadSkill_Namespaced` | `git:commit` -> reads `git/commit.toml` |
| `TestGeminiWriteSkill_Basic` | Writes skill, reads TOML back, verifies fields |
| `TestGeminiWriteSkill_NoDescription` | Skill with empty description -> TOML has no `description` field (or empty) |
| `TestGeminiWriteSkill_CreatesDir` | baseDir doesn't exist -> MkdirAll behavior |
| `TestGeminiWriteSkill_Namespaced` | `git:commit` -> creates `git/commit.toml` with subdirectory |
| `TestGeminiWriteSkill_RoundTrip` | Write then Read, compare all fields |
| `TestGeminiWriteSkill_RoundTrip_Namespaced` | Namespaced write then read, verify fields and path |
| `TestGeminiListSkills_MultipleFiles` | Multiple .toml files -> correct count and names |
| `TestGeminiListSkills_EmptyDir` | Empty directory -> `([]Skill{}, nil)` |
| `TestGeminiListSkills_NonExistentDir` | Missing directory -> error |
| `TestGeminiListSkills_WithSubdirs` | Files in subdirectories -> namespaced skill names |
| `TestGeminiListSkills_MixedDepths` | Top-level + nested files all discovered |

### Commands
```bash
go test ./internal/provider/ -v -run TestGemini
go test ./internal/provider/ -race
```

### Integration tests
N/A -- filesystem-based provider; unit tests with `t.TempDir()` provide sufficient coverage. True E2E tests are owned by the QE Lead in the validation phase.

### Property/fuzz tests
Optional future work: fuzz `ReadSkill` with random TOML contents to verify no panics. Not in scope for this phase.

---

## Benchmarks + "Success"

N/A -- The Gemini provider reads a handful of small TOML files from the local filesystem. There is no performance-critical path to benchmark. Success is defined by:
- All unit tests pass (including TOML round-trip and namespace tests)
- Round-trip fidelity is proven for both flat and namespaced commands
- Edge cases (empty file, no description, malformed TOML, deeply nested namespaces) are handled gracefully
- TOML parsing via BurntSushi/toml works correctly for multi-line strings
- Code integrates cleanly with the Provider interface

---

## Engineering Decisions & Tradeoffs

### Decision 1: Use `github.com/BurntSushi/toml` instead of hand-rolling TOML parsing
- **Decision:** Use the BurntSushi/toml library for TOML parsing and writing.
- **Alternatives considered:** (a) Hand-roll a simple TOML parser for just `description` and `prompt` fields; (b) Use `pelletier/go-toml/v2` (another popular TOML library)
- **Why:** BurntSushi/toml is the most widely used Go TOML library, is well-tested, and handles edge cases like multi-line strings, escape sequences, and unicode correctly. Hand-rolling would be fragile and miss edge cases. The pelletier library is also solid but BurntSushi is more established and has a simpler API for our needs.
- **Tradeoff acknowledged:** Adds an external dependency to go.mod. The project already minimizes deps (only cobra + yaml.v3), so this is a meaningful addition. However, TOML parsing is complex enough that a library is clearly warranted -- the alternative is reimplementing a spec.

### Decision 2: Provider-specific option type (`GeminiOption`) instead of shared `Option`
- **Decision:** Define `GeminiOption func(*GeminiProvider)` separately from Claude's `Option func(*ClaudeProvider)`.
- **Alternatives considered:** (a) Generic option type with interface{}; (b) Shared `Option` type that accepts any provider via type assertion
- **Why:** Go's type system makes it impossible to reuse `func(*ClaudeProvider)` for `*GeminiProvider` without unsafe casts. Provider-specific option types are clean, type-safe, and follow the same pattern the Claude provider established. Each provider is a distinct concrete type.
- **Tradeoff acknowledged:** More boilerplate -- each provider defines its own option type and `WithXxxBaseDir` function. This is acceptable because there are only 4 providers, and type safety is more important than DRY for constructor options.

### Decision 3: `filepath.WalkDir` for recursive discovery instead of flat glob
- **Decision:** Use `filepath.WalkDir` to recursively discover `.toml` files in the commands directory and subdirectories.
- **Alternatives considered:** (a) Flat `filepath.Glob("*.toml")` (like Claude provider); (b) Fixed single-level subdirectory glob (`*/*.toml` + `*.toml`)
- **Why:** Gemini CLI supports arbitrary subdirectory nesting for namespaced commands. `WalkDir` handles any depth. A fixed-depth approach would miss deeply nested commands and would need to be extended later.
- **Tradeoff acknowledged:** `WalkDir` is slightly more complex than `Glob` and visits all entries (including non-.toml files). This is negligible for a small commands directory. We filter by `.toml` extension in the walk function.

### Decision 4: Omit `description` field from TOML when empty (rather than writing `description = ""`)
- **Decision:** When `Skill.Description` is empty, omit the `description` field from the written TOML file entirely.
- **Alternatives considered:** Always write both fields, with `description = ""`
- **Why:** Gemini docs list `description` as optional. Writing an empty string is technically valid but adds noise and diverges from how real Gemini commands look. Omitting it produces cleaner files that match hand-authored commands.
- **Tradeoff acknowledged:** Slight asymmetry in round-trip: a file written without description then read back will have `Description=""`, which matches the original, so there is no actual fidelity loss. The TOML struct can use `toml:",omitempty"` tag or conditional encoding.

---

## Risks & Mitigations

### Risk 1: BurntSushi/toml multi-line string encoding behavior
- **Risk:** The TOML library may encode multi-line prompt strings in unexpected ways (e.g., escaping newlines instead of using `"""` multi-line strings, or adding trailing whitespace).
- **Impact:** Round-trip fidelity could break -- WriteSkill then ReadSkill returns different Content.
- **Mitigation:** Write a focused round-trip test with multi-line content including special characters (quotes, backslashes, newlines). If the default encoder behavior is wrong, use `toml.Marshal` with explicit formatting or fall back to manual TOML string construction for the prompt field.
- **Validation time:** < 10 minutes (write test, run it, inspect output)

### Risk 2: Namespace path traversal security
- **Risk:** A skill name like `../../etc:passwd` could be converted to a path that writes outside baseDir.
- **Impact:** WriteSkill could overwrite arbitrary files on the filesystem.
- **Mitigation:** After converting name to path, resolve the absolute path and verify it is within baseDir using `filepath.Rel()` or prefix checking. Reject names containing `..` segments. Add a test case for this.
- **Validation time:** < 5 minutes (add a test with malicious name, verify rejection)

### Risk 3: Conflict with Claude provider's `Option` type
- **Risk:** Since both providers are in the same package (`internal/provider`), the Claude provider already exports `Option` and `WithBaseDir`. Adding `GeminiOption` and `WithGeminiBaseDir` is fine, but if other devs try to reuse `Option` or `WithBaseDir` for Gemini, it won't compile.
- **Impact:** Confusing API surface within the `provider` package.
- **Mitigation:** Use clearly prefixed names (`GeminiOption`, `WithGeminiBaseDir`). Document in the code that each provider has its own option type. Consider whether a future refactor should move providers to sub-packages (`provider/claude`, `provider/gemini`), but that is out of scope.
- **Validation time:** < 5 minutes (verify code compiles with both providers in same package)

### Risk 4: Gemini CLI format may have undocumented conventions
- **Risk:** Real Gemini command files may use TOML features not covered in the spec (e.g., TOML tables, arrays, additional fields beyond `description` and `prompt`).
- **Impact:** ReadSkill may fail on unexpected TOML structures, or WriteSkill may produce files that Gemini CLI doesn't recognize.
- **Mitigation:** The TOML struct ignores unknown fields by default (BurntSushi/toml only decodes fields mapped to struct fields). WriteSkill only writes known fields. If real files have extra fields, they are silently dropped during round-trip -- this is acceptable for MVP since we only sync the prompt content.
- **Validation time:** < 5 minutes (if user has Gemini CLI installed, inspect `~/.gemini/commands/` for real files)

### Risk 5: go.mod dependency addition requires coordination
- **Risk:** Adding `github.com/BurntSushi/toml` to go.mod changes a shared file. If multiple PRs touch go.mod simultaneously, merge conflicts arise.
- **Impact:** Minor delay resolving conflicts.
- **Mitigation:** Add the dependency in the first task, commit immediately, so it's available for all subsequent work. The dependency is well-established (v1.x stable) so version pinning is straightforward. Run `go mod tidy` to keep go.mod clean.
- **Validation time:** < 2 minutes (`go get github.com/BurntSushi/toml && go mod tidy`)

---

## Recommended API Surface

### Exported functions/types (in `internal/provider/gemini.go`)

| Symbol | Kind | Behavior |
|--------|------|----------|
| `GeminiProvider` | struct | Holds `baseDir` field |
| `GeminiOption` | type | `func(*GeminiProvider)` |
| `WithGeminiBaseDir(dir string)` | func | Returns GeminiOption that sets baseDir |
| `NewGeminiProvider(opts ...GeminiOption)` | func | Constructor; defaults baseDir to `~/.gemini/commands/` |
| `(p *GeminiProvider) Name()` | method | Returns `"gemini"` |
| `(p *GeminiProvider) ListSkills()` | method | WalkDir for `*.toml`, parses each, returns `[]Skill` |
| `(p *GeminiProvider) ReadSkill(name)` | method | Converts `:` to `/`, reads and parses `<path>.toml` |
| `(p *GeminiProvider) WriteSkill(skill)` | method | Converts `:` to `/`, writes TOML to `<path>.toml` |
| `(p *GeminiProvider) SkillDir()` | method | Returns baseDir |
| `init()` | func | Registers "gemini" provider in registry |

### Unexported helpers (in `internal/provider/gemini.go`)

| Symbol | Kind | Behavior |
|--------|------|----------|
| `geminiCommand` | struct | TOML data structure with `Description` and `Prompt` fields |
| `geminiNameToPath(baseDir, name)` | func | Converts `git:commit` -> `<baseDir>/git/commit.toml` |
| `geminiPathToName(baseDir, absPath)` | func | Converts `<baseDir>/git/commit.toml` -> `git:commit` |
| `geminiExtractArgs(prompt)` | func | Finds `{{args}}` in prompt, returns deduplicated list |

---

## Folder Structure

```
internal/provider/
    provider.go      # Provider interface + Skill model  (Provider Architect)
    registry.go      # Register/Get                       (Provider Architect)
    claude.go        # ClaudeProvider                     (Claude Provider Dev)
    claude_test.go   # Claude tests                       (Claude Provider Dev)
    gemini.go        # GeminiProvider                     (ME)
    gemini_test.go   # Gemini tests                       (ME)
```

---

## Tighten the plan into 4-7 small tasks

### Task 1: Add BurntSushi/toml dependency and scaffold GeminiProvider struct
- **Outcome:** `go.mod` includes `github.com/BurntSushi/toml`. `GeminiProvider` struct, `GeminiOption` type, `WithGeminiBaseDir`, `NewGeminiProvider`, `Name()`, and `SkillDir()` all compile. `init()` registers the provider as `"gemini"`.
- **Files to create/modify:** `go.mod`, `go.sum`, `internal/provider/gemini.go`
- **Exact verification:** `go mod tidy && go build ./internal/provider/`
- **Suggested commit message:** `feat(provider): scaffold GeminiProvider with TOML dependency`

### Task 2: Implement ReadSkill with TOML parsing and argument extraction
- **Outcome:** `ReadSkill(name)` reads a `.toml` file, parses `description` and `prompt` fields, extracts `{{args}}` placeholders, populates all Skill fields including SourcePath. Handles `:` to `/` path conversion for namespaced names. Returns error for missing `prompt` field or malformed TOML.
- **Files to create/modify:** `internal/provider/gemini.go`
- **Exact verification:** `go build ./internal/provider/`
- **Suggested commit message:** `feat(provider): implement GeminiProvider.ReadSkill with TOML parsing`

### Task 3: Implement WriteSkill with TOML encoding and namespace support
- **Outcome:** `WriteSkill(skill)` writes a TOML file with `prompt` field (and `description` if non-empty). Converts `:` to `/` in name for path construction. Creates subdirectories as needed. Validates against path traversal.
- **Files to create/modify:** `internal/provider/gemini.go`
- **Exact verification:** `go build ./internal/provider/`
- **Suggested commit message:** `feat(provider): implement GeminiProvider.WriteSkill with TOML encoding`

### Task 4: Implement ListSkills with recursive directory walk
- **Outcome:** `ListSkills()` uses `filepath.WalkDir` to recursively find all `*.toml` files. Converts paths to namespaced names. Parses each file. Returns empty slice for empty directory, error for non-existent directory.
- **Files to create/modify:** `internal/provider/gemini.go`
- **Exact verification:** `go build ./internal/provider/`
- **Suggested commit message:** `feat(provider): implement GeminiProvider.ListSkills with recursive walk`

### Task 5: Write comprehensive unit tests
- **Outcome:** All 21 test cases listed in the Tests section pass. Covers ReadSkill (description, args, namespaced, edge cases), WriteSkill (basic, namespaced, round-trip, creates dir), ListSkills (multiple, empty, subdirs, non-existent), Name/SkillDir.
- **Files to create/modify:** `internal/provider/gemini_test.go`
- **Exact verification:** `go test ./internal/provider/ -v -race -run TestGemini`
- **Suggested commit message:** `test(provider): add comprehensive Gemini provider unit tests`

### Task 6: Path traversal protection and edge case hardening
- **Outcome:** Names containing `..` are rejected by ReadSkill and WriteSkill. Test cases verify that `../../etc:passwd` and similar names return errors. Empty names return errors.
- **Files to create/modify:** `internal/provider/gemini.go`, `internal/provider/gemini_test.go`
- **Exact verification:** `go test ./internal/provider/ -v -race -run TestGemini`
- **Suggested commit message:** `fix(provider): add path traversal protection to GeminiProvider`

---

## CLAUDE.md contributions (proposed content, do NOT write the file)

### From Gemini Provider Dev
- **Coding style:**
  - All errors in `gemini.go` wrapped with `fmt.Errorf("gemini: <context>: %w", err)`
  - Use `filepath` package for all path operations (not `path` or string concat)
  - Use `os.UserHomeDir()` for `~` expansion; never hardcode home directory paths
  - Provider-specific option types: `GeminiOption func(*GeminiProvider)`, not shared `Option`
  - TOML struct tags: use lowercase field names matching Gemini CLI conventions
- **Dev commands:**
  - `go test ./internal/provider/ -v -race -run TestGemini` -- run Gemini provider tests
  - `go mod tidy` -- keep go.mod clean after dependency changes
- **Before you commit checklist:**
  - [ ] `go vet ./internal/provider/`
  - [ ] `go test ./internal/provider/ -race`
  - [ ] All tests pass, no data races
  - [ ] Round-trip test (WriteSkill -> ReadSkill) passes for both flat and namespaced commands
  - [ ] `go mod tidy` produces no changes
- **Guardrails:**
  - Never use `os.UserHomeDir()` in tests -- always use `t.TempDir()`
  - Skill names containing `..` must be rejected to prevent path traversal
  - The `prompt` field is required -- return an error if it's missing or empty
  - TOML encoding must preserve multi-line strings faithfully

---

## EXPLAIN.md contributions (proposed outline bullets)

### Flow / Architecture
- GeminiProvider reads/writes skills from `~/.gemini/commands/*.toml`
- Each `.toml` file = one command; TOML `prompt` field = skill content, `description` = skill description
- Subdirectories create namespaced commands: `git/commit.toml` -> skill name `git:commit`
- Uses `github.com/BurntSushi/toml` for parsing and encoding

### Key Engineering Decisions + Tradeoffs
- BurntSushi/toml chosen over hand-rolled parsing -- correctness for multi-line strings and escaping
- Provider-specific option types -- type safety over DRY, since all providers live in one package
- `filepath.WalkDir` for recursive discovery -- supports arbitrary nesting depth
- `description` field omitted from TOML when empty -- matches real Gemini command conventions

### Limits of MVP + Next Steps
- User-level skills only (`~/.gemini/commands/`); project-level (`<project>/.gemini/commands/`) is future work
- No file watching or caching; reads from disk on every call
- `{{args}}` is the only argument placeholder extracted; `!{shell}` and `@{file}` placeholders are preserved in Content but not parsed into Arguments
- Extra TOML fields beyond `description` and `prompt` are silently dropped on round-trip

### How to Run Locally + How to Validate
- `go test ./internal/provider/ -v -race -run TestGemini` to run all Gemini provider tests
- Create a test command: write a `.toml` file to `~/.gemini/commands/test.toml` with `prompt = "Hello"` and verify parsing
- Test namespacing: create `~/.gemini/commands/git/status.toml` and verify it appears as `git:status`

---

## READY FOR APPROVAL
