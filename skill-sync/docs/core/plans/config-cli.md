# Master Plan: Config & CLI Foundation

> Phase: core | Role: Config & CLI Foundation

---

## You are in PLAN MODE.

### Project
I want to build **skill-sync**, a Go CLI that syncs AI assistant skills from a primary provider to all others with drift detection.

**Goal:** build the **Config & CLI Foundation** layer -- Go module setup, cobra CLI skeleton, config parsing, and `skill-sync init` command -- so that all other components have a runnable CLI frame and config loading to build on.

### Role + Scope
- **Role:** Config & CLI Foundation
- **Scope:** Go module (`go.mod`), `main.go` entrypoint, `cmd/root.go` (root command + global flags + config loading), `cmd/init.go` (`skill-sync init` command), `internal/config/config.go` (Config struct, Load, Validate). I do NOT own the provider interface, provider implementations, sync engine, or diff engine.
- **File I will write:** `/docs/core/plans/config-cli.md`
- **No-touch zones:** do not edit any other files; do not write code.

---

## Functional Requirements
- **FR1:** `go.mod` declares module `github.com/user/skill-sync`, Go 1.22, with cobra and yaml.v3 dependencies.
- **FR2:** `main.go` is a thin entrypoint that calls `cmd.Execute()`.
- **FR3:** `cmd/root.go` defines the root cobra command "skill-sync" with a `--config` flag (default `.skill-sync.yaml`), a `PersistentPreRunE` that loads config from the flag path and stores it for subcommands.
- **FR4:** `cmd/init.go` implements `skill-sync init` which accepts `--source` and `--targets` flags, validates provider names against the registry, and writes a `.skill-sync.yaml` file to the current directory.
- **FR5:** `internal/config/config.go` provides a `Config` struct (Source, Targets, Skills), `Load(path) (*Config, error)` for YAML parsing, and `Validate(registryNames []string) error` for checking that source/target names exist.
- **Tests required:** unit tests for config Load (valid YAML, invalid YAML, missing file) and Validate (valid names, unknown provider names, source in targets).

## Non-Functional Requirements
- Language/runtime: Go 1.22+
- Local dev: `go build`, `go test ./...`, `go vet ./...`
- Observability: N/A for this layer (CLI config parsing)
- Safety: graceful error messages if config file is missing or malformed; never panic
- Documentation: CLAUDE.md + EXPLAIN.md contributions proposed (not written)
- Performance: N/A -- config parsing is not a hot path

---

## Assumptions / System Model
- Deployment environment: local CLI binary, no containers
- Failure modes: config file not found (clear error), malformed YAML (wrapped parse error), unknown provider name in config (validation error listing valid providers)
- Delivery guarantees: N/A (local CLI, no network)
- Multi-tenancy: none

---

## Data Model

### Config struct
```go
type Config struct {
    Source  string   `yaml:"source"`
    Targets []string `yaml:"targets"`
    Skills  []string `yaml:"skills"`
}
```

**Validation rules:**
- `Source` must be non-empty and exist in the provider registry
- `Targets` must have at least one entry; each must exist in the registry
- `Source` must not appear in `Targets`
- `Skills` is optional; empty means "sync all"

**Versioning:** N/A for MVP -- config format is simple enough that versioning is not needed yet.

**Persistence:** config is a YAML file on disk (`.skill-sync.yaml`), read at startup, written by `init` command.

---

## APIs

### Config package API
```go
// Load reads and parses a .skill-sync.yaml file.
func Load(path string) (*Config, error)

// Validate checks that source and target names exist in the provided
// list of registered provider names.
func (c *Config) Validate(registeredNames []string) error
```

**Error semantics:**
- `Load` returns a wrapped `os.ErrNotExist` if the file is missing, or a wrapped yaml parse error if malformed.
- `Validate` returns an error listing all invalid provider names found.

### CLI commands

**Root command (`skill-sync`):**
- `--config string` (default `.skill-sync.yaml`) -- path to config file
- `PersistentPreRunE` loads config via `config.Load()`, stores result in a package-level `var Cfg *config.Config`
- Subcommands that don't need config (like `init`) set `DisableFlagParsing` or skip config loading via annotation

**Init command (`skill-sync init`):**
- `--source string` (required) -- source provider name
- `--targets strings` (required) -- comma-separated target provider names
- Behavior: validates names against registry, writes `.skill-sync.yaml`, prints confirmation
- If `.skill-sync.yaml` already exists, prints error and exits (no overwrite without `--force`)

