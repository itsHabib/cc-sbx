# Edge Case Analyst Plan -- skill-sync Validation Phase

## You are in PLAN MODE.

### Project
I want to do a **validation phase for skill-sync**.

**Goal:** build an **edge case catalog** documenting tricky scenarios, expected behavior, and whether each case is currently handled by the codebase.

### Role + Scope (fill in)
- **Role:** Edge Case Analyst
- **Scope:** Document all edge cases across file system, content, sync logic, format translation, and config. I do NOT own the smoke test, the QE test plan, or any code implementation.
- **File you will write:** `/docs/validation/plans/edge-cases.md`
- **No-touch zones:** do not edit any other files; do not write code.

---

## Functional Requirements
- FR1: Catalog file system edge cases (missing dirs, permissions, symlinks, special chars)
- FR2: Catalog content edge cases (empty files, large files, binary, encoding)
- FR3: Catalog sync logic edge cases (idempotency, concurrent runs, filter behavior)
- FR4: Catalog format translation edge cases ($ARGUMENTS mapping, TOML conversion, frontmatter handling)
- FR5: Catalog config edge cases (validation failures, malformed YAML, missing fields)
- Tests required: N/A -- this role produces documentation, not tests
- Metrics required: N/A

## Non-Functional Requirements
- Language/runtime: N/A (documentation only)
- Local dev: N/A
- Observability: N/A
- Safety: edge case doc must clearly state which cases are handled vs. unhandled vs. out of scope
- Documentation: this IS the documentation deliverable
- Performance: N/A

---

## Assumptions / System Model
- Deployment environment: local CLI tool, no containers
- Failure modes: file I/O errors, malformed config, provider dir missing, TOML/YAML parse errors, permission denied
- Delivery guarantees: sync is NOT atomic -- partial writes possible if interrupted mid-sync
- Multi-tenancy: N/A (single-user CLI)

---

## Data Model (as relevant to your role)

The core data model relevant to edge cases:

- **Skill** -- `Name`, `Description`, `Content`, `Arguments []string`, `SourcePath`
  - Validation: Name derived from filename; no explicit length or charset validation exists
  - Content: read as raw bytes, cast to string -- no encoding validation
  - Arguments: regex-extracted, provider-specific patterns

- **Config** (`.skill-sync.yaml`) -- `Source string`, `Targets []string`, `Skills []string`
  - Validation: source must not be empty, must be registered; targets must have >= 1 entry; source must not appear in targets
  - Missing validation: no duplicate check in targets list, no skill name validation

- **Provider implementations** -- each has a `baseDir` and file extension convention
  - Claude: `*.md` in flat dir
  - Copilot: `*.prompt.md` in flat dir
  - Gemini: `*.toml` recursive with `:` namespace mapping
  - Factory: `<name>/SKILL.md` subdirectory structure with optional YAML frontmatter

---

## APIs (as relevant to your role)

N/A -- not in scope for this role. The edge case document catalogs scenarios against the existing Provider interface and CLI commands, not new APIs.

---

## Architecture / Component Boundaries (as relevant)

Components I analyze for edge cases:

1. **Provider layer** (`internal/provider/`) -- each provider's `ListSkills`, `ReadSkill`, `WriteSkill`
2. **Config layer** (`internal/config/`) -- `Load` and `Validate`
3. **Sync engine** (`internal/sync/engine.go`) -- `Sync` with filter
4. **Diff engine** (`internal/sync/diff.go`) -- `Status`, `Diff`, content normalization
5. **CLI commands** (`cmd/`) -- `init`, `sync`, `status`, `diff` with flags

---

## Correctness Invariants (must be explicit)

These are the invariants the edge case catalog tests against:

1. `sync` followed by `status` must report all skills as `in-sync` (idempotency)
2. `sync --dry-run` must never write any files
3. Config validation must reject: empty source, unknown providers, source == target, empty targets
4. A provider whose directory does not exist must return a clear error, not panic
5. `diff` on an unknown target name must return an error
6. Content comparison uses `normalizeContent` (trims trailing whitespace) -- edge cases around this normalization

