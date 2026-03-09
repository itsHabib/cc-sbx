---
name: team-code-execute
description: Execute approved plan docs by spawning teammates to write actual code. Takes the same phase config YAML used by /team-kickoff. Each teammate reads their approved plan doc and writes the code files described in it, then self-verifies with build and test commands.
argument-hint: [path/to/phase-config.yaml]
disable-model-invocation: false
---

# Team Code Execute Skill

You are the **team lead**. Your job is to:
1. Read the phase config YAML at `$ARGUMENTS`
2. Verify that plan docs exist for each teammate
3. Create a Claude Code Team for code execution
4. Spawn all teammates **in parallel** to implement their plans
5. Wait for all code to be written and verified

---

## Step 1 — Read the phase config

Read the phase config at `$ARGUMENTS`. It has this structure:

```yaml
phase: "Build"
goal: "..."
system_context: |
  ...
dependencies:
  - PROJECT.md

plans_dir: docs/build/plans       # where plan docs live (written by /team-kickoff)
content_dir: docs/build/content   # not used for code execution — teammates write code directly

teammates:
  - role: "Backend Engineer"
    doc: backend.md                # plan doc at <plans_dir>/<doc>
    focus: |
      - what this role focuses on
```

**Path resolution:**
- Each teammate's **plan doc** (input) is at `<plans_dir>/<doc>`
- Unlike `/team-execute`, teammates write **code files directly** to the project — not to a single output doc

Then verify that **each teammate's plan doc already exists on disk** at `<plans_dir>/<doc>`. If any plan doc is missing, stop and tell the user which docs are missing — they need to run `/team-kickoff` first.

---

## Step 2 — Print execution summary

Before spawning, print:

```
## Phase: <phase> — Code Execution

**Goal:** <goal>

**Executing plans:**
| # | Role | Plan (input) | Status |
|---|------|-------------|--------|
| 1 | <role> | <plans_dir>/<doc> | plan exists |
...

Spawning all teammates to write code from their plans...
```

---

## Step 3 — Create the team

Use `TeamCreate` to set up the team:
- `team_name`: slugified phase name with `-code` suffix (e.g., `build-code`)
- `description`: "Code execution: " + the phase goal

Then use `TaskCreate` to create one task per teammate. Each task:
- `subject`: "<role> — Write Code"
- `description`: "Read approved plan at <plans_dir>/<doc>, then write all code files described in the plan."
- `activeForm`: present continuous (e.g., "Writing backend code")

---

## Step 4 — Spawn all teammates in parallel

For **each** entry in `teammates`, spawn one Task tool call with:
- `subagent_type: general-purpose`
- `team_name`: the team name from Step 3
- `name`: a slugified version of the role (e.g., `backend-engineer`)
- `run_in_background: true`

Use the following prompt for each teammate:

---

**Teammate prompt template:**

```
ROLE: <role>

You are EXECUTING an approved plan. Your job is to write all the code described in your plan.

Read your plan: <plans_dir>/<doc>
Write the code files described in your plan. You may create and edit any files listed in your plan.
Do NOT modify files owned by other teammates.

## Team Coordination
When you finish writing and verifying all your code:
1. Use TaskUpdate to mark your task as completed (find your task by role name in TaskList)
2. Use SendMessage (type: "message") to notify the team lead:
   - recipient: "team-lead"
   - content: "Code complete. All files written and verified. Ready for review."
   - summary: "<role> code complete"

## System Context
<system_context>

## Goal
<goal>

## Your Focus Areas
<focus>

## Dependencies — read all that exist for context
<dependencies as bullet list>

## Other teammates' plans — read for alignment (do NOT write their code)
<list all OTHER teammate plan paths as bullets: <plans_dir>/<other_doc>>

## Instructions

1. **Read your approved plan doc first:** `<plans_dir>/<doc>`
   - This contains your tasks, file list, and verification commands.
   - Treat it as your blueprint.

2. **Read all dependency docs** for project context:
   <dependencies as bullet list>

3. **Read other teammates' plan docs** (read-only):
   <list all OTHER teammate plan paths>
   - Understand their interfaces so your code integrates correctly.
   - If you need types/interfaces from their packages, define minimal stubs
     or interfaces that match the agreed contract in their plan.

4. **Write all code files** described in your plan.
   - Follow the task order from your plan.
   - Write clean, production-quality code.
   - Include all imports, error handling, and documentation as specified.

5. **Self-verify after EVERY file you write:**
   - Run `go build ./...` (or equivalent for your language) to check compilation.
   - If the build fails due to missing sibling packages (other teammates' code),
     create minimal stubs so YOUR code compiles. Mark stubs clearly with
     `// STUB: will be replaced by <role>'s implementation` comments.
   - Run any tests you write: `go test ./... -v`
   - Fix any failures before moving to the next file.

6. **Final verification — run ALL of these before marking complete:**
   - Build: `go build ./...`
   - Tests: `go test ./... -v`
   - Vet: `go vet ./...`
   - If any fail, fix them. Do not mark complete with failing tests or builds.

## IMPORTANT RULES
- Write real code, not pseudocode or documentation.
- Follow the exact file paths from your plan.
- Do NOT modify files owned by other teammates.
- If you need to stub a dependency from another teammate's package, keep the stub
  minimal and clearly marked — it will be replaced when their code lands.
- Self-verify after every file. Fix issues immediately.
- Only mark your task complete when ALL verification commands pass.
```

---

## Step 5 — Wait for all teammates

Do not summarize until every teammate has sent a completion message. Then:

1. Run a final project-wide build check: `go build ./...` (or equivalent).
2. Run a final project-wide test: `go test ./... -v` (or equivalent).
3. Print an **execution summary**:
   - Which teammates completed successfully
   - Any stub files that need resolution (grep for `// STUB:`)
   - Final build/test status
   - List of all files created or modified
4. If there are stub conflicts (two teammates stubbed the same file), resolve them
   by keeping the real implementation and removing the stub.
5. Ask the user: "All code is written. Would you like me to do a final integration check and resolve any stub conflicts?"
6. When the session is complete, shut down teammates with `SendMessage` (type: `shutdown_request`) and call `TeamDelete` to clean up.
