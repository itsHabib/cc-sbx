# QE Lead Plan -- End-to-End Test Plan for skill-sync

## You are in PLAN MODE.

### Project
I want to produce a **comprehensive end-to-end test plan** for skill-sync.

**Goal:** Create a practical, developer-runnable test plan document (`docs/validation/content/test-plan.md`) that validates all CLI commands, provider interactions, drift detection, and cross-provider format translation in 15-20 minutes.

### Role + Scope (fill in)
- **Role:** QE Lead
- **Scope:** End-to-end test plan document covering init, sync, status, diff commands across all 4 providers (Claude, Copilot, Gemini, Factory). I own the test plan document only. I do NOT own the smoke test Go code (Smoke Test Dev) or edge case catalog (Edge Case Analyst).
- **File you will write:** `docs/validation/content/test-plan.md`
- **No-touch zones:** Do not edit any other files; do not write code.

---

## Functional Requirements
- **FR1:** Test plan covers all 4 CLI commands: `init`, `sync`, `status`, `diff`
- **FR2:** Test plan covers all 4 providers: Claude Code (`~/.claude/commands/*.md`), Copilot (`.github/prompts/*.prompt.md`), Gemini CLI (`~/.gemini/commands/*.toml`), Factory AI Droid (`.factory/skills/<name>/SKILL.md`)
- **FR3:** Test plan includes setup instructions to build the binary and create isolated test directories
- **FR4:** Test plan covers drift detection scenarios: in-sync, modified, missing-in-target, extra-in-target
- **FR5:** Test plan covers CLI flags: `--dry-run`, `--skill`, `--config`
- **FR6:** Test plan covers cross-provider format translation (Markdown to TOML for Gemini, frontmatter for Factory)
- **FR7:** Test plan includes CI integration guidance using `skill-sync status` exit codes
- **FR8:** Test plan defines clear pass/fail criteria
- Tests required: N/A (this is a test plan document, not code)
- Metrics required: N/A

## Non-Functional Requirements
- Language/runtime: N/A (document, not code)
- Local dev: Test plan assumes the binary is built with `go build -o skill-sync .` from the project root
- Observability: N/A
- Safety: Test plan uses temp directories to avoid touching real provider skill files
- Documentation: The test plan IS the documentation deliverable
- Performance: N/A

---

## Assumptions / System Model
- Deployment environment: Local developer workstation (macOS/Linux); no Docker needed for manual testing
- Failure modes: Test plan covers error cases (missing config, unknown provider, nonexistent skill directory)
- Delivery guarantees: N/A
- Multi-tenancy: N/A

---

## Data Model (as relevant to your role)
The test plan exercises these data structures:

- **Config (`.skill-sync.yaml`)**
  - `source`: string (provider name: "claude", "copilot", "gemini", "factory")
  - `targets`: []string (list of target provider names)
  - `skills`: []string (optional filter list)
  - Validation: source must be a registered provider, must not appear in targets, targets must have at least one entry

- **Skill (internal model)**
  - Name, Description, Content, Arguments, SourcePath
  - Each provider has its own file format mapping (see Provider-specific checks)

- **SkillStatus (drift states)**
  - `in-sync`: source and target content match (after trailing whitespace normalization)
  - `modified`: both exist but content differs
  - `missing-in-target`: source has skill, target does not
  - `extra-in-target`: target has skill not in source

---

## APIs (as relevant to your role)
The test plan validates the CLI command surface:

### CLI Commands
- `skill-sync init --source <provider> --targets <p1,p2,...>` -- creates `.skill-sync.yaml`
- `skill-sync sync [--dry-run] [--skill <name>]` -- syncs skills from source to targets
- `skill-sync status` -- shows drift status per target; exits non-zero on drift
- `skill-sync diff [provider]` -- shows unified diffs for modified skills
- Global flag: `--config <path>` -- override config file path (default `.skill-sync.yaml`)

### Exit Code Contract (critical for CI)
- `status` returns exit code 0 when all targets are in-sync
- `status` returns exit code 1 when any drift is detected
- This is the key mechanism for CI integration (`skill-sync status || exit 1`)

### Error Semantics
- `init` fails if `.skill-sync.yaml` already exists
- `init` fails if source or target names are unknown providers
- `sync`/`status`/`diff` fail if config file not found
- `diff <provider>` fails if provider name is not in the config targets list

