---
name: team-execute
description: Execute approved plan docs by re-spawning teammates to write full deliverables. Takes the same phase config YAML used by /team-kickoff. Each teammate reads their approved plan doc and produces the final detailed content.
argument-hint: [path/to/phase-config.yaml]
disable-model-invocation: false
---

# Team Execute Skill

You are the **team lead**. Your job is to:
1. Read the phase config YAML at `$ARGUMENTS`
2. Verify that plan docs exist for each teammate
3. Create a Claude Code Team for execution
4. Spawn all teammates **in parallel** to execute their plans
5. Wait for all deliverables to be written

---

## Step 1 — Read the phase config

Read the phase config at `$ARGUMENTS`. It has this structure:

```yaml
phase: "Performance"
goal: "..."
system_context: |
  ...
dependencies:
  - README.md

plans_dir: docs/perf/plans       # where plan docs live (written by /team-kickoff)
content_dir: docs/perf/content   # where final deliverables are written (by this skill)

teammates:
  - role: "Load Test Engineer"
    doc: loadtest.md               # filename only — plans at <plans_dir>/<doc>, output at <content_dir>/<doc>
    focus: |
      - what this role focuses on
```

**Path resolution:**
- Each teammate's **plan doc** (input) is at `<plans_dir>/<doc>`
- Each teammate's **deliverable** (output) is at `<content_dir>/<doc>`
- Plans are NEVER overwritten — they are preserved as a record

Then verify that **each teammate's plan doc already exists on disk** at `<plans_dir>/<doc>`. If any plan doc is missing, stop and tell the user which docs are missing — they need to run `/team-kickoff` first.

Also ensure the `content_dir` directory exists (create it if needed).

---

## Step 2 — Print execution summary

Before spawning, print:

```
## Phase: <phase> — Execution

**Goal:** <goal>

**Executing plans:**
| # | Role | Plan (input) | Deliverable (output) | Status |
|---|------|-------------|---------------------|--------|
| 1 | <role> | <plans_dir>/<doc> | <content_dir>/<doc> | ✓ plan exists |
...

Spawning all teammates to execute their plans...
```

---

## Step 3 — Create the team

Use `TeamCreate` to set up the team:
- `team_name`: slugified phase name with `-exec` suffix (e.g., `crossfit-open-prep-exec`)
- `description`: "Executing: " + the phase goal

Then use `TaskCreate` to create one task per teammate. Each task:
- `subject`: "<role> — Execute Plan"
- `description`: "Read the approved plan at <plans_dir>/<doc>, then write the full deliverable to <content_dir>/<doc>."
- `activeForm`: present continuous (e.g., "Writing full nutrition plan")

---

## Step 4 — Spawn all teammates in parallel

For **each** entry in `teammates`, spawn one Task tool call with:
- `subagent_type: general-purpose`
- `team_name`: the team name from Step 3
- `name`: a slugified version of the role (e.g., `nutritionist`)
- `run_in_background: true`

Use the following prompt for each teammate. Fill in every `<placeholder>` from the config.

---

**Teammate prompt template:**

```
ROLE: <role>

You are EXECUTING an approved plan. Your job is to write the full, detailed deliverable.

Read from: <plans_dir>/<doc>  (your approved plan — DO NOT modify this file)
Write ONLY to: <content_dir>/<doc>  (your final deliverable)
Do NOT edit any other files.

## Team Coordination
When you finish writing your deliverable:
1. Use TaskUpdate to mark your task as completed (find your task by role name in TaskList)
2. Use SendMessage (type: "message") to notify the team lead:
   - recipient: "team-lead"
   - content: "Execution complete. Full deliverable written to <content_dir>/<doc>."
   - summary: "<role> deliverable complete"

## System Context
<system_context>

## Goal
<goal>

## Your Focus Areas
<focus>

## Instructions

1. **Read your approved plan doc first:** `<plans_dir>/<doc>`
   - This contains your approved plan with tasks, structure, and scope.
   - Treat it as your blueprint.

2. **Read all dependency docs for cross-domain context:**
   <dependencies as bullet list>

3. **Read the other teammates' plan docs** (read-only — do NOT edit them):
   <list all OTHER teammate plan paths as bullets: <plans_dir>/<other_doc>>
   - Use these to ensure your deliverable aligns with the other teammates' plans.
   - Reference shared timelines, phase boundaries, and cross-domain touchpoints.

4. **Write the full deliverable** to `<content_dir>/<doc>`.
   - This is the FINAL output — comprehensive, detailed, actionable.
   - Follow the structure outlined in your approved plan.
   - Include specific protocols, schedules, targets, and rationale.
   - Cross-reference other domains where coordination matters.
   - The audience is defined by the project — write clearly and practically.

## Quality Bar
- Comprehensive: covers the full scope defined in your plan
- Specific: concrete details — not vague guidance
- Aligned: references the shared timeline and other teammates' plans
- Actionable: someone could follow this document starting tomorrow
- Well-structured: clear headings, tables where appropriate, easy to navigate

## IMPORTANT
- Do NOT leave placeholder text or TODOs.
- Do NOT modify your plan doc at <plans_dir>/<doc> — it is read-only.
- Do NOT edit any file other than <content_dir>/<doc>.
- When done, mark your task complete and message the team lead.
```

---

## Step 5 — Wait for all teammates

Do not summarize until every teammate has sent a completion message. Then:

1. Confirm each deliverable file exists on disk in `content_dir`.
2. Print a brief **execution summary**:
   - Which deliverables are complete
   - Any cross-domain alignment notes worth flagging
   - Suggested reading order for the user
3. Ask the user: "All deliverables are written. Would you like me to do a cross-domain consistency review?"
4. When the session is complete, shut down teammates with `SendMessage` (type: `shutdown_request`) and call `TeamDelete` to clean up.

---

## Step 6 — Update project state (if PROJECT.state.yaml exists)

After all teammates have completed and deliverables are written, check if `PROJECT.state.yaml` exists in the project root. If it does:

1. Find the phase entry whose `config` path matches the kickoff YAML you just ran (`$ARGUMENTS`).
2. Set that phase's `status` to `done`.
3. Set `completed:` to today's date.
4. Check if any phases in the same track have `depends_on` that includes this phase. If all their dependencies are now `done`, update their status from `pending` to `next`.
5. Update `updated:` to today's date.

If the state file doesn't exist, skip this step — it's optional.

This keeps the state file current so `/where-are-we` can report accurate progress.
