# Edge Case Catalog -- skill-sync

This document catalogs edge cases, tricky scenarios, and expected behavior for
skill-sync. Each entry includes a scenario description, expected behavior, and
a status indicating whether it is currently handled by the codebase.

**Status legend:**

| Label | Meaning |
|-------|---------|
| **Handled** | Code explicitly handles this case correctly |
| **Needs Handling** | The case is not handled and could cause incorrect behavior |
| **Out of Scope** | Explicitly excluded by PROJECT.md or by design |

---

## Category 1: File System Edge Cases

| # | Scenario | Expected Behavior | Status | Code Reference |
|---|----------|-------------------|--------|----------------|
| FS-1 | Source provider directory does not exist | `ListSkills` returns an error wrapping the OS error. Claude/Copilot check `os.Stat` when glob returns 0 matches (`claude.go:62-64`, `copilot.go:55-57`). Gemini checks `os.Stat` + `IsDir` up front (`gemini.go:64-69`). Factory's `os.ReadDir` returns the error directly (`factory.go:58-60`). CLI prints the error and exits non-zero. | **Handled** | `claude.go:62-64`, `copilot.go:55-57`, `gemini.go:64-69`, `factory.go:58-60` |
| FS-2 | Target provider directory does not exist | `WriteSkill` calls `os.MkdirAll` to create the directory before writing. All four providers do this. | **Handled** | `claude.go:92`, `copilot.go:85`, `gemini.go:130`, `factory.go:98` |
| FS-3 | Target directory is read-only (permission denied on write) | `WriteSkill` returns an error wrapping `os.ErrPermission`. Sync engine records it as a `SyncError` per-skill and continues to the next skill. | **Handled** | `engine.go:93-100` captures per-skill write errors |
| FS-4 | Symlinks in skill directories | `filepath.Glob` (Claude/Copilot) and `filepath.WalkDir` (Gemini) follow symlinks by default. Symlinked files are read normally. However, circular symlinks in Gemini's `WalkDir` could cause an infinite loop -- Go's `filepath.WalkDir` does NOT protect against circular symlinks. | **Handled** (implicit, with caveat) | Go stdlib behavior; no explicit symlink handling in any provider |
| FS-5 | Skill filenames with spaces | All providers derive skill name from filename. Spaces in filenames become spaces in skill names. `filepath.Join` handles spaces correctly. No name sanitization exists. | **Handled** (implicit) | No sanitization code; relies on OS/filepath behavior |
| FS-6 | Skill filenames with special characters (`!@#$%`) | Passed through without validation. Could cause issues on some filesystems or with shell globbing. Gemini has `geminiValidateName` but only checks for empty and `..` (`gemini.go:222-229`). Claude, Copilot, and Factory have no name validation. | **Needs Handling** | `gemini.go:222-229` (partial); Claude/Copilot/Factory: none |
| FS-7 | Very long filenames (>255 chars) | OS returns an error on write. Sync engine captures it as a `SyncError`. No explicit length check exists, but the OS enforces limits. | **Handled** (by OS) | Error propagates through `os.WriteFile` |
| FS-8 | Path traversal in skill names (`../../../etc/passwd`) | Gemini validates against `..` in `geminiValidateName` (`gemini.go:226`). Claude, Copilot, and Factory do NOT validate -- a skill named `../foo` would cause writes outside the skill directory. Factory is especially concerning since it uses the skill name as a subdirectory (`factory.go:97`), so `../foo/SKILL.md` would escape the base dir. | **Needs Handling** | `gemini.go:222-229` (handled); Claude `claude.go:96`, Copilot `copilot.go:89`, Factory `factory.go:97` (not handled) |
| FS-9 | Empty skill directory (exists but contains no skill files) | All providers return an empty `[]Skill{}` slice, not an error. | **Handled** | `claude.go:65`, `copilot.go:58`, `gemini.go:100-102`, `factory.go:79-81` |
| FS-10 | Non-skill files in skill directory (e.g., `.DS_Store`, `README.md`) | Claude/Copilot filter by glob pattern (`*.md`, `*.prompt.md`). Gemini filters by `.toml` extension in `WalkDir` (`gemini.go:80`). Factory only reads `SKILL.md` inside subdirectories (`factory.go:68-70`). Non-matching files are ignored. | **Handled** | Extension/pattern filtering in each provider's `ListSkills` |