---

## Architecture / Component Boundaries (as relevant)
The test plan validates the full stack end-to-end:

- **CLI layer** (`cmd/`): Cobra commands parse flags, load config, resolve providers
- **Config** (`internal/config/`): Loads and validates `.skill-sync.yaml`
- **Provider registry** (`internal/provider/registry.go`): Maps names to provider instances; 4 providers registered via `init()`
- **Providers** (`internal/provider/`): Each implements `ListSkills()`, `ReadSkill()`, `WriteSkill()`, `SkillDir()`
  - Claude: `*.md` in base dir
  - Copilot: `*.prompt.md` in base dir
  - Gemini: `*.toml` (recursive walk, TOML format, `":"` namespace separator)
  - Factory: `<name>/SKILL.md` subdirectories with YAML frontmatter
- **Sync engine** (`internal/sync/engine.go`): Reads source skills, writes to targets with optional skill filter
- **Diff engine** (`internal/sync/diff.go`): Compares source vs target with LCS-based unified diff

Key observation: The default providers use hardcoded base directories (`~/.claude/commands/`, `.github/prompts/`, etc.). For testing, the binary must be pointed at temp directories. Since providers are registered in `init()` with default paths, the test plan must either:
1. Use the actual default directories (risky -- touches real files), OR
2. Seed test files in isolated temp dirs and use `--config` with a config that the providers can find

Looking at the code: providers are registered with default base dirs in `init()` and the registry is global. The `--config` flag only changes the config path, not the provider base directories. This means **manual testing must use the actual provider directories** or we need to accept this as a known limitation and test with files placed in the default locations.

For safety, the test plan will create a dedicated project directory where project-level providers (Copilot: `.github/prompts/`, Factory: `.factory/skills/`) resolve correctly, and for user-level providers (Claude: `~/.claude/commands/`, Gemini: `~/.gemini/commands/`), we will back up existing files before testing and restore after.

---

## Correctness Invariants (must be explicit)
The test plan will verify these invariants:

1. **Init idempotency guard:** `init` refuses to overwrite an existing `.skill-sync.yaml`.
2. **Sync completeness:** After `sync`, every source skill exists in every target (with format translation).
3. **Status accuracy:** `status` correctly classifies every skill as in-sync, modified, missing, or extra.
4. **Diff correctness:** `diff` produces unified diff output only for modified skills; in-sync skills produce no diff.
5. **Dry-run safety:** `sync --dry-run` produces output but writes zero files to disk.
6. **Skill filter:** `sync --skill X` only syncs skill X; other skills are untouched.
7. **Exit code contract:** `status` exits 0 when all in-sync, non-zero when drift detected.
8. **Format translation:** Claude Markdown content synced to Gemini appears in TOML `prompt` field; synced to Factory appears with YAML frontmatter.

---

## Tests
This role produces a **test plan document**, not executable tests. The plan defines manual test scenarios.

The test plan document will contain:
- 12 numbered test scenarios with preconditions, steps, and expected outcomes
- Provider-specific format verification checks
- CI integration recipe
- Pass/fail criteria

Verification that the test plan itself is correct:
- Review against PROJECT.md requirements
- Trace each Definition of Done item to at least one test scenario
- Confirm every CLI command and flag is covered

Commands referenced in the test plan:
- `go build -o skill-sync .` (build the binary)
- `./skill-sync init --source claude --targets copilot,gemini,factory`
- `./skill-sync sync`
- `./skill-sync sync --dry-run`
- `./skill-sync sync --skill deploy`
- `./skill-sync status`
- `./skill-sync diff copilot`
- `cat`, `ls`, `diff` (shell commands for verification)

---

## Benchmarks + "Success"
N/A -- not relevant for a test plan document. The success criterion is: a developer can follow the test plan and complete all scenarios in 15-20 minutes with clear pass/fail for each.

---

## Engineering Decisions & Tradeoffs (REQUIRED)

### Decision 1: Test against real provider directories vs. mocked directories
- **Decision:** Test plan uses real provider directories (with backup/restore) for user-level providers and a dedicated temp project directory for project-level providers.
- **Alternatives considered:** (A) Only test project-level providers in temp dirs, skip user-level. (B) Require code changes to make provider base dirs configurable at runtime.
- **Why:** The binary as built uses hardcoded default base directories registered in `init()`. There is no runtime flag to override provider paths. Testing against real directories is the only way to validate the actual binary end-to-end without code changes.
- **Tradeoff acknowledged:** Risk of accidentally modifying real user skill files. Mitigated by backup/restore steps and using a dedicated test skill name prefix (`_test-`).