---

## Architecture / Component Boundaries

```
main.go
  -> cmd.Execute()
       -> cmd/root.go: rootCmd (PersistentPreRunE loads config)
            -> cmd/init.go: initCmd (writes .skill-sync.yaml)
       -> (future) cmd/sync.go, cmd/status.go, cmd/diff.go

cmd/root.go uses:
  -> internal/config.Load()
  -> internal/config.Config.Validate()
  -> internal/provider.List() (for validation)

cmd/init.go uses:
  -> internal/provider.List() (to validate chosen names)
  -> internal/config.Config (to marshal to YAML)
  -> gopkg.in/yaml.v3 (to write .skill-sync.yaml)
```

The config package has NO dependency on the provider package -- it accepts a `[]string` of valid names rather than importing the registry directly. This keeps the dependency graph clean and makes testing easier.

---

## Correctness Invariants
1. **Config round-trip:** `Load(path)` on a file written by `init` must produce a Config equal to the original.
2. **Validation rejects unknowns:** `Validate()` must return an error if any source or target name is not in the provided list.
3. **Source not in targets:** `Validate()` must reject configs where source appears in the targets list.
4. **Init never overwrites:** `init` must not silently overwrite an existing `.skill-sync.yaml`.
5. **PersistentPreRunE skipped for init:** the `init` command must not require a pre-existing config file.

---

## Tests

### Unit tests: `internal/config/config_test.go`
- **TestLoad_ValidConfig:** write a valid YAML to t.TempDir(), Load it, assert fields match
- **TestLoad_MissingFile:** Load a nonexistent path, assert error wraps os.ErrNotExist
- **TestLoad_MalformedYAML:** write invalid YAML, Load it, assert error is returned
- **TestLoad_EmptySkills:** write config with `skills: []`, assert Skills is empty slice
- **TestValidate_ValidNames:** validate with correct registry names, assert no error
- **TestValidate_UnknownSource:** validate with source not in registry, assert error mentions the name
- **TestValidate_UnknownTarget:** validate with unknown target, assert error
- **TestValidate_SourceInTargets:** validate with source also in targets, assert error

### Unit tests: `cmd/init_test.go` (or integration-style)
- **TestInitWritesConfig:** run init with valid flags, assert .skill-sync.yaml created with correct content
- **TestInitRejectsUnknownProvider:** run init with invalid source name, assert error
- **TestInitNoOverwrite:** create existing config, run init, assert error

### Commands
```bash
go test ./internal/config/...
go test ./cmd/...
go vet ./...
```

---

## Benchmarks + "Success"
N/A -- config parsing and CLI init are not performance-sensitive. Success is defined by all tests passing and the `init` command producing a valid, loadable config file.

---

## Engineering Decisions & Tradeoffs (REQUIRED)

### Decision 1: Package-level var for config vs context passing
- **Decision:** Store loaded config in a package-level `var Cfg *config.Config` in `cmd/root.go`, set during `PersistentPreRunE`.
- **Alternatives considered:** Pass config through cobra command context (`cmd.SetContext()`); pass config as constructor arg to each subcommand.
- **Why:** Package-level var is the simplest approach and is the idiomatic cobra pattern. Context-based passing adds complexity without benefit for a single-binary CLI.
- **Tradeoff acknowledged:** Package-level state makes unit testing subcommands slightly harder (must set/reset the var). Acceptable for a CLI tool.

### Decision 2: Config.Validate takes []string, not the registry directly
- **Decision:** `Validate(registeredNames []string)` rather than `Validate(registry *provider.Registry)`.
- **Alternatives considered:** Import provider package and call registry.Get() directly inside Validate.
- **Why:** Avoids a circular or unnecessary dependency from config -> provider. Makes config package independently testable with plain string slices.
- **Tradeoff acknowledged:** Caller must extract names from registry before calling Validate -- one extra step, but keeps packages decoupled.

### Decision 3: Flags-based init (not interactive prompts)
- **Decision:** `skill-sync init --source claude --targets copilot,gemini` using cobra flags.
- **Alternatives considered:** Interactive terminal prompts (e.g., using bubbletea or survey).
- **Why:** Flags are scriptable, testable, and require no additional dependencies. Interactive prompts can be added later if desired.
- **Tradeoff acknowledged:** Less user-friendly for first-time users who may not know provider names. Mitigated by listing valid providers in `--help` text and error messages.