---

## Tests

N/A -- not in scope for this role. The edge case document informs what SHOULD be tested; the QE Lead and Smoke Test Dev own actual test implementation.

Suggested verification for the document itself:
- Manual review: every edge case has scenario, expected behavior, and handled/unhandled/out-of-scope status
- Cross-reference: each edge case maps to a specific code path or config field

---

## Benchmarks + "Success"

N/A -- this role produces documentation, not code. Success = comprehensive, accurate edge case catalog that the QE Lead and Smoke Test Dev can reference.

---

## Engineering Decisions & Tradeoffs (REQUIRED)

### Decision 1: Organize edge cases by category, not by provider
- **Decision:** Group edge cases into File System, Content, Sync Logic, Format Translation, and Config categories
- **Alternatives considered:** Organize by provider (Claude edge cases, Copilot edge cases, etc.)
- **Why:** Category-based organization avoids duplication (most file system edge cases apply to all providers) and makes it easier for QE/Smoke Test authors to find relevant scenarios
- **Tradeoff acknowledged:** Provider-specific nuances are spread across categories rather than consolidated in one place

### Decision 2: Three-state classification (Handled / Needs Handling / Out of Scope)
- **Decision:** Every edge case gets one of three labels based on code review
- **Alternatives considered:** Binary handled/unhandled, or a priority-ranked backlog
- **Why:** The "out of scope" label is critical -- PROJECT.md explicitly excludes binary files, bidirectional sync, and auto-sync. Labeling these prevents wasted QE effort.
- **Tradeoff acknowledged:** "Needs Handling" items are not prioritized -- that is a product decision, not a QE decision

### Decision 3: Reference specific code locations for "Handled" cases
- **Decision:** For each handled edge case, cite the file and function where the handling occurs
- **Alternatives considered:** Just say "handled" without code references
- **Why:** Allows QE to write targeted tests and verify the handling is correct, not just present
- **Tradeoff acknowledged:** Code references become stale if code is refactored -- but this is a point-in-time validation artifact

---

## Risks & Mitigations (REQUIRED)

### Risk 1: Edge cases documented as "Handled" may have bugs
- **Impact:** False sense of security -- QE skips testing something that is actually broken
- **Mitigation:** For each "Handled" case, provide a concrete scenario the smoke test can exercise. The smoke test is the verification layer.
- **Validation time:** 10 minutes (spot-check 3-4 "Handled" cases against actual code)

### Risk 2: Missing edge cases due to incomplete provider format knowledge
- **Impact:** A real-world failure mode goes undocumented and untested
- **Mitigation:** Cross-reference every provider's file format spec in PROJECT.md against the actual read/write code. Flag any spec item not exercised.
- **Validation time:** 15 minutes (one pass per provider)

### Risk 3: Edge case catalog becomes too large to be actionable
- **Impact:** QE and Smoke Test Dev ignore the document because it's overwhelming
- **Mitigation:** Keep each entry to 3-4 lines. Use tables for quick scanning. Mark "Out of Scope" items separately so they don't clutter the actionable list.
- **Validation time:** 5 minutes (count entries, ensure < 40 total)

### Risk 4: Diff engine LCS implementation has O(n*m) memory for large files
- **Impact:** `diff` or `status` commands could OOM on very large skill files
- **Mitigation:** Document as an edge case with a concrete threshold estimate. This is a known tradeoff of the current implementation.
- **Validation time:** 5 minutes (estimate memory for 1MB file)

---

# Edge Case Catalog

## Category 1: File System Edge Cases