### Decision 2: Numbered scenario format vs. BDD-style
- **Decision:** Numbered scenarios with explicit preconditions/steps/expected-outcome.
- **Alternatives considered:** BDD Gherkin-style (Given/When/Then).
- **Why:** Numbered format is more compact, easier for a developer to run through sequentially, and doesn't require familiarity with BDD conventions.
- **Tradeoff acknowledged:** Less formal than Gherkin; harder to auto-generate from. Acceptable since this is a manual test plan.

### Decision 3: Scope limited to happy path + key error paths
- **Decision:** Cover the primary happy path for each command plus 2-3 key error scenarios. Defer exhaustive edge cases to the Edge Case Analyst.
- **Alternatives considered:** Comprehensive edge case coverage in the test plan itself.
- **Why:** Keeps the test plan runnable in 15-20 minutes. Edge cases are cataloged separately by the Edge Case Analyst role.
- **Tradeoff acknowledged:** Some edge cases won't be validated by this plan. The edge case document fills that gap.

---

## Risks & Mitigations (REQUIRED)

### Risk 1: Provider base directories are hardcoded
- **Risk:** The CLI binary uses hardcoded paths (`~/.claude/commands/`, `~/.gemini/commands/`, `.github/prompts/`, `.factory/skills/`). Cannot redirect to temp dirs without code changes.
- **Impact:** Testing user-level providers (Claude, Gemini) touches real directories, risking data loss.
- **Mitigation:** Test plan includes explicit backup/restore steps. Use `_test-*` prefixed skill names that won't collide with real skills. Clean up after each scenario.
- **Validation time:** 5 minutes (verify backup/restore works on one provider).

### Risk 2: Gemini provider requires BurntSushi/toml dependency
- **Risk:** If `go build` fails due to missing dependency, no binary to test.
- **Impact:** Blocked -- cannot run any test scenarios.
- **Mitigation:** Test plan starts with build verification. `go mod tidy && go build -o skill-sync .` is step 1.
- **Validation time:** 2 minutes.

### Risk 3: Cross-provider format translation may lose data
- **Risk:** Syncing from Claude (Markdown) to Gemini (TOML) wraps content in `prompt` field. The Gemini provider stores the full `skill.Content` as the TOML `prompt` value, including the `# description` first line. When read back, the content comparison normalizes trailing whitespace but the TOML encoding may introduce differences.
- **Impact:** Status may report false "modified" for skills that were just synced.
- **Mitigation:** Test scenario explicitly verifies re-sync produces "in-sync" status. If it doesn't, this is a bug to report, not a test plan failure.
- **Validation time:** 5 minutes.

### Risk 4: Test plan may not cover all provider directory structures
- **Risk:** Factory uses `<name>/SKILL.md` subdirectory pattern; Gemini uses recursive walk with `:` namespace separator. These patterns are different from flat-file providers.
- **Impact:** Test plan may miss structural validation for nested providers.
- **Mitigation:** Include dedicated provider-specific verification steps that check exact file paths and directory structures after sync.
- **Validation time:** 3 minutes.

---

## Recommended API Surface
N/A -- this role produces a document, not code.

## Folder Structure

```
docs/validation/
  content/
    test-plan.md        # <-- primary deliverable (written by QE Lead)
  plans/
    qe-lead.md          # <-- this plan document
```

## Step-by-Step Task Plan

### Task 1: Write test plan setup section
- **Outcome:** `docs/validation/content/test-plan.md` created with build instructions, directory setup, and backup/restore guidance
- **Files to create/modify:** `docs/validation/content/test-plan.md`
- **Verification:** Review setup steps; confirm `go build` command is correct; confirm backup commands are safe
- **Commit message:** `docs: add test plan setup and prerequisites`

### Task 2: Write core test scenarios (init, sync, status, diff)
- **Outcome:** 8 numbered test scenarios covering happy path for all 4 commands plus dry-run, skill filter, and re-sync
- **Files to create/modify:** `docs/validation/content/test-plan.md` (append)
- **Verification:** Trace each scenario to a PROJECT.md Definition of Done item; confirm all commands covered
- **Commit message:** `docs: add core test scenarios for all CLI commands`