---

## Category 2: Content Edge Cases

| # | Scenario | Expected Behavior | Status | Code Reference |
|---|----------|-------------------|--------|----------------|
| CT-1 | Empty skill file (0 bytes) | Claude/Copilot: returns a Skill with empty Content, no Description, no Arguments. Gemini: `strings.TrimSpace(cmd.Prompt)` produces empty string, returns error "missing or empty prompt field" (`gemini.go:163-165`). Factory: no frontmatter detected, empty body, returns Skill with empty content. | **Mixed** | `gemini.go:163-165` rejects; others allow silently |
| CT-2 | Very large skill file (>1MB) | `os.ReadFile` has no size limit, so read succeeds. The diff engine's LCS algorithm (`computeLCS`) allocates a `(m+1) * (n+1)` int table where m and n are line counts (`diff.go:337-354`). A 1MB file with ~20K lines would allocate ~1.6GB. No size guard exists. | **Needs Handling** | `diff.go:337-354` (`computeLCS` -- unbounded allocation) |
| CT-3 | Binary content (images, compiled files) | `os.ReadFile` reads any bytes. Cast to `string` produces garbled text. Diff produces nonsensical output. | **Out of Scope** | PROJECT.md: "Skills are text files -- no binary file support needed" |
| CT-4 | Non-UTF8 text (Latin-1, Shift-JIS) | Go strings are byte sequences. Read/write preserves bytes exactly. Content comparison works byte-by-byte. Description extraction (`strings.HasPrefix("# ")`) works correctly on ASCII-prefix bytes. | **Handled** (implicit) | Byte-level preservation; no encoding conversion needed |
| CT-5 | Skills with only whitespace content | Claude/Copilot: returns Skill with whitespace Content. Gemini: `strings.TrimSpace` produces empty string, returns error (`gemini.go:163-165`). For diff/status: `normalizeContent` trims trailing whitespace (`diff.go:43-44`), so whitespace-only source vs. empty target would be reported as "in-sync". | **Mixed** | `gemini.go:163-165` rejects; `diff.go:43-44` normalizes |
| CT-6 | Skills with Windows line endings (CRLF) | `splitLines` in diff.go replaces `\r\n` with `\n` before splitting (`diff.go:327`). Content comparison handles CRLF correctly for diff purposes. However, sync preserves the original bytes verbatim -- source CRLF is written as-is to targets. | **Handled** (for diff comparison) | `diff.go:327` |
| CT-7 | Skill content with no trailing newline | `normalizeContent` trims trailing whitespace and newlines (`diff.go:43-44`). Two files differing only by a trailing newline are reported as `InSync`. | **Handled** | `diff.go:43-44` |
| CT-8 | Skill with `---` in content (resembles frontmatter delimiter) | Factory: `parseFrontmatter` checks if content starts with `---`, then finds the next `\n---` as the closing delimiter (`factory.go:155-167`). If a file starts with `---` followed by non-YAML content then `---`, the YAML parse would fail and return an error. If there is no closing `\n---`, the file is treated as having no frontmatter (`factory.go:168-169`). Claude/Copilot: no frontmatter parsing, so `---` is safe. | **Needs Handling** | `factory.go:155-183` -- content starting with `---` could be misparsed |
| CT-9 | Gemini TOML with extra/unknown fields | `toml.Unmarshal` into the `geminiCommand` struct ignores unknown fields by default (BurntSushi/toml behavior). Extra fields are silently dropped on read. A round-trip (read then write) loses unknown fields because `WriteSkill` only serializes `description` and `prompt`. | **Handled** (with data loss on round-trip) | `gemini.go:159` (read), `gemini.go:134-137` (write) |
| CT-10 | Factory frontmatter with extra fields (e.g., `model: inherit`) | `yaml.Unmarshal` into `factoryFrontmatter` reads the `model` field (`factory.go:16`), but `serializeFrontmatter` only writes `name` and `description` (`factory.go:188-191`). The `Skill` struct has no `Model` field, so the model value is not preserved. Round-trip loses the `model` field. | **Needs Handling** | `factory.go:13-17` (struct has `Model`), `factory.go:187-201` (`serializeFrontmatter` omits it) |