---

## Risks & Mitigations (REQUIRED)

### Risk 1: PersistentPreRunE runs for init command (which doesn't need existing config)
- **Risk:** `init` fails because no `.skill-sync.yaml` exists yet, but PersistentPreRunE tries to load it.
- **Impact:** `skill-sync init` would be unusable on first run -- a showstopper.
- **Mitigation:** Use cobra annotations or check the command name in PersistentPreRunE to skip config loading for `init`. Alternatively, make PersistentPreRunE tolerate missing config and let subcommands that need it check explicitly.
- **Validation time:** < 5 minutes (write a test that runs init without config file).

### Risk 2: Provider registry not available when init validates names
- **Risk:** If providers register via `init()` functions and the registry lives in `internal/provider`, the `cmd` package must import `internal/provider` to trigger init registration. If this import is missing, no providers are registered and init always fails validation.
- **Impact:** Init rejects all provider names as unknown.
- **Mitigation:** Ensure `cmd/root.go` has a blank import `_ "github.com/user/skill-sync/internal/provider"` or the provider package is imported transitively. Test this in CI.
- **Validation time:** < 5 minutes (run `skill-sync init --source claude --targets copilot` and check output).

### Risk 3: YAML serialization of empty Skills slice
- **Risk:** `yaml.Marshal` may omit `skills: []` or serialize it as `skills: null`, leading to ambiguity on reload.
- **Impact:** Config round-trip inconsistency; user confusion about whether skills filter is active.
- **Mitigation:** Use `yaml:"skills,flow"` tag or explicitly set `Skills: []string{}` (not nil). Add a round-trip test.
- **Validation time:** < 5 minutes (TestLoad_EmptySkills covers this).

### Risk 4: Cobra/yaml.v3 version compatibility with Go 1.22
- **Risk:** Latest cobra or yaml.v3 might require Go 1.23+.
- **Impact:** Build failure.
- **Mitigation:** Pin specific versions known to support Go 1.22 in go.mod. `cobra v1.8.x` and `yaml.v3 v3.0.1` both support Go 1.22.
- **Validation time:** < 2 minutes (`go build` after go.mod setup).

---

# Recommended API Surface

### `internal/config` package
| Function | Behavior |
|----------|----------|
| `Load(path string) (*Config, error)` | Read YAML file, unmarshal into Config. Wraps os/yaml errors. |
| `(c *Config) Validate(names []string) error` | Checks Source and each Target exist in names. Checks Source not in Targets. Returns descriptive error. |

### `cmd` package
| Function | Behavior |
|----------|----------|
| `Execute()` | Calls `rootCmd.Execute()`. Entry point from main. |
| `rootCmd` (internal) | Root cobra command. `--config` flag. PersistentPreRunE loads config (skipped for init). |
| `initCmd` (internal) | `--source`, `--targets` flags. Validates against registry. Writes `.skill-sync.yaml`. |

---

# Folder Structure

```
skill-sync/
├── main.go                      # cmd.Execute() -- owned by this role
├── go.mod                       # module + deps -- owned by this role
├── go.sum                       # auto-generated
├── cmd/
│   ├── root.go                  # root command + config loading -- owned by this role
│   └── init.go                  # init command -- owned by this role
├── internal/
│   ├── config/
│   │   ├── config.go            # Config struct, Load, Validate -- owned by this role
│   │   └── config_test.go       # unit tests -- owned by this role
│   └── provider/                # NOT owned by this role
│       ├── provider.go          # Provider Architect
│       ├── registry.go          # Provider Architect
│       ├── claude.go            # Claude Provider Dev
│       └── claude_test.go       # Claude Provider Dev
```

---

# Step-by-step Task Plan (4-7 Small Tasks)

## Task 1: Go module + main.go entrypoint
- **Outcome:** `go build` succeeds, produces a `skill-sync` binary that prints cobra help.
- **Files to create:** `go.mod`, `main.go`, `cmd/root.go`
- **Verification:**
  ```bash
  cd skill-sync && go build -o skill-sync .
  ./skill-sync --help  # should print help text
  go vet ./...
  ```
- **Commit message:** `feat: add go module, main.go, and root cobra command`