| # | Scenario | Expected Behavior | Status | Code Reference |
|---|----------|-------------------|--------|----------------|
| FS-1 | Source provider dir does not exist | `ListSkills` returns error wrapping `os.ErrNotExist`. CLI prints error and exits non-zero. | **Handled** | `claude.go:61-65` checks `os.Stat` on empty glob; `gemini.go:64-69` checks `os.Stat`+`IsDir`; `factory.go:59` `os.ReadDir` returns error; `copilot.go:54-58` same pattern as claude |
| FS-2 | Target provider dir does not exist | `WriteSkill` calls `os.MkdirAll` to create the directory before writing. Should succeed. | **Handled** | `claude.go:92`, `copilot.go:85`, `gemini.go:130`, `factory.go:98` -- all call `MkdirAll` |
| FS-3 | Target dir is read-only (permission denied on write) | `WriteSkill` returns error wrapping `os.ErrPermission`. Sync engine records it as `SyncError` per-skill, continues to next skill. | **Handled** (partial) | `engine.go:93-99` captures per-skill write errors. However, `MkdirAll` failure on a read-only parent would also error -- both paths return errors. |
| FS-4 | Symlinks in skill directories | `filepath.Glob` and `filepath.WalkDir` follow symlinks by default. Symlinked files will be read normally. Symlinked directories in Gemini (recursive walk) will be followed. | **Handled** (implicit) | Go stdlib behavior. No explicit symlink detection or rejection. Circular symlinks in Gemini's `WalkDir` could cause infinite loop -- Go's `WalkDir` does NOT protect against this. |
| FS-5 | Skill filenames with spaces | All providers derive name from filename. Spaces in filenames become spaces in skill names. No sanitization. Read/write will work on most filesystems. | **Handled** (implicit) | `filepath.Join` handles spaces correctly. No name sanitization exists. |
| FS-6 | Skill filenames with special chars (`!@#$%`) | Same as FS-5 -- passed through. Could cause issues on some filesystems or with shell globbing. | **Needs Handling** | No validation on skill names. Gemini has `geminiValidateName` but only checks for empty and `..`. Claude/Copilot/Factory have no name validation. |
| FS-7 | Very long filenames (>255 chars) | OS will return error on write. Sync engine captures as `SyncError`. | **Handled** (by OS) | No explicit length check, but OS enforces. Error propagates correctly. |
| FS-8 | Path traversal in skill names (`../../../etc/passwd`) | Gemini validates against `..` in `geminiValidateName`. Claude/Copilot/Factory do NOT validate -- a skill named `../foo` would write outside the skill dir. | **Partially Handled** | `gemini.go:222-229` rejects `..`. Other providers: no check. Factory uses skill name as subdir, so `../foo/SKILL.md` would escape. |
| FS-9 | Empty skill directory (dir exists, no skill files) | All providers return empty `[]Skill{}` slice, not an error. | **Handled** | `claude.go:63-66`, `copilot.go:56-59`, `gemini.go:100-103`, `factory.go:79-82` |
| FS-10 | Non-skill files in skill directory (e.g., `.DS_Store`, `README.md` in Gemini dir) | Claude/Copilot filter by extension (`*.md`, `*.prompt.md`). Gemini filters by `.toml`. Factory only reads `SKILL.md` in subdirs. Non-matching files ignored. | **Handled** | Extension filtering in each provider's `ListSkills`. |

## Category 2: Content Edge Cases