### Task 3: Write provider-specific verification checks
- **Outcome:** Per-provider verification table: correct directory, file extension, format (Markdown/TOML/frontmatter), metadata mapping
- **Files to create/modify:** `docs/validation/content/test-plan.md` (append)
- **Verification:** Cross-reference with PROJECT.md provider format reference; confirm all 4 providers covered
- **Commit message:** `docs: add provider-specific verification checks to test plan`

### Task 4: Write drift detection and error scenarios
- **Outcome:** 4 additional scenarios: drift detection (modify target, verify status/diff), extra-in-target, error cases (missing config, unknown provider)
- **Files to create/modify:** `docs/validation/content/test-plan.md` (append)
- **Verification:** Confirm all 4 SkillStatus values are tested; confirm error messages match code
- **Commit message:** `docs: add drift detection and error test scenarios`

### Task 5: Write CI integration section and pass/fail criteria
- **Outcome:** CI recipe using `skill-sync status` exit codes; clear pass/fail table for the full test plan
- **Files to create/modify:** `docs/validation/content/test-plan.md` (append)
- **Verification:** Confirm exit code behavior matches `cmd/status.go` implementation (returns error on drift)
- **Commit message:** `docs: add CI integration and pass/fail criteria to test plan`

### Task 6: Write cleanup section and final review
- **Outcome:** Cleanup instructions to remove test files and restore backups; final review checklist
- **Files to create/modify:** `docs/validation/content/test-plan.md` (append)
- **Verification:** Full read-through of complete document; confirm 15-20 minute completion time estimate
- **Commit message:** `docs: finalize test plan with cleanup and review checklist`

---

## CLAUDE.md contributions (do NOT write the file; propose content)

## From QE Lead
- **Dev commands:**
  - `go build -o skill-sync .` -- build the CLI binary
  - `./skill-sync init --source claude --targets copilot,gemini,factory` -- initialize config
  - `./skill-sync sync` -- sync all skills
  - `./skill-sync status` -- check drift (exits non-zero on drift)
  - `./skill-sync diff <provider>` -- show diffs for a specific target
- **Before you commit checklist:**
  - Run `go test ./...` -- all unit tests pass
  - Run `go vet ./...` -- no vet warnings
  - If you changed a provider: verify the provider's test file covers your changes
  - If you changed CLI flags: update the test plan (`docs/validation/content/test-plan.md`)
- **Guardrails:**
  - Do not change provider `init()` registration without updating all test fixtures
  - Do not change `status` exit code behavior -- CI integrations depend on non-zero exit on drift
  - Do not change the config schema (`source`, `targets`, `skills`) without updating the test plan

---

## EXPLAIN.md contributions (do NOT write the file; propose outline bullets)

### Flow / Architecture
- CLI parses flags -> loads `.skill-sync.yaml` -> resolves providers from global registry -> delegates to sync/diff engine
- Providers are registered in `init()` with hardcoded default base directories
- Sync engine reads from source provider, writes to each target provider
- Diff engine compares source vs target content (normalized trailing whitespace), produces LCS-based unified diffs

### Key Engineering Decisions + Tradeoffs
- Global provider registry with `init()` registration: simple but makes base dirs non-configurable at runtime
- Content comparison uses trailing whitespace normalization: prevents false drifts from line ending differences
- Gemini TOML format requires serialize/deserialize: content is stored in `prompt` field, description in `description` field
- Factory frontmatter parsing handles both with-frontmatter and without-frontmatter cases

### Limits of MVP + Next Steps
- No runtime override for provider base directories (hardcoded in `init()`)
- No argument/placeholder translation between providers (content synced verbatim)
- User-level providers only for Claude/Gemini; project-level only for Copilot/Factory
- No watch mode or auto-sync
- Next: configurable base dirs, argument translation, bidirectional drift resolution

### How to Run Locally + How to Validate
- `go build -o skill-sync . && ./skill-sync init --source claude --targets copilot`
- Follow `docs/validation/content/test-plan.md` for full validation walkthrough
- Run `go test ./...` for unit test coverage

---

## READY FOR APPROVAL