## Task 2: Config struct and Load function
- **Outcome:** Config YAML can be parsed and loaded; unit tests pass.
- **Files to create:** `internal/config/config.go`, `internal/config/config_test.go`
- **Verification:**
  ```bash
  go test ./internal/config/... -v
  ```
- **Commit message:** `feat: add config parsing with Load and YAML support`

## Task 3: Config Validate function
- **Outcome:** Config validation checks source/target names against a provided list; rejects source-in-targets. Unit tests pass.
- **Files to modify:** `internal/config/config.go`, `internal/config/config_test.go`
- **Verification:**
  ```bash
  go test ./internal/config/... -v -run TestValidate
  ```
- **Commit message:** `feat: add config validation against provider names`

## Task 4: PersistentPreRunE config loading in root command
- **Outcome:** Root command loads config from `--config` flag path in PersistentPreRunE. Init command is excluded from config loading.
- **Files to modify:** `cmd/root.go`
- **Verification:**
  ```bash
  go build -o skill-sync . && ./skill-sync --help  # no error
  # (init command won't fail on missing config)
  ```
- **Commit message:** `feat: wire config loading into root command PersistentPreRunE`

## Task 5: Init command
- **Outcome:** `skill-sync init --source claude --targets copilot,gemini` writes a valid `.skill-sync.yaml`. Rejects unknown providers. Refuses to overwrite existing config.
- **Files to create:** `cmd/init.go`
- **Verification:**
  ```bash
  go build -o skill-sync .
  ./skill-sync init --source claude --targets copilot,gemini
  cat .skill-sync.yaml  # verify content
  ./skill-sync init --source claude --targets copilot  # should error (already exists)
  go test ./cmd/... -v
  ```
- **Commit message:** `feat: add skill-sync init command with config generation`

## Task 6: Full integration verification
- **Outcome:** All tests pass, vet is clean, binary runs end-to-end.
- **Files to modify:** none (verification only)
- **Verification:**
  ```bash
  go test ./... -v
  go vet ./...
  go build -o skill-sync .
  ```
- **Commit message:** N/A (no code change, just verification gate)

---

# Benchmark Plan + "Success"
N/A -- config parsing and CLI scaffolding are not performance-sensitive. Success = all tests pass, `go vet` clean, `skill-sync init` produces a loadable config, `skill-sync --help` prints usage.

---

# CLAUDE.md Contributions (do NOT write the file; propose content)

## From Config & CLI Foundation
### Coding style
- Use `cobra.Command` for all CLI commands; register subcommands in `init()` functions
- Config is loaded once in `PersistentPreRunE` and stored in `cmd.Cfg`
- Commands that don't need config (e.g., `init`) should be annotated to skip config loading
- All errors wrapped with `fmt.Errorf("context: %w", err)`

### Dev commands
```bash
go build -o skill-sync .       # build
go test ./... -v               # run all tests
go vet ./...                   # lint
./skill-sync init --source claude --targets copilot,gemini  # bootstrap config
```

### Before you commit
- [ ] `go test ./...` passes
- [ ] `go vet ./...` is clean
- [ ] `go build` succeeds
- [ ] If you changed config format, update the init command to match

### Guardrails
- Never load config in `init` command -- it creates the config, it can't depend on one existing
- Config package must not import provider package -- pass `[]string` of names instead
- Default config path is `.skill-sync.yaml` (relative to cwd)

---

# EXPLAIN.md Contributions (do NOT write the file; propose outline bullets)

### Flow / architecture
- `main.go` -> `cmd.Execute()` -> cobra dispatch
- Root command's `PersistentPreRunE` loads `.skill-sync.yaml` via `config.Load()`
- `init` subcommand bypasses config loading, generates the config file from flags
- Config is a simple struct: source provider name, target provider names, optional skill filter

### Key engineering decisions + tradeoffs
- Package-level config var (simple but slightly harder to test) over context passing
- Config.Validate takes `[]string` not registry import (keeps packages decoupled)
- Flags-based init over interactive prompts (scriptable, no extra deps)

### Limits of MVP + next steps
- No interactive mode for `init` (flags only)
- No config migration/versioning
- No `--force` flag for init overwrite (future enhancement)
- Project-level config (`.skill-sync.yaml` in project root vs home dir) not yet supported

### How to run locally + validate
```bash
go build -o skill-sync .
./skill-sync init --source claude --targets copilot,gemini
cat .skill-sync.yaml
./skill-sync --help
go test ./... -v
```

---

## READY FOR APPROVAL
