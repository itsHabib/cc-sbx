---
name: team-kickoff
description: Spawn a multi-agent planning team from a phase config YAML. Use when the user wants to kick off a new planning phase with multiple specialized teammates, each writing a dedicated plan doc. Each teammate stays in PLAN MODE and follows the master plan template verbatim.
argument-hint: [path/to/phase-config.yaml]
disable-model-invocation: false
---

# Team Kickoff Skill

You are the **team lead**. Your job is to:
1. Read the phase config at `$ARGUMENTS`
2. Read the master plan template at `~/.claude/skills/team-kickoff/TEMPLATE.md`
3. Create a Claude Code Team for this phase
4. Create tasks for each teammate
5. Spawn all teammates **in parallel** using the Task tool with `team_name`
6. Wait for **all** plan docs to be written before doing any synthesis

---

## Step 1 — Read inputs

Read both files before doing anything else:
- The phase config: `$ARGUMENTS`
- The master template: `~/.claude/skills/team-kickoff/TEMPLATE.md`

The phase config has this structure. All paths are relative to the project root (where Claude Code is running):

```yaml
phase: "Performance"          # human-readable phase name
goal: "..."                   # one sentence goal for this phase
system_context: |             # describe your system — as much or little as needed
  ...

dependencies:                 # existing docs/files teammates should read as truth
  - README.md                 # can be any paths relative to project root
  - src/some/file.go

plans_dir: docs/perf/plans       # where plan docs are written (by this skill)
content_dir: docs/perf/content   # where final deliverables are written (by /team-execute)

teammates:                           # 3-5 teammates per phase (best practice)
  - role: "Load Test Engineer"
    doc: loadtest.md               # filename only — resolved as <plans_dir>/<doc>
    focus: |
      - what this role should focus on
      - bullet points, as specific as you want

  - role: "Perf Analyst"
    doc: analysis.md
    focus: |
      - ...
```

**Path resolution:** Each teammate's full plan path is `<plans_dir>/<doc>`. For example, if `plans_dir: docs/perf/plans` and `doc: loadtest.md`, the plan is written to `docs/perf/plans/loadtest.md`.

---

## Step 2 — Print kickoff summary

Before spawning, print:

```
## Phase: <phase> Kickoff

**Goal:** <goal>

**Team:**
| # | Role | Plan Doc |
|---|------|----------|
| 1 | <role> | <plans_dir>/<doc> |
...

**Dependencies:** <list>

Creating team and spawning all teammates in parallel...
```

---

## Step 3 — Create the team

Use `TeamCreate` to set up the team before spawning any agents:
- `team_name`: slugified phase name (e.g., `curriculum-architecture`)
- `description`: the phase goal

Then use `TaskCreate` to create one task per teammate **before** spawning agents. Each task:
- `subject`: the role name (e.g., "Curriculum Architect")
- `description`: what this teammate will produce (their output doc path + focus areas)
- `activeForm`: present continuous (e.g., "Writing curriculum blueprint")

---

## Step 4 — Spawn all teammates in parallel

For **each** entry in `teammates`, spawn one Task tool call with:
- `subagent_type: general-purpose`
- `team_name`: the team name from Step 3
- `name`: a slugified version of the role (e.g., `curriculum-architect`)
- `run_in_background: true`

Use the following prompt for each teammate. Fill in every `<placeholder>` from the config. Embed the full template content directly in the prompt.

**Important path resolution:** Resolve each teammate's plan path as `<plans_dir>/<doc>`.

---

**Teammate prompt template:**

```
ROLE: <role>

Write ONLY to: <plans_dir>/<doc>
Do NOT edit any other files.
Do NOT write code unless and until the plan is approved.

## Team Coordination
You are part of a Claude Code team. When you finish writing your plan doc:
1. Use TaskUpdate to mark your task as completed (find your task by role name in TaskList)
2. Use SendMessage (type: "message") to notify the team lead that your plan is ready.
   - recipient: "team-lead"
   - content: "Plan complete. Doc written to <plans_dir>/<doc>. Ready for review."
   - summary: "<role> plan ready for review"

## System Context
<system_context>

## Goal for this phase
<goal>

## Your Focus Areas
<focus>

## Inputs — read all that exist, treat as truth
<dependencies as bullet list>

---

## Master Plan Template

You must follow this template VERBATIM. Fill in every section for your role.
Sections that say "as relevant to your role" should be scoped to your focus area.
Sections not applicable should be stated as "N/A — not in scope for this role" with a brief reason.

<full contents of TEMPLATE.md pasted here>

---

## IMPORTANT PROCESS RULES
- You are in PLAN MODE. Stay in PLAN MODE until the plan is explicitly approved.
- Follow the template above verbatim — do not skip or rename sections.
- Keep the "Tighten the plan into 4–7 small tasks" section strictly to **4–7 tasks**.
- Do not write code.
- End your document with the exact line: READY FOR APPROVAL
- After writing the doc, mark your task complete and message the team lead.
```

---

## Step 5 — Wait for all teammates

Do not synthesize or summarize until every teammate has sent you a completion message. Then:

1. Confirm each expected doc file exists on disk.
2. Print a brief **cross-team synthesis**:
   - Overlapping concerns or shared dependencies between plans
   - Risks that span multiple roles
   - Suggested approval order (which plan to approve and start first)
3. Ask the user: "Which plan would you like to approve first?"
4. When the session is complete, shut down teammates with `SendMessage` (type: `shutdown_request`) and call `TeamDelete` to clean up.

---

## Step 6 — Update project state (if PROJECT.state.yaml exists)

After all teammates have completed and the kickoff is done, check if `PROJECT.state.yaml` exists in the project root. If it does:

1. Find the phase entry whose `config` path matches the kickoff YAML you just ran (`$ARGUMENTS`).
2. Set that phase's `plan_status` to `done` (plans written, awaiting approval + execution).
3. Update `updated:` to today's date.

If the state file doesn't exist, skip this step — it's optional.

This keeps the state file current so `/where-are-we` can report accurate progress.
