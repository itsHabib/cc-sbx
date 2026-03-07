# Master Plan: README Author

## You are in PLAN MODE.

### Project
I want to polish **skill-sync** for open-source release.

**Goal:** write a **README.md** that makes developers want to use skill-sync immediately -- clear problem statement, 60-second quickstart, realistic terminal output, and an architecture diagram.

### Role + Scope
- **Role:** README Author
- **Scope:** Owns `README.md` at the project root. Responsible for the full README content: tagline, problem statement, quickstart, usage examples with realistic output, mermaid architecture diagram, provider table, configuration reference, "how it works" explanation, and contributing section.
- **No-touch zones:** All Go source code, all other documentation files, CLI help text, docstrings. Does not modify any file except `README.md`.
- **File you will write:** `/docs/polish/plans/readme-author.md`
- **No-touch zones:** do not edit any other files; do not write code.

---

## Functional Requirements
- FR1: README must have a one-liner + tagline that communicates what skill-sync does in one sentence
- FR2: README must include a "Quick Start" section that gets a developer from zero to synced in under 5 commands
- FR3: README must show realistic terminal output for every command (`init`, `sync`, `status`, `diff`) including table formatting and status symbols
- FR4: README must include a mermaid architecture diagram showing source provider -> Skill model (IR) -> target providers
- FR5: README must include a provider comparison table (name, skill location, format, file extension)
- FR6: README must include the `.skill-sync.yaml` configuration format with all options documented
- FR7: README must include a CI integration example (GitHub Actions) using `status` as a drift gate
- Tests required: N/A (documentation only -- verification is human review)
- Metrics required: N/A

## Non-Functional Requirements
- Language/runtime: Markdown (GitHub-flavored)
- Tone: developer-facing, practical, no marketing fluff
- Show, don't tell: use real examples and terminal output
- The "wow this is useful" moment should happen within the first 30 seconds of reading
- No badges (user can add later), no emojis
- Mermaid diagram must render correctly on GitHub

---

## Assumptions / System Model
- Deployment environment: N/A (documentation)
- The existing README.md at the project root will be fully replaced
- All terminal output examples are based on the actual CLI behavior observed in the source code (tabwriter formatting, status symbols, exit codes)
- Provider names are: `claude`, `copilot`, `gemini`, `factory`
- The install path `github.com/user/skill-sync` is a placeholder that the user will update

---

## Data Model

N/A -- not in scope for this role. This is a documentation-only deliverable.

---

## APIs

N/A -- not in scope for this role. The README documents the CLI surface but does not define or modify it.

---

## Architecture / Component Boundaries

The README will include a mermaid diagram documenting the following architecture:

```
Source Provider (e.g., Claude Code)
    |
    v
  ListSkills() / ReadSkill()
    |
    v
  Skill (normalized intermediate representation)
    |-- Name
    |-- Description
    |-- Content
    |-- Arguments
    |
    +---> WriteSkill() ---> Target: Copilot (.prompt.md)
    +---> WriteSkill() ---> Target: Gemini (.toml)
    +---> WriteSkill() ---> Target: Factory (SKILL.md + frontmatter)
```

The diagram will also show the DiffEngine path for status/drift detection.

---

## Correctness Invariants

- Every terminal output example must match what the actual CLI produces (verified against `cmd/sync.go`, `cmd/status.go`, `cmd/diff.go` tabwriter output)
- The provider table must match the actual provider implementations (skill directories, file extensions, format)
- The config YAML example must match the `config.Config` struct fields: `source`, `targets`, `skills`
- The exit code table must match actual behavior: `status` exits 1 on drift, `sync` exits 1 on errors
- All command flags documented must exist in the source code (`--config`, `--source`, `--targets`, `--dry-run`, `--skill`)

---

## Tests

Verification is manual review against source code:

1. **Provider paths check:** Compare provider table entries against `claude.go:33`, `copilot.go:29`, `gemini.go:44`, `factory.go:39` default base directories
2. **CLI output check:** Compare example outputs against tabwriter format strings in `cmd/sync.go:68,91`, `cmd/status.go:45-49`, `cmd/diff.go:46`
3. **Config format check:** Compare YAML example against `config.Config` struct in `config.go:12-16`
4. **Exit code check:** Verify exit code semantics against `cmd/status.go:59-61` (drift -> error), `cmd/sync.go:102-103` (errors -> error)
5. **Mermaid render check:** Paste diagram into GitHub or mermaid.live to confirm it renders

Commands:
- `go build -o skill-sync .` -- verify build succeeds (README install instructions must work)

---

## Benchmarks + "Success"

N/A -- this is a documentation deliverable. Success is measured by:
- A developer can go from reading the README to having skill-sync running in under 60 seconds
- All example output matches actual CLI behavior
- The mermaid diagram renders correctly on GitHub
- The README structure flows logically: problem -> solution -> quickstart -> detailed usage -> internals

---