---

## Category 3: Sync Logic Edge Cases

| # | Scenario | Expected Behavior | Status | Code Reference |
|---|----------|-------------------|--------|----------------|
| SY-1 | Source and target are the same provider | Config validation rejects: "source must not appear in targets". | **Handled** | `config.go:55` |
| SY-2 | Config with no targets (empty list) | Config validation rejects: "targets must have at least one entry". | **Handled** | `config.go:47-48` |
| SY-3 | Config with invalid/unknown provider name | Config validation rejects: "unknown source/target provider". | **Handled** | `config.go:43-44` (source), `config.go:51-52` (targets) |
| SY-4 | Running sync twice in a row (idempotency) | Second sync overwrites files with identical content. `os.WriteFile` truncates and rewrites. Status after second sync reports all `InSync`. Functionally idempotent. | **Handled** | `os.WriteFile` is inherently idempotent for same content |
| SY-5 | Concurrent syncs (two processes running simultaneously) | No file locking. Race condition: both processes read source, both write to same target files. With small files and `os.WriteFile`, interleaved byte-level corruption is unlikely but not impossible. No protection exists. | **Needs Handling** | No locking mechanism in the codebase |
| SY-6 | Sync interrupted mid-operation (Ctrl+C) | Partial sync: some skills written, others not. No rollback mechanism. A subsequent sync run will complete the remaining skills. | **Out of Scope** | PROJECT.md lists no atomicity requirement; manual re-run fixes state |
| SY-7 | Skill filter with name that matches no source skills | Filter produces an empty skill list. Sync completes with 0 synced, 0 errors. No warning about unmatched filter names. | **Handled** (no warning) | `engine.go:60-72` -- filter silently drops non-matching names |
| SY-8 | Skill filter with `--skill` flag (repeatable) | Multiple `--skill` flags are collected into `syncSkills []string`. All matching skills are synced. | **Handled** | `cmd/sync.go:27` uses `StringSliceVar` |
| SY-9 | Source has skills, target has extra skills not in source | `status` reports extra skills as `ExtraInTarget` (`diff.go:121-128`). `sync` does NOT delete extra skills from target -- it only writes source skills. | **Handled** (by design) | `diff.go:121-128` (detection), `engine.go` (write-only, no delete) |
| SY-10 | Source skill file deleted between ListSkills and ReadSkill | Sync engine records an error for all targets for that skill, then continues to next skill. Not a fatal error. | **Handled** | `engine.go:77-89` -- per-skill read error captured, loop continues |
| SY-11 | Dry run with no source skills | `doSyncDryRun` lists skills, finds 0, prints header only. The summary line is guarded by `if len(skills) > 0` (`cmd/sync.go:76`), so no summary is printed. Silent success. | **Handled** (but quiet) | `cmd/sync.go:76-78` |
| SY-12 | Duplicate target names in config | Config validation does NOT check for duplicate entries in the targets list (`config.go:47-58`). The same provider would be synced twice, writing the same content twice. Harmless but wasteful. | **Needs Handling** | `config.go:47-58` -- no dedup check |

---