| # | Scenario | Expected Behavior | Status | Code Reference |
|---|----------|-------------------|--------|----------------|
| CT-1 | Empty skill file (0 bytes) | Claude/Copilot: returns Skill with empty Content, no Description, no Arguments. Gemini: TOML parse succeeds but `prompt` field empty -- returns error "missing or empty prompt field". Factory: no frontmatter, empty body -- returns Skill with empty content. | **Mixed** | `gemini.go:164-166` explicitly rejects empty prompt. Others allow empty content silently. |
| CT-2 | Very large skill file (>1MB) | Read succeeds (os.ReadFile has no size limit). Diff engine's LCS uses O(n*m) memory where n,m = line counts. A 1MB file with 20k lines would allocate ~1.6GB for LCS table (`[][]int`). | **Needs Handling** | `diff.go:337-354` `computeLCS` allocates `(m+1)*(n+1)` ints. No size guard. |
| CT-3 | Binary content (images, compiled files) | Read succeeds -- `os.ReadFile` reads any bytes. Cast to `string` produces garbled text. Write preserves bytes. Diff produces nonsensical output. | **Out of Scope** | PROJECT.md: "Skills are text files -- no binary file support needed" |
| CT-4 | Non-UTF8 text (Latin-1, Shift-JIS) | Go strings are byte sequences. Read/write preserves bytes. Content comparison works byte-by-byte. Description extraction (`strings.HasPrefix("# ")`) works on ASCII prefix. | **Handled** (implicit) | No encoding detection or conversion. Byte-level preservation is correct for passthrough. |
| CT-5 | Skills with only whitespace content | Claude/Copilot: returns Skill with whitespace Content. Gemini: `strings.TrimSpace(cmd.Prompt)` produces empty string, returns error. Status/diff: `normalizeContent` trims trailing whitespace, so whitespace-only source vs empty target = "in-sync". | **Mixed** | `gemini.go:163-166` trims and rejects. `diff.go:43-45` normalizes for comparison. |
| CT-6 | Skills with Windows line endings (CRLF) | `splitLines` in diff.go converts `\r\n` to `\n` before splitting. Content comparison handles CRLF correctly for diff. However, synced content preserves original line endings -- source CRLF is written verbatim to target. | **Handled** (for diff) | `diff.go:327` `strings.ReplaceAll(s, "\r\n", "\n")`. Sync preserves original bytes. |
| CT-7 | Skill content with no trailing newline | `normalizeContent` trims trailing whitespace/newlines. Two files differing only by trailing newline are reported as `InSync`. | **Handled** | `diff.go:43-45` |
| CT-8 | Skill with `---` in content (looks like frontmatter delimiter) | Factory: `parseFrontmatter` looks for `---` at start of content and closing `\n---`. A skill whose body contains `---` on its own line would be misparsed if it appears early. Claude/Copilot: no frontmatter parsing, safe. | **Needs Handling** | `factory.go:155-183` -- the parser finds the FIRST `\n---` after opening `---`. Body content with `---` after frontmatter is fine, but a file starting with `---` followed by non-YAML then `---` would produce a parse error. |
| CT-9 | Gemini TOML with extra/unknown fields | `toml.Unmarshal` into `geminiCommand` struct ignores unknown fields by default (BurntSushi/toml behavior). Extra fields are silently dropped on read. Round-trip (read then write) loses unknown fields. | **Handled** (with data loss) | `gemini.go:159`. Write only serializes `description` and `prompt`. |
| CT-10 | Factory frontmatter with extra fields (e.g., `model: gpt-4`) | `yaml.Unmarshal` into `factoryFrontmatter` ignores unknown fields. `model` IS a known field but is not written back by `serializeFrontmatter`. Round-trip loses `model`. | **Needs Handling** | `factory.go:187-202` -- `serializeFrontmatter` only writes `name` and `description`, drops `model`. |

## Category 3: Sync Logic Edge Cases