## Engineering Decisions & Tradeoffs

### Decision 1: Full README rewrite vs. incremental edit
- **Decision:** Full rewrite of the existing README.md
- **Alternatives considered:** Incremental additions to the existing README
- **Why:** The existing README already covers most sections but was written during Phase 2 without the full context of Phase 3 validation work. A coherent rewrite ensures consistent tone, accurate examples, and the addition of the architecture diagram and "how it works" section. The existing README is a solid foundation and most content will be preserved and refined.
- **Tradeoff acknowledged:** Risk of losing any user-made edits to the current README. Mitigated by the fact that this is a controlled phase -- we own the entire README.

### Decision 2: Mermaid diagram vs. ASCII art vs. image
- **Decision:** Mermaid diagram (```mermaid code block)
- **Alternatives considered:** ASCII art diagram, PNG/SVG image file
- **Why:** Mermaid renders natively on GitHub, is version-controllable as text, and is easy to update when the architecture changes. ASCII art looks poor on narrow screens. Image files require separate tooling and are harder to keep in sync.
- **Tradeoff acknowledged:** Mermaid rendering depends on the viewer (GitHub renders it, some editors do not). Users viewing the raw markdown will see the mermaid source, not a diagram.

### Decision 3: Realistic terminal output vs. simplified examples
- **Decision:** Use realistic terminal output that matches actual tabwriter formatting
- **Alternatives considered:** Simplified/prettified output that doesn't match the real CLI
- **Why:** Developers trust documentation more when the output matches what they see. Using realistic output also serves as implicit documentation of the CLI output format.
- **Tradeoff acknowledged:** If the CLI output format changes, the README examples become stale. Mitigated by the CLAUDE.md contribution below (guardrail: update README when CLI output changes).

---

## Risks & Mitigations

### Risk 1: Terminal output examples don't match actual CLI
- **Risk:** Example output in the README diverges from what the CLI actually produces
- **Impact:** Developers lose trust in the documentation; confusion during onboarding
- **Mitigation:** Derive all example output from reading the actual `cmd/*.go` source code, specifically the tabwriter format strings and status symbols. Cross-reference with the test plan's expected output (docs/validation/content/test-plan.md)
- **Validation time:** 5 minutes (read source, write matching output)

### Risk 2: Mermaid diagram doesn't render on GitHub
- **Risk:** Syntax error in mermaid code block causes a raw text dump instead of a diagram
- **Impact:** Architecture section looks broken; poor first impression
- **Mitigation:** Use simple mermaid graph syntax (flowchart LR); validate on mermaid.live before committing
- **Validation time:** 3 minutes

### Risk 3: Install path is a placeholder
- **Risk:** The `go install github.com/user/skill-sync@latest` path is not a real importable module
- **Impact:** Copy-paste install command fails for users
- **Mitigation:** Document both `go install` (with a note that the path should be updated) and "build from source" as the primary install method. The build-from-source path always works.
- **Validation time:** 2 minutes

### Risk 4: README becomes stale after future CLI changes
- **Risk:** Future phases add commands, flags, or change output format without updating README
- **Impact:** Documentation drift (ironic for a drift-detection tool)
- **Mitigation:** Add a guardrail in CLAUDE.md contributions: "Update README.md when CLI output format or flags change"
- **Validation time:** 1 minute (add the guardrail)

---

## Recommended API Surface

N/A -- this role produces documentation, not code. The README documents the existing CLI surface:

| Command | Arguments | Flags | Behavior |
|---------|-----------|-------|----------|
| `init` | (none) | `--source`, `--targets`, `--config` | Creates `.skill-sync.yaml` |
| `sync` | (none) | `--dry-run`, `--skill` (repeatable), `--config`, `--source`, `--targets` | Syncs skills from source to targets |
| `status` | (none) | `--config`, `--source`, `--targets` | Reports drift; exits 1 if drift found |
| `diff` | `[provider]` (optional) | `--config`, `--source`, `--targets` | Shows unified diffs for modified skills |

---

## Folder Structure

No new folders or packages. Single file deliverable:

```
skill-sync/
  README.md          <-- REPLACED (full rewrite)
```

---

## Step-by-Step Task Plan

### Tighten the plan into 4-7 small tasks

#### Task 1: Write header + problem statement + quickstart
- **Outcome:** README has a compelling one-liner, 3-4 sentence problem statement, and a quickstart section with <5 commands that gets a developer from zero to synced
- **Files to create/modify:** `skill-sync/README.md`
- **Exact verification:** Read the first 40 lines; confirm one-liner is one sentence, problem is 3-4 sentences, quickstart has `init` + `sync` in under 5 commands
- **Suggested commit message:** `docs: README header, problem statement, and quickstart`

#### Task 2: Write usage examples with realistic terminal output
- **Outcome:** Each command (`init`, `sync`, `sync --dry-run`, `sync --skill`, `status`, `diff`) has a code block showing the command and its realistic terminal output matching actual tabwriter formatting
- **Files to create/modify:** `skill-sync/README.md`
- **Exact verification:** Compare each example's output format against source: `cmd/sync.go:68` (dry-run header), `cmd/sync.go:91` (sync header), `cmd/status.go:45-49` (status symbols), `cmd/diff.go:46` (diff output)
- **Suggested commit message:** `docs: README usage examples with realistic terminal output`

#### Task 3: Write mermaid architecture diagram
- **Outcome:** A mermaid flowchart showing: Source Provider -> Skill (IR) -> Target Providers, plus DiffEngine comparison path. Renders correctly on GitHub.
- **Files to create/modify:** `skill-sync/README.md`
- **Exact verification:** Paste mermaid block into mermaid.live; confirm it renders a readable diagram with correct labels
- **Suggested commit message:** `docs: README architecture diagram`

#### Task 4: Write provider table + configuration reference
- **Outcome:** A table with 4 providers showing name, skill location, format, file extension. A YAML config example showing all fields (`source`, `targets`, `skills`) with inline comments.
- **Files to create/modify:** `skill-sync/README.md`
- **Exact verification:** Compare table against provider defaults in source: `claude.go:33` (`~/.claude/commands/`), `copilot.go:29` (`.github/prompts/`), `gemini.go:44` (`~/.gemini/commands/`), `factory.go:39` (`.factory/skills/`)
- **Suggested commit message:** `docs: README provider table and configuration reference`

#### Task 5: Write "how it works" + CI integration + contributing + development
- **Outcome:** Brief explanation of sync/status/diff flow, GitHub Actions YAML for CI drift gate, contributing section, development commands (`go build`, `go test`, `go vet`)
- **Files to create/modify:** `skill-sync/README.md`
- **Exact verification:** Verify CI YAML uses `skill-sync status` with correct exit code semantics per `cmd/status.go:59-61`. Verify dev commands match `go.mod` module path.
- **Suggested commit message:** `docs: README how-it-works, CI integration, contributing, dev`

#### Task 6: Final review pass -- accuracy + flow
- **Outcome:** All sections reviewed for accuracy against source code, consistent tone, logical flow, no stale references
- **Files to create/modify:** `skill-sync/README.md` (minor edits only)
- **Exact verification:** `go build -o skill-sync . && echo "build OK"` (confirms install instructions work). Read full README end-to-end and verify no section references nonexistent features.
- **Suggested commit message:** `docs: README final review and polish`

---

## CLAUDE.md contributions (do NOT write the file; propose content)

### From README Author

**Coding style rules:**
- README uses GitHub-flavored Markdown; no HTML tags
- Terminal output examples must use ``` code fences with no language tag (plain text, not bash)
- Command examples use ```bash fenced blocks
- No emojis, no badges

**Dev commands:**
- `go build -o skill-sync .` -- build from source
- `go test ./...` -- run all tests
- `go vet ./...` -- lint

**Before you commit checklist:**
- If you changed CLI output format (tabwriter headers, status symbols, summary lines), update the corresponding examples in README.md
- If you added/removed/renamed CLI flags, update the "Global Flags" and command sections in README.md
- If you added a new provider, add a row to the "Supported Providers" table in README.md
- If you changed `.skill-sync.yaml` fields, update the "Configuration" section in README.md

**Guardrails:**
- No breaking changes to CLI output format without updating README.md examples
- Keep README.md terminal output examples in sync with actual CLI behavior

---

## EXPLAIN.md contributions (do NOT write the file; propose outline bullets)

### From README Author

**Flow / architecture explanation:**
- skill-sync reads skills from a single source provider, normalizes them into the Skill model (name, description, content, arguments), then writes them to one or more target providers with format translation
- The Skill struct is the intermediate representation -- all cross-provider comparison and syncing goes through this normalized form
- DiffEngine compares source vs. target by reading skills from both and comparing normalized content

**Key engineering decisions + tradeoffs:**
- Arguments ($ARGUMENTS, {{args}}, etc.) are passed through verbatim -- no cross-provider argument translation in MVP
- Sync is write-only: it does not delete extra skills in targets. `status` reports extras but `sync` does not remove them
- Content comparison uses `normalizeContent` (trims trailing whitespace) so minor whitespace differences don't trigger false drift

**Limits of MVP + next steps:**
- No bidirectional sync -- one source, multiple targets
- No argument placeholder translation between providers
- No watching/auto-sync -- manual CLI invocation only
- Hardcoded 4 providers, no plugin system

**How to run locally + how to validate:**
- `go build -o skill-sync .` to build
- `./skill-sync init --source claude --targets copilot,gemini,factory` to create config
- `./skill-sync sync` to sync all skills
- `./skill-sync status` to check for drift (exit code 0 = clean, 1 = drift)
- `./skill-sync diff copilot` to see unified diffs for a specific target

---

## READY FOR APPROVAL