## Category 4: Format Translation Edge Cases

| # | Scenario | Expected Behavior | Status | Code Reference |
|---|----------|-------------------|--------|----------------|
| FT-1 | Claude `$ARGUMENTS` synced to Copilot | Content is written verbatim. `$ARGUMENTS` appears as literal text in the Copilot `.prompt.md` file. Copilot does not recognize this syntax -- it is inert but harmless. | **Handled** (by design) | PROJECT.md: "arguments are stored verbatim and NOT translated between providers" |
| FT-2 | Claude `$ARGUMENTS` synced to Gemini | Content becomes the `prompt` field in TOML. `$ARGUMENTS` is literal text. Gemini expects `{{args}}` for argument injection. The placeholder will not work in Gemini. | **Handled** (by design, known limitation) | Same design decision as FT-1 |
| FT-3 | Claude `$ARGUMENTS` synced to Factory | Content becomes the markdown body after frontmatter. `$ARGUMENTS` is literal text. Factory has no argument syntax -- inert. | **Handled** (by design) | Same design decision as FT-1 |
| FT-4 | Gemini skill with `{{args}}` synced to Claude | `prompt` TOML field becomes `Content` in Skill model. Written as `.md` file. `{{args}}` is literal text in Claude -- not recognized as an argument placeholder. | **Handled** (by design) | Same passthrough behavior |
| FT-5 | Claude skill with description (`# Deploy`) synced to Gemini | Description extracted as "Deploy" (`claude.go:123-127`). Written to `description` TOML field. Content (including the `# Deploy` line) written to `prompt` field. Gemini gets both fields correctly. | **Handled** | `claude.go:123-127` (extract), `gemini.go:134-137` (write) |
| FT-6 | Gemini skill synced to Claude -- description lost | Gemini's `description` field maps to `Skill.Description`. Gemini's `prompt` field maps to `Skill.Content` -- but does NOT include the description as a `# ` header line. Claude's `WriteSkill` writes `skill.Content` only (`claude.go:97`). The description is lost in the Claude output. | **Needs Handling** | `claude.go:97` writes `Content` only; Gemini `Content` = raw `prompt` field without description header |
| FT-7 | Factory skill synced to Claude -- description lost | Factory reads body (after frontmatter) as `Content` and frontmatter `description` as `Description`. Claude writes `Content` only (`claude.go:97`) -- description is lost. Same issue as FT-6. | **Needs Handling** | Same pattern as FT-6; `factory.go:130` sets `Content = body` |
| FT-8 | Gemini namespaced skill (`git:commit`) synced to Claude | Skill name is `git:commit`. Claude writes `git:commit.md`. The `:` character is valid on macOS/Linux but invalid on Windows. | **Handled** (on Unix) | `claude.go:96` uses `skill.Name + ".md"` |
| FT-9 | Content with TOML-special characters synced TO Gemini | BurntSushi's TOML encoder handles quoting and escaping automatically. Content with `"`, `\`, and newlines is safely encoded. | **Handled** | `gemini.go:140-143` uses `toml.NewEncoder` |
| FT-10 | Diff comparison across format boundaries (source=Gemini, target=Claude) | Diff compares `Skill.Content` from each provider's `ReadSkill`. For Gemini source, Content = raw `prompt` value. For Claude target, Content = full file text. If the Claude file was written by sync, content matches. If manually edited, diff shows real changes. | **Handled** | `diff.go:106` compares normalized content from both providers |
| FT-11 | Factory round-trip loses `model` field | `factoryFrontmatter` struct includes `Model` (`factory.go:16`), but `serializeFrontmatter` constructs a new struct with only `Name` and `Description` (`factory.go:188-191`). A sync from Factory source to Factory target (if allowed by config) would drop the `model` field. Same-provider sync is blocked by config validation, but cross-provider round-trips involving Factory lose `model`. | **Needs Handling** | `factory.go:187-201` |