| # | Scenario | Expected Behavior | Status | Code Reference |
|---|----------|-------------------|--------|----------------|
| SY-1 | Source and target are the same provider | Config validation rejects this: "source must not appear in targets". | **Handled** | `config.go:54-56` |
| SY-2 | Config with no targets (empty list) | Config validation rejects: "targets must have at least one entry". | **Handled** | `config.go:47-49` |
| SY-3 | Config with invalid/unknown provider name | Config validation rejects: "unknown source/target provider". | **Handled** | `config.go:43-44` (source), `config.go:51-53` (targets) |
| SY-4 | Running sync twice in a row (idempotency) | Second sync overwrites files with identical content. `WriteSkill` uses `os.WriteFile` which truncates and rewrites. Status after should show all `in-sync`. Functionally idempotent. | **Handled** | `os.WriteFile` is inherently idempotent for same content. |
| SY-5 | Concurrent syncs (two processes running simultaneously) | No file locking. Race condition: both read source, both write to same target files. Result depends on OS write ordering. Could produce partial/corrupted files if writes interleave at byte level (unlikely with small files and `os.WriteFile`). | **Needs Handling** | No locking mechanism exists. `os.WriteFile` is not atomic on all platforms (writes in-place, not rename). |
| SY-6 | Sync interrupted mid-operation (Ctrl+C) | Partial sync: some skills written, others not. No rollback mechanism. Next sync will complete the remaining skills. | **Out of Scope** | PROJECT.md lists no atomicity requirement. Manual re-run fixes state. |
| SY-7 | Skill filter with name that doesn't match any source skill | Filter produces empty skill list. Sync completes with 0 synced, 0 errors. No warning about unmatched filter names. | **Handled** (no warning) | `engine.go:60-72` -- filter silently drops non-matching names. |
| SY-8 | Skill filter with `--skill` flag (repeatable) | Multiple `--skill` flags are collected into `syncSkills []string`. All matching skills are synced. | **Handled** | `cmd/sync.go:27` uses `StringSliceVar`. |
| SY-9 | Source has skills, target has extra skills not in source | `status` reports extra skills as `ExtraInTarget`. `sync` does NOT delete extra skills from target -- it only writes source skills. | **Handled** (by design) | `diff.go:121-128` reports extras. `engine.go` only writes, never deletes. |
| SY-10 | Source skill read fails after ListSkills succeeds (file deleted between calls) | Sync engine records error for all targets for that skill, then continues to next skill. Not a fatal error. | **Handled** | `engine.go:78-89` -- per-skill read error captured, loop continues. |
| SY-11 | Dry run with no source skills | Prints header row only, then "Would sync: 0 skill(s)..." -- actually, the code skips the summary when `len(skills) == 0` due to the `if len(skills) > 0` guard. Silent success. | **Handled** (but quiet) | `cmd/sync.go:76-78` |
| SY-12 | Duplicate target names in config | Config validation does NOT check for duplicates in targets list. Same provider would be synced twice, writing the same content twice. Harmless but wasteful. | **Needs Handling** | `config.go:47-58` -- no dedup check. |

## Category 4: Format Translation Edge Cases