---

## Category 5: Config Edge Cases

| # | Scenario | Expected Behavior | Status | Code Reference |
|---|----------|-------------------|--------|----------------|
| CF-1 | Missing `.skill-sync.yaml` file | `config.Load` returns error wrapping `os.ErrNotExist`. Root command's `PersistentPreRunE` propagates it. | **Handled** | `config.go:21-22`, `root.go:26-28` |
| CF-2 | Malformed YAML in config file | `yaml.Unmarshal` returns a parse error. Wrapped and returned to user. | **Handled** | `config.go:25-26` |
| CF-3 | Valid YAML but wrong structure (e.g., `source: [array]`) | `yaml.Unmarshal` into `Config` struct causes a type mismatch error. | **Handled** | Go yaml.v3 strict typing |
| CF-4 | Config `skills` filter referencing nonexistent skill names | No validation against actual skill files at config load time. Filter silently matches nothing at sync time. Sync completes with 0 skills synced, 0 errors. | **Handled** (no warning) | `engine.go:60-72` |
| CF-5 | Config with extra unknown YAML fields | `yaml.Unmarshal` ignores unknown fields by default. No error, no warning. | **Handled** (silent) | Go yaml.v3 default behavior |
| CF-6 | Running `init` when `.skill-sync.yaml` already exists | Returns error: "already exists; remove it first or use a different --config path". | **Handled** | `cmd/init.go:35-36` |
| CF-7 | Running `init` without required `--source` and `--targets` flags | Cobra marks both flags as required (`cmd/init.go:28-29`). Returns usage error before `runInit` executes. | **Handled** | `cmd/init.go:28-29` |
| CF-8 | Custom `--config` path pointing to non-writable location | `os.WriteFile` returns permission error. Wrapped and returned. | **Handled** | `cmd/init.go:55-56` |
| CF-9 | Config YAML with duplicate keys (e.g., two `source:` lines) | Go yaml.v3 uses the last value for duplicate keys. No error or warning. | **Handled** (silent, potentially surprising) | Go yaml.v3 behavior |

---

## Summary: Items Needing Attention

### Needs Handling (potential bugs or missing validation)

| # | Issue | Severity | Recommendation |
|---|-------|----------|----------------|
| FS-6 | No skill name validation for special characters in Claude/Copilot/Factory | Low | Add name validation similar to Gemini's `geminiValidateName` |
| FS-8 | Path traversal protection missing in Claude/Copilot/Factory | **High** | Add `..` validation to all providers' `WriteSkill` and `ReadSkill` |
| CT-2 | LCS diff algorithm has O(n*m) memory with no guard | Medium | Add a line-count threshold; fall back to simpler diff for large files |
| CT-8 | Factory frontmatter parser can misparse content starting with `---` | Low | Validate YAML between delimiters before treating as frontmatter |
| CT-10 | Factory `serializeFrontmatter` drops the `model` field on round-trip | Medium | Preserve extra frontmatter fields through the Skill model or serialize them |
| FT-6 | Description lost when syncing FROM Gemini TO Claude | Medium | Claude's `WriteSkill` should prepend `# Description` if `Skill.Description` is set and `Content` lacks a `# ` header |
| FT-7 | Description lost when syncing FROM Factory TO Claude | Medium | Same fix as FT-6 |
| FT-11 | Factory round-trip loses `model` field | Medium | Same root cause as CT-10 |
| SY-5 | No file locking for concurrent sync operations | Low | Add advisory file lock or document as a known limitation |
| SY-12 | No duplicate target detection in config validation | Low | Add dedup check in `config.Validate` |

### Out of Scope (per PROJECT.md)

| # | Issue | Reason |
|---|-------|--------|
| CT-3 | Binary file support | "Skills are text files -- no binary file support needed" |
| SY-6 | Atomic/transactional sync with rollback | No atomicity requirement; manual re-run fixes partial state |