| # | Scenario | Expected Behavior | Status | Code Reference |
|---|----------|-------------------|--------|----------------|
| FT-1 | Claude `$ARGUMENTS` synced to Copilot | Content is written verbatim. `$ARGUMENTS` appears as literal text in Copilot prompt file. Copilot does not recognize this syntax -- it's inert but harmless. | **Handled** (by design) | PROJECT.md: "arguments are stored verbatim and NOT translated between providers". |
| FT-2 | Claude `$ARGUMENTS` synced to Gemini | Content becomes `prompt` field in TOML. `$ARGUMENTS` is literal text. Gemini expects `{{args}}`. The placeholder won't work in Gemini. | **Handled** (by design, known limitation) | Same as FT-1. No translation by design. |
| FT-3 | Claude `$ARGUMENTS` synced to Factory | Content becomes markdown body after frontmatter. `$ARGUMENTS` is literal text. Factory has no argument syntax -- inert. | **Handled** (by design) | Same as FT-1. |
| FT-4 | Gemini skill with `{{args}}` synced to Claude | `prompt` TOML field becomes `Content` in Skill model. Written as `.md` file. `{{args}}` is literal text in Claude -- not recognized. | **Handled** (by design) | Same passthrough. |
| FT-5 | Claude skill with description (`# Deploy`) synced to Gemini | Description extracted as "Deploy". Written to `description` TOML field. Content (including `# Deploy` line) written to `prompt` field. Gemini gets both fields correctly. | **Handled** | `claude.go:123-128` extracts description. `gemini.go:134-137` writes description + content. |
| FT-6 | Gemini skill synced to Claude -- description mapping | Gemini `description` field maps to `Skill.Description`. When written to Claude, the description is NOT prepended as `# Description` -- Claude's `WriteSkill` writes `Content` only. Description is lost in Claude format. | **Needs Handling** | `claude.go:96-98` writes `skill.Content` only. For Gemini source, `Content` is the raw `prompt` field, which does NOT include the description line. |
| FT-7 | Factory skill with frontmatter synced to Claude | Factory reads body (after frontmatter) as `Content`. Description comes from frontmatter. Claude writes `Content` only -- description lost (same issue as FT-6). | **Needs Handling** | Same pattern as FT-6. `Content` for Factory = body only, not full file. |
| FT-8 | Gemini namespaced skill (`git:commit`) synced to Claude | Skill name is `git:commit`. Claude writes `git:commit.md`. The `:` character is valid on macOS/Linux filenames but invalid on Windows. | **Handled** (on Unix) | `claude.go:96` uses `skill.Name+".md"`. Works on Unix. |
| FT-9 | Content with TOML-special characters synced TO Gemini | TOML encoder (BurntSushi) handles quoting/escaping automatically. Content with `"`, `\`, newlines is safely encoded in TOML. | **Handled** | `gemini.go:140-143` uses `toml.NewEncoder`. |
| FT-10 | Diff comparison across format boundaries (source=gemini, target=claude) | Diff compares `Skill.Content` from source vs target. For Gemini source, Content = raw `prompt` value. For Claude target, Content = full file text. If Claude file was written by sync, content matches. If manually edited, diff shows real changes. | **Handled** | `diff.go:106` compares normalized content from both providers' `ReadSkill`. |
| FT-11 | Factory round-trip loses `model` field | Read extracts `model` from frontmatter but does not store it in Skill. Write reconstructs frontmatter without `model`. A sync from Factory to Factory (if allowed) would drop the `model` field. | **Needs Handling** | `factory.go:12-17` defines `Model` in frontmatter struct but `serializeFrontmatter` at line 187 ignores it. Also, `Skill` struct has no `Model` field. |

## Category 5: Config Edge Cases

| # | Scenario | Expected Behavior | Status | Code Reference |
|---|----------|-------------------|--------|----------------|
| CF-1 | Missing `.skill-sync.yaml` file | `config.Load` returns error wrapping `os.ErrNotExist`. Root command's `PersistentPreRunE` propagates it. | **Handled** | `config.go:21-23`, `root.go:26-28` |
| CF-2 | Malformed YAML in config file | `yaml.Unmarshal` returns parse error. Wrapped and returned to user. | **Handled** | `config.go:25-27` |
| CF-3 | Valid YAML but wrong structure (e.g., `source: [array]`) | `yaml.Unmarshal` into `Config` struct -- type mismatch causes parse error. | **Handled** | Go yaml.v3 strict typing |
| CF-4 | Config with `skills` filter referencing nonexistent skill names | No validation against actual skill files. Filter silently matches nothing. Sync completes with 0 skills synced, 0 errors. | **Handled** (no warning) | `engine.go:60-72` |
| CF-5 | Config with extra unknown fields | `yaml.Unmarshal` ignores unknown fields by default. No error, no warning. | **Handled** (silent) | Go yaml.v3 default behavior |
| CF-6 | Running `init` when `.skill-sync.yaml` already exists | Returns error: "already exists; remove it first or use a different --config path". | **Handled** | `cmd/init.go:35-37` |
| CF-7 | Running `init` with `--source` and `--targets` omitted | Cobra marks both flags as required. Returns usage error before `runInit` executes. | **Handled** | `cmd/init.go:28-29` |
| CF-8 | Custom `--config` path pointing to non-writable location | `os.WriteFile` returns permission error. Wrapped and returned. | **Handled** | `cmd/init.go:55-57` |
| CF-9 | Config YAML with duplicate keys (e.g., two `source:` lines) | yaml.v3 uses last value for duplicate keys. No error or warning. | **Handled** (silent, potentially surprising) | Go yaml.v3 behavior |

---

## Summary: Items Needing Attention

### Needs Handling (potential bugs or missing validation)
1. **FS-6**: No skill name validation for special characters in Claude/Copilot/Factory
2. **FS-8**: Path traversal protection missing in Claude/Copilot/Factory (only Gemini validates)
3. **CT-2**: LCS diff algorithm has O(n*m) memory -- no guard against large files
4. **CT-8**: Factory frontmatter parser can misparse content starting with `---`
5. **CT-10**: Factory `serializeFrontmatter` drops `model` field on round-trip
6. **FT-6/FT-7**: Description lost when syncing FROM Gemini/Factory TO Claude (Content doesn't include description line)
7. **FT-11**: Factory-to-Factory sync would drop `model` field
8. **SY-5**: No file locking for concurrent sync operations
9. **SY-12**: No duplicate target detection in config validation

### Out of Scope (per PROJECT.md)
1. **CT-3**: Binary file support
2. **SY-6**: Atomic/transactional sync with rollback

---

# Recommended API Surface

N/A -- not in scope for this role. This role produces a documentation artifact, not code.

---

# Folder Structure

The edge case document lives at:
```
skill-sync/
  docs/
    validation/
      plans/
        edge-cases.md    <-- this document
```

No new packages, modules, or code files.

---

# Step-by-step task plan (small commits)

N/A -- this is a single documentation file delivered in one commit. No iterative code changes.

---

# Tighten the plan into 4-7 small tasks (STRICT)

### Task 1: Read and audit all provider implementations
- **Outcome:** Verified understanding of every code path in claude.go, copilot.go, gemini.go, factory.go
- **Files to create/modify:** None (reading only)
- **Exact verification:** Cross-reference each edge case's "Code Reference" column against actual code
- **Suggested commit message:** N/A (no file changes)

### Task 2: Read and audit sync + diff engines
- **Outcome:** Verified understanding of engine.go, diff.go error handling and content normalization
- **Files to create/modify:** None (reading only)
- **Exact verification:** Confirm each SY-* and CT-* edge case references the correct line numbers
- **Suggested commit message:** N/A (no file changes)

### Task 3: Read and audit config + CLI commands
- **Outcome:** Verified understanding of config validation, init guard, flag parsing
- **Files to create/modify:** None (reading only)
- **Exact verification:** Confirm each CF-* edge case references the correct behavior
- **Suggested commit message:** N/A (no file changes)

### Task 4: Write the edge case catalog document
- **Outcome:** Complete `docs/validation/plans/edge-cases.md` with all 5 categories
- **Files to create/modify:** `docs/validation/plans/edge-cases.md`
- **Exact verification:** `cat docs/validation/plans/edge-cases.md | head -5` (file exists and has content)
- **Suggested commit message:** `docs(validation): add edge case catalog for skill-sync`

### Task 5: Cross-review with QE Lead and Smoke Test Dev plans
- **Outcome:** Ensure edge cases referenced by other plans are covered; no gaps
- **Files to create/modify:** Possibly update `edge-cases.md` if gaps found
- **Exact verification:** Manual review of cross-references
- **Suggested commit message:** `docs(validation): update edge cases after cross-review`

---

# CLAUDE.md contributions (do NOT write the file; propose content)

## From Edge Case Analyst
- When adding a new provider, audit it against the edge case catalog in `docs/validation/plans/edge-cases.md` -- especially FS-6 (special chars), FS-8 (path traversal), and CT-1 (empty files)
- All providers should validate skill names to prevent path traversal (currently only Gemini does this)
- Before committing changes to `WriteSkill` or `ReadSkill`: verify round-trip preservation (read -> write -> read produces identical Skill)
- The diff engine's LCS algorithm is O(n*m) memory -- do not use on files >10K lines without a size guard
- Config validation should be extended if new constraints are discovered (duplicate targets, skill name format)

---

# EXPLAIN.md contributions (do NOT write the file; propose outline bullets)

- **Edge case analysis methodology:** read every provider + engine code path, map to failure modes
- **Key findings:** path traversal unguarded in 3/4 providers, description loss on cross-provider sync, LCS memory unbounded
- **Format translation limitations:** arguments are passed through verbatim (by design), description field mapping is lossy in one direction
- **Idempotency:** sync is functionally idempotent (overwrite with same content), but no file locking for concurrent runs
- **What's out of scope:** binary files, atomic sync, bidirectional merge
- **How to validate:** run each edge case scenario manually or via smoke test; cross-reference "Handled" cases against actual behavior

---

## READY FOR APPROVAL
